package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	redislib "github.com/redis/go-redis/v9"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/cache"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/database"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/providers/geolocation"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/providers/scheduling"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/search"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/handlers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/routes"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/application/services"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/postgres"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/providerapi"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/redis"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/typesense"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/observability"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
)

func main() {

	// Load configuration

	cfg, err := config.Load()

	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize OpenTelemetry if enabled
	var shutdown func(context.Context) error
	if cfg.OTEL.Enabled && cfg.OTEL.Endpoint != "" {
		shutdown, err = observability.Setup(
			ctx,
			cfg.OTEL.ServiceName,
			cfg.OTEL.ServiceVersion,
			cfg.OTEL.Endpoint,
		)
		if err != nil {
			log.Printf("Warning: Failed to set up OpenTelemetry: %v", err)
		} else {
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := shutdown(ctx); err != nil {
					log.Printf("Error shutting down OpenTelemetry: %v", err)
				}
			}()
			log.Println("OpenTelemetry initialized successfully")
		}
	}

	// Initialize metrics
	metrics, err := observability.InitMetrics()
	if err != nil {
		log.Fatalf("Failed to initialize metrics: %v", err)
	}

	// Initialize database client
	pgClient, err := postgres.NewClient(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize PostgreSQL client: %v", err)
	}
	defer pgClient.Close()
	log.Println("PostgreSQL client initialized successfully")

	// Initialize Redis client
	redisClient, err := redis.NewClient(&cfg.Redis)
	if err != nil {
		log.Printf("Warning: Failed to initialize Redis client: %v", err)
		// Continue without Redis - the application can work without caching
	} else {
		defer redisClient.Close()
		log.Println("Redis client initialized successfully")
	}

	// Initialize Typesense client
	typesenseClient, err := typesense.NewClient(&cfg.Typesense)
	if err != nil {
		log.Printf("Warning: Failed to initialize Typesense client: %v", err)
	} else {
		log.Println("Typesense client initialized successfully")
	}

	// Initialize Provider API client
	var providerClient providerapi.Client
	if cfg.ProviderAPI.BaseURL != "" {
		providerClient = providerapi.NewClient(cfg.ProviderAPI.BaseURL)
		log.Println("Provider API client initialized successfully")
	}

	// Initialize adapters

	facilityAdapter := database.NewFacilityAdapter(pgClient)

	appointmentAdapter := database.NewAppointmentAdapter(pgClient)

	procedureAdapter := database.NewProcedureAdapter(pgClient)
	facilityProcedureAdapter := database.NewFacilityProcedureAdapter(pgClient)

	insuranceAdapter := database.NewInsuranceAdapter(pgClient)
	feedbackAdapter := database.NewFeedbackAdapter(pgClient)

	var searchRepo repositories.FacilitySearchRepository

	if typesenseClient != nil {

		adapter := search.NewTypesenseAdapter(typesenseClient)

		// Ensure schema exists

		if err := adapter.InitSchema(context.Background()); err != nil {

			log.Printf("Warning: Failed to init Typesense schema: %v", err)

		}

		searchRepo = adapter

	}

	var cacheProvider providers.CacheProvider
	if redisClient != nil {
		cacheProvider = cache.NewRedisAdapter(redisClient)
	}

	var geolocationProvider providers.GeolocationProvider
	switch cfg.Geolocation.Provider {
	case "google":
		if cfg.Geolocation.APIKey == "" {
			log.Println("Warning: GEOLOCATION_API_KEY is not set; using mock geolocation provider")
			geolocationProvider = geolocation.NewMockGeolocationProvider()
		} else {
			geolocationProvider = geolocation.NewGoogleGeolocationProvider(cfg.Geolocation.APIKey, cacheProvider)
		}
	default:
		geolocationProvider = geolocation.NewMockGeolocationProvider()
	}

	appointmentProvider := scheduling.NewCalendlyAdapter(os.Getenv("CALENDLY_API_KEY"))

	// Initialize services

	facilityService := services.NewFacilityService(
		facilityAdapter,
		searchRepo,
		facilityProcedureAdapter,
		procedureAdapter,
		insuranceAdapter,
	)

	appointmentService := services.NewAppointmentService(appointmentAdapter, appointmentProvider)
	feedbackService := services.NewFeedbackService(feedbackAdapter)

	// Initialize handlers

	facilityHandler := handlers.NewFacilityHandler(facilityService)

	appointmentHandler := handlers.NewAppointmentHandler(appointmentService)

	procedureHandler := handlers.NewProcedureHandler(procedureAdapter)

	insuranceHandler := handlers.NewInsuranceHandler(insuranceAdapter)

	geolocationHandler := handlers.NewGeolocationHandler(geolocationProvider)

	mapsHandler := handlers.NewMapsHandler(cfg.Geolocation.APIKey, cacheProvider)
	feedbackHandler := handlers.NewFeedbackHandler(feedbackService, cacheProvider)

	providerPriceHandler := handlers.NewProviderPriceHandler(providerClient)
	pageSize := 0
	if value := strings.TrimSpace(os.Getenv("PROVIDER_INGEST_PAGE_SIZE")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			pageSize = parsed
		}
	}
	ingestionService := services.NewProviderIngestionService(
		providerClient,
		facilityAdapter,
		facilityService,
		procedureAdapter,
		facilityProcedureAdapter,
		pageSize,
	)
	idempotencyTTL := 24 * time.Hour
	if value := strings.TrimSpace(os.Getenv("PROVIDER_INGESTION_IDEMPOTENCY_TTL_MINUTES")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			idempotencyTTL = time.Duration(parsed) * time.Minute
		}
	}
	var redisRaw *redislib.Client
	if redisClient != nil {
		redisRaw = redisClient.Client()
	}
	providerIngestionHandler := handlers.NewProviderIngestionHandler(ingestionService, redisRaw, idempotencyTTL)

	// Set up router

	router := routes.NewRouter(
		facilityHandler,
		appointmentHandler,
		procedureHandler,
		insuranceHandler,
		geolocationHandler,
		mapsHandler,
		feedbackHandler,
		providerPriceHandler,
		providerIngestionHandler,
		metrics,
	)

	handler := router.SetupRoutes()

	// Create HTTP server
	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server starting on %s", serverAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Optional: ingest provider data on startup
	if strings.EqualFold(os.Getenv("PROVIDER_INGEST_ON_START"), "true") && providerClient != nil {
		providerID := strings.TrimSpace(os.Getenv("PROVIDER_INGEST_PROVIDER_ID"))
		go func() {
			ingestCtx, ingestCancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer ingestCancel()
			for attempt := 1; attempt <= 5; attempt++ {
				summary, err := ingestionService.SyncCurrentData(ingestCtx, providerID)
				if err == nil {
					log.Printf("Provider ingestion completed: %+v", summary)
					return
				}
				log.Printf("Provider ingestion failed (attempt %d): %v", attempt, err)
				time.Sleep(5 * time.Second)
			}
		}()
	}

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error during server shutdown: %v", err)
	}

	log.Println("Server stopped")

	// Suppress unused variable warnings (would be used in production)
	_ = geolocationProvider
}

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/cache"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/database"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/events"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/providers/geolocation"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/providers/scheduling"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/search"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/handlers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/middleware"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/routes"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/application/services"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/openai"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/postgres"
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

	// Initialize adapters

	// Create base facility adapter
	baseFacilityAdapter := database.NewFacilityAdapter(pgClient)

	// Wrap with caching if Redis is available (for read performance optimization)
	var facilityAdapter repositories.FacilityRepository
	if cacheProvider != nil {
		facilityAdapter = database.NewCachedFacilityAdapter(baseFacilityAdapter, cacheProvider)
		log.Println("✓ Facility adapter wrapped with caching layer")
	} else {
		facilityAdapter = baseFacilityAdapter
		log.Println("⚠ Facility adapter running without cache (Redis unavailable)")
	}

	appointmentAdapter := database.NewAppointmentAdapter(pgClient)

	procedureAdapter := database.NewProcedureAdapter(pgClient)
	facilityProcedureAdapter := database.NewFacilityProcedureAdapter(pgClient)
	procedureEnrichmentAdapter := database.NewProcedureEnrichmentAdapter(pgClient)

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

	// Initialize event bus for real-time updates
	var eventBus providers.EventBus
	if redisClient != nil {
		eventBus = events.NewRedisEventBus(redisClient)
		log.Println("Event bus initialized successfully")
	} else {
		log.Println("Event bus disabled (Redis not available)")
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

	var enrichmentProvider providers.ProcedureEnrichmentProvider
	if cfg.OpenAI.APIKey == "" {
		log.Println("Warning: OPENAI_API_KEY is not set; procedure enrichment disabled")
	} else {
		openaiClient, err := openai.NewClient(&cfg.OpenAI)
		if err != nil {
			log.Printf("Warning: Failed to initialize OpenAI client: %v", err)
		} else {
			enrichmentProvider = openaiClient
		}
	}

	// Initialize services

	facilityService := services.NewFacilityService(
		facilityAdapter,
		searchRepo,
		facilityProcedureAdapter,
		procedureAdapter,
		insuranceAdapter,
	)

	// Set event bus for real-time updates
	if eventBus != nil {
		facilityService.SetEventBus(eventBus)
		log.Println("Event bus configured for facility service")
	}

	// Initialize cache invalidation service
	var cacheInvalidationService *services.CacheInvalidationService
	if cacheProvider != nil && eventBus != nil {
		cacheInvalidationService = services.NewCacheInvalidationService(cacheProvider, eventBus)
		if err := cacheInvalidationService.Start(); err != nil {
			log.Printf("Warning: Failed to start cache invalidation service: %v", err)
		} else {
			log.Println("Cache invalidation service started successfully")
		}
	}

	appointmentService := services.NewAppointmentService(appointmentAdapter, appointmentProvider)
	feedbackService := services.NewFeedbackService(feedbackAdapter)
	procedureEnrichmentService := services.NewProcedureEnrichmentService(
		procedureEnrichmentAdapter,
		procedureAdapter,
		enrichmentProvider,
	)

	// Start cache warming service for improved read performance
	if cacheProvider != nil {
		warmingService := services.NewCacheWarmingService(
			facilityAdapter, // Use cached adapter to warm cache
			cacheProvider,
		)
		go warmingService.StartPeriodicWarming(ctx, 5*time.Minute)
		log.Println("✓ Cache warming service started (refreshes every 5 minutes)")
	}

	// Initialize handlers

	facilityHandler := handlers.NewFacilityHandler(facilityService)

	appointmentHandler := handlers.NewAppointmentHandler(appointmentService)

	procedureHandler := handlers.NewProcedureHandler(procedureAdapter, procedureEnrichmentService)

	insuranceHandler := handlers.NewInsuranceHandler(insuranceAdapter)

	geolocationHandler := handlers.NewGeolocationHandler(geolocationProvider)

	mapsHandler := handlers.NewMapsHandler(cfg.Geolocation.APIKey, cacheProvider)
	feedbackHandler := handlers.NewFeedbackHandler(feedbackService, cacheProvider)

	// Initialize cache middleware
	var cacheMiddleware *middleware.CacheMiddleware
	if cacheProvider != nil {
		cacheMiddleware = middleware.NewCacheMiddleware(cacheProvider)
		log.Println("Cache middleware initialized successfully")
	}

	// Set up router

	router := routes.NewRouter(
		facilityHandler,
		appointmentHandler,
		procedureHandler,
		insuranceHandler,
		geolocationHandler,
		mapsHandler,
		feedbackHandler,
		cacheMiddleware,
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

	// Close event bus
	if eventBus != nil {
		if err := eventBus.Close(); err != nil {
			log.Printf("Error closing event bus: %v", err)
		}
	}

	// Stop cache invalidation service
	if cacheInvalidationService != nil {
		cacheInvalidationService.Stop()
	}

	log.Println("Server stopped")

	// Suppress unused variable warnings (would be used in production)
	_ = geolocationProvider
}

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	redislib "github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
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
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/providerapi"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/redis"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/typesense"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/observability"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/secrets"
)

func main() {

	// Load secrets from Vault (optional)
	vaultCfg := secrets.LoadVaultConfigFromEnv(os.Getenv("VAULT_API_PATH"))
	if vaultCfg.Enabled {
		vaultResult, err := secrets.ApplyVaultSecrets(context.Background(), vaultCfg)
		if err != nil {
			log.Warn().Err(err).Msg("Vault secrets not loaded")
		} else {
			log.Info().
				Str("path", vaultResult.Path).
				Int("loaded", vaultResult.Loaded).
				Int("skipped", vaultResult.Skipped).
				Msg("Vault secrets loaded")
		}
	}

	// Load configuration

	cfg, err := config.Load()

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Initialize structured logging
	env := os.Getenv("ENV")
	if env == "" {
		env = "production"
	}
	observability.InitLogger(cfg.OTEL.ServiceName, env)

	log.Info().
		Str("service", cfg.OTEL.ServiceName).
		Str("version", cfg.OTEL.ServiceVersion).
		Str("env", env).
		Msg("Starting Patient Price Discovery API")

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
			log.Warn().Err(err).Msg("Failed to set up OpenTelemetry")
		} else {
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := shutdown(ctx); err != nil {
					log.Error().Err(err).Msg("Error shutting down OpenTelemetry")
				}
			}()
			log.Info().Msg("OpenTelemetry initialized successfully")
		}
	}

	// Initialize metrics
	metrics, err := observability.InitMetrics()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize metrics")
	}

	// Initialize database client
	pgClient, err := postgres.NewClient(&cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize PostgreSQL client")
	}
	defer pgClient.Close()
	log.Info().Msg("PostgreSQL client initialized successfully")

	// Initialize Redis client
	redisClient, err := redis.NewClient(&cfg.Redis)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize Redis client")
		// Continue without Redis - the application can work without caching
	} else {
		defer redisClient.Close()
		log.Info().Msg("Redis client initialized successfully")
	}

	var cacheProvider providers.CacheProvider
	if redisClient != nil {
		cacheProvider = cache.NewRedisAdapter(redisClient)
	}

	// Initialize Typesense client
	typesenseClient, err := typesense.NewClient(&cfg.Typesense)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize Typesense client")
	} else {
		log.Info().Msg("Typesense client initialized successfully")
	}

	// Initialize Provider API client
	var providerClient providerapi.Client
	if cfg.ProviderAPI.BaseURL != "" {
		providerClient = providerapi.NewClientWithTimeout(
			cfg.ProviderAPI.BaseURL,
			time.Duration(cfg.ProviderAPI.TimeoutSeconds)*time.Second,
		)
		log.Info().Msg("Provider API client initialized successfully")
	}

	// Initialize adapters (facility adapter will be created after cache provider)
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

			log.Warn().Err(err).Msg("Failed to init Typesense schema")

		}

		searchRepo = adapter

	}

	// Initialize facility adapter with caching (now that cacheProvider is available)
	baseFacilityAdapter := database.NewFacilityAdapter(pgClient)
	var facilityAdapter repositories.FacilityRepository
	if cacheProvider != nil {
		facilityAdapter = database.NewCachedFacilityAdapter(baseFacilityAdapter, cacheProvider)
		log.Info().Msg("Facility adapter wrapped with caching layer")
	} else {
		facilityAdapter = baseFacilityAdapter
		log.Warn().Msg("Facility adapter running without cache (Redis unavailable)")
	}

	// Initialize event bus for real-time updates
	var eventBus providers.EventBus
	if redisClient != nil {
		eventBus = events.NewRedisEventBus(redisClient)
		log.Info().Msg("Event bus initialized successfully")
	} else {
		log.Warn().Msg("Event bus disabled (Redis not available)")
	}

	var geolocationProvider providers.GeolocationProvider
	switch cfg.Geolocation.Provider {
	case "google":
		if cfg.Geolocation.APIKey == "" {
			log.Warn().Msg("GEOLOCATION_API_KEY is not set; using mock geolocation provider")
			geolocationProvider = geolocation.NewMockGeolocationProvider()
		} else {
			geolocationProvider = geolocation.NewGoogleGeolocationProvider(cfg.Geolocation.APIKey, cacheProvider)
		}
	default:
		geolocationProvider = geolocation.NewMockGeolocationProvider()
	}

	calendlyAPIKey := strings.TrimSpace(os.Getenv("CALENDLY_API_KEY"))
	allowMockScheduling := strings.EqualFold(os.Getenv("ALLOW_MOCK_SCHEDULING"), "true") || calendlyAPIKey == ""
	if calendlyAPIKey == "" {
		log.Warn().Msg("CALENDLY_API_KEY is not set; using mock scheduling provider")
	}
	appointmentProvider := scheduling.NewAppointmentProvider(scheduling.AppointmentProviderConfig{
		CalendlyAPIKey:         calendlyAPIKey,
		AllowMockFallback:      allowMockScheduling,
		AllowMissingExternalID: allowMockScheduling,
	})

	var enrichmentProvider providers.ProcedureEnrichmentProvider
	if cfg.OpenAI.APIKey == "" {
		log.Warn().Msg("OPENAI_API_KEY is not set; procedure enrichment disabled")
	} else {
		openaiClient, err := openai.NewClient(&cfg.OpenAI)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to initialize OpenAI client")
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
		log.Info().Msg("Event bus configured for facility service")
	}

	// Initialize cache invalidation service
	var cacheInvalidationService *services.CacheInvalidationService
	if cacheProvider != nil && eventBus != nil {
		cacheInvalidationService = services.NewCacheInvalidationService(cacheProvider, eventBus)
		if err := cacheInvalidationService.Start(); err != nil {
			log.Warn().Err(err).Msg("Failed to start cache invalidation service")
		} else {
			log.Info().Msg("Cache invalidation service started successfully")
		}
	}

	appointmentService := services.NewAppointmentService(appointmentAdapter, facilityAdapter, appointmentProvider, allowMockScheduling)
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
		log.Info().Msg("Cache warming service started (refreshes every 5 minutes)")
	}

	// Initialize handlers

	facilityHandler := handlers.NewFacilityHandler(facilityService)

	appointmentHandler := handlers.NewAppointmentHandler(appointmentService)

	procedureHandler := handlers.NewProcedureHandler(procedureAdapter, procedureEnrichmentService)

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
		procedureEnrichmentAdapter,
		enrichmentProvider,
		geolocationProvider,
		cacheProvider,
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

	// Initialize cache middleware
	var cacheMiddleware *middleware.CacheMiddleware
	if cacheProvider != nil {
		cacheMiddleware = middleware.NewCacheMiddleware(cacheProvider)
		log.Info().Msg("Cache middleware initialized successfully")
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
		log.Info().Str("address", serverAddr).Msg("Server starting")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed to start")
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
					log.Info().
						Str("provider_id", providerID).
						Int("facilities_created", summary.FacilitiesCreated).
						Int("procedures_created", summary.ProceduresCreated).
						Msg("Provider ingestion completed")
					return
				}
				log.Warn().
					Err(err).
					Int("attempt", attempt).
					Str("provider_id", providerID).
					Msg("Provider ingestion failed")
				time.Sleep(5 * time.Second)
			}
		}()
	}

	// Optional: periodic provider ingestion
	ingestIntervalMinutes := 0
	if value := strings.TrimSpace(os.Getenv("PROVIDER_INGEST_INTERVAL_MINUTES")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			ingestIntervalMinutes = parsed
		}
	}
	if ingestIntervalMinutes > 0 && providerClient != nil {
		providerID := strings.TrimSpace(os.Getenv("PROVIDER_INGEST_PROVIDER_ID"))
		timeoutSeconds := 120
		if value := strings.TrimSpace(os.Getenv("PROVIDER_INGEST_TIMEOUT_SECONDS")); value != "" {
			if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
				timeoutSeconds = parsed
			}
		}

		var ingestRunning atomic.Bool
		interval := time.Duration(ingestIntervalMinutes) * time.Minute
		ticker := time.NewTicker(interval)
		go func() {
			for {
				select {
				case <-ctx.Done():
					ticker.Stop()
					return
				case <-ticker.C:
					if !ingestRunning.CompareAndSwap(false, true) {
						log.Warn().Msg("Provider ingestion already running; skipping scheduled run")
						continue
					}
					go func() {
						defer ingestRunning.Store(false)
						ingestCtx, ingestCancel := context.WithTimeout(context.Background(), time.Duration(timeoutSeconds)*time.Second)
						defer ingestCancel()
						summary, err := ingestionService.SyncCurrentData(ingestCtx, providerID)
						if err != nil {
							log.Error().Err(err).Str("provider_id", providerID).Msg("Provider ingestion failed")
							return
						}
						log.Info().
							Str("provider_id", providerID).
							Int("facilities_created", summary.FacilitiesCreated).
							Int("procedures_created", summary.ProceduresCreated).
							Msg("Provider ingestion completed")
					}()
				}
			}
		}()
		log.Info().Int("interval_minutes", ingestIntervalMinutes).Msg("Provider ingestion scheduled")
	}

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Server shutting down...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Error during server shutdown")
	}

	// Close event bus
	if eventBus != nil {
		if err := eventBus.Close(); err != nil {
			log.Error().Err(err).Msg("Error closing event bus")
		}
	}

	// Stop cache invalidation service
	if cacheInvalidationService != nil {
		cacheInvalidationService.Stop()
	}

	log.Info().Msg("Server stopped")

	// Suppress unused variable warnings (would be used in production)
	_ = geolocationProvider
}

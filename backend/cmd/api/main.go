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

	       "github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/providers/geolocation"

	              "github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/search"

	              "github.com/zatekoja/Patientpricediscoverydesign/backend/internal/application/services"

	              "github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/handlers"

	              "github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/routes"

	              "github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"

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
		              facilityAdapter := database.NewFacilityAdapter(pgClient)
		              
		              var searchRepo repositories.FacilitySearchRepository
		              if typesenseClient != nil {
		                      adapter := search.NewTypesenseAdapter(typesenseClient)
		                      // Ensure schema exists
		                      if err := adapter.InitSchema(context.Background()); err != nil {
		                              log.Printf("Warning: Failed to init Typesense schema: %v", err)
		                      }
		                      searchRepo = adapter
		              }
		       
		              var cacheAdapter cache.RedisAdapter
		              if redisClient != nil {
		                      cacheAdapter = *cache.NewRedisAdapter(redisClient).(*cache.RedisAdapter)
		              }
		              geolocationProvider := geolocation.NewMockGeolocationProvider()
		       
		              // Initialize services
		              facilityService := services.NewFacilityService(facilityAdapter, searchRepo)
		       		       // Initialize handlers
		       facilityHandler := handlers.NewFacilityHandler(facilityService)
		
		       // Set up router
		       router := routes.NewRouter(facilityHandler, metrics)
		
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

	log.Println("Server stopped")

	// Suppress unused variable warnings (would be used in production)
	_ = cacheAdapter
	_ = geolocationProvider
}

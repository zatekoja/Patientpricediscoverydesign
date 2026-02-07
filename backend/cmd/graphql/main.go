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

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/cache"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/database"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/search"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/graphql/generated"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/graphql/resolvers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/postgres"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/redis"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/typesense"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/observability"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/query/adapters"
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
			cfg.OTEL.ServiceName+"-graphql",
			cfg.OTEL.ServiceVersion,
			cfg.OTEL.Endpoint,
		)
		if err != nil {
			log.Printf("Warning: Failed to set up OpenTelemetry: %v", err)
		} else {
			defer func() {
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer shutdownCancel()
				if err := shutdown(shutdownCtx); err != nil {
					log.Printf("Error shutting down OpenTelemetry: %v", err)
				}
			}()
			log.Println("OpenTelemetry initialized successfully")
		}
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
	} else {
		defer redisClient.Close()
		log.Println("Redis client initialized successfully")
	}

	// Initialize Typesense client
	typesenseClient, err := typesense.NewClient(&cfg.Typesense)
	if err != nil {
		log.Fatalf("Failed to initialize Typesense client: %v", err)
	}
	log.Println("Typesense client initialized successfully")

	// Initialize adapters
	facilityDBAdapter := database.NewFacilityAdapter(pgClient)

	// Create search adapter (Typesense)
	searchAdapter := search.NewTypesenseAdapter(typesenseClient)

	// Initialize cache adapter with QueryCacheProvider wrapper
	var queryCacheProvider adapters.QueryCacheAdapter
	if redisClient != nil {
		domainCacheProvider := cache.NewRedisAdapter(redisClient)
		queryCacheProvider = *adapters.NewQueryCacheAdapter(domainCacheProvider)
	}

	// Initialize GraphQL resolver with dependencies
	resolver := resolvers.NewResolver(
		searchAdapter,
		facilityDBAdapter,
		&queryCacheProvider,
	)

	// Create GraphQL server
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))

	// Set up HTTP routes
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"graphql"}`))
	})

	// GraphQL endpoint
	mux.Handle("/graphql", srv)

	// Playground endpoint (dev only)
	if os.Getenv("ENV") != "production" {
		mux.Handle("/playground", playground.Handler("GraphQL Playground", "/graphql"))
		log.Println("🚀 GraphQL Playground available at http://localhost:8081/playground")
	}

	// Create HTTP server
	port := 8081
	serverAddr := fmt.Sprintf(":%d", port)
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("🚀 GraphQL server starting on %s", serverAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("GraphQL server shutting down...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error during server shutdown: %v", err)
	}

	log.Println("GraphQL server stopped")
}

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
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/cache"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/database"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/search"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/middleware"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/application/services"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/graphql/generated"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/graphql/loaders"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/graphql/resolvers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/postgres"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/providerapi"
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

	// Initialize Provider API client
	var providerClient providerapi.Client
	if cfg.ProviderAPI.BaseURL != "" {
		providerClient = providerapi.NewClient(cfg.ProviderAPI.BaseURL)
		log.Println("Provider API client initialized successfully")
	}

	// Initialize adapters
	baseFacilityDBAdapter := database.NewFacilityAdapter(pgClient)

	// Wrap facility adapter with caching for read performance optimization
	var facilityDBAdapter repositories.FacilityRepository
	if redisClient != nil {
		domainCacheProvider := cache.NewRedisAdapter(redisClient)
		facilityDBAdapter = database.NewCachedFacilityAdapter(baseFacilityDBAdapter, domainCacheProvider)
		log.Println("✓ GraphQL: Facility adapter wrapped with caching layer")
	} else {
		facilityDBAdapter = baseFacilityDBAdapter
		log.Println("⚠ GraphQL: Facility adapter running without cache (Redis unavailable)")
	}

	appointmentDBAdapter := database.NewAppointmentAdapter(pgClient)
	procedureDBAdapter := database.NewProcedureAdapter(pgClient)
	facilityProcedureDBAdapter := database.NewFacilityProcedureAdapter(pgClient)
	insuranceDBAdapter := database.NewInsuranceAdapter(pgClient)

	// Create search adapter (Typesense)
	searchAdapter := search.NewTypesenseAdapter(typesenseClient)

	// Initialize cache adapter with QueryCacheProvider wrapper
	var queryCacheProvider adapters.QueryCacheAdapter
	if redisClient != nil {
		domainCacheProvider := cache.NewRedisAdapter(redisClient)
		queryCacheProvider = *adapters.NewQueryCacheAdapter(domainCacheProvider)

		// Start cache warming service for improved read performance
		warmingService := services.NewCacheWarmingService(
			facilityDBAdapter, // Use cached adapter
			domainCacheProvider,
		)
		go warmingService.StartPeriodicWarming(ctx, 5*time.Minute)
		log.Println("✓ GraphQL: Cache warming service started (refreshes every 5 minutes)")
	}

	// Initialize GraphQL resolver with dependencies
	resolver := resolvers.NewResolver(
		searchAdapter,
		facilityDBAdapter,
		appointmentDBAdapter,
		procedureDBAdapter,
		facilityProcedureDBAdapter,
		insuranceDBAdapter,
		&queryCacheProvider,
		providerClient,
	)

	// Create GraphQL server
	srv := handler.New(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))

	// Configure transports
	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.MultipartForm{})

	// Set up Query Cache (LRU)
	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	// Enable Introspection (disable in production if needed)
	srv.Use(extension.Introspection{})

	/*
		// Automatic Persisted Queries (APQ)
		srv.Use(extension.AutomaticPersistedQueries{
			Cache: lru.New[string](100),
		})
	*/

	// Set up HTTP routes
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"graphql"}`))
	})

	// Create DataLoader middleware
	loaderMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ldrs := loaders.NewLoaders(facilityDBAdapter, procedureDBAdapter)
			ctx := loaders.WithLoaders(r.Context(), ldrs)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}

	// Apply middleware: Performance -> Logging -> CORS -> DataLoader
	httpHandler := middleware.Compression( // Apply gzip compression
		middleware.CacheControl( // Apply cache headers
			middleware.LoggingMiddleware(
				middleware.CORSMiddleware(
					loaderMiddleware(srv),
				),
			),
		),
	)

	// GraphQL endpoint
	mux.Handle("/graphql", httpHandler)

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

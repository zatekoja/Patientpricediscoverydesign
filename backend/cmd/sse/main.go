package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/events"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/handlers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/middleware"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/redis"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/observability"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
)

func main() {
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
	observability.InitLogger(cfg.OTEL.ServiceName+"-sse", env)

	log.Info().
		Str("service", cfg.OTEL.ServiceName+"-sse").
		Str("version", cfg.OTEL.ServiceVersion).
		Str("env", env).
		Msg("Starting SSE Server")

	// Initialize Redis client (required for SSE and caching)
	redisClient, err := redis.NewClient(&cfg.Redis)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize Redis client")
	}
	defer redisClient.Close()
	log.Info().Msg("Redis client initialized successfully")

	// Initialize event bus for real-time updates
	eventBus := events.NewRedisEventBus(redisClient)
	log.Info().Msg("Event bus initialized successfully")

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize OpenTelemetry if enabled
	var shutdown func(context.Context) error
	if cfg.OTEL.Enabled && cfg.OTEL.Endpoint != "" {
		shutdown, err = observability.Setup(
			ctx,
			cfg.OTEL.ServiceName+"-sse",
			cfg.OTEL.ServiceVersion,
			cfg.OTEL.Endpoint,
		)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to set up OpenTelemetry")
		} else {
			defer func() {
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer shutdownCancel()
				if err := shutdown(shutdownCtx); err != nil {
					log.Error().Err(err).Msg("Error shutting down OpenTelemetry")
				}
			}()
			log.Info().Msg("OpenTelemetry initialized successfully")
		}
	}

	// Initialize metrics
	metrics, err := observability.InitMetrics()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize metrics")
	}

	// Initialize SSE handler
	sseHandler := handlers.NewSSEHandler(eventBus)
	log.Info().Msg("SSE handler initialized successfully")

	// Register SSE metrics callback
	if metrics != nil {
		err = metrics.RegisterSSECallback(func() int64 {
			return int64(sseHandler.GetClientCount())
		})
		if err != nil {
			log.Warn().Err(err).Msg("Failed to register SSE metrics callback")
		}
	}

	// Set up router
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// SSE streaming endpoints
	mux.HandleFunc("GET /api/stream/facilities/{id}", sseHandler.StreamFacilityUpdates)
	mux.HandleFunc("GET /api/stream/facilities/region", sseHandler.StreamRegionalUpdates)

	// SSE stats endpoint
	mux.HandleFunc("GET /api/stream/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"connected_clients": %d}`, sseHandler.GetClientCount())
	})

	// Apply middleware
	var handler http.Handler = mux
	handler = middleware.CORSMiddleware(handler)
	handler = middleware.LoggingMiddleware(handler)

	if metrics != nil {
		handler = middleware.ObservabilityMiddleware(metrics)(handler)
	}

	// Create HTTP server
	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,  // Longer timeout for SSE
		WriteTimeout: 0,                 // No timeout for SSE streaming
		IdleTimeout:  120 * time.Second, // Allow long-lived connections
	}

	// Start server in a goroutine
	go func() {
		log.Info().Str("address", serverAddr).Msg("SSE Server starting")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("SSE Server failed to start")
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("SSE Server shutting down...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Error during server shutdown")
	}

	// Close event bus
	if err := eventBus.Close(); err != nil {
		log.Error().Err(err).Msg("Error closing event bus")
	}

	log.Info().Msg("SSE Server stopped")
}

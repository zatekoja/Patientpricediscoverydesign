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

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/events"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/handlers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/middleware"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/redis"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Println("Starting SSE Server...")

	// Initialize Redis client (required for SSE and caching)
	redisClient, err := redis.NewClient(&cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to initialize Redis client: %v", err)
	}
	defer redisClient.Close()
	log.Println("Redis client initialized successfully")

	// Initialize event bus for real-time updates
	eventBus := events.NewRedisEventBus(redisClient)
	log.Println("Event bus initialized successfully")

	// Initialize SSE handler
	sseHandler := handlers.NewSSEHandler(eventBus)
	log.Println("SSE handler initialized successfully")

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
		log.Printf("SSE Server starting on %s", serverAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("SSE Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("SSE Server shutting down...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error during server shutdown: %v", err)
	}

	// Close event bus
	if err := eventBus.Close(); err != nil {
		log.Printf("Error closing event bus: %v", err)
	}

	log.Println("SSE Server stopped")
}

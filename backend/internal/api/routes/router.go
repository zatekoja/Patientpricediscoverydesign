package routes

import (
	"net/http"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/handlers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/middleware"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/observability"
)

// Router holds all route handlers
type Router struct {
	mux             *http.ServeMux
	facilityHandler *handlers.FacilityHandler
	metrics         *observability.Metrics
}

// NewRouter creates a new router
func NewRouter(
	facilityHandler *handlers.FacilityHandler,
	metrics *observability.Metrics,
) *Router {
	return &Router{
		mux:             http.NewServeMux(),
		facilityHandler: facilityHandler,
		metrics:         metrics,
	}
}

// SetupRoutes configures all application routes
func (r *Router) SetupRoutes() http.Handler {
	// Health check endpoint
	r.mux.HandleFunc("GET /health", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Facility endpoints
	r.mux.HandleFunc("GET /api/facilities", r.facilityHandler.ListFacilities)
	r.mux.HandleFunc("GET /api/facilities/search", r.facilityHandler.SearchFacilities)
	r.mux.HandleFunc("GET /api/facilities/{id}", r.facilityHandler.GetFacility)

	// Apply middleware in reverse order (last middleware wraps first)
	var handler http.Handler = r.mux
	handler = middleware.CORSMiddleware(handler)
	handler = middleware.LoggingMiddleware(handler)
	handler = middleware.ObservabilityMiddleware(r.metrics)(handler)

	return handler
}

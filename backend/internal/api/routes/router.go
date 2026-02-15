package routes

import (
	"net/http"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/handlers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/middleware"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/observability"
)

// Router holds all route handlers

type Router struct {
	mux *http.ServeMux

	facilityHandler *handlers.FacilityHandler

	appointmentHandler *handlers.AppointmentHandler

	procedureHandler *handlers.ProcedureHandler

	insuranceHandler *handlers.InsuranceHandler

	geolocationHandler *handlers.GeolocationHandler

	mapsHandler     *handlers.MapsHandler
	feedbackHandler *handlers.FeedbackHandler

	providerPriceHandler     *handlers.ProviderPriceHandler
	providerIngestionHandler *handlers.ProviderIngestionHandler

	calendlyWebhookHandler *handlers.CalendlyWebhookHandler
	feeWaiverHandler       *handlers.FeeWaiverHandler

	cacheMiddleware *middleware.CacheMiddleware
	metrics         *observability.Metrics
}

// NewRouter creates a new router

func NewRouter(

	facilityHandler *handlers.FacilityHandler,

	appointmentHandler *handlers.AppointmentHandler,

	procedureHandler *handlers.ProcedureHandler,

	insuranceHandler *handlers.InsuranceHandler,

	geolocationHandler *handlers.GeolocationHandler,

	mapsHandler *handlers.MapsHandler,
	feedbackHandler *handlers.FeedbackHandler,
	cacheMiddleware *middleware.CacheMiddleware,

	providerPriceHandler *handlers.ProviderPriceHandler,
	providerIngestionHandler *handlers.ProviderIngestionHandler,

	calendlyWebhookHandler *handlers.CalendlyWebhookHandler,
	feeWaiverHandler *handlers.FeeWaiverHandler,

	metrics *observability.Metrics,

) *Router {

	return &Router{

		mux: http.NewServeMux(),

		facilityHandler: facilityHandler,

		appointmentHandler: appointmentHandler,

		procedureHandler: procedureHandler,

		insuranceHandler: insuranceHandler,

		geolocationHandler: geolocationHandler,

		mapsHandler:     mapsHandler,
		feedbackHandler: feedbackHandler,

		providerPriceHandler:     providerPriceHandler,
		providerIngestionHandler: providerIngestionHandler,

		calendlyWebhookHandler: calendlyWebhookHandler,
		feeWaiverHandler:       feeWaiverHandler,

		cacheMiddleware: cacheMiddleware,
		metrics:         metrics,
	}

}

// SetupRoutes configures all application routes

func (r *Router) SetupRoutes() http.Handler {

	// Health check endpoint

	r.mux.HandleFunc("GET /health", func(w http.ResponseWriter, req *http.Request) {

		w.WriteHeader(http.StatusOK)

		if _, err := w.Write([]byte("OK")); err != nil {
			return
		}

	})

	// Facility endpoints

	r.mux.HandleFunc("GET /api/facilities", r.facilityHandler.ListFacilities)

	r.mux.HandleFunc("GET /api/facilities/search", r.facilityHandler.SearchFacilities)

	r.mux.HandleFunc("GET /api/facilities/suggest", r.facilityHandler.SuggestFacilities)

	r.mux.HandleFunc("GET /api/facilities/{id}", r.facilityHandler.GetFacility)

	r.mux.HandleFunc("GET /api/facilities/{id}/services", r.facilityHandler.GetFacilityServices)
	r.mux.HandleFunc("GET /api/facilities/{id}/service-fees", r.facilityHandler.GetFacilityServiceFees)

	r.mux.HandleFunc("PATCH /api/facilities/{id}", r.facilityHandler.UpdateFacility)
	r.mux.HandleFunc("PATCH /api/facilities/{id}/services/{procedureId}", r.facilityHandler.UpdateServiceAvailability)

	// Appointment endpoints

	r.mux.HandleFunc("POST /api/appointments", r.appointmentHandler.BookAppointment)

	r.mux.HandleFunc("GET /api/facilities/{id}/availability", r.appointmentHandler.GetAvailability)

	// Procedure endpoints

	r.mux.HandleFunc("GET /api/procedures", r.procedureHandler.ListProcedures)

	r.mux.HandleFunc("GET /api/procedures/{id}", r.procedureHandler.GetProcedure)
	r.mux.HandleFunc("GET /api/procedures/{id}/enrichment", r.procedureHandler.GetProcedureEnrichment)

	// Insurance endpoints

	r.mux.HandleFunc("GET /api/insurance-providers", r.insuranceHandler.ListInsuranceProviders)

	r.mux.HandleFunc("GET /api/insurance-providers/{id}", r.insuranceHandler.GetInsuranceProvider)

	// Geolocation endpoints

	r.mux.HandleFunc("GET /api/geocode", r.geolocationHandler.Geocode)

	r.mux.HandleFunc("GET /api/reverse-geocode", r.geolocationHandler.ReverseGeocode)

	// Maps endpoints

	r.mux.HandleFunc("GET /api/maps/static", r.mapsHandler.GetStaticMap)

	// Feedback endpoints

	r.mux.HandleFunc("POST /api/feedback", r.feedbackHandler.SubmitFeedback)

	// Provider data endpoints

	r.mux.HandleFunc("GET /api/provider/prices/current", r.providerPriceHandler.GetCurrentData)

	r.mux.HandleFunc("GET /api/provider/prices/previous", r.providerPriceHandler.GetPreviousData)

	r.mux.HandleFunc("GET /api/provider/prices/historical", r.providerPriceHandler.GetHistoricalData)

	r.mux.HandleFunc("GET /api/provider/health", r.providerPriceHandler.GetProviderHealth)

	r.mux.HandleFunc("GET /api/provider/list", r.providerPriceHandler.ListProviders)

	r.mux.HandleFunc("POST /api/provider/sync/trigger", r.providerPriceHandler.TriggerSync)

	r.mux.HandleFunc("GET /api/provider/sync/status", r.providerPriceHandler.GetSyncStatus)

	// Provider ingestion endpoint (hydrate core DB from provider API)

	r.mux.HandleFunc("POST /api/provider/ingest", r.providerIngestionHandler.TriggerIngestion)

	// Analytics endpoints
	r.mux.HandleFunc("GET /api/analytics/zero-result-queries", r.facilityHandler.GetZeroResultQueries)

	// Fee waiver endpoints
	if r.feeWaiverHandler != nil {
		r.mux.HandleFunc("GET /api/facilities/{id}/fee-waiver", r.feeWaiverHandler.GetFacilityFeeWaiver)
		r.mux.HandleFunc("POST /api/admin/fee-waivers", r.feeWaiverHandler.CreateFeeWaiver)
	}

	// Calendly webhook endpoint for appointment notifications
	if r.calendlyWebhookHandler != nil {
		r.mux.HandleFunc("POST /webhooks/calendly", r.calendlyWebhookHandler.HandleWebhook)
	}

	// Apply middleware in reverse order (last middleware wraps first)
	// CORS must be outermost so cached responses also get CORS headers.

	var handler http.Handler = r.mux
	handler = middleware.LoggingMiddleware(handler)

	// Apply cache middleware if available
	if r.cacheMiddleware != nil {
		handler = r.cacheMiddleware.Middleware(handler)
	}

	handler = middleware.ObservabilityMiddleware(r.metrics)(handler)

	// Apply HTTP performance optimizations (compression, ETag, cache headers)
	handler = middleware.ResponseOptimization(handler)

	// CORS wraps everything so headers are set even on cache HITs
	handler = middleware.CORSMiddleware(handler)

	return handler
}

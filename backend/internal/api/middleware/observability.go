package middleware

import (
	"net/http"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/observability"
	"go.opentelemetry.io/otel/attribute"
)

// ObservabilityMiddleware adds OpenTelemetry tracing and metrics to HTTP requests
func ObservabilityMiddleware(metrics *observability.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use route pattern instead of raw path to avoid high cardinality
			route := r.Pattern
			if route == "" {
				route = r.URL.Path
			}

			// Start a new span
			ctx, span := observability.StartSpan(r.Context(), route)
			defer span.End()

			// Add request attributes to span
			observability.SetSpanAttributes(span,
				attribute.String("http.method", r.Method),
				attribute.String("http.route", route),
				attribute.String("http.user_agent", r.UserAgent()),
			)

			// Create a response writer wrapper to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Record start time
			start := time.Now()

			// Call the next handler
			next.ServeHTTP(rw, r.WithContext(ctx))

			// Record metrics
			duration := time.Since(start)
			observability.RecordRequestMetric(ctx, metrics, r.Method, route, rw.statusCode, duration)

			// Add status code to span
			observability.SetSpanAttributes(span, attribute.Int("http.status_code", rw.statusCode))
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

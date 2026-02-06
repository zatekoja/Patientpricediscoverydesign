package middleware

import (
	"net/http"
	"os"
	"strings"
)

// getAllowedOrigins returns the list of allowed origins from environment or defaults
func getAllowedOrigins() []string {
	allowedOriginsEnv := os.Getenv("ALLOWED_ORIGINS")
	if allowedOriginsEnv != "" {
		return strings.Split(allowedOriginsEnv, ",")
	}
	// Default to wildcard only in development
	// In production, this should be configured via environment variable
	return []string{"*"}
}

// isAllowedOrigin checks if an origin is in the allowed list
func isAllowedOrigin(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

// CORSMiddleware adds CORS headers to HTTP responses
func CORSMiddleware(next http.Handler) http.Handler {
	allowedOrigins := getAllowedOrigins()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Set CORS headers based on allowed origins
		if origin != "" && isAllowedOrigin(origin, allowedOrigins) {
			if allowedOrigins[0] == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
			}
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

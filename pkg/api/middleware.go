package api

import (
	"context"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"
)

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if request ID already exists in header
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = uuid.New().String()
			}

			// Store in context
			ctx := context.WithValue(r.Context(), "request_id", requestID)

			// Add to response headers
			w.Header().Set("X-Request-ID", requestID)

			// Continue with updated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// LoggingMiddleware logs HTTP requests and responses
func LoggingMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create response writer wrapper to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Log request
			requestID := GetRequestID(r)
			log.Printf("[%s] %s %s - Started", requestID, r.Method, r.URL.Path)

			// Process request
			next.ServeHTTP(wrapped, r)

			// Log response
			duration := time.Since(start)
			log.Printf("[%s] %s %s - Completed %d in %v",
				requestID, r.Method, r.URL.Path, wrapped.statusCode, duration)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	if !rw.written {
		rw.statusCode = statusCode
		rw.written = true
		rw.ResponseWriter.WriteHeader(statusCode)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns a permissive CORS configuration for development
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders: []string{"X-Request-ID"},
		MaxAge:         3600,
	}
}

// CORSMiddleware handles Cross-Origin Resource Sharing
func CORSMiddleware(config CORSConfig) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			if origin != "" && isOriginAllowed(origin, config.AllowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)

				if config.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}

				if len(config.ExposedHeaders) > 0 {
					w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
				}
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
				if config.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", string(rune(config.MaxAge)))
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isOriginAllowed checks if an origin is in the allowed list
func isOriginAllowed(origin string, allowed []string) bool {
	for _, o := range allowed {
		if o == "*" || o == origin {
			return true
		}
		// Support wildcard subdomains like "*.example.com"
		if strings.HasPrefix(o, "*.") {
			suffix := o[1:] // Remove leading '*'
			if strings.HasSuffix(origin, suffix) {
				return true
			}
		}
	}
	return false
}

// RecoveryMiddleware recovers from panics and returns 500 errors
func RecoveryMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					requestID := GetRequestID(r)

					// Log the panic
					log.Printf("[%s] PANIC: %v\n%s", requestID, err, debug.Stack())

					// Return 500 error
					apiErr := InternalServerError("internal server error occurred")
					WriteError(w, r, apiErr)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// AuthMiddleware placeholder for future authentication (T027)
// This will be implemented in the next task with API key validation
func AuthMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO (T027): Implement API key authentication
			// For now, pass through all requests
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitMiddleware placeholder for future rate limiting
// This will be implemented as part of T031 (Rate Limiting)
func RateLimitMiddleware(requestsPerMinute int) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// TODO (T031): Implement rate limiting with token bucket or similar
			// For now, pass through all requests
			next.ServeHTTP(w, r)
		})
	}
}

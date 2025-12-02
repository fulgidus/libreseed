package api

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

// Router handles HTTP routing for the API
type Router struct {
	mux        *http.ServeMux
	middleware []Middleware
	startTime  time.Time
	version    string
}

// Middleware is a function that wraps an http.Handler
type Middleware func(http.Handler) http.Handler

// NewRouter creates a new API router
func NewRouter(version string) *Router {
	return &Router{
		mux:       http.NewServeMux(),
		startTime: time.Now(),
		version:   version,
	}
}

// Use adds middleware to the router
func (r *Router) Use(middleware Middleware) {
	r.middleware = append(r.middleware, middleware)
}

// Handle registers a handler for a specific pattern
func (r *Router) Handle(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc(pattern, handler)
}

// ServeHTTP implements http.Handler
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Build middleware chain
	handler := http.Handler(r.mux)

	// Apply middleware in reverse order (last added = outermost)
	for i := len(r.middleware) - 1; i >= 0; i-- {
		handler = r.middleware[i](handler)
	}

	handler.ServeHTTP(w, req)
}

// RegisterRoutes registers all API routes
func (r *Router) RegisterRoutes() {
	// Health check endpoint
	r.Handle("/api/v1/health", r.handleHealth())

	// Version info endpoint
	r.Handle("/api/v1/version", r.handleVersion())

	// TODO: Package management routes (T028)
	// r.Handle("/api/v1/packages", r.handlePackagesList())
	// r.Handle("/api/v1/packages/{id}", r.handlePackageGet())

	// TODO: Statistics routes (T029)
	// r.Handle("/api/v1/stats", r.handleStats())

	// TODO: DHT routes (T030)
	// r.Handle("/api/v1/dht/stats", r.handleDHTStats())
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string        `json:"status"`
	Uptime    time.Duration `json:"uptime_seconds"`
	Version   string        `json:"version"`
	Timestamp time.Time     `json:"timestamp"`
}

// handleHealth returns the health check handler
func (r *Router) handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			WriteError(w, req, BadRequest("method not allowed"))
			return
		}

		uptime := time.Since(r.startTime)

		response := HealthResponse{
			Status:    "ok",
			Uptime:    uptime,
			Version:   r.version,
			Timestamp: time.Now(),
		}

		WriteSuccess(w, response)
	}
}

// VersionResponse represents the version info response
type VersionResponse struct {
	Version   string `json:"version"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
	BuildTime string `json:"build_time,omitempty"`
}

// handleVersion returns the version info handler
func (r *Router) handleVersion() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			WriteError(w, req, BadRequest("method not allowed"))
			return
		}

		response := VersionResponse{
			Version:   r.version,
			GoVersion: runtime.Version(),
			Platform:  runtime.GOOS + "/" + runtime.GOARCH,
			// BuildTime will be injected via ldflags in production builds
		}

		WriteSuccess(w, response)
	}
}

// NotFoundHandler returns a 404 handler
func NotFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, r, NotFound("endpoint"))
	}
}

// MethodNotAllowedHandler returns a 405 handler
func MethodNotAllowedHandler(allowed []string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", joinMethods(allowed))
		WriteError(w, r, BadRequest("method not allowed"))
	}
}

// joinMethods joins HTTP methods into a comma-separated string
func joinMethods(methods []string) string {
	if len(methods) == 0 {
		return ""
	}
	result := methods[0]
	for i := 1; i < len(methods); i++ {
		result += ", " + methods[i]
	}
	return result
}

// ParseJSON parses JSON request body into target struct
func ParseJSON(r *http.Request, target interface{}) error {
	if r.Body == nil {
		return BadRequest("missing request body")
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return BadRequest("invalid JSON: " + err.Error())
	}

	return nil
}

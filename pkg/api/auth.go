package api

import (
	"context"
	"log"
	"net/http"
	"strings"
)

// Context keys for storing authentication information
type contextKey string

const (
	APIKeyContextKey contextKey = "api_key"
)

// AuthenticationMiddleware validates API keys and enforces permission levels
// The keyStore parameter allows injecting the store without modifying Router
func AuthenticationMiddleware(keyStore *APIKeyStore, requiredLevel string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Extract Authorization header
			authHeader := req.Header.Get("Authorization")
			if authHeader == "" {
				WriteError(w, req, Unauthorized("missing authorization header"))
				return
			}

			// Check for Bearer token format
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				WriteError(w, req, Unauthorized("invalid authorization format, expected 'Bearer <token>'"))
				return
			}

			plaintextKey := parts[1]

			// Validate the API key
			apiKey, err := keyStore.ValidateKey(plaintextKey)
			if err != nil {
				requestID := GetRequestID(req)
				log.Printf("[%s] Authentication failed: %v", requestID, err)

				switch err {
				case ErrInvalidKeyFormat:
					WriteError(w, req, Unauthorized("invalid API key format"))
				case ErrKeyNotFound:
					WriteError(w, req, Unauthorized("invalid API key"))
				case ErrKeyRevoked:
					WriteError(w, req, Unauthorized("API key has been revoked"))
				default:
					WriteError(w, req, InternalServerError("authentication error"))
				}
				return
			}

			// Check permission level
			if !HasPermission(apiKey.Level, requiredLevel) {
				WriteError(w, req, Forbidden("insufficient permissions"))
				return
			}

			// Update last used timestamp asynchronously to avoid blocking the request
			go func() {
				if err := keyStore.UpdateLastUsed(apiKey.ID); err != nil {
					log.Printf("Failed to update last used timestamp for key %s: %v", apiKey.ID, err)
				}
			}()

			// Inject API key into request context
			ctx := WithAPIKey(req.Context(), apiKey)

			// Continue to next handler
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	}
}

// WithAPIKey adds an API key to the context
func WithAPIKey(ctx context.Context, key *APIKey) context.Context {
	return context.WithValue(ctx, APIKeyContextKey, key)
}

// GetAPIKeyFromContext retrieves the API key from the request context
func GetAPIKeyFromContext(ctx context.Context) *APIKey {
	key, ok := ctx.Value(APIKeyContextKey).(*APIKey)
	if !ok {
		return nil
	}
	return key
}

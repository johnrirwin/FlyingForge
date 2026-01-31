package auth

import (
	"context"
	"net/http"
	"strings"
)

// contextKey is a type for context keys
type contextKey string

const (
	// UserIDKey is the context key for the authenticated user ID
	UserIDKey contextKey = "userId"
)

// Middleware provides authentication middleware for HTTP handlers
type Middleware struct {
	authService *Service
}

// NewMiddleware creates a new auth middleware
func NewMiddleware(authService *Service) *Middleware {
	return &Middleware{authService: authService}
}

// RequireAuth is middleware that requires a valid JWT token
func (m *Middleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			http.Error(w, `{"error":"authorization required"}`, http.StatusUnauthorized)
			return
		}

		userID, err := m.authService.ValidateAccessToken(token)
		if err != nil {
			http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		// Add user ID to context
		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next(w, r.WithContext(ctx))
	}
}

// OptionalAuth is middleware that validates JWT if present but doesn't require it
func (m *Middleware) OptionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token != "" {
			userID, err := m.authService.ValidateAccessToken(token)
			if err == nil {
				ctx := context.WithValue(r.Context(), UserIDKey, userID)
				r = r.WithContext(ctx)
			}
		}
		next(w, r)
	}
}

// GetUserID extracts the user ID from the request context
func GetUserID(ctx context.Context) string {
	userID, _ := ctx.Value(UserIDKey).(string)
	return userID
}

// extractToken extracts the JWT token from the Authorization header
func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return parts[1]
}

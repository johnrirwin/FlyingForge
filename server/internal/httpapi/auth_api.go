package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

// AuthAPI handles authentication HTTP endpoints
type AuthAPI struct {
	authService    *auth.Service
	authMiddleware *auth.Middleware
	logger         *logging.Logger
	frontendURL    string
}

// NewAuthAPI creates a new auth API handler
func NewAuthAPI(authService *auth.Service, authMiddleware *auth.Middleware, logger *logging.Logger) *AuthAPI {
	frontendURL := os.Getenv("AUTH_FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	return &AuthAPI{
		authService:    authService,
		authMiddleware: authMiddleware,
		logger:         logger,
		frontendURL:    frontendURL,
	}
}

// RegisterRoutes registers auth routes on the given mux
func (api *AuthAPI) RegisterRoutes(mux *http.ServeMux, corsMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("/api/auth/google", corsMiddleware(api.handleGoogleLogin))
	mux.HandleFunc("/api/auth/google/callback", api.handleGoogleCallback)
	mux.HandleFunc("/api/auth/refresh", corsMiddleware(api.handleRefresh))
	mux.HandleFunc("/api/auth/logout", corsMiddleware(api.authMiddleware.RequireAuth(api.handleLogout)))
	mux.HandleFunc("/api/auth/me", corsMiddleware(api.authMiddleware.RequireAuth(api.handleGetMe)))
}

func (api *AuthAPI) handleGoogleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params models.GoogleLoginParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		api.writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	response, err := api.authService.LoginWithGoogle(r.Context(), params)
	if err != nil {
		if authErr, ok := err.(*auth.AuthError); ok {
			status := http.StatusUnauthorized
			if authErr.Code == "account_disabled" {
				status = http.StatusForbidden
			}
			api.writeError(w, status, authErr.Code, authErr.Message)
			return
		}
		api.logger.Error("Google login failed", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "google login failed")
		return
	}

	api.writeJSON(w, http.StatusOK, response)
}

func (api *AuthAPI) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		api.writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	if params.RefreshToken == "" {
		api.writeError(w, http.StatusBadRequest, "invalid_request", "refresh token is required")
		return
	}

	tokens, err := api.authService.RefreshTokens(r.Context(), params.RefreshToken)
	if err != nil {
		if authErr, ok := err.(*auth.AuthError); ok {
			api.writeError(w, http.StatusUnauthorized, authErr.Code, authErr.Message)
			return
		}
		api.logger.Error("Token refresh failed", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "refresh failed")
		return
	}

	api.writeJSON(w, http.StatusOK, tokens)
}

func (api *AuthAPI) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		api.writeError(w, http.StatusUnauthorized, "unauthorized", "not authenticated")
		return
	}

	if err := api.authService.Logout(r.Context(), userID); err != nil {
		api.logger.Error("Logout failed", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "logout failed")
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]string{"status": "logged out"})
}

func (api *AuthAPI) handleGetMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		api.writeError(w, http.StatusUnauthorized, "unauthorized", "not authenticated")
		return
	}

	user, err := api.authService.GetUser(r.Context(), userID)
	if err != nil {
		api.logger.Error("Failed to get user", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to get user")
		return
	}

	if user == nil {
		api.writeError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}

	// Build response with effective avatar URL
	response := map[string]interface{}{
		"id":          user.ID,
		"email":       user.Email,
		"displayName": user.DisplayName,
		"avatarUrl":   user.EffectiveAvatarURL(),
		"status":      user.Status,
		"createdAt":   user.CreatedAt,
		"callSign":    user.CallSign,
	}
	if user.LastLoginAt != nil {
		response["lastLoginAt"] = user.LastLoginAt
	}

	api.writeJSON(w, http.StatusOK, response)
}

func (api *AuthAPI) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	code := r.URL.Query().Get("code")
	errorParam := r.URL.Query().Get("error")

	if errorParam != "" {
		errorDesc := r.URL.Query().Get("error_description")
		redirectURL := fmt.Sprintf("%s/login?error=%s&error_description=%s",
			api.frontendURL,
			url.QueryEscape(errorParam),
			url.QueryEscape(errorDesc))
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}

	if code == "" {
		redirectURL := fmt.Sprintf("%s/login?error=missing_code", api.frontendURL)
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}

	// Exchange code for tokens and authenticate user
	response, err := api.authService.LoginWithGoogle(r.Context(), models.GoogleLoginParams{
		Code: code,
	})
	if err != nil {
		api.logger.Error("Google callback failed", logging.WithField("error", err.Error()))
		redirectURL := fmt.Sprintf("%s/login?error=auth_failed&error_description=%s",
			api.frontendURL,
			url.QueryEscape(err.Error()))
		http.Redirect(w, r, redirectURL, http.StatusFound)
		return
	}

	// Redirect to frontend with tokens in URL fragment (more secure than query params)
	redirectURL := fmt.Sprintf("%s/auth/callback#access_token=%s&refresh_token=%s",
		api.frontendURL,
		url.QueryEscape(response.Tokens.AccessToken),
		url.QueryEscape(response.Tokens.RefreshToken))
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (api *AuthAPI) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (api *AuthAPI) writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":   code,
		"message": message,
	})
}

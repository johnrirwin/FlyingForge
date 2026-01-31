package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/johnrirwin/mcp-news-feed/internal/auth"
	"github.com/johnrirwin/mcp-news-feed/internal/logging"
	"github.com/johnrirwin/mcp-news-feed/internal/models"
)

// AuthAPI handles authentication HTTP endpoints
type AuthAPI struct {
	authService    *auth.Service
	authMiddleware *auth.Middleware
	logger         *logging.Logger
}

// NewAuthAPI creates a new auth API handler
func NewAuthAPI(authService *auth.Service, authMiddleware *auth.Middleware, logger *logging.Logger) *AuthAPI {
	return &AuthAPI{
		authService:    authService,
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

// RegisterRoutes registers auth routes on the given mux
func (api *AuthAPI) RegisterRoutes(mux *http.ServeMux, corsMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("/api/auth/signup", corsMiddleware(api.handleSignup))
	mux.HandleFunc("/api/auth/login", corsMiddleware(api.handleLogin))
	mux.HandleFunc("/api/auth/google", corsMiddleware(api.handleGoogleLogin))
	mux.HandleFunc("/api/auth/refresh", corsMiddleware(api.handleRefresh))
	mux.HandleFunc("/api/auth/logout", corsMiddleware(api.authMiddleware.RequireAuth(api.handleLogout)))
	mux.HandleFunc("/api/auth/me", corsMiddleware(api.authMiddleware.RequireAuth(api.handleGetMe)))
}

func (api *AuthAPI) handleSignup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params models.SignupParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		api.writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	response, err := api.authService.SignupWithEmail(r.Context(), params)
	if err != nil {
		if authErr, ok := err.(*auth.AuthError); ok {
			status := http.StatusBadRequest
			if authErr.Code == "user_exists" {
				status = http.StatusConflict
			}
			api.writeError(w, status, authErr.Code, authErr.Message)
			return
		}
		api.logger.Error("Signup failed", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "signup failed")
		return
	}

	api.writeJSON(w, http.StatusCreated, response)
}

func (api *AuthAPI) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var params models.LoginParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		api.writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	response, err := api.authService.LoginWithEmail(r.Context(), params)
	if err != nil {
		if authErr, ok := err.(*auth.AuthError); ok {
			status := http.StatusUnauthorized
			if authErr.Code == "account_disabled" {
				status = http.StatusForbidden
			}
			api.writeError(w, status, authErr.Code, authErr.Message)
			return
		}
		api.logger.Error("Login failed", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "login failed")
		return
	}

	api.writeJSON(w, http.StatusOK, response)
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

	api.writeJSON(w, http.StatusOK, user)
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

package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/database"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

// SocialAPI handles social feature HTTP endpoints
type SocialAPI struct {
	userStore      *database.UserStore
	authMiddleware *auth.Middleware
	logger         *logging.Logger
}

// NewSocialAPI creates a new social API handler
func NewSocialAPI(userStore *database.UserStore, authMiddleware *auth.Middleware, logger *logging.Logger) *SocialAPI {
	return &SocialAPI{
		userStore:      userStore,
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

// RegisterRoutes registers social routes on the given mux
func (api *SocialAPI) RegisterRoutes(mux *http.ServeMux, corsMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	// Follow endpoints
	mux.HandleFunc("/api/social/follow/", corsMiddleware(api.authMiddleware.RequireAuth(api.handleFollow)))

	// Followers/following lists
	mux.HandleFunc("/api/social/", corsMiddleware(api.authMiddleware.RequireAuth(api.handleSocialLists)))

	// Social settings
	mux.HandleFunc("/api/me/social-settings", corsMiddleware(api.authMiddleware.RequireAuth(api.handleSocialSettings)))
}

// handleFollow handles POST/DELETE /api/social/follow/:userId
func (api *SocialAPI) handleFollow(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Extract target user ID from path: /api/social/follow/{userId}
	path := strings.TrimPrefix(r.URL.Path, "/api/social/follow/")
	targetUserID := strings.TrimSuffix(path, "/")

	if targetUserID == "" {
		api.writeError(w, http.StatusBadRequest, "invalid_request", "user ID required")
		return
	}

	userID := auth.GetUserID(r.Context())

	switch r.Method {
	case http.MethodPost:
		api.followUser(w, r, userID, targetUserID)
	case http.MethodDelete:
		api.unfollowUser(w, r, userID, targetUserID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// followUser handles POST /api/social/follow/:userId
func (api *SocialAPI) followUser(w http.ResponseWriter, r *http.Request, followerID, followedID string) {
	ctx := r.Context()

	// Check if follower has a callsign set (required to follow)
	follower, err := api.userStore.GetByID(ctx, followerID)
	if err != nil {
		api.logger.Error("Failed to get follower user", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to follow user")
		return
	}
	if follower == nil || follower.CallSign == "" {
		api.writeError(w, http.StatusBadRequest, "callsign_required", "you must set a call sign before following other pilots")
		return
	}

	// Check if target user exists
	targetUser, err := api.userStore.GetByID(ctx, followedID)
	if err != nil {
		api.logger.Error("Failed to get target user", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to follow user")
		return
	}
	if targetUser == nil {
		api.writeError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}

	// Check if target user has private profile
	if targetUser.SocialSettings.ProfileVisibility == models.ProfileVisibilityPrivate {
		api.writeError(w, http.StatusForbidden, "private_profile", "cannot follow a private profile")
		return
	}

	// Create follow relationship
	follow, err := api.userStore.CreateFollow(ctx, followerID, followedID)
	if err != nil {
		if strings.Contains(err.Error(), "cannot follow yourself") {
			api.writeError(w, http.StatusBadRequest, "invalid_request", "cannot follow yourself")
			return
		}
		api.logger.Error("Failed to create follow", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to follow user")
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"following": true,
		"followId":  follow.ID,
	})
}

// unfollowUser handles DELETE /api/social/follow/:userId
func (api *SocialAPI) unfollowUser(w http.ResponseWriter, r *http.Request, followerID, followedID string) {
	ctx := r.Context()

	err := api.userStore.DeleteFollow(ctx, followerID, followedID)
	if err != nil {
		api.logger.Error("Failed to delete follow", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to unfollow user")
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"following": false,
	})
}

// handleSocialLists handles GET /api/social/:userId/followers and /api/social/:userId/following
func (api *SocialAPI) handleSocialLists(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path: /api/social/{userId}/followers or /api/social/{userId}/following
	path := strings.TrimPrefix(r.URL.Path, "/api/social/")
	parts := strings.Split(strings.TrimSuffix(path, "/"), "/")

	if len(parts) < 2 {
		api.writeError(w, http.StatusBadRequest, "invalid_request", "invalid path")
		return
	}

	userID := parts[0]
	listType := parts[1]

	// Parse pagination
	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	ctx := r.Context()

	switch listType {
	case "followers":
		response, err := api.userStore.GetFollowers(ctx, userID, limit, offset)
		if err != nil {
			api.logger.Error("Failed to get followers", logging.WithField("error", err.Error()))
			api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to get followers")
			return
		}
		api.writeJSON(w, http.StatusOK, response)

	case "following":
		response, err := api.userStore.GetFollowing(ctx, userID, limit, offset)
		if err != nil {
			api.logger.Error("Failed to get following", logging.WithField("error", err.Error()))
			api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to get following")
			return
		}
		api.writeJSON(w, http.StatusOK, response)

	default:
		api.writeError(w, http.StatusBadRequest, "invalid_request", "invalid list type")
	}
}

// handleSocialSettings handles GET/PUT /api/me/social-settings
func (api *SocialAPI) handleSocialSettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	userID := auth.GetUserID(r.Context())

	switch r.Method {
	case http.MethodGet:
		api.getSocialSettings(w, r, userID)
	case http.MethodPut:
		api.updateSocialSettings(w, r, userID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getSocialSettings handles GET /api/me/social-settings
func (api *SocialAPI) getSocialSettings(w http.ResponseWriter, r *http.Request, userID string) {
	user, err := api.userStore.GetByID(r.Context(), userID)
	if err != nil {
		api.logger.Error("Failed to get user", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to get settings")
		return
	}
	if user == nil {
		api.writeError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}

	api.writeJSON(w, http.StatusOK, user.SocialSettings)
}

// updateSocialSettings handles PUT /api/me/social-settings
func (api *SocialAPI) updateSocialSettings(w http.ResponseWriter, r *http.Request, userID string) {
	var params models.UpdateSocialSettingsParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		api.writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}

	// Validate profile visibility if provided
	if params.ProfileVisibility != nil {
		if *params.ProfileVisibility != models.ProfileVisibilityPublic && *params.ProfileVisibility != models.ProfileVisibilityPrivate {
			api.writeError(w, http.StatusBadRequest, "invalid_value", "profileVisibility must be 'public' or 'private'")
			return
		}
	}

	if err := api.userStore.UpdateSocialSettings(r.Context(), userID, params); err != nil {
		api.logger.Error("Failed to update social settings", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to update settings")
		return
	}

	// Return updated settings
	user, err := api.userStore.GetByID(r.Context(), userID)
	if err != nil {
		api.logger.Error("Failed to get updated user", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to get updated settings")
		return
	}

	api.writeJSON(w, http.StatusOK, user.SocialSettings)
}

// writeJSON writes a JSON response
func (api *SocialAPI) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func (api *SocialAPI) writeError(w http.ResponseWriter, status int, code, message string) {
	api.writeJSON(w, status, map[string]string{
		"error":   code,
		"message": message,
	})
}

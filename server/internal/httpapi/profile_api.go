package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/database"
	"github.com/johnrirwin/flyingforge/internal/images"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

// ProfileAPI handles profile HTTP endpoints
type ProfileAPI struct {
	userStore      *database.UserStore
	imageSvc       *images.Service
	authMiddleware *auth.Middleware
	logger         *logging.Logger
}

// NewProfileAPI creates a new profile API handler
func NewProfileAPI(userStore *database.UserStore, imageSvc *images.Service, authMiddleware *auth.Middleware, logger *logging.Logger) *ProfileAPI {
	return &ProfileAPI{
		userStore:      userStore,
		imageSvc:       imageSvc,
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

// RegisterRoutes registers profile routes on the given mux
func (api *ProfileAPI) RegisterRoutes(mux *http.ServeMux, corsMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("/api/me/profile", corsMiddleware(api.authMiddleware.RequireAuth(api.handleProfile)))
	mux.HandleFunc("/api/me/avatar", corsMiddleware(api.authMiddleware.RequireAuth(api.handleAvatar)))
	mux.HandleFunc("/api/users/avatar", corsMiddleware(api.authMiddleware.RequireAuth(api.handleAvatar)))
}

// handleProfile handles GET, PUT, and DELETE /api/me/profile
func (api *ProfileAPI) handleProfile(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		api.handleGetProfile(w, r)
	case http.MethodPut:
		api.handleUpdateProfile(w, r)
	case http.MethodDelete:
		api.handleDeleteProfile(w, r)
	case http.MethodOptions:
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetProfile returns the current user's profile
func (api *ProfileAPI) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(auth.UserIDKey).(string)

	user, err := api.userStore.GetByID(r.Context(), userID)
	if err != nil {
		api.logger.Error("Failed to get user", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to get profile")
		return
	}
	if user == nil {
		api.writeError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}

	// Build profile response with effective avatar URL
	response := map[string]interface{}{
		"id":                 user.ID,
		"email":              user.Email,
		"displayName":        user.DisplayName,
		"callSign":           user.CallSign,
		"googleName":         user.GoogleName,
		"googleAvatarUrl":    user.GoogleAvatarURL,
		"avatarType":         user.AvatarType,
		"customAvatarUrl":    user.CustomAvatarURL,
		"avatarImageAssetId": user.AvatarImageID,
		"effectiveAvatarUrl": user.EffectiveAvatarURL(),
		"createdAt":          user.CreatedAt,
		"updatedAt":          user.UpdatedAt,
	}

	api.writeJSON(w, http.StatusOK, response)
}

// handleUpdateProfile updates the current user's profile
func (api *ProfileAPI) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(auth.UserIDKey).(string)

	var params models.UpdateProfileParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		api.writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	// Validate callsign if provided
	if params.CallSign != nil {
		trimmedCallSign := strings.TrimSpace(*params.CallSign)
		if trimmedCallSign != "" {
			if err := models.ValidateCallSign(trimmedCallSign); err != nil {
				if validErr, ok := err.(*models.ValidationError); ok {
					api.writeError(w, http.StatusBadRequest, "validation_error", validErr.Message)
					return
				}
				api.writeError(w, http.StatusBadRequest, "validation_error", err.Error())
				return
			}

			// Check if callsign is taken by another user
			existing, err := api.userStore.GetByCallSign(r.Context(), trimmedCallSign)
			if err != nil {
				api.logger.Error("Failed to check callsign", logging.WithField("error", err.Error()))
				api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to check callsign")
				return
			}
			if existing != nil && existing.ID != userID {
				api.writeError(w, http.StatusConflict, "callsign_taken", "this callsign is already in use")
				return
			}
		} else {
			// Callsign is being cleared - check if user had one and delete all follows
			currentUser, err := api.userStore.GetByID(r.Context(), userID)
			if err != nil {
				api.logger.Error("Failed to get current user", logging.WithField("error", err.Error()))
				api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to update profile")
				return
			}
			if currentUser != nil && currentUser.CallSign != "" {
				// User had a callsign and is clearing it - delete all follow relationships
				if err := api.userStore.DeleteAllFollowsForUser(r.Context(), userID); err != nil {
					api.logger.Error("Failed to delete follows when clearing callsign",
						logging.WithField("error", err.Error()),
						logging.WithField("userID", userID))
					// Continue anyway - the callsign update is more important
				} else {
					api.logger.Info("Deleted all follows for user clearing callsign",
						logging.WithField("userID", userID))
				}
			}
		}
	}

	// Build update params
	updateParams := models.UpdateUserParams{}
	if params.CallSign != nil {
		trimmed := strings.TrimSpace(*params.CallSign)
		updateParams.CallSign = &trimmed
	}
	if params.DisplayName != nil {
		trimmed := strings.TrimSpace(*params.DisplayName)
		updateParams.DisplayName = &trimmed
	}
	if params.AvatarType != nil {
		updateParams.AvatarType = params.AvatarType
	}

	user, err := api.userStore.Update(r.Context(), userID, updateParams)
	if err != nil {
		api.logger.Error("Failed to update profile", logging.WithField("error", err.Error()))
		if strings.Contains(err.Error(), "duplicate key") {
			api.writeError(w, http.StatusConflict, "callsign_taken", "this callsign is already in use")
			return
		}
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to update profile")
		return
	}

	// Build response
	response := map[string]interface{}{
		"id":                 user.ID,
		"email":              user.Email,
		"displayName":        user.DisplayName,
		"callSign":           user.CallSign,
		"googleName":         user.GoogleName,
		"googleAvatarUrl":    user.GoogleAvatarURL,
		"avatarType":         user.AvatarType,
		"customAvatarUrl":    user.CustomAvatarURL,
		"avatarImageAssetId": user.AvatarImageID,
		"effectiveAvatarUrl": user.EffectiveAvatarURL(),
		"createdAt":          user.CreatedAt,
		"updatedAt":          user.UpdatedAt,
	}

	api.writeJSON(w, http.StatusOK, response)
}

// handleAvatar handles POST /api/me/avatar for uploading custom avatars
func (api *ProfileAPI) handleAvatar(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.Context().Value(auth.UserIDKey).(string)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	currentUser, err := api.userStore.GetByID(ctx, userID)
	if err != nil {
		api.logger.Error("Failed to load user before avatar save", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to save avatar")
		return
	}
	if currentUser == nil {
		api.writeError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}

	var (
		asset *models.ImageAsset
	)

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	switch {
	case strings.HasPrefix(contentType, "application/json"):
		var req struct {
			UploadID string `json:"uploadId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			api.writeError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		req.UploadID = strings.TrimSpace(req.UploadID)
		if req.UploadID == "" {
			api.writeError(w, http.StatusBadRequest, "invalid_request", "uploadId is required")
			return
		}

		var err error
		asset, err = api.imageSvc.PersistApprovedUpload(ctx, userID, req.UploadID, models.ImageEntityAvatar, userID)
		if err != nil {
			switch err {
			case images.ErrPendingUploadNotFound:
				api.writeError(w, http.StatusUnprocessableEntity, "not_approved", "image approval token expired or missing")
				return
			case images.ErrUploadNotApproved:
				api.writeError(w, http.StatusUnprocessableEntity, "not_approved", "image is not approved")
				return
			default:
				api.logger.Error("Failed to persist approved avatar image", logging.WithField("error", err.Error()))
				api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to save avatar")
				return
			}
		}
	default:
		const maxSize = int64(3 * 1024 * 1024)
		r.Body = http.MaxBytesReader(w, r.Body, maxSize)
		if err := r.ParseMultipartForm(maxSize); err != nil {
			api.writeError(w, http.StatusBadRequest, "invalid_request", "invalid upload payload")
			return
		}

		file, _, err := r.FormFile("image")
		if err != nil {
			file, _, err = r.FormFile("avatar")
		}
		if err != nil {
			api.writeError(w, http.StatusBadRequest, "invalid_request", "image file is required")
			return
		}
		defer file.Close()

		imageData, err := io.ReadAll(file)
		if err != nil {
			api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to read image")
			return
		}
		if len(imageData) > 2*1024*1024 {
			api.writeError(w, http.StatusBadRequest, "invalid_request", "image must be less than 2MB")
			return
		}
		if _, ok := detectAllowedImageContentType(imageData); !ok {
			api.writeError(w, http.StatusBadRequest, "invalid_request", "only JPEG, PNG, and WebP images are allowed")
			return
		}

		decision, savedAsset, err := api.imageSvc.ModerateAndPersist(ctx, images.SaveRequest{
			OwnerUserID: userID,
			EntityType:  models.ImageEntityAvatar,
			EntityID:    userID,
			ImageBytes:  imageData,
		})
		if err != nil {
			api.logger.Error("Failed to moderate avatar image", logging.WithField("error", err.Error()))
			api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to save avatar")
			return
		}
		if decision == nil || decision.Status != models.ImageModerationApproved {
			if decision != nil && decision.Status == models.ImageModerationPendingReview {
				api.writeError(w, http.StatusServiceUnavailable, "not_approved", "unable to verify right now")
				return
			}
			reason := "image is not approved"
			if decision != nil && strings.TrimSpace(decision.Reason) != "" {
				reason = decision.Reason
			}
			api.writeError(w, http.StatusUnprocessableEntity, "not_approved", reason)
			return
		}
		asset = savedAsset
	}

	if asset == nil {
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to save avatar")
		return
	}

	avatarURL := "/api/images/" + asset.ID
	avatarType := models.AvatarTypeCustom
	updateParams := models.UpdateUserParams{
		CustomAvatarURL: &avatarURL, // compatibility field for older clients
		AvatarImageID:   &asset.ID,
		AvatarType:      &avatarType,
	}

	user, err := api.userStore.Update(ctx, userID, updateParams)
	if err != nil {
		api.logger.Error("Failed to update avatar", logging.WithField("error", err.Error()))
		_ = api.imageSvc.Delete(ctx, asset.ID)
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to update avatar")
		return
	}
	if currentUser.AvatarImageID != "" && currentUser.AvatarImageID != asset.ID {
		_ = api.imageSvc.Delete(ctx, currentUser.AvatarImageID)
	}

	response := map[string]interface{}{
		"avatarUrl":          user.CustomAvatarURL,
		"avatarType":         user.AvatarType,
		"avatarImageAssetId": user.AvatarImageID,
		"effectiveAvatar":    user.EffectiveAvatarURL(),
	}

	api.writeJSON(w, http.StatusOK, response)
}

// handleDeleteProfile permanently deletes the current user's account and all associated data
func (api *ProfileAPI) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(auth.UserIDKey).(string)

	// Verify user exists
	user, err := api.userStore.GetByID(r.Context(), userID)
	if err != nil {
		api.logger.Error("Failed to get user for deletion", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to verify account")
		return
	}
	if user == nil {
		api.writeError(w, http.StatusNotFound, "not_found", "user not found")
		return
	}

	// Log the deletion attempt (avoid logging PII like email)
	api.logger.Info("User account deletion requested",
		logging.WithField("userID", userID))

	// Delete the user (cascades to related data via DB constraints)
	if err := api.userStore.HardDelete(r.Context(), userID); err != nil {
		api.logger.Error("Failed to delete user account",
			logging.WithField("error", err.Error()),
			logging.WithField("userID", userID))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete account")
		return
	}

	api.logger.Info("User account deleted successfully",
		logging.WithField("userID", userID))

	// Return success with no content
	w.WriteHeader(http.StatusNoContent)
}

func (api *ProfileAPI) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (api *ProfileAPI) writeError(w http.ResponseWriter, status int, code, message string) {
	api.writeJSON(w, status, map[string]string{
		"code":    code,
		"message": message,
	})
}

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/database"
	"github.com/johnrirwin/flyingforge/internal/images"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

// AdminAPI handles admin-only endpoints
type AdminAPI struct {
	catalogStore   *database.GearCatalogStore
	userStore      *database.UserStore
	imageSvc       *images.Service
	authMiddleware *auth.Middleware
	logger         *logging.Logger
}

// NewAdminAPI creates a new admin API handler
func NewAdminAPI(catalogStore *database.GearCatalogStore, userStore *database.UserStore, imageSvc *images.Service, authMiddleware *auth.Middleware, logger *logging.Logger) *AdminAPI {
	return &AdminAPI{
		catalogStore:   catalogStore,
		userStore:      userStore,
		imageSvc:       imageSvc,
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

// RegisterRoutes registers admin routes
func (api *AdminAPI) RegisterRoutes(mux *http.ServeMux, corsMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	if api.authMiddleware == nil {
		api.logger.Error("Admin API routes not registered: authMiddleware is nil")
		return
	}

	// Gear moderation routes: admin OR gear-admin role
	mux.HandleFunc("/api/admin/gear", corsMiddleware(api.authMiddleware.RequireAuth(api.requireGearModerator(api.handleAdminGear))))
	mux.HandleFunc("/api/admin/gear/", corsMiddleware(api.authMiddleware.RequireAuth(api.requireGearModerator(api.handleAdminGearByID))))

	// User admin routes: admin role only
	mux.HandleFunc("/api/admin/users", corsMiddleware(api.authMiddleware.RequireAuth(api.requireAdmin(api.handleAdminUsers))))
	mux.HandleFunc("/api/admin/users/", corsMiddleware(api.authMiddleware.RequireAuth(api.requireAdmin(api.handleAdminUserByID))))
}

func canModerateGear(user *models.User) bool {
	return user != nil && (user.IsAdmin || user.IsGearAdmin)
}

func canManageUsers(user *models.User) bool {
	return user != nil && user.IsAdmin
}

// requireRole is middleware that checks role-based access for admin endpoints.
func (api *AdminAPI) requireRole(next http.HandlerFunc, allowed func(*models.User) bool, deniedMessage string, deniedLogMessage string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.GetUserID(r.Context())
		if userID == "" {
			http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		user, err := api.userStore.GetByID(ctx, userID)
		if err != nil {
			api.logger.Error("Failed to get user for admin check", logging.WithField("error", err.Error()))
			http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
			return
		}
		if user == nil {
			api.logger.Warn("Authenticated user missing during admin check", logging.WithField("userId", userID))
			http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
			return
		}

		if !allowed(user) {
			api.logger.Warn(deniedLogMessage, logging.WithField("userId", userID))
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, deniedMessage), http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

// requireAdmin is middleware that checks if the authenticated user is a full admin.
func (api *AdminAPI) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return api.requireRole(next, canManageUsers, "admin access required", "Non-admin user attempted user-admin access")
}

// requireGearModerator is middleware that allows full admins and gear-admin users.
func (api *AdminAPI) requireGearModerator(next http.HandlerFunc) http.HandlerFunc {
	return api.requireRole(next, canModerateGear, "admin or gear-admin access required", "User without gear moderation role attempted admin gear access")
}

// handleAdminGear handles GET /api/admin/gear (list gear for moderation)
func (api *AdminAPI) handleAdminGear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	query := r.URL.Query()

	params := models.AdminGearSearchParams{
		Query:       query.Get("query"),
		GearType:    models.GearType(query.Get("gearType")),
		Brand:       query.Get("brand"),
		ImageStatus: models.ImageStatus(query.Get("imageStatus")),
		Limit:       parseIntQuery(query.Get("limit"), 20),
		Offset:      parseIntQuery(query.Get("offset"), 0),
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	response, err := api.catalogStore.AdminSearch(ctx, params)
	if err != nil {
		api.logger.Error("Failed to admin search gear", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to search gear catalog",
		})
		return
	}

	api.writeJSON(w, http.StatusOK, response)
}

// handleAdminGearByID handles GET/PUT/DELETE /api/admin/gear/{id} and /api/admin/gear/{id}/image
func (api *AdminAPI) handleAdminGearByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/gear/")

	// Check if this is an image request
	if strings.HasSuffix(path, "/image") {
		id := strings.TrimSuffix(path, "/image")
		api.handleGearImage(w, r, id)
		return
	}

	id := strings.TrimSuffix(path, "/")
	if id == "" {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "gear ID required"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		api.handleGetGear(w, r, id)
	case http.MethodPut:
		api.handleUpdateGear(w, r, id)
	case http.MethodDelete:
		api.handleDeleteGear(w, r, id)
	default:
		api.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// handleGetGear handles GET /api/admin/gear/{id}
func (api *AdminAPI) handleGetGear(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	item, err := api.catalogStore.Get(ctx, id)
	if err != nil {
		api.logger.Error("Failed to get gear item", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to get gear item",
		})
		return
	}

	if item == nil {
		api.writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "gear item not found",
		})
		return
	}

	api.writeJSON(w, http.StatusOK, item)
}

// handleUpdateGear handles PUT /api/admin/gear/{id}
func (api *AdminAPI) handleUpdateGear(w http.ResponseWriter, r *http.Request, id string) {
	userID := auth.GetUserID(r.Context())

	var params models.AdminUpdateGearCatalogParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	// Validate ImageURL if provided
	if params.ImageURL != nil && *params.ImageURL != "" {
		if err := validateImageURL(*params.ImageURL); err != nil {
			api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Verify the item exists first
	existing, err := api.catalogStore.Get(ctx, id)
	if err != nil {
		api.logger.Error("Failed to get gear item for update", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to get gear item",
		})
		return
	}

	if existing == nil {
		api.writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "gear item not found",
		})
		return
	}

	// Perform the update
	item, err := api.catalogStore.AdminUpdate(ctx, id, userID, params)
	if err != nil {
		api.logger.Error("Failed to update gear item", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to update gear item",
		})
		return
	}

	api.logger.Info("Admin updated gear item",
		logging.WithField("gearId", id),
		logging.WithField("adminId", userID),
	)

	api.writeJSON(w, http.StatusOK, item)
}

// handleDeleteGear handles DELETE /api/admin/gear/{id}
func (api *AdminAPI) handleDeleteGear(w http.ResponseWriter, r *http.Request, id string) {
	userID := auth.GetUserID(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	if err := api.catalogStore.AdminDelete(ctx, id); err != nil {
		if errors.Is(err, database.ErrCatalogItemNotFound) {
			api.writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "gear item not found",
			})
			return
		}

		api.logger.Error("Failed to delete gear item", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to delete gear item",
		})
		return
	}

	api.logger.Info("Admin deleted gear item",
		logging.WithField("gearId", id),
		logging.WithField("adminId", userID),
	)

	api.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Gear item deleted successfully",
	})
}

// handleAdminUsers handles GET /api/admin/users for searching users.
func (api *AdminAPI) handleAdminUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	query := r.URL.Query()
	status := models.UserStatus(strings.TrimSpace(query.Get("status")))
	if status != "" && !models.IsValidUserStatus(status) {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid status filter"})
		return
	}

	limit := parseIntQuery(query.Get("limit"), 20)
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset := parseIntQuery(query.Get("offset"), 0)
	if offset < 0 {
		offset = 0
	}

	params := models.UserFilterParams{
		Query:  strings.TrimSpace(query.Get("query")),
		Status: status,
		Limit:  limit,
		Offset: offset,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	response, err := api.userStore.List(ctx, params)
	if err != nil {
		api.logger.Error("Failed to list users for admin", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to list users",
		})
		return
	}

	api.writeJSON(w, http.StatusOK, response)
}

// handleAdminUserByID handles GET/PATCH/DELETE /api/admin/users/{id}
func (api *AdminAPI) handleAdminUserByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/users/")

	if strings.HasSuffix(path, "/avatar") {
		id := strings.TrimSuffix(path, "/avatar")
		id = strings.TrimSuffix(id, "/")
		if id == "" {
			api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user ID required"})
			return
		}

		if r.Method != http.MethodDelete {
			api.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		api.handleDeleteAdminUserAvatar(w, r, id)
		return
	}

	id := strings.TrimSuffix(path, "/")
	if id == "" {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "user ID required"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		api.handleGetAdminUser(w, r, id)
	case http.MethodPatch, http.MethodPut:
		api.handleUpdateAdminUser(w, r, id)
	case http.MethodDelete:
		api.handleDeleteAdminUser(w, r, id)
	default:
		api.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// handleGetAdminUser handles GET /api/admin/users/{id}
func (api *AdminAPI) handleGetAdminUser(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	user, err := api.userStore.GetByID(ctx, id)
	if err != nil {
		api.logger.Error("Failed to get user for admin", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to get user",
		})
		return
	}
	if user == nil {
		api.writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "user not found",
		})
		return
	}

	api.writeJSON(w, http.StatusOK, user)
}

// handleUpdateAdminUser handles PATCH /api/admin/users/{id}
func (api *AdminAPI) handleUpdateAdminUser(w http.ResponseWriter, r *http.Request, id string) {
	adminUserID := auth.GetUserID(r.Context())

	var params models.AdminUpdateUserParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if params.Status == nil && params.IsAdmin == nil && params.IsGearAdmin == nil {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "at least one updatable field is required"})
		return
	}

	if params.Status != nil && !models.IsValidUserStatus(*params.Status) {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user status"})
		return
	}

	if id == adminUserID {
		if params.IsAdmin != nil && !*params.IsAdmin {
			api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot remove your own admin role"})
			return
		}
		if params.Status != nil && *params.Status != models.UserStatusActive {
			api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot disable your own account from user admin"})
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	existing, err := api.userStore.GetByID(ctx, id)
	if err != nil {
		api.logger.Error("Failed to get user for admin update", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to get user",
		})
		return
	}
	if existing == nil {
		api.writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "user not found",
		})
		return
	}

	updated, err := api.userStore.AdminUpdate(ctx, id, params)
	if err != nil {
		api.logger.Error("Failed to update user from admin", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to update user",
		})
		return
	}

	api.logger.Info("Admin updated user access",
		logging.WithField("targetUserId", id),
		logging.WithField("adminId", adminUserID),
	)

	api.writeJSON(w, http.StatusOK, updated)
}

// handleDeleteAdminUser handles DELETE /api/admin/users/{id}
func (api *AdminAPI) handleDeleteAdminUser(w http.ResponseWriter, r *http.Request, id string) {
	adminUserID := auth.GetUserID(r.Context())
	if id == adminUserID {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "use profile settings to delete your own account",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	existing, err := api.userStore.GetByID(ctx, id)
	if err != nil {
		api.logger.Error("Failed to get user for admin delete", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to get user",
		})
		return
	}
	if existing == nil {
		api.writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "user not found",
		})
		return
	}

	if err := api.userStore.HardDelete(ctx, id); err != nil {
		api.logger.Error("Failed to delete user from admin", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to delete user",
		})
		return
	}

	api.logger.Info("Admin deleted user account",
		logging.WithField("targetUserId", id),
		logging.WithField("adminId", adminUserID),
	)

	api.writeJSON(w, http.StatusOK, map[string]string{
		"message": "User deleted successfully",
	})
}

// handleDeleteAdminUserAvatar handles DELETE /api/admin/users/{id}/avatar
func (api *AdminAPI) handleDeleteAdminUserAvatar(w http.ResponseWriter, r *http.Request, id string) {
	adminUserID := auth.GetUserID(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	existing, err := api.userStore.GetByID(ctx, id)
	if err != nil {
		api.logger.Error("Failed to get user for avatar delete", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to get user",
		})
		return
	}
	if existing == nil {
		api.writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "user not found",
		})
		return
	}

	updated, err := api.userStore.AdminClearAvatar(ctx, id)
	if err != nil {
		api.logger.Error("Failed to remove user avatar from admin", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to remove profile picture",
		})
		return
	}
	if existing.AvatarImageID != "" {
		_ = api.imageSvc.Delete(ctx, existing.AvatarImageID)
	}

	api.logger.Info("Admin removed user avatar",
		logging.WithField("targetUserId", id),
		logging.WithField("adminId", adminUserID),
	)

	api.writeJSON(w, http.StatusOK, updated)
}

// writeJSON writes a JSON response
func (api *AdminAPI) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	// Prevent browser caching of admin API responses to ensure fresh data after edits
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// parseIntQuery parses an integer from query string with a default
func parseIntQuery(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return val
}

// handleGearImage handles POST /api/admin/gear/{id}/image for image upload
// and GET for serving the image
func (api *AdminAPI) handleGearImage(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodPost:
		api.uploadGearImage(w, r, id)
	case http.MethodGet:
		api.getGearImage(w, r, id)
	case http.MethodDelete:
		api.deleteGearImage(w, r, id)
	default:
		api.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

// uploadGearImage handles POST /api/admin/gear/{id}/image
// Max file size: 1MB, accepts JPEG/PNG/WebP only
func (api *AdminAPI) uploadGearImage(w http.ResponseWriter, r *http.Request, id string) {
	userID := auth.GetUserID(r.Context())

	// Limit request body to 1.5MB (slightly more than 1MB limit to account for multipart overhead)
	maxSize := int64(1.5 * 1024 * 1024)
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	// Parse multipart form
	if err := r.ParseMultipartForm(maxSize); err != nil {
		api.logger.Error("Failed to parse multipart form", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "File too large. Maximum size is 1MB.",
		})
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		api.logger.Error("Failed to get image from form", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Image file required",
		})
		return
	}
	defer file.Close()

	// Read image data
	imageData, err := io.ReadAll(file)
	if err != nil {
		api.logger.Error("Failed to read image data", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to read image",
		})
		return
	}
	if len(imageData) > 1024*1024 {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "File too large. Maximum size is 1MB.",
		})
		return
	}
	contentType, ok := detectAllowedImageContentType(imageData)
	if !ok {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Image must be JPEG, PNG, or WebP",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Verify the gear item exists
	existing, err := api.catalogStore.Get(ctx, id)
	if err != nil {
		api.logger.Error("Failed to verify gear item", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to verify gear item",
		})
		return
	}
	if existing == nil {
		api.writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "Gear item not found",
		})
		return
	}

	// Store the image
	decision, asset, err := api.imageSvc.ModerateAndPersist(ctx, images.SaveRequest{
		OwnerUserID: userID,
		EntityType:  models.ImageEntityGear,
		EntityID:    id,
		ImageBytes:  imageData,
	})
	if err != nil {
		api.logger.Error("Failed to moderate gear image", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to moderate image",
		})
		return
	}
	if decision.Status != models.ImageModerationApproved {
		statusCode := http.StatusUnprocessableEntity
		if decision.Status == models.ImageModerationPendingReview {
			statusCode = http.StatusServiceUnavailable
		}
		api.writeJSON(w, statusCode, map[string]string{
			"status": string(decision.Status),
			"error":  decision.Reason,
		})
		return
	}

	previousAssetID, err := api.catalogStore.SetImage(ctx, id, userID, contentType, asset.ID)
	if err != nil {
		api.logger.Error("Failed to store gear image", logging.WithFields(map[string]interface{}{
			"gearId": id,
			"error":  err.Error(),
		}))
		_ = api.imageSvc.Delete(ctx, asset.ID)
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to store image",
		})
		return
	}
	if previousAssetID != "" && previousAssetID != asset.ID {
		_ = api.imageSvc.Delete(ctx, previousAssetID)
	}

	api.logger.Info("Admin uploaded gear image",
		logging.WithField("gearId", id),
		logging.WithField("adminId", userID),
		logging.WithField("size", len(imageData)),
	)

	api.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Image uploaded successfully",
	})
}

// getGearImage handles GET /api/admin/gear/{id}/image
func (api *AdminAPI) getGearImage(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	imageData, imageType, err := api.catalogStore.GetImage(ctx, id)
	if err != nil {
		api.logger.Error("Failed to get gear image", logging.WithFields(map[string]interface{}{
			"gearId": id,
			"error":  err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get image"})
		return
	}

	if imageData == nil {
		api.writeJSON(w, http.StatusNotFound, map[string]string{"error": "no image for this gear item"})
		return
	}
	if imageType == "" {
		imageType = http.DetectContentType(imageData)
	}

	// No caching for admin endpoint - admins need to see latest image
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Content-Type", imageType)
	w.Header().Set("Content-Length", strconv.Itoa(len(imageData)))
	w.Write(imageData)
}

// deleteGearImage handles DELETE /api/admin/gear/{id}/image
func (api *AdminAPI) deleteGearImage(w http.ResponseWriter, r *http.Request, id string) {
	userID := auth.GetUserID(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	previousAssetID, err := api.catalogStore.DeleteImage(ctx, id)
	if err != nil {
		api.logger.Error("Failed to delete gear image", logging.WithFields(map[string]interface{}{
			"gearId": id,
			"error":  err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete image",
		})
		return
	}
	if previousAssetID != "" {
		_ = api.imageSvc.Delete(ctx, previousAssetID)
	}

	api.logger.Info("Admin deleted gear image",
		logging.WithField("gearId", id),
		logging.WithField("adminId", userID),
	)

	api.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Image deleted successfully",
	})
}

package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/database"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

// AdminAPI handles admin-only endpoints
type AdminAPI struct {
	catalogStore   *database.GearCatalogStore
	userStore      *database.UserStore
	authMiddleware *auth.Middleware
	logger         *logging.Logger
}

// NewAdminAPI creates a new admin API handler
func NewAdminAPI(catalogStore *database.GearCatalogStore, userStore *database.UserStore, authMiddleware *auth.Middleware, logger *logging.Logger) *AdminAPI {
	return &AdminAPI{
		catalogStore:   catalogStore,
		userStore:      userStore,
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

	// All admin routes require authentication AND admin role
	mux.HandleFunc("/api/admin/gear", corsMiddleware(api.authMiddleware.RequireAuth(api.requireAdmin(api.handleAdminGear))))
	mux.HandleFunc("/api/admin/gear/", corsMiddleware(api.authMiddleware.RequireAuth(api.requireAdmin(api.handleAdminGearByID))))
}

// requireAdmin is middleware that checks if the authenticated user is an admin
func (api *AdminAPI) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.GetUserID(r.Context())
		if userID == "" {
			http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		user, err := api.userStore.GetByID(ctx, userID)
		if err != nil || user == nil {
			api.logger.Error("Failed to get user for admin check", logging.WithField("error", err))
			http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
			return
		}

		if !user.IsAdmin {
			api.logger.Warn("Non-admin user attempted admin access", logging.WithField("userId", userID))
			http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
			return
		}

		next(w, r)
	}
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

// handleAdminGearByID handles GET/PUT /api/admin/gear/{id} and /api/admin/gear/{id}/image
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

	file, header, err := r.FormFile("image")
	if err != nil {
		api.logger.Error("Failed to get image from form", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Image file required",
		})
		return
	}
	defer file.Close()

	// Validate file size (1MB max)
	if header.Size > 1024*1024 {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "File too large. Maximum size is 1MB.",
		})
		return
	}

	// Validate content type
	contentType := header.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Image must be JPEG, PNG, or WebP",
		})
		return
	}

	// Read image data
	imageData, err := io.ReadAll(file)
	if err != nil {
		api.logger.Error("Failed to read image data", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to read image",
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
	if err := api.catalogStore.SetImage(ctx, id, userID, contentType, imageData); err != nil {
		api.logger.Error("Failed to store gear image", logging.WithFields(map[string]interface{}{
			"gearId": id,
			"error":  err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to store image",
		})
		return
	}

	api.logger.Info("Admin uploaded gear image",
		logging.WithField("gearId", id),
		logging.WithField("adminId", userID),
		logging.WithField("size", header.Size),
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

	if err := api.catalogStore.DeleteImage(ctx, id); err != nil {
		api.logger.Error("Failed to delete gear image", logging.WithFields(map[string]interface{}{
			"gearId": id,
			"error":  err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete image",
		})
		return
	}

	api.logger.Info("Admin deleted gear image",
		logging.WithField("gearId", id),
		logging.WithField("adminId", userID),
	)

	api.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Image deleted successfully",
	})
}

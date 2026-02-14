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

	"github.com/google/uuid"

	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/builds"
	"github.com/johnrirwin/flyingforge/internal/database"
	"github.com/johnrirwin/flyingforge/internal/images"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

// AdminAPI handles admin-only endpoints
type AdminAPI struct {
	catalogStore   *database.GearCatalogStore
	userStore      *database.UserStore
	buildSvc       *builds.Service
	imageSvc       *images.Service
	authMiddleware *auth.Middleware
	logger         *logging.Logger
}

// NewAdminAPI creates a new admin API handler
func NewAdminAPI(catalogStore *database.GearCatalogStore, userStore *database.UserStore, buildSvc *builds.Service, imageSvc *images.Service, authMiddleware *auth.Middleware, logger *logging.Logger) *AdminAPI {
	return &AdminAPI{
		catalogStore:   catalogStore,
		userStore:      userStore,
		buildSvc:       buildSvc,
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

	// Content moderation routes: admin OR content-admin role.
	mux.HandleFunc("/api/admin/gear", corsMiddleware(api.authMiddleware.RequireAuth(api.requireContentModerator(api.handleAdminGear))))
	mux.HandleFunc("/api/admin/gear/bulk-delete", corsMiddleware(api.authMiddleware.RequireAuth(api.requireContentModerator(api.handleAdminGearBulkDelete))))
	mux.HandleFunc("/api/admin/gear/near-matches", corsMiddleware(api.authMiddleware.RequireAuth(api.requireContentModerator(api.handleAdminGearNearMatches))))
	mux.HandleFunc("/api/admin/gear/", corsMiddleware(api.authMiddleware.RequireAuth(api.requireContentModerator(api.handleAdminGearByID))))
	if api.buildSvc != nil {
		mux.HandleFunc("/api/admin/builds", corsMiddleware(api.authMiddleware.RequireAuth(api.requireContentModerator(api.handleAdminBuilds))))
		mux.HandleFunc("/api/admin/builds/", corsMiddleware(api.authMiddleware.RequireAuth(api.requireContentModerator(api.handleAdminBuildByID))))
	}

	// User admin routes: admin role only
	mux.HandleFunc("/api/admin/users", corsMiddleware(api.authMiddleware.RequireAuth(api.requireAdmin(api.handleAdminUsers))))
	mux.HandleFunc("/api/admin/users/", corsMiddleware(api.authMiddleware.RequireAuth(api.requireAdmin(api.handleAdminUserByID))))
}

func canModerateContent(user *models.User) bool {
	return user != nil && (user.IsAdmin || user.IsContentAdmin)
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

// requireContentModerator is middleware that allows full admins and content-admin users.
func (api *AdminAPI) requireContentModerator(next http.HandlerFunc) http.HandlerFunc {
	return api.requireRole(next, canModerateContent, "admin or content-admin access required", "User without content moderation role attempted admin content access")
}

// handleAdminGear handles GET /api/admin/gear (list gear for moderation)
func (api *AdminAPI) handleAdminGear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	query := r.URL.Query()
	rawStatus := models.CatalogItemStatus(strings.TrimSpace(query.Get("status")))
	status := models.CatalogItemStatus("")
	if rawStatus != "" {
		status = models.NormalizeCatalogStatus(rawStatus)
		if !models.IsValidCatalogStatus(status) {
			api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid status"})
			return
		}
	}

	params := models.AdminGearSearchParams{
		Query:       query.Get("query"),
		GearType:    models.GearType(query.Get("gearType")),
		Brand:       query.Get("brand"),
		Status:      status,
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

// handleAdminGearNearMatches handles POST /api/admin/gear/near-matches.
// Used by content admins to warn about potential duplicates during bulk imports.
func (api *AdminAPI) handleAdminGearNearMatches(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var body struct {
		GearType  models.GearType `json:"gearType"`
		Brand     string          `json:"brand"`
		Model     string          `json:"model"`
		Threshold float64         `json:"threshold,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	body.Brand = strings.TrimSpace(body.Brand)
	body.Model = strings.TrimSpace(body.Model)
	if body.GearType == "" || body.Brand == "" || body.Model == "" {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "gearType, brand, and model are required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	matches, err := api.catalogStore.FindNearMatchesAdmin(ctx, body.GearType, body.Brand, body.Model, body.Threshold)
	if err != nil {
		api.logger.Error("Failed to find near matches (admin)", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	api.writeJSON(w, http.StatusOK, models.NearMatchResponse{
		Matches: matches,
	})
}

// handleAdminGearBulkDelete handles POST /api/admin/gear/bulk-delete.
func (api *AdminAPI) handleAdminGearBulkDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if len(req.IDs) == 0 {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ids is required"})
		return
	}
	if len(req.IDs) > 500 {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "too many ids (max 500)"})
		return
	}

	seen := make(map[string]struct{}, len(req.IDs))
	ids := make([]string, 0, len(req.IDs))
	for _, raw := range req.IDs {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, err := uuid.Parse(id); err != nil {
			api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id: " + id})
			return
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "ids is required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	deletedIDs, err := api.catalogStore.AdminBulkDelete(ctx, ids)
	if err != nil {
		api.logger.Error("Failed to bulk delete gear items", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to delete gear items",
		})
		return
	}

	deletedSet := make(map[string]struct{}, len(deletedIDs))
	for _, id := range deletedIDs {
		deletedSet[id] = struct{}{}
	}
	notFound := make([]string, 0)
	for _, id := range ids {
		if _, ok := deletedSet[id]; !ok {
			notFound = append(notFound, id)
		}
	}

	api.writeJSON(w, http.StatusOK, map[string]interface{}{
		"deletedIds":    deletedIDs,
		"deletedCount":  len(deletedIDs),
		"notFoundIds":   notFound,
		"notFoundCount": len(notFound),
	})
}

// handleAdminGearByID handles GET/PUT/DELETE /api/admin/gear/{id} and /api/admin/gear/{id}/image
func (api *AdminAPI) handleAdminGearByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/gear/")

	// Check if this is an image approval request
	if strings.HasSuffix(path, "/image/approve") {
		id := strings.TrimSuffix(path, "/image/approve")
		if r.Method != http.MethodPost {
			api.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		api.approveGearImage(w, r, id)
		return
	}

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

	if params.Status != nil {
		normalizedStatus := models.NormalizeCatalogStatus(*params.Status)
		if !models.IsValidCatalogStatus(normalizedStatus) {
			api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid status"})
			return
		}
		params.Status = &normalizedStatus
	}

	if params.GearType != nil {
		normalizedGearType := models.GearType(strings.ToLower(strings.TrimSpace(string(*params.GearType))))
		if normalizedGearType == "" {
			api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid gearType"})
			return
		}

		valid := false
		for _, gt := range models.AllGearTypes() {
			if gt == normalizedGearType {
				valid = true
				break
			}
		}
		if !valid {
			api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid gearType"})
			return
		}
		params.GearType = &normalizedGearType
	}

	if params.ImageStatus != nil {
		switch *params.ImageStatus {
		case models.ImageStatusMissing, models.ImageStatusScanned, models.ImageStatusApproved:
			// valid
		default:
			api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid imageStatus"})
			return
		}
	}

	if params.Specs != nil {
		var decoded any
		if err := json.Unmarshal(params.Specs, &decoded); err != nil {
			api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid specs JSON"})
			return
		}
		if _, ok := decoded.(map[string]any); !ok {
			api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "specs must be an object"})
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

// handleAdminBuilds handles GET /api/admin/builds (list builds for moderation).
func (api *AdminAPI) handleAdminBuilds(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if api.buildSvc == nil {
		api.writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "build moderation unavailable"})
		return
	}

	query := r.URL.Query()
	status := models.NormalizeBuildStatus(models.BuildStatus(strings.TrimSpace(query.Get("status"))))
	if status == "" {
		status = models.BuildStatusPendingReview
	}
	switch status {
	case models.BuildStatusPendingReview, models.BuildStatusDraft, models.BuildStatusPublished, models.BuildStatusUnpublished:
		// valid
	default:
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid status"})
		return
	}

	params := models.BuildModerationListParams{
		Query:  strings.TrimSpace(query.Get("query")),
		Status: status,
		Limit:  parseIntQuery(query.Get("limit"), 20),
		Offset: parseIntQuery(query.Get("offset"), 0),
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	response, err := api.buildSvc.ListForModeration(ctx, params)
	if err != nil {
		api.logger.Error("Failed to list moderation builds", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list builds"})
		return
	}

	api.writeJSON(w, http.StatusOK, response)
}

// handleAdminBuildByID handles /api/admin/builds/{id} actions.
func (api *AdminAPI) handleAdminBuildByID(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/admin/builds/"), "/")
	if path == "" {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "build ID required"})
		return
	}

	parts := strings.Split(path, "/")
	buildID := strings.TrimSpace(parts[0])
	if buildID == "" {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "build ID required"})
		return
	}

	if len(parts) > 1 {
		switch parts[1] {
		case "image":
			api.handleAdminBuildImage(w, r, buildID)
			return
		case "publish":
			if r.Method != http.MethodPost {
				api.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
				return
			}
			api.handlePublishAdminBuild(w, r, buildID)
			return
		default:
			api.writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown build action"})
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		api.handleGetAdminBuild(w, r, buildID)
	case http.MethodPut:
		api.handleUpdateAdminBuild(w, r, buildID)
	default:
		api.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (api *AdminAPI) handleGetAdminBuild(w http.ResponseWriter, r *http.Request, buildID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	build, err := api.buildSvc.GetForModeration(ctx, buildID)
	if err != nil {
		api.logger.Error("Failed to get moderation build", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get build"})
		return
	}
	if build == nil {
		api.writeJSON(w, http.StatusNotFound, map[string]string{"error": "build not found"})
		return
	}

	api.writeJSON(w, http.StatusOK, build)
}

func (api *AdminAPI) handleUpdateAdminBuild(w http.ResponseWriter, r *http.Request, buildID string) {
	var params models.UpdateBuildParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	updated, err := api.buildSvc.UpdateForModeration(ctx, buildID, params)
	if err != nil {
		api.logger.Error("Failed to update moderation build", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update build"})
		return
	}
	if updated == nil {
		api.writeJSON(w, http.StatusNotFound, map[string]string{"error": "build not found"})
		return
	}

	api.writeJSON(w, http.StatusOK, updated)
}

func (api *AdminAPI) handlePublishAdminBuild(w http.ResponseWriter, r *http.Request, buildID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	updated, validation, err := api.buildSvc.ApproveForModeration(ctx, buildID)
	if err != nil {
		var validationErr *builds.ValidationError
		if errors.As(err, &validationErr) {
			api.writeJSON(w, http.StatusBadRequest, models.BuildPublishResponse{Validation: validationErr.Validation})
			return
		}
		var svcErr *builds.ServiceError
		if errors.As(err, &svcErr) && strings.EqualFold(strings.TrimSpace(svcErr.Message), "build is not pending moderation") {
			api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": svcErr.Message})
			return
		}
		api.logger.Error("Failed to publish moderation build", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to publish build"})
		return
	}
	if updated == nil {
		api.writeJSON(w, http.StatusNotFound, map[string]string{"error": "build not found"})
		return
	}

	api.writeJSON(w, http.StatusOK, models.BuildPublishResponse{
		Build:      updated,
		Validation: validation,
	})
}

func (api *AdminAPI) handleAdminBuildImage(w http.ResponseWriter, r *http.Request, buildID string) {
	switch r.Method {
	case http.MethodGet:
		api.getAdminBuildImage(w, r, buildID)
	case http.MethodPost:
		api.uploadAdminBuildImage(w, r, buildID)
	case http.MethodDelete:
		api.deleteAdminBuildImage(w, r, buildID)
	default:
		api.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (api *AdminAPI) uploadAdminBuildImage(w http.ResponseWriter, r *http.Request, buildID string) {
	moderatorID := auth.GetUserID(r.Context())

	maxSize := int64(3 * 1024 * 1024)
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	if err := r.ParseMultipartForm(maxSize); err != nil {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file too large. maximum size is 2MB"})
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "image file required"})
		return
	}
	defer file.Close()

	imageData, err := io.ReadAll(file)
	if err != nil {
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read image"})
		return
	}
	if len(imageData) > 2*1024*1024 {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file too large. maximum size is 2MB"})
		return
	}
	contentType, ok := detectAllowedImageContentType(imageData)
	if !ok {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "image must be JPEG or PNG"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	decision, err := api.buildSvc.SetImageForModeration(ctx, moderatorID, models.SetBuildImageParams{
		BuildID:   buildID,
		ImageType: contentType,
		ImageData: imageData,
	})
	if err != nil {
		var svcErr *builds.ServiceError
		if errors.As(err, &svcErr) && strings.EqualFold(strings.TrimSpace(svcErr.Message), "build not found") {
			api.writeJSON(w, http.StatusNotFound, map[string]string{"error": "build not found"})
			return
		}
		api.logger.Error("Failed to upload moderation build image", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to upload image"})
		return
	}
	if decision == nil {
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to upload image"})
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

	api.writeJSON(w, http.StatusOK, map[string]string{"message": "Image uploaded successfully"})
}

func (api *AdminAPI) getAdminBuildImage(w http.ResponseWriter, r *http.Request, buildID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	imageData, imageType, err := api.buildSvc.GetImageForModeration(ctx, buildID)
	if err != nil {
		api.logger.Error("Failed to get moderation build image", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get image"})
		return
	}
	if len(imageData) == 0 {
		api.writeJSON(w, http.StatusNotFound, map[string]string{"error": "no image for this build"})
		return
	}
	if imageType == "" {
		imageType = http.DetectContentType(imageData)
	}

	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Content-Type", imageType)
	w.Header().Set("Content-Length", strconv.Itoa(len(imageData)))
	_, _ = w.Write(imageData)
}

func (api *AdminAPI) deleteAdminBuildImage(w http.ResponseWriter, r *http.Request, buildID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	if err := api.buildSvc.DeleteImageForModeration(ctx, buildID); err != nil {
		var svcErr *builds.ServiceError
		if errors.As(err, &svcErr) && strings.EqualFold(strings.TrimSpace(svcErr.Message), "build not found") {
			api.writeJSON(w, http.StatusNotFound, map[string]string{"error": "build not found"})
			return
		}
		api.logger.Error("Failed to delete moderation build image", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete image"})
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]string{"message": "Image deleted successfully"})
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

	if params.Status == nil && params.IsAdmin == nil && params.IsContentAdmin == nil && params.IsGearAdmin == nil {
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

// uploadGearImage handles POST /api/admin/gear/{id}/image.
// Supports either:
//   - multipart image uploads (moderate + persist immediately), or
//   - JSON {uploadId} for persisting a previously approved moderation token.
func (api *AdminAPI) uploadGearImage(w http.ResponseWriter, r *http.Request, id string) {
	if api.imageSvc == nil {
		api.writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "image moderation unavailable",
		})
		return
	}

	userID := auth.GetUserID(r.Context())
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Verify the gear item exists before processing upload payload.
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

	if isJSONContentType(r.Header.Get("Content-Type")) {
		api.persistApprovedGearUpload(w, r, ctx, id, userID)
		return
	}

	// Limit request body to 3MB (slightly more than 2MB limit to account for multipart overhead)
	maxSize := int64(3 * 1024 * 1024)
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	// Parse multipart form
	if err := r.ParseMultipartForm(maxSize); err != nil {
		api.logger.Error("Failed to parse multipart form", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "File too large. Maximum size is 2MB.",
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
	if len(imageData) > 2*1024*1024 {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "File too large. Maximum size is 2MB.",
		})
		return
	}
	contentType, ok := detectAllowedImageContentType(imageData)
	if !ok {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Image must be JPEG or PNG",
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

	if err := api.attachAdminGearImageAsset(ctx, id, userID, contentType, asset.ID); err != nil {
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
		logging.WithField("size", len(imageData)),
	)

	api.writeJSON(w, http.StatusOK, map[string]string{
		"status":  string(models.ImageModerationApproved),
		"message": "Image uploaded successfully",
	})
}

func (api *AdminAPI) persistApprovedGearUpload(w http.ResponseWriter, r *http.Request, ctx context.Context, id string, userID string) {
	var req struct {
		UploadID string `json:"uploadId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}

	req.UploadID = strings.TrimSpace(req.UploadID)
	if req.UploadID == "" {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "uploadId is required",
		})
		return
	}

	asset, err := api.imageSvc.PersistApprovedUpload(ctx, userID, req.UploadID, models.ImageEntityGear, id)
	if err != nil {
		switch err {
		case images.ErrPendingUploadNotFound:
			api.writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
				"error": "image approval token expired or missing",
			})
			return
		case images.ErrUploadNotApproved:
			api.writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
				"error": "image is not approved",
			})
			return
		default:
			api.logger.Error("Failed to persist approved gear image upload", logging.WithField("error", err.Error()))
			api.writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "Failed to store image",
			})
			return
		}
	}

	contentType, ok := detectAllowedImageContentType(asset.ImageBytes)
	if !ok {
		_ = api.imageSvc.Delete(ctx, asset.ID)
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Image must be JPEG or PNG",
		})
		return
	}

	if err := api.attachAdminGearImageAsset(ctx, id, userID, contentType, asset.ID); err != nil {
		api.logger.Error("Failed to store approved gear upload", logging.WithFields(map[string]interface{}{
			"gearId": id,
			"error":  err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to store image",
		})
		return
	}

	api.logger.Info("Admin attached approved moderated gear image",
		logging.WithField("gearId", id),
		logging.WithField("adminId", userID),
	)

	api.writeJSON(w, http.StatusOK, map[string]string{
		"status":  string(models.ImageModerationApproved),
		"message": "Image uploaded successfully",
	})
}

func (api *AdminAPI) attachAdminGearImageAsset(ctx context.Context, gearID string, adminUserID string, contentType string, assetID string) error {
	previousAssetID, err := api.catalogStore.SetImage(ctx, gearID, adminUserID, contentType, assetID)
	if err != nil {
		_ = api.imageSvc.Delete(ctx, assetID)
		return err
	}
	if previousAssetID != "" && previousAssetID != assetID {
		_ = api.imageSvc.Delete(ctx, previousAssetID)
	}
	return nil
}

func isJSONContentType(raw string) bool {
	contentType := strings.ToLower(strings.TrimSpace(raw))
	return strings.HasPrefix(contentType, "application/json")
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

// approveGearImage handles POST /api/admin/gear/{id}/image/approve
func (api *AdminAPI) approveGearImage(w http.ResponseWriter, r *http.Request, id string) {
	userID := auth.GetUserID(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	if err := api.catalogStore.ApproveImage(ctx, id, userID); err != nil {
		if errors.Is(err, database.ErrCatalogItemNotFound) {
			api.writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "gear item not found",
			})
			return
		}
		if errors.Is(err, database.ErrCatalogImageMissing) {
			api.writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
				"error": "gear item has no image to approve",
			})
			return
		}
		api.logger.Error("Failed to approve gear image", logging.WithFields(map[string]interface{}{
			"gearId": id,
			"error":  err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Failed to approve image",
		})
		return
	}

	api.logger.Info("Admin approved gear image",
		logging.WithField("gearId", id),
		logging.WithField("adminId", userID),
	)

	api.writeJSON(w, http.StatusOK, map[string]string{
		"status":  string(models.ImageStatusApproved),
		"message": "Image approved",
	})
}

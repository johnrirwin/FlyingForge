package httpapi

import (
	"context"
	"encoding/json"
	"errors"
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

// GearCatalogAPI handles HTTP API requests for the gear catalog
type GearCatalogAPI struct {
	catalogStore   *database.GearCatalogStore
	imageSvc       *images.Service
	authMiddleware *auth.Middleware
	logger         *logging.Logger
}

// NewGearCatalogAPI creates a new gear catalog API handler
func NewGearCatalogAPI(catalogStore *database.GearCatalogStore, imageSvc *images.Service, authMiddleware *auth.Middleware, logger *logging.Logger) *GearCatalogAPI {
	return &GearCatalogAPI{
		catalogStore:   catalogStore,
		imageSvc:       imageSvc,
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

// RegisterRoutes registers gear catalog routes on the given mux
func (api *GearCatalogAPI) RegisterRoutes(mux *http.ServeMux, corsMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	if api.authMiddleware == nil {
		api.logger.Error("Gear Catalog API routes not registered: authMiddleware is nil")
		return
	}

	// Public routes (read-only access to the shared gear catalog)
	// These are intentionally unauthenticated to allow users to browse/search
	// the crowd-sourced gear database without requiring login
	mux.HandleFunc("/api/gear-catalog/search", corsMiddleware(api.handleSearch))
	mux.HandleFunc("/api/gear-catalog/popular", corsMiddleware(api.handleGetPopular))

	// Mixed auth routes (GET is public, POST requires auth)
	// GET: delegates to handleSearch (public read access)
	// POST: requires authentication to create new catalog entries
	mux.HandleFunc("/api/gear-catalog", corsMiddleware(api.handleCatalog))

	// Authenticated routes
	mux.HandleFunc("/api/gear-catalog/", corsMiddleware(api.handleCatalogItem))
	mux.HandleFunc("/api/gear-catalog/near-matches", corsMiddleware(api.authMiddleware.RequireAuth(api.handleNearMatches)))
}

// handleSearch handles GET /api/gear-catalog/search
func (api *GearCatalogAPI) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()

	params := models.GearCatalogSearchParams{
		Query:    query.Get("q"),
		GearType: models.GearType(query.Get("gearType")),
		Brand:    query.Get("brand"),
	}

	if limit := query.Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			params.Limit = l
		}
	}
	if params.Limit == 0 {
		params.Limit = 20
	}

	if offset := query.Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			params.Offset = o
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	response, err := api.catalogStore.Search(ctx, params)
	if err != nil {
		api.logger.Error("Gear catalog search failed", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	api.writeJSON(w, http.StatusOK, response)
}

// handleGetPopular handles GET /api/gear-catalog/popular
func (api *GearCatalogAPI) handleGetPopular(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	gearType := models.GearType(query.Get("gearType"))

	limit := 10
	if l := query.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	// Cap limit to prevent excessive queries
	if limit > 100 {
		limit = 100
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	items, err := api.catalogStore.GetPopular(ctx, gearType, limit)
	if err != nil {
		api.logger.Error("Failed to get popular catalog items", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]interface{}{
		"items": items,
	})
}

// handleCatalog handles GET/POST /api/gear-catalog
func (api *GearCatalogAPI) handleCatalog(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		api.handleSearch(w, r)
	case http.MethodPost:
		api.authMiddleware.RequireAuth(api.createCatalogItem)(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// createCatalogItem handles POST /api/gear-catalog
func (api *GearCatalogAPI) createCatalogItem(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	var params models.CreateGearCatalogParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if params.GearType == "" {
		http.Error(w, "gearType is required", http.StatusBadRequest)
		return
	}
	if params.Brand == "" {
		http.Error(w, "brand is required", http.StatusBadRequest)
		return
	}
	if params.Model == "" {
		http.Error(w, "model is required", http.StatusBadRequest)
		return
	}

	// Note: imageUrl is no longer accepted from users - admin curation only

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	response, err := api.catalogStore.Create(ctx, userID, params)
	if err != nil {
		api.logger.Error("Failed to create catalog item", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	status := http.StatusCreated
	if response.Existing {
		status = http.StatusOK
	}

	api.writeJSON(w, status, response)
}

// handleCatalogItem handles GET/POST /api/gear-catalog/{id}
func (api *GearCatalogAPI) handleCatalogItem(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	path := r.URL.Path
	id := path[len("/api/gear-catalog/"):]
	if id == "" || id == "search" || id == "popular" || id == "near-matches" {
		http.NotFound(w, r)
		return
	}

	// Handle image endpoint (public, no auth required)
	if strings.HasSuffix(id, "/image") {
		id = strings.TrimSuffix(id, "/image")
		switch r.Method {
		case http.MethodGet:
			api.getGearImage(w, r, id)
		case http.MethodPost:
			api.authMiddleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
				api.uploadGearImage(w, r, id)
			})(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Handle flag endpoint
	if len(id) > 5 && id[len(id)-5:] == "/flag" {
		id = id[:len(id)-5]
		if r.Method == http.MethodPost {
			api.authMiddleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
				api.flagCatalogItem(w, r, id)
			})(w, r)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	switch r.Method {
	case http.MethodGet:
		api.getCatalogItem(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getCatalogItem handles GET /api/gear-catalog/{id}
func (api *GearCatalogAPI) getCatalogItem(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	item, err := api.catalogStore.Get(ctx, id)
	if err != nil {
		api.logger.Error("Failed to get catalog item", logging.WithFields(map[string]interface{}{
			"id":    id,
			"error": err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	if item == nil {
		http.NotFound(w, r)
		return
	}

	api.writeJSON(w, http.StatusOK, item)
}

// flagCatalogItem handles POST /api/gear-catalog/{id}/flag
func (api *GearCatalogAPI) flagCatalogItem(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	err := api.catalogStore.UpdateStatus(ctx, id, models.CatalogStatusRemoved)
	if err != nil {
		api.logger.Error("Failed to remove catalog item", logging.WithFields(map[string]interface{}{
			"id":     id,
			"reason": body.Reason,
			"error":  err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	api.logger.Info("Catalog item removed", logging.WithFields(map[string]interface{}{
		"id":     id,
		"reason": body.Reason,
	}))

	api.writeJSON(w, http.StatusOK, map[string]string{
		"status": string(models.CatalogStatusRemoved),
	})
}

// handleNearMatches handles POST /api/gear-catalog/near-matches
func (api *GearCatalogAPI) handleNearMatches(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		GearType  models.GearType `json:"gearType"`
		Brand     string          `json:"brand"`
		Model     string          `json:"model"`
		Threshold float64         `json:"threshold,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if body.GearType == "" || body.Brand == "" || body.Model == "" {
		http.Error(w, "gearType, brand, and model are required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	matches, err := api.catalogStore.FindNearMatches(ctx, body.GearType, body.Brand, body.Model, body.Threshold)
	if err != nil {
		api.logger.Error("Failed to find near matches", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	api.writeJSON(w, http.StatusOK, models.NearMatchResponse{
		Matches: matches,
	})
}

func (api *GearCatalogAPI) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// getGearImage serves the uploaded image for a gear catalog item
// Public endpoint - no auth required
func (api *GearCatalogAPI) getGearImage(w http.ResponseWriter, r *http.Request, id string) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	imageData, imageType, err := api.catalogStore.GetImage(ctx, id)
	if err != nil {
		api.logger.Error("Failed to get gear image", logging.WithFields(map[string]interface{}{
			"gearId": id,
			"error":  err.Error(),
		}))
		http.Error(w, "Failed to get image", http.StatusInternalServerError)
		return
	}

	if imageData == nil {
		http.Error(w, "No image for this gear item", http.StatusNotFound)
		return
	}
	if imageType == "" {
		imageType = http.DetectContentType(imageData)
	}

	// Set caching headers (images cached for 60 seconds - allows quick refresh after admin updates)
	w.Header().Set("Cache-Control", "public, max-age=60")
	w.Header().Set("Content-Type", imageType)
	w.Header().Set("Content-Length", strconv.Itoa(len(imageData)))
	w.Write(imageData)
}

// uploadGearImage persists a previously moderated user-submitted gear image.
// This sets image_status=scanned so admins can still curate/approve the catalog image.
func (api *GearCatalogAPI) uploadGearImage(w http.ResponseWriter, r *http.Request, id string) {
	if api.imageSvc == nil {
		api.writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "image moderation unavailable",
		})
		return
	}

	userID := auth.GetUserID(r.Context())
	if userID == "" {
		api.writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "authentication required",
		})
		return
	}

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

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

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
			api.logger.Error("Failed to persist approved gear image", logging.WithField("error", err.Error()))
			api.writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "failed to save image",
			})
			return
		}
	}

	contentType, ok := detectAllowedImageContentType(asset.ImageBytes)
	if !ok {
		_ = api.imageSvc.Delete(ctx, asset.ID)
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "only JPEG and PNG images are allowed",
		})
		return
	}

	previousAssetID, err := api.catalogStore.SetUserSubmittedImage(ctx, id, contentType, asset.ID)
	if err != nil {
		_ = api.imageSvc.Delete(ctx, asset.ID)
		if errors.Is(err, database.ErrCatalogItemNotFound) {
			api.writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "gear item not found",
			})
			return
		}
		if errors.Is(err, database.ErrCatalogImageAlreadyCurated) {
			api.writeJSON(w, http.StatusConflict, map[string]string{
				"error": "gear item already has a curated image",
			})
			return
		}
		api.logger.Error("Failed to set user-submitted gear image", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to save image",
		})
		return
	}
	if previousAssetID != "" && previousAssetID != asset.ID {
		_ = api.imageSvc.Delete(ctx, previousAssetID)
	}

	api.writeJSON(w, http.StatusOK, map[string]string{
		"status":  string(models.ImageModerationApproved),
		"message": "Image uploaded and queued for admin review",
	})
}

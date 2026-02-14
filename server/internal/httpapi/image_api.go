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
	"github.com/johnrirwin/flyingforge/internal/images"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

// ImageAPI handles shared image moderation + serving endpoints.
type ImageAPI struct {
	imageSvc       *images.Service
	authMiddleware *auth.Middleware
	logger         *logging.Logger
}

// NewImageAPI creates a new image API handler.
func NewImageAPI(imageSvc *images.Service, authMiddleware *auth.Middleware, logger *logging.Logger) *ImageAPI {
	return &ImageAPI{
		imageSvc:       imageSvc,
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

// RegisterRoutes registers image routes.
func (api *ImageAPI) RegisterRoutes(mux *http.ServeMux, corsMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("/api/images/upload", corsMiddleware(api.authMiddleware.RequireAuth(api.handleUpload)))
	mux.HandleFunc("/api/images/", corsMiddleware(api.handleGetImage))
}

func (api *ImageAPI) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())

	maxSize := int64(3 * 1024 * 1024)
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)
	if err := r.ParseMultipartForm(maxSize); err != nil {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status": "PENDING_REVIEW",
			"reason": "Invalid upload payload",
		})
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status": "PENDING_REVIEW",
			"reason": "Image file is required",
		})
		return
	}
	defer file.Close()

	if header.Size > 2*1024*1024 {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status": "PENDING_REVIEW",
			"reason": "Image must be less than 2MB",
		})
		return
	}

	entityType := models.ImageEntityOther
	if raw := strings.TrimSpace(r.FormValue("entityType")); raw != "" {
		switch models.ImageEntityType(raw) {
		case models.ImageEntityAvatar, models.ImageEntityAircraft, models.ImageEntityBuild, models.ImageEntityGear, models.ImageEntityOther:
			entityType = models.ImageEntityType(raw)
		default:
			api.writeJSON(w, http.StatusBadRequest, map[string]string{
				"status": "PENDING_REVIEW",
				"reason": "Invalid entity type",
			})
			return
		}
	}

	imageData, err := io.ReadAll(file)
	if err != nil {
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status": "PENDING_REVIEW",
			"reason": "Failed to read image",
		})
		return
	}
	if len(imageData) > 2*1024*1024 {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status": "PENDING_REVIEW",
			"reason": "Image must be less than 2MB",
		})
		return
	}
	if _, ok := detectAllowedImageContentType(imageData); !ok {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{
			"status": "PENDING_REVIEW",
			"reason": "Only JPEG and PNG images are allowed",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	decision, uploadID, err := api.imageSvc.ModerateUpload(ctx, userID, entityType, imageData)
	if err != nil {
		api.logger.Error("image moderation upload failed", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"status": string(models.ImageModerationPendingReview),
			"reason": "Unable to verify right now",
		})
		return
	}

	response := map[string]string{
		"status": string(decision.Status),
	}
	if decision.Reason != "" {
		response["reason"] = decision.Reason
	}
	if uploadID != "" {
		response["uploadId"] = uploadID
	}

	api.writeJSON(w, http.StatusOK, response)
}

func (api *ImageAPI) handleGetImage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/api/images/")
	id = strings.TrimSuffix(id, "/")
	if id == "" {
		http.Error(w, "image id required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	asset, err := api.imageSvc.Load(ctx, id)
	if err != nil {
		api.logger.Error("failed to load image asset", logging.WithField("error", err.Error()))
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}
	if asset == nil || asset.Status != models.ImageModerationApproved || asset.EntityType != models.ImageEntityAvatar {
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}

	contentType, ok := detectAllowedImageContentType(asset.ImageBytes)
	if !ok {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.Error(w, "unsupported media type", http.StatusUnsupportedMediaType)
		return
	}

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(asset.ImageBytes)))
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(http.StatusOK)
	w.Write(asset.ImageBytes)
}

func (api *ImageAPI) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/builds"
	"github.com/johnrirwin/flyingforge/internal/images"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
	"github.com/johnrirwin/flyingforge/internal/ratelimit"
)

// BuildAPI handles public, temporary, and authenticated build endpoints.
type BuildAPI struct {
	service         *builds.Service
	authMiddleware  *auth.Middleware
	tempRateLimiter ratelimit.RateLimiter
	logger          *logging.Logger
}

// NewBuildAPI creates a build API handler.
func NewBuildAPI(service *builds.Service, authMiddleware *auth.Middleware, tempRateLimiter ratelimit.RateLimiter, logger *logging.Logger) *BuildAPI {
	return &BuildAPI{
		service:         service,
		authMiddleware:  authMiddleware,
		tempRateLimiter: tempRateLimiter,
		logger:          logger,
	}
}

// RegisterRoutes registers build routes.
func (api *BuildAPI) RegisterRoutes(mux *http.ServeMux, corsMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("/api/public/builds", corsMiddleware(api.authMiddleware.OptionalAuth(api.handlePublicBuilds)))
	mux.HandleFunc("/api/public/builds/", corsMiddleware(api.authMiddleware.OptionalAuth(api.handlePublicBuildItem)))

	mux.HandleFunc("/api/builds/temp", corsMiddleware(api.authMiddleware.OptionalAuth(api.handleTempCollection)))
	mux.HandleFunc("/api/builds/temp/", corsMiddleware(api.handleTempItem))

	mux.HandleFunc("/api/builds/from-aircraft/", corsMiddleware(api.authMiddleware.RequireAuth(api.handleBuildFromAircraft)))
	mux.HandleFunc("/api/builds", corsMiddleware(api.authMiddleware.RequireAuth(api.handleBuildCollection)))
	mux.HandleFunc("/api/builds/", corsMiddleware(api.authMiddleware.RequireAuth(api.handleBuildItem)))
}

func (api *BuildAPI) handlePublicBuilds(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	params := api.parseListParams(r)
	viewerUserID := auth.GetUserID(r.Context())
	response, err := api.service.ListPublic(r.Context(), viewerUserID, params)
	if err != nil {
		api.logger.Error("List public builds failed", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to load builds")
		return
	}

	api.writeJSON(w, http.StatusOK, response)
}

func (api *BuildAPI) handlePublicBuildItem(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/public/builds/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		api.writeError(w, http.StatusBadRequest, "invalid_id", "build id is required")
		return
	}
	buildID := strings.TrimSpace(parts[0])

	if len(parts) > 1 {
		switch parts[1] {
		case "image":
			if r.Method != http.MethodGet {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			api.getPublicBuildImage(w, r, buildID)
			return
		default:
			api.writeError(w, http.StatusNotFound, "not_found", "unknown build action")
			return
		}
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	viewerUserID := auth.GetUserID(r.Context())
	build, err := api.service.GetPublic(r.Context(), buildID, viewerUserID)
	if err != nil {
		api.logger.Error("Get public build failed", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to load build")
		return
	}
	if build == nil {
		api.writeError(w, http.StatusNotFound, "not_found", "build not found")
		return
	}

	api.writeJSON(w, http.StatusOK, build)
}

func (api *BuildAPI) handleTempCollection(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if api.tempRateLimiter != nil {
		if !api.tempRateLimiter.Allow(api.getClientIP(r)) {
			api.writeError(w, http.StatusTooManyRequests, "rate_limited", "too many temporary builds created from this IP")
			return
		}
	}

	var params models.CreateBuildParams
	if err := decodeJSONAllowEmpty(r, &params); err != nil {
		api.writeError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	ownerUserID := auth.GetUserID(r.Context())
	response, err := api.service.CreateTemp(r.Context(), ownerUserID, params)
	if err != nil {
		api.logger.Error("Create temp build failed", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to create temporary build")
		return
	}

	api.writeJSON(w, http.StatusCreated, response)
}

func (api *BuildAPI) handleTempItem(w http.ResponseWriter, r *http.Request) {
	path := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/builds/temp/"), "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		api.writeError(w, http.StatusBadRequest, "invalid_token", "temp build token is required")
		return
	}
	token := strings.TrimSpace(parts[0])

	if len(parts) > 1 {
		switch parts[1] {
		case "share":
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			shared, err := api.service.ShareTempByToken(r.Context(), token)
			if err != nil {
				api.logger.Error("Share temp build failed", logging.WithField("error", err.Error()))
				api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to share temporary build")
				return
			}
			if shared == nil {
				api.writeError(w, http.StatusNotFound, "not_found", "temporary build not found or expired")
				return
			}

			api.writeJSON(w, http.StatusOK, shared)
			return
		default:
			api.writeError(w, http.StatusNotFound, "not_found", "unknown temporary build action")
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		build, err := api.service.GetTempByToken(r.Context(), token)
		if err != nil {
			api.logger.Error("Get temp build failed", logging.WithField("error", err.Error()))
			api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to load temporary build")
			return
		}
		if build == nil {
			api.writeError(w, http.StatusNotFound, "not_found", "temporary build not found or expired")
			return
		}
		api.writeJSON(w, http.StatusOK, build)
	case http.MethodPut:
		var params models.UpdateBuildParams
		if err := decodeJSONAllowEmpty(r, &params); err != nil {
			api.writeError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
			return
		}

		updated, err := api.service.UpdateTempByToken(r.Context(), token, params)
		if err != nil {
			api.logger.Error("Update temp build failed", logging.WithField("error", err.Error()))
			api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to update temporary build")
			return
		}
		if updated == nil {
			api.writeError(w, http.StatusNotFound, "not_found", "temporary build not found or expired")
			return
		}

		api.writeJSON(w, http.StatusOK, updated)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (api *BuildAPI) handleBuildCollection(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	switch r.Method {
	case http.MethodGet:
		params := api.parseListParams(r)
		response, err := api.service.ListByOwner(r.Context(), userID, params)
		if err != nil {
			api.logger.Error("List my builds failed", logging.WithField("error", err.Error()))
			api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to list builds")
			return
		}
		api.writeJSON(w, http.StatusOK, response)
	case http.MethodPost:
		var params models.CreateBuildParams
		if err := decodeJSONAllowEmpty(r, &params); err != nil {
			api.writeError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
			return
		}

		build, err := api.service.CreateDraft(r.Context(), userID, params)
		if err != nil {
			api.logger.Error("Create draft build failed", logging.WithField("error", err.Error()))
			api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to create build")
			return
		}
		api.writeJSON(w, http.StatusCreated, build)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (api *BuildAPI) handleBuildFromAircraft(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())
	aircraftID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/builds/from-aircraft/"))
	if aircraftID == "" {
		api.writeError(w, http.StatusBadRequest, "invalid_aircraft", "aircraft id is required")
		return
	}

	build, err := api.service.CreateDraftFromAircraft(r.Context(), userID, aircraftID)
	if err != nil {
		api.logger.Error("Create build from aircraft failed", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to create build from aircraft")
		return
	}
	if build == nil {
		api.writeError(w, http.StatusNotFound, "not_found", "aircraft not found")
		return
	}

	api.writeJSON(w, http.StatusCreated, build)
}

func (api *BuildAPI) handleBuildItem(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	path := strings.TrimPrefix(r.URL.Path, "/api/builds/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		api.writeError(w, http.StatusBadRequest, "invalid_id", "build id is required")
		return
	}
	buildID := strings.TrimSpace(parts[0])

	if len(parts) > 1 {
		switch parts[1] {
		case "image":
			switch r.Method {
			case http.MethodGet:
				api.getBuildImage(w, r, buildID, userID)
			case http.MethodPost, http.MethodPut:
				api.uploadBuildImage(w, r, buildID, userID)
			case http.MethodDelete:
				api.deleteBuildImage(w, r, buildID, userID)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		case "reaction":
			switch r.Method {
			case http.MethodPost, http.MethodPut:
				var params models.SetBuildReactionParams
				if err := decodeJSONAllowEmpty(r, &params); err != nil {
					api.writeError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
					return
				}

				build, err := api.service.SetReaction(r.Context(), buildID, userID, params.Reaction)
				if err != nil {
					var svcErr *builds.ServiceError
					if errors.As(err, &svcErr) {
						api.writeError(w, http.StatusBadRequest, "invalid_reaction", svcErr.Message)
						return
					}
					api.logger.Error("Set build reaction failed", logging.WithFields(map[string]interface{}{
						"build_id": buildID,
						"user_id":  userID,
						"error":    err.Error(),
					}))
					api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to set build reaction")
					return
				}
				if build == nil {
					api.writeError(w, http.StatusNotFound, "not_found", "build not found")
					return
				}
				api.writeJSON(w, http.StatusOK, build)
			case http.MethodDelete:
				build, err := api.service.ClearReaction(r.Context(), buildID, userID)
				if err != nil {
					var svcErr *builds.ServiceError
					if errors.As(err, &svcErr) {
						api.writeError(w, http.StatusBadRequest, "invalid_reaction", svcErr.Message)
						return
					}
					api.logger.Error("Clear build reaction failed", logging.WithFields(map[string]interface{}{
						"build_id": buildID,
						"user_id":  userID,
						"error":    err.Error(),
					}))
					api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to clear build reaction")
					return
				}
				if build == nil {
					api.writeError(w, http.StatusNotFound, "not_found", "build not found")
					return
				}
				api.writeJSON(w, http.StatusOK, build)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		case "publish":
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			build, validation, err := api.service.Publish(r.Context(), buildID, userID)
			if err != nil {
				var validationErr *builds.ValidationError
				if errors.As(err, &validationErr) {
					api.writeJSON(w, http.StatusBadRequest, models.BuildPublishResponse{Validation: validationErr.Validation})
					return
				}
				api.logger.Error("Publish build failed", logging.WithField("error", err.Error()))
				api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to publish build")
				return
			}
			if build == nil {
				api.writeError(w, http.StatusNotFound, "not_found", "build not found")
				return
			}
			api.writeJSON(w, http.StatusOK, models.BuildPublishResponse{Build: build, Validation: validation})
			return
		case "unpublish":
			if r.Method != http.MethodPost {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			build, err := api.service.Unpublish(r.Context(), buildID, userID)
			if err != nil {
				api.logger.Error("Unpublish build failed", logging.WithField("error", err.Error()))
				api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to unpublish build")
				return
			}
			if build == nil {
				api.writeError(w, http.StatusNotFound, "not_found", "build not found")
				return
			}
			api.writeJSON(w, http.StatusOK, build)
			return
		default:
			api.writeError(w, http.StatusNotFound, "not_found", "unknown build action")
			return
		}
	}

	switch r.Method {
	case http.MethodGet:
		build, err := api.service.GetByOwner(r.Context(), buildID, userID)
		if err != nil {
			api.logger.Error("Get build failed", logging.WithField("error", err.Error()))
			api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to load build")
			return
		}
		if build == nil {
			api.writeError(w, http.StatusNotFound, "not_found", "build not found")
			return
		}
		api.writeJSON(w, http.StatusOK, build)
	case http.MethodPut:
		var params models.UpdateBuildParams
		if err := decodeJSONAllowEmpty(r, &params); err != nil {
			api.writeError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
			return
		}
		build, err := api.service.UpdateByOwner(r.Context(), buildID, userID, params)
		if err != nil {
			api.logger.Error("Update build failed", logging.WithField("error", err.Error()))
			api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to update build")
			return
		}
		if build == nil {
			api.writeError(w, http.StatusNotFound, "not_found", "build not found")
			return
		}
		api.writeJSON(w, http.StatusOK, build)
	case http.MethodDelete:
		deleted, err := api.service.DeleteByOwner(r.Context(), buildID, userID)
		if err != nil {
			var svcErr *builds.ServiceError
			if errors.As(err, &svcErr) {
				api.writeError(w, http.StatusBadRequest, "invalid_request", svcErr.Message)
				return
			}
			api.logger.Error("Delete build failed", logging.WithField("error", err.Error()))
			api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete build")
			return
		}
		if !deleted {
			api.writeError(w, http.StatusNotFound, "not_found", "build not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (api *BuildAPI) uploadBuildImage(w http.ResponseWriter, r *http.Request, buildID string, userID string) {
	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))

	if strings.HasPrefix(contentType, "application/json") {
		var req struct {
			UploadID string `json:"uploadId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			api.writeError(w, http.StatusBadRequest, "invalid_body", "invalid request body")
			return
		}
		req.UploadID = strings.TrimSpace(req.UploadID)
		if req.UploadID == "" {
			api.writeError(w, http.StatusBadRequest, "invalid_upload", "uploadId is required")
			return
		}

		decision, err := api.service.SetImage(r.Context(), userID, models.SetBuildImageParams{
			BuildID:  buildID,
			UploadID: req.UploadID,
		})
		if err != nil {
			switch err {
			case images.ErrPendingUploadNotFound:
				api.writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
					"status": string(models.ImageModerationRejected),
					"reason": "Image approval token expired or missing",
					"error":  "image approval token expired or missing",
				})
				return
			case images.ErrUploadNotApproved:
				api.writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
					"status": string(models.ImageModerationRejected),
					"reason": "Image is not approved",
					"error":  "image is not approved",
				})
				return
			default:
				var svcErr *builds.ServiceError
				if errors.As(err, &svcErr) {
					switch strings.ToLower(strings.TrimSpace(svcErr.Message)) {
					case "build not found":
						api.writeError(w, http.StatusNotFound, "not_found", "build not found")
					default:
						api.writeError(w, http.StatusBadRequest, "invalid_request", svcErr.Message)
					}
					return
				}
				api.logger.Error("Set build image from approved upload failed", logging.WithFields(map[string]interface{}{
					"build_id": buildID,
					"error":    err.Error(),
				}))
				api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to set build image")
				return
			}
		}
		if decision == nil {
			api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to set build image")
			return
		}

		api.writeJSON(w, http.StatusOK, map[string]string{
			"status":  string(decision.Status),
			"message": "Image uploaded successfully",
		})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 3*1024*1024)
	if err := r.ParseMultipartForm(3 * 1024 * 1024); err != nil {
		api.writeError(w, http.StatusBadRequest, "invalid_upload", "file too large or invalid form")
		return
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		api.writeError(w, http.StatusBadRequest, "missing_image", "image file required")
		return
	}
	defer file.Close()

	imageData, err := io.ReadAll(file)
	if err != nil {
		api.writeError(w, http.StatusInternalServerError, "read_error", "failed to read image")
		return
	}
	if len(imageData) > 2*1024*1024 {
		api.writeError(w, http.StatusBadRequest, "invalid_upload", "image must be less than 2MB")
		return
	}
	detectedContentType, ok := detectAllowedImageContentType(imageData)
	if !ok {
		api.writeError(w, http.StatusBadRequest, "invalid_upload", "image must be JPEG or PNG")
		return
	}

	decision, err := api.service.SetImage(r.Context(), userID, models.SetBuildImageParams{
		BuildID:   buildID,
		ImageType: detectedContentType,
		ImageData: imageData,
	})
	if err != nil {
		var svcErr *builds.ServiceError
		if errors.As(err, &svcErr) {
			switch strings.ToLower(strings.TrimSpace(svcErr.Message)) {
			case "build not found":
				api.writeError(w, http.StatusNotFound, "not_found", "build not found")
			default:
				api.writeError(w, http.StatusBadRequest, "invalid_request", svcErr.Message)
			}
			return
		}
		api.logger.Error("Set build image failed", logging.WithFields(map[string]interface{}{
			"build_id": buildID,
			"error":    err.Error(),
		}))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to set build image")
		return
	}
	if decision == nil {
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to set build image")
		return
	}
	if decision.Status != models.ImageModerationApproved {
		statusCode := http.StatusUnprocessableEntity
		if decision.Status == models.ImageModerationPendingReview {
			statusCode = http.StatusServiceUnavailable
		}
		api.writeJSON(w, statusCode, map[string]string{
			"status": string(decision.Status),
			"reason": decision.Reason,
			"error":  decision.Reason,
		})
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]string{
		"status":  string(decision.Status),
		"message": "Image uploaded successfully",
	})
}

func (api *BuildAPI) getBuildImage(w http.ResponseWriter, r *http.Request, buildID string, userID string) {
	imageData, imageType, err := api.service.GetImage(r.Context(), buildID, userID)
	if err != nil {
		api.logger.Error("Get build image failed", logging.WithFields(map[string]interface{}{
			"build_id": buildID,
			"error":    err.Error(),
		}))
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}
	if len(imageData) == 0 {
		http.Error(w, "no image for this build", http.StatusNotFound)
		return
	}

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", imageType)
	w.Header().Set("Content-Length", strconv.Itoa(len(imageData)))
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(imageData)
}

func (api *BuildAPI) getPublicBuildImage(w http.ResponseWriter, r *http.Request, buildID string) {
	imageData, imageType, err := api.service.GetPublicImage(r.Context(), buildID)
	if err != nil {
		api.logger.Error("Get public build image failed", logging.WithFields(map[string]interface{}{
			"build_id": buildID,
			"error":    err.Error(),
		}))
		http.Error(w, "image not found", http.StatusNotFound)
		return
	}
	if len(imageData) == 0 {
		http.Error(w, "no image for this build", http.StatusNotFound)
		return
	}

	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", imageType)
	w.Header().Set("Content-Length", strconv.Itoa(len(imageData)))
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(imageData)
}

func (api *BuildAPI) deleteBuildImage(w http.ResponseWriter, r *http.Request, buildID string, userID string) {
	if err := api.service.DeleteImage(r.Context(), buildID, userID); err != nil {
		var svcErr *builds.ServiceError
		if errors.As(err, &svcErr) && strings.EqualFold(strings.TrimSpace(svcErr.Message), "build not found") {
			api.writeError(w, http.StatusNotFound, "not_found", "build not found")
			return
		}
		api.logger.Error("Delete build image failed", logging.WithFields(map[string]interface{}{
			"build_id": buildID,
			"error":    err.Error(),
		}))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to delete build image")
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Image deleted successfully",
	})
}

func (api *BuildAPI) parseListParams(r *http.Request) models.BuildListParams {
	query := r.URL.Query()

	params := models.BuildListParams{
		Sort:        models.BuildSort(strings.TrimSpace(query.Get("sort"))),
		FrameFilter: strings.TrimSpace(query.Get("frameFilter")),
	}
	if params.Sort == "" {
		params.Sort = models.BuildSortNewest
	}

	if limit := strings.TrimSpace(query.Get("limit")); limit != "" {
		if parsed, err := strconv.Atoi(limit); err == nil {
			params.Limit = parsed
		}
	}
	if offset := strings.TrimSpace(query.Get("offset")); offset != "" {
		if parsed, err := strconv.Atoi(offset); err == nil {
			params.Offset = parsed
		}
	}

	return params
}

func (api *BuildAPI) getClientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return xff
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	remoteAddr := strings.TrimSpace(r.RemoteAddr)
	if remoteAddr == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil && strings.TrimSpace(host) != "" {
		return host
	}
	return remoteAddr
}

func decodeJSONAllowEmpty(r *http.Request, dst interface{}) error {
	if r.Body == nil {
		return nil
	}
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return nil
}

func (api *BuildAPI) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (api *BuildAPI) writeError(w http.ResponseWriter, status int, code, message string) {
	api.writeJSON(w, status, map[string]string{
		"code":    code,
		"message": message,
	})
}

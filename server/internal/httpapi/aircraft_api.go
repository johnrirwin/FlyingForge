package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/johnrirwin/flyingforge/internal/aircraft"
	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

// AircraftAPI handles HTTP API requests for aircraft management
type AircraftAPI struct {
	aircraftSvc    *aircraft.Service
	authMiddleware *auth.Middleware
	logger         *logging.Logger
}

// NewAircraftAPI creates a new aircraft API handler
func NewAircraftAPI(aircraftSvc *aircraft.Service, authMiddleware *auth.Middleware, logger *logging.Logger) *AircraftAPI {
	return &AircraftAPI{
		aircraftSvc:    aircraftSvc,
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

// RegisterRoutes registers aircraft routes on the given mux
func (api *AircraftAPI) RegisterRoutes(mux *http.ServeMux, corsMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	// Aircraft routes (require authentication)
	mux.HandleFunc("/api/aircraft", corsMiddleware(api.authMiddleware.RequireAuth(api.handleAircraft)))
	mux.HandleFunc("/api/aircraft/", corsMiddleware(api.authMiddleware.RequireAuth(api.handleAircraftItem)))
}

// handleAircraft handles list and create operations
func (api *AircraftAPI) handleAircraft(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		api.listAircraft(w, r)
	case http.MethodPost:
		api.createAircraft(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// listAircraft returns all aircraft for the authenticated user
func (api *AircraftAPI) listAircraft(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	query := r.URL.Query()

	params := models.AircraftListParams{
		Type: models.AircraftType(query.Get("type")),
	}

	if limit := query.Get("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			params.Limit = l
		}
	}

	if offset := query.Get("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			params.Offset = o
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	response, err := api.aircraftSvc.List(ctx, userID, params)
	if err != nil {
		api.logger.Error("Aircraft list failed", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	api.writeJSON(w, http.StatusOK, response)
}

// createAircraft creates a new aircraft
func (api *AircraftAPI) createAircraft(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	var params models.CreateAircraftParams

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	aircraft, err := api.aircraftSvc.Create(ctx, userID, params)
	if err != nil {
		api.logger.Error("Create aircraft failed", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	api.writeJSON(w, http.StatusCreated, aircraft)
}

// handleAircraftItem handles single aircraft operations (get, update, delete, components, receiver)
func (api *AircraftAPI) handleAircraftItem(w http.ResponseWriter, r *http.Request) {
	// Extract path after /api/aircraft/
	path := strings.TrimPrefix(r.URL.Path, "/api/aircraft/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Aircraft ID required", http.StatusBadRequest)
		return
	}

	aircraftID := parts[0]

	// Check for sub-resources: /api/aircraft/{id}/components or /api/aircraft/{id}/receiver
	if len(parts) > 1 {
		switch parts[1] {
		case "components":
			api.handleComponents(w, r, aircraftID)
			return
		case "receiver":
			api.handleReceiver(w, r, aircraftID)
			return
		case "details":
			api.getAircraftDetails(w, r, aircraftID)
			return
		case "image":
			api.handleImage(w, r, aircraftID)
			return
		default:
			http.Error(w, "Unknown resource", http.StatusNotFound)
			return
		}
	}

	// Handle base aircraft CRUD
	switch r.Method {
	case http.MethodGet:
		api.getAircraft(w, r, aircraftID)
	case http.MethodPut, http.MethodPatch:
		api.updateAircraft(w, r, aircraftID)
	case http.MethodDelete:
		api.deleteAircraft(w, r, aircraftID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getAircraft retrieves a single aircraft
func (api *AircraftAPI) getAircraft(w http.ResponseWriter, r *http.Request, id string) {
	userID := auth.GetUserID(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	aircraft, err := api.aircraftSvc.Get(ctx, id, userID)
	if err != nil {
		api.logger.Error("Get aircraft failed", logging.WithFields(map[string]interface{}{
			"id":    id,
			"error": err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	if aircraft == nil {
		http.Error(w, "Aircraft not found", http.StatusNotFound)
		return
	}

	api.writeJSON(w, http.StatusOK, aircraft)
}

// getAircraftDetails retrieves full aircraft details including components and receiver settings
func (api *AircraftAPI) getAircraftDetails(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	details, err := api.aircraftSvc.GetDetails(ctx, id, userID)
	if err != nil {
		api.logger.Error("Get aircraft details failed", logging.WithFields(map[string]interface{}{
			"id":    id,
			"error": err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	if details == nil || details.Aircraft.ID == "" {
		http.Error(w, "Aircraft not found", http.StatusNotFound)
		return
	}

	api.writeJSON(w, http.StatusOK, details)
}

// updateAircraft updates an aircraft
func (api *AircraftAPI) updateAircraft(w http.ResponseWriter, r *http.Request, id string) {
	userID := auth.GetUserID(r.Context())

	var params models.UpdateAircraftParams

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	params.ID = id

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	aircraft, err := api.aircraftSvc.Update(ctx, userID, params)
	if err != nil {
		api.logger.Error("Update aircraft failed", logging.WithFields(map[string]interface{}{
			"id":    id,
			"error": err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	api.writeJSON(w, http.StatusOK, aircraft)
}

// deleteAircraft deletes an aircraft
func (api *AircraftAPI) deleteAircraft(w http.ResponseWriter, r *http.Request, id string) {
	userID := auth.GetUserID(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	if err := api.aircraftSvc.Delete(ctx, id, userID); err != nil {
		api.logger.Error("Delete aircraft failed", logging.WithFields(map[string]interface{}{
			"id":    id,
			"error": err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleComponents handles aircraft component operations
func (api *AircraftAPI) handleComponents(w http.ResponseWriter, r *http.Request, aircraftID string) {
	switch r.Method {
	case http.MethodGet:
		api.getComponents(w, r, aircraftID)
	case http.MethodPost, http.MethodPut:
		api.setComponent(w, r, aircraftID)
	case http.MethodDelete:
		api.removeComponent(w, r, aircraftID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getComponents retrieves all components for an aircraft
func (api *AircraftAPI) getComponents(w http.ResponseWriter, r *http.Request, aircraftID string) {
	userID := auth.GetUserID(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	components, err := api.aircraftSvc.GetComponents(ctx, aircraftID, userID)
	if err != nil {
		api.logger.Error("Get components failed", logging.WithFields(map[string]interface{}{
			"aircraft_id": aircraftID,
			"error":       err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]interface{}{
		"components": components,
		"count":      len(components),
	})
}

// setComponent sets a component on an aircraft (with optional auto-add gear)
func (api *AircraftAPI) setComponent(w http.ResponseWriter, r *http.Request, aircraftID string) {
	userID := auth.GetUserID(r.Context())

	var params models.SetComponentParams

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	params.AircraftID = aircraftID

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	component, err := api.aircraftSvc.SetComponent(ctx, userID, params)
	if err != nil {
		api.logger.Error("Set component failed", logging.WithFields(map[string]interface{}{
			"aircraft_id": aircraftID,
			"category":    params.Category,
			"error":       err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	if component == nil {
		// Component was removed
		w.WriteHeader(http.StatusNoContent)
		return
	}

	api.writeJSON(w, http.StatusOK, component)
}

// removeComponent removes a component from an aircraft
func (api *AircraftAPI) removeComponent(w http.ResponseWriter, r *http.Request, aircraftID string) {
	userID := auth.GetUserID(r.Context())

	category := r.URL.Query().Get("category")
	if category == "" {
		http.Error(w, "Category required", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Use SetComponent with empty inventory item ID to remove
	params := models.SetComponentParams{
		AircraftID: aircraftID,
		Category:   models.ComponentCategory(category),
	}

	_, err := api.aircraftSvc.SetComponent(ctx, userID, params)
	if err != nil {
		api.logger.Error("Remove component failed", logging.WithFields(map[string]interface{}{
			"aircraft_id": aircraftID,
			"category":    category,
			"error":       err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleReceiver handles receiver settings operations
func (api *AircraftAPI) handleReceiver(w http.ResponseWriter, r *http.Request, aircraftID string) {
	switch r.Method {
	case http.MethodGet:
		api.getReceiverSettings(w, r, aircraftID)
	case http.MethodPost, http.MethodPut:
		api.setReceiverSettings(w, r, aircraftID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getReceiverSettings retrieves receiver settings for an aircraft
func (api *AircraftAPI) getReceiverSettings(w http.ResponseWriter, r *http.Request, aircraftID string) {
	userID := auth.GetUserID(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	settings, err := api.aircraftSvc.GetReceiverSettings(ctx, aircraftID, userID)
	if err != nil {
		api.logger.Error("Get receiver settings failed", logging.WithFields(map[string]interface{}{
			"aircraft_id": aircraftID,
			"error":       err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	if settings == nil {
		// Return empty settings
		api.writeJSON(w, http.StatusOK, map[string]interface{}{
			"aircraftId": aircraftID,
			"settings":   map[string]interface{}{},
		})
		return
	}

	api.writeJSON(w, http.StatusOK, settings)
}

// setReceiverSettings sets receiver settings for an aircraft
func (api *AircraftAPI) setReceiverSettings(w http.ResponseWriter, r *http.Request, aircraftID string) {
	userID := auth.GetUserID(r.Context())

	var params models.SetReceiverSettingsParams

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	params.AircraftID = aircraftID

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	settings, err := api.aircraftSvc.SetReceiverSettings(ctx, userID, params)
	if err != nil {
		api.logger.Error("Set receiver settings failed", logging.WithFields(map[string]interface{}{
			"aircraft_id": aircraftID,
			"error":       err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	api.writeJSON(w, http.StatusOK, settings)
}

// handleImage handles image upload, retrieval, and deletion
func (api *AircraftAPI) handleImage(w http.ResponseWriter, r *http.Request, aircraftID string) {
	switch r.Method {
	case http.MethodGet:
		api.getImage(w, r, aircraftID)
	case http.MethodPost, http.MethodPut:
		api.uploadImage(w, r, aircraftID)
	case http.MethodDelete:
		api.deleteImage(w, r, aircraftID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// uploadImage handles image upload for an aircraft
func (api *AircraftAPI) uploadImage(w http.ResponseWriter, r *http.Request, aircraftID string) {
	userID := auth.GetUserID(r.Context())

	// Limit request body to 6MB (slightly more than our 5MB limit to account for multipart overhead)
	r.Body = http.MaxBytesReader(w, r.Body, 6*1024*1024)

	// Parse multipart form
	if err := r.ParseMultipartForm(6 * 1024 * 1024); err != nil {
		api.logger.Error("Failed to parse multipart form", logging.WithField("error", err.Error()))
		http.Error(w, "File too large or invalid form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		api.logger.Error("Failed to get image from form", logging.WithField("error", err.Error()))
		http.Error(w, "Image file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Validate content type
	contentType := header.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" {
		http.Error(w, "Image must be JPEG or PNG", http.StatusBadRequest)
		return
	}

	// Read image data
	imageData := make([]byte, header.Size)
	if _, err := file.Read(imageData); err != nil {
		api.logger.Error("Failed to read image data", logging.WithField("error", err.Error()))
		http.Error(w, "Failed to read image", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	params := models.SetAircraftImageParams{
		AircraftID: aircraftID,
		ImageType:  contentType,
		ImageData:  imageData,
	}

	if err := api.aircraftSvc.SetImage(ctx, userID, params); err != nil {
		api.logger.Error("Failed to set aircraft image", logging.WithFields(map[string]interface{}{
			"aircraft_id": aircraftID,
			"error":       err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Image uploaded successfully",
	})
}

// getImage retrieves an aircraft's image
func (api *AircraftAPI) getImage(w http.ResponseWriter, r *http.Request, aircraftID string) {
	userID := auth.GetUserID(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	imageData, imageType, err := api.aircraftSvc.GetImage(ctx, aircraftID, userID)
	if err != nil {
		api.logger.Error("Failed to get aircraft image", logging.WithFields(map[string]interface{}{
			"aircraft_id": aircraftID,
			"error":       err.Error(),
		}))
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}

	if imageData == nil {
		http.Error(w, "No image for this aircraft", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", imageType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.WriteHeader(http.StatusOK)
	w.Write(imageData)
}

// deleteImage removes an aircraft's image
func (api *AircraftAPI) deleteImage(w http.ResponseWriter, r *http.Request, aircraftID string) {
	userID := auth.GetUserID(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	if err := api.aircraftSvc.DeleteImage(ctx, aircraftID, userID); err != nil {
		api.logger.Error("Failed to delete aircraft image", logging.WithFields(map[string]interface{}{
			"aircraft_id": aircraftID,
			"error":       err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]string{
		"message": "Image deleted successfully",
	})
}

func (api *AircraftAPI) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/johnrirwin/rotorlife/internal/auth"
	"github.com/johnrirwin/rotorlife/internal/logging"
	"github.com/johnrirwin/rotorlife/internal/models"
	radiosvc "github.com/johnrirwin/rotorlife/internal/radio"
)

// RadioAPI handles HTTP API requests for radios and backups
type RadioAPI struct {
	radioSvc       *radiosvc.Service
	authMiddleware *auth.Middleware
	logger         *logging.Logger
}

// NewRadioAPI creates a new radio API handler
func NewRadioAPI(radioSvc *radiosvc.Service, authMiddleware *auth.Middleware, logger *logging.Logger) *RadioAPI {
	return &RadioAPI{
		radioSvc:       radioSvc,
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

// RegisterRoutes registers radio routes on the given mux
func (api *RadioAPI) RegisterRoutes(mux *http.ServeMux, corsMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	// Radio models (public)
	mux.HandleFunc("/api/radio/models", corsMiddleware(api.handleGetRadioModels))

	// Radios (require authentication)
	mux.HandleFunc("/api/radios", corsMiddleware(api.authMiddleware.RequireAuth(api.handleRadios)))
	mux.HandleFunc("/api/radios/", corsMiddleware(api.authMiddleware.RequireAuth(api.handleRadioItem)))

	// Backups (require authentication)
	mux.HandleFunc("/api/radios/", corsMiddleware(api.authMiddleware.RequireAuth(api.handleRadioItem)))
}

// handleGetRadioModels returns the list of available radio models
func (api *RadioAPI) handleGetRadioModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	response := api.radioSvc.GetRadioModels(ctx)
	api.writeJSON(w, http.StatusOK, response)
}

// handleRadios handles requests to /api/radios
func (api *RadioAPI) handleRadios(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	switch r.Method {
	case http.MethodGet:
		api.handleListRadios(w, r, userID)
	case http.MethodPost:
		api.handleCreateRadio(w, r, userID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListRadios lists all radios for the user
func (api *RadioAPI) handleListRadios(w http.ResponseWriter, r *http.Request, userID string) {
	query := r.URL.Query()

	params := models.RadioListParams{}
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

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	response, err := api.radioSvc.ListRadios(ctx, userID, params)
	if err != nil {
		api.logger.Error("Failed to list radios", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	api.writeJSON(w, http.StatusOK, response)
}

// handleCreateRadio creates a new radio
func (api *RadioAPI) handleCreateRadio(w http.ResponseWriter, r *http.Request, userID string) {
	var params models.CreateRadioParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	radio, err := api.radioSvc.CreateRadio(ctx, userID, params)
	if err != nil {
		status := http.StatusInternalServerError
		if _, ok := err.(*radiosvc.ServiceError); ok {
			status = http.StatusBadRequest
		}
		api.writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	api.writeJSON(w, http.StatusCreated, radio)
}

// handleRadioItem handles requests to /api/radios/{id} and /api/radios/{id}/backups/*
func (api *RadioAPI) handleRadioItem(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	// Parse the path: /api/radios/{radioId}[/backups[/{backupId}[/download]]]
	path := strings.TrimPrefix(r.URL.Path, "/api/radios/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Radio ID required", http.StatusBadRequest)
		return
	}

	radioID := parts[0]

	// Handle backups sub-routes
	if len(parts) >= 2 && parts[1] == "backups" {
		if len(parts) == 2 {
			// /api/radios/{radioId}/backups
			switch r.Method {
			case http.MethodGet:
				api.handleListBackups(w, r, radioID, userID)
			case http.MethodPost:
				api.handleCreateBackup(w, r, radioID, userID)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		if len(parts) >= 3 {
			backupID := parts[2]

			// Check for download
			if len(parts) == 4 && parts[3] == "download" {
				// /api/radios/{radioId}/backups/{backupId}/download
				if r.Method == http.MethodGet {
					api.handleDownloadBackup(w, r, radioID, backupID, userID)
				} else {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				}
				return
			}

			// /api/radios/{radioId}/backups/{backupId}
			switch r.Method {
			case http.MethodGet:
				api.handleGetBackup(w, r, radioID, backupID, userID)
			case http.MethodDelete:
				api.handleDeleteBackup(w, r, radioID, backupID, userID)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}
	}

	// /api/radios/{radioId}
	switch r.Method {
	case http.MethodGet:
		api.handleGetRadio(w, r, radioID, userID)
	case http.MethodPut:
		api.handleUpdateRadio(w, r, radioID, userID)
	case http.MethodDelete:
		api.handleDeleteRadio(w, r, radioID, userID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetRadio retrieves a single radio
func (api *RadioAPI) handleGetRadio(w http.ResponseWriter, r *http.Request, radioID string, userID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	radio, err := api.radioSvc.GetRadio(ctx, radioID, userID)
	if err != nil {
		api.logger.Error("Failed to get radio", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if radio == nil {
		api.writeJSON(w, http.StatusNotFound, map[string]string{"error": "Radio not found"})
		return
	}

	api.writeJSON(w, http.StatusOK, radio)
}

// handleUpdateRadio updates a radio
func (api *RadioAPI) handleUpdateRadio(w http.ResponseWriter, r *http.Request, radioID string, userID string) {
	var params models.UpdateRadioParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	radio, err := api.radioSvc.UpdateRadio(ctx, radioID, userID, params)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		api.writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	api.writeJSON(w, http.StatusOK, radio)
}

// handleDeleteRadio deletes a radio
func (api *RadioAPI) handleDeleteRadio(w http.ResponseWriter, r *http.Request, radioID string, userID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	err := api.radioSvc.DeleteRadio(ctx, radioID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		api.writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleListBackups lists backups for a radio
func (api *RadioAPI) handleListBackups(w http.ResponseWriter, r *http.Request, radioID string, userID string) {
	query := r.URL.Query()

	params := models.RadioBackupListParams{}
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

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	response, err := api.radioSvc.ListBackups(ctx, radioID, userID, params)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		api.writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	api.writeJSON(w, http.StatusOK, response)
}

// handleCreateBackup creates a new backup
func (api *RadioAPI) handleCreateBackup(w http.ResponseWriter, r *http.Request, radioID string, userID string) {
	// Parse multipart form (max 100MB)
	if err := r.ParseMultipartForm(radiosvc.MaxBackupFileSize); err != nil {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Failed to parse form: " + err.Error()})
		return
	}

	// Get form fields
	backupName := r.FormValue("backupName")
	backupType := r.FormValue("backupType")
	if backupName == "" {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "backupName is required"})
		return
	}

	// Get the file
	file, header, err := r.FormFile("file")
	if err != nil {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file is required"})
		return
	}
	defer file.Close()

	params := models.CreateRadioBackupParams{
		BackupName: backupName,
		BackupType: models.BackupType(backupType),
		FileName:   header.Filename,
		FileSize:   header.Size,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	backup, err := api.radioSvc.CreateBackup(ctx, radioID, userID, params, file)
	if err != nil {
		status := http.StatusInternalServerError
		if _, ok := err.(*radiosvc.ServiceError); ok {
			status = http.StatusBadRequest
		} else if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		api.writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	api.writeJSON(w, http.StatusCreated, backup)
}

// handleGetBackup retrieves a backup
func (api *RadioAPI) handleGetBackup(w http.ResponseWriter, r *http.Request, radioID string, backupID string, userID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	backup, err := api.radioSvc.GetBackup(ctx, backupID, radioID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		api.writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	if backup == nil {
		api.writeJSON(w, http.StatusNotFound, map[string]string{"error": "Backup not found"})
		return
	}

	api.writeJSON(w, http.StatusOK, backup)
}

// handleDownloadBackup downloads a backup file
func (api *RadioAPI) handleDownloadBackup(w http.ResponseWriter, r *http.Request, radioID string, backupID string, userID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	file, backup, err := api.radioSvc.GetBackupFile(ctx, backupID, radioID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		api.writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	defer file.Close()

	// Set headers for file download
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+backup.FileName+"\"")
	w.Header().Set("Content-Length", strconv.FormatInt(backup.FileSize, 10))

	// Stream the file
	io.Copy(w, file)
}

// handleDeleteBackup deletes a backup
func (api *RadioAPI) handleDeleteBackup(w http.ResponseWriter, r *http.Request, radioID string, backupID string, userID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	err := api.radioSvc.DeleteBackup(ctx, backupID, radioID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		api.writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// writeJSON writes a JSON response
func (api *RadioAPI) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

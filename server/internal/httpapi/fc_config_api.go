package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/betaflight"
	"github.com/johnrirwin/flyingforge/internal/database"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

// FCConfigAPI handles HTTP API requests for flight controller configs
type FCConfigAPI struct {
	fcConfigStore  *database.FCConfigStore
	parser         *betaflight.Parser
	authMiddleware *auth.Middleware
	logger         *logging.Logger
}

// NewFCConfigAPI creates a new FC config API handler
func NewFCConfigAPI(fcConfigStore *database.FCConfigStore, authMiddleware *auth.Middleware, logger *logging.Logger) *FCConfigAPI {
	return &FCConfigAPI{
		fcConfigStore:  fcConfigStore,
		parser:         betaflight.NewParser(),
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

// RegisterRoutes registers FC config routes on the given mux
func (api *FCConfigAPI) RegisterRoutes(mux *http.ServeMux, corsMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("/api/fc-configs", corsMiddleware(api.authMiddleware.RequireAuth(api.handleFCConfigs)))
	mux.HandleFunc("/api/fc-configs/", corsMiddleware(api.authMiddleware.RequireAuth(api.handleFCConfigItem)))
	// Aircraft tuning routes use different paths to avoid conflicts
	mux.HandleFunc("/api/tuning/aircraft/", corsMiddleware(api.authMiddleware.RequireAuth(api.handleAircraftTuningRoutes)))
}

// handleAircraftTuningRoutes handles tuning routes under /api/tuning/aircraft/
func (api *FCConfigAPI) handleAircraftTuningRoutes(w http.ResponseWriter, r *http.Request) {
	// Parse path like /api/tuning/aircraft/{id} or /api/tuning/aircraft/{id}/snapshots
	path := strings.TrimPrefix(r.URL.Path, "/api/tuning/aircraft/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Aircraft ID required", http.StatusBadRequest)
		return
	}

	aircraftID := parts[0]

	if len(parts) >= 2 && parts[1] == "snapshots" {
		// /api/tuning/aircraft/{id}/snapshots
		switch r.Method {
		case http.MethodGet:
			api.listTuningSnapshots(w, r, aircraftID)
		case http.MethodPost:
			api.createTuningSnapshot(w, r, aircraftID)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// /api/tuning/aircraft/{id}
	switch r.Method {
	case http.MethodGet:
		api.getAircraftTuning(w, r, aircraftID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleFCConfigs handles list and create operations
func (api *FCConfigAPI) handleFCConfigs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		api.listFCConfigs(w, r)
	case http.MethodPost:
		api.createFCConfig(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleFCConfigItem handles single config operations
func (api *FCConfigAPI) handleFCConfigItem(w http.ResponseWriter, r *http.Request) {
	// Parse config ID from path like /api/fc-configs/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/fc-configs/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Config ID required", http.StatusBadRequest)
		return
	}

	configID := parts[0]

	switch r.Method {
	case http.MethodGet:
		api.getFCConfig(w, r, configID)
	case http.MethodPut:
		api.updateFCConfig(w, r, configID)
	case http.MethodDelete:
		api.deleteFCConfig(w, r, configID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// createFCConfig creates a new FC config from a CLI dump
func (api *FCConfigAPI) createFCConfig(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	var req models.SaveFCConfigParams
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if req.RawCLIDump == "" {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "CLI dump is required"})
		return
	}

	if req.Name == "" {
		req.Name = "Untitled Config"
	}

	// Parse the CLI dump
	result := api.parser.Parse(req.RawCLIDump)

	// Create the config
	config := &models.FlightControllerConfig{
		InventoryItemID: req.InventoryItemID,
		Name:            req.Name,
		Notes:           req.Notes,
		RawCLIDump:      req.RawCLIDump,
		FirmwareName:    result.FirmwareName,
		FirmwareVersion: result.FirmwareVersion,
		BoardTarget:     result.BoardTarget,
		BoardName:       result.BoardName,
		MCUType:         result.MCUType,
		ParseStatus:     result.ParseStatus,
		ParseWarnings:   result.ParseWarnings,
		ParsedTuning:    result.ParsedTuning,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	if err := api.fcConfigStore.SaveConfig(ctx, userID, config); err != nil {
		api.logger.Error("Failed to save FC config", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save config"})
		return
	}

	// If linked to an inventory item, check if there's an aircraft with that FC
	// and auto-create a tuning snapshot
	if config.InventoryItemID != "" {
		aircraft, err := api.fcConfigStore.GetAircraftByFC(ctx, userID, config.InventoryItemID)
		if err == nil && aircraft != nil {
			api.createTuningSnapshotFromConfig(ctx, aircraft.ID, config)
		}
	}

	api.writeJSON(w, http.StatusCreated, config)
}

// getFCConfig returns a single FC config
func (api *FCConfigAPI) getFCConfig(w http.ResponseWriter, r *http.Request, configID string) {
	userID := auth.GetUserID(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	config, err := api.fcConfigStore.GetConfig(ctx, configID, userID)
	if err != nil {
		api.logger.Error("Failed to get FC config", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get config"})
		return
	}

	if config == nil {
		api.writeJSON(w, http.StatusNotFound, map[string]string{"error": "Config not found"})
		return
	}

	api.writeJSON(w, http.StatusOK, config)
}

// listFCConfigs returns all FC configs for the user
func (api *FCConfigAPI) listFCConfigs(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	query := r.URL.Query()
	params := models.FCConfigListParams{
		InventoryItemID: query.Get("inventory_item_id"),
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

	response, err := api.fcConfigStore.ListConfigs(ctx, userID, params)
	if err != nil {
		api.logger.Error("Failed to list FC configs", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list configs"})
		return
	}

	api.writeJSON(w, http.StatusOK, response)
}

// updateFCConfig updates a config's metadata
func (api *FCConfigAPI) updateFCConfig(w http.ResponseWriter, r *http.Request, configID string) {
	userID := auth.GetUserID(r.Context())

	var req models.UpdateFCConfigParams
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	config, err := api.fcConfigStore.UpdateConfig(ctx, configID, userID, req)
	if err != nil {
		api.logger.Error("Failed to update FC config", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update config"})
		return
	}

	if config == nil {
		api.writeJSON(w, http.StatusNotFound, map[string]string{"error": "Config not found"})
		return
	}

	api.writeJSON(w, http.StatusOK, config)
}

// deleteFCConfig deletes a config
func (api *FCConfigAPI) deleteFCConfig(w http.ResponseWriter, r *http.Request, configID string) {
	userID := auth.GetUserID(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	if err := api.fcConfigStore.DeleteConfig(ctx, configID, userID); err != nil {
		if err.Error() == "config not found" {
			api.writeJSON(w, http.StatusNotFound, map[string]string{"error": "Config not found"})
			return
		}
		api.logger.Error("Failed to delete FC config", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to delete config"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// getAircraftTuning returns the latest tuning data for an aircraft
func (api *FCConfigAPI) getAircraftTuning(w http.ResponseWriter, r *http.Request, aircraftID string) {
	userID := auth.GetUserID(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	snapshot, err := api.fcConfigStore.GetLatestTuningSnapshot(ctx, aircraftID, userID)
	if err != nil {
		api.logger.Error("Failed to get tuning snapshot", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get tuning data"})
		return
	}

	if snapshot == nil {
		api.writeJSON(w, http.StatusOK, models.AircraftTuningResponse{
			AircraftID: aircraftID,
			HasTuning:  false,
		})
		return
	}

	// Parse the tuning data
	var tuningData *models.ParsedTuning
	if len(snapshot.TuningData) > 0 {
		tuningData = &models.ParsedTuning{}
		json.Unmarshal(snapshot.TuningData, tuningData)
	}

	api.writeJSON(w, http.StatusOK, models.AircraftTuningResponse{
		AircraftID:      aircraftID,
		HasTuning:       true,
		FirmwareName:    snapshot.FirmwareName,
		FirmwareVersion: snapshot.FirmwareVersion,
		BoardTarget:     snapshot.BoardTarget,
		BoardName:       snapshot.BoardName,
		Tuning:          tuningData,
		SnapshotID:      snapshot.ID,
		SnapshotDate:    snapshot.CreatedAt,
		ParseStatus:     snapshot.ParseStatus,
		ParseWarnings:   snapshot.ParseWarnings,
		HasDiffBackup:   snapshot.DiffBackup != "",
		DiffBackup:      snapshot.DiffBackup,
	})
}

// listTuningSnapshots returns all tuning snapshots for an aircraft
func (api *FCConfigAPI) listTuningSnapshots(w http.ResponseWriter, r *http.Request, aircraftID string) {
	userID := auth.GetUserID(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	snapshots, err := api.fcConfigStore.ListTuningSnapshots(ctx, aircraftID, userID)
	if err != nil {
		api.logger.Error("Failed to list tuning snapshots", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list snapshots"})
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]interface{}{
		"snapshots":   snapshots,
		"total_count": len(snapshots),
	})
}

// createTuningSnapshot creates a tuning snapshot from a CLI dump
func (api *FCConfigAPI) createTuningSnapshot(w http.ResponseWriter, r *http.Request, aircraftID string) {
	userID := auth.GetUserID(r.Context())

	var req struct {
		RawCLIDump string `json:"rawCliDump"`
		DiffBackup string `json:"diffBackup"`
		Notes      string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if req.RawCLIDump == "" {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "CLI dump is required"})
		return
	}

	// Parse the CLI dump
	result := api.parser.Parse(req.RawCLIDump)

	tuningData, err := json.Marshal(result.ParsedTuning)
	if err != nil {
		tuningData = []byte("{}")
	}

	snapshot := &models.AircraftTuningSnapshot{
		AircraftID:      aircraftID,
		FirmwareName:    result.FirmwareName,
		FirmwareVersion: result.FirmwareVersion,
		BoardTarget:     result.BoardTarget,
		BoardName:       result.BoardName,
		TuningData:      tuningData,
		ParseStatus:     result.ParseStatus,
		ParseWarnings:   result.ParseWarnings,
		DiffBackup:      req.DiffBackup,
		Notes:           req.Notes,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Verify user owns the aircraft (the store method does this via join)
	if err := api.fcConfigStore.SaveTuningSnapshot(ctx, snapshot); err != nil {
		api.logger.Error("Failed to save tuning snapshot",
			logging.WithField("error", err.Error()),
			logging.WithField("userID", userID),
		)
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to save tuning snapshot"})
		return
	}

	api.writeJSON(w, http.StatusCreated, snapshot)
}

// createTuningSnapshotFromConfig creates a tuning snapshot from an existing FC config
func (api *FCConfigAPI) createTuningSnapshotFromConfig(ctx context.Context, aircraftID string, config *models.FlightControllerConfig) error {
	tuningData, err := json.Marshal(config.ParsedTuning)
	if err != nil {
		tuningData = []byte("{}")
	}

	snapshot := &models.AircraftTuningSnapshot{
		AircraftID:               aircraftID,
		FlightControllerConfigID: config.ID,
		FirmwareName:             config.FirmwareName,
		FirmwareVersion:          config.FirmwareVersion,
		BoardTarget:              config.BoardTarget,
		BoardName:                config.BoardName,
		TuningData:               tuningData,
		ParseStatus:              config.ParseStatus,
		ParseWarnings:            config.ParseWarnings,
		Notes:                    "Auto-created from FC config: " + config.Name,
	}

	return api.fcConfigStore.SaveTuningSnapshot(ctx, snapshot)
}

// writeJSON writes a JSON response
func (api *FCConfigAPI) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

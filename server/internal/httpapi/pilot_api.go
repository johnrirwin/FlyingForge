package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/database"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

// PilotAPI handles pilot directory HTTP endpoints
type PilotAPI struct {
	userStore     *database.UserStore
	aircraftStore *database.AircraftStore
	authMiddleware *auth.Middleware
	logger        *logging.Logger
}

// NewPilotAPI creates a new pilot API handler
func NewPilotAPI(userStore *database.UserStore, aircraftStore *database.AircraftStore, authMiddleware *auth.Middleware, logger *logging.Logger) *PilotAPI {
	return &PilotAPI{
		userStore:     userStore,
		aircraftStore: aircraftStore,
		authMiddleware: authMiddleware,
		logger:        logger,
	}
}

// RegisterRoutes registers pilot routes on the given mux
func (api *PilotAPI) RegisterRoutes(mux *http.ServeMux, corsMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	// Search pilots - requires auth
	mux.HandleFunc("/api/pilots/search", corsMiddleware(api.authMiddleware.RequireAuth(api.handleSearch)))
	// Get pilot profile - requires auth
	mux.HandleFunc("/api/pilots/", corsMiddleware(api.authMiddleware.RequireAuth(api.handlePilotProfile)))
}

// handleSearch handles GET /api/pilots/search?q=searchterm
func (api *PilotAPI) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		api.writeJSON(w, http.StatusOK, map[string]interface{}{
			"pilots": []interface{}{},
			"total":  0,
		})
		return
	}

	// Require at least 2 characters for search
	if len(query) < 2 {
		api.writeError(w, http.StatusBadRequest, "query_too_short", "search query must be at least 2 characters")
		return
	}

	searchParams := models.PilotSearchParams{
		Query: query,
		Limit: 50,
	}
	pilots, err := api.userStore.SearchPilots(r.Context(), searchParams)
	if err != nil {
		api.logger.Error("Failed to search pilots", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to search pilots")
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]interface{}{
		"pilots": pilots,
		"total":  len(pilots),
	})
}

// handlePilotProfile handles GET /api/pilots/:id
func (api *PilotAPI) handlePilotProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract pilot ID from path: /api/pilots/{id}
	path := strings.TrimPrefix(r.URL.Path, "/api/pilots/")
	pilotID := strings.TrimSuffix(path, "/")
	
	if pilotID == "" || pilotID == "search" {
		// This shouldn't happen as search is handled separately, but just in case
		http.Error(w, "Pilot ID required", http.StatusBadRequest)
		return
	}

	// Get pilot user
	user, err := api.userStore.GetByID(r.Context(), pilotID)
	if err != nil {
		api.logger.Error("Failed to get pilot", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to get pilot")
		return
	}
	if user == nil {
		api.writeError(w, http.StatusNotFound, "not_found", "pilot not found")
		return
	}

	// Get pilot's aircraft (public view)
	aircraft, err := api.aircraftStore.ListByUserID(r.Context(), pilotID)
	if err != nil {
		api.logger.Error("Failed to get pilot aircraft", logging.WithField("error", err.Error()))
		// Don't fail the whole request, just return empty aircraft list
		aircraft = []*models.Aircraft{}
	}

	// Build public aircraft list
	publicAircraft := make([]models.AircraftPublic, 0, len(aircraft))
	for _, a := range aircraft {
		publicAircraft = append(publicAircraft, models.AircraftPublic{
			ID:          a.ID,
			Name:        a.Name,
			Nickname:    a.Nickname,
			Type:        a.Type,
			HasImage:    a.HasImage,
			Description: a.Description,
			CreatedAt:   a.CreatedAt,
		})
	}

	// Build pilot profile response
	profile := models.PilotProfile{
		ID:                 user.ID,
		CallSign:           user.CallSign,
		DisplayName:        user.DisplayName,
		GoogleName:         user.GoogleName,
		EffectiveAvatarURL: user.EffectiveAvatarURL(),
		CreatedAt:          user.CreatedAt,
		Aircraft:           publicAircraft,
	}

	api.writeJSON(w, http.StatusOK, profile)
}

func (api *PilotAPI) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (api *PilotAPI) writeError(w http.ResponseWriter, status int, code, message string) {
	api.writeJSON(w, status, map[string]string{
		"code":    code,
		"message": message,
	})
}

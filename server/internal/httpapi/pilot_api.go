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
	userStore      *database.UserStore
	aircraftStore  *database.AircraftStore
	fcConfigStore  *database.FCConfigStore
	authMiddleware *auth.Middleware
	logger         *logging.Logger
}

// NewPilotAPI creates a new pilot API handler
func NewPilotAPI(userStore *database.UserStore, aircraftStore *database.AircraftStore, fcConfigStore *database.FCConfigStore, authMiddleware *auth.Middleware, logger *logging.Logger) *PilotAPI {
	return &PilotAPI{
		userStore:      userStore,
		aircraftStore:  aircraftStore,
		fcConfigStore:  fcConfigStore,
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

// RegisterRoutes registers pilot routes on the given mux
func (api *PilotAPI) RegisterRoutes(mux *http.ServeMux, corsMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	// Search pilots - requires auth
	mux.HandleFunc("/api/pilots/search", corsMiddleware(api.authMiddleware.RequireAuth(api.handleSearch)))
	// Public aircraft image - requires auth but checks owner's visibility settings
	mux.HandleFunc("/api/pilots/aircraft/", corsMiddleware(api.authMiddleware.RequireAuth(api.handleAircraftImage)))
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

	ctx := r.Context()
	currentUserID := auth.GetUserID(ctx)

	// Get pilot user
	user, err := api.userStore.GetByID(ctx, pilotID)
	if err != nil {
		api.logger.Error("Failed to get pilot", logging.WithField("error", err.Error()))
		api.writeError(w, http.StatusInternalServerError, "internal_error", "failed to get pilot")
		return
	}
	if user == nil {
		api.writeError(w, http.StatusNotFound, "not_found", "pilot not found")
		return
	}

	// Check profile visibility (owner always sees own profile)
	isOwner := currentUserID == pilotID

	// Require callsign to be set for social visibility (privacy protection)
	// Users without callsigns are not discoverable in social features
	if !isOwner && (user.CallSign == "") {
		api.writeError(w, http.StatusNotFound, "not_found", "pilot not found")
		return
	}

	if !isOwner && user.SocialSettings.ProfileVisibility == models.ProfileVisibilityPrivate {
		api.writeError(w, http.StatusNotFound, "private_profile", "this profile is private")
		return
	}

	// Check if current user is following this pilot
	isFollowing := false
	if !isOwner {
		isFollowing, _ = api.userStore.IsFollowing(ctx, currentUserID, pilotID)
	}

	// Get follower/following counts
	followerCount, _ := api.userStore.GetFollowerCount(ctx, pilotID)
	followingCount, _ := api.userStore.GetFollowingCount(ctx, pilotID)

	// Get pilot's aircraft based on visibility settings
	var publicAircraft []models.AircraftPublic
	if isOwner || user.SocialSettings.ShowAircraft {
		aircraft, err := api.aircraftStore.ListByUserID(ctx, pilotID)
		if err != nil {
			api.logger.Error("Failed to get pilot aircraft", logging.WithField("error", err.Error()))
			// Don't fail the whole request, just return empty aircraft list
			aircraft = []*models.Aircraft{}
		}

		// Build public aircraft list with sanitized receiver data and components
		publicAircraft = make([]models.AircraftPublic, 0, len(aircraft))
		for _, a := range aircraft {
			aircraftPublic := models.AircraftPublic{
				ID:          a.ID,
				Name:        a.Name,
				Nickname:    a.Nickname,
				Type:        a.Type,
				HasImage:    a.HasImage,
				Description: a.Description,
				CreatedAt:   a.CreatedAt,
			}

			// Get components (sanitized - only public info)
			components, err := api.aircraftStore.GetComponents(ctx, a.ID)
			if err == nil && len(components) > 0 {
				publicComponents := make([]models.AircraftComponentPublic, 0, len(components))
				for _, c := range components {
					pc := models.AircraftComponentPublic{
						Category: c.Category,
					}
					// Include inventory item info (sanitized - no purchase details)
					if c.InventoryItem != nil {
						pc.Name = c.InventoryItem.Name
						pc.Manufacturer = c.InventoryItem.Manufacturer
						pc.ImageURL = c.InventoryItem.ImageURL
					}
					publicComponents = append(publicComponents, pc)
				}
				aircraftPublic.Components = publicComponents
			}

			// Get receiver settings - sanitize for non-owners (hide sensitive data)
			receiverSettings, err := api.aircraftStore.GetReceiverSettings(ctx, a.ID)
			if err == nil && receiverSettings != nil {
				// Always sanitize - removes bind phrase, model match, uid etc.
				sanitized := models.SanitizeReceiverSettings(receiverSettings)
				aircraftPublic.ReceiverSettings = sanitized
				// Log that sanitization was performed for observability
				api.logger.Debug("Sanitized receiver settings for aircraft", logging.WithField("aircraft_id", a.ID))
			}

			// Get tuning data if available
			if api.fcConfigStore != nil {
				tuningSnapshot, err := api.fcConfigStore.GetLatestTuningSnapshotPublic(ctx, a.ID)
				if err == nil && tuningSnapshot != nil {
					// Parse the tuning data
					var tuningData *models.ParsedTuning
					if len(tuningSnapshot.TuningData) > 0 {
						tuningData = &models.ParsedTuning{}
						json.Unmarshal(tuningSnapshot.TuningData, tuningData)
					}
					aircraftPublic.Tuning = &models.AircraftTuningPublic{
						FirmwareName:    tuningSnapshot.FirmwareName,
						FirmwareVersion: tuningSnapshot.FirmwareVersion,
						BoardTarget:     tuningSnapshot.BoardTarget,
						BoardName:       tuningSnapshot.BoardName,
						ParsedTuning:    tuningData,
						SnapshotDate:    tuningSnapshot.CreatedAt,
					}
				}
			}

			publicAircraft = append(publicAircraft, aircraftPublic)
		}
	} else {
		publicAircraft = []models.AircraftPublic{}
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
		IsFollowing:        isFollowing,
		FollowerCount:      followerCount,
		FollowingCount:     followingCount,
	}

	api.writeJSON(w, http.StatusOK, profile)
}

// handleAircraftImage handles GET /api/pilots/aircraft/:id/image
// Serves aircraft images for public profiles (respects visibility settings)
func (api *PilotAPI) handleAircraftImage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract aircraft ID from path: /api/pilots/aircraft/{id}/image
	path := strings.TrimPrefix(r.URL.Path, "/api/pilots/aircraft/")
	aircraftID := strings.TrimSuffix(path, "/image")
	aircraftID = strings.TrimSuffix(aircraftID, "/")

	if aircraftID == "" {
		http.Error(w, "Aircraft ID required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Get public image (checks owner's visibility settings)
	imageData, imageType, err := api.aircraftStore.GetPublicImage(ctx, aircraftID)
	if err != nil {
		api.logger.Error("Failed to get aircraft image", logging.WithField("error", err.Error()))
		http.Error(w, "Failed to get image", http.StatusInternalServerError)
		return
	}

	if imageData == nil {
		http.Error(w, "Image not found or not public", http.StatusNotFound)
		return
	}

	// Set cache headers
	w.Header().Set("Content-Type", imageType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	w.Write(imageData)
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

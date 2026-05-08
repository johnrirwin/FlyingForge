package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/johnrirwin/flyingforge/internal/announcements"
	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

type AnnouncementAPI struct {
	service        *announcements.Service
	authMiddleware *auth.Middleware
	logger         *logging.Logger
}

func NewAnnouncementAPI(service *announcements.Service, authMiddleware *auth.Middleware, logger *logging.Logger) *AnnouncementAPI {
	return &AnnouncementAPI{
		service:        service,
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

func (api *AnnouncementAPI) RegisterRoutes(mux *http.ServeMux, corsMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	handler := api.handleActiveAnnouncements
	if api.authMiddleware != nil {
		handler = api.authMiddleware.OptionalAuth(handler)
	}
	mux.HandleFunc("/api/announcements/active", corsMiddleware(handler))
}

func (api *AnnouncementAPI) handleActiveAnnouncements(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	placement := models.NormalizeAnnouncementPlacement(models.AnnouncementPlacement(strings.TrimSpace(r.URL.Query().Get("placement"))))
	if !models.IsValidAnnouncementPlacement(placement) {
		api.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid placement"})
		return
	}

	isAuthenticated := strings.TrimSpace(auth.GetUserID(r.Context())) != ""
	items, err := api.service.ListActive(r.Context(), placement, isAuthenticated)
	if err != nil {
		api.logger.Error("Failed to list active announcements", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load announcements"})
		return
	}

	api.writeJSON(w, http.StatusOK, models.AnnouncementListResponse{
		Announcements: items,
		TotalCount:    len(items),
	})
}

func (api *AnnouncementAPI) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

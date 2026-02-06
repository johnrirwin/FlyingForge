package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/database"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

// AdminAPI handles admin-only endpoints
type AdminAPI struct {
	catalogStore   *database.GearCatalogStore
	userStore      *database.UserStore
	authMiddleware *auth.Middleware
	logger         *logging.Logger
}

// NewAdminAPI creates a new admin API handler
func NewAdminAPI(catalogStore *database.GearCatalogStore, userStore *database.UserStore, authMiddleware *auth.Middleware, logger *logging.Logger) *AdminAPI {
	return &AdminAPI{
		catalogStore:   catalogStore,
		userStore:      userStore,
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

	// All admin routes require authentication AND admin role
	mux.HandleFunc("/api/admin/gear", corsMiddleware(api.authMiddleware.RequireAuth(api.requireAdmin(api.handleAdminGear))))
	mux.HandleFunc("/api/admin/gear/", corsMiddleware(api.authMiddleware.RequireAuth(api.requireAdmin(api.handleAdminGearByID))))
}

// requireAdmin is middleware that checks if the authenticated user is an admin
func (api *AdminAPI) requireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := auth.GetUserID(r.Context())
		if userID == "" {
			http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		user, err := api.userStore.GetByID(ctx, userID)
		if err != nil || user == nil {
			api.logger.Error("Failed to get user for admin check", logging.WithField("error", err))
			http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
			return
		}

		if !user.IsAdmin {
			api.logger.Warn("Non-admin user attempted admin access", logging.WithField("userId", userID))
			http.Error(w, `{"error":"admin access required"}`, http.StatusForbidden)
			return
		}

		next(w, r)
	}
}

// handleAdminGear handles GET /api/admin/gear (list gear for moderation)
func (api *AdminAPI) handleAdminGear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()

	params := models.AdminGearSearchParams{
		Query:       query.Get("query"),
		GearType:    models.GearType(query.Get("gearType")),
		Brand:       query.Get("brand"),
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

// handleAdminGearByID handles GET/PUT /api/admin/gear/{id}
func (api *AdminAPI) handleAdminGearByID(w http.ResponseWriter, r *http.Request) {
	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/gear/")
	id := strings.TrimSuffix(path, "/")
	if id == "" {
		http.Error(w, "gear ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		api.handleGetGear(w, r, id)
	case http.MethodPut:
		api.handleUpdateGear(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	// Validate ImageURL if provided
	if params.ImageURL != nil && *params.ImageURL != "" {
		if err := validateImageURL(*params.ImageURL); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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

// writeJSON writes a JSON response
func (api *AdminAPI) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
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

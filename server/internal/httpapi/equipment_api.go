package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/equipment"
	"github.com/johnrirwin/flyingforge/internal/inventory"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

// EquipmentAPI handles HTTP API requests for equipment and inventory
type EquipmentAPI struct {
	equipmentSvc   *equipment.Service
	inventorySvc   inventory.InventoryManager
	authMiddleware *auth.Middleware
	logger         *logging.Logger
}

// NewEquipmentAPI creates a new equipment API handler
func NewEquipmentAPI(equipmentSvc *equipment.Service, inventorySvc inventory.InventoryManager, authMiddleware *auth.Middleware, logger *logging.Logger) *EquipmentAPI {
	return &EquipmentAPI{
		equipmentSvc:   equipmentSvc,
		inventorySvc:   inventorySvc,
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

// RegisterRoutes registers equipment and inventory routes on the given mux
func (api *EquipmentAPI) RegisterRoutes(mux *http.ServeMux, corsMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	// Equipment routes (public)
	mux.HandleFunc("/api/equipment/search", corsMiddleware(api.handleSearchEquipment))
	mux.HandleFunc("/api/equipment/category/", corsMiddleware(api.handleGetByCategory))
	mux.HandleFunc("/api/equipment/sellers", corsMiddleware(api.handleGetSellers))
	mux.HandleFunc("/api/equipment/sync", corsMiddleware(api.handleSyncProducts))

	// Inventory routes (require authentication)
	mux.HandleFunc("/api/inventory", corsMiddleware(api.authMiddleware.RequireAuth(api.handleInventory)))
	mux.HandleFunc("/api/inventory/summary", corsMiddleware(api.authMiddleware.RequireAuth(api.handleInventorySummary)))
	mux.HandleFunc("/api/inventory/", corsMiddleware(api.authMiddleware.RequireAuth(api.handleInventoryItem)))
}

// Equipment handlers

func (api *EquipmentAPI) handleSearchEquipment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()

	params := models.EquipmentSearchParams{
		Query:       query.Get("q"),
		Category:    models.EquipmentCategory(query.Get("category")),
		Seller:      query.Get("seller"),
		InStockOnly: query.Get("inStock") == "true",
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

	if minPrice := query.Get("minPrice"); minPrice != "" {
		if p, err := strconv.ParseFloat(minPrice, 64); err == nil {
			params.MinPrice = &p
		}
	}

	if maxPrice := query.Get("maxPrice"); maxPrice != "" {
		if p, err := strconv.ParseFloat(maxPrice, 64); err == nil {
			params.MaxPrice = &p
		}
	}

	if sort := query.Get("sort"); sort != "" {
		params.Sort = sort
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	response, err := api.equipmentSvc.Search(ctx, params)
	if err != nil {
		api.logger.Error("Equipment search failed", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	api.writeJSON(w, http.StatusOK, response)
}

func (api *EquipmentAPI) handleGetByCategory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract category from path: /api/equipment/category/{category}
	path := r.URL.Path
	category := path[len("/api/equipment/category/"):]
	if category == "" {
		http.Error(w, "Category required", http.StatusBadRequest)
		return
	}

	query := r.URL.Query()

	limit := 20
	if l := query.Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if o := query.Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	response, err := api.equipmentSvc.GetByCategory(ctx, models.EquipmentCategory(category), limit, offset)
	if err != nil {
		api.logger.Error("Category fetch failed", logging.WithFields(map[string]interface{}{
			"category": category,
			"error":    err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	api.writeJSON(w, http.StatusOK, response)
}

func (api *EquipmentAPI) handleGetSellers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := api.equipmentSvc.GetSellers()
	api.writeJSON(w, http.StatusOK, response)
}

func (api *EquipmentAPI) handleSyncProducts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	err := api.equipmentSvc.SyncProducts(ctx)
	if err != nil {
		api.logger.Error("Sync failed", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	api.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Product sync triggered",
	})
}

// Inventory handlers

func (api *EquipmentAPI) handleInventory(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		api.listInventory(w, r)
	case http.MethodPost:
		api.addInventoryItem(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (api *EquipmentAPI) listInventory(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	query := r.URL.Query()

	params := models.InventoryFilterParams{
		Category:  models.EquipmentCategory(query.Get("category")),
		Condition: models.ItemCondition(query.Get("condition")),
		BuildID:   query.Get("buildId"),
		Query:     query.Get("q"),
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

	response, err := api.inventorySvc.GetInventory(ctx, userID, params)
	if err != nil {
		api.logger.Error("Inventory list failed", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	api.writeJSON(w, http.StatusOK, response)
}

func (api *EquipmentAPI) addInventoryItem(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())

	var params models.AddInventoryParams

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	item, err := api.inventorySvc.AddItem(ctx, userID, params)
	if err != nil {
		api.logger.Error("Add inventory item failed", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	api.writeJSON(w, http.StatusCreated, item)
}

func (api *EquipmentAPI) handleInventoryItem(w http.ResponseWriter, r *http.Request) {
	// Extract item ID from path: /api/inventory/{id}
	path := r.URL.Path
	id := path[len("/api/inventory/"):]
	if id == "" || id == "summary" {
		http.Error(w, "Item ID required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		api.getInventoryItem(w, r, id)
	case http.MethodPut, http.MethodPatch:
		api.updateInventoryItem(w, r, id)
	case http.MethodDelete:
		api.deleteInventoryItem(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (api *EquipmentAPI) getInventoryItem(w http.ResponseWriter, r *http.Request, id string) {
	userID := auth.GetUserID(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	item, err := api.inventorySvc.GetItem(ctx, id, userID)
	if err != nil {
		api.logger.Error("Get inventory item failed", logging.WithFields(map[string]interface{}{
			"id":    id,
			"error": err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	if item == nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	api.writeJSON(w, http.StatusOK, item)
}

func (api *EquipmentAPI) updateInventoryItem(w http.ResponseWriter, r *http.Request, id string) {
	userID := auth.GetUserID(r.Context())

	var params models.UpdateInventoryParams

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	params.ID = id

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	item, err := api.inventorySvc.UpdateItem(ctx, userID, params)
	if err != nil {
		api.logger.Error("Update inventory item failed", logging.WithFields(map[string]interface{}{
			"id":    id,
			"error": err.Error(),
		}))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	api.writeJSON(w, http.StatusOK, item)
}

func (api *EquipmentAPI) deleteInventoryItem(w http.ResponseWriter, r *http.Request, id string) {
	userID := auth.GetUserID(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	if err := api.inventorySvc.RemoveItem(ctx, id, userID); err != nil {
		api.logger.Error("Delete inventory item failed", logging.WithFields(map[string]interface{}{
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

func (api *EquipmentAPI) handleInventorySummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := auth.GetUserID(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	summary, err := api.inventorySvc.GetSummary(ctx, userID)
	if err != nil {
		api.logger.Error("Get inventory summary failed", logging.WithField("error", err.Error()))
		api.writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	api.writeJSON(w, http.StatusOK, summary)
}

func (api *EquipmentAPI) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

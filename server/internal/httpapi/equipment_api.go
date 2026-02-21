package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/equipment"
	"github.com/johnrirwin/flyingforge/internal/inventory"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

const (
	defaultBatteryCells       = 4
	defaultBatteryCapacityMah = 1500
)

var (
	batteryCellsPattern     = regexp.MustCompile(`(?i)\b([1-8])\s*s(?:\d*p)?\b`)
	batteryCapacityPattern  = regexp.MustCompile(`(?i)\b(\d{3,5})\s*m?ah\b`)
	batteryAhPattern        = regexp.MustCompile(`(?i)\b(\d+(?:\.\d+)?)\s*ah\b`)
	batteryCRatingPattern   = regexp.MustCompile(`(?i)\b(\d{1,3})\s*c\b`)
	batteryWeightPattern    = regexp.MustCompile(`(?i)\b(\d{2,4})\s*g\b`)
	positiveIntPattern      = regexp.MustCompile(`\d+`)
	batteryConnectorPattern = regexp.MustCompile(
		`(?i)\b(xt30|xt60|xt90|bt2\.0|bt3\.0|a30|a60|ec3|ec5|ph2\.0|t[- ]plug|deans)\b`,
	)
)

type batteryCreator interface {
	Create(ctx context.Context, userID string, params models.CreateBatteryParams) (*models.Battery, error)
}

// EquipmentAPI handles HTTP API requests for equipment and inventory
type EquipmentAPI struct {
	equipmentSvc   *equipment.Service
	inventorySvc   inventory.InventoryManager
	batterySvc     batteryCreator
	authMiddleware *auth.Middleware
	logger         *logging.Logger
}

// NewEquipmentAPI creates a new equipment API handler
func NewEquipmentAPI(equipmentSvc *equipment.Service, inventorySvc inventory.InventoryManager, batterySvc batteryCreator, authMiddleware *auth.Middleware, logger *logging.Logger) *EquipmentAPI {
	return &EquipmentAPI{
		equipmentSvc:   equipmentSvc,
		inventorySvc:   inventorySvc,
		batterySvc:     batterySvc,
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

// RegisterRoutes registers equipment and inventory routes on the given mux
func (api *EquipmentAPI) RegisterRoutes(mux *http.ServeMux, corsMiddleware func(http.HandlerFunc) http.HandlerFunc) {
	if api.authMiddleware == nil {
		api.logger.Error("Equipment API routes not registered: authMiddleware is nil")
		return
	}

	// Equipment routes (require authentication)
	mux.HandleFunc("/api/equipment/search", corsMiddleware(api.authMiddleware.RequireAuth(api.handleSearchEquipment)))
	mux.HandleFunc("/api/equipment/category/", corsMiddleware(api.authMiddleware.RequireAuth(api.handleGetByCategory)))
	mux.HandleFunc("/api/equipment/sellers", corsMiddleware(api.authMiddleware.RequireAuth(api.handleGetSellers)))
	mux.HandleFunc("/api/equipment/sync", corsMiddleware(api.authMiddleware.RequireAuth(api.handleSyncProducts)))

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
		"message": "No external seller sync configured",
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
		Category: models.EquipmentCategory(query.Get("category")),
		BuildID:  query.Get("buildId"),
		Query:    query.Get("q"),
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

	api.createTrackedBatteries(ctx, userID, params)

	api.writeJSON(w, http.StatusCreated, item)
}

func (api *EquipmentAPI) createTrackedBatteries(ctx context.Context, userID string, params models.AddInventoryParams) int {
	if api.batterySvc == nil || params.Category != models.CategoryBatteries {
		return 0
	}

	quantity := params.Quantity
	if quantity <= 0 {
		quantity = 1
	}

	createParams := inferBatteryCreateParams(params)
	created := 0
	for i := 0; i < quantity; i++ {
		if _, err := api.batterySvc.Create(ctx, userID, createParams); err != nil {
			api.logger.Error("Failed to create battery from inventory item", logging.WithFields(map[string]interface{}{
				"user_id":    userID,
				"item_name":  params.Name,
				"item_index": i + 1,
				"quantity":   quantity,
				"error":      err.Error(),
			}))
			return created
		}
		created++
	}

	return created
}

func inferBatteryCreateParams(params models.AddInventoryParams) models.CreateBatteryParams {
	specs := parseInventorySpecs(params.Specs)
	name := strings.TrimSpace(params.Name)

	chemistry := inferBatteryChemistry(name, specs)
	cells := inferBatteryCells(name, specs)
	capacityMah := inferBatteryCapacityMah(name, specs)
	cRating := inferBatteryCRating(name, specs)
	connector := inferBatteryConnector(name, specs)
	weightGrams := inferBatteryWeightGrams(name, specs)

	createParams := models.CreateBatteryParams{
		Name:        name,
		Chemistry:   chemistry,
		Cells:       cells,
		CapacityMah: capacityMah,
		Connector:   connector,
		Brand:       strings.TrimSpace(params.Manufacturer),
		Notes:       strings.TrimSpace(params.Notes),
	}

	if cRating > 0 {
		createParams.CRating = &cRating
	}
	if weightGrams > 0 {
		createParams.WeightGrams = &weightGrams
	}

	return createParams
}

func inferBatteryChemistry(name string, specs map[string]string) models.BatteryChemistry {
	specText := firstSpecValue(specs, "chemistry", "battery_chemistry", "chem")
	if chemistry, ok := parseBatteryChemistry(specText); ok {
		return chemistry
	}
	if chemistry, ok := parseBatteryChemistry(name); ok {
		return chemistry
	}
	return models.ChemistryLIPO
}

func inferBatteryCells(name string, specs map[string]string) int {
	specText := firstSpecValue(specs, "cells", "cell_count", "cellcount", "series")
	if cells, ok := parseBatteryCells(specText, true); ok {
		return cells
	}
	if cells, ok := parseBatteryCells(name, false); ok {
		return cells
	}
	return defaultBatteryCells
}

func inferBatteryCapacityMah(name string, specs map[string]string) int {
	specText := firstSpecValue(specs, "capacity_mah", "capacitymah", "capacity", "mah")
	if capacity, ok := parseBatteryCapacityMah(specText, true); ok {
		return capacity
	}
	if capacity, ok := parseBatteryCapacityMah(name, false); ok {
		return capacity
	}
	return defaultBatteryCapacityMah
}

func inferBatteryCRating(name string, specs map[string]string) int {
	specText := firstSpecValue(specs, "c_rating", "crating", "discharge_rating", "dischargerating")
	if cRating, ok := parseBatteryCRating(specText, true); ok {
		return cRating
	}
	if cRating, ok := parseBatteryCRating(name, false); ok {
		return cRating
	}
	return 0
}

func inferBatteryConnector(name string, specs map[string]string) string {
	specText := firstSpecValue(specs, "connector", "plug")
	if connector, ok := parseBatteryConnector(specText); ok {
		return connector
	}
	if connector, ok := parseBatteryConnector(name); ok {
		return connector
	}
	return ""
}

func inferBatteryWeightGrams(name string, specs map[string]string) int {
	specText := firstSpecValue(specs, "weight_grams", "weightgrams", "weight_g", "weight")
	if weight, ok := parseBatteryWeightGrams(specText, true); ok {
		return weight
	}
	if weight, ok := parseBatteryWeightGrams(name, false); ok {
		return weight
	}
	return 0
}

func parseInventorySpecs(raw json.RawMessage) map[string]string {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil
	}

	normalized := make(map[string]string, len(decoded))
	for key, value := range decoded {
		text := strings.TrimSpace(fmt.Sprintf("%v", value))
		if text == "" || text == "<nil>" {
			continue
		}
		normalized[normalizeSpecKey(key)] = text
	}

	return normalized
}

func normalizeSpecKey(key string) string {
	replacer := strings.NewReplacer("_", "", "-", "", " ", "")
	return replacer.Replace(strings.ToLower(strings.TrimSpace(key)))
}

func firstSpecValue(specs map[string]string, keys ...string) string {
	for _, key := range keys {
		if value, ok := specs[normalizeSpecKey(key)]; ok {
			value = strings.TrimSpace(value)
			if value != "" && value != "<nil>" {
				return value
			}
		}
	}
	return ""
}

func parseBatteryChemistry(text string) (models.BatteryChemistry, bool) {
	normalized := strings.ToUpper(strings.TrimSpace(text))
	if normalized == "" {
		return "", false
	}

	normalized = strings.NewReplacer("-", "", "_", "", " ", "").Replace(normalized)

	switch {
	case strings.Contains(normalized, "LIION") || strings.Contains(normalized, "LION"):
		return models.ChemistryLIION, true
	case strings.Contains(normalized, "LIPOHV") || strings.Contains(normalized, "LIHV"):
		return models.ChemistryLIPOHV, true
	case strings.Contains(normalized, "LIPO"):
		return models.ChemistryLIPO, true
	default:
		return "", false
	}
}

func parseBatteryCells(text string, allowBareNumber bool) (int, bool) {
	match := batteryCellsPattern.FindStringSubmatch(text)
	if len(match) < 2 {
		if allowBareNumber {
			value, ok := parsePositiveInt(text)
			if ok && value >= 1 && value <= 8 {
				return value, true
			}
		}
		return 0, false
	}

	value, err := strconv.Atoi(match[1])
	if err != nil || value < 1 || value > 8 {
		return 0, false
	}
	return value, true
}

func parseBatteryCapacityMah(text string, allowBareNumber bool) (int, bool) {
	match := batteryCapacityPattern.FindStringSubmatch(text)
	if len(match) >= 2 {
		value, err := strconv.Atoi(match[1])
		if err == nil && value > 0 {
			return value, true
		}
	}

	ahMatch := batteryAhPattern.FindStringSubmatch(text)
	if len(ahMatch) >= 2 {
		value, err := strconv.ParseFloat(ahMatch[1], 64)
		if err == nil && value > 0 {
			return int(math.Round(value * 1000)), true
		}
	}

	if allowBareNumber {
		value, ok := parsePositiveInt(text)
		if ok && value > 0 {
			return value, true
		}
	}
	return 0, false
}

func parseBatteryCRating(text string, allowBareNumber bool) (int, bool) {
	match := batteryCRatingPattern.FindStringSubmatch(text)
	if len(match) >= 2 {
		value, err := strconv.Atoi(match[1])
		if err == nil && value > 0 {
			return value, true
		}
	}
	if allowBareNumber {
		value, ok := parsePositiveInt(text)
		if ok && value > 0 {
			return value, true
		}
	}
	return 0, false
}

func parseBatteryConnector(text string) (string, bool) {
	match := batteryConnectorPattern.FindStringSubmatch(text)
	if len(match) < 2 {
		return "", false
	}
	connector := strings.ToUpper(strings.TrimSpace(match[1]))
	connector = strings.ReplaceAll(connector, " ", "-")
	return connector, connector != ""
}

func parseBatteryWeightGrams(text string, allowBareNumber bool) (int, bool) {
	match := batteryWeightPattern.FindStringSubmatch(text)
	if len(match) >= 2 {
		value, err := strconv.Atoi(match[1])
		if err == nil && value > 0 {
			return value, true
		}
	}
	if allowBareNumber {
		value, ok := parsePositiveInt(text)
		if ok && value > 0 {
			return value, true
		}
	}
	return 0, false
}

func parsePositiveInt(text string) (int, bool) {
	digits := positiveIntPattern.FindString(text)
	if digits == "" {
		return 0, false
	}
	value, err := strconv.Atoi(digits)
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
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

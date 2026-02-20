package mcp

import (
	"context"
	"encoding/json"

	"github.com/johnrirwin/flyingforge/internal/equipment"
	"github.com/johnrirwin/flyingforge/internal/inventory"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

// EquipmentHandler handles MCP tool calls for equipment and inventory
type EquipmentHandler struct {
	equipmentSvc *equipment.Service
	inventorySvc inventory.InventoryManager
	logger       *logging.Logger
}

// NewEquipmentHandler creates a new equipment handler
func NewEquipmentHandler(equipmentSvc *equipment.Service, inventorySvc inventory.InventoryManager, logger *logging.Logger) *EquipmentHandler {
	return &EquipmentHandler{
		equipmentSvc: equipmentSvc,
		inventorySvc: inventorySvc,
		logger:       logger,
	}
}

// GetTools returns the tool definitions for equipment and inventory
func (h *EquipmentHandler) GetTools() []ToolDefinition {
	return []ToolDefinition{
		// Equipment tools
		{
			Name:        "search_equipment",
			Description: "Search for drone equipment across all supported sellers. Returns matching products with prices, availability, and specifications.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {
						"type": "string",
						"description": "Search query (e.g., 'nazgul frame', 'emax motor')"
					},
					"category": {
						"type": "string",
						"enum": ["frames", "vtx", "flight_controllers", "esc", "aio", "stacks", "motors", "propellers", "receivers", "batteries", "cameras", "antennas", "accessories"],
						"description": "Filter by equipment category"
					},
					"seller": {
						"type": "string",
						"description": "Filter by seller ID (e.g., 'racedayquads', 'getfpv')"
					},
					"minPrice": {
						"type": "number",
						"description": "Minimum price filter"
					},
					"maxPrice": {
						"type": "number",
						"description": "Maximum price filter"
					},
					"inStockOnly": {
						"type": "boolean",
						"description": "Only show items currently in stock"
					},
					"limit": {
						"type": "integer",
						"description": "Maximum number of results (default: 20)"
					}
				},
				"required": ["query"]
			}`),
		},
		{
			Name:        "get_equipment_by_category",
			Description: "Browse drone equipment by category. Returns products from all supported sellers sorted by price.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"category": {
						"type": "string",
						"enum": ["frames", "vtx", "flight_controllers", "esc", "aio", "stacks", "motors", "propellers", "receivers", "batteries", "cameras", "antennas", "accessories"],
						"description": "Equipment category to browse"
					},
					"limit": {
						"type": "integer",
						"description": "Maximum number of results (default: 20)"
					},
					"offset": {
						"type": "integer",
						"description": "Offset for pagination"
					}
				},
				"required": ["category"]
			}`),
		},
		{
			Name:        "get_sellers",
			Description: "Get a list of all supported equipment sellers/retailers.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
		},
		{
			Name:        "sync_seller_products",
			Description: "Trigger a sync of products from a specific seller and category. Returns the number of products synced.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"seller": {
						"type": "string",
						"description": "Seller ID to sync (e.g., 'racedayquads', 'getfpv')"
					},
					"category": {
						"type": "string",
						"enum": ["frames", "vtx", "flight_controllers", "esc", "aio", "stacks", "motors", "propellers", "receivers", "batteries", "cameras", "antennas", "accessories"],
						"description": "Category to sync"
					}
				},
				"required": ["seller", "category"]
			}`),
		},
		// Inventory tools
		{
			Name:        "add_inventory_item",
			Description: "Add an item to your personal drone equipment inventory. Can add manually or from a searched equipment item.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"name": {
						"type": "string",
						"description": "Name of the item"
					},
					"category": {
						"type": "string",
						"enum": ["frames", "vtx", "flight_controllers", "esc", "aio", "stacks", "motors", "propellers", "receivers", "batteries", "cameras", "antennas", "accessories"],
						"description": "Equipment category"
					},
					"manufacturer": {
						"type": "string",
						"description": "Manufacturer/brand name"
					},
					"quantity": {
						"type": "integer",
						"description": "Quantity owned (default: 1)"
					},
					"notes": {
						"type": "string",
						"description": "Personal notes about this item"
					},
					"buildId": {
						"type": "string",
						"description": "ID of the drone build this item belongs to"
					},
					"purchasePrice": {
						"type": "number",
						"description": "Purchase price"
					},
					"purchaseDate": {
						"type": "string",
						"description": "Purchase date (YYYY-MM-DD format)"
					},
					"purchaseSeller": {
						"type": "string",
						"description": "Where the item was purchased"
					},
					"sourceEquipmentId": {
						"type": "string",
						"description": "ID of the equipment item if adding from search results"
					}
				},
				"required": ["name", "category"]
			}`),
		},
		{
			Name:        "get_inventory",
			Description: "Get your personal drone equipment inventory with optional filtering.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"category": {
						"type": "string",
						"enum": ["frames", "vtx", "flight_controllers", "esc", "aio", "stacks", "motors", "propellers", "receivers", "batteries", "cameras", "antennas", "accessories"],
						"description": "Filter by category"
					},
					"buildId": {
						"type": "string",
						"description": "Filter by build ID"
					},
					"query": {
						"type": "string",
						"description": "Search query to filter items"
					},
					"limit": {
						"type": "integer",
						"description": "Maximum number of results (default: 50)"
					},
					"offset": {
						"type": "integer",
						"description": "Offset for pagination"
					}
				}
			}`),
		},
		{
			Name:        "update_inventory_item",
			Description: "Update an existing item in your inventory (quantity, notes, etc.).",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"id": {
						"type": "string",
						"description": "ID of the inventory item to update"
					},
					"quantity": {
						"type": "integer",
						"description": "New quantity"
					},
					"notes": {
						"type": "string",
						"description": "Updated notes"
					},
					"buildId": {
						"type": "string",
						"description": "Assign to a build (or empty string to unassign)"
					}
				},
				"required": ["id"]
			}`),
		},
		{
			Name:        "remove_inventory_item",
			Description: "Remove an item from your personal inventory.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"id": {
						"type": "string",
						"description": "ID of the inventory item to remove"
					}
				},
				"required": ["id"]
			}`),
		},
	}
}

// HandleToolCall handles MCP tool calls for equipment and inventory
func (h *EquipmentHandler) HandleToolCall(ctx context.Context, name string, arguments json.RawMessage) (interface{}, error) {
	switch name {
	// Equipment tools
	case "search_equipment":
		return h.handleSearchEquipment(ctx, arguments)
	case "get_equipment_by_category":
		return h.handleGetEquipmentByCategory(ctx, arguments)
	case "get_sellers":
		return h.handleGetSellers(ctx)
	case "sync_seller_products":
		return h.handleSyncSellerProducts(ctx, arguments)

	// Inventory tools
	case "add_inventory_item":
		return h.handleAddInventoryItem(ctx, arguments)
	case "get_inventory":
		return h.handleGetInventory(ctx, arguments)
	case "update_inventory_item":
		return h.handleUpdateInventoryItem(ctx, arguments)
	case "remove_inventory_item":
		return h.handleRemoveInventoryItem(ctx, arguments)

	default:
		return nil, nil // Not handled by this handler
	}
}

// Equipment handlers

func (h *EquipmentHandler) handleSearchEquipment(ctx context.Context, arguments json.RawMessage) (interface{}, error) {
	var params struct {
		Query       string   `json:"query"`
		Category    string   `json:"category"`
		Seller      string   `json:"seller"`
		MinPrice    *float64 `json:"minPrice"`
		MaxPrice    *float64 `json:"maxPrice"`
		InStockOnly bool     `json:"inStockOnly"`
		Limit       int      `json:"limit"`
	}

	if err := json.Unmarshal(arguments, &params); err != nil {
		return nil, &ToolError{Message: "Invalid arguments: " + err.Error()}
	}

	if params.Limit == 0 {
		params.Limit = 20
	}

	searchParams := models.EquipmentSearchParams{
		Query:       params.Query,
		Category:    models.EquipmentCategory(params.Category),
		Seller:      params.Seller,
		MinPrice:    params.MinPrice,
		MaxPrice:    params.MaxPrice,
		InStockOnly: params.InStockOnly,
		Limit:       params.Limit,
	}

	response, err := h.equipmentSvc.Search(ctx, searchParams)
	if err != nil {
		return nil, &ToolError{Message: "Search failed: " + err.Error()}
	}

	return response, nil
}

func (h *EquipmentHandler) handleGetEquipmentByCategory(ctx context.Context, arguments json.RawMessage) (interface{}, error) {
	var params struct {
		Category string `json:"category"`
		Limit    int    `json:"limit"`
		Offset   int    `json:"offset"`
	}

	if err := json.Unmarshal(arguments, &params); err != nil {
		return nil, &ToolError{Message: "Invalid arguments: " + err.Error()}
	}

	if params.Limit == 0 {
		params.Limit = 20
	}

	response, err := h.equipmentSvc.GetByCategory(ctx, models.EquipmentCategory(params.Category), params.Limit, params.Offset)
	if err != nil {
		return nil, &ToolError{Message: "Failed to get category: " + err.Error()}
	}

	return response, nil
}

func (h *EquipmentHandler) handleGetSellers(ctx context.Context) (interface{}, error) {
	return h.equipmentSvc.GetSellers(), nil
}

func (h *EquipmentHandler) handleSyncSellerProducts(ctx context.Context, arguments json.RawMessage) (interface{}, error) {
	var params struct {
		Seller string `json:"seller"`
	}

	if len(arguments) > 0 {
		if err := json.Unmarshal(arguments, &params); err != nil {
			return nil, &ToolError{Message: "Invalid arguments: " + err.Error()}
		}
	}

	if err := h.equipmentSvc.SyncProducts(ctx); err != nil {
		return nil, &ToolError{Message: "Sync failed: " + err.Error()}
	}

	return map[string]interface{}{
		"status":  "success",
		"message": "Product sync triggered",
	}, nil
}

// Inventory handlers

func (h *EquipmentHandler) handleAddInventoryItem(ctx context.Context, arguments json.RawMessage) (interface{}, error) {
	var params models.AddInventoryParams

	if err := json.Unmarshal(arguments, &params); err != nil {
		return nil, &ToolError{Message: "Invalid arguments: " + err.Error()}
	}

	// MCP mode operates without user authentication - use empty userID for system/admin access
	item, err := h.inventorySvc.AddItem(ctx, "", params)
	if err != nil {
		return nil, &ToolError{Message: "Failed to add item: " + err.Error()}
	}

	return map[string]interface{}{
		"status": "success",
		"item":   item,
	}, nil
}

func (h *EquipmentHandler) handleGetInventory(ctx context.Context, arguments json.RawMessage) (interface{}, error) {
	var params struct {
		Category string `json:"category"`
		BuildID  string `json:"buildId"`
		Query    string `json:"query"`
		Limit    int    `json:"limit"`
		Offset   int    `json:"offset"`
	}

	if len(arguments) > 0 {
		if err := json.Unmarshal(arguments, &params); err != nil {
			return nil, &ToolError{Message: "Invalid arguments: " + err.Error()}
		}
	}

	filterParams := models.InventoryFilterParams{
		Category: models.EquipmentCategory(params.Category),
		BuildID:  params.BuildID,
		Query:    params.Query,
		Limit:    params.Limit,
		Offset:   params.Offset,
	}

	// MCP mode operates without user authentication - use empty userID
	response, err := h.inventorySvc.GetInventory(ctx, "", filterParams)
	if err != nil {
		return nil, &ToolError{Message: "Failed to get inventory: " + err.Error()}
	}

	return response, nil
}

func (h *EquipmentHandler) handleUpdateInventoryItem(ctx context.Context, arguments json.RawMessage) (interface{}, error) {
	var params models.UpdateInventoryParams

	if err := json.Unmarshal(arguments, &params); err != nil {
		return nil, &ToolError{Message: "Invalid arguments: " + err.Error()}
	}

	// MCP mode operates without user authentication - use empty userID
	item, err := h.inventorySvc.UpdateItem(ctx, "", params)
	if err != nil {
		return nil, &ToolError{Message: "Failed to update item: " + err.Error()}
	}

	return map[string]interface{}{
		"status": "success",
		"item":   item,
	}, nil
}

func (h *EquipmentHandler) handleRemoveInventoryItem(ctx context.Context, arguments json.RawMessage) (interface{}, error) {
	var params struct {
		ID string `json:"id"`
	}

	if err := json.Unmarshal(arguments, &params); err != nil {
		return nil, &ToolError{Message: "Invalid arguments: " + err.Error()}
	}

	// MCP mode operates without user authentication - use empty userID
	if err := h.inventorySvc.RemoveItem(ctx, params.ID, ""); err != nil {
		return nil, &ToolError{Message: "Failed to remove item: " + err.Error()}
	}

	return map[string]interface{}{
		"status":  "success",
		"message": "Item removed from inventory",
	}, nil
}

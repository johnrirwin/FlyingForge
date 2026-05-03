package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

// EquipmentHandler handles public read-only MCP tool calls for the gear catalog.
type EquipmentHandler struct {
	equipmentSvc EquipmentReader
	logger       *logging.Logger
}

// NewEquipmentHandler creates a new equipment handler.
func NewEquipmentHandler(equipmentSvc EquipmentReader, logger *logging.Logger) *EquipmentHandler {
	if equipmentSvc == nil {
		return nil
	}

	return &EquipmentHandler{
		equipmentSvc: equipmentSvc,
		logger:       logger,
	}
}

// GetTools returns public read-only tool definitions for the equipment catalog.
func (h *EquipmentHandler) GetTools() []ToolDefinition {
	if h == nil {
		return nil
	}

	return []ToolDefinition{
		{
			Name:        "search_equipment",
			Title:       "Search equipment",
			Description: "Search the moderated FlyingForge gear catalog for drone components and accessories.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"query": {
						"type": "string",
						"description": "Search query (for example 'ELRS receiver' or '5 inch frame')."
					},
					"category": {
						"type": "string",
						"enum": ["frames", "vtx", "flight_controllers", "esc", "aio", "stacks", "motors", "propellers", "receivers", "batteries", "cameras", "antennas", "gps", "accessories"],
						"description": "Optional equipment category filter."
					},
					"seller": {
						"type": "string",
						"description": "Optional seller/source filter. Currently only 'gear-catalog' is supported."
					},
					"minPrice": {
						"type": "number",
						"description": "Optional minimum price filter."
					},
					"maxPrice": {
						"type": "number",
						"description": "Optional maximum price filter."
					},
					"inStockOnly": {
						"type": "boolean",
						"description": "Whether to show only items with a shopping link."
					},
					"limit": {
						"type": "integer",
						"description": "Maximum number of results to return (default: 20)."
					},
					"offset": {
						"type": "integer",
						"description": "Pagination offset."
					},
					"sort": {
						"type": "string",
						"description": "Optional sort order supported by the equipment service."
					}
				}
			}`),
			SecuritySchemes: []SecurityScheme{{Type: "noauth"}},
			Annotations:     &ToolAnnotations{ReadOnlyHint: true},
		},
		{
			Name:        "get_equipment_by_category",
			Title:       "Browse equipment by category",
			Description: "Browse catalog gear for a specific category.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"category": {
						"type": "string",
						"enum": ["frames", "vtx", "flight_controllers", "esc", "aio", "stacks", "motors", "propellers", "receivers", "batteries", "cameras", "antennas", "gps", "accessories"],
						"description": "Equipment category to browse."
					},
					"limit": {
						"type": "integer",
						"description": "Maximum number of results to return (default: 20)."
					},
					"offset": {
						"type": "integer",
						"description": "Pagination offset."
					}
				},
				"required": ["category"]
			}`),
			SecuritySchemes: []SecurityScheme{{Type: "noauth"}},
			Annotations:     &ToolAnnotations{ReadOnlyHint: true},
		},
		{
			Name:            "get_sellers",
			Title:           "List equipment sellers",
			Description:     "List the available FlyingForge equipment data sources.",
			InputSchema:     json.RawMessage(`{"type":"object","properties":{}}`),
			SecuritySchemes: []SecurityScheme{{Type: "noauth"}},
			Annotations:     &ToolAnnotations{ReadOnlyHint: true},
		},
	}
}

// HandleToolCall handles public equipment tool calls.
func (h *EquipmentHandler) HandleToolCall(ctx context.Context, name string, arguments json.RawMessage) (interface{}, error) {
	if h == nil {
		return nil, nil
	}

	switch name {
	case "search_equipment":
		return h.handleSearchEquipment(ctx, arguments)
	case "get_equipment_by_category":
		return h.handleGetEquipmentByCategory(ctx, arguments)
	case "get_sellers":
		return h.handleGetSellers(ctx)
	default:
		return nil, nil
	}
}

func (h *EquipmentHandler) handleSearchEquipment(ctx context.Context, arguments json.RawMessage) (interface{}, error) {
	var params struct {
		Query       string   `json:"query"`
		Category    string   `json:"category"`
		Seller      string   `json:"seller"`
		MinPrice    *float64 `json:"minPrice"`
		MaxPrice    *float64 `json:"maxPrice"`
		InStockOnly bool     `json:"inStockOnly"`
		Limit       int      `json:"limit"`
		Offset      int      `json:"offset"`
		Sort        string   `json:"sort"`
	}

	if len(arguments) > 0 {
		if err := json.Unmarshal(arguments, &params); err != nil {
			return nil, &ToolError{Message: "Invalid arguments: " + err.Error()}
		}
	}

	if params.Limit == 0 {
		params.Limit = 20
	}

	response, err := h.equipmentSvc.Search(ctx, models.EquipmentSearchParams{
		Query:       strings.TrimSpace(params.Query),
		Category:    models.EquipmentCategory(strings.TrimSpace(params.Category)),
		Seller:      strings.TrimSpace(params.Seller),
		MinPrice:    params.MinPrice,
		MaxPrice:    params.MaxPrice,
		InStockOnly: params.InStockOnly,
		Limit:       params.Limit,
		Offset:      params.Offset,
		Sort:        strings.TrimSpace(params.Sort),
	})
	if err != nil {
		return nil, &ToolError{Message: "Search failed: " + err.Error()}
	}
	if response == nil {
		response = &models.EquipmentSearchResponse{}
	}

	query := strings.TrimSpace(params.Query)
	if query == "" {
		query = "catalog"
	}

	return ToolResultData{
		StructuredContent: response,
		Text:              fmt.Sprintf("Found %d equipment items for %q.", response.TotalCount, query),
	}, nil
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

	category := models.EquipmentCategory(strings.TrimSpace(params.Category))
	if category == "" {
		return nil, &ToolError{Message: "category is required"}
	}
	if params.Limit == 0 {
		params.Limit = 20
	}

	response, err := h.equipmentSvc.GetByCategory(ctx, category, params.Limit, params.Offset)
	if err != nil {
		return nil, &ToolError{Message: "Browse failed: " + err.Error()}
	}
	if response == nil {
		response = &models.EquipmentSearchResponse{}
	}

	return ToolResultData{
		StructuredContent: response,
		Text:              fmt.Sprintf("Found %d %s items.", response.TotalCount, category),
	}, nil
}

func (h *EquipmentHandler) handleGetSellers(ctx context.Context) (interface{}, error) {
	sellers := h.equipmentSvc.GetSellers()
	payload := map[string]any{
		"sellers": sellers,
		"count":   len(sellers),
	}

	return ToolResultData{
		StructuredContent: payload,
		Text:              fmt.Sprintf("Found %d equipment data source(s).", len(sellers)),
	}, nil
}

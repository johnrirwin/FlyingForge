package mcp

import (
	"context"
	"encoding/json"

	"github.com/johnrirwin/flyingforge/internal/aggregator"
	"github.com/johnrirwin/flyingforge/internal/equipment"
	"github.com/johnrirwin/flyingforge/internal/inventory"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

type Handler struct {
	agg          *aggregator.Aggregator
	equipmentSvc *equipment.Service
	inventorySvc inventory.InventoryManager
	logger       *logging.Logger
}

func NewHandler(agg *aggregator.Aggregator, equipmentSvc *equipment.Service, inventorySvc inventory.InventoryManager, logger *logging.Logger) *Handler {
	return &Handler{
		agg:          agg,
		equipmentSvc: equipmentSvc,
		inventorySvc: inventorySvc,
		logger:       logger,
	}
}

type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type GetNewsParams struct {
	Limit   int      `json:"limit"`
	Sources []string `json:"sources"`
	Tag     string   `json:"tag"`
	Query   string   `json:"query"`
}

func (h *Handler) GetTools() []ToolDefinition {
	// Get news tools
	tools := []ToolDefinition{
		{
			Name:        "get_drone_news",
			Description: "Get the latest drone news and community posts from various sources including news sites, Reddit, and forums.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"limit": {
						"type": "integer",
						"description": "Maximum number of items to return (default: 20)"
					},
					"source": {
						"type": "string",
						"description": "Filter by source name"
					},
					"tag": {
						"type": "string",
						"description": "Filter by tag (e.g., DJI, FPV, FAA)"
					},
					"search": {
						"type": "string",
						"description": "Search query to filter items"
					}
				}
			}`),
		},
		{
			Name:        "get_drone_news_sources",
			Description: "Get a list of all available drone news sources being aggregated.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
		},
		{
			Name:        "refresh_drone_news",
			Description: "Manually refresh the drone news feed from all sources.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {}
			}`),
		},
	}

	// Add equipment/inventory tools
	equipmentHandler := NewEquipmentHandler(h.equipmentSvc, h.inventorySvc, h.logger)
	tools = append(tools, equipmentHandler.GetTools()...)

	return tools
}

func (h *Handler) HandleToolCall(ctx context.Context, name string, arguments json.RawMessage) (interface{}, error) {
	// Try equipment/inventory tools first
	equipmentHandler := NewEquipmentHandler(h.equipmentSvc, h.inventorySvc, h.logger)
	result, err := equipmentHandler.HandleToolCall(ctx, name, arguments)
	if result != nil || err != nil {
		return result, err
	}

	// Handle news tools
	switch name {
	case "get_drone_news":
		return h.handleGetNews(ctx, arguments)
	case "get_drone_news_sources":
		return h.handleGetSources(ctx)
	case "refresh_drone_news":
		return h.handleRefresh(ctx)
	default:
		return nil, &ToolError{Message: "Unknown tool: " + name}
	}
}

func (h *Handler) handleGetNews(ctx context.Context, arguments json.RawMessage) (interface{}, error) {
	var params GetNewsParams
	if len(arguments) > 0 {
		if err := json.Unmarshal(arguments, &params); err != nil {
			return nil, &ToolError{Message: "Invalid arguments: " + err.Error()}
		}
	}

	if params.Limit == 0 {
		params.Limit = 20
	}

	filterParams := models.FilterParams{
		Limit:   params.Limit,
		Sources: params.Sources,
		Tag:     params.Tag,
		Query:   params.Query,
	}

	response := h.agg.GetItems(filterParams)
	return response, nil
}

func (h *Handler) handleGetSources(ctx context.Context) (interface{}, error) {
	sources := h.agg.GetSources()
	return map[string]interface{}{
		"sources": sources,
		"count":   len(sources),
	}, nil
}

func (h *Handler) handleRefresh(ctx context.Context) (interface{}, error) {
	if err := h.agg.Refresh(ctx); err != nil {
		return nil, &ToolError{Message: "Failed to refresh: " + err.Error()}
	}

	return map[string]interface{}{
		"status":  "success",
		"message": "Feed refreshed successfully",
	}, nil
}

type ToolError struct {
	Message string
}

func (e *ToolError) Error() string {
	return e.Message
}

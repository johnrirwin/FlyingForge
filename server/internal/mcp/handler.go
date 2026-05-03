package mcp

import (
	"context"
	"encoding/json"

	"github.com/johnrirwin/flyingforge/internal/aggregator"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

type EquipmentReader interface {
	Search(ctx context.Context, params models.EquipmentSearchParams) (*models.EquipmentSearchResponse, error)
	GetByCategory(ctx context.Context, category models.EquipmentCategory, limit, offset int) (*models.EquipmentSearchResponse, error)
	GetSellers() []models.SellerInfo
}

type AircraftReader interface {
	List(ctx context.Context, userID string, params models.AircraftListParams) (*models.AircraftListResponse, error)
	GetDetails(ctx context.Context, id string, userID string) (*models.AircraftDetailsResponse, error)
	GetReceiverSettings(ctx context.Context, aircraftID string, userID string) (*models.AircraftReceiverSettings, error)
}

type RadioReader interface {
	ListRadios(ctx context.Context, userID string, params models.RadioListParams) (*models.RadioListResponse, error)
	GetRadio(ctx context.Context, id string, userID string) (*models.Radio, error)
	ListBackups(ctx context.Context, radioID string, userID string, params models.RadioBackupListParams) (*models.RadioBackupListResponse, error)
}

type AircraftTuningReader interface {
	GetLatestTuningSnapshot(ctx context.Context, aircraftID string, userID string) (*models.AircraftTuningSnapshot, error)
}

type Handler struct {
	agg           *aggregator.Aggregator
	equipmentSvc  EquipmentReader
	aircraftSvc   AircraftReader
	radioSvc      RadioReader
	tuningReader  AircraftTuningReader
	privateScopes []string
	logger        *logging.Logger
}

func NewHandler(
	agg *aggregator.Aggregator,
	equipmentSvc EquipmentReader,
	aircraftSvc AircraftReader,
	radioSvc RadioReader,
	tuningReader AircraftTuningReader,
	privateScopes []string,
	logger *logging.Logger,
) *Handler {
	if len(privateScopes) == 0 {
		privateScopes = []string{"flyingforge.read"}
	}

	return &Handler{
		agg:           agg,
		equipmentSvc:  equipmentSvc,
		aircraftSvc:   aircraftSvc,
		radioSvc:      radioSvc,
		tuningReader:  tuningReader,
		privateScopes: append([]string(nil), privateScopes...),
		logger:        logger,
	}
}

type GetNewsParams struct {
	Limit   int      `json:"limit"`
	Sources []string `json:"sources"`
	Tag     string   `json:"tag"`
	Query   string   `json:"query"`
}

func (h *Handler) GetTools() []ToolDefinition {
	tools := []ToolDefinition{
		{
			Name:        "get_drone_news",
			Title:       "Get drone news",
			Description: "Get the latest drone news and community posts from FlyingForge news sources.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"limit": {
						"type": "integer",
						"description": "Maximum number of items to return (default: 20)"
					},
					"sources": {
						"type": "array",
						"items": { "type": "string" },
						"description": "Filter by source IDs"
					},
					"tag": {
						"type": "string",
						"description": "Filter by tag (for example DJI, FPV, or FAA)"
					},
					"query": {
						"type": "string",
						"description": "Search query to filter items"
					}
				}
			}`),
			SecuritySchemes: []SecurityScheme{{Type: "noauth"}},
			Annotations:     &ToolAnnotations{ReadOnlyHint: true},
		},
		{
			Name:            "get_drone_news_sources",
			Title:           "List drone news sources",
			Description:     "Get all available FlyingForge drone news sources.",
			InputSchema:     json.RawMessage(`{"type":"object","properties":{}}`),
			SecuritySchemes: []SecurityScheme{{Type: "noauth"}},
			Annotations:     &ToolAnnotations{ReadOnlyHint: true},
		},
	}

	equipmentHandler := NewEquipmentHandler(h.equipmentSvc, h.logger)
	tools = append(tools, equipmentHandler.GetTools()...)
	tools = append(tools, h.getPrivateReadOnlyTools()...)

	return tools
}

func (h *Handler) HandleToolCall(ctx context.Context, name string, arguments json.RawMessage) (interface{}, error) {
	if equipmentHandler := NewEquipmentHandler(h.equipmentSvc, h.logger); equipmentHandler != nil {
		result, err := equipmentHandler.HandleToolCall(ctx, name, arguments)
		if result != nil || err != nil {
			return result, err
		}
	}

	if result, err := h.handlePrivateToolCall(ctx, name, arguments); result != nil || err != nil {
		return result, err
	}

	switch name {
	case "get_drone_news":
		return h.handleGetNews(ctx, arguments)
	case "get_drone_news_sources":
		return h.handleGetSources(ctx)
	default:
		return nil, &ToolError{Message: "Unknown tool: " + name}
	}
}

func (h *Handler) IsPrivateTool(name string) bool {
	switch name {
	case "list_my_aircraft",
		"get_aircraft_details",
		"get_aircraft_receiver_summary",
		"get_aircraft_tuning",
		"list_my_radios",
		"get_radio_details",
		"list_radio_backups":
		return true
	default:
		return false
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

	response := h.agg.GetItems(ctx, filterParams)
	return ToolResultData{
		StructuredContent: response,
		Text:              "Fetched the latest FlyingForge drone news items.",
	}, nil
}

func (h *Handler) handleGetSources(ctx context.Context) (interface{}, error) {
	sources := h.agg.GetSources()
	payload := map[string]interface{}{
		"sources": sources,
		"count":   len(sources),
	}

	return ToolResultData{
		StructuredContent: payload,
		Text:              "Fetched the available FlyingForge drone news sources.",
	}, nil
}

type ToolError struct {
	Message string
}

func (e *ToolError) Error() string {
	return e.Message
}

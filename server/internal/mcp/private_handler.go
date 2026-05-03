package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/johnrirwin/flyingforge/internal/models"
)

type aircraftSummary struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Nickname    string              `json:"nickname,omitempty"`
	Type        models.AircraftType `json:"type,omitempty"`
	HasImage    bool                `json:"hasImage"`
	Description string              `json:"description,omitempty"`
	CreatedAt   time.Time           `json:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt"`
}

type aircraftInventoryItemSummary struct {
	ID           string                   `json:"id"`
	Name         string                   `json:"name"`
	Category     models.EquipmentCategory `json:"category,omitempty"`
	Manufacturer string                   `json:"manufacturer,omitempty"`
	Quantity     int                      `json:"quantity,omitempty"`
	CatalogID    string                   `json:"catalogId,omitempty"`
	ImageURL     string                   `json:"imageUrl,omitempty"`
	ProductURL   string                   `json:"productUrl,omitempty"`
}

type aircraftComponentAssignment struct {
	ID              string                        `json:"id"`
	AircraftID      string                        `json:"aircraftId"`
	Category        models.ComponentCategory      `json:"category"`
	InventoryItemID string                        `json:"inventoryItemId,omitempty"`
	Notes           string                        `json:"notes,omitempty"`
	CreatedAt       time.Time                     `json:"createdAt"`
	UpdatedAt       time.Time                     `json:"updatedAt"`
	InventoryItem   *aircraftInventoryItemSummary `json:"inventoryItem,omitempty"`
}

type aircraftDetailsToolResponse struct {
	Aircraft            aircraftSummary               `json:"aircraft"`
	Components          []aircraftComponentAssignment `json:"components"`
	HasReceiverSettings bool                          `json:"hasReceiverSettings"`
}

type aircraftReceiverSummaryResponse struct {
	AircraftID        string                            `json:"aircraftId"`
	ReceiverSettings  *models.ReceiverSanitizedSettings `json:"receiverSettings,omitempty"`
	HasReceiverConfig bool                              `json:"hasReceiverConfig"`
}

type aircraftTuningToolResponse struct {
	AircraftID      string                  `json:"aircraftId"`
	HasTuning       bool                    `json:"hasTuning"`
	FirmwareName    models.FCConfigFirmware `json:"firmwareName,omitempty"`
	FirmwareVersion string                  `json:"firmwareVersion,omitempty"`
	BoardTarget     string                  `json:"boardTarget,omitempty"`
	BoardName       string                  `json:"boardName,omitempty"`
	Tuning          *models.ParsedTuning    `json:"tuning,omitempty"`
	SnapshotID      string                  `json:"snapshotId,omitempty"`
	SnapshotDate    time.Time               `json:"snapshotDate,omitempty"`
	ParseStatus     models.ParseStatus      `json:"parseStatus,omitempty"`
	ParseWarnings   []string                `json:"parseWarnings,omitempty"`
	HasDiffBackup   bool                    `json:"hasDiffBackup"`
}

type radioSummary struct {
	ID             string                   `json:"id"`
	Manufacturer   models.RadioManufacturer `json:"manufacturer"`
	Model          string                   `json:"model"`
	FirmwareFamily models.FirmwareFamily    `json:"firmwareFamily,omitempty"`
	Notes          string                   `json:"notes,omitempty"`
	CreatedAt      time.Time                `json:"createdAt"`
	UpdatedAt      time.Time                `json:"updatedAt"`
}

type radioBackupSummary struct {
	ID         string            `json:"id"`
	RadioID    string            `json:"radioId"`
	BackupName string            `json:"backupName"`
	BackupType models.BackupType `json:"backupType"`
	FileName   string            `json:"fileName"`
	FileSize   int64             `json:"fileSize"`
	Checksum   string            `json:"checksum,omitempty"`
	CreatedAt  time.Time         `json:"createdAt"`
}

type listAircraftResponse struct {
	Aircraft   []aircraftSummary `json:"aircraft"`
	TotalCount int               `json:"totalCount"`
}

type listRadiosResponse struct {
	Radios     []radioSummary `json:"radios"`
	TotalCount int            `json:"totalCount"`
}

type listRadioBackupsResponse struct {
	Radio      radioSummary         `json:"radio"`
	Backups    []radioBackupSummary `json:"backups"`
	TotalCount int                  `json:"totalCount"`
}

func (h *Handler) getPrivateReadOnlyTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "list_my_aircraft",
			Title:       "List my aircraft",
			Description: "List the linked user's aircraft in FlyingForge.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"type": {
						"type": "string",
						"description": "Optional aircraft type filter."
					},
					"limit": {
						"type": "integer",
						"description": "Maximum number of aircraft to return (default: 20)."
					},
					"offset": {
						"type": "integer",
						"description": "Pagination offset."
					}
				}
			}`),
			SecuritySchemes: []SecurityScheme{{Type: "oauth2", Scopes: h.privateScopes}},
			Annotations:     &ToolAnnotations{ReadOnlyHint: true},
		},
		{
			Name:        "get_aircraft_details",
			Title:       "Get aircraft details",
			Description: "Get an aircraft plus its component assignments for the linked user. Raw receiver JSON is never returned.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"aircraftId": {
						"type": "string",
						"description": "The FlyingForge aircraft ID."
					}
				},
				"required": ["aircraftId"]
			}`),
			SecuritySchemes: []SecurityScheme{{Type: "oauth2", Scopes: h.privateScopes}},
			Annotations:     &ToolAnnotations{ReadOnlyHint: true},
		},
		{
			Name:        "get_aircraft_receiver_summary",
			Title:       "Get aircraft receiver summary",
			Description: "Get sanitized receiver settings for an aircraft. Sensitive fields like bind phrases, UID, Wi-Fi, and model-match values are removed.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"aircraftId": {
						"type": "string",
						"description": "The FlyingForge aircraft ID."
					}
				},
				"required": ["aircraftId"]
			}`),
			SecuritySchemes: []SecurityScheme{{Type: "oauth2", Scopes: h.privateScopes}},
			Annotations:     &ToolAnnotations{ReadOnlyHint: true},
		},
		{
			Name:        "get_aircraft_tuning",
			Title:       "Get aircraft tuning",
			Description: "Get the latest parsed tuning snapshot for an aircraft. Raw CLI dumps and diff backups are never returned.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"aircraftId": {
						"type": "string",
						"description": "The FlyingForge aircraft ID."
					}
				},
				"required": ["aircraftId"]
			}`),
			SecuritySchemes: []SecurityScheme{{Type: "oauth2", Scopes: h.privateScopes}},
			Annotations:     &ToolAnnotations{ReadOnlyHint: true},
		},
		{
			Name:        "list_my_radios",
			Title:       "List my radios",
			Description: "List the linked user's radios in FlyingForge.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"limit": {
						"type": "integer",
						"description": "Maximum number of radios to return (default: 20)."
					},
					"offset": {
						"type": "integer",
						"description": "Pagination offset."
					}
				}
			}`),
			SecuritySchemes: []SecurityScheme{{Type: "oauth2", Scopes: h.privateScopes}},
			Annotations:     &ToolAnnotations{ReadOnlyHint: true},
		},
		{
			Name:        "get_radio_details",
			Title:       "Get radio details",
			Description: "Get a linked user's radio details.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"radioId": {
						"type": "string",
						"description": "The FlyingForge radio ID."
					}
				},
				"required": ["radioId"]
			}`),
			SecuritySchemes: []SecurityScheme{{Type: "oauth2", Scopes: h.privateScopes}},
			Annotations:     &ToolAnnotations{ReadOnlyHint: true},
		},
		{
			Name:        "list_radio_backups",
			Title:       "List radio backups",
			Description: "List metadata for a radio's backups. Backup file bytes are never returned.",
			InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"radioId": {
						"type": "string",
						"description": "The FlyingForge radio ID."
					},
					"limit": {
						"type": "integer",
						"description": "Maximum number of backups to return (default: 20)."
					},
					"offset": {
						"type": "integer",
						"description": "Pagination offset."
					}
				},
				"required": ["radioId"]
			}`),
			SecuritySchemes: []SecurityScheme{{Type: "oauth2", Scopes: h.privateScopes}},
			Annotations:     &ToolAnnotations{ReadOnlyHint: true},
		},
	}
}

func (h *Handler) handlePrivateToolCall(ctx context.Context, name string, arguments json.RawMessage) (interface{}, error) {
	userID, err := h.requireAuthenticatedUser(ctx)
	if err != nil {
		if h.IsPrivateTool(name) {
			return nil, err
		}
		return nil, nil
	}

	switch name {
	case "list_my_aircraft":
		return h.handleListMyAircraft(ctx, userID, arguments)
	case "get_aircraft_details":
		return h.handleGetAircraftDetails(ctx, userID, arguments)
	case "get_aircraft_receiver_summary":
		return h.handleGetAircraftReceiverSummary(ctx, userID, arguments)
	case "get_aircraft_tuning":
		return h.handleGetAircraftTuning(ctx, userID, arguments)
	case "list_my_radios":
		return h.handleListMyRadios(ctx, userID, arguments)
	case "get_radio_details":
		return h.handleGetRadioDetails(ctx, userID, arguments)
	case "list_radio_backups":
		return h.handleListRadioBackups(ctx, userID, arguments)
	default:
		return nil, nil
	}
}

func (h *Handler) requireAuthenticatedUser(ctx context.Context) (string, error) {
	authState := RequestAuthFromContext(ctx)
	if strings.TrimSpace(authState.UserID) != "" {
		return strings.TrimSpace(authState.UserID), nil
	}

	if strings.TrimSpace(authState.ChallengeMessage) != "" {
		return "", &ToolError{Message: authState.ChallengeMessage}
	}

	return "", &ToolError{Message: "This tool requires a linked FlyingForge account authorized for MCP read access."}
}

func (h *Handler) handleListMyAircraft(ctx context.Context, userID string, arguments json.RawMessage) (interface{}, error) {
	if h.aircraftSvc == nil {
		return nil, &ToolError{Message: "Aircraft service is unavailable"}
	}

	var params struct {
		Type   string `json:"type"`
		Limit  int    `json:"limit"`
		Offset int    `json:"offset"`
	}
	if len(arguments) > 0 {
		if err := json.Unmarshal(arguments, &params); err != nil {
			return nil, &ToolError{Message: "Invalid arguments: " + err.Error()}
		}
	}
	if params.Limit == 0 {
		params.Limit = 20
	}

	response, err := h.aircraftSvc.List(ctx, userID, models.AircraftListParams{
		Type:   models.AircraftType(strings.TrimSpace(params.Type)),
		Limit:  params.Limit,
		Offset: params.Offset,
	})
	if err != nil {
		return nil, &ToolError{Message: "Failed to list aircraft: " + err.Error()}
	}
	if response == nil {
		response = &models.AircraftListResponse{}
	}

	items := make([]aircraftSummary, 0, len(response.Aircraft))
	for _, aircraft := range response.Aircraft {
		items = append(items, summarizeAircraft(aircraft))
	}

	payload := listAircraftResponse{
		Aircraft:   items,
		TotalCount: response.TotalCount,
	}

	return ToolResultData{
		StructuredContent: payload,
		Text:              fmt.Sprintf("Found %d aircraft in your FlyingForge hangar.", payload.TotalCount),
	}, nil
}

func (h *Handler) handleGetAircraftDetails(ctx context.Context, userID string, arguments json.RawMessage) (interface{}, error) {
	if h.aircraftSvc == nil {
		return nil, &ToolError{Message: "Aircraft service is unavailable"}
	}

	var params struct {
		AircraftID string `json:"aircraftId"`
	}
	if err := json.Unmarshal(arguments, &params); err != nil {
		return nil, &ToolError{Message: "Invalid arguments: " + err.Error()}
	}
	if strings.TrimSpace(params.AircraftID) == "" {
		return nil, &ToolError{Message: "aircraftId is required"}
	}

	details, err := h.aircraftSvc.GetDetails(ctx, strings.TrimSpace(params.AircraftID), userID)
	if err != nil {
		return nil, &ToolError{Message: "Failed to get aircraft details: " + err.Error()}
	}
	if details == nil {
		return nil, &ToolError{Message: "Aircraft not found"}
	}

	payload := aircraftDetailsToolResponse{
		Aircraft:            summarizeAircraft(details.Aircraft),
		Components:          summarizeAircraftComponents(details.Components),
		HasReceiverSettings: details.ReceiverSettings != nil,
	}

	return ToolResultData{
		StructuredContent: payload,
		Text:              fmt.Sprintf("Fetched details for aircraft %q.", payload.Aircraft.Name),
	}, nil
}

func (h *Handler) handleGetAircraftReceiverSummary(ctx context.Context, userID string, arguments json.RawMessage) (interface{}, error) {
	if h.aircraftSvc == nil {
		return nil, &ToolError{Message: "Aircraft service is unavailable"}
	}

	var params struct {
		AircraftID string `json:"aircraftId"`
	}
	if err := json.Unmarshal(arguments, &params); err != nil {
		return nil, &ToolError{Message: "Invalid arguments: " + err.Error()}
	}
	if strings.TrimSpace(params.AircraftID) == "" {
		return nil, &ToolError{Message: "aircraftId is required"}
	}

	settings, err := h.aircraftSvc.GetReceiverSettings(ctx, strings.TrimSpace(params.AircraftID), userID)
	if err != nil {
		return nil, &ToolError{Message: "Failed to get receiver settings: " + err.Error()}
	}

	payload := aircraftReceiverSummaryResponse{
		AircraftID:        strings.TrimSpace(params.AircraftID),
		ReceiverSettings:  models.SanitizeReceiverSettings(settings),
		HasReceiverConfig: settings != nil,
	}

	if payload.ReceiverSettings == nil {
		return ToolResultData{
			StructuredContent: payload,
			Text:              fmt.Sprintf("No sanitized receiver settings are available for aircraft %s.", payload.AircraftID),
		}, nil
	}

	return ToolResultData{
		StructuredContent: payload,
		Text:              fmt.Sprintf("Fetched sanitized receiver settings for aircraft %s.", payload.AircraftID),
	}, nil
}

func (h *Handler) handleGetAircraftTuning(ctx context.Context, userID string, arguments json.RawMessage) (interface{}, error) {
	if h.tuningReader == nil {
		return nil, &ToolError{Message: "Tuning service is unavailable"}
	}

	var params struct {
		AircraftID string `json:"aircraftId"`
	}
	if err := json.Unmarshal(arguments, &params); err != nil {
		return nil, &ToolError{Message: "Invalid arguments: " + err.Error()}
	}
	if strings.TrimSpace(params.AircraftID) == "" {
		return nil, &ToolError{Message: "aircraftId is required"}
	}

	snapshot, err := h.tuningReader.GetLatestTuningSnapshot(ctx, strings.TrimSpace(params.AircraftID), userID)
	if err != nil {
		return nil, &ToolError{Message: "Failed to get aircraft tuning: " + err.Error()}
	}

	payload := aircraftTuningToolResponse{
		AircraftID: strings.TrimSpace(params.AircraftID),
		HasTuning:  snapshot != nil,
	}

	if snapshot == nil {
		return ToolResultData{
			StructuredContent: payload,
			Text:              fmt.Sprintf("No tuning snapshot is available for aircraft %s.", payload.AircraftID),
		}, nil
	}

	parsedTuning, err := parseTuningData(snapshot.TuningData)
	if err != nil {
		return nil, &ToolError{Message: "Failed to parse aircraft tuning: " + err.Error()}
	}

	payload.FirmwareName = snapshot.FirmwareName
	payload.FirmwareVersion = snapshot.FirmwareVersion
	payload.BoardTarget = snapshot.BoardTarget
	payload.BoardName = snapshot.BoardName
	payload.Tuning = parsedTuning
	payload.SnapshotID = snapshot.ID
	payload.SnapshotDate = snapshot.CreatedAt
	payload.ParseStatus = snapshot.ParseStatus
	payload.ParseWarnings = snapshot.ParseWarnings
	payload.HasDiffBackup = snapshot.DiffBackup != ""

	return ToolResultData{
		StructuredContent: payload,
		Text:              fmt.Sprintf("Fetched the latest parsed tuning snapshot for aircraft %s.", payload.AircraftID),
	}, nil
}

func (h *Handler) handleListMyRadios(ctx context.Context, userID string, arguments json.RawMessage) (interface{}, error) {
	if h.radioSvc == nil {
		return nil, &ToolError{Message: "Radio service is unavailable"}
	}

	var params struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	}
	if len(arguments) > 0 {
		if err := json.Unmarshal(arguments, &params); err != nil {
			return nil, &ToolError{Message: "Invalid arguments: " + err.Error()}
		}
	}
	if params.Limit == 0 {
		params.Limit = 20
	}

	response, err := h.radioSvc.ListRadios(ctx, userID, models.RadioListParams{
		Limit:  params.Limit,
		Offset: params.Offset,
	})
	if err != nil {
		return nil, &ToolError{Message: "Failed to list radios: " + err.Error()}
	}
	if response == nil {
		response = &models.RadioListResponse{}
	}

	items := make([]radioSummary, 0, len(response.Radios))
	for _, radio := range response.Radios {
		items = append(items, summarizeRadio(radio))
	}

	payload := listRadiosResponse{
		Radios:     items,
		TotalCount: response.TotalCount,
	}

	return ToolResultData{
		StructuredContent: payload,
		Text:              fmt.Sprintf("Found %d radios in your FlyingForge account.", payload.TotalCount),
	}, nil
}

func (h *Handler) handleGetRadioDetails(ctx context.Context, userID string, arguments json.RawMessage) (interface{}, error) {
	if h.radioSvc == nil {
		return nil, &ToolError{Message: "Radio service is unavailable"}
	}

	var params struct {
		RadioID string `json:"radioId"`
	}
	if err := json.Unmarshal(arguments, &params); err != nil {
		return nil, &ToolError{Message: "Invalid arguments: " + err.Error()}
	}
	if strings.TrimSpace(params.RadioID) == "" {
		return nil, &ToolError{Message: "radioId is required"}
	}

	radio, err := h.radioSvc.GetRadio(ctx, strings.TrimSpace(params.RadioID), userID)
	if err != nil {
		return nil, &ToolError{Message: "Failed to get radio details: " + err.Error()}
	}
	if radio == nil {
		return nil, &ToolError{Message: "Radio not found"}
	}

	payload := summarizeRadio(*radio)

	return ToolResultData{
		StructuredContent: payload,
		Text:              fmt.Sprintf("Fetched radio details for %s %s.", payload.Manufacturer, payload.Model),
	}, nil
}

func (h *Handler) handleListRadioBackups(ctx context.Context, userID string, arguments json.RawMessage) (interface{}, error) {
	if h.radioSvc == nil {
		return nil, &ToolError{Message: "Radio service is unavailable"}
	}

	var params struct {
		RadioID string `json:"radioId"`
		Limit   int    `json:"limit"`
		Offset  int    `json:"offset"`
	}
	if err := json.Unmarshal(arguments, &params); err != nil {
		return nil, &ToolError{Message: "Invalid arguments: " + err.Error()}
	}
	if strings.TrimSpace(params.RadioID) == "" {
		return nil, &ToolError{Message: "radioId is required"}
	}
	if params.Limit == 0 {
		params.Limit = 20
	}

	radio, err := h.radioSvc.GetRadio(ctx, strings.TrimSpace(params.RadioID), userID)
	if err != nil {
		return nil, &ToolError{Message: "Failed to verify radio: " + err.Error()}
	}
	if radio == nil {
		return nil, &ToolError{Message: "Radio not found"}
	}

	response, err := h.radioSvc.ListBackups(ctx, radio.ID, userID, models.RadioBackupListParams{
		Limit:  params.Limit,
		Offset: params.Offset,
	})
	if err != nil {
		return nil, &ToolError{Message: "Failed to list radio backups: " + err.Error()}
	}
	if response == nil {
		response = &models.RadioBackupListResponse{}
	}

	backups := make([]radioBackupSummary, 0, len(response.Backups))
	for _, backup := range response.Backups {
		backups = append(backups, summarizeRadioBackup(backup))
	}

	payload := listRadioBackupsResponse{
		Radio:      summarizeRadio(*radio),
		Backups:    backups,
		TotalCount: response.TotalCount,
	}

	return ToolResultData{
		StructuredContent: payload,
		Text:              fmt.Sprintf("Found %d backup(s) for radio %s.", payload.TotalCount, payload.Radio.Model),
	}, nil
}

func summarizeAircraft(aircraft models.Aircraft) aircraftSummary {
	return aircraftSummary{
		ID:          aircraft.ID,
		Name:        aircraft.Name,
		Nickname:    aircraft.Nickname,
		Type:        aircraft.Type,
		HasImage:    aircraft.HasImage,
		Description: aircraft.Description,
		CreatedAt:   aircraft.CreatedAt,
		UpdatedAt:   aircraft.UpdatedAt,
	}
}

func summarizeAircraftComponents(components []models.AircraftComponent) []aircraftComponentAssignment {
	summaries := make([]aircraftComponentAssignment, 0, len(components))
	for _, component := range components {
		summary := aircraftComponentAssignment{
			ID:              component.ID,
			AircraftID:      component.AircraftID,
			Category:        component.Category,
			InventoryItemID: component.InventoryItemID,
			Notes:           component.Notes,
			CreatedAt:       component.CreatedAt,
			UpdatedAt:       component.UpdatedAt,
		}
		if component.InventoryItem != nil {
			summary.InventoryItem = &aircraftInventoryItemSummary{
				ID:           component.InventoryItem.ID,
				Name:         component.InventoryItem.Name,
				Category:     component.InventoryItem.Category,
				Manufacturer: component.InventoryItem.Manufacturer,
				Quantity:     component.InventoryItem.Quantity,
				CatalogID:    component.InventoryItem.CatalogID,
				ImageURL:     component.InventoryItem.ImageURL,
				ProductURL:   component.InventoryItem.ProductURL,
			}
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

func parseTuningData(data json.RawMessage) (*models.ParsedTuning, error) {
	if len(data) == 0 {
		return nil, nil
	}

	parsed := &models.ParsedTuning{}
	if err := json.Unmarshal(data, parsed); err != nil {
		return nil, err
	}

	return parsed, nil
}

func summarizeRadio(radio models.Radio) radioSummary {
	return radioSummary{
		ID:             radio.ID,
		Manufacturer:   radio.Manufacturer,
		Model:          radio.Model,
		FirmwareFamily: radio.FirmwareFamily,
		Notes:          radio.Notes,
		CreatedAt:      radio.CreatedAt,
		UpdatedAt:      radio.UpdatedAt,
	}
}

func summarizeRadioBackup(backup models.RadioBackup) radioBackupSummary {
	return radioBackupSummary{
		ID:         backup.ID,
		RadioID:    backup.RadioID,
		BackupName: backup.BackupName,
		BackupType: backup.BackupType,
		FileName:   backup.FileName,
		FileSize:   backup.FileSize,
		Checksum:   backup.Checksum,
		CreatedAt:  backup.CreatedAt,
	}
}

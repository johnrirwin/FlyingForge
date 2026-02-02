package models

import (
	"encoding/json"
	"time"
)

// AircraftType represents the type of aircraft
type AircraftType string

const (
	AircraftTypeQuad      AircraftType = "quad"
	AircraftTypeFixedWing AircraftType = "fixed_wing"
	AircraftTypeWhoop     AircraftType = "whoop"
	AircraftTypeCineLift  AircraftType = "cine_lift"
	AircraftTypeLongRange AircraftType = "long_range"
	AircraftTypeOther     AircraftType = "other"
)

// ComponentCategory represents the type of component on an aircraft
type ComponentCategory string

const (
	ComponentCategoryFC         ComponentCategory = "fc"
	ComponentCategoryESC        ComponentCategory = "esc"
	ComponentCategoryELRSModule ComponentCategory = "elrs_module"
	ComponentCategoryVTX        ComponentCategory = "vtx"
	ComponentCategoryMotors     ComponentCategory = "motors"
	ComponentCategoryCamera     ComponentCategory = "camera"
	ComponentCategoryFrame      ComponentCategory = "frame"
	ComponentCategoryProps      ComponentCategory = "propellers"
	ComponentCategoryAntenna    ComponentCategory = "antenna"
)

// Aircraft represents a user's aircraft/drone
type Aircraft struct {
	ID          string       `json:"id"`
	UserID      string       `json:"userId,omitempty"`
	Name        string       `json:"name"`
	Nickname    string       `json:"nickname,omitempty"`
	Type        AircraftType `json:"type,omitempty"`
	HasImage    bool         `json:"hasImage"`
	ImageType   string       `json:"-"` // MIME type: image/jpeg or image/png
	ImageData   []byte       `json:"-"` // Binary image data, not serialized to JSON
	Description string       `json:"description,omitempty"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`

	// Related data (populated on details fetch)
	Components   []AircraftComponent   `json:"components,omitempty"`
	ELRSSettings *AircraftELRSSettings `json:"elrsSettings,omitempty"`
}

// AircraftComponent represents a component installed on an aircraft
type AircraftComponent struct {
	ID              string            `json:"id"`
	AircraftID      string            `json:"aircraftId"`
	Category        ComponentCategory `json:"category"`
	InventoryItemID string            `json:"inventoryItemId"`
	Notes           string            `json:"notes,omitempty"`
	CreatedAt       time.Time         `json:"createdAt"`
	UpdatedAt       time.Time         `json:"updatedAt"`

	// Populated from inventory item on fetch
	InventoryItem *InventoryItem `json:"inventoryItem,omitempty"`
}

// AircraftELRSSettings holds ELRS configuration for an aircraft
type AircraftELRSSettings struct {
	ID         string          `json:"id"`
	AircraftID string          `json:"aircraftId"`
	Settings   json.RawMessage `json:"settings"` // Flexible JSON for any ELRS settings
	CreatedAt  time.Time       `json:"createdAt"`
	UpdatedAt  time.Time       `json:"updatedAt"`
}

// ELRSSettingsData represents the structured ELRS settings
type ELRSSettingsData struct {
	ModelMatch     *bool                  `json:"modelMatch,omitempty"`
	ModelID        *int                   `json:"modelId,omitempty"`
	BindPhrase     string                 `json:"bindPhrase,omitempty"`
	PacketRate     string                 `json:"packetRate,omitempty"`     // e.g., "250Hz", "500Hz"
	TelemetryRatio string                 `json:"telemetryRatio,omitempty"` // e.g., "1:128", "1:64"
	TXPower        string                 `json:"txPower,omitempty"`        // e.g., "250mW", "500mW"
	SwitchMode     string                 `json:"switchMode,omitempty"`     // e.g., "Hybrid", "Wide"
	RFProfile      string                 `json:"rfProfile,omitempty"`
	Extra          map[string]interface{} `json:"extra,omitempty"` // Any additional fields
}

// CreateAircraftParams defines parameters for creating an aircraft
type CreateAircraftParams struct {
	Name        string       `json:"name"`
	Nickname    string       `json:"nickname,omitempty"`
	Type        AircraftType `json:"type,omitempty"`
	Description string       `json:"description,omitempty"`
}

// UpdateAircraftParams defines parameters for updating an aircraft
type UpdateAircraftParams struct {
	ID          string        `json:"id"`
	Name        *string       `json:"name,omitempty"`
	Nickname    *string       `json:"nickname,omitempty"`
	Type        *AircraftType `json:"type,omitempty"`
	Description *string       `json:"description,omitempty"`
}

// SetAircraftImageParams defines parameters for uploading an aircraft image
type SetAircraftImageParams struct {
	AircraftID string
	ImageType  string // "image/jpeg" or "image/png"
	ImageData  []byte
}

// SetComponentParams defines parameters for setting a component on an aircraft
type SetComponentParams struct {
	AircraftID      string            `json:"aircraftId"`
	Category        ComponentCategory `json:"category"`
	InventoryItemID string            `json:"inventoryItemId,omitempty"`
	Notes           string            `json:"notes,omitempty"`

	// If inventory item doesn't exist, create it with these fields
	NewGear *AddInventoryParams `json:"newGear,omitempty"`
}

// SetELRSSettingsParams defines parameters for setting ELRS settings
type SetELRSSettingsParams struct {
	AircraftID string          `json:"aircraftId"`
	Settings   json.RawMessage `json:"settings"`
}

// AircraftListParams defines filters for listing aircraft
type AircraftListParams struct {
	Type   AircraftType `json:"type,omitempty"`
	Limit  int          `json:"limit,omitempty"`
	Offset int          `json:"offset,omitempty"`
}

// AircraftListResponse is the response for listing aircraft
type AircraftListResponse struct {
	Aircraft   []Aircraft `json:"aircraft"`
	TotalCount int        `json:"totalCount"`
}

// AircraftDetailsResponse includes all related data for an aircraft
type AircraftDetailsResponse struct {
	Aircraft     Aircraft              `json:"aircraft"`
	Components   []AircraftComponent   `json:"components"`
	ELRSSettings *AircraftELRSSettings `json:"elrsSettings,omitempty"`
}

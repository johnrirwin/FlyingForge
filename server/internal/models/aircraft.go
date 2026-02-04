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
	ComponentCategoryFC       ComponentCategory = "fc"
	ComponentCategoryESC      ComponentCategory = "esc"
	ComponentCategoryAIO      ComponentCategory = "aio"
	ComponentCategoryReceiver ComponentCategory = "receiver"
	ComponentCategoryVTX      ComponentCategory = "vtx"
	ComponentCategoryMotors   ComponentCategory = "motors"
	ComponentCategoryCamera   ComponentCategory = "camera"
	ComponentCategoryFrame    ComponentCategory = "frame"
	ComponentCategoryProps    ComponentCategory = "propellers"
	ComponentCategoryAntenna  ComponentCategory = "antenna"
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
	Components       []AircraftComponent       `json:"components,omitempty"`
	ReceiverSettings *AircraftReceiverSettings `json:"receiverSettings,omitempty"`
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

// AircraftReceiverSettings holds receiver configuration for an aircraft
type AircraftReceiverSettings struct {
	ID         string          `json:"id"`
	AircraftID string          `json:"aircraftId"`
	Settings   json.RawMessage `json:"settings"` // Flexible JSON for receiver settings
	CreatedAt  time.Time       `json:"createdAt"`
	UpdatedAt  time.Time       `json:"updatedAt"`
}

// ReceiverSettingsData represents the structured receiver settings
// WARNING: This struct contains SENSITIVE fields that must never be exposed publicly
type ReceiverSettingsData struct {
	// SENSITIVE - Never expose publicly
	ModelMatch    *int   `json:"modelMatch,omitempty"`    // SENSITIVE: Model match number (0-63) - frontend uses this
	ModelMatchNum *int   `json:"modelMatchNum,omitempty"` // SENSITIVE: Model match number (alternate name)
	ModelID       *int   `json:"modelId,omitempty"`       // SENSITIVE: Model ID
	BindPhrase    string `json:"bindPhrase,omitempty"`    // SENSITIVE: Bind phrase secret
	BindingPhrase string `json:"bindingPhrase,omitempty"` // SENSITIVE: Bind phrase (frontend uses this name)
	UID           string `json:"uid,omitempty"`           // SENSITIVE: Receiver UID
	WifiPassword  string `json:"wifiPassword,omitempty"`  // SENSITIVE: WiFi password
	WifiSSID      string `json:"wifiSSID,omitempty"`      // May contain personal info

	// SAFE to expose publicly
	ReceiverModel    string `json:"receiverModel,omitempty"`    // e.g., "EP1", "RP1", "RP3"
	PacketRate       string `json:"packetRate,omitempty"`       // e.g., "250Hz", "500Hz"
	Rate             *int   `json:"rate,omitempty"`             // Numeric rate (frontend uses this)
	TelemetryRatio   string `json:"telemetryRatio,omitempty"`   // e.g., "1:128", "1:64"
	Tlm              *int   `json:"tlm,omitempty"`              // Numeric telemetry ratio (frontend uses this)
	TXPower          string `json:"txPower,omitempty"`          // e.g., "250mW", "500mW"
	Power            *int   `json:"power,omitempty"`            // Numeric power in mW (frontend uses this)
	SwitchMode       string `json:"switchMode,omitempty"`       // e.g., "Hybrid", "Wide"
	OutputPower      string `json:"outputPower,omitempty"`      // Output power (static or dynamic)
	RegulatoryDomain string `json:"regulatoryDomain,omitempty"` // e.g., "FCC", "LBT"
	FirmwareVersion  string `json:"firmwareVersion,omitempty"`  // e.g., "3.4.0"
	RXProtocol       string `json:"rxProtocol,omitempty"`       // Protocol type
	RFProfile        string `json:"rfProfile,omitempty"`
	DeviceName       string `json:"deviceName,omitempty"` // Device name (safe)

	Extra map[string]interface{} `json:"extra,omitempty"` // Any additional fields (may contain sensitive data)
}

// Sanitize returns a sanitized copy of receiver settings safe for public exposure
// This method strips all sensitive fields: BindPhrase, ModelMatch, UID, etc.
func (e *ReceiverSettingsData) Sanitize() *ReceiverSanitizedSettings {
	if e == nil {
		return nil
	}

	// Simply pass through the safe fields using the same names
	return &ReceiverSanitizedSettings{
		Rate:       e.Rate,
		Tlm:        e.Tlm,
		Power:      e.Power,
		DeviceName: e.DeviceName,
	}
}

// SanitizeReceiverSettings parses raw JSON receiver settings and returns a sanitized version
// This is the primary function to use when exposing receiver data publicly
func SanitizeReceiverSettings(settings *AircraftReceiverSettings) *ReceiverSanitizedSettings {
	if settings == nil || len(settings.Settings) == 0 {
		return nil
	}

	var data ReceiverSettingsData
	if err := json.Unmarshal(settings.Settings, &data); err != nil {
		return nil
	}

	return data.Sanitize()
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

// SetReceiverSettingsParams defines parameters for setting receiver settings
type SetReceiverSettingsParams struct {
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
	Aircraft         Aircraft                  `json:"aircraft"`
	Components       []AircraftComponent       `json:"components"`
	ReceiverSettings *AircraftReceiverSettings `json:"receiverSettings,omitempty"`
}

package models

import (
	"encoding/json"
	"time"
)

// FCConfigFirmware represents the flight controller firmware type
type FCConfigFirmware string

const (
	FirmwareBetaflight FCConfigFirmware = "betaflight"
	FirmwareINAV       FCConfigFirmware = "inav"
	FirmwareArdupilot  FCConfigFirmware = "ardupilot"
	FirmwareUnknown    FCConfigFirmware = "unknown"
)

// ParseStatus represents the status of config parsing
type ParseStatus string

const (
	ParseStatusSuccess ParseStatus = "success"
	ParseStatusPartial ParseStatus = "partial"
	ParseStatusFailed  ParseStatus = "failed"
)

// FlightControllerConfig represents a saved Betaflight CLI dump configuration
type FlightControllerConfig struct {
	ID              string           `json:"id"`
	UserID          string           `json:"userId,omitempty"`
	InventoryItemID string           `json:"inventoryItemId"` // Links to the FC in inventory
	Name            string           `json:"name"`            // User-given name for this config backup
	Notes           string           `json:"notes,omitempty"`
	RawCLIDump      string           `json:"rawCliDump"`      // Full CLI dump text, always preserved
	FirmwareName    FCConfigFirmware `json:"firmwareName"`    // betaflight, inav, etc.
	FirmwareVersion string           `json:"firmwareVersion"` // e.g., "4.4.2"
	BoardTarget     string           `json:"boardTarget"`     // e.g., "STM32F405"
	BoardName       string           `json:"boardName"`       // e.g., "MATEKF405"
	MCUType         string           `json:"mcuType"`         // e.g., "STM32F405"
	ParseStatus     ParseStatus      `json:"parseStatus"`
	ParseWarnings   []string         `json:"parseWarnings,omitempty"`
	ParsedTuning    *ParsedTuning    `json:"parsedTuning,omitempty"` // Extracted tuning data
	CreatedAt       time.Time        `json:"createdAt"`
	UpdatedAt       time.Time        `json:"updatedAt"`
}

// ParsedTuning contains all extracted tuning data from a Betaflight CLI dump
type ParsedTuning struct {
	PIDs       *PIDProfile       `json:"pids,omitempty"`
	Rates      *RateProfile      `json:"rates,omitempty"`
	Filters    *FilterSettings   `json:"filters,omitempty"`
	MotorMixer *MotorMixerConfig `json:"motorMixer,omitempty"`
	Features   *FeatureFlags     `json:"features,omitempty"`
	Misc       *MiscSettings     `json:"misc,omitempty"`

	// All parsed profiles
	PIDProfiles  []PIDProfile  `json:"pidProfiles,omitempty"`
	RateProfiles []RateProfile `json:"rateProfiles,omitempty"`

	// Active profile indexes
	ActivePIDProfile  int `json:"activePidProfile"`
	ActiveRateProfile int `json:"activeRateProfile"`
}

// PIDProfile contains PID tuning values for a profile
type PIDProfile struct {
	ProfileIndex int    `json:"profileIndex"`
	ProfileName  string `json:"profileName,omitempty"`

	// PID values for each axis
	Roll  AxisPID `json:"roll"`
	Pitch AxisPID `json:"pitch"`
	Yaw   AxisPID `json:"yaw"`

	// Level mode PIDs
	Level *AxisPID `json:"level,omitempty"`

	// Additional PID-related settings
	AntiGravityGain         int    `json:"antiGravityGain,omitempty"`       // Anti-gravity gain (0-250)
	AntiGravityMode         string `json:"antiGravityMode,omitempty"`       // SMOOTH, STEP
	FeedforwardTransition   int    `json:"feedforwardTransition,omitempty"` // 0-100
	FeedforwardAveraging    int    `json:"feedforwardAveraging,omitempty"`
	FeedforwardSmooth       int    `json:"feedforwardSmooth,omitempty"`
	FeedforwardJitterFactor int    `json:"feedforwardJitterFactor,omitempty"`
	FeedforwardBoost        int    `json:"feedforwardBoost,omitempty"`

	// D-term settings
	DMinRoll    int `json:"dMinRoll,omitempty"`
	DMinPitch   int `json:"dMinPitch,omitempty"`
	DMinYaw     int `json:"dMinYaw,omitempty"`
	DMinGain    int `json:"dMinGain,omitempty"`
	DMinAdvance int `json:"dMinAdvance,omitempty"`

	// I-term settings
	ITermRelax       string `json:"iTermRelax,omitempty"`     // OFF, RP, RPY
	ITermRelaxType   string `json:"iTermRelaxType,omitempty"` // GYRO, SETPOINT
	ITermRelaxCutoff int    `json:"iTermRelaxCutoff,omitempty"`

	// TPA (Throttle PID Attenuation)
	TPARate       int    `json:"tpaRate,omitempty"`
	TPABreakpoint int    `json:"tpaBreakpoint,omitempty"`
	TPAMode       string `json:"tpaMode,omitempty"` // PD, D
}

// AxisPID contains P, I, D, and FF values for a single axis
type AxisPID struct {
	P  int `json:"p"`
	I  int `json:"i"`
	D  int `json:"d"`
	FF int `json:"ff,omitempty"` // Feedforward
}

// RateProfile contains rate settings
type RateProfile struct {
	ProfileIndex int    `json:"profileIndex"`
	ProfileName  string `json:"profileName,omitempty"`
	RateType     string `json:"rateType,omitempty"` // BETAFLIGHT, RACEFLIGHT, ACTUAL, QUICK

	// RC Rates
	RCRates RateAxisValues `json:"rcRates"`

	// Super Rates
	SuperRates RateAxisValues `json:"superRates"`

	// RC Expo
	RCExpo RateAxisValues `json:"rcExpo"`

	// For ACTUAL rate type
	CenterSensitivity RateAxisValues `json:"centerSensitivity,omitempty"`
	MaxRate           RateAxisValues `json:"maxRate,omitempty"`

	// Throttle settings
	ThrottleMid          int    `json:"throttleMid,omitempty"`
	ThrottleExpo         int    `json:"throttleExpo,omitempty"`
	ThrottleLimitType    string `json:"throttleLimitType,omitempty"`
	ThrottleLimitPercent int    `json:"throttleLimitPercent,omitempty"`
}

// RateAxisValues contains values for roll, pitch, yaw
type RateAxisValues struct {
	Roll  int `json:"roll"`
	Pitch int `json:"pitch"`
	Yaw   int `json:"yaw"`
}

// FilterSettings contains filter configuration
type FilterSettings struct {
	// Gyro lowpass filters
	GyroLowpassEnabled  bool   `json:"gyroLowpassEnabled"`
	GyroLowpassHz       int    `json:"gyroLowpassHz,omitempty"`
	GyroLowpassType     string `json:"gyroLowpassType,omitempty"` // PT1, BIQUAD, PT2, PT3
	GyroLowpass2Enabled bool   `json:"gyroLowpass2Enabled"`
	GyroLowpass2Hz      int    `json:"gyroLowpass2Hz,omitempty"`
	GyroLowpass2Type    string `json:"gyroLowpass2Type,omitempty"`

	// Dynamic lowpass
	GyroDynLowpassEnabled bool `json:"gyroDynLowpassEnabled"`
	GyroDynLowpassMinHz   int  `json:"gyroDynLowpassMinHz,omitempty"`
	GyroDynLowpassMaxHz   int  `json:"gyroDynLowpassMaxHz,omitempty"`

	// Gyro notch filters
	GyroNotch1Enabled bool `json:"gyroNotch1Enabled"`
	GyroNotch1Hz      int  `json:"gyroNotch1Hz,omitempty"`
	GyroNotch1Cutoff  int  `json:"gyroNotch1Cutoff,omitempty"`
	GyroNotch2Enabled bool `json:"gyroNotch2Enabled"`
	GyroNotch2Hz      int  `json:"gyroNotch2Hz,omitempty"`
	GyroNotch2Cutoff  int  `json:"gyroNotch2Cutoff,omitempty"`

	// D-term lowpass filters
	DTermLowpassEnabled  bool   `json:"dtermLowpassEnabled"`
	DTermLowpassHz       int    `json:"dtermLowpassHz,omitempty"`
	DTermLowpassType     string `json:"dtermLowpassType,omitempty"`
	DTermLowpass2Enabled bool   `json:"dtermLowpass2Enabled"`
	DTermLowpass2Hz      int    `json:"dtermLowpass2Hz,omitempty"`
	DTermLowpass2Type    string `json:"dtermLowpass2Type,omitempty"`

	// Dynamic D-term lowpass
	DTermDynLowpassEnabled bool `json:"dtermDynLowpassEnabled"`
	DTermDynLowpassMinHz   int  `json:"dtermDynLowpassMinHz,omitempty"`
	DTermDynLowpassMaxHz   int  `json:"dtermDynLowpassMaxHz,omitempty"`

	// D-term notch
	DTermNotchEnabled bool `json:"dtermNotchEnabled"`
	DTermNotchHz      int  `json:"dtermNotchHz,omitempty"`
	DTermNotchCutoff  int  `json:"dtermNotchCutoff,omitempty"`

	// RPM filtering
	RPMFilterEnabled   bool `json:"rpmFilterEnabled"`
	RPMFilterHarmonics int  `json:"rpmFilterHarmonics,omitempty"`
	RPMFilterMinHz     int  `json:"rpmFilterMinHz,omitempty"`
	RPMFilterFadeRange int  `json:"rpmFilterFadeRange,omitempty"`
	RPMFilterQFactor   int  `json:"rpmFilterQFactor,omitempty"`

	// Dynamic notch
	DynNotchEnabled bool `json:"dynNotchEnabled"`
	DynNotchCount   int  `json:"dynNotchCount,omitempty"`
	DynNotchQ       int  `json:"dynNotchQ,omitempty"`
	DynNotchMinHz   int  `json:"dynNotchMinHz,omitempty"`
	DynNotchMaxHz   int  `json:"dynNotchMaxHz,omitempty"`
}

// MotorMixerConfig contains motor and mixer settings
type MotorMixerConfig struct {
	MotorProtocol      string `json:"motorProtocol,omitempty"` // DSHOT150, DSHOT300, DSHOT600, etc.
	MotorPWMRate       int    `json:"motorPwmRate,omitempty"`
	MotorIdlePercent   int    `json:"motorIdlePercent,omitempty"`   // motor_idle_throttle_percent * 100
	DigitalIdlePercent int    `json:"digitalIdlePercent,omitempty"` // dshot_idle_value
	MotorPoles         int    `json:"motorPoles,omitempty"`
	MixerType          string `json:"mixerType,omitempty"`

	// Loop times
	GyroSyncDenom int `json:"gyroSyncDenom,omitempty"`
	PIDLoopDenom  int `json:"pidLoopDenom,omitempty"`
	GyroHz        int `json:"gyroHz,omitempty"` // Calculated from gyro_sync_denom
	PIDHz         int `json:"pidHz,omitempty"`  // Calculated from pid_process_denom

	// Bidirectional DSHOT
	DShotBidir   bool   `json:"dshotBidir,omitempty"`
	DShotBitbang string `json:"dshotBitbang,omitempty"` // OFF, ON, AUTO
}

// FeatureFlags contains enabled/disabled features
type FeatureFlags struct {
	// Common features
	GPS           bool `json:"gps"`
	Telemetry     bool `json:"telemetry"`
	OSD           bool `json:"osd"`
	LED_STRIP     bool `json:"ledStrip"`
	Airmode       bool `json:"airmode"`
	AntiGravity   bool `json:"antiGravity"`
	DynamicFilter bool `json:"dynamicFilter"`
	RPMFilter     bool `json:"rpmFilter"`
}

// MiscSettings contains other settings of interest
type MiscSettings struct {
	Name           string `json:"name,omitempty"`          // Pilot/craft name
	CrashRecovery  string `json:"crashRecovery,omitempty"` // OFF, ON, BEEP, DISARM
	GyroCalibNoise int    `json:"gyroCalibNoise,omitempty"`
	AccCalibX      int    `json:"accCalibX,omitempty"`
	AccCalibY      int    `json:"accCalibY,omitempty"`
	AccCalibZ      int    `json:"accCalibZ,omitempty"`

	// Battery
	VBatMinCellVoltage     int `json:"vbatMinCellVoltage,omitempty"`
	VBatMaxCellVoltage     int `json:"vbatMaxCellVoltage,omitempty"`
	VBatWarningCellVoltage int `json:"vbatWarningCellVoltage,omitempty"`
}

// AircraftTuningSnapshot represents a point-in-time tuning state for an aircraft
type AircraftTuningSnapshot struct {
	ID                       string `json:"id"`
	AircraftID               string `json:"aircraftId"`
	FlightControllerID       string `json:"flightControllerId,omitempty"`       // inventory_item_id of the FC
	FlightControllerConfigID string `json:"flightControllerConfigId,omitempty"` // config that produced this snapshot

	// Firmware info (copied from config for quick access)
	FirmwareName    FCConfigFirmware `json:"firmwareName"`
	FirmwareVersion string           `json:"firmwareVersion"`
	BoardTarget     string           `json:"boardTarget"`
	BoardName       string           `json:"boardName"`

	// Parsed tuning data (denormalized from config for quick access)
	TuningData json.RawMessage `json:"tuningData"` // Serialized ParsedTuning

	// Parse status
	ParseStatus   ParseStatus `json:"parseStatus"`
	ParseWarnings []string    `json:"parseWarnings,omitempty"`

	Notes     string    `json:"notes,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// SaveFCConfigParams represents parameters for saving a new FC config
type SaveFCConfigParams struct {
	InventoryItemID string `json:"inventoryItemId"` // Required: which FC in inventory
	Name            string `json:"name"`            // Optional: name for this backup
	Notes           string `json:"notes,omitempty"`
	RawCLIDump      string `json:"rawCliDump"` // Required: full CLI dump text
}

// UpdateFCConfigParams represents parameters for updating an FC config
type UpdateFCConfigParams struct {
	Name  *string `json:"name,omitempty"`
	Notes *string `json:"notes,omitempty"`
}

// FCConfigListParams represents parameters for listing FC configs
type FCConfigListParams struct {
	InventoryItemID string `json:"inventoryItemId,omitempty"` // Filter by FC
	Limit           int    `json:"limit,omitempty"`
	Offset          int    `json:"offset,omitempty"`
}

// FCConfigListResponse represents the response for listing FC configs
type FCConfigListResponse struct {
	Configs    []FlightControllerConfig `json:"configs"`
	TotalCount int                      `json:"totalCount"`
}

// TuningSnapshotResponse represents the response for aircraft tuning
type TuningSnapshotResponse struct {
	Snapshot     *AircraftTuningSnapshot `json:"snapshot,omitempty"`
	ParsedTuning *ParsedTuning           `json:"parsedTuning,omitempty"` // Convenience: unmarshaled TuningData
}

// AircraftTuningResponse represents the response for getting aircraft tuning data
type AircraftTuningResponse struct {
	AircraftID      string           `json:"aircraftId"`
	HasTuning       bool             `json:"hasTuning"`
	FirmwareName    FCConfigFirmware `json:"firmwareName,omitempty"`
	FirmwareVersion string           `json:"firmwareVersion,omitempty"`
	BoardTarget     string           `json:"boardTarget,omitempty"`
	BoardName       string           `json:"boardName,omitempty"`
	Tuning          *ParsedTuning    `json:"tuning,omitempty"`
	SnapshotID      string           `json:"snapshotId,omitempty"`
	SnapshotDate    time.Time        `json:"snapshotDate,omitempty"`
	ParseStatus     ParseStatus      `json:"parseStatus,omitempty"`
	ParseWarnings   []string         `json:"parseWarnings,omitempty"`
}

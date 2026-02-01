package models

import (
	"time"
)

// RadioManufacturer represents a radio transmitter manufacturer
type RadioManufacturer string

const (
	ManufacturerRadioMaster RadioManufacturer = "RadioMaster"
	ManufacturerFrSky       RadioManufacturer = "FrSky"
	ManufacturerJumper      RadioManufacturer = "Jumper"
	ManufacturerTBS         RadioManufacturer = "TBS"
)

// FirmwareFamily represents the firmware running on the radio
type FirmwareFamily string

const (
	FirmwareFamilyEdgeTX FirmwareFamily = "EdgeTX"
	FirmwareFamilyOpenTX FirmwareFamily = "OpenTX"
)

// BackupType represents the type of radio configuration backup
type BackupType string

const (
	BackupTypeEdgeTXModels  BackupType = "edgetx-models"
	BackupTypeRadioFirmware BackupType = "radio-firmware"
	BackupTypeSDCardPack    BackupType = "sd-card-pack"
	BackupTypeFullBackup    BackupType = "full-backup"
	BackupTypeOther         BackupType = "other"
)

// RadioModel represents a known radio model for selection
type RadioModel struct {
	ID           string            `json:"id"`
	Manufacturer RadioManufacturer `json:"manufacturer"`
	Model        string            `json:"model"`
	DisplayName  string            `json:"displayName"`
}

// GetRadioModels returns the list of known radio models
func GetRadioModels() []RadioModel {
	return []RadioModel{
		{ID: "radiomaster-tx16s-mk2", Manufacturer: ManufacturerRadioMaster, Model: "TX16S Mark II", DisplayName: "RadioMaster TX16S Mark II"},
		{ID: "radiomaster-tx12", Manufacturer: ManufacturerRadioMaster, Model: "TX12", DisplayName: "RadioMaster TX12"},
		{ID: "radiomaster-tx12-mk2", Manufacturer: ManufacturerRadioMaster, Model: "TX12 MKII", DisplayName: "RadioMaster TX12 MKII"},
		{ID: "radiomaster-tx15", Manufacturer: ManufacturerRadioMaster, Model: "TX15", DisplayName: "RadioMaster TX15"},
		{ID: "radiomaster-zorro", Manufacturer: ManufacturerRadioMaster, Model: "Zorro", DisplayName: "RadioMaster Zorro"},
		{ID: "radiomaster-boxer", Manufacturer: ManufacturerRadioMaster, Model: "Boxer", DisplayName: "RadioMaster Boxer"},
		{ID: "radiomaster-pocket", Manufacturer: ManufacturerRadioMaster, Model: "Pocket", DisplayName: "RadioMaster Pocket"},
		// Additional manufacturers can be added here
	}
}

// Radio represents a user's radio transmitter
type Radio struct {
	ID             string            `json:"id"`
	UserID         string            `json:"userId,omitempty"`
	Manufacturer   RadioManufacturer `json:"manufacturer"`
	Model          string            `json:"model"`
	FirmwareFamily FirmwareFamily    `json:"firmwareFamily,omitempty"`
	Notes          string            `json:"notes,omitempty"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
}

// RadioBackup represents a configuration backup for a radio
type RadioBackup struct {
	ID          string     `json:"id"`
	RadioID     string     `json:"radioId"`
	BackupName  string     `json:"backupName"`
	BackupType  BackupType `json:"backupType"`
	FileName    string     `json:"fileName"`
	FileSize    int64      `json:"fileSize"`
	Checksum    string     `json:"checksum,omitempty"`
	StoragePath string     `json:"-"` // Internal storage path, not exposed in JSON
	CreatedAt   time.Time  `json:"createdAt"`
}

// CreateRadioParams defines parameters for creating a radio
type CreateRadioParams struct {
	Manufacturer   RadioManufacturer `json:"manufacturer"`
	Model          string            `json:"model"`
	FirmwareFamily FirmwareFamily    `json:"firmwareFamily,omitempty"`
	Notes          string            `json:"notes,omitempty"`
}

// UpdateRadioParams defines parameters for updating a radio
type UpdateRadioParams struct {
	FirmwareFamily *FirmwareFamily `json:"firmwareFamily,omitempty"`
	Notes          *string         `json:"notes,omitempty"`
}

// CreateRadioBackupParams defines parameters for creating a backup record
type CreateRadioBackupParams struct {
	BackupName string     `json:"backupName"`
	BackupType BackupType `json:"backupType"`
	FileName   string     `json:"fileName"`
	FileSize   int64      `json:"fileSize"`
	Checksum   string     `json:"checksum,omitempty"`
}

// RadioListParams defines parameters for listing radios
type RadioListParams struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// RadioListResponse represents the response for listing radios
type RadioListResponse struct {
	Radios     []Radio `json:"radios"`
	TotalCount int     `json:"totalCount"`
}

// RadioBackupListParams defines parameters for listing backups
type RadioBackupListParams struct {
	Limit  int `json:"limit,omitempty"`
	Offset int `json:"offset,omitempty"`
}

// RadioBackupListResponse represents the response for listing backups
type RadioBackupListResponse struct {
	Backups    []RadioBackup `json:"backups"`
	TotalCount int           `json:"totalCount"`
}

// RadioModelsResponse represents the response for listing available radio models
type RadioModelsResponse struct {
	Models []RadioModel `json:"models"`
}

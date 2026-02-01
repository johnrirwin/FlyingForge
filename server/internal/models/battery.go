package models

import (
	"encoding/json"
	"time"
)

// BatteryChemistry represents the type of battery chemistry
type BatteryChemistry string

const (
	ChemistryLIPO   BatteryChemistry = "LIPO"
	ChemistryLIPOHV BatteryChemistry = "LIPO_HV"
	ChemistryLIION  BatteryChemistry = "LIION"
)

// ValidChemistries returns all valid battery chemistries
func ValidChemistries() []BatteryChemistry {
	return []BatteryChemistry{ChemistryLIPO, ChemistryLIPOHV, ChemistryLIION}
}

// IsValidChemistry checks if a chemistry value is valid
func IsValidChemistry(c BatteryChemistry) bool {
	for _, valid := range ValidChemistries() {
		if c == valid {
			return true
		}
	}
	return false
}

// Battery represents a user's battery
type Battery struct {
	ID           string           `json:"id"`
	UserID       string           `json:"user_id,omitempty"`
	BatteryCode  string           `json:"battery_code"`          // Human-friendly ID like "BAT-A1B2"
	Name         string           `json:"name,omitempty"`        // Optional friendly name
	Chemistry    BatteryChemistry `json:"chemistry"`
	Cells        int              `json:"cells"`                 // 1S through 8S
	CapacityMah  int              `json:"capacity_mah"`
	CRating      *int             `json:"c_rating,omitempty"`
	Connector    string           `json:"connector,omitempty"`   // e.g., XT30, XT60
	WeightGrams  *int             `json:"weight_grams,omitempty"`
	Brand        string           `json:"brand,omitempty"`
	Model        string           `json:"model,omitempty"`
	PurchaseDate *time.Time       `json:"purchase_date,omitempty"`
	Notes        string           `json:"notes,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`

	// Computed fields (populated on detail fetch)
	TotalCycles    int        `json:"total_cycles,omitempty"`
	LastLoggedDate *time.Time `json:"last_logged_date,omitempty"`
}

// BatteryLog represents a health/usage log entry for a battery
type BatteryLog struct {
	ID            string          `json:"id"`
	BatteryID     string          `json:"battery_id"`
	UserID        string          `json:"user_id,omitempty"`
	LoggedAt      time.Time       `json:"log_date"`
	CycleDelta    int             `json:"cycle_count,omitempty"`        // Usually 1, but can be more
	IRMohmPerCell json.RawMessage `json:"ir_milliohms,omitempty"`       // JSON array of IR values per cell
	MinCellV      *float64        `json:"min_cell_v,omitempty"`         // Min cell voltage observed
	MaxCellV      *float64        `json:"max_cell_v,omitempty"`         // Max cell voltage observed
	StorageOk     *bool           `json:"storage_voltage_ok,omitempty"` // Was it stored at storage voltage?
	Notes         string          `json:"notes,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

// CreateBatteryParams defines parameters for creating a battery
type CreateBatteryParams struct {
	Name         string           `json:"name,omitempty"`
	Chemistry    BatteryChemistry `json:"chemistry"`
	Cells        int              `json:"cells"`
	CapacityMah  int              `json:"capacity_mah"`
	CRating      *int             `json:"c_rating,omitempty"`
	Connector    string           `json:"connector,omitempty"`
	WeightGrams  *int             `json:"weight_grams,omitempty"`
	Brand        string           `json:"brand,omitempty"`
	Model        string           `json:"model,omitempty"`
	PurchaseDate *time.Time       `json:"purchase_date,omitempty"`
	Notes        string           `json:"notes,omitempty"`
}

// UpdateBatteryParams defines parameters for updating a battery
type UpdateBatteryParams struct {
	ID           string            `json:"id"`
	Name         *string           `json:"name,omitempty"`
	Chemistry    *BatteryChemistry `json:"chemistry,omitempty"`
	Cells        *int              `json:"cells,omitempty"`
	CapacityMah  *int              `json:"capacity_mah,omitempty"`
	CRating      *int              `json:"c_rating,omitempty"`
	Connector    *string           `json:"connector,omitempty"`
	WeightGrams  *int              `json:"weight_grams,omitempty"`
	Brand        *string           `json:"brand,omitempty"`
	Model        *string           `json:"model,omitempty"`
	PurchaseDate *time.Time        `json:"purchase_date,omitempty"`
	Notes        *string           `json:"notes,omitempty"`
}

// BatteryListParams defines parameters for listing batteries
type BatteryListParams struct {
	Chemistry   BatteryChemistry `json:"chemistry,omitempty"`
	Cells       int              `json:"cells,omitempty"`
	MinCapacity int              `json:"min_capacity,omitempty"`
	MaxCapacity int              `json:"max_capacity,omitempty"`
	Query       string           `json:"query,omitempty"`
	Sort        string           `json:"sort_by,omitempty"` // name, updated, logged, cycles
	SortOrder   string           `json:"sort_order,omitempty"` // ASC, DESC
	Limit       int              `json:"limit,omitempty"`
	Offset      int              `json:"offset,omitempty"`
}

// BatteryListResponse represents the response for listing batteries
type BatteryListResponse struct {
	Batteries  []Battery `json:"batteries"`
	TotalCount int       `json:"total"`
}

// BatteryDetailsResponse includes battery with logs
type BatteryDetailsResponse struct {
	Battery
	Logs []BatteryLog `json:"logs"`
}

// CreateBatteryLogParams defines parameters for creating a log entry
type CreateBatteryLogParams struct {
	BatteryID     string          `json:"battery_id"`
	LoggedAt      *time.Time      `json:"log_date,omitempty"` // Defaults to now
	CycleDelta    int             `json:"cycle_count,omitempty"`
	IRMohmPerCell json.RawMessage `json:"ir_milliohms,omitempty"`
	MinCellV      *float64        `json:"min_cell_v,omitempty"`
	MaxCellV      *float64        `json:"max_cell_v,omitempty"`
	StorageOk     *bool           `json:"storage_voltage_ok,omitempty"`
	Notes         string          `json:"notes,omitempty"`
}

// BatteryLogListResponse represents the response for listing logs
type BatteryLogListResponse struct {
	Logs       []BatteryLog `json:"logs"`
	TotalCount int          `json:"totalCount"`
}

// LabelSize represents the size of a printed label
type LabelSize string

const (
	LabelSizeSmall    LabelSize = "small"
	LabelSizeStandard LabelSize = "standard"
)

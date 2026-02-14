package models

import (
	"encoding/json"
	"time"
)

// InventoryItem represents a piece of equipment in the user's personal inventory
type InventoryItem struct {
	ID           string            `json:"id"`
	UserID       string            `json:"userId,omitempty"`
	Name         string            `json:"name"`
	Category     EquipmentCategory `json:"category"`
	Manufacturer string            `json:"manufacturer,omitempty"`
	Quantity     int               `json:"quantity"`
	Notes        string            `json:"notes,omitempty"`

	// Catalog link - for crowd-sourced gear
	CatalogID   string           `json:"catalogId,omitempty"`
	CatalogItem *GearCatalogItem `json:"catalogItem,omitempty"` // Populated when fetching with catalog data

	// Purchase tracking
	BuildID        string   `json:"buildId,omitempty"`
	PurchasePrice  *float64 `json:"purchasePrice,omitempty"`
	PurchaseSeller string   `json:"purchaseSeller,omitempty"`

	// Links and images
	ProductURL string `json:"productUrl,omitempty"`
	ImageURL   string `json:"imageUrl,omitempty"`

	// Extended data
	Specs json.RawMessage `json:"specs,omitempty"`

	// Source tracking - if added from equipment search
	SourceEquipmentID string `json:"sourceEquipmentId,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AddInventoryParams represents the parameters for adding an inventory item
type AddInventoryParams struct {
	Name              string            `json:"name"`
	Category          EquipmentCategory `json:"category"`
	Manufacturer      string            `json:"manufacturer,omitempty"`
	Quantity          int               `json:"quantity"`
	Notes             string            `json:"notes,omitempty"`
	BuildID           string            `json:"buildId,omitempty"`
	PurchasePrice     *float64          `json:"purchasePrice,omitempty"`
	PurchaseSeller    string            `json:"purchaseSeller,omitempty"`
	ProductURL        string            `json:"productUrl,omitempty"`
	Specs             json.RawMessage   `json:"specs,omitempty"`
	SourceEquipmentID string            `json:"sourceEquipmentId,omitempty"`
	CatalogID         string            `json:"catalogId,omitempty"` // Link to gear catalog item
}

// UpdateInventoryParams represents the parameters for updating an inventory item
type UpdateInventoryParams struct {
	ID             string             `json:"id"`
	Name           *string            `json:"name,omitempty"`
	Category       *EquipmentCategory `json:"category,omitempty"`
	Manufacturer   *string            `json:"manufacturer,omitempty"`
	Quantity       *int               `json:"quantity,omitempty"`
	Notes          *string            `json:"notes,omitempty"`
	BuildID        *string            `json:"buildId,omitempty"`
	PurchasePrice  *float64           `json:"purchasePrice,omitempty"`
	PurchaseSeller *string            `json:"purchaseSeller,omitempty"`
	ProductURL     *string            `json:"productUrl,omitempty"`
	Specs          json.RawMessage    `json:"specs,omitempty"`
}

// InventoryFilterParams defines parameters for filtering inventory
type InventoryFilterParams struct {
	Category EquipmentCategory `json:"category,omitempty"`
	BuildID  string            `json:"buildId,omitempty"`
	Query    string            `json:"query,omitempty"`
	Limit    int               `json:"limit,omitempty"`
	Offset   int               `json:"offset,omitempty"`
}

// InventoryResponse represents the response for inventory queries
type InventoryResponse struct {
	Items      []InventoryItem           `json:"items"`
	TotalCount int                       `json:"totalCount"`
	Categories map[EquipmentCategory]int `json:"categories,omitempty"`
}

// InventorySummary provides a summary of the user's inventory
type InventorySummary struct {
	TotalItems int                       `json:"totalItems"`
	TotalValue float64                   `json:"totalValue"`
	ByCategory map[EquipmentCategory]int `json:"byCategory"`
}

package models

import (
	"encoding/json"
	"time"
)

// EquipmentCategory represents the category of drone equipment
type EquipmentCategory string

const (
	CategoryFrames      EquipmentCategory = "frames"
	CategoryVTX         EquipmentCategory = "vtx"
	CategoryFC          EquipmentCategory = "flight_controllers"
	CategoryESC         EquipmentCategory = "esc"
	CategoryAIO         EquipmentCategory = "aio"
	CategoryStacks      EquipmentCategory = "stacks"
	CategoryMotors      EquipmentCategory = "motors"
	CategoryPropellers  EquipmentCategory = "propellers"
	CategoryReceivers   EquipmentCategory = "receivers"
	CategoryBatteries   EquipmentCategory = "batteries"
	CategoryCameras     EquipmentCategory = "cameras"
	CategoryAntennas    EquipmentCategory = "antennas"
	CategoryGPS         EquipmentCategory = "gps"
	CategoryAccessories EquipmentCategory = "accessories"
)

// AllCategories returns all valid equipment categories
func AllCategories() []EquipmentCategory {
	return []EquipmentCategory{
		CategoryFrames,
		CategoryVTX,
		CategoryFC,
		CategoryESC,
		CategoryAIO,
		CategoryStacks,
		CategoryMotors,
		CategoryPropellers,
		CategoryReceivers,
		CategoryBatteries,
		CategoryCameras,
		CategoryAntennas,
		CategoryGPS,
		CategoryAccessories,
	}
}

// EquipmentItem represents a normalized product from any seller
type EquipmentItem struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Category     EquipmentCategory `json:"category"`
	Manufacturer string            `json:"manufacturer"`
	Price        float64           `json:"price"`
	Currency     string            `json:"currency"`
	Seller       string            `json:"seller"`
	SellerID     string            `json:"sellerId"`
	ProductURL   string            `json:"productUrl"`
	ImageURL     string            `json:"imageUrl"`
	KeySpecs     json.RawMessage   `json:"keySpecs,omitempty"`
	InStock      bool              `json:"inStock"`
	StockQty     *int              `json:"stockQty,omitempty"`
	LastChecked  time.Time         `json:"lastChecked"`
	Description  string            `json:"description,omitempty"`
	SKU          string            `json:"sku,omitempty"`
	Rating       *float64          `json:"rating,omitempty"`
	ReviewCount  *int              `json:"reviewCount,omitempty"`
}

// SellerInfo represents metadata about a seller/retailer
type SellerInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	URL         string   `json:"url"`
	Description string   `json:"description"`
	LogoURL     string   `json:"logoUrl,omitempty"`
	Categories  []string `json:"categories"`
	Enabled     bool     `json:"enabled"`
	Region      string   `json:"region,omitempty"`
}

// EquipmentSearchParams defines parameters for searching equipment
type EquipmentSearchParams struct {
	Query       string            `json:"query,omitempty"`
	Category    EquipmentCategory `json:"category,omitempty"`
	Seller      string            `json:"seller,omitempty"`
	MinPrice    *float64          `json:"minPrice,omitempty"`
	MaxPrice    *float64          `json:"maxPrice,omitempty"`
	InStockOnly bool              `json:"inStockOnly,omitempty"`
	Limit       int               `json:"limit,omitempty"`
	Offset      int               `json:"offset,omitempty"`
	Sort        string            `json:"sort,omitempty"` // "price_asc", "price_desc", "name", "newest"
}

// EquipmentSearchResponse represents the response from equipment search
type EquipmentSearchResponse struct {
	Items      []EquipmentItem `json:"items"`
	TotalCount int             `json:"totalCount"`
	Page       int             `json:"page"`
	PageSize   int             `json:"pageSize"`
	Query      string          `json:"query,omitempty"`
	Filters    struct {
		Category    string    `json:"category,omitempty"`
		Seller      string    `json:"seller,omitempty"`
		PriceRange  []float64 `json:"priceRange,omitempty"`
		InStockOnly bool      `json:"inStockOnly"`
	} `json:"filters"`
}

// SellersResponse represents the list of available sellers
type SellersResponse struct {
	Sellers []SellerInfo `json:"sellers"`
	Count   int          `json:"count"`
}

package sellers

import (
	"context"

	"github.com/johnrirwin/flyingforge/internal/models"
)

// Adapter is the interface that all seller integrations must implement
type Adapter interface {
	// ID returns the unique identifier for this seller
	ID() string

	// Name returns the display name of the seller
	Name() string

	// BaseURL returns the seller's website URL
	BaseURL() string

	// Search searches for equipment matching the query
	Search(ctx context.Context, query string, category models.EquipmentCategory, limit int) ([]models.EquipmentItem, error)

	// GetByCategory returns equipment in a specific category
	GetByCategory(ctx context.Context, category models.EquipmentCategory, limit, offset int) ([]models.EquipmentItem, error)

	// GetProduct returns details for a specific product
	GetProduct(ctx context.Context, productID string) (*models.EquipmentItem, error)

	// SyncProducts syncs the product catalog (background job)
	SyncProducts(ctx context.Context) error
}

// Registry manages seller adapters
type Registry struct {
	adapters map[string]Adapter
}

// NewRegistry creates a new seller registry
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]Adapter),
	}
}

// Register adds a seller adapter to the registry
func (r *Registry) Register(adapter Adapter) {
	r.adapters[adapter.ID()] = adapter
}

// Get returns a seller adapter by ID
func (r *Registry) Get(id string) Adapter {
	return r.adapters[id]
}

// List returns all registered seller adapters
func (r *Registry) List() []Adapter {
	adapters := make([]Adapter, 0, len(r.adapters))
	for _, a := range r.adapters {
		adapters = append(adapters, a)
	}
	return adapters
}

// GetSellerInfo returns seller information for all registered adapters
func (r *Registry) GetSellerInfo() []models.SellerInfo {
	sellers := make([]models.SellerInfo, 0, len(r.adapters))
	for _, a := range r.adapters {
		sellers = append(sellers, models.SellerInfo{
			ID:   a.ID(),
			Name: a.Name(),
			URL:  a.BaseURL(),
		})
	}
	return sellers
}

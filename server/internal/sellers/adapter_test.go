package sellers

import (
	"context"
	"testing"

	"github.com/johnrirwin/flyingforge/internal/models"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	if registry == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	if registry.adapters == nil {
		t.Fatal("NewRegistry() returned registry with nil adapters map")
	}
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()
	adapter := &mockAdapter{id: "test-seller", name: "Test Seller", baseURL: "https://example.com"}

	registry.Register(adapter)

	if got := registry.Get("test-seller"); got == nil {
		t.Error("Register() did not register adapter")
	} else if got.ID() != "test-seller" {
		t.Errorf("Get().ID() = %q, want %q", got.ID(), "test-seller")
	}
}

func TestRegistry_Get_NotFound(t *testing.T) {
	registry := NewRegistry()

	if got := registry.Get("nonexistent"); got != nil {
		t.Error("Get() should return nil for non-existent adapter")
	}
}

func TestRegistry_Get_Found(t *testing.T) {
	registry := NewRegistry()
	adapter := &mockAdapter{id: "getfpv", name: "GetFPV", baseURL: "https://getfpv.com"}
	registry.Register(adapter)

	got := registry.Get("getfpv")
	if got == nil {
		t.Fatal("Get() returned nil for registered adapter")
	}
	if got.Name() != "GetFPV" {
		t.Errorf("Get().Name() = %q, want %q", got.Name(), "GetFPV")
	}
}

func TestRegistry_List_Empty(t *testing.T) {
	registry := NewRegistry()

	adapters := registry.List()
	if adapters == nil {
		t.Error("List() should return empty slice, not nil")
	}
	if len(adapters) != 0 {
		t.Errorf("List() on empty registry = %d adapters, want 0", len(adapters))
	}
}

func TestRegistry_List_Multiple(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockAdapter{id: "seller1", name: "Seller 1", baseURL: "https://seller1.com"})
	registry.Register(&mockAdapter{id: "seller2", name: "Seller 2", baseURL: "https://seller2.com"})
	registry.Register(&mockAdapter{id: "seller3", name: "Seller 3", baseURL: "https://seller3.com"})

	adapters := registry.List()
	if len(adapters) != 3 {
		t.Errorf("List() = %d adapters, want 3", len(adapters))
	}
}

func TestRegistry_GetSellerInfo_Empty(t *testing.T) {
	registry := NewRegistry()

	sellers := registry.GetSellerInfo()
	if sellers == nil {
		t.Error("GetSellerInfo() should return empty slice, not nil")
	}
	if len(sellers) != 0 {
		t.Errorf("GetSellerInfo() on empty registry = %d sellers, want 0", len(sellers))
	}
}

func TestRegistry_GetSellerInfo_Multiple(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&mockAdapter{id: "getfpv", name: "GetFPV", baseURL: "https://getfpv.com"})
	registry.Register(&mockAdapter{id: "rdq", name: "RaceDayQuads", baseURL: "https://racedayquads.com"})

	sellers := registry.GetSellerInfo()
	if len(sellers) != 2 {
		t.Fatalf("GetSellerInfo() = %d sellers, want 2", len(sellers))
	}

	foundGetFPV := false
	foundRDQ := false
	for _, seller := range sellers {
		if seller.ID == "getfpv" {
			foundGetFPV = true
			if seller.Name != "GetFPV" {
				t.Errorf("GetFPV seller name = %q, want %q", seller.Name, "GetFPV")
			}
			if seller.URL != "https://getfpv.com" {
				t.Errorf("GetFPV seller URL = %q, want %q", seller.URL, "https://getfpv.com")
			}
		}
		if seller.ID == "rdq" {
			foundRDQ = true
			if seller.Name != "RaceDayQuads" {
				t.Errorf("RDQ seller name = %q, want %q", seller.Name, "RaceDayQuads")
			}
		}
	}
	if !foundGetFPV {
		t.Error("GetSellerInfo() missing GetFPV seller")
	}
	if !foundRDQ {
		t.Error("GetSellerInfo() missing RaceDayQuads seller")
	}
}

func TestRegistry_Register_OverwritesSameID(t *testing.T) {
	registry := NewRegistry()
	adapter1 := &mockAdapter{id: "test", name: "Original", baseURL: "https://original.com"}
	adapter2 := &mockAdapter{id: "test", name: "Replacement", baseURL: "https://replacement.com"}

	registry.Register(adapter1)
	registry.Register(adapter2)

	got := registry.Get("test")
	if got == nil {
		t.Fatal("Get() returned nil")
	}
	if got.Name() != "Replacement" {
		t.Errorf("Register() should overwrite existing adapter, got name %q, want %q", got.Name(), "Replacement")
	}

	if len(registry.List()) != 1 {
		t.Errorf("List() = %d adapters after overwrite, want 1", len(registry.List()))
	}
}

type mockAdapter struct {
	id      string
	name    string
	baseURL string
}

func (m *mockAdapter) ID() string {
	return m.id
}

func (m *mockAdapter) Name() string {
	return m.name
}

func (m *mockAdapter) BaseURL() string {
	return m.baseURL
}

func (m *mockAdapter) Search(ctx context.Context, query string, category models.EquipmentCategory, limit int) ([]models.EquipmentItem, error) {
	return nil, nil
}

func (m *mockAdapter) GetByCategory(ctx context.Context, category models.EquipmentCategory, limit, offset int) ([]models.EquipmentItem, error) {
	return nil, nil
}

func (m *mockAdapter) GetProduct(ctx context.Context, productID string) (*models.EquipmentItem, error) {
	return nil, nil
}

func (m *mockAdapter) SyncProducts(ctx context.Context) error {
	return nil
}

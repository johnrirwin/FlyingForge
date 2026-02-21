package equipment

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/johnrirwin/flyingforge/internal/models"
)

func TestServiceError(t *testing.T) {
	err := &ServiceError{Message: "test error message"}
	if err.Error() != "test error message" {
		t.Errorf("ServiceError.Error() = %s, want test error message", err.Error())
	}
}

func TestSearch_UsesCatalogData(t *testing.T) {
	now := time.Now().UTC()
	msrp := 89.99

	catalog := &fakeGearCatalog{
		searchResponse: &models.GearCatalogSearchResponse{
			Items: []models.GearCatalogItem{
				{
					ID:       "gear-1",
					GearType: models.GearTypeStack,
					Brand:    "SpeedyBee",
					Model:    "F405 V4",
					MSRP:     &msrp,
					Specs:    json.RawMessage(`{"esc":"55A"}`),
					ShoppingLinks: []string{
						"https://example.com/product/speedybee-f405-v4-stack",
					},
					ImageURL:   "https://example.com/image.jpg",
					UpdatedAt:  now,
					Status:     models.CatalogStatusPublished,
					UsageCount: 5,
				},
			},
		},
	}

	svc := NewService(catalog, nil)
	result, err := svc.Search(context.Background(), models.EquipmentSearchParams{
		Query: "speedybee",
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}

	if len(result.Items) != 1 {
		t.Fatalf("len(result.Items) = %d, want 1", len(result.Items))
	}

	item := result.Items[0]
	if item.SellerID != catalogSellerID {
		t.Fatalf("item.SellerID = %q, want %q", item.SellerID, catalogSellerID)
	}
	if item.Category != models.CategoryStacks {
		t.Fatalf("item.Category = %q, want %q", item.Category, models.CategoryStacks)
	}
	if item.Price != msrp {
		t.Fatalf("item.Price = %v, want %v", item.Price, msrp)
	}
	if item.ProductURL != "https://example.com/product/speedybee-f405-v4-stack" {
		t.Fatalf("item.ProductURL = %q", item.ProductURL)
	}
	if !item.InStock {
		t.Fatalf("item.InStock = %v, want true", item.InStock)
	}
	if item.Name != "SpeedyBee F405 V4" {
		t.Fatalf("item.Name = %q", item.Name)
	}
}

func TestSearch_OffsetBeyondWindowReturnsEmptyWithoutCatalogQuery(t *testing.T) {
	catalog := &fakeGearCatalog{
		searchResponse: &models.GearCatalogSearchResponse{
			Items: []models.GearCatalogItem{
				{ID: "gear-1", Brand: "Brand", Model: "Model", GearType: models.GearTypeOther},
			},
		},
	}

	svc := NewService(catalog, nil)
	result, err := svc.Search(context.Background(), models.EquipmentSearchParams{
		Query:  "anything",
		Limit:  20,
		Offset: 100,
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(result.Items) != 0 || result.TotalCount != 0 {
		t.Fatalf("expected empty result, got %+v", result)
	}
	if catalog.searchCalls != 0 {
		t.Fatalf("catalog.searchCalls = %d, want 0", catalog.searchCalls)
	}
}

func TestSearch_SkipsUnknownPriceWhenFilteringAndSortsUnknownLast(t *testing.T) {
	knownPrice := 120.0
	catalog := &fakeGearCatalog{
		searchResponse: &models.GearCatalogSearchResponse{
			Items: []models.GearCatalogItem{
				{
					ID:       "unknown-price",
					GearType: models.GearTypeStack,
					Brand:    "NoPrice",
					Model:    "Stack",
				},
				{
					ID:       "known-price",
					GearType: models.GearTypeStack,
					Brand:    "Known",
					Model:    "Stack",
					MSRP:     &knownPrice,
				},
			},
		},
	}

	svc := NewService(catalog, nil)

	// Unknown price should be sorted after known price.
	sorted, err := svc.Search(context.Background(), models.EquipmentSearchParams{
		Query: "stack",
		Sort:  "price_asc",
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(sorted.Items) != 2 {
		t.Fatalf("len(sorted.Items) = %d, want 2", len(sorted.Items))
	}
	if sorted.Items[0].ID != "known-price" || sorted.Items[1].ID != "unknown-price" {
		t.Fatalf("unexpected order: %+v", sorted.Items)
	}

	// Unknown price should be excluded when price filters are active.
	min := 100.0
	filtered, err := svc.Search(context.Background(), models.EquipmentSearchParams{
		Query:    "stack",
		MinPrice: &min,
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(filtered.Items) != 1 || filtered.Items[0].ID != "known-price" {
		t.Fatalf("unexpected filtered result: %+v", filtered.Items)
	}
}

func TestSearch_InvalidShoppingLinkFallsBackToCatalogURLAndOutOfStock(t *testing.T) {
	catalog := &fakeGearCatalog{
		searchResponse: &models.GearCatalogSearchResponse{
			Items: []models.GearCatalogItem{
				{
					ID:       "gear-unsafe-link",
					GearType: models.GearTypeOther,
					Brand:    "Unsafe",
					Model:    "Link",
					ShoppingLinks: []string{
						"javascript:alert(1)",
						"data:text/html,<h1>x</h1>",
					},
				},
			},
		},
	}

	svc := NewService(catalog, nil)
	result, err := svc.Search(context.Background(), models.EquipmentSearchParams{
		Query: "unsafe",
	})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("len(result.Items) = %d, want 1", len(result.Items))
	}
	item := result.Items[0]
	if item.ProductURL != "/gear-catalog" {
		t.Fatalf("item.ProductURL = %q, want /gear-catalog", item.ProductURL)
	}
	if item.InStock {
		t.Fatalf("item.InStock = %v, want false", item.InStock)
	}
}

func TestSearch_UnknownSeller(t *testing.T) {
	svc := NewService(&fakeGearCatalog{}, nil)
	_, err := svc.Search(context.Background(), models.EquipmentSearchParams{
		Query:  "stack",
		Seller: "racedayquads",
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if err.Error() != "Unknown seller: racedayquads" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearch_WithoutCatalogReturnsEmpty(t *testing.T) {
	svc := NewService(nil, nil)
	result, err := svc.Search(context.Background(), models.EquipmentSearchParams{})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if result.TotalCount != 0 || len(result.Items) != 0 {
		t.Fatalf("expected empty result, got %+v", result)
	}
}

func TestGetSellers_ReturnsCatalogSource(t *testing.T) {
	svc := NewService(nil, nil)
	sellers := svc.GetSellers()
	if len(sellers) != 1 {
		t.Fatalf("len(sellers) = %d, want 1", len(sellers))
	}
	if sellers[0].ID != catalogSellerID {
		t.Fatalf("sellers[0].ID = %q, want %q", sellers[0].ID, catalogSellerID)
	}
	if sellers[0].Name != catalogSellerName {
		t.Fatalf("sellers[0].Name = %q, want %q", sellers[0].Name, catalogSellerName)
	}
}

func TestGetProduct_LoadsCatalogItem(t *testing.T) {
	catalog := &fakeGearCatalog{
		getByID: map[string]*models.GearCatalogItem{
			"abc123": {
				ID:        "abc123",
				GearType:  models.GearTypeAIO,
				Brand:     "Brand",
				Model:     "Model",
				UpdatedAt: time.Now().UTC(),
			},
		},
	}
	svc := NewService(catalog, nil)

	item, err := svc.GetProduct(context.Background(), "catalog-abc123")
	if err != nil {
		t.Fatalf("GetProduct returned error: %v", err)
	}
	if item == nil {
		t.Fatalf("expected item")
	}
	if item.ID != "abc123" {
		t.Fatalf("item.ID = %q, want abc123", item.ID)
	}
	if item.Category != models.CategoryAIO {
		t.Fatalf("item.Category = %q, want %q", item.Category, models.CategoryAIO)
	}
}

type fakeGearCatalog struct {
	searchResponse *models.GearCatalogSearchResponse
	popularItems   []models.GearCatalogItem
	getByID        map[string]*models.GearCatalogItem
	searchErr      error
	popularErr     error
	getErr         error
	searchCalls    int
}

func (f *fakeGearCatalog) Search(ctx context.Context, params models.GearCatalogSearchParams) (*models.GearCatalogSearchResponse, error) {
	f.searchCalls++
	if f.searchErr != nil {
		return nil, f.searchErr
	}
	if f.searchResponse == nil {
		return &models.GearCatalogSearchResponse{}, nil
	}
	return f.searchResponse, nil
}

func (f *fakeGearCatalog) GetPopular(ctx context.Context, gearType models.GearType, limit int) ([]models.GearCatalogItem, error) {
	if f.popularErr != nil {
		return nil, f.popularErr
	}
	return f.popularItems, nil
}

func (f *fakeGearCatalog) Get(ctx context.Context, id string) (*models.GearCatalogItem, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.getByID == nil {
		return nil, nil
	}
	return f.getByID[id], nil
}

func TestSearch_HandlesCatalogFailure(t *testing.T) {
	svc := NewService(&fakeGearCatalog{searchErr: errors.New("db down")}, nil)
	_, err := svc.Search(context.Background(), models.EquipmentSearchParams{Query: "f7"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if err.Error() != "search failed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

package inventory

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/johnrirwin/flyingforge/internal/models"
	"github.com/johnrirwin/flyingforge/internal/testutil"
)

func TestInMemoryServiceGetSummary_UsesQuantities(t *testing.T) {
	svc := NewInMemoryService(testutil.NullLogger())

	if _, err := svc.AddItem(context.Background(), "user-1", models.AddInventoryParams{
		Name:          "Frame A",
		Category:      models.CategoryFrames,
		Quantity:      2,
		PurchasePrice: floatPtr(90),
	}); err != nil {
		t.Fatalf("AddItem() error = %v", err)
	}

	if _, err := svc.AddItem(context.Background(), "user-1", models.AddInventoryParams{
		Name:          "Frame B",
		Category:      models.CategoryFrames,
		Quantity:      1,
		PurchasePrice: floatPtr(100),
		Specs: json.RawMessage(`{
			"__ff_inventory_item_details": [
				{"purchasePrice": 100}
			]
		}`),
	}); err != nil {
		t.Fatalf("AddItem() error = %v", err)
	}

	summary, err := svc.GetSummary(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("GetSummary() error = %v", err)
	}

	if summary.TotalItems != 3 {
		t.Fatalf("summary.TotalItems = %d, want 3", summary.TotalItems)
	}

	if got := summary.ByCategory[models.CategoryFrames]; got != 3 {
		t.Fatalf("summary.ByCategory[frames] = %d, want 3", got)
	}

	if summary.TotalValue != 280 {
		t.Fatalf("summary.TotalValue = %v, want 280", summary.TotalValue)
	}
}

func floatPtr(v float64) *float64 {
	return &v
}

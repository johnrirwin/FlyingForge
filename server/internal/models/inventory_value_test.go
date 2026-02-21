package models

import (
	"encoding/json"
	"testing"
)

func TestCalculateInventoryItemTotalValue(t *testing.T) {
	t.Run("uses top-level purchase price when detail specs are absent", func(t *testing.T) {
		price := 25.0

		got := CalculateInventoryItemTotalValue(3, &price, json.RawMessage(`{"kv":"1950"}`))
		want := 75.0

		if got != want {
			t.Fatalf("CalculateInventoryItemTotalValue() = %v, want %v", got, want)
		}
	})

	t.Run("uses per-item detail prices when present", func(t *testing.T) {
		price := 25.0
		specs := json.RawMessage(`{
			"__ff_inventory_item_details": [
				{"purchasePrice": 20},
				{"purchasePrice": 30}
			]
		}`)

		got := CalculateInventoryItemTotalValue(2, &price, specs)
		want := 50.0

		if got != want {
			t.Fatalf("CalculateInventoryItemTotalValue() = %v, want %v", got, want)
		}
	})

	t.Run("falls back for missing detail entries", func(t *testing.T) {
		price := 40.0
		specs := json.RawMessage(`{
			"__ff_inventory_item_details": [
				{"purchasePrice": 35}
			]
		}`)

		got := CalculateInventoryItemTotalValue(2, &price, specs)
		want := 75.0 // 35 + fallback 40

		if got != want {
			t.Fatalf("CalculateInventoryItemTotalValue() = %v, want %v", got, want)
		}
	})

	t.Run("ignores missing detail prices when detail rows exist", func(t *testing.T) {
		price := 40.0
		specs := json.RawMessage(`{
			"__ff_inventory_item_details": [
				{"purchasePrice": 35},
				{}
			]
		}`)

		got := CalculateInventoryItemTotalValue(2, &price, specs)
		want := 35.0

		if got != want {
			t.Fatalf("CalculateInventoryItemTotalValue() = %v, want %v", got, want)
		}
	})
}

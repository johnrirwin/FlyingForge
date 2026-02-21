package models

import (
	"encoding/json"
	"strings"
)

// InventoryItemDetailsSpecKey is the specs key that stores per-item purchase/build metadata.
const InventoryItemDetailsSpecKey = "__ff_inventory_item_details"

type inventoryItemDetailSpec struct {
	PurchasePrice *float64 `json:"purchasePrice"`
}

// CalculateInventoryItemTotalValue returns the total known value for an inventory row.
//
// Behavior:
//   - When per-item detail specs exist, sum known per-item purchasePrice values.
//   - If per-item detail count is lower than quantity, fallback to top-level purchasePrice
//     for the remaining quantity.
//   - Without per-item detail specs, use top-level purchasePrice * quantity.
func CalculateInventoryItemTotalValue(quantity int, purchasePrice *float64, specs json.RawMessage) float64 {
	if quantity <= 0 {
		return 0
	}

	details, hasDetails := extractInventoryItemDetailSpecs(specs)
	if hasDetails {
		total := 0.0
		for _, detail := range details {
			if detail.PurchasePrice == nil || *detail.PurchasePrice < 0 {
				continue
			}
			total += *detail.PurchasePrice
		}

		if len(details) < quantity && purchasePrice != nil && *purchasePrice >= 0 {
			total += *purchasePrice * float64(quantity-len(details))
		}

		return total
	}

	if purchasePrice == nil || *purchasePrice < 0 {
		return 0
	}

	return *purchasePrice * float64(quantity)
}

func extractInventoryItemDetailSpecs(specs json.RawMessage) ([]inventoryItemDetailSpec, bool) {
	trimmed := strings.TrimSpace(string(specs))
	if trimmed == "" || trimmed == "null" {
		return nil, false
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(specs, &payload); err != nil {
		return nil, false
	}

	rawDetails, ok := payload[InventoryItemDetailsSpecKey]
	if !ok {
		return nil, false
	}

	var details []inventoryItemDetailSpec
	if err := json.Unmarshal(rawDetails, &details); err != nil {
		return nil, false
	}

	return details, true
}

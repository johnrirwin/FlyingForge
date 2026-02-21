package models

import (
	"testing"
)

func TestEquipmentCategory_Values(t *testing.T) {
	tests := []struct {
		name     string
		value    EquipmentCategory
		expected string
	}{
		{"frames", CategoryFrames, "frames"},
		{"vtx", CategoryVTX, "vtx"},
		{"flight_controllers", CategoryFC, "flight_controllers"},
		{"esc", CategoryESC, "esc"},
		{"aio", CategoryAIO, "aio"},
		{"stacks", CategoryStacks, "stacks"},
		{"motors", CategoryMotors, "motors"},
		{"propellers", CategoryPropellers, "propellers"},
		{"receivers", CategoryReceivers, "receivers"},
		{"batteries", CategoryBatteries, "batteries"},
		{"cameras", CategoryCameras, "cameras"},
		{"antennas", CategoryAntennas, "antennas"},
		{"gps", CategoryGPS, "gps"},
		{"accessories", CategoryAccessories, "accessories"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.value) != tt.expected {
				t.Errorf("EquipmentCategory %s = %s, want %s", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestAllCategories(t *testing.T) {
	categories := AllCategories()

	// Should have 14 categories
	if len(categories) != 14 {
		t.Errorf("AllCategories() returned %d categories, want 14", len(categories))
	}

	// All categories should be non-empty
	for i, cat := range categories {
		if cat == "" {
			t.Errorf("AllCategories()[%d] is empty", i)
		}
	}

	// Check specific categories are present
	expected := map[EquipmentCategory]bool{
		CategoryFrames:      false,
		CategoryVTX:         false,
		CategoryFC:          false,
		CategoryESC:         false,
		CategoryAIO:         false,
		CategoryStacks:      false,
		CategoryMotors:      false,
		CategoryPropellers:  false,
		CategoryReceivers:   false,
		CategoryBatteries:   false,
		CategoryCameras:     false,
		CategoryAntennas:    false,
		CategoryGPS:         false,
		CategoryAccessories: false,
	}

	for _, cat := range categories {
		if _, ok := expected[cat]; ok {
			expected[cat] = true
		}
	}

	for cat, found := range expected {
		if !found {
			t.Errorf("AllCategories() missing %s", cat)
		}
	}
}

func TestEquipmentSearchParams_Defaults(t *testing.T) {
	params := EquipmentSearchParams{}

	// Default values
	if params.Query != "" {
		t.Errorf("Default Query should be empty, got %s", params.Query)
	}
	if params.Category != "" {
		t.Errorf("Default Category should be empty, got %s", params.Category)
	}
	if params.Limit != 0 {
		t.Errorf("Default Limit should be 0, got %d", params.Limit)
	}
	if params.InStockOnly != false {
		t.Error("Default InStockOnly should be false")
	}
}

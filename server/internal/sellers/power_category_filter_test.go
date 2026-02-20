package sellers

import (
	"testing"

	"github.com/johnrirwin/flyingforge/internal/models"
)

func TestMatchesPowerCategoryIntent(t *testing.T) {
	tests := []struct {
		name     string
		itemName string
		category models.EquipmentCategory
		want     bool
	}{
		{
			name:     "aio token matches aio category",
			itemName: "SpeedyBee F405 AIO 40A",
			category: models.CategoryAIO,
			want:     true,
		},
		{
			name:     "all in one phrase matches aio category",
			itemName: "All In One F4 Controller 35A",
			category: models.CategoryAIO,
			want:     true,
		},
		{
			name:     "stack token does not match aio category",
			itemName: "Diatone Mamba F722 Stack",
			category: models.CategoryAIO,
			want:     false,
		},
		{
			name:     "stack token matches stacks category",
			itemName: "Diatone Mamba F722 Stack",
			category: models.CategoryStacks,
			want:     true,
		},
		{
			name:     "aio token does not match stacks category",
			itemName: "SpeedyBee F405 AIO 40A",
			category: models.CategoryStacks,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesPowerCategoryIntent(tt.itemName, tt.category)
			if got != tt.want {
				t.Fatalf("matchesPowerCategoryIntent(%q, %q) = %v, want %v", tt.itemName, tt.category, got, tt.want)
			}
		})
	}
}

func TestFilterItemsByPowerCategoryIntent(t *testing.T) {
	items := []models.EquipmentItem{
		{Name: "SpeedyBee F405 AIO 40A"},
		{Name: "Diatone Mamba F722 Stack"},
		{Name: "General FC Board"},
	}

	aioFiltered := filterItemsByPowerCategoryIntent(items, models.CategoryAIO)
	if len(aioFiltered) != 1 || aioFiltered[0].Name != "SpeedyBee F405 AIO 40A" {
		t.Fatalf("unexpected aio filter result: %+v", aioFiltered)
	}

	stackFiltered := filterItemsByPowerCategoryIntent(items, models.CategoryStacks)
	if len(stackFiltered) != 1 || stackFiltered[0].Name != "Diatone Mamba F722 Stack" {
		t.Fatalf("unexpected stack filter result: %+v", stackFiltered)
	}
}

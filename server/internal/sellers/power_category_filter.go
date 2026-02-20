package sellers

import (
	"strings"

	"github.com/johnrirwin/flyingforge/internal/models"
)

var powerCategoryNameCleaner = strings.NewReplacer(
	"-", " ",
	"_", " ",
	"/", " ",
	"\\", " ",
	".", " ",
	",", " ",
	":", " ",
	";", " ",
	"(", " ",
	")", " ",
	"[", " ",
	"]", " ",
	"{", " ",
	"}", " ",
	"\"", " ",
	"'", " ",
	"&", " ",
	"+", " ",
)

func filterItemsByPowerCategoryIntent(items []models.EquipmentItem, category models.EquipmentCategory) []models.EquipmentItem {
	if category != models.CategoryAIO && category != models.CategoryStacks {
		return items
	}

	filtered := make([]models.EquipmentItem, 0, len(items))
	for _, item := range items {
		if matchesPowerCategoryIntent(item.Name, category) {
			filtered = append(filtered, item)
		}
	}

	return filtered
}

func matchesPowerCategoryIntent(name string, category models.EquipmentCategory) bool {
	tokens, normalized := tokenizePowerCategoryName(name)
	if len(tokens) == 0 {
		return false
	}

	has := func(target string) bool {
		for _, token := range tokens {
			if token == target {
				return true
			}
		}
		return false
	}

	switch category {
	case models.CategoryAIO:
		return has("aio") || strings.Contains(normalized, "all in one")
	case models.CategoryStacks:
		return has("stack") || has("stacks")
	default:
		return true
	}
}

func tokenizePowerCategoryName(name string) ([]string, string) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return nil, ""
	}

	normalized = powerCategoryNameCleaner.Replace(normalized)
	normalized = strings.Join(strings.Fields(normalized), " ")
	if normalized == "" {
		return nil, ""
	}

	return strings.Fields(normalized), normalized
}

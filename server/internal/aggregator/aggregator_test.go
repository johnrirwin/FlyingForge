package aggregator

import (
	"sort"
	"testing"
	"time"

	"github.com/johnrirwin/flyingforge/internal/models"
	"github.com/johnrirwin/flyingforge/internal/sources"
)

func TestSortByDate(t *testing.T) {
	now := time.Now()
	items := []models.FeedItem{
		{ID: "1", Title: "Old", PublishedAt: now.Add(-2 * time.Hour)},
		{ID: "2", Title: "New", PublishedAt: now},
		{ID: "3", Title: "Middle", PublishedAt: now.Add(-1 * time.Hour)},
	}

	sortByDate(items)

	if items[0].Title != "New" {
		t.Errorf("First item should be 'New', got %q", items[0].Title)
	}
	if items[1].Title != "Middle" {
		t.Errorf("Second item should be 'Middle', got %q", items[1].Title)
	}
	if items[2].Title != "Old" {
		t.Errorf("Third item should be 'Old', got %q", items[2].Title)
	}
}

func TestSortByDate_Empty(t *testing.T) {
	items := []models.FeedItem{}
	sortByDate(items)
}

func TestSortByDate_SingleItem(t *testing.T) {
	items := []models.FeedItem{
		{ID: "1", Title: "Only"},
	}
	sortByDate(items)
	if items[0].Title != "Only" {
		t.Error("Single item should remain unchanged")
	}
}

func TestContainsTag(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		target   string
		expected bool
	}{
		{
			name:     "exact match",
			tags:     []string{"DJI", "FPV", "Racing"},
			target:   "FPV",
			expected: true,
		},
		{
			name:     "case insensitive match",
			tags:     []string{"DJI", "FPV", "Racing"},
			target:   "fpv",
			expected: true,
		},
		{
			name:     "not found",
			tags:     []string{"DJI", "FPV", "Racing"},
			target:   "Photography",
			expected: false,
		},
		{
			name:     "empty tags",
			tags:     []string{},
			target:   "FPV",
			expected: false,
		},
		{
			name:     "empty target",
			tags:     []string{"DJI", "FPV"},
			target:   "",
			expected: false,
		},
		{
			name:     "nil tags",
			tags:     nil,
			target:   "FPV",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := containsTag(tt.tags, tt.target)
			if got != tt.expected {
				t.Errorf("containsTag(%v, %q) = %v, want %v", tt.tags, tt.target, got, tt.expected)
			}
		})
	}
}

func TestMergeTags(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		inferred []string
		expected []string
	}{
		{
			name:     "no duplicates",
			existing: []string{"DJI", "News"},
			inferred: []string{"FPV", "Review"},
			expected: []string{"DJI", "News", "FPV", "Review"},
		},
		{
			name:     "with duplicates",
			existing: []string{"DJI", "FPV"},
			inferred: []string{"FPV", "Review"},
			expected: []string{"DJI", "FPV", "Review"},
		},
		{
			name:     "case insensitive duplicates",
			existing: []string{"DJI", "fpv"},
			inferred: []string{"FPV", "Review"},
			expected: []string{"DJI", "fpv", "Review"},
		},
		{
			name:     "empty existing",
			existing: []string{},
			inferred: []string{"FPV", "Review"},
			expected: []string{"FPV", "Review"},
		},
		{
			name:     "empty inferred",
			existing: []string{"DJI", "News"},
			inferred: []string{},
			expected: []string{"DJI", "News"},
		},
		{
			name:     "both empty",
			existing: []string{},
			inferred: []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeTags(tt.existing, tt.inferred)

			if len(got) != len(tt.expected) {
				t.Errorf("mergeTags() length = %d, want %d", len(got), len(tt.expected))
				return
			}

			sort.Strings(got)
			sort.Strings(tt.expected)

			for i, tag := range got {
				if tag != tt.expected[i] {
					t.Errorf("mergeTags()[%d] = %q, want %q", i, tag, tt.expected[i])
				}
			}
		})
	}
}

func TestMergeTags_PreservesOriginalCase(t *testing.T) {
	existing := []string{"DJI"}
	inferred := []string{"dji"}

	got := mergeTags(existing, inferred)

	if len(got) != 1 {
		t.Fatalf("mergeTags() should have 1 tag, got %d: %v", len(got), got)
	}
	if got[0] != "DJI" {
		t.Errorf("mergeTags() should preserve original case %q, got %q", "DJI", got[0])
	}
}

func TestDeduplicate(t *testing.T) {
	a := &Aggregator{}

	tests := []struct {
		name     string
		items    []models.FeedItem
		expected int
	}{
		{
			name: "no duplicates",
			items: []models.FeedItem{
				{ID: "1", Title: "Article One"},
				{ID: "2", Title: "Article Two"},
				{ID: "3", Title: "Article Three"},
			},
			expected: 3,
		},
		{
			name: "duplicate IDs",
			items: []models.FeedItem{
				{ID: "1", Title: "Article One"},
				{ID: "1", Title: "Article One Copy"},
				{ID: "2", Title: "Article Two"},
			},
			expected: 2,
		},
		{
			name: "duplicate titles",
			items: []models.FeedItem{
				{ID: "1", Title: "Same Title"},
				{ID: "2", Title: "Same Title"},
				{ID: "3", Title: "Different Title"},
			},
			expected: 2,
		},
		{
			name: "case insensitive title dedup",
			items: []models.FeedItem{
				{ID: "1", Title: "Same Title"},
				{ID: "2", Title: "SAME TITLE"},
				{ID: "3", Title: "  Same Title  "},
			},
			expected: 1,
		},
		{
			name:     "empty list",
			items:    []models.FeedItem{},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := a.deduplicate(tt.items)
			if len(got) != tt.expected {
				t.Errorf("deduplicate() returned %d items, want %d", len(got), tt.expected)
			}
		})
	}
}

func TestDeduplicate_PreservesOrder(t *testing.T) {
	a := &Aggregator{}

	items := []models.FeedItem{
		{ID: "1", Title: "First"},
		{ID: "2", Title: "Second"},
		{ID: "1", Title: "First Duplicate"},
		{ID: "3", Title: "Third"},
	}

	got := a.deduplicate(items)

	if len(got) != 3 {
		t.Fatalf("deduplicate() returned %d items, want 3", len(got))
	}
	if got[0].Title != "First" {
		t.Errorf("First item should be 'First', got %q", got[0].Title)
	}
	if got[1].Title != "Second" {
		t.Errorf("Second item should be 'Second', got %q", got[1].Title)
	}
	if got[2].Title != "Third" {
		t.Errorf("Third item should be 'Third', got %q", got[2].Title)
	}
}

func TestAggregator_GetSources_Empty(t *testing.T) {
	a := &Aggregator{
		fetchers: []sources.Fetcher{},
	}

	srcs := a.GetSources()
	if srcs == nil {
		t.Error("GetSources() should return empty slice, not nil")
	}
	if len(srcs) != 0 {
		t.Errorf("GetSources() on empty aggregator = %d sources, want 0", len(srcs))
	}
}

func TestAggregator_GetItems_Empty(t *testing.T) {
	a := &Aggregator{
		items:    []models.FeedItem{},
		fetchers: []sources.Fetcher{},
	}

	resp := a.GetItems(models.FilterParams{})

	if resp.Items == nil {
		t.Error("GetItems().Items should not be nil")
	}
	if len(resp.Items) != 0 {
		t.Errorf("GetItems() on empty = %d items, want 0", len(resp.Items))
	}
	if resp.TotalCount != 0 {
		t.Errorf("GetItems().TotalCount = %d, want 0", resp.TotalCount)
	}
}

func TestAggregator_GetItems_Pagination(t *testing.T) {
	items := make([]models.FeedItem, 100)
	for i := 0; i < 100; i++ {
		items[i] = models.FeedItem{
			ID:          string(rune('a' + i%26)),
			Title:       "Article",
			PublishedAt: time.Now().Add(-time.Duration(i) * time.Hour),
		}
	}

	a := &Aggregator{
		items:    items,
		fetchers: []sources.Fetcher{},
	}

	tests := []struct {
		name          string
		params        models.FilterParams
		expectedCount int
		expectedTotal int
	}{
		{
			name:          "first page",
			params:        models.FilterParams{Limit: 10, Offset: 0},
			expectedCount: 10,
			expectedTotal: 100,
		},
		{
			name:          "middle page",
			params:        models.FilterParams{Limit: 10, Offset: 50},
			expectedCount: 10,
			expectedTotal: 100,
		},
		{
			name:          "last page partial",
			params:        models.FilterParams{Limit: 10, Offset: 95},
			expectedCount: 5,
			expectedTotal: 100,
		},
		{
			name:          "offset beyond end",
			params:        models.FilterParams{Limit: 10, Offset: 200},
			expectedCount: 0,
			expectedTotal: 100,
		},
		{
			name:          "no limit returns all",
			params:        models.FilterParams{Limit: 0, Offset: 0},
			expectedCount: 100,
			expectedTotal: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := a.GetItems(tt.params)
			if len(resp.Items) != tt.expectedCount {
				t.Errorf("GetItems() returned %d items, want %d", len(resp.Items), tt.expectedCount)
			}
			if resp.TotalCount != tt.expectedTotal {
				t.Errorf("GetItems().TotalCount = %d, want %d", resp.TotalCount, tt.expectedTotal)
			}
		})
	}
}

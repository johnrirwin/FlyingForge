package tagging

import (
	"sort"
	"testing"
)

func TestNew(t *testing.T) {
	tagger := New()
	if tagger == nil {
		t.Fatal("New() returned nil")
	}
	if tagger.rules == nil {
		t.Fatal("New() returned tagger with nil rules")
	}
	if len(tagger.rules) == 0 {
		t.Fatal("New() returned tagger with empty rules")
	}
}

func TestInferTags_SingleMatch(t *testing.T) {
	tagger := New()

	tests := []struct {
		name        string
		title       string
		content     string
		expectedTag string
	}{
		{
			name:        "DJI in title",
			title:       "New DJI Mavic 4 Announced",
			content:     "",
			expectedTag: "DJI",
		},
		{
			name:        "FPV in content",
			title:       "Building My First Drone",
			content:     "I decided to try FPV flying",
			expectedTag: "FPV",
		},
		{
			name:        "FAA keyword",
			title:       "New Part 107 Regulations Released",
			content:     "",
			expectedTag: "FAA",
		},
		{
			name:        "Racing keyword",
			title:       "Join the local drone racing league",
			content:     "",
			expectedTag: "Racing",
		},
		{
			name:        "Tutorial keyword",
			title:       "How to build a racing quad",
			content:     "",
			expectedTag: "Tutorial",
		},
		{
			name:        "Review keyword",
			title:       "Hands-on review of new goggles",
			content:     "",
			expectedTag: "Review",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags := tagger.InferTags(tt.title, tt.content)
			found := false
			for _, tag := range tags {
				if tag == tt.expectedTag {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("InferTags() did not return expected tag %q, got: %v", tt.expectedTag, tags)
			}
		})
	}
}

func TestInferTags_MultipleMatches(t *testing.T) {
	tagger := New()

	tags := tagger.InferTags("DJI Mavic 3 Review: Best Photography Drone?", "This FPV camera drone is great for video footage")

	expectedTags := []string{"DJI", "Review", "Photography", "FPV", "Videography"}
	for _, expected := range expectedTags {
		found := false
		for _, tag := range tags {
			if tag == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected tag %q not found in %v", expected, tags)
		}
	}
}

func TestInferTags_NoMatches(t *testing.T) {
	tagger := New()

	tags := tagger.InferTags("Random unrelated title", "Some content about cooking recipes")

	if len(tags) != 0 {
		t.Errorf("InferTags() should return empty slice for non-matching content, got: %v", tags)
	}
}

func TestInferTags_CaseInsensitive(t *testing.T) {
	tagger := New()

	tests := []struct {
		title       string
		expectedTag string
	}{
		{"DJI news", "DJI"},
		{"dji news", "DJI"},
		{"FPV racing", "FPV"},
		{"fpv racing", "FPV"},
	}

	for _, tt := range tests {
		tags := tagger.InferTags(tt.title, "")
		found := false
		for _, tag := range tags {
			if tag == tt.expectedTag {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("InferTags(%q) should match %q case-insensitively, got: %v", tt.title, tt.expectedTag, tags)
		}
	}
}

func TestInferTags_EmptyInput(t *testing.T) {
	tagger := New()

	tags := tagger.InferTags("", "")
	if tags == nil {
		t.Error("InferTags() should return empty slice, not nil")
	}
	if len(tags) != 0 {
		t.Errorf("InferTags() on empty input should return empty slice, got: %v", tags)
	}
}

func TestAddRule(t *testing.T) {
	tagger := New()

	tagger.AddRule("Custom", []string{"custom", "special"})

	tags := tagger.InferTags("This is a custom drone", "")
	found := false
	for _, tag := range tags {
		if tag == "Custom" {
			found = true
			break
		}
	}
	if !found {
		t.Error("AddRule() did not add custom rule properly")
	}
}

func TestRemoveRule(t *testing.T) {
	tagger := New()

	// Verify DJI tag works before removal
	tags := tagger.InferTags("New DJI drone", "")
	hasDJI := false
	for _, tag := range tags {
		if tag == "DJI" {
			hasDJI = true
			break
		}
	}
	if !hasDJI {
		t.Fatal("Expected DJI tag before removal")
	}

	// Remove the rule
	tagger.RemoveRule("DJI")

	// Verify DJI tag no longer works
	tags = tagger.InferTags("New DJI drone", "")
	for _, tag := range tags {
		if tag == "DJI" {
			t.Error("DJI tag should not be inferred after RemoveRule()")
		}
	}
}

func TestGetRules(t *testing.T) {
	tagger := New()

	rules := tagger.GetRules()

	// Verify rules is a copy (modifications don't affect original)
	rules["Test"] = []string{"test"}

	originalRules := tagger.GetRules()
	if _, exists := originalRules["Test"]; exists {
		t.Error("GetRules() should return a copy, not the original map")
	}

	// Verify some expected rules exist
	expectedRules := []string{"DJI", "FPV", "FAA", "Racing"}
	for _, rule := range expectedRules {
		if _, exists := originalRules[rule]; !exists {
			t.Errorf("Expected rule %q not found in GetRules()", rule)
		}
	}
}

func TestDefaultRulesContent(t *testing.T) {
	tagger := New()
	rules := tagger.GetRules()

	// Test that DJI rule contains expected keywords
	djiKeywords := rules["DJI"]
	expectedDJIKeywords := []string{"dji", "mavic", "phantom", "mini", "avata"}
	for _, expected := range expectedDJIKeywords {
		found := false
		for _, kw := range djiKeywords {
			if kw == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("DJI rule missing expected keyword %q", expected)
		}
	}

	// Test that FPV rule contains expected keywords
	fpvKeywords := rules["FPV"]
	expectedFPVKeywords := []string{"fpv", "goggles", "betaflight", "freestyle"}
	for _, expected := range expectedFPVKeywords {
		found := false
		for _, kw := range fpvKeywords {
			if kw == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("FPV rule missing expected keyword %q", expected)
		}
	}
}

func TestInferTags_OnlyMatchesOnce(t *testing.T) {
	tagger := New()

	// Content with multiple DJI keywords should only produce one DJI tag
	tags := tagger.InferTags("DJI Mavic, DJI Mini, DJI Air", "phantom avata inspire")

	djiCount := 0
	for _, tag := range tags {
		if tag == "DJI" {
			djiCount++
		}
	}
	if djiCount != 1 {
		t.Errorf("Expected exactly 1 DJI tag, got %d", djiCount)
	}
}

func TestInferTags_UniqueResults(t *testing.T) {
	tagger := New()

	tags := tagger.InferTags("FPV FPV FPV drone racing racing", "fpv racing")

	// Check for duplicates
	seen := make(map[string]bool)
	for _, tag := range tags {
		if seen[tag] {
			t.Errorf("InferTags() returned duplicate tag: %s", tag)
		}
		seen[tag] = true
	}
}

func TestInferTags_ResultsAreDeterministic(t *testing.T) {
	tagger := New()
	title := "DJI FPV Racing Drone Review for Commercial Use"
	content := "Agricultural mapping with autonomous waypoint flight"

	// Run multiple times and ensure consistent tags (sorted)
	var firstResult []string
	for i := 0; i < 5; i++ {
		tags := tagger.InferTags(title, content)
		sort.Strings(tags)
		if i == 0 {
			firstResult = tags
		} else {
			if len(tags) != len(firstResult) {
				t.Errorf("Iteration %d: tag count mismatch, expected %d, got %d", i, len(firstResult), len(tags))
			}
			for j, tag := range tags {
				if j < len(firstResult) && tag != firstResult[j] {
					t.Errorf("Iteration %d: tag mismatch at index %d, expected %q, got %q", i, j, firstResult[j], tag)
				}
			}
		}
	}
}

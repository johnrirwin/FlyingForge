package sources

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/johnrirwin/flyingforge/internal/ratelimit"
)

func TestLoadFeedsConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "feeds.json")

	configContent := `{
		"sources": [
			{"name": "Test RSS", "url": "https://example.com/feed", "type": "rss", "category": "news", "enabled": true},
			{"name": "Test YouTube", "url": "https://www.youtube.com/feeds/videos.xml?channel_id=UC123", "type": "youtube", "category": "creator", "enabled": true},
			{"name": "r/test", "url": "https://www.reddit.com/r/test/.rss", "type": "reddit", "category": "community", "enabled": false}
		]
	}`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	config, err := LoadFeedsConfig(configPath)
	if err != nil {
		t.Fatalf("LoadFeedsConfig() error = %v", err)
	}

	if len(config.Sources) != 3 {
		t.Errorf("Expected 3 sources, got %d", len(config.Sources))
	}

	// Verify first source
	if config.Sources[0].Name != "Test RSS" {
		t.Errorf("Expected 'Test RSS', got '%s'", config.Sources[0].Name)
	}
	if config.Sources[0].Type != "rss" {
		t.Errorf("Expected type 'rss', got '%s'", config.Sources[0].Type)
	}

	// Verify YouTube source
	if config.Sources[1].Type != "youtube" {
		t.Errorf("Expected type 'youtube', got '%s'", config.Sources[1].Type)
	}

	// Verify disabled source
	if config.Sources[2].Enabled != false {
		t.Errorf("Expected enabled=false for r/test")
	}
}

func TestCreateFetchersFromConfig(t *testing.T) {
	config := &FeedsConfig{
		Sources: []FeedSource{
			{Name: "Test RSS", URL: "https://example.com/feed", Type: "rss", Category: "news", Enabled: true},
			{Name: "Test YouTube", URL: "https://www.youtube.com/feeds/videos.xml?channel_id=UC123", Type: "youtube", Category: "creator", Enabled: true},
			{Name: "r/test", URL: "https://www.reddit.com/r/test/.rss", Type: "reddit", Category: "community", Enabled: true},
			{Name: "Disabled", URL: "https://example.com/disabled", Type: "rss", Category: "news", Enabled: false},
		},
	}

	limiter := ratelimit.New(time.Second)
	fetcherConfig := DefaultConfig()

	fetchers := CreateFetchersFromConfig(config, limiter, fetcherConfig)

	// Should have 3 fetchers (disabled one should be skipped)
	if len(fetchers) != 3 {
		t.Errorf("Expected 3 fetchers, got %d", len(fetchers))
	}

	// Verify fetcher names
	names := make(map[string]bool)
	for _, f := range fetchers {
		names[f.Name()] = true
	}

	if !names["Test RSS"] {
		t.Error("Missing Test RSS fetcher")
	}
	if !names["Test YouTube"] {
		t.Error("Missing Test YouTube fetcher")
	}
	if !names["r/test"] {
		t.Error("Missing r/test fetcher")
	}
}

func TestExtractSubreddit(t *testing.T) {
	tests := []struct {
		url      string
		fallback string
		expected string
	}{
		{"https://www.reddit.com/r/fpv/.rss", "r/fpv", "fpv"},
		{"https://www.reddit.com/r/Multicopter/hot.json", "r/Multicopter", "Multicopter"},
		{"", "r/drones", "drones"},
		{"invalid-url", "fallback", "fallback"},
	}

	for _, tt := range tests {
		result := extractSubreddit(tt.url, tt.fallback)
		if result != tt.expected {
			t.Errorf("extractSubreddit(%q, %q) = %q, want %q", tt.url, tt.fallback, result, tt.expected)
		}
	}
}

func TestGetDefaultFeedsConfig(t *testing.T) {
	config := GetDefaultFeedsConfig()

	if len(config.Sources) == 0 {
		t.Error("GetDefaultFeedsConfig() returned empty sources")
	}

	// Verify we have at least some RSS and Reddit sources
	hasRSS := false
	hasReddit := false
	for _, s := range config.Sources {
		if s.Type == "rss" {
			hasRSS = true
		}
		if s.Type == "reddit" {
			hasReddit = true
		}
	}

	if !hasRSS {
		t.Error("Default config missing RSS sources")
	}
	if !hasReddit {
		t.Error("Default config missing Reddit sources")
	}
}

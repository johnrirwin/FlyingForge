package sources

import (
	"testing"
	"time"

	"github.com/johnrirwin/flyingforge/internal/ratelimit"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Timeout != 30*time.Second {
		t.Errorf("DefaultConfig().Timeout = %v, want %v", config.Timeout, 30*time.Second)
	}
	if config.MaxItems != 50 {
		t.Errorf("DefaultConfig().MaxItems = %d, want %d", config.MaxItems, 50)
	}
	if config.UserAgent != "DroneNewsAggregator/1.0" {
		t.Errorf("DefaultConfig().UserAgent = %q, want %q", config.UserAgent, "DroneNewsAggregator/1.0")
	}
}

func TestNewRSSFetcher(t *testing.T) {
	limiter := ratelimit.New(time.Second)
	config := FetcherConfig{
		Timeout:   10 * time.Second,
		MaxItems:  20,
		UserAgent: "TestAgent/1.0",
	}

	fetcher := NewRSSFetcher("TestFeed", "https://example.com/feed", limiter, config)

	if fetcher == nil {
		t.Fatal("NewRSSFetcher() returned nil")
	}
	if fetcher.name != "TestFeed" {
		t.Errorf("NewRSSFetcher() name = %q, want %q", fetcher.name, "TestFeed")
	}
	if fetcher.url != "https://example.com/feed" {
		t.Errorf("NewRSSFetcher() url = %q, want %q", fetcher.url, "https://example.com/feed")
	}
	if fetcher.parser == nil {
		t.Error("NewRSSFetcher() parser should not be nil")
	}
}

func TestRSSFetcher_Name(t *testing.T) {
	limiter := ratelimit.New(time.Second)
	config := DefaultConfig()

	fetcher := NewRSSFetcher("DroneDJ", "https://dronedj.com/feed/", limiter, config)

	if got := fetcher.Name(); got != "DroneDJ" {
		t.Errorf("Name() = %q, want %q", got, "DroneDJ")
	}
}

func TestRSSFetcher_SourceInfo(t *testing.T) {
	limiter := ratelimit.New(time.Second)
	config := DefaultConfig()

	fetcher := NewRSSFetcher("Drone DJ", "https://dronedj.com/feed/", limiter, config)

	info := fetcher.SourceInfo()

	if info.ID != "drone-dj" {
		t.Errorf("SourceInfo().ID = %q, want %q", info.ID, "drone-dj")
	}
	if info.Name != "Drone DJ" {
		t.Errorf("SourceInfo().Name = %q, want %q", info.Name, "Drone DJ")
	}
	if info.URL != "https://dronedj.com/feed/" {
		t.Errorf("SourceInfo().URL = %q, want %q", info.URL, "https://dronedj.com/feed/")
	}
	if info.SourceType != "news" {
		t.Errorf("SourceInfo().SourceType = %q, want %q", info.SourceType, "news")
	}
	if info.FeedType != "rss" {
		t.Errorf("SourceInfo().FeedType = %q, want %q", info.FeedType, "rss")
	}
	if !info.Enabled {
		t.Error("SourceInfo().Enabled should be true")
	}
}

func TestRSSFetcher_SourceInfo_SpaceReplacement(t *testing.T) {
	limiter := ratelimit.New(time.Second)
	config := DefaultConfig()

	tests := []struct {
		name       string
		expectedID string
	}{
		{"Simple", "simple"},
		{"Two Words", "two-words"},
		{"Three Word Name", "three-word-name"},
		{"Already-Dashed", "already-dashed"},
	}

	for _, tt := range tests {
		fetcher := NewRSSFetcher(tt.name, "https://example.com/feed/", limiter, config)
		info := fetcher.SourceInfo()
		if info.ID != tt.expectedID {
			t.Errorf("SourceInfo(%q).ID = %q, want %q", tt.name, info.ID, tt.expectedID)
		}
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID("source1", "http://example.com/article1")
	id2 := generateID("source1", "http://example.com/article1")
	if id1 != id2 {
		t.Error("generateID() should be deterministic")
	}

	id3 := generateID("source1", "http://example.com/article2")
	if id1 == id3 {
		t.Error("generateID() should produce different IDs for different URLs")
	}

	id4 := generateID("source2", "http://example.com/article1")
	if id1 == id4 {
		t.Error("generateID() should produce different IDs for different sources")
	}

	if len(id1) != 16 {
		t.Errorf("generateID() length = %d, want 16", len(id1))
	}
}

func TestCreateDroneRSSFetchers(t *testing.T) {
	limiter := ratelimit.New(time.Second)
	config := DefaultConfig()

	fetchers := CreateDroneRSSFetchers(limiter, config)

	if len(fetchers) == 0 {
		t.Fatal("CreateDroneRSSFetchers() returned empty slice")
	}

	for i, f := range fetchers {
		if f == nil {
			t.Errorf("CreateDroneRSSFetchers()[%d] is nil", i)
			continue
		}
		if f.Name() == "" {
			t.Errorf("CreateDroneRSSFetchers()[%d].Name() is empty", i)
		}
		info := f.SourceInfo()
		if info.ID == "" {
			t.Errorf("CreateDroneRSSFetchers()[%d].SourceInfo().ID is empty", i)
		}
	}

	expectedSources := []string{"DroneDJ", "DroneLife", "sUAS News"}
	names := make(map[string]bool)
	for _, f := range fetchers {
		names[f.Name()] = true
	}
	for _, expected := range expectedSources {
		if !names[expected] {
			t.Errorf("CreateDroneRSSFetchers() missing expected source %q", expected)
		}
	}
}

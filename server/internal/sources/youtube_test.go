package sources

import (
	"testing"
	"time"

	"github.com/johnrirwin/flyingforge/internal/ratelimit"
)

func TestNewYouTubeFetcher(t *testing.T) {
	limiter := ratelimit.New(time.Second)
	config := DefaultConfig()

	fetcher := NewYouTubeFetcher("Joshua Bardwell", "https://www.youtube.com/feeds/videos.xml?channel_id=UCX3eufnI7A2I7IkKHZn8KSQ", limiter, config)

	if fetcher == nil {
		t.Fatal("NewYouTubeFetcher() returned nil")
	}

	if fetcher.Name() != "Joshua Bardwell" {
		t.Errorf("Name() = %q, want %q", fetcher.Name(), "Joshua Bardwell")
	}
}

func TestYouTubeFetcherSourceInfo(t *testing.T) {
	limiter := ratelimit.New(time.Second)
	config := DefaultConfig()

	fetcher := NewYouTubeFetcher("Test Channel", "https://www.youtube.com/feeds/videos.xml?channel_id=UC123ABC", limiter, config)

	info := fetcher.SourceInfo()

	if info.ID != "yt-test-channel" {
		t.Errorf("SourceInfo().ID = %q, want %q", info.ID, "yt-test-channel")
	}

	if info.Name != "Test Channel" {
		t.Errorf("SourceInfo().Name = %q, want %q", info.Name, "Test Channel")
	}

	if info.SourceType != "youtube" {
		t.Errorf("SourceInfo().SourceType = %q, want %q", info.SourceType, "youtube")
	}

	if info.FeedType != "youtube" {
		t.Errorf("SourceInfo().FeedType = %q, want %q", info.FeedType, "youtube")
	}

	if info.ChannelID != "UC123ABC" {
		t.Errorf("SourceInfo().ChannelID = %q, want %q", info.ChannelID, "UC123ABC")
	}
}

func TestExtractChannelID(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://www.youtube.com/feeds/videos.xml?channel_id=UCX3eufnI7A2I7IkKHZn8KSQ", "UCX3eufnI7A2I7IkKHZn8KSQ"},
		{"https://www.youtube.com/feeds/videos.xml?channel_id=UC123", "UC123"},
		{"https://example.com/no-channel", ""},
		{"", ""},
	}

	for _, tt := range tests {
		result := extractChannelID(tt.url)
		if result != tt.expected {
			t.Errorf("extractChannelID(%q) = %q, want %q", tt.url, result, tt.expected)
		}
	}
}

func TestExtractVideoID(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"https://youtu.be/dQw4w9WgXcQ", "dQw4w9WgXcQ"},
		{"https://example.com/not-a-video", ""},
		{"", ""},
	}

	for _, tt := range tests {
		result := extractVideoID(tt.url)
		if result != tt.expected {
			t.Errorf("extractVideoID(%q) = %q, want %q", tt.url, result, tt.expected)
		}
	}
}

func TestTruncateSummary(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"Short text", 100, "Short text"},
		{"This is a longer text that needs truncation", 20, "This is a longer tex..."},
		{"<p>HTML content</p>", 100, "HTML content"},
		{"<div><span>Nested</span> tags</div>", 100, "Nested tags"},
	}

	for _, tt := range tests {
		result := truncateSummary(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateSummary(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

package sources

import (
	"testing"
	"time"

	"github.com/johnrirwin/rotorlife/internal/ratelimit"
)

func TestNewForumFetcher(t *testing.T) {
	limiter := ratelimit.New(time.Second)
	config := FetcherConfig{
		Timeout:   10 * time.Second,
		MaxItems:  20,
		UserAgent: "TestAgent/1.0",
	}
	selectors := ForumSelectors{
		Container: ".thread",
		Title:     ".title",
		Link:      ".title a",
		Author:    ".author",
		Date:      ".date",
		Summary:   ".preview",
	}

	fetcher := NewForumFetcher("TestForum", "https://example.com/forum", selectors, limiter, config)

	if fetcher == nil {
		t.Fatal("NewForumFetcher() returned nil")
	}
	if fetcher.name != "TestForum" {
		t.Errorf("NewForumFetcher() name = %q, want %q", fetcher.name, "TestForum")
	}
	if fetcher.url != "https://example.com/forum" {
		t.Errorf("NewForumFetcher() url = %q, want %q", fetcher.url, "https://example.com/forum")
	}
	if fetcher.client == nil {
		t.Error("NewForumFetcher() client should not be nil")
	}
}

func TestForumFetcher_Name(t *testing.T) {
	limiter := ratelimit.New(time.Second)
	config := DefaultConfig()
	selectors := ForumSelectors{}

	fetcher := NewForumFetcher("RC Groups", "https://rcgroups.com", selectors, limiter, config)

	if got := fetcher.Name(); got != "RC Groups" {
		t.Errorf("Name() = %q, want %q", got, "RC Groups")
	}
}

func TestForumFetcher_SourceInfo(t *testing.T) {
	limiter := ratelimit.New(time.Second)
	config := DefaultConfig()
	selectors := ForumSelectors{}

	fetcher := NewForumFetcher("RC Groups", "https://rcgroups.com", selectors, limiter, config)

	info := fetcher.SourceInfo()

	if info.ID != "rc-groups" {
		t.Errorf("SourceInfo().ID = %q, want %q", info.ID, "rc-groups")
	}
	if info.Name != "RC Groups" {
		t.Errorf("SourceInfo().Name = %q, want %q", info.Name, "RC Groups")
	}
	if info.URL != "https://rcgroups.com" {
		t.Errorf("SourceInfo().URL = %q, want %q", info.URL, "https://rcgroups.com")
	}
	if info.SourceType != "community" {
		t.Errorf("SourceInfo().SourceType = %q, want %q", info.SourceType, "community")
	}
	if info.FeedType != "forum" {
		t.Errorf("SourceInfo().FeedType = %q, want %q", info.FeedType, "forum")
	}
	if !info.Enabled {
		t.Error("SourceInfo().Enabled should be true")
	}
}

func TestForumFetcher_SourceInfo_SpaceReplacement(t *testing.T) {
	limiter := ratelimit.New(time.Second)
	config := DefaultConfig()
	selectors := ForumSelectors{}

	tests := []struct {
		name       string
		expectedID string
	}{
		{"SimpleForum", "simpleforum"},
		{"RC Groups", "rc-groups"},
		{"FPV Lab Forum", "fpv-lab-forum"},
	}

	for _, tt := range tests {
		fetcher := NewForumFetcher(tt.name, "https://example.com", selectors, limiter, config)
		info := fetcher.SourceInfo()
		if info.ID != tt.expectedID {
			t.Errorf("SourceInfo(%q).ID = %q, want %q", tt.name, info.ID, tt.expectedID)
		}
	}
}

func TestForumSelectors(t *testing.T) {
	selectors := ForumSelectors{
		Container: ".thread-list .thread",
		Title:     "h2.title",
		Link:      "h2.title a",
		Author:    ".meta .author",
		Date:      ".meta .date",
		Summary:   ".thread-preview",
	}

	if selectors.Container != ".thread-list .thread" {
		t.Error("ForumSelectors.Container not set correctly")
	}
	if selectors.Title != "h2.title" {
		t.Error("ForumSelectors.Title not set correctly")
	}
	if selectors.Link != "h2.title a" {
		t.Error("ForumSelectors.Link not set correctly")
	}
	if selectors.Author != ".meta .author" {
		t.Error("ForumSelectors.Author not set correctly")
	}
	if selectors.Date != ".meta .date" {
		t.Error("ForumSelectors.Date not set correctly")
	}
	if selectors.Summary != ".thread-preview" {
		t.Error("ForumSelectors.Summary not set correctly")
	}
}

func TestForumFetcher_ConfigTimeout(t *testing.T) {
	limiter := ratelimit.New(time.Second)
	config := FetcherConfig{
		Timeout:   15 * time.Second,
		MaxItems:  25,
		UserAgent: "CustomAgent/2.0",
	}
	selectors := ForumSelectors{}

	fetcher := NewForumFetcher("TestForum", "https://example.com", selectors, limiter, config)

	if fetcher.client.Timeout != 15*time.Second {
		t.Errorf("ForumFetcher client timeout = %v, want %v", fetcher.client.Timeout, 15*time.Second)
	}
	if fetcher.config.MaxItems != 25 {
		t.Errorf("ForumFetcher config.MaxItems = %d, want %d", fetcher.config.MaxItems, 25)
	}
	if fetcher.config.UserAgent != "CustomAgent/2.0" {
		t.Errorf("ForumFetcher config.UserAgent = %q, want %q", fetcher.config.UserAgent, "CustomAgent/2.0")
	}
}

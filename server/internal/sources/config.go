package sources

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/johnrirwin/flyingforge/internal/ratelimit"
)

// FeedSource represents a single feed source from config
type FeedSource struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Type     string `json:"type"`     // "rss", "reddit", "youtube"
	Category string `json:"category"` // "news", "community", "creator"
	Enabled  bool   `json:"enabled"`
}

// FeedsConfig holds the feeds configuration
type FeedsConfig struct {
	Sources []FeedSource `json:"sources"`
}

// LoadFeedsConfig loads feed sources from a JSON config file
func LoadFeedsConfig(configPath string) (*FeedsConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read feeds config: %w", err)
	}

	var config FeedsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse feeds config: %w", err)
	}

	return &config, nil
}

// FindFeedsConfig searches for feeds.json in common locations
func FindFeedsConfig() string {
	// Check common locations in order of priority
	locations := []string{
		"feeds.json",        // Current directory
		"./feeds.json",      // Explicit current directory
		"../feeds.json",     // Parent directory (for running from cmd/server)
		"/app/feeds.json",   // Docker container path
		"server/feeds.json", // Project root
		"config/feeds.json", // Config subdirectory
	}

	// Also check FEEDS_CONFIG_PATH environment variable
	if envPath := os.Getenv("FEEDS_CONFIG_PATH"); envPath != "" {
		locations = append([]string{envPath}, locations...)
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			absPath, _ := filepath.Abs(loc)
			return absPath
		}
	}

	return ""
}

// CreateFetchersFromConfig creates fetchers from the feeds configuration
func CreateFetchersFromConfig(config *FeedsConfig, limiter *ratelimit.Limiter, fetcherConfig FetcherConfig) []Fetcher {
	fetchers := make([]Fetcher, 0, len(config.Sources))

	for _, source := range config.Sources {
		if !source.Enabled {
			continue
		}

		var fetcher Fetcher
		switch source.Type {
		case "rss", "news":
			fetcher = NewRSSFetcher(source.Name, source.URL, limiter, fetcherConfig)
		case "reddit":
			// Extract subreddit name from URL or use name
			subreddit := extractSubreddit(source.URL, source.Name)
			fetcher = NewRedditFetcher(subreddit, limiter, fetcherConfig)
		case "youtube":
			fetcher = NewYouTubeFetcher(source.Name, source.URL, limiter, fetcherConfig)
		default:
			// Unknown type, skip
			continue
		}

		fetchers = append(fetchers, fetcher)
	}

	return fetchers
}

// extractSubreddit extracts the subreddit name from a Reddit URL
func extractSubreddit(url, fallbackName string) string {
	// Match /r/subredditname
	re := regexp.MustCompile(`/r/([^/.\s]+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}

	// Fallback: try to get from name like "r/fpv"
	if strings.HasPrefix(fallbackName, "r/") {
		return strings.TrimPrefix(fallbackName, "r/")
	}

	return fallbackName
}

// GetDefaultFeedsConfig returns a default configuration when no config file is found
func GetDefaultFeedsConfig() *FeedsConfig {
	return &FeedsConfig{
		Sources: []FeedSource{
			// News sources
			{Name: "DroneDJ", URL: "https://dronedj.com/feed/", Type: "rss", Category: "news", Enabled: true},
			{Name: "DroneLife", URL: "https://dronelife.com/feed/", Type: "rss", Category: "news", Enabled: true},
			{Name: "sUAS News", URL: "https://www.suasnews.com/feed/", Type: "rss", Category: "news", Enabled: true},
			// Reddit sources
			{Name: "r/fpv", URL: "https://www.reddit.com/r/fpv/.rss", Type: "reddit", Category: "community", Enabled: true},
			{Name: "r/Multicopter", URL: "https://www.reddit.com/r/Multicopter/.rss", Type: "reddit", Category: "community", Enabled: true},
			{Name: "r/drones", URL: "https://www.reddit.com/r/drones/.rss", Type: "reddit", Category: "community", Enabled: true},
		},
	}
}

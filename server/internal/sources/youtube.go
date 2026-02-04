package sources

import (
	"context"
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/johnrirwin/flyingforge/internal/models"
	"github.com/johnrirwin/flyingforge/internal/ratelimit"
)

// YouTubeFetcher fetches videos from YouTube channel RSS feeds
type YouTubeFetcher struct {
	name    string
	url     string
	parser  *gofeed.Parser
	limiter *ratelimit.Limiter
	config  FetcherConfig
}

// NewYouTubeFetcher creates a new YouTube RSS fetcher
func NewYouTubeFetcher(name, url string, limiter *ratelimit.Limiter, config FetcherConfig) *YouTubeFetcher {
	return &YouTubeFetcher{
		name:    name,
		url:     url,
		parser:  gofeed.NewParser(),
		limiter: limiter,
		config:  config,
	}
}

func (f *YouTubeFetcher) Name() string {
	return f.name
}

func (f *YouTubeFetcher) SourceInfo() models.SourceInfo {
	// Extract channel ID from URL for unique ID
	channelID := extractChannelID(f.url)
	sourceID := strings.ToLower(strings.ReplaceAll(f.name, " ", "-"))
	
	return models.SourceInfo{
		ID:          "yt-" + sourceID,
		Name:        f.name,
		URL:         f.url,
		SourceType:  "youtube",
		Description: "YouTube channel: " + f.name,
		FeedType:    "youtube",
		ChannelID:   channelID,
		Enabled:     true,
	}
}

func (f *YouTubeFetcher) Fetch(ctx context.Context) ([]models.FeedItem, error) {
	f.limiter.Wait("youtube.com")

	ctxWithTimeout, cancel := context.WithTimeout(ctx, f.config.Timeout)
	defer cancel()

	feed, err := f.parser.ParseURLWithContext(f.url, ctxWithTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YouTube feed %s: %w", f.url, err)
	}

	items := make([]models.FeedItem, 0, len(feed.Items))
	for i, item := range feed.Items {
		if i >= f.config.MaxItems {
			break
		}

		publishedAt := time.Now()
		if item.PublishedParsed != nil {
			publishedAt = *item.PublishedParsed
		}

		// Extract video ID for thumbnail
		videoID := extractVideoID(item.Link)
		thumbnail := ""
		if videoID != "" {
			thumbnail = fmt.Sprintf("https://img.youtube.com/vi/%s/mqdefault.jpg", videoID)
		}

		// Author is the channel name
		author := f.name
		if item.Author != nil && item.Author.Name != "" {
			author = item.Author.Name
		}

		// Clean up description/summary
		summary := item.Description
		if summary == "" && item.Content != "" {
			summary = truncateSummary(item.Content, 300)
		}

		feedItem := models.FeedItem{
			ID:          generateYouTubeID(f.name, item.Link),
			Title:       item.Title,
			URL:         item.Link,
			Source:      f.name,
			SourceType:  "youtube",
			Author:      author,
			Summary:     summary,
			Content:     item.Content,
			PublishedAt: publishedAt,
			FetchedAt:   time.Now(),
			Thumbnail:   thumbnail,
			Tags:        []string{"youtube", "video"},
			Media: &models.MediaInfo{
				Type:     "video",
				ImageUrl: thumbnail,
				VideoUrl: item.Link,
			},
		}
		items = append(items, feedItem)
	}

	return items, nil
}

// extractChannelID extracts the channel ID from a YouTube RSS feed URL
func extractChannelID(url string) string {
	re := regexp.MustCompile(`channel_id=([a-zA-Z0-9_-]+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractVideoID extracts the video ID from a YouTube video URL
func extractVideoID(url string) string {
	// Handle youtube.com/watch?v=VIDEO_ID
	re := regexp.MustCompile(`(?:youtube\.com/watch\?v=|youtu\.be/)([a-zA-Z0-9_-]+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// generateYouTubeID creates a unique ID for a YouTube video
func generateYouTubeID(source, url string) string {
	hash := sha256.Sum256([]byte("yt:" + source + url))
	return fmt.Sprintf("%x", hash[:8])
}

// truncateSummary truncates text to maxLen characters with ellipsis
func truncateSummary(s string, maxLen int) string {
	// Strip HTML tags for cleaner summary
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

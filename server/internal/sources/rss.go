package sources

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"

	"github.com/johnrirwin/mcp-news-feed/internal/models"
	"github.com/johnrirwin/mcp-news-feed/internal/ratelimit"
)

type RSSFetcher struct {
	name    string
	url     string
	parser  *gofeed.Parser
	limiter *ratelimit.Limiter
	config  FetcherConfig
}

func NewRSSFetcher(name, url string, limiter *ratelimit.Limiter, config FetcherConfig) *RSSFetcher {
	return &RSSFetcher{
		name:    name,
		url:     url,
		parser:  gofeed.NewParser(),
		limiter: limiter,
		config:  config,
	}
}

func (f *RSSFetcher) Name() string {
	return f.name
}

func (f *RSSFetcher) SourceInfo() models.SourceInfo {
	return models.SourceInfo{
		ID:          strings.ToLower(strings.ReplaceAll(f.name, " ", "-")),
		Name:        f.name,
		URL:         f.url,
		SourceType:  "news",
		Description: "RSS feed from " + f.name,
		FeedType:    "rss",
		Enabled:     true,
	}
}

func (f *RSSFetcher) Fetch(ctx context.Context) ([]models.FeedItem, error) {
	f.limiter.Wait(f.url)

	ctxWithTimeout, cancel := context.WithTimeout(ctx, f.config.Timeout)
	defer cancel()

	feed, err := f.parser.ParseURLWithContext(f.url, ctxWithTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSS feed %s: %w", f.url, err)
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

		author := ""
		if item.Author != nil {
			author = item.Author.Name
		}

		thumbnail := ""
		if item.Image != nil {
			thumbnail = item.Image.URL
		}

		feedItem := models.FeedItem{
			ID:          generateID(f.name, item.Link),
			Title:       item.Title,
			URL:         item.Link,
			Source:      f.name,
			SourceType:  "rss",
			Author:      author,
			Summary:     item.Description,
			Content:     item.Content,
			PublishedAt: publishedAt,
			FetchedAt:   time.Now(),
			Thumbnail:   thumbnail,
			Tags:        item.Categories,
		}
		items = append(items, feedItem)
	}

	return items, nil
}

func generateID(source, url string) string {
	hash := sha256.Sum256([]byte(source + url))
	return fmt.Sprintf("%x", hash[:8])
}

func CreateDroneRSSFetchers(limiter *ratelimit.Limiter, config FetcherConfig) []Fetcher {
	sources := []struct {
		name string
		url  string
	}{
		{"DroneDJ", "https://dronedj.com/feed/"},
		{"DroneLife", "https://dronelife.com/feed/"},
		{"sUAS News", "https://www.suasnews.com/feed/"},
		{"DroneBlog", "https://www.droneblog.com/feed/"},
		{"Haye's UAV", "https://www.yourdroneadvisor.com/feed/"},
		{"Commercial UAV News", "https://www.commercialuavnews.com/feed"},
	}

	fetchers := make([]Fetcher, 0, len(sources))
	for _, s := range sources {
		fetchers = append(fetchers, NewRSSFetcher(s.name, s.url, limiter, config))
	}
	return fetchers
}

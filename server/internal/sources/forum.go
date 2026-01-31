package sources

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/johnrirwin/mcp-news-feed/internal/models"
	"github.com/johnrirwin/mcp-news-feed/internal/ratelimit"
)

type ForumFetcher struct {
	name      string
	url       string
	selectors ForumSelectors
	limiter   *ratelimit.Limiter
	config    FetcherConfig
	client    *http.Client
}

type ForumSelectors struct {
	Container string
	Title     string
	Link      string
	Author    string
	Date      string
	Summary   string
}

func NewForumFetcher(name, url string, selectors ForumSelectors, limiter *ratelimit.Limiter, config FetcherConfig) *ForumFetcher {
	return &ForumFetcher{
		name:      name,
		url:       url,
		selectors: selectors,
		limiter:   limiter,
		config:    config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

func (f *ForumFetcher) Name() string {
	return f.name
}

func (f *ForumFetcher) SourceInfo() models.SourceInfo {
	return models.SourceInfo{
		ID:          strings.ToLower(strings.ReplaceAll(f.name, " ", "-")),
		Name:        f.name,
		URL:         f.url,
		SourceType:  "community",
		Description: "Forum " + f.name,
		FeedType:    "forum",
		Enabled:     true,
	}
}

func (f *ForumFetcher) Fetch(ctx context.Context) ([]models.FeedItem, error) {
	f.limiter.Wait(f.url)

	req, err := http.NewRequestWithContext(ctx, "GET", f.url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", f.config.UserAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch forum page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("forum returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse forum HTML: %w", err)
	}

	items := make([]models.FeedItem, 0)
	doc.Find(f.selectors.Container).Each(func(i int, s *goquery.Selection) {
		if i >= f.config.MaxItems {
			return
		}

		title := strings.TrimSpace(s.Find(f.selectors.Title).Text())
		if title == "" {
			return
		}

		link, _ := s.Find(f.selectors.Link).Attr("href")
		if link == "" {
			link, _ = s.Find(f.selectors.Title).Attr("href")
		}
		if link != "" && !strings.HasPrefix(link, "http") {
			link = resolveURL(f.url, link)
		}

		author := strings.TrimSpace(s.Find(f.selectors.Author).Text())
		summary := strings.TrimSpace(s.Find(f.selectors.Summary).Text())

		item := models.FeedItem{
			ID:          generateID(f.name, link),
			Title:       title,
			URL:         link,
			Source:      f.name,
			SourceType:  "forum",
			Author:      author,
			Summary:     truncate(summary, 300),
			PublishedAt: time.Now(),
			FetchedAt:   time.Now(),
			Tags:        []string{},
		}
		items = append(items, item)
	})

	return items, nil
}

func resolveURL(base, relative string) string {
	if strings.HasPrefix(relative, "/") {
		parts := strings.SplitN(base, "/", 4)
		if len(parts) >= 3 {
			return parts[0] + "//" + parts[2] + relative
		}
	}
	return base + "/" + relative
}

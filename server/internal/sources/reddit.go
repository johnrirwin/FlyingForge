package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/johnrirwin/mcp-news-feed/internal/models"
	"github.com/johnrirwin/mcp-news-feed/internal/ratelimit"
)

type RedditFetcher struct {
	subreddit string
	limiter   *ratelimit.Limiter
	config    FetcherConfig
	client    *http.Client
}

type redditResponse struct {
	Data struct {
		Children []struct {
			Data redditPost `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

type redditPost struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Selftext  string  `json:"selftext"`
	Author    string  `json:"author"`
	URL       string  `json:"url"`
	Permalink string  `json:"permalink"`
	Created   float64 `json:"created_utc"`
	Score     int     `json:"score"`
	NumComms  int     `json:"num_comments"`
	Thumbnail string  `json:"thumbnail"`
	Flair     string  `json:"link_flair_text"`
}

func NewRedditFetcher(subreddit string, limiter *ratelimit.Limiter, config FetcherConfig) *RedditFetcher {
	return &RedditFetcher{
		subreddit: subreddit,
		limiter:   limiter,
		config:    config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

func (f *RedditFetcher) Name() string {
	return "r/" + f.subreddit
}

func (f *RedditFetcher) SourceInfo() models.SourceInfo {
	return models.SourceInfo{
		Name:   "r/" + f.subreddit,
		Type:   "reddit",
		URL:    fmt.Sprintf("https://www.reddit.com/r/%s", f.subreddit),
		Active: true,
	}
}

func (f *RedditFetcher) Fetch(ctx context.Context) ([]models.FeedItem, error) {
	f.limiter.Wait("reddit.com")

	url := fmt.Sprintf("https://www.reddit.com/r/%s/hot.json?limit=%d", f.subreddit, f.config.MaxItems)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", f.config.UserAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch reddit posts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reddit returned status %d", resp.StatusCode)
	}

	var data redditResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode reddit response: %w", err)
	}

	items := make([]models.FeedItem, 0, len(data.Data.Children))
	for _, child := range data.Data.Children {
		post := child.Data

		publishedAt := time.Unix(int64(post.Created), 0)

		thumbnail := ""
		if post.Thumbnail != "self" && post.Thumbnail != "default" && post.Thumbnail != "nsfw" && post.Thumbnail != "" {
			thumbnail = post.Thumbnail
		}

		tags := []string{}
		if post.Flair != "" {
			tags = append(tags, post.Flair)
		}

		item := models.FeedItem{
			ID:          generateID("reddit", post.ID),
			Title:       post.Title,
			URL:         "https://www.reddit.com" + post.Permalink,
			Source:      "r/" + f.subreddit,
			SourceType:  "reddit",
			Author:      post.Author,
			Summary:     truncate(post.Selftext, 300),
			Content:     post.Selftext,
			PublishedAt: publishedAt,
			FetchedAt:   time.Now(),
			Thumbnail:   thumbnail,
			Tags:        tags,
			Engagement: &models.Engagement{
				Upvotes:  post.Score,
				Comments: post.NumComms,
			},
		}
		items = append(items, item)
	}

	return items, nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func CreateDroneRedditFetchers(limiter *ratelimit.Limiter, config FetcherConfig) []Fetcher {
	subreddits := []string{
		"drones",
		"djimavic",
		"fpv",
		"Multicopter",
	}

	fetchers := make([]Fetcher, 0, len(subreddits))
	for _, sub := range subreddits {
		fetchers = append(fetchers, NewRedditFetcher(sub, limiter, config))
	}
	return fetchers
}

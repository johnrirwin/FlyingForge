package sources

import (
	"context"
	"time"

	"github.com/johnrirwin/mcp-news-feed/internal/models"
)

type Fetcher interface {
	Name() string
	Fetch(ctx context.Context) ([]models.FeedItem, error)
	SourceInfo() models.SourceInfo
}

type FetchResult struct {
	Items  []models.FeedItem
	Source models.SourceInfo
	Error  error
}

type FetcherConfig struct {
	Timeout   time.Duration
	MaxItems  int
	UserAgent string
}

func DefaultConfig() FetcherConfig {
	return FetcherConfig{
		Timeout:   30 * time.Second,
		MaxItems:  50,
		UserAgent: "DroneNewsAggregator/1.0",
	}
}

package aggregator

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/johnrirwin/mcp-news-feed/internal/cache"
	"github.com/johnrirwin/mcp-news-feed/internal/logging"
	"github.com/johnrirwin/mcp-news-feed/internal/models"
	"github.com/johnrirwin/mcp-news-feed/internal/sources"
	"github.com/johnrirwin/mcp-news-feed/internal/tagging"
)

type Aggregator struct {
	fetchers []sources.Fetcher
	cache    *cache.Cache
	tagger   *tagging.Tagger
	logger   *logging.Logger
	mu       sync.RWMutex
	items    []models.FeedItem
}

func New(fetchers []sources.Fetcher, c *cache.Cache, tagger *tagging.Tagger, logger *logging.Logger) *Aggregator {
	return &Aggregator{
		fetchers: fetchers,
		cache:    c,
		tagger:   tagger,
		logger:   logger,
		items:    make([]models.FeedItem, 0),
	}
}

func (a *Aggregator) Refresh(ctx context.Context) error {
	var wg sync.WaitGroup
	results := make(chan sources.FetchResult, len(a.fetchers))

	for _, fetcher := range a.fetchers {
		wg.Add(1)
		go func(f sources.Fetcher) {
			defer wg.Done()

			items, err := f.Fetch(ctx)
			results <- sources.FetchResult{
				Items:  items,
				Source: f.SourceInfo(),
				Error:  err,
			}
		}(fetcher)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	allItems := make([]models.FeedItem, 0)
	for result := range results {
		if result.Error != nil {
			a.logger.Warn("Failed to fetch from source", logging.WithFields(map[string]interface{}{
				"source": result.Source.Name,
				"error":  result.Error.Error(),
			}))
			continue
		}

		a.logger.Info("Fetched items from source", logging.WithFields(map[string]interface{}{
			"source": result.Source.Name,
			"count":  len(result.Items),
		}))

		for i := range result.Items {
			inferredTags := a.tagger.InferTags(result.Items[i].Title, result.Items[i].Summary)
			result.Items[i].Tags = mergeTags(result.Items[i].Tags, inferredTags)
		}

		allItems = append(allItems, result.Items...)
	}

	dedupedItems := a.deduplicate(allItems)
	sortByDate(dedupedItems)

	a.mu.Lock()
	a.items = dedupedItems
	a.mu.Unlock()

	a.cache.Set("all_items", dedupedItems)

	a.logger.Info("Aggregation complete", logging.WithFields(map[string]interface{}{
		"total_items":  len(dedupedItems),
		"sources_used": len(a.fetchers),
	}))

	return nil
}

func (a *Aggregator) GetItems(params models.FilterParams) models.AggregatedResponse {
	a.mu.RLock()
	items := a.items
	a.mu.RUnlock()

	filtered := a.filterItems(items, params)
	total := len(filtered)

	if params.Limit > 0 {
		offset := params.Offset
		if offset >= len(filtered) {
			filtered = []models.FeedItem{}
		} else {
			end := offset + params.Limit
			if end > len(filtered) {
				end = len(filtered)
			}
			filtered = filtered[offset:end]
		}
	}

	return models.AggregatedResponse{
		Items:       filtered,
		TotalCount:  total,
		FetchedAt:   time.Now(),
		SourceCount: len(a.fetchers),
	}
}

func (a *Aggregator) GetSources() []models.SourceInfo {
	sourcesInfo := make([]models.SourceInfo, 0, len(a.fetchers))
	for _, f := range a.fetchers {
		sourcesInfo = append(sourcesInfo, f.SourceInfo())
	}
	return sourcesInfo
}

func (a *Aggregator) filterItems(items []models.FeedItem, params models.FilterParams) []models.FeedItem {
	if params.Source == "" && params.Tag == "" && params.Search == "" {
		return items
	}

	filtered := make([]models.FeedItem, 0)
	for _, item := range items {
		if params.Source != "" && !strings.EqualFold(item.Source, params.Source) {
			continue
		}

		if params.Tag != "" && !containsTag(item.Tags, params.Tag) {
			continue
		}

		if params.Search != "" {
			search := strings.ToLower(params.Search)
			title := strings.ToLower(item.Title)
			summary := strings.ToLower(item.Summary)
			if !strings.Contains(title, search) && !strings.Contains(summary, search) {
				continue
			}
		}

		filtered = append(filtered, item)
	}

	return filtered
}

func (a *Aggregator) deduplicate(items []models.FeedItem) []models.FeedItem {
	seen := make(map[string]bool)
	titleSeen := make(map[string]bool)
	result := make([]models.FeedItem, 0, len(items))

	for _, item := range items {
		if seen[item.ID] {
			continue
		}

		normalizedTitle := strings.ToLower(strings.TrimSpace(item.Title))
		if titleSeen[normalizedTitle] {
			continue
		}

		seen[item.ID] = true
		titleSeen[normalizedTitle] = true
		result = append(result, item)
	}

	return result
}

func sortByDate(items []models.FeedItem) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].PublishedAt.After(items[j].PublishedAt)
	})
}

func containsTag(tags []string, target string) bool {
	target = strings.ToLower(target)
	for _, tag := range tags {
		if strings.ToLower(tag) == target {
			return true
		}
	}
	return false
}

func mergeTags(existing, inferred []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, tag := range existing {
		lower := strings.ToLower(tag)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, tag)
		}
	}

	for _, tag := range inferred {
		lower := strings.ToLower(tag)
		if !seen[lower] {
			seen[lower] = true
			result = append(result, tag)
		}
	}

	return result
}

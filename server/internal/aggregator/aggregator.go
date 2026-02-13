package aggregator

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/johnrirwin/flyingforge/internal/cache"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
	"github.com/johnrirwin/flyingforge/internal/sources"
	"github.com/johnrirwin/flyingforge/internal/tagging"
)

const (
	allItemsCacheKey = "all_items"
	allItemsCacheTTL = 36 * time.Hour
)

// FeedItemStore persists aggregated feed items for long-term history.
type FeedItemStore interface {
	UpsertItems(ctx context.Context, items []models.FeedItem) error
	DeleteItemsOlderThan(ctx context.Context, cutoff time.Time) (int64, error)
	QueryItems(ctx context.Context, params models.FilterParams, resolvedSources []string) ([]models.FeedItem, int, error)
}

type Aggregator struct {
	fetchers      []sources.Fetcher
	cache         cache.Cache
	store         FeedItemStore
	retentionDays int
	tagger        *tagging.Tagger
	logger        *logging.Logger
	mu            sync.RWMutex
	items         []models.FeedItem
}

func New(fetchers []sources.Fetcher, c cache.Cache, tagger *tagging.Tagger, logger *logging.Logger) *Aggregator {
	return &Aggregator{
		fetchers:      fetchers,
		cache:         c,
		tagger:        tagger,
		logger:        logger,
		retentionDays: 90,
		items:         make([]models.FeedItem, 0),
	}
}

func (a *Aggregator) SetStore(store FeedItemStore) {
	a.store = store
}

func (a *Aggregator) SetRetentionDays(days int) {
	a.retentionDays = days
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

	if a.cache != nil {
		a.cache.SetWithTTL(allItemsCacheKey, dedupedItems, allItemsCacheTTL)
	}

	if a.store != nil {
		if err := a.store.UpsertItems(ctx, dedupedItems); err != nil {
			return err
		}

		// Enforce retention policy to cap DB growth.
		// NOTE: configurable via config/env (default: 90 days). A value <= 0 disables retention cleanup.
		if a.retentionDays > 0 {
			cutoff := time.Now().AddDate(0, 0, -a.retentionDays)
			if deleted, err := a.store.DeleteItemsOlderThan(ctx, cutoff); err != nil {
				return err
			} else if deleted > 0 && a.logger != nil {
				a.logger.Info("Deleted old feed items", logging.WithFields(map[string]interface{}{
					"count":  deleted,
					"cutoff": cutoff.Format(time.RFC3339),
				}))
			}
		}
	}

	a.logger.Info("Aggregation complete", logging.WithFields(map[string]interface{}{
		"total_items":  len(dedupedItems),
		"sources_used": len(a.fetchers),
	}))

	return nil
}

func (a *Aggregator) GetItems(ctx context.Context, params models.FilterParams) models.AggregatedResponse {
	// When a persistent store is configured, prefer it so we can serve history
	// across runs (not just the last cached refresh).
	if a.store != nil {
		resolvedSources := a.resolveSourceNames(params.Sources)
		items, total, err := a.store.QueryItems(ctx, params, resolvedSources)
		if err == nil {
			fetchedAt := time.Time{}
			for _, item := range items {
				if item.FetchedAt.After(fetchedAt) {
					fetchedAt = item.FetchedAt
				}
			}
			if fetchedAt.IsZero() {
				fetchedAt = time.Now()
			}

			return models.AggregatedResponse{
				Items:       items,
				TotalCount:  total,
				FetchedAt:   fetchedAt,
				SourceCount: len(a.fetchers),
			}
		}
		if a.logger != nil {
			a.logger.Warn("Failed to load feed items from database, falling back to cache", logging.WithField("error", err.Error()))
		}
	}

	a.mu.RLock()
	items := a.items
	a.mu.RUnlock()

	if len(items) == 0 {
		if cachedItems, ok := a.loadItemsFromCache(); ok {
			a.mu.Lock()
			if len(a.items) == 0 {
				a.items = cachedItems
			}
			a.mu.Unlock()
		}

		// Re-read items after the cache warm-up attempt in case another goroutine
		// refreshed in-memory state while we were reading from cache.
		a.mu.RLock()
		items = a.items
		a.mu.RUnlock()
	}

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

// resolveSourceNames maps source IDs (as returned by /api/sources) into the source
// name values stored on FeedItem.Source.
func (a *Aggregator) resolveSourceNames(sourceIDs []string) []string {
	if len(sourceIDs) == 0 {
		return nil
	}

	idToName := make(map[string]string)
	for _, f := range a.fetchers {
		info := f.SourceInfo()
		idToName[strings.ToLower(info.ID)] = strings.ToLower(info.Name)
	}

	resolved := make([]string, 0, len(sourceIDs))
	for _, srcID := range sourceIDs {
		srcIDLower := strings.ToLower(srcID)
		if name, ok := idToName[srcIDLower]; ok {
			resolved = append(resolved, name)
		} else {
			// Fallback: also try the ID as-is in case it matches a name
			resolved = append(resolved, srcIDLower)
		}
	}
	return resolved
}

func (a *Aggregator) loadItemsFromCache() ([]models.FeedItem, bool) {
	if a.cache == nil {
		return nil, false
	}

	cached, ok := a.cache.Get(allItemsCacheKey)
	if !ok || cached == nil {
		return nil, false
	}

	items, ok := cached.([]models.FeedItem)
	if ok {
		return items, true
	}

	raw, err := json.Marshal(cached)
	if err != nil {
		return nil, false
	}

	var decoded []models.FeedItem
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, false
	}

	if len(decoded) == 0 {
		return nil, false
	}

	return decoded, true
}

func (a *Aggregator) GetSources() []models.SourceInfo {
	sourcesInfo := make([]models.SourceInfo, 0, len(a.fetchers))
	for _, f := range a.fetchers {
		sourcesInfo = append(sourcesInfo, f.SourceInfo())
	}
	return sourcesInfo
}

func (a *Aggregator) filterItems(items []models.FeedItem, params models.FilterParams) []models.FeedItem {
	// Early return if no filters
	if len(params.Sources) == 0 && params.SourceType == "" && params.Tag == "" && params.Query == "" && params.FromDate == "" && params.ToDate == "" {
		return a.sortItems(items, params.Sort)
	}

	// Build source name lookup map from source IDs
	// This maps source IDs (e.g., "r-fpv") to source names (e.g., "r/fpv")
	sourceNameMap := make(map[string]bool)
	if len(params.Sources) > 0 {
		// Get all source info to build ID -> Name mapping
		idToName := make(map[string]string)
		for _, f := range a.fetchers {
			info := f.SourceInfo()
			idToName[strings.ToLower(info.ID)] = strings.ToLower(info.Name)
		}
		// Convert requested source IDs to source names
		for _, srcID := range params.Sources {
			srcIDLower := strings.ToLower(srcID)
			if name, ok := idToName[srcIDLower]; ok {
				sourceNameMap[name] = true
			} else {
				// Fallback: also try the ID as-is in case it matches a name
				sourceNameMap[srcIDLower] = true
			}
		}
	}

	// Parse date filters
	var fromTime, toTime time.Time
	if t, ok := models.ParseDateFilter(params.FromDate); ok {
		fromTime = t
	}
	if t, ok := models.ParseDateFilter(params.ToDate); ok {
		toTime = t.Add(24*time.Hour - time.Nanosecond) // End of day
	}

	filtered := make([]models.FeedItem, 0)
	for _, item := range items {
		// Filter by sources
		if len(sourceNameMap) > 0 && !sourceNameMap[strings.ToLower(item.Source)] {
			continue
		}

		// Filter by source type (supports UI groupings like "community" and "news")
		if params.SourceType != "" {
			switch strings.ToLower(strings.TrimSpace(params.SourceType)) {
			case "community":
				// Community = reddit + forums
				if !strings.EqualFold(item.SourceType, "reddit") && !strings.EqualFold(item.SourceType, "forum") {
					continue
				}
			case "news":
				// News = RSS feeds
				if !strings.EqualFold(item.SourceType, "rss") {
					continue
				}
			default:
				if !strings.EqualFold(item.SourceType, params.SourceType) {
					continue
				}
			}
		}

		// Filter by tag
		if params.Tag != "" && !containsTag(item.Tags, params.Tag) {
			continue
		}

		// Filter by search query
		if params.Query != "" {
			search := strings.ToLower(params.Query)
			title := strings.ToLower(item.Title)
			summary := strings.ToLower(item.Summary)
			content := strings.ToLower(item.Content)
			source := strings.ToLower(item.Source)
			if !strings.Contains(title, search) && !strings.Contains(summary, search) && !strings.Contains(content, search) && !strings.Contains(source, search) {
				continue
			}
		}

		// Filter by date range
		if !fromTime.IsZero() && item.PublishedAt.Before(fromTime) {
			continue
		}
		if !toTime.IsZero() && item.PublishedAt.After(toTime) {
			continue
		}

		filtered = append(filtered, item)
	}

	return a.sortItems(filtered, params.Sort)
}

func (a *Aggregator) sortItems(items []models.FeedItem, sortBy string) []models.FeedItem {
	switch sortBy {
	case "score", "top":
		sort.Slice(items, func(i, j int) bool {
			scoreI := 0
			scoreJ := 0
			if items[i].Engagement != nil {
				scoreI = items[i].Engagement.Upvotes + items[i].Engagement.Comments
			}
			if items[j].Engagement != nil {
				scoreJ = items[j].Engagement.Upvotes + items[j].Engagement.Comments
			}
			return scoreI > scoreJ
		})
	default: // "newest" or empty
		sort.Slice(items, func(i, j int) bool {
			return items[i].PublishedAt.After(items[j].PublishedAt)
		})
	}
	return items
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

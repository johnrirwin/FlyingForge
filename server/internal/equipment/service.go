package equipment

import (
	"context"
	"sort"
	"strings"
	"sync"

	"github.com/johnrirwin/flyingforge/internal/cache"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
	"github.com/johnrirwin/flyingforge/internal/sellers"
)

// Service handles equipment aggregation from multiple sellers
type Service struct {
	registry *sellers.Registry
	cache    cache.Cache
	logger   *logging.Logger
	products map[string][]models.EquipmentItem // Cached products by category
}

// ServiceError represents an equipment service error
type ServiceError struct {
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}

// NewService creates a new equipment service
func NewService(registry *sellers.Registry, c cache.Cache, logger *logging.Logger) *Service {
	return &Service{
		registry: registry,
		cache:    c,
		logger:   logger,
		products: make(map[string][]models.EquipmentItem),
	}
}

// Search searches for equipment across all registered sellers
func (s *Service) Search(ctx context.Context, params models.EquipmentSearchParams) (*models.EquipmentSearchResponse, error) {
	adapters := s.registry.List()
	if len(adapters) == 0 {
		return &models.EquipmentSearchResponse{
			Items:      []models.EquipmentItem{},
			TotalCount: 0,
			Page:       1,
			PageSize:   params.Limit,
		}, nil
	}

	// If a specific seller is requested, only search that one
	if params.Seller != "" {
		adapter := s.registry.Get(params.Seller)
		if adapter == nil {
			return nil, &ServiceError{Message: "Unknown seller: " + params.Seller}
		}
		adapters = []sellers.Adapter{adapter}
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}

	// If no query and no category, return featured products from all categories
	if params.Query == "" && params.Category == "" {
		return s.getFeaturedProducts(ctx, adapters, limit, params)
	}

	// Search all adapters in parallel
	var wg sync.WaitGroup
	resultChan := make(chan []models.EquipmentItem, len(adapters))

	for _, adapter := range adapters {
		wg.Add(1)
		go func(a sellers.Adapter) {
			defer wg.Done()

			items, err := a.Search(ctx, params.Query, params.Category, limit)
			if err != nil {
				s.logger.Warn("Search failed for seller", logging.WithFields(map[string]interface{}{
					"seller": a.ID(),
					"error":  err.Error(),
				}))
				return
			}
			resultChan <- items
		}(adapter)
	}

	// Wait and collect results
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var allItems []models.EquipmentItem
	for items := range resultChan {
		allItems = append(allItems, items...)
	}

	// Apply filters
	allItems = s.applyFilters(allItems, params)

	// Sort results
	allItems = s.sortItems(allItems, params.Sort)

	// Paginate
	totalCount := len(allItems)
	offset := params.Offset
	if offset > totalCount {
		offset = totalCount
	}
	end := offset + limit
	if end > totalCount {
		end = totalCount
	}

	pagedItems := allItems[offset:end]

	return &models.EquipmentSearchResponse{
		Items:      pagedItems,
		TotalCount: totalCount,
		Page:       (offset / limit) + 1,
		PageSize:   limit,
		Query:      params.Query,
	}, nil
}

// GetByCategory returns equipment for a specific category
func (s *Service) GetByCategory(ctx context.Context, category models.EquipmentCategory, limit, offset int) (*models.EquipmentSearchResponse, error) {
	adapters := s.registry.List()
	if len(adapters) == 0 {
		return &models.EquipmentSearchResponse{
			Items:      []models.EquipmentItem{},
			TotalCount: 0,
		}, nil
	}

	if limit <= 0 {
		limit = 20
	}

	// Get from all adapters in parallel
	var wg sync.WaitGroup
	resultChan := make(chan []models.EquipmentItem, len(adapters))

	for _, adapter := range adapters {
		wg.Add(1)
		go func(a sellers.Adapter) {
			defer wg.Done()

			items, err := a.GetByCategory(ctx, category, limit, 0)
			if err != nil {
				s.logger.Warn("GetByCategory failed for seller", logging.WithFields(map[string]interface{}{
					"seller":   a.ID(),
					"category": category,
					"error":    err.Error(),
				}))
				return
			}
			resultChan <- items
		}(adapter)
	}

	// Wait and collect results
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var allItems []models.EquipmentItem
	for items := range resultChan {
		allItems = append(allItems, items...)
	}

	// Sort by price ascending
	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].Price < allItems[j].Price
	})

	totalCount := len(allItems)
	if offset > totalCount {
		offset = totalCount
	}
	end := offset + limit
	if end > totalCount {
		end = totalCount
	}

	pagedItems := allItems[offset:end]

	return &models.EquipmentSearchResponse{
		Items:      pagedItems,
		TotalCount: totalCount,
		Page:       (offset / limit) + 1,
		PageSize:   limit,
	}, nil
}

// GetSellers returns information about all registered sellers
func (s *Service) GetSellers() []models.SellerInfo {
	return s.registry.GetSellerInfo()
}

// getFeaturedProducts returns a mix of products from all categories for browsing
func (s *Service) getFeaturedProducts(ctx context.Context, adapters []sellers.Adapter, limit int, params models.EquipmentSearchParams) (*models.EquipmentSearchResponse, error) {
	// Featured categories to show on initial browse
	featuredCategories := []models.EquipmentCategory{
		models.CategoryFrames,
		models.CategoryMotors,
		models.CategoryFC,
		models.CategoryVTX,
		models.CategoryCameras,
		models.CategoryBatteries,
	}

	var wg sync.WaitGroup
	resultChan := make(chan []models.EquipmentItem, len(adapters)*len(featuredCategories))

	// Get a few items from each featured category from each adapter
	itemsPerCategory := 3
	if limit > 0 {
		itemsPerCategory = (limit / len(featuredCategories)) + 1
	}

	for _, adapter := range adapters {
		for _, cat := range featuredCategories {
			wg.Add(1)
			go func(a sellers.Adapter, category models.EquipmentCategory) {
				defer wg.Done()

				items, err := a.GetByCategory(ctx, category, itemsPerCategory, 0)
				if err != nil {
					s.logger.Debug("GetByCategory failed for featured", logging.WithFields(map[string]interface{}{
						"seller":   a.ID(),
						"category": category,
						"error":    err.Error(),
					}))
					return
				}
				resultChan <- items
			}(adapter, cat)
		}
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var allItems []models.EquipmentItem
	for items := range resultChan {
		allItems = append(allItems, items...)
	}

	// Apply any filters
	allItems = s.applyFilters(allItems, params)

	// Sort by category then price for nice browsing
	sort.Slice(allItems, func(i, j int) bool {
		if allItems[i].Category != allItems[j].Category {
			return allItems[i].Category < allItems[j].Category
		}
		return allItems[i].Price < allItems[j].Price
	})

	totalCount := len(allItems)
	offset := params.Offset
	if offset > totalCount {
		offset = totalCount
	}
	end := offset + limit
	if end > totalCount {
		end = totalCount
	}

	pagedItems := allItems[offset:end]

	return &models.EquipmentSearchResponse{
		Items:      pagedItems,
		TotalCount: totalCount,
		Page:       (offset / limit) + 1,
		PageSize:   limit,
	}, nil
}

// GetProduct gets a specific product by ID
func (s *Service) GetProduct(ctx context.Context, productID string) (*models.EquipmentItem, error) {
	// Determine seller from product ID prefix
	var sellerID string
	if strings.HasPrefix(productID, "rdq-") {
		sellerID = "racedayquads"
	} else if strings.HasPrefix(productID, "gfpv-") {
		sellerID = "getfpv"
	} else {
		return nil, &ServiceError{Message: "Unknown product ID format"}
	}

	adapter := s.registry.Get(sellerID)
	if adapter == nil {
		return nil, &ServiceError{Message: "Unknown seller for product"}
	}

	return adapter.GetProduct(ctx, productID)
}

// SyncProducts triggers a product sync for all sellers
func (s *Service) SyncProducts(ctx context.Context) error {
	adapters := s.registry.List()

	var wg sync.WaitGroup
	for _, adapter := range adapters {
		wg.Add(1)
		go func(a sellers.Adapter) {
			defer wg.Done()
			if err := a.SyncProducts(ctx); err != nil {
				s.logger.Warn("SyncProducts failed for seller", logging.WithFields(map[string]interface{}{
					"seller": a.ID(),
					"error":  err.Error(),
				}))
			}
		}(adapter)
	}

	wg.Wait()
	return nil
}

// applyFilters applies search filters to items
func (s *Service) applyFilters(items []models.EquipmentItem, params models.EquipmentSearchParams) []models.EquipmentItem {
	filtered := make([]models.EquipmentItem, 0, len(items))

	for _, item := range items {
		// Filter by price range
		if params.MinPrice != nil && item.Price < *params.MinPrice {
			continue
		}
		if params.MaxPrice != nil && item.Price > *params.MaxPrice {
			continue
		}

		// Filter by in-stock
		if params.InStockOnly && !item.InStock {
			continue
		}

		// Filter by category if specified
		if params.Category != "" && item.Category != params.Category {
			continue
		}

		filtered = append(filtered, item)
	}

	return filtered
}

// sortItems sorts items by the specified criteria
func (s *Service) sortItems(items []models.EquipmentItem, sortBy string) []models.EquipmentItem {
	switch sortBy {
	case "price_asc":
		sort.Slice(items, func(i, j int) bool {
			return items[i].Price < items[j].Price
		})
	case "price_desc":
		sort.Slice(items, func(i, j int) bool {
			return items[i].Price > items[j].Price
		})
	case "name":
		sort.Slice(items, func(i, j int) bool {
			return items[i].Name < items[j].Name
		})
	default:
		// Default sort by name
		sort.Slice(items, func(i, j int) bool {
			return items[i].Name < items[j].Name
		})
	}

	return items
}

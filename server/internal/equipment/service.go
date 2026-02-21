package equipment

import (
	"context"
	"sort"
	"strings"

	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

const (
	catalogSellerID   = "gear-catalog"
	catalogSellerName = "Gear Catalog"
)

type gearCatalogReader interface {
	Search(ctx context.Context, params models.GearCatalogSearchParams) (*models.GearCatalogSearchResponse, error)
	GetPopular(ctx context.Context, gearType models.GearType, limit int) ([]models.GearCatalogItem, error)
	Get(ctx context.Context, id string) (*models.GearCatalogItem, error)
}

// Service serves equipment data from the moderated gear catalog.
type Service struct {
	catalog gearCatalogReader
	logger  *logging.Logger
}

// ServiceError represents an equipment service error.
type ServiceError struct {
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}

// NewService creates a new equipment service.
func NewService(catalog gearCatalogReader, logger *logging.Logger) *Service {
	return &Service{
		catalog: catalog,
		logger:  logger,
	}
}

// Search searches for equipment in the shared gear catalog.
func (s *Service) Search(ctx context.Context, params models.EquipmentSearchParams) (*models.EquipmentSearchResponse, error) {
	limit := normalizeLimit(params.Limit)
	offset := normalizeOffset(params.Offset)

	if params.Seller != "" && params.Seller != catalogSellerID {
		return nil, &ServiceError{Message: "Unknown seller: " + params.Seller}
	}

	if s.catalog == nil {
		return emptyResponse(limit, offset, params.Query), nil
	}

	fetchLimit := limit + offset
	if fetchLimit > 100 {
		fetchLimit = 100
	}

	var (
		items []models.EquipmentItem
		err   error
	)

	if strings.TrimSpace(params.Query) == "" && strings.TrimSpace(string(params.Category)) == "" {
		items, err = s.getPopularCatalogItems(ctx, fetchLimit)
	} else {
		items, err = s.searchCatalogItems(ctx, params, fetchLimit)
	}
	if err != nil {
		return nil, err
	}

	filtered := s.applyFilters(items, params)
	filtered = s.sortItems(filtered, params.Sort)
	paged := paginate(filtered, limit, offset)

	return &models.EquipmentSearchResponse{
		Items:      paged,
		TotalCount: len(filtered),
		Page:       (offset / limit) + 1,
		PageSize:   limit,
		Query:      params.Query,
	}, nil
}

// GetByCategory returns catalog equipment for a specific category.
func (s *Service) GetByCategory(ctx context.Context, category models.EquipmentCategory, limit, offset int) (*models.EquipmentSearchResponse, error) {
	return s.Search(ctx, models.EquipmentSearchParams{
		Category: category,
		Limit:    limit,
		Offset:   offset,
		Sort:     "price_asc",
	})
}

// GetSellers returns the only catalog source exposed by the app.
func (s *Service) GetSellers() []models.SellerInfo {
	return []models.SellerInfo{
		{
			ID:          catalogSellerID,
			Name:        catalogSellerName,
			URL:         "/gear-catalog",
			Description: "Curated community gear catalog",
			Categories:  equipmentCategoryStrings(),
			Enabled:     true,
		},
	}
}

// GetProduct gets a specific catalog-backed product by ID.
func (s *Service) GetProduct(ctx context.Context, productID string) (*models.EquipmentItem, error) {
	if s.catalog == nil {
		return nil, &ServiceError{Message: "catalog unavailable"}
	}

	id := strings.TrimSpace(productID)
	id = strings.TrimPrefix(id, "catalog-")
	if id == "" {
		return nil, &ServiceError{Message: "Unknown product ID format"}
	}

	item, err := s.catalog.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, &ServiceError{Message: "product not found"}
	}

	normalized := mapCatalogItemToEquipmentItem(*item)
	return &normalized, nil
}

// SyncProducts is kept as a no-op for backward compatibility.
func (s *Service) SyncProducts(ctx context.Context) error {
	return nil
}

func (s *Service) getPopularCatalogItems(ctx context.Context, limit int) ([]models.EquipmentItem, error) {
	items, err := s.catalog.GetPopular(ctx, "", limit)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("Failed to get popular catalog items", logging.WithField("error", err.Error()))
		}
		return nil, &ServiceError{Message: "failed to fetch catalog items"}
	}

	return mapCatalogItems(items), nil
}

func (s *Service) searchCatalogItems(ctx context.Context, params models.EquipmentSearchParams, limit int) ([]models.EquipmentItem, error) {
	searchParams := models.GearCatalogSearchParams{
		Query:  strings.TrimSpace(params.Query),
		Limit:  limit,
		Offset: 0,
	}

	if gearType, ok := categoryToGearTypeFilter(params.Category); ok {
		searchParams.GearType = gearType
	}

	response, err := s.catalog.Search(ctx, searchParams)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("Catalog-backed equipment search failed", logging.WithField("error", err.Error()))
		}
		return nil, &ServiceError{Message: "search failed"}
	}

	return mapCatalogItems(response.Items), nil
}

func categoryToGearTypeFilter(category models.EquipmentCategory) (models.GearType, bool) {
	switch category {
	case "":
		return "", false
	case models.CategoryAccessories:
		// "accessories" maps to multiple catalog gear types (radio/other), so filter post-query.
		return "", false
	default:
		return models.GearTypeFromEquipmentCategory(category), true
	}
}

func mapCatalogItems(items []models.GearCatalogItem) []models.EquipmentItem {
	normalized := make([]models.EquipmentItem, 0, len(items))
	for _, item := range items {
		normalized = append(normalized, mapCatalogItemToEquipmentItem(item))
	}
	return normalized
}

func mapCatalogItemToEquipmentItem(item models.GearCatalogItem) models.EquipmentItem {
	price := 0.0
	if item.MSRP != nil {
		price = *item.MSRP
	}

	name := item.DisplayName()
	if name == "" {
		name = strings.TrimSpace(item.Brand + " " + item.Model)
	}
	if name == "" {
		name = item.CanonicalKey
	}

	productURL := "/gear-catalog"
	for _, link := range item.ShoppingLinks {
		trimmed := strings.TrimSpace(link)
		if trimmed != "" {
			productURL = trimmed
			break
		}
	}

	return models.EquipmentItem{
		ID:           item.ID,
		Name:         name,
		Category:     item.GearType.ToEquipmentCategory(),
		Manufacturer: item.Brand,
		Price:        price,
		Currency:     "USD",
		Seller:       catalogSellerName,
		SellerID:     catalogSellerID,
		ProductURL:   productURL,
		ImageURL:     item.ImageURL,
		KeySpecs:     item.Specs,
		InStock:      true,
		LastChecked:  item.UpdatedAt,
		Description:  item.Description,
	}
}

func equipmentCategoryStrings() []string {
	categories := models.AllCategories()
	out := make([]string, 0, len(categories))
	for _, category := range categories {
		out = append(out, string(category))
	}
	return out
}

func emptyResponse(limit, offset int, query string) *models.EquipmentSearchResponse {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return &models.EquipmentSearchResponse{
		Items:      []models.EquipmentItem{},
		TotalCount: 0,
		Page:       (offset / limit) + 1,
		PageSize:   limit,
		Query:      query,
	}
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func normalizeOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}

func paginate(items []models.EquipmentItem, limit, offset int) []models.EquipmentItem {
	if offset >= len(items) {
		return []models.EquipmentItem{}
	}

	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}

// applyFilters applies search filters to items.
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

// sortItems sorts items by the specified criteria.
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

package inventory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/johnrirwin/rotorlife/internal/database"
	"github.com/johnrirwin/rotorlife/internal/logging"
	"github.com/johnrirwin/rotorlife/internal/models"
)

// InventoryManager defines the interface for inventory operations
type InventoryManager interface {
	AddItem(ctx context.Context, userID string, params models.AddInventoryParams) (*models.InventoryItem, error)
	AddFromEquipment(ctx context.Context, userID string, equipment models.EquipmentItem, quantity int, condition models.ItemCondition, notes string) (*models.InventoryItem, error)
	GetItem(ctx context.Context, id string, userID string) (*models.InventoryItem, error)
	GetInventory(ctx context.Context, userID string, params models.InventoryFilterParams) (*models.InventoryResponse, error)
	UpdateItem(ctx context.Context, userID string, params models.UpdateInventoryParams) (*models.InventoryItem, error)
	RemoveItem(ctx context.Context, id string, userID string) error
	GetSummary(ctx context.Context, userID string) (*models.InventorySummary, error)
}

// Service handles inventory operations backed by PostgreSQL
type Service struct {
	store  *database.InventoryStore
	logger *logging.Logger
}

// NewService creates a new inventory service
func NewService(store *database.InventoryStore, logger *logging.Logger) *Service {
	return &Service{
		store:  store,
		logger: logger,
	}
}

// AddItem adds a new item to the inventory
func (s *Service) AddItem(ctx context.Context, userID string, params models.AddInventoryParams) (*models.InventoryItem, error) {
	if params.Category == "" {
		return nil, &ServiceError{Message: "category is required"}
	}

	if params.Name == "" {
		return nil, &ServiceError{Message: "name is required"}
	}

	if params.Condition != "" && !isValidCondition(params.Condition) {
		return nil, &ServiceError{Message: "invalid condition: " + string(params.Condition)}
	}

	s.logger.Debug("Adding inventory item", logging.WithFields(map[string]interface{}{
		"name":     params.Name,
		"category": params.Category,
		"user_id":  userID,
	}))

	item, err := s.store.Add(ctx, userID, params)
	if err != nil {
		s.logger.Error("Failed to add inventory item", logging.WithField("error", err.Error()))
		return nil, err
	}

	s.logger.Info("Added inventory item", logging.WithField("id", item.ID))
	return item, nil
}

// AddFromEquipment adds an equipment item to the inventory
func (s *Service) AddFromEquipment(ctx context.Context, userID string, equipment models.EquipmentItem, quantity int, condition models.ItemCondition, notes string) (*models.InventoryItem, error) {
	if quantity <= 0 {
		quantity = 1
	}
	if condition == "" {
		condition = models.ConditionNew
	}

	params := models.AddInventoryParams{
		Name:              equipment.Name,
		Category:          equipment.Category,
		Manufacturer:      equipment.Manufacturer,
		Quantity:          quantity,
		Condition:         condition,
		Notes:             notes,
		PurchasePrice:     &equipment.Price,
		PurchaseSeller:    equipment.Seller,
		ProductURL:        equipment.ProductURL,
		ImageURL:          equipment.ImageURL,
		Specs:             equipment.KeySpecs,
		SourceEquipmentID: equipment.ID,
	}

	return s.AddItem(ctx, userID, params)
}

// GetItem retrieves an item by ID
func (s *Service) GetItem(ctx context.Context, id string, userID string) (*models.InventoryItem, error) {
	return s.store.Get(ctx, id, userID)
}

// GetInventory retrieves inventory items with optional filtering
func (s *Service) GetInventory(ctx context.Context, userID string, params models.InventoryFilterParams) (*models.InventoryResponse, error) {
	return s.store.List(ctx, userID, params)
}

// UpdateItem updates an inventory item
func (s *Service) UpdateItem(ctx context.Context, userID string, params models.UpdateInventoryParams) (*models.InventoryItem, error) {
	if params.ID == "" {
		return nil, &ServiceError{Message: "item ID is required"}
	}

	if params.Condition != nil && !isValidCondition(*params.Condition) {
		return nil, &ServiceError{Message: "invalid condition: " + string(*params.Condition)}
	}

	s.logger.Debug("Updating inventory item", logging.WithField("id", params.ID))

	item, err := s.store.Update(ctx, userID, params)
	if err != nil {
		s.logger.Error("Failed to update inventory item", logging.WithFields(map[string]interface{}{
			"id":    params.ID,
			"error": err.Error(),
		}))
		return nil, err
	}

	s.logger.Info("Updated inventory item", logging.WithField("id", params.ID))
	return item, nil
}

// RemoveItem removes an item from the inventory
func (s *Service) RemoveItem(ctx context.Context, id string, userID string) error {
	if id == "" {
		return &ServiceError{Message: "item ID is required"}
	}

	s.logger.Debug("Removing inventory item", logging.WithField("id", id))

	if err := s.store.Delete(ctx, id, userID); err != nil {
		s.logger.Error("Failed to remove inventory item", logging.WithFields(map[string]interface{}{
			"id":    id,
			"error": err.Error(),
		}))
		return err
	}

	s.logger.Info("Removed inventory item", logging.WithField("id", id))
	return nil
}

// GetSummary returns a summary of the inventory
func (s *Service) GetSummary(ctx context.Context, userID string) (*models.InventorySummary, error) {
	return s.store.GetSummary(ctx, userID)
}

// InMemoryService is an in-memory implementation for development/testing
type InMemoryService struct {
	items  map[string]models.InventoryItem
	logger *logging.Logger
}

// NewInMemoryService creates an inventory service with in-memory storage
func NewInMemoryService(logger *logging.Logger) *InMemoryService {
	return &InMemoryService{
		items:  make(map[string]models.InventoryItem),
		logger: logger,
	}
}

// AddItem adds a new item to the in-memory inventory
func (s *InMemoryService) AddItem(ctx context.Context, userID string, params models.AddInventoryParams) (*models.InventoryItem, error) {
	if params.Category == "" {
		return nil, &ServiceError{Message: "category is required"}
	}
	if params.Name == "" {
		return nil, &ServiceError{Message: "name is required"}
	}

	id := generateID()
	now := time.Now()

	quantity := params.Quantity
	if quantity <= 0 {
		quantity = 1
	}

	condition := params.Condition
	if condition == "" {
		condition = models.ConditionNew
	}

	item := models.InventoryItem{
		ID:                id,
		UserID:            userID,
		Name:              params.Name,
		Category:          params.Category,
		Manufacturer:      params.Manufacturer,
		Quantity:          quantity,
		Condition:         condition,
		Notes:             params.Notes,
		BuildID:           params.BuildID,
		PurchasePrice:     params.PurchasePrice,
		PurchaseSeller:    params.PurchaseSeller,
		ProductURL:        params.ProductURL,
		ImageURL:          params.ImageURL,
		Specs:             params.Specs,
		SourceEquipmentID: params.SourceEquipmentID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	s.items[id] = item
	s.logger.Info("Added inventory item (in-memory)", logging.WithField("id", id))

	return &item, nil
}

// AddFromEquipment adds an equipment item to the in-memory inventory
func (s *InMemoryService) AddFromEquipment(ctx context.Context, userID string, equipment models.EquipmentItem, quantity int, condition models.ItemCondition, notes string) (*models.InventoryItem, error) {
	if quantity <= 0 {
		quantity = 1
	}
	if condition == "" {
		condition = models.ConditionNew
	}

	return s.AddItem(ctx, userID, models.AddInventoryParams{
		Name:              equipment.Name,
		Category:          equipment.Category,
		Manufacturer:      equipment.Manufacturer,
		Quantity:          quantity,
		Condition:         condition,
		Notes:             notes,
		PurchasePrice:     &equipment.Price,
		PurchaseSeller:    equipment.Seller,
		ProductURL:        equipment.ProductURL,
		ImageURL:          equipment.ImageURL,
		Specs:             equipment.KeySpecs,
		SourceEquipmentID: equipment.ID,
	})
}

// GetItem retrieves an item by ID
func (s *InMemoryService) GetItem(ctx context.Context, id string, userID string) (*models.InventoryItem, error) {
	item, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	// Filter by userID if provided
	if userID != "" && item.UserID != userID {
		return nil, nil
	}
	return &item, nil
}

// GetInventory retrieves inventory items with optional filtering
func (s *InMemoryService) GetInventory(ctx context.Context, userID string, params models.InventoryFilterParams) (*models.InventoryResponse, error) {
	items := make([]models.InventoryItem, 0, len(s.items))
	categories := make(map[models.EquipmentCategory]int)

	for _, item := range s.items {
		// Filter by userID if provided
		if userID != "" && item.UserID != userID {
			continue
		}
		if params.Category != "" && item.Category != params.Category {
			continue
		}
		if params.Condition != "" && item.Condition != params.Condition {
			continue
		}
		if params.BuildID != "" && item.BuildID != params.BuildID {
			continue
		}
		if params.Query != "" {
			query := strings.ToLower(params.Query)
			name := strings.ToLower(item.Name)
			mfg := strings.ToLower(item.Manufacturer)
			if !strings.Contains(name, query) && !strings.Contains(mfg, query) {
				continue
			}
		}

		items = append(items, item)
		categories[item.Category]++
	}

	totalCount := len(items)
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := params.Offset

	if offset >= len(items) {
		items = []models.InventoryItem{}
	} else {
		end := offset + limit
		if end > len(items) {
			end = len(items)
		}
		items = items[offset:end]
	}

	return &models.InventoryResponse{
		Items:      items,
		TotalCount: totalCount,
		Categories: categories,
	}, nil
}

// UpdateItem updates an inventory item
func (s *InMemoryService) UpdateItem(ctx context.Context, userID string, params models.UpdateInventoryParams) (*models.InventoryItem, error) {
	item, ok := s.items[params.ID]
	if !ok {
		return nil, &ServiceError{Message: "inventory item not found"}
	}
	// Check ownership if userID provided
	if userID != "" && item.UserID != userID {
		return nil, &ServiceError{Message: "inventory item not found"}
	}

	if params.Name != nil {
		item.Name = *params.Name
	}
	if params.Category != nil {
		item.Category = *params.Category
	}
	if params.Manufacturer != nil {
		item.Manufacturer = *params.Manufacturer
	}
	if params.Quantity != nil {
		item.Quantity = *params.Quantity
	}
	if params.Condition != nil {
		item.Condition = *params.Condition
	}
	if params.Notes != nil {
		item.Notes = *params.Notes
	}
	if params.BuildID != nil {
		item.BuildID = *params.BuildID
	}
	if params.PurchasePrice != nil {
		item.PurchasePrice = params.PurchasePrice
	}
	if params.PurchaseSeller != nil {
		item.PurchaseSeller = *params.PurchaseSeller
	}
	if params.ProductURL != nil {
		item.ProductURL = *params.ProductURL
	}
	if params.ImageURL != nil {
		item.ImageURL = *params.ImageURL
	}
	if params.Specs != nil {
		item.Specs = params.Specs
	}

	item.UpdatedAt = time.Now()
	s.items[params.ID] = item

	return &item, nil
}

// RemoveItem removes an item from the inventory
func (s *InMemoryService) RemoveItem(ctx context.Context, id string, userID string) error {
	item, ok := s.items[id]
	if !ok {
		return &ServiceError{Message: "inventory item not found"}
	}
	// Check ownership if userID provided
	if userID != "" && item.UserID != userID {
		return &ServiceError{Message: "inventory item not found"}
	}
	delete(s.items, id)
	return nil
}

// GetSummary returns a summary of the inventory
func (s *InMemoryService) GetSummary(ctx context.Context, userID string) (*models.InventorySummary, error) {
	summary := &models.InventorySummary{
		ByCategory:  make(map[models.EquipmentCategory]int),
		ByCondition: make(map[models.ItemCondition]int),
	}

	for _, item := range s.items {
		// Filter by userID if provided
		if userID != "" && item.UserID != userID {
			continue
		}
		summary.TotalItems += item.Quantity
		if item.PurchasePrice != nil {
			summary.TotalValue += *item.PurchasePrice * float64(item.Quantity)
		}
		summary.ByCategory[item.Category]++
		summary.ByCondition[item.Condition]++
	}

	return summary, nil
}

// ServiceError represents an error from the inventory service
type ServiceError struct {
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}

// isValidCondition checks if a condition is valid
func isValidCondition(condition models.ItemCondition) bool {
	switch condition {
	case models.ConditionNew, models.ConditionUsed, models.ConditionBroken, models.ConditionSpare:
		return true
	default:
		return false
	}
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

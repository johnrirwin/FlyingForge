package aircraft

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/johnrirwin/mcp-news-feed/internal/database"
	"github.com/johnrirwin/mcp-news-feed/internal/inventory"
	"github.com/johnrirwin/mcp-news-feed/internal/logging"
	"github.com/johnrirwin/mcp-news-feed/internal/models"
)

// Service handles aircraft operations
type Service struct {
	store        *database.AircraftStore
	inventorySvc inventory.InventoryManager
	logger       *logging.Logger
}

// NewService creates a new aircraft service
func NewService(store *database.AircraftStore, inventorySvc inventory.InventoryManager, logger *logging.Logger) *Service {
	return &Service{
		store:        store,
		inventorySvc: inventorySvc,
		logger:       logger,
	}
}

// Create creates a new aircraft
func (s *Service) Create(ctx context.Context, userID string, params models.CreateAircraftParams) (*models.Aircraft, error) {
	if params.Name == "" {
		return nil, &ServiceError{Message: "name is required"}
	}

	s.logger.Debug("Creating aircraft", logging.WithFields(map[string]interface{}{
		"name":    params.Name,
		"user_id": userID,
	}))

	aircraft, err := s.store.Create(ctx, userID, params)
	if err != nil {
		s.logger.Error("Failed to create aircraft", logging.WithField("error", err.Error()))
		return nil, err
	}

	s.logger.Info("Created aircraft", logging.WithField("id", aircraft.ID))
	return aircraft, nil
}

// Get retrieves an aircraft by ID
func (s *Service) Get(ctx context.Context, id string, userID string) (*models.Aircraft, error) {
	return s.store.Get(ctx, id, userID)
}

// Update updates an aircraft
func (s *Service) Update(ctx context.Context, userID string, params models.UpdateAircraftParams) (*models.Aircraft, error) {
	if params.ID == "" {
		return nil, &ServiceError{Message: "id is required"}
	}

	aircraft, err := s.store.Update(ctx, userID, params)
	if err != nil {
		s.logger.Error("Failed to update aircraft", logging.WithField("error", err.Error()))
		return nil, err
	}

	if aircraft == nil {
		return nil, &ServiceError{Message: "aircraft not found"}
	}

	s.logger.Info("Updated aircraft", logging.WithField("id", aircraft.ID))
	return aircraft, nil
}

// Delete deletes an aircraft
func (s *Service) Delete(ctx context.Context, id string, userID string) error {
	if id == "" {
		return &ServiceError{Message: "id is required"}
	}

	if err := s.store.Delete(ctx, id, userID); err != nil {
		s.logger.Error("Failed to delete aircraft", logging.WithField("error", err.Error()))
		return err
	}

	s.logger.Info("Deleted aircraft", logging.WithField("id", id))
	return nil
}

// List lists all aircraft for a user
func (s *Service) List(ctx context.Context, userID string, params models.AircraftListParams) (*models.AircraftListResponse, error) {
	return s.store.List(ctx, userID, params)
}

// GetDetails retrieves full aircraft details
func (s *Service) GetDetails(ctx context.Context, id string, userID string) (*models.AircraftDetailsResponse, error) {
	return s.store.GetDetails(ctx, id, userID)
}

// SetComponent assigns a component to an aircraft
// If newGear is provided, creates the inventory item first (auto-add gear feature)
func (s *Service) SetComponent(ctx context.Context, userID string, params models.SetComponentParams) (*models.AircraftComponent, error) {
	if params.AircraftID == "" {
		return nil, &ServiceError{Message: "aircraftId is required"}
	}
	if params.Category == "" {
		return nil, &ServiceError{Message: "category is required"}
	}

	// Verify the aircraft belongs to the user
	aircraft, err := s.store.Get(ctx, params.AircraftID, userID)
	if err != nil {
		return nil, err
	}
	if aircraft == nil {
		return nil, &ServiceError{Message: "aircraft not found"}
	}

	inventoryItemID := params.InventoryItemID

	// If newGear is provided, create the inventory item first (auto-add feature)
	if params.NewGear != nil {
		s.logger.Debug("Auto-adding new gear to inventory", logging.WithFields(map[string]interface{}{
			"name":     params.NewGear.Name,
			"category": params.NewGear.Category,
		}))

		// Map component category to equipment category if not set
		if params.NewGear.Category == "" {
			params.NewGear.Category = mapComponentToEquipmentCategory(params.Category)
		}

		newItem, err := s.inventorySvc.AddItem(ctx, userID, *params.NewGear)
		if err != nil {
			s.logger.Error("Failed to auto-add gear", logging.WithField("error", err.Error()))
			return nil, fmt.Errorf("failed to create inventory item: %w", err)
		}

		inventoryItemID = newItem.ID
		s.logger.Info("Auto-added gear to inventory", logging.WithFields(map[string]interface{}{
			"item_id": newItem.ID,
			"name":    newItem.Name,
		}))
	}

	// If still no inventory item ID, we're just removing the component
	if inventoryItemID == "" && params.NewGear == nil {
		if err := s.store.RemoveComponent(ctx, params.AircraftID, params.Category); err != nil {
			return nil, err
		}
		return nil, nil
	}

	// Verify the inventory item belongs to the user (if not just created)
	if params.NewGear == nil && inventoryItemID != "" {
		item, err := s.inventorySvc.GetItem(ctx, inventoryItemID, userID)
		if err != nil {
			return nil, err
		}
		if item == nil {
			return nil, &ServiceError{Message: "inventory item not found"}
		}
	}

	component, err := s.store.SetComponent(ctx, params.AircraftID, params.Category, inventoryItemID, params.Notes)
	if err != nil {
		s.logger.Error("Failed to set component", logging.WithField("error", err.Error()))
		return nil, err
	}

	s.logger.Info("Set aircraft component", logging.WithFields(map[string]interface{}{
		"aircraft_id": params.AircraftID,
		"category":    params.Category,
		"item_id":     inventoryItemID,
	}))

	return component, nil
}

// SetELRSSettings sets ELRS settings for an aircraft
func (s *Service) SetELRSSettings(ctx context.Context, userID string, params models.SetELRSSettingsParams) (*models.AircraftELRSSettings, error) {
	if params.AircraftID == "" {
		return nil, &ServiceError{Message: "aircraftId is required"}
	}

	// Verify the aircraft belongs to the user
	aircraft, err := s.store.Get(ctx, params.AircraftID, userID)
	if err != nil {
		return nil, err
	}
	if aircraft == nil {
		return nil, &ServiceError{Message: "aircraft not found"}
	}

	// Validate JSON
	if len(params.Settings) == 0 {
		params.Settings = json.RawMessage(`{}`)
	} else {
		var test interface{}
		if err := json.Unmarshal(params.Settings, &test); err != nil {
			return nil, &ServiceError{Message: "invalid JSON in settings"}
		}
	}

	settings, err := s.store.SetELRSSettings(ctx, params.AircraftID, params.Settings)
	if err != nil {
		s.logger.Error("Failed to set ELRS settings", logging.WithField("error", err.Error()))
		return nil, err
	}

	s.logger.Info("Set aircraft ELRS settings", logging.WithField("aircraft_id", params.AircraftID))
	return settings, nil
}

// GetELRSSettings retrieves ELRS settings for an aircraft
func (s *Service) GetELRSSettings(ctx context.Context, aircraftID string, userID string) (*models.AircraftELRSSettings, error) {
	// Verify the aircraft belongs to the user
	aircraft, err := s.store.Get(ctx, aircraftID, userID)
	if err != nil {
		return nil, err
	}
	if aircraft == nil {
		return nil, &ServiceError{Message: "aircraft not found"}
	}

	return s.store.GetELRSSettings(ctx, aircraftID)
}

// GetComponents retrieves all components for an aircraft
func (s *Service) GetComponents(ctx context.Context, aircraftID string, userID string) ([]models.AircraftComponent, error) {
	// Verify the aircraft belongs to the user
	aircraft, err := s.store.Get(ctx, aircraftID, userID)
	if err != nil {
		return nil, err
	}
	if aircraft == nil {
		return nil, &ServiceError{Message: "aircraft not found"}
	}

	return s.store.GetComponents(ctx, aircraftID)
}

// SetImage uploads an image for an aircraft
func (s *Service) SetImage(ctx context.Context, userID string, params models.SetAircraftImageParams) error {
	if params.AircraftID == "" {
		return &ServiceError{Message: "aircraftId is required"}
	}
	if len(params.ImageData) == 0 {
		return &ServiceError{Message: "image data is required"}
	}
	if params.ImageType != "image/jpeg" && params.ImageType != "image/png" {
		return &ServiceError{Message: "image must be JPEG or PNG"}
	}

	// Validate file size (max 5MB)
	const maxImageSize = 5 * 1024 * 1024
	if len(params.ImageData) > maxImageSize {
		return &ServiceError{Message: "image must be less than 5MB"}
	}

	// Verify the aircraft belongs to the user
	aircraft, err := s.store.Get(ctx, params.AircraftID, userID)
	if err != nil {
		return err
	}
	if aircraft == nil {
		return &ServiceError{Message: "aircraft not found"}
	}

	if err := s.store.SetImage(ctx, params.AircraftID, userID, params.ImageType, params.ImageData); err != nil {
		s.logger.Error("Failed to set aircraft image", logging.WithField("error", err.Error()))
		return err
	}

	s.logger.Info("Set aircraft image", logging.WithFields(map[string]interface{}{
		"aircraft_id": params.AircraftID,
		"size":        len(params.ImageData),
	}))
	return nil
}

// GetImage retrieves an aircraft's image
func (s *Service) GetImage(ctx context.Context, aircraftID string, userID string) ([]byte, string, error) {
	return s.store.GetImage(ctx, aircraftID, userID)
}

// DeleteImage removes an aircraft's image
func (s *Service) DeleteImage(ctx context.Context, aircraftID string, userID string) error {
	// Verify the aircraft belongs to the user
	aircraft, err := s.store.Get(ctx, aircraftID, userID)
	if err != nil {
		return err
	}
	if aircraft == nil {
		return &ServiceError{Message: "aircraft not found"}
	}

	return s.store.DeleteImage(ctx, aircraftID, userID)
}

// mapComponentToEquipmentCategory maps aircraft component category to equipment category
func mapComponentToEquipmentCategory(category models.ComponentCategory) models.EquipmentCategory {
	switch category {
	case models.ComponentCategoryFC:
		return models.CategoryFC
	case models.ComponentCategoryESC:
		return models.CategoryESC
	case models.ComponentCategoryELRSModule:
		return models.CategoryReceivers
	case models.ComponentCategoryVTX:
		return models.CategoryVTX
	case models.ComponentCategoryMotors:
		return models.CategoryMotors
	case models.ComponentCategoryCamera:
		return models.CategoryCameras
	case models.ComponentCategoryFrame:
		return models.CategoryFrames
	case models.ComponentCategoryProps:
		return models.CategoryPropellers
	case models.ComponentCategoryBattery:
		return models.CategoryBatteries
	case models.ComponentCategoryAntenna:
		return models.CategoryAntennas
	default:
		return models.CategoryAccessories
	}
}

// ServiceError represents a service-level error
type ServiceError struct {
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}

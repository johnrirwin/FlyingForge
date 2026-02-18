package aircraft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/johnrirwin/flyingforge/internal/database"
	"github.com/johnrirwin/flyingforge/internal/images"
	"github.com/johnrirwin/flyingforge/internal/inventory"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

// Service handles aircraft operations
type Service struct {
	store            *database.AircraftStore
	inventorySvc     inventory.InventoryManager
	gearCatalogStore *database.GearCatalogStore
	imageSvc         *images.Service
	logger           *logging.Logger
}

// NewService creates a new aircraft service
func NewService(store *database.AircraftStore, inventorySvc inventory.InventoryManager, gearCatalogStore *database.GearCatalogStore, imageSvc *images.Service, logger *logging.Logger) *Service {
	return &Service{
		store:            store,
		inventorySvc:     inventorySvc,
		gearCatalogStore: gearCatalogStore,
		imageSvc:         imageSvc,
		logger:           logger,
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
	params.Category = models.NormalizeComponentCategory(params.Category)

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

		// Contribute to the shared gear catalog (crowd-sourced feature)
		if s.gearCatalogStore != nil {
			specs := params.NewGear.Specs
			if specs == nil {
				specs = json.RawMessage(`{}`)
			}
			catalogItem, err := s.gearCatalogStore.MigrateInventoryItem(
				ctx,
				newItem.ID,
				userID,
				newItem.Name,
				newItem.Manufacturer,
				params.NewGear.Category,
				specs,
			)
			if err != nil {
				// Log but don't fail the operation - catalog contribution is best-effort
				s.logger.Warn("Failed to contribute gear to catalog", logging.WithFields(map[string]interface{}{
					"error":   err.Error(),
					"item_id": newItem.ID,
				}))
			} else {
				s.logger.Info("Contributed gear to catalog", logging.WithFields(map[string]interface{}{
					"item_id":    newItem.ID,
					"catalog_id": catalogItem.ID,
				}))
			}
		}
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

// SetReceiverSettings sets receiver settings for an aircraft
func (s *Service) SetReceiverSettings(ctx context.Context, userID string, params models.SetReceiverSettingsParams) (*models.AircraftReceiverSettings, error) {
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

	settings, err := s.store.SetReceiverSettings(ctx, params.AircraftID, params.Settings)
	if err != nil {
		s.logger.Error("Failed to set receiver settings", logging.WithField("error", err.Error()))
		return nil, err
	}

	s.logger.Info("Set aircraft receiver settings", logging.WithField("aircraft_id", params.AircraftID))
	return settings, nil
}

// GetReceiverSettings retrieves receiver settings for an aircraft
func (s *Service) GetReceiverSettings(ctx context.Context, aircraftID string, userID string) (*models.AircraftReceiverSettings, error) {
	// Verify the aircraft belongs to the user
	aircraft, err := s.store.Get(ctx, aircraftID, userID)
	if err != nil {
		return nil, err
	}
	if aircraft == nil {
		return nil, &ServiceError{Message: "aircraft not found"}
	}

	return s.store.GetReceiverSettings(ctx, aircraftID)
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
func (s *Service) SetImage(ctx context.Context, userID string, params models.SetAircraftImageParams) (*models.ModerationDecision, error) {
	if params.AircraftID == "" {
		return nil, &ServiceError{Message: "aircraftId is required"}
	}
	if s.imageSvc == nil {
		return nil, &ServiceError{Message: "image moderation unavailable"}
	}

	// Verify the aircraft belongs to the user
	aircraft, err := s.store.Get(ctx, params.AircraftID, userID)
	if err != nil {
		return nil, err
	}
	if aircraft == nil {
		return nil, &ServiceError{Message: "aircraft not found"}
	}

	var (
		decision *models.ModerationDecision
		asset    *models.ImageAsset
	)

	uploadID := strings.TrimSpace(params.UploadID)
	if uploadID != "" {
		asset, err = s.imageSvc.PersistApprovedUpload(ctx, userID, uploadID, models.ImageEntityAircraft, params.AircraftID)
		if err != nil {
			return nil, err
		}
		decision = &models.ModerationDecision{
			Status: models.ImageModerationApproved,
			Reason: "Approved",
		}
		if params.ImageType == "" {
			params.ImageType = http.DetectContentType(asset.ImageBytes)
		}
	} else {
		if len(params.ImageData) == 0 {
			return nil, &ServiceError{Message: "image data is required"}
		}
		if params.ImageType != "image/jpeg" && params.ImageType != "image/png" {
			return nil, &ServiceError{Message: "image must be JPEG or PNG"}
		}

		// Validate file size (max 2MB)
		const maxImageSize = 2 * 1024 * 1024
		if len(params.ImageData) > maxImageSize {
			return nil, &ServiceError{Message: "image must be less than 2MB"}
		}

		decision, asset, err = s.imageSvc.ModerateAndPersist(ctx, images.SaveRequest{
			OwnerUserID: userID,
			EntityType:  models.ImageEntityAircraft,
			EntityID:    params.AircraftID,
			ImageBytes:  params.ImageData,
		})
		if err != nil {
			s.logger.Error("Failed to moderate aircraft image", logging.WithField("error", err.Error()))
			return nil, err
		}
		if decision.Status != models.ImageModerationApproved {
			return decision, nil
		}
	}

	if params.ImageType == "" {
		params.ImageType = http.DetectContentType(asset.ImageBytes)
	}
	if params.ImageType == "" {
		params.ImageType = "application/octet-stream"
	}

	previousAssetID, err := s.store.SetImage(ctx, params.AircraftID, userID, params.ImageType, asset.ID)
	if err != nil {
		s.logger.Error("Failed to set aircraft image", logging.WithField("error", err.Error()))
		_ = s.imageSvc.Delete(ctx, asset.ID)
		return nil, err
	}
	if previousAssetID != "" && previousAssetID != asset.ID {
		_ = s.imageSvc.Delete(ctx, previousAssetID)
	}

	s.logger.Info("Set aircraft image", logging.WithFields(map[string]interface{}{
		"aircraft_id": params.AircraftID,
		"size":        len(asset.ImageBytes),
	}))
	return decision, nil
}

// GetImage retrieves an aircraft's image
func (s *Service) GetImage(ctx context.Context, aircraftID string, userID string) ([]byte, string, error) {
	imageData, imageType, err := s.store.GetImage(ctx, aircraftID, userID)
	if err != nil {
		return nil, "", err
	}
	if len(imageData) == 0 {
		return imageData, imageType, nil
	}
	if imageType == "" {
		imageType = http.DetectContentType(imageData)
	}
	return imageData, imageType, nil
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

	previousAssetID, err := s.store.DeleteImage(ctx, aircraftID, userID)
	if err != nil {
		return err
	}
	if previousAssetID != "" {
		_ = s.imageSvc.Delete(ctx, previousAssetID)
	}
	return nil
}

// mapComponentToEquipmentCategory maps aircraft component category to equipment category
func mapComponentToEquipmentCategory(category models.ComponentCategory) models.EquipmentCategory {
	switch models.NormalizeComponentCategory(category) {
	case models.ComponentCategoryFC:
		return models.CategoryFC
	case models.ComponentCategoryESC:
		return models.CategoryESC
	case models.ComponentCategoryReceiver:
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

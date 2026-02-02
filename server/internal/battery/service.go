package battery

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/johnrirwin/rotorlife/internal/database"
	"github.com/johnrirwin/rotorlife/internal/logging"
	"github.com/johnrirwin/rotorlife/internal/models"
)

// ServiceError represents a service-level error
type ServiceError struct {
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}

// Store defines the interface for battery storage operations
type Store interface {
	BatteryCodeExists(ctx context.Context, userID, code string) (bool, error)
	Create(ctx context.Context, userID, batteryCode string, params models.CreateBatteryParams) (*models.Battery, error)
	Get(ctx context.Context, id, userID string) (*models.Battery, error)
	GetByCode(ctx context.Context, code, userID string) (*models.Battery, error)
	Update(ctx context.Context, userID string, params models.UpdateBatteryParams) (*models.Battery, error)
	Delete(ctx context.Context, id, userID string) error
	List(ctx context.Context, userID string, params models.BatteryListParams) (*models.BatteryListResponse, error)
	CreateLog(ctx context.Context, userID string, params models.CreateBatteryLogParams) (*models.BatteryLog, error)
	ListLogs(ctx context.Context, batteryID, userID string, limit int) (*models.BatteryLogListResponse, error)
	DeleteLog(ctx context.Context, logID, userID string) error
}

// Service handles battery operations
type Service struct {
	store  Store
	logger *logging.Logger
}

// NewService creates a new battery service
func NewService(store *database.BatteryStore, logger *logging.Logger) *Service {
	return &Service{
		store:  store,
		logger: logger,
	}
}

// generateBatteryCode generates a unique battery code for a user
func (s *Service) generateBatteryCode(ctx context.Context, userID string) (string, error) {
	for attempts := 0; attempts < 10; attempts++ {
		code := s.randomCode()
		exists, err := s.store.BatteryCodeExists(ctx, userID, code)
		if err != nil {
			return "", fmt.Errorf("failed to check code existence: %w", err)
		}
		if !exists {
			return code, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique battery code after 10 attempts")
}

// randomCode generates a random alphanumeric code like "BAT-A1B2"
func (s *Service) randomCode() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	// Use base32 encoding (A-Z, 2-7) and take first 4 characters
	code := base32.StdEncoding.EncodeToString(b)[:4]
	return "BAT-" + strings.ToUpper(code)
}

// Create creates a new battery
func (s *Service) Create(ctx context.Context, userID string, params models.CreateBatteryParams) (*models.Battery, error) {
	// Validate
	if err := s.validateCreateParams(params); err != nil {
		return nil, err
	}

	// Generate unique battery code
	code, err := s.generateBatteryCode(ctx, userID)
	if err != nil {
		return nil, err
	}

	s.logger.Debug("Creating battery", logging.WithFields(map[string]interface{}{
		"user_id":   userID,
		"code":      code,
		"chemistry": params.Chemistry,
		"cells":     params.Cells,
	}))

	battery, err := s.store.Create(ctx, userID, code, params)
	if err != nil {
		s.logger.Error("Failed to create battery", logging.WithField("error", err.Error()))
		return nil, err
	}

	s.logger.Info("Created battery", logging.WithFields(map[string]interface{}{
		"id":   battery.ID,
		"code": battery.BatteryCode,
	}))
	return battery, nil
}

func (s *Service) validateCreateParams(params models.CreateBatteryParams) error {
	if !models.IsValidChemistry(params.Chemistry) {
		return &ServiceError{Message: "invalid chemistry type"}
	}
	if params.Cells < 1 || params.Cells > 8 {
		return &ServiceError{Message: "cells must be between 1 and 8"}
	}
	if params.CapacityMah <= 0 {
		return &ServiceError{Message: "capacity must be greater than 0"}
	}
	return nil
}

// Get retrieves a battery by ID
func (s *Service) Get(ctx context.Context, id string, userID string) (*models.Battery, error) {
	return s.store.Get(ctx, id, userID)
}

// GetByCode retrieves a battery by battery code
func (s *Service) GetByCode(ctx context.Context, code string, userID string) (*models.Battery, error) {
	return s.store.GetByCode(ctx, code, userID)
}

// Update updates a battery
func (s *Service) Update(ctx context.Context, userID string, params models.UpdateBatteryParams) (*models.Battery, error) {
	if params.ID == "" {
		return nil, &ServiceError{Message: "id is required"}
	}

	// Validate fields if provided
	if params.Chemistry != nil && !models.IsValidChemistry(*params.Chemistry) {
		return nil, &ServiceError{Message: "invalid chemistry type"}
	}
	if params.Cells != nil && (*params.Cells < 1 || *params.Cells > 8) {
		return nil, &ServiceError{Message: "cells must be between 1 and 8"}
	}
	if params.CapacityMah != nil && *params.CapacityMah <= 0 {
		return nil, &ServiceError{Message: "capacity must be greater than 0"}
	}

	battery, err := s.store.Update(ctx, userID, params)
	if err != nil {
		s.logger.Error("Failed to update battery", logging.WithField("error", err.Error()))
		return nil, err
	}

	if battery == nil {
		return nil, &ServiceError{Message: "battery not found"}
	}

	s.logger.Info("Updated battery", logging.WithField("id", battery.ID))
	return battery, nil
}

// Delete deletes a battery
func (s *Service) Delete(ctx context.Context, id string, userID string) error {
	if id == "" {
		return &ServiceError{Message: "id is required"}
	}

	if err := s.store.Delete(ctx, id, userID); err != nil {
		s.logger.Error("Failed to delete battery", logging.WithField("error", err.Error()))
		return err
	}

	s.logger.Info("Deleted battery", logging.WithField("id", id))
	return nil
}

// List lists all batteries for a user
func (s *Service) List(ctx context.Context, userID string, params models.BatteryListParams) (*models.BatteryListResponse, error) {
	return s.store.List(ctx, userID, params)
}

// GetDetails retrieves a battery with its logs
func (s *Service) GetDetails(ctx context.Context, id string, userID string) (*models.BatteryDetailsResponse, error) {
	battery, err := s.store.Get(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if battery == nil {
		return nil, nil
	}

	logsResp, err := s.store.ListLogs(ctx, id, userID, 50)
	if err != nil {
		return nil, err
	}

	return &models.BatteryDetailsResponse{
		Battery: *battery,
		Logs:    logsResp.Logs,
	}, nil
}

// CreateLog creates a new battery log entry
func (s *Service) CreateLog(ctx context.Context, userID string, params models.CreateBatteryLogParams) (*models.BatteryLog, error) {
	if params.BatteryID == "" {
		return nil, &ServiceError{Message: "batteryId is required"}
	}

	// Validate IR array length if provided
	if params.IRMohmPerCell != nil {
		// Get battery to check cell count
		battery, err := s.store.Get(ctx, params.BatteryID, userID)
		if err != nil {
			return nil, err
		}
		if battery == nil {
			return nil, &ServiceError{Message: "battery not found"}
		}

		// Parse IR array and check length
		var irValues []float64
		if err := json.Unmarshal(params.IRMohmPerCell, &irValues); err == nil {
			if len(irValues) != battery.Cells {
				return nil, &ServiceError{Message: fmt.Sprintf("IR array length (%d) must match cell count (%d)", len(irValues), battery.Cells)}
			}
		}
	}

	log, err := s.store.CreateLog(ctx, userID, params)
	if err != nil {
		s.logger.Error("Failed to create battery log", logging.WithField("error", err.Error()))
		return nil, err
	}

	s.logger.Info("Created battery log", logging.WithFields(map[string]interface{}{
		"log_id":     log.ID,
		"battery_id": log.BatteryID,
	}))
	return log, nil
}

// ListLogs lists logs for a battery
func (s *Service) ListLogs(ctx context.Context, batteryID string, userID string, limit int) (*models.BatteryLogListResponse, error) {
	return s.store.ListLogs(ctx, batteryID, userID, limit)
}

// DeleteLog deletes a battery log entry
func (s *Service) DeleteLog(ctx context.Context, logID string, userID string) error {
	if logID == "" {
		return &ServiceError{Message: "logId is required"}
	}

	if err := s.store.DeleteLog(ctx, logID, userID); err != nil {
		s.logger.Error("Failed to delete battery log", logging.WithField("error", err.Error()))
		return err
	}

	s.logger.Info("Deleted battery log", logging.WithField("id", logID))
	return nil
}

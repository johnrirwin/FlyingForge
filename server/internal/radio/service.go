package radio

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/johnrirwin/flyingforge/internal/database"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/models"
)

const (
	// MaxBackupFileSize is the maximum allowed backup file size (100MB)
	MaxBackupFileSize = 100 * 1024 * 1024
	// DefaultStorageDir is the default directory for backup storage
	DefaultStorageDir = "./data/radio_backups"
)

// ServiceError represents a service-level error
type ServiceError struct {
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}

// Service handles radio operations
type Service struct {
	store      *database.RadioStore
	storageDir string
	logger     *logging.Logger
}

// NewService creates a new radio service
func NewService(store *database.RadioStore, storageDir string, logger *logging.Logger) *Service {
	if storageDir == "" {
		storageDir = DefaultStorageDir
	}
	return &Service{
		store:      store,
		storageDir: storageDir,
		logger:     logger,
	}
}

// GetRadioModels returns the list of available radio models
func (s *Service) GetRadioModels(ctx context.Context) *models.RadioModelsResponse {
	return &models.RadioModelsResponse{
		Models: models.GetRadioModels(),
	}
}

// CreateRadio creates a new radio for a user
func (s *Service) CreateRadio(ctx context.Context, userID string, params models.CreateRadioParams) (*models.Radio, error) {
	if params.Manufacturer == "" {
		return nil, &ServiceError{Message: "manufacturer is required"}
	}
	if params.Model == "" {
		return nil, &ServiceError{Message: "model is required"}
	}

	s.logger.Debug("Creating radio", logging.WithFields(map[string]interface{}{
		"user_id":      userID,
		"manufacturer": params.Manufacturer,
		"model":        params.Model,
	}))

	radio, err := s.store.CreateRadio(ctx, userID, params)
	if err != nil {
		s.logger.Error("Failed to create radio", logging.WithField("error", err.Error()))
		return nil, err
	}

	s.logger.Info("Created radio", logging.WithField("id", radio.ID))
	return radio, nil
}

// GetRadio retrieves a radio by ID
func (s *Service) GetRadio(ctx context.Context, id string, userID string) (*models.Radio, error) {
	return s.store.GetRadio(ctx, id, userID)
}

// ListRadios lists radios for a user
func (s *Service) ListRadios(ctx context.Context, userID string, params models.RadioListParams) (*models.RadioListResponse, error) {
	return s.store.ListRadios(ctx, userID, params)
}

// UpdateRadio updates a radio
func (s *Service) UpdateRadio(ctx context.Context, id string, userID string, params models.UpdateRadioParams) (*models.Radio, error) {
	s.logger.Debug("Updating radio", logging.WithField("id", id))

	radio, err := s.store.UpdateRadio(ctx, id, userID, params)
	if err != nil {
		s.logger.Error("Failed to update radio", logging.WithFields(map[string]interface{}{
			"id":    id,
			"error": err.Error(),
		}))
		return nil, err
	}

	s.logger.Info("Updated radio", logging.WithField("id", id))
	return radio, nil
}

// DeleteRadio deletes a radio and all its backups
func (s *Service) DeleteRadio(ctx context.Context, id string, userID string) error {
	s.logger.Debug("Deleting radio", logging.WithField("id", id))

	// First, delete all backup files
	backups, err := s.store.ListBackups(ctx, id, models.RadioBackupListParams{Limit: 1000})
	if err != nil {
		s.logger.Warn("Failed to list backups for deletion", logging.WithField("error", err.Error()))
	} else {
		for _, backup := range backups.Backups {
			if backup.StoragePath != "" {
				if err := os.Remove(backup.StoragePath); err != nil && !os.IsNotExist(err) {
					s.logger.Warn("Failed to delete backup file", logging.WithFields(map[string]interface{}{
						"path":  backup.StoragePath,
						"error": err.Error(),
					}))
				}
			}
		}
	}

	// Delete the radio directory if it exists
	radioDir := filepath.Join(s.storageDir, id)
	if err := os.RemoveAll(radioDir); err != nil && !os.IsNotExist(err) {
		s.logger.Warn("Failed to delete radio directory", logging.WithField("error", err.Error()))
	}

	// Delete the radio record (cascades to backups in DB)
	if err := s.store.DeleteRadio(ctx, id, userID); err != nil {
		s.logger.Error("Failed to delete radio", logging.WithFields(map[string]interface{}{
			"id":    id,
			"error": err.Error(),
		}))
		return err
	}

	s.logger.Info("Deleted radio", logging.WithField("id", id))
	return nil
}

// CreateBackup creates a backup record and stores the file
func (s *Service) CreateBackup(ctx context.Context, radioID string, userID string, params models.CreateRadioBackupParams, fileReader io.Reader) (*models.RadioBackup, error) {
	// Validate parameters
	if params.BackupName == "" {
		return nil, &ServiceError{Message: "backup name is required"}
	}
	if params.FileName == "" {
		return nil, &ServiceError{Message: "file name is required"}
	}
	if params.BackupType == "" {
		params.BackupType = models.BackupTypeOther
	}
	if params.FileSize > MaxBackupFileSize {
		return nil, &ServiceError{Message: fmt.Sprintf("file size exceeds maximum allowed (%d bytes)", MaxBackupFileSize)}
	}

	// Verify the radio exists and belongs to user
	radio, err := s.store.GetRadio(ctx, radioID, userID)
	if err != nil {
		return nil, err
	}
	if radio == nil {
		return nil, &ServiceError{Message: "radio not found"}
	}

	// Create storage directory
	radioDir := filepath.Join(s.storageDir, radioID)
	if err := os.MkdirAll(radioDir, 0755); err != nil {
		s.logger.Error("Failed to create storage directory", logging.WithField("error", err.Error()))
		return nil, &ServiceError{Message: "failed to create storage directory"}
	}

	// Sanitize filename and create storage path
	safeFileName := sanitizeFileName(params.FileName)
	storagePath := filepath.Join(radioDir, safeFileName)

	// Ensure unique filename
	storagePath = ensureUniquePath(storagePath)

	// Create the file
	file, err := os.Create(storagePath)
	if err != nil {
		s.logger.Error("Failed to create backup file", logging.WithField("error", err.Error()))
		return nil, &ServiceError{Message: "failed to create backup file"}
	}
	defer file.Close()

	// Copy file content and calculate checksum
	// Enforce max size while copying by using LimitReader
	// We add 1 to the limit to detect if the stream exceeds the max size
	hasher := sha256.New()
	limitedReader := io.LimitReader(fileReader, MaxBackupFileSize+1)
	teeReader := io.TeeReader(limitedReader, hasher)

	written, err := io.Copy(file, teeReader)
	if err != nil {
		os.Remove(storagePath)
		s.logger.Error("Failed to write backup file", logging.WithField("error", err.Error()))
		return nil, &ServiceError{Message: "failed to write backup file"}
	}

	// Verify actual written size doesn't exceed limit
	if written > MaxBackupFileSize {
		os.Remove(storagePath)
		return nil, &ServiceError{Message: fmt.Sprintf("actual file size (%d bytes) exceeds maximum allowed (%d bytes)", written, MaxBackupFileSize)}
	}

	// Update params with actual values
	params.FileSize = written
	params.Checksum = hex.EncodeToString(hasher.Sum(nil))

	s.logger.Debug("Creating backup record", logging.WithFields(map[string]interface{}{
		"radio_id":    radioID,
		"backup_name": params.BackupName,
		"file_name":   params.FileName,
		"file_size":   params.FileSize,
	}))

	// Create the database record
	backup, err := s.store.CreateBackup(ctx, radioID, params, storagePath)
	if err != nil {
		os.Remove(storagePath)
		s.logger.Error("Failed to create backup record", logging.WithField("error", err.Error()))
		return nil, err
	}

	s.logger.Info("Created backup", logging.WithField("id", backup.ID))
	return backup, nil
}

// ListBackups lists backups for a radio
func (s *Service) ListBackups(ctx context.Context, radioID string, userID string, params models.RadioBackupListParams) (*models.RadioBackupListResponse, error) {
	// Verify the radio belongs to user
	radio, err := s.store.GetRadio(ctx, radioID, userID)
	if err != nil {
		return nil, err
	}
	if radio == nil {
		return nil, &ServiceError{Message: "radio not found"}
	}

	return s.store.ListBackups(ctx, radioID, params)
}

// GetBackup retrieves a backup by ID
func (s *Service) GetBackup(ctx context.Context, backupID string, radioID string, userID string) (*models.RadioBackup, error) {
	// Verify the radio belongs to user
	radio, err := s.store.GetRadio(ctx, radioID, userID)
	if err != nil {
		return nil, err
	}
	if radio == nil {
		return nil, &ServiceError{Message: "radio not found"}
	}

	return s.store.GetBackup(ctx, backupID, radioID)
}

// GetBackupFile returns a reader for a backup file
func (s *Service) GetBackupFile(ctx context.Context, backupID string, radioID string, userID string) (io.ReadCloser, *models.RadioBackup, error) {
	backup, err := s.GetBackup(ctx, backupID, radioID, userID)
	if err != nil {
		return nil, nil, err
	}
	if backup == nil {
		return nil, nil, &ServiceError{Message: "backup not found"}
	}

	file, err := os.Open(backup.StoragePath)
	if err != nil {
		s.logger.Error("Failed to open backup file", logging.WithFields(map[string]interface{}{
			"path":  backup.StoragePath,
			"error": err.Error(),
		}))
		return nil, nil, &ServiceError{Message: "backup file not found"}
	}

	return file, backup, nil
}

// DeleteBackup deletes a backup and its file
func (s *Service) DeleteBackup(ctx context.Context, backupID string, radioID string, userID string) error {
	// Verify the radio belongs to user
	radio, err := s.store.GetRadio(ctx, radioID, userID)
	if err != nil {
		return err
	}
	if radio == nil {
		return &ServiceError{Message: "radio not found"}
	}

	s.logger.Debug("Deleting backup", logging.WithField("id", backupID))

	// Delete from database (returns the backup with storage path)
	backup, err := s.store.DeleteBackup(ctx, backupID, radioID)
	if err != nil {
		s.logger.Error("Failed to delete backup", logging.WithFields(map[string]interface{}{
			"id":    backupID,
			"error": err.Error(),
		}))
		return err
	}

	// Delete the file
	if backup.StoragePath != "" {
		if err := os.Remove(backup.StoragePath); err != nil && !os.IsNotExist(err) {
			s.logger.Warn("Failed to delete backup file", logging.WithFields(map[string]interface{}{
				"path":  backup.StoragePath,
				"error": err.Error(),
			}))
		}
	}

	s.logger.Info("Deleted backup", logging.WithField("id", backupID))
	return nil
}

// Helper functions

// sanitizeFileName removes unsafe characters from a filename
func sanitizeFileName(filename string) string {
	// Remove path separators and other dangerous characters
	re := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	safe := re.ReplaceAllString(filename, "_")

	// Limit length
	if len(safe) > 200 {
		ext := filepath.Ext(safe)
		base := strings.TrimSuffix(safe, ext)
		if len(base) > 200-len(ext) {
			base = base[:200-len(ext)]
		}
		safe = base + ext
	}

	return safe
}

// ensureUniquePath adds a suffix if the path already exists
func ensureUniquePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}

	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)

	for i := 1; i < 1000; i++ {
		newPath := fmt.Sprintf("%s_%d%s", base, i, ext)
		if _, err := os.Stat(newPath); os.IsNotExist(err) {
			return newPath
		}
	}

	// Fallback: use timestamp
	return fmt.Sprintf("%s_%d%s", base, os.Getpid(), ext)
}

package images

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/johnrirwin/flyingforge/internal/models"
)

var (
	// ErrPendingUploadNotFound is returned when a pending upload token is missing/expired.
	ErrPendingUploadNotFound = errors.New("approved upload not found")
	// ErrUploadNotApproved is returned when trying to persist a non-approved upload token.
	ErrUploadNotApproved = errors.New("upload is not approved")
)

// Moderator defines the moderation abstraction used by image flows.
type Moderator interface {
	ModerateImageBytes(ctx context.Context, imageBytes []byte) (*models.ModerationDecision, error)
}

// SaveRequest defines a single image save operation.
type SaveRequest struct {
	OwnerUserID             string
	EntityType              models.ImageEntityType
	EntityID                string
	ImageBytes              []byte
	ModerationLabels        []models.ModerationLabel
	ModerationMaxConfidence float64
}

// Storage abstracts image persistence so DB storage can later be swapped for S3.
type Storage interface {
	Save(ctx context.Context, req SaveRequest) (*models.ImageAsset, error)
	Load(ctx context.Context, imageID string) (*models.ImageAsset, error)
	Delete(ctx context.Context, imageID string) error
}

// PendingUpload is an approved but not-yet-persisted image token.
type PendingUpload struct {
	ID          string
	OwnerUserID string
	EntityType  models.ImageEntityType
	ImageBytes  []byte
	Decision    models.ModerationDecision
	ExpiresAt   time.Time
}

// PendingStore tracks approved uploads that still require explicit Save.
type PendingStore interface {
	Put(upload PendingUpload) string
	Get(ownerUserID, uploadID string) (*PendingUpload, bool)
	Delete(uploadID string)
}

// Service orchestrates moderation + storage flow for uploads.
type Service struct {
	moderator Moderator
	storage   Storage
	pending   PendingStore
	timeout   time.Duration
}

// NewService creates a new image pipeline service.
func NewService(moderator Moderator, storage Storage, pending PendingStore, timeout time.Duration) *Service {
	return &Service{
		moderator: moderator,
		storage:   storage,
		pending:   pending,
		timeout:   timeout,
	}
}

// ModerateUpload runs synchronous moderation and, if approved, stores a pending token.
func (s *Service) ModerateUpload(ctx context.Context, ownerUserID string, entityType models.ImageEntityType, imageBytes []byte) (*models.ModerationDecision, string, error) {
	decision := s.moderate(ctx, imageBytes)
	if decision.Status != models.ImageModerationApproved {
		return decision, "", nil
	}
	if s.pending == nil {
		return &models.ModerationDecision{
			Status: models.ImageModerationPendingReview,
			Reason: "Unable to verify right now",
		}, "", nil
	}

	uploadID := s.pending.Put(PendingUpload{
		OwnerUserID: ownerUserID,
		EntityType:  entityType,
		ImageBytes:  imageBytes,
		Decision:    *decision,
	})
	if strings.TrimSpace(uploadID) == "" {
		return &models.ModerationDecision{
			Status: models.ImageModerationPendingReview,
			Reason: "Unable to verify right now",
		}, "", nil
	}

	return decision, uploadID, nil
}

// ModerateAndPersist runs moderation and immediately persists approved images.
func (s *Service) ModerateAndPersist(ctx context.Context, req SaveRequest) (*models.ModerationDecision, *models.ImageAsset, error) {
	decision := s.moderate(ctx, req.ImageBytes)
	if decision.Status != models.ImageModerationApproved {
		return decision, nil, nil
	}

	req.ModerationLabels = decision.Labels
	req.ModerationMaxConfidence = decision.MaxConfidence
	asset, err := s.storage.Save(ctx, req)
	if err != nil {
		return decision, nil, err
	}

	return decision, asset, nil
}

// PersistApprovedUpload stores a previously approved pending upload.
func (s *Service) PersistApprovedUpload(ctx context.Context, ownerUserID, uploadID string, entityType models.ImageEntityType, entityID string) (*models.ImageAsset, error) {
	if s.pending == nil {
		return nil, ErrPendingUploadNotFound
	}

	pendingUpload, ok := s.pending.Get(ownerUserID, uploadID)
	if !ok {
		return nil, ErrPendingUploadNotFound
	}
	if pendingUpload.Decision.Status != models.ImageModerationApproved {
		return nil, ErrUploadNotApproved
	}
	if pendingUpload.EntityType != entityType {
		return nil, ErrUploadNotApproved
	}

	asset, err := s.storage.Save(ctx, SaveRequest{
		OwnerUserID:             ownerUserID,
		EntityType:              entityType,
		EntityID:                entityID,
		ImageBytes:              pendingUpload.ImageBytes,
		ModerationLabels:        pendingUpload.Decision.Labels,
		ModerationMaxConfidence: pendingUpload.Decision.MaxConfidence,
	})
	if err != nil {
		return nil, err
	}

	s.pending.Delete(uploadID)
	return asset, nil
}

// Load proxies image loading to the configured storage backend.
func (s *Service) Load(ctx context.Context, imageID string) (*models.ImageAsset, error) {
	return s.storage.Load(ctx, imageID)
}

// Delete proxies image deletion to the configured storage backend.
func (s *Service) Delete(ctx context.Context, imageID string) error {
	return s.storage.Delete(ctx, imageID)
}

func (s *Service) moderate(ctx context.Context, imageBytes []byte) *models.ModerationDecision {
	timeout := s.timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	moderationCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	decision, err := s.moderator.ModerateImageBytes(moderationCtx, imageBytes)
	if err != nil || decision == nil {
		return &models.ModerationDecision{
			Status: models.ImageModerationPendingReview,
			Reason: "Unable to verify right now",
		}
	}

	return decision
}

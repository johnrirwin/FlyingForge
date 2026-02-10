package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/johnrirwin/flyingforge/internal/images"
	"github.com/johnrirwin/flyingforge/internal/models"
)

// ImageAssetStore persists moderated image assets in PostgreSQL.
type ImageAssetStore struct {
	db *DB
}

// NewImageAssetStore creates a new image asset store.
func NewImageAssetStore(db *DB) *ImageAssetStore {
	return &ImageAssetStore{db: db}
}

// Save stores approved image bytes and moderation metadata.
func (s *ImageAssetStore) Save(ctx context.Context, req images.SaveRequest) (*models.ImageAsset, error) {
	if len(req.ImageBytes) == 0 {
		return nil, fmt.Errorf("image bytes are required")
	}
	if req.OwnerUserID == "" {
		return nil, fmt.Errorf("owner user id is required")
	}
	if req.EntityType == "" {
		req.EntityType = models.ImageEntityOther
	}

	labelsJSON, err := json.Marshal(req.ModerationLabels)
	if err != nil {
		return nil, fmt.Errorf("marshal moderation labels: %w", err)
	}

	entityID := sql.NullString{}
	if req.EntityID != "" {
		entityID = sql.NullString{String: req.EntityID, Valid: true}
	}

	query := `
		INSERT INTO image_assets (
			owner_user_id,
			entity_type,
			entity_id,
			image_bytes,
			status,
			moderation_labels,
			moderation_max_confidence
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, owner_user_id, entity_type, entity_id, image_bytes, status, moderation_labels, moderation_max_confidence, created_at, updated_at
	`

	var asset models.ImageAsset
	var status string
	var scanEntityID sql.NullString
	err = s.db.QueryRowContext(
		ctx,
		query,
		req.OwnerUserID,
		string(req.EntityType),
		entityID,
		req.ImageBytes,
		string(models.ImageModerationApproved),
		labelsJSON,
		req.ModerationMaxConfidence,
	).Scan(
		&asset.ID,
		&asset.OwnerUserID,
		&asset.EntityType,
		&scanEntityID,
		&asset.ImageBytes,
		&status,
		&asset.ModerationLabels,
		&asset.ModerationMaxConfidence,
		&asset.CreatedAt,
		&asset.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("save image asset: %w", err)
	}
	asset.Status = models.ImageModerationStatus(status)
	if scanEntityID.Valid {
		asset.EntityID = scanEntityID.String
	}

	return &asset, nil
}

// Load retrieves an image asset by ID.
func (s *ImageAssetStore) Load(ctx context.Context, imageID string) (*models.ImageAsset, error) {
	query := `
		SELECT id, owner_user_id, entity_type, entity_id, image_bytes, status, moderation_labels, moderation_max_confidence, created_at, updated_at
		FROM image_assets
		WHERE id = $1
	`

	var asset models.ImageAsset
	var status string
	var scanEntityID sql.NullString
	err := s.db.QueryRowContext(ctx, query, imageID).Scan(
		&asset.ID,
		&asset.OwnerUserID,
		&asset.EntityType,
		&scanEntityID,
		&asset.ImageBytes,
		&status,
		&asset.ModerationLabels,
		&asset.ModerationMaxConfidence,
		&asset.CreatedAt,
		&asset.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load image asset: %w", err)
	}

	asset.Status = models.ImageModerationStatus(status)
	if scanEntityID.Valid {
		asset.EntityID = scanEntityID.String
	}

	return &asset, nil
}

// Delete removes an image asset by ID.
func (s *ImageAssetStore) Delete(ctx context.Context, imageID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM image_assets WHERE id = $1`, imageID)
	if err != nil {
		return fmt.Errorf("delete image asset: %w", err)
	}
	return nil
}

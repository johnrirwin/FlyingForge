package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/johnrirwin/flyingforge/internal/models"
)

// BuildStore handles build persistence.
type BuildStore struct {
	db *DB
}

// NewBuildStore creates a new build store.
func NewBuildStore(db *DB) *BuildStore {
	return &BuildStore{db: db}
}

// Create inserts a build and optional parts.
func (s *BuildStore) Create(
	ctx context.Context,
	ownerUserID string,
	status models.BuildStatus,
	title string,
	description string,
	youtubeURL string,
	flightYouTubeURL string,
	sourceAircraftID string,
	token string,
	expiresAt *time.Time,
	parts []models.BuildPartInput,
) (*models.Build, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO builds (owner_user_id, status, token, expires_at, title, description, build_video_url, flight_video_url, source_aircraft_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	var buildID string
	var expiresArg interface{}
	if expiresAt != nil {
		expiresArg = *expiresAt
	}

	err = tx.QueryRowContext(
		ctx,
		query,
		nullString(ownerUserID),
		status,
		nullString(token),
		expiresArg,
		title,
		nullString(description),
		nullString(youtubeURL),
		nullString(flightYouTubeURL),
		nullString(sourceAircraftID),
	).Scan(&buildID)
	if err != nil {
		return nil, fmt.Errorf("failed to create build: %w", err)
	}

	if err := s.replacePartsTx(ctx, tx, buildID, parts); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit build create: %w", err)
	}

	return s.GetByID(ctx, buildID)
}

// ListByOwner returns non-temp builds for an owner.
func (s *BuildStore) ListByOwner(ctx context.Context, ownerUserID string, params models.BuildListParams) (*models.BuildListResponse, error) {
	if params.Limit <= 0 {
		params.Limit = 50
	}
	if params.Limit > 100 {
		params.Limit = 100
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	countQuery := `
		SELECT COUNT(*)
		FROM builds b
		WHERE b.owner_user_id = $1
		  AND b.revision_of_build_id IS NULL
		  AND b.status IN ('DRAFT', 'PENDING_REVIEW', 'PUBLISHED', 'UNPUBLISHED')
	`
	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, ownerUserID).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count owner builds: %w", err)
	}

	query := `
		SELECT
			b.id,
			b.owner_user_id,
			COALESCE(
				CASE
					WHEN b.status = 'PUBLISHED' THEN r.image_asset_id
					ELSE NULL
				END,
				b.image_asset_id
			) AS image_asset_id,
			b.status,
			b.token,
			b.expires_at,
			COALESCE(
				CASE
					WHEN b.status = 'PUBLISHED' THEN r.title
					ELSE NULL
				END,
				b.title
			) AS title,
			COALESCE(
				CASE
					WHEN b.status = 'PUBLISHED' THEN r.description
					ELSE NULL
				END,
				b.description
			) AS description,
			COALESCE(
				CASE
					WHEN b.status = 'PUBLISHED' THEN r.build_video_url
					ELSE NULL
				END,
				b.build_video_url
			) AS build_video_url,
			COALESCE(
				CASE
					WHEN b.status = 'PUBLISHED' THEN r.flight_video_url
					ELSE NULL
				END,
				b.flight_video_url
			) AS flight_video_url,
			COALESCE(
				CASE
					WHEN b.status = 'PUBLISHED' THEN r.source_aircraft_id
				ELSE NULL
				END,
				b.source_aircraft_id
			) AS source_aircraft_id,
			COALESCE(
				CASE
					WHEN b.status = 'PUBLISHED' THEN r.moderation_reason
					ELSE NULL
				END,
				b.moderation_reason
			) AS moderation_reason,
			b.created_at,
			COALESCE(
				CASE
					WHEN b.status = 'PUBLISHED' THEN r.updated_at
					ELSE NULL
				END,
				b.updated_at
			) AS updated_at,
			b.published_at,
			u.id,
			u.call_sign,
			COALESCE(NULLIF(u.display_name, ''), NULLIF(u.google_name, ''), NULLIF(u.call_sign, ''), 'Pilot'),
			COALESCE(u.profile_visibility, 'public') = 'public',
			CASE
				WHEN b.status = 'PUBLISHED' THEN r.id
				ELSE NULL
			END AS staged_revision_id,
			CASE
				WHEN b.status = 'PUBLISHED' THEN r.status
				ELSE NULL
			END AS staged_revision_status
		FROM builds b
		LEFT JOIN builds r
		  ON r.revision_of_build_id = b.id
		 AND r.owner_user_id = b.owner_user_id
		 AND r.status IN ('DRAFT', 'PENDING_REVIEW', 'UNPUBLISHED')
		LEFT JOIN users u ON b.owner_user_id = u.id
		WHERE b.owner_user_id = $1
		  AND b.revision_of_build_id IS NULL
		  AND b.status IN ('DRAFT', 'PENDING_REVIEW', 'PUBLISHED', 'UNPUBLISHED')
		ORDER BY updated_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := s.db.QueryContext(ctx, query, ownerUserID, params.Limit, params.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list owner builds: %w", err)
	}
	defer rows.Close()

	builds, err := scanOwnerBuildRows(rows)
	if err != nil {
		return nil, err
	}

	buildPtrs := make([]*models.Build, 0, len(builds))
	for i := range builds {
		buildPtrs = append(buildPtrs, &builds[i])
	}
	if err := s.attachParts(ctx, buildPtrs); err != nil {
		return nil, err
	}
	if err := s.attachStagedRevisionParts(ctx, buildPtrs); err != nil {
		return nil, err
	}
	s.setMainImageURLs(buildPtrs, false)
	if err := s.attachReactionSummary(ctx, buildPtrs, ownerUserID); err != nil {
		return nil, err
	}

	return &models.BuildListResponse{
		Builds:     builds,
		TotalCount: totalCount,
		Sort:       models.BuildSortNewest,
	}, nil
}

// ListPublic returns published builds for browsing.
func (s *BuildStore) ListPublic(ctx context.Context, params models.BuildListParams, viewerUserID string) (*models.BuildListResponse, error) {
	if params.Sort == "" {
		params.Sort = models.BuildSortNewest
	}
	if params.Limit <= 0 {
		params.Limit = 24
	}
	if params.Limit > 100 {
		params.Limit = 100
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	conditions := []string{"b.status = 'PUBLISHED'"}
	args := []interface{}{}
	argIndex := 1

	if strings.TrimSpace(params.FrameFilter) != "" {
		conditions = append(conditions, fmt.Sprintf(`
			EXISTS (
				SELECT 1
				FROM build_parts bp
				JOIN gear_catalog gc ON gc.id = bp.catalog_item_id
				WHERE bp.build_id = b.id
				  AND bp.gear_type = 'frame'
				  AND (
					LOWER(gc.brand) LIKE LOWER($%d)
					OR LOWER(gc.model) LIKE LOWER($%d)
					OR LOWER(COALESCE(gc.variant, '')) LIKE LOWER($%d)
					OR LOWER(COALESCE(gc.specs->>'size', '')) LIKE LOWER($%d)
				  )
			)
		`, argIndex, argIndex, argIndex, argIndex))
		args = append(args, "%"+strings.TrimSpace(params.FrameFilter)+"%")
		argIndex++
	}

	whereClause := strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM builds b WHERE %s`, whereClause)
	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count public builds: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT
			b.id,
			b.owner_user_id,
			b.image_asset_id,
			b.status,
			b.token,
			b.expires_at,
			b.title,
			b.description,
			b.build_video_url,
			b.flight_video_url,
			b.source_aircraft_id,
			b.moderation_reason,
			b.created_at,
			b.updated_at,
			b.published_at,
			u.id,
			u.call_sign,
			COALESCE(NULLIF(u.display_name, ''), NULLIF(u.google_name, ''), NULLIF(u.call_sign, ''), 'Pilot'),
			COALESCE(u.profile_visibility, 'public') = 'public'
		FROM builds b
		LEFT JOIN users u ON b.owner_user_id = u.id
		WHERE %s
		ORDER BY b.published_at DESC NULLS LAST, b.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, params.Limit, params.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list public builds: %w", err)
	}
	defer rows.Close()

	builds, err := scanBuildRows(rows)
	if err != nil {
		return nil, err
	}
	buildPtrs := make([]*models.Build, 0, len(builds))
	for i := range builds {
		buildPtrs = append(buildPtrs, &builds[i])
	}
	if err := s.attachParts(ctx, buildPtrs); err != nil {
		return nil, err
	}
	s.setMainImageURLs(buildPtrs, true)
	if err := s.attachReactionSummary(ctx, buildPtrs, viewerUserID); err != nil {
		return nil, err
	}

	return &models.BuildListResponse{
		Builds:      builds,
		TotalCount:  totalCount,
		Sort:        params.Sort,
		FrameFilter: strings.TrimSpace(params.FrameFilter),
	}, nil
}

// ListPublishedByOwner returns published builds for a single pilot profile.
func (s *BuildStore) ListPublishedByOwner(ctx context.Context, ownerUserID string, viewerUserID string, limit int) ([]models.Build, error) {
	ownerUserID = strings.TrimSpace(ownerUserID)
	viewerUserID = strings.TrimSpace(viewerUserID)

	if ownerUserID == "" {
		return []models.Build{}, nil
	}

	if limit <= 0 {
		limit = 24
	}
	if limit > 100 {
		limit = 100
	}

	query := `
		SELECT
			b.id,
			b.owner_user_id,
			b.image_asset_id,
			b.status,
			b.token,
			b.expires_at,
			b.title,
			b.description,
			b.build_video_url,
			b.flight_video_url,
			b.source_aircraft_id,
			b.moderation_reason,
			b.created_at,
			b.updated_at,
			b.published_at,
			u.id,
			u.call_sign,
			COALESCE(NULLIF(u.display_name, ''), NULLIF(u.google_name, ''), NULLIF(u.call_sign, ''), 'Pilot'),
			COALESCE(u.profile_visibility, 'public') = 'public'
		FROM builds b
		LEFT JOIN users u ON b.owner_user_id = u.id
		WHERE b.owner_user_id = $1 AND b.status = 'PUBLISHED'
		ORDER BY b.published_at DESC NULLS LAST, b.created_at DESC
		LIMIT $2
	`
	rows, err := s.db.QueryContext(ctx, query, ownerUserID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list published builds by owner: %w", err)
	}
	defer rows.Close()

	builds, err := scanBuildRows(rows)
	if err != nil {
		return nil, err
	}

	buildPtrs := make([]*models.Build, 0, len(builds))
	for i := range builds {
		buildPtrs = append(buildPtrs, &builds[i])
	}
	if err := s.attachParts(ctx, buildPtrs); err != nil {
		return nil, err
	}
	s.setMainImageURLs(buildPtrs, true)
	if err := s.attachReactionSummary(ctx, buildPtrs, viewerUserID); err != nil {
		return nil, err
	}

	return builds, nil
}

// GetByID returns a build without owner/public filtering.
func (s *BuildStore) GetByID(ctx context.Context, id string) (*models.Build, error) {
	query := baseBuildSelect + ` WHERE b.id = $1`
	build, err := s.scanBuild(ctx, query, id)
	if err != nil || build == nil {
		return build, err
	}
	if err := s.attachParts(ctx, []*models.Build{build}); err != nil {
		return nil, err
	}
	s.setMainImageURLs([]*models.Build{build}, false)
	if err := s.attachReactionSummary(ctx, []*models.Build{build}, ""); err != nil {
		return nil, err
	}
	return build, nil
}

// GetForOwner returns a build that belongs to the supplied owner.
func (s *BuildStore) GetForOwner(ctx context.Context, id string, ownerUserID string) (*models.Build, error) {
	query := ownerBuildSelect + ` WHERE b.id = $1 AND b.owner_user_id = $2`
	build, err := s.scanOwnerBuild(ctx, query, id, ownerUserID)
	if err != nil || build == nil {
		return build, err
	}
	if err := s.attachParts(ctx, []*models.Build{build}); err != nil {
		return nil, err
	}
	if err := s.attachStagedRevisionParts(ctx, []*models.Build{build}); err != nil {
		return nil, err
	}
	s.setMainImageURLs([]*models.Build{build}, false)
	if err := s.attachReactionSummary(ctx, []*models.Build{build}, ownerUserID); err != nil {
		return nil, err
	}
	return build, nil
}

// GetPublic returns a published build.
func (s *BuildStore) GetPublic(ctx context.Context, id string, viewerUserID string) (*models.Build, error) {
	query := baseBuildSelect + ` WHERE b.id = $1 AND b.status = 'PUBLISHED'`
	build, err := s.scanBuild(ctx, query, id)
	if err != nil || build == nil {
		return build, err
	}
	if err := s.attachParts(ctx, []*models.Build{build}); err != nil {
		return nil, err
	}
	s.setMainImageURLs([]*models.Build{build}, true)
	if err := s.attachReactionSummary(ctx, []*models.Build{build}, viewerUserID); err != nil {
		return nil, err
	}
	return build, nil
}

// GetTempByToken fetches an unexpired temp build by secret token.
func (s *BuildStore) GetTempByToken(ctx context.Context, token string) (*models.Build, error) {
	query := baseBuildSelect + `
		WHERE b.token = $1
		  AND (
			(b.status = 'TEMP' AND (b.expires_at IS NULL OR b.expires_at > NOW()))
			OR b.status = 'SHARED'
		  )`
	build, err := s.scanBuild(ctx, query, token)
	if err != nil || build == nil {
		return build, err
	}
	if err := s.attachParts(ctx, []*models.Build{build}); err != nil {
		return nil, err
	}
	s.setMainImageURLs([]*models.Build{build}, false)
	if err := s.attachReactionSummary(ctx, []*models.Build{build}, ""); err != nil {
		return nil, err
	}
	return build, nil
}

// Update updates mutable build fields and optionally replaces parts.
func (s *BuildStore) Update(ctx context.Context, id string, ownerUserID string, params models.UpdateBuildParams) (*models.Build, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	targetBuildID := id
	var currentStatus models.BuildStatus
	if err := tx.QueryRowContext(
		ctx,
		`SELECT status FROM builds WHERE id = $1 AND owner_user_id = $2 AND status IN ('DRAFT', 'PENDING_REVIEW', 'PUBLISHED', 'UNPUBLISHED')`,
		id,
		ownerUserID,
	).Scan(&currentStatus); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load build before update: %w", err)
	}

	if currentStatus == models.BuildStatusPublished {
		targetBuildID, err = s.ensurePublishedRevisionDraftTx(ctx, tx, id, ownerUserID)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(targetBuildID) == "" {
			return nil, nil
		}
	}

	setClauses := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIndex := 1

	if params.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIndex))
		args = append(args, strings.TrimSpace(*params.Title))
		argIndex++
	}
	if params.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, strings.TrimSpace(*params.Description))
		argIndex++
	}
	if params.YouTubeURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("build_video_url = $%d", argIndex))
		args = append(args, nullString(strings.TrimSpace(*params.YouTubeURL)))
		argIndex++
	}
	if params.FlightYouTubeURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("flight_video_url = $%d", argIndex))
		args = append(args, nullString(strings.TrimSpace(*params.FlightYouTubeURL)))
		argIndex++
	}

	query := fmt.Sprintf(`
		UPDATE builds
		SET %s
		WHERE id = $%d AND owner_user_id = $%d AND status IN ('DRAFT', 'PENDING_REVIEW', 'UNPUBLISHED')
	`, strings.Join(setClauses, ", "), argIndex, argIndex+1)
	args = append(args, targetBuildID, ownerUserID)

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update build: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, nil
	}

	if params.Parts != nil {
		if err := s.replacePartsTx(ctx, tx, targetBuildID, params.Parts); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit build update: %w", err)
	}

	responseBuildID := targetBuildID
	if currentStatus == models.BuildStatusPublished {
		responseBuildID = id
	}
	return s.GetForOwner(ctx, responseBuildID, ownerUserID)
}

// UpdateTempByToken creates a new temp build revision with a rotated token.
func (s *BuildStore) UpdateTempByToken(ctx context.Context, token string, params models.UpdateBuildParams, nextToken string) (*models.Build, error) {
	build, err := s.GetTempByToken(ctx, token)
	if err != nil || build == nil {
		return build, err
	}
	if build.Status != models.BuildStatusTemp {
		return nil, nil
	}

	title := strings.TrimSpace(build.Title)
	if params.Title != nil {
		title = strings.TrimSpace(*params.Title)
	}

	description := strings.TrimSpace(build.Description)
	if params.Description != nil {
		description = strings.TrimSpace(*params.Description)
	}
	youtubeURL := strings.TrimSpace(build.YouTubeURL)
	if params.YouTubeURL != nil {
		youtubeURL = strings.TrimSpace(*params.YouTubeURL)
	}
	flightYouTubeURL := strings.TrimSpace(build.FlightYouTubeURL)
	if params.FlightYouTubeURL != nil {
		flightYouTubeURL = strings.TrimSpace(*params.FlightYouTubeURL)
	}

	parts := models.BuildPartInputsFromParts(build.Parts)
	if params.Parts != nil {
		parts = params.Parts
	}

	return s.Create(
		ctx,
		build.OwnerUserID,
		models.BuildStatusTemp,
		title,
		description,
		youtubeURL,
		flightYouTubeURL,
		build.SourceAircraftID,
		nextToken,
		build.ExpiresAt,
		parts,
	)
}

// ShareTempByToken promotes a temp build token to a permanent shared link.
func (s *BuildStore) ShareTempByToken(ctx context.Context, token string) (*models.Build, error) {
	build, err := s.GetTempByToken(ctx, token)
	if err != nil || build == nil {
		return build, err
	}

	if build.Status == models.BuildStatusShared {
		return build, nil
	}

	result, err := s.db.ExecContext(
		ctx,
		`UPDATE builds SET status = 'SHARED', expires_at = NULL, updated_at = NOW() WHERE id = $1 AND status = 'TEMP'`,
		build.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to share temp build: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, nil
	}

	return s.GetTempByToken(ctx, token)
}

// SetStatus updates a build's publication status.
func (s *BuildStore) SetStatus(ctx context.Context, id string, ownerUserID string, status models.BuildStatus) (*models.Build, error) {
	status = models.NormalizeBuildStatus(status)
	var query string
	updateBuildID := id

	switch status {
	case models.BuildStatusPendingReview:
		query = `
			UPDATE builds
			SET status = 'PENDING_REVIEW', published_at = NULL, moderation_reason = NULL, updated_at = NOW()
			WHERE id = $1 AND owner_user_id = $2 AND status IN ('DRAFT', 'UNPUBLISHED')
		`
	case models.BuildStatusPublished:
		query = `
			UPDATE builds
			SET status = 'PUBLISHED', published_at = NOW(), moderation_reason = NULL, updated_at = NOW()
			WHERE id = $1 AND owner_user_id = $2 AND status IN ('DRAFT', 'UNPUBLISHED', 'PENDING_REVIEW')
		`
	case models.BuildStatusUnpublished:
		query = `
			UPDATE builds
			SET status = 'UNPUBLISHED', published_at = NULL, moderation_reason = NULL, updated_at = NOW()
			WHERE id = $1 AND owner_user_id = $2 AND status IN ('PUBLISHED', 'PENDING_REVIEW')
		`
	default:
		return nil, fmt.Errorf("unsupported status transition to %q", status)
	}

	result, err := s.db.ExecContext(ctx, query, id, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to update build status: %w", err)
	}
	rows, _ := result.RowsAffected()

	if status == models.BuildStatusPendingReview && rows == 0 {
		revisionResult, revisionErr := s.db.ExecContext(
			ctx,
			`
				UPDATE builds AS revision
				SET status = 'PENDING_REVIEW', published_at = NULL, moderation_reason = NULL, updated_at = NOW()
				FROM builds AS published
				WHERE published.id = $1
				  AND published.owner_user_id = $2
				  AND published.status = 'PUBLISHED'
				  AND revision.revision_of_build_id = published.id
				  AND revision.owner_user_id = published.owner_user_id
				  AND revision.status IN ('DRAFT', 'UNPUBLISHED')
			`,
			id,
			ownerUserID,
		)
		if revisionErr != nil {
			return nil, fmt.Errorf("failed to update staged revision status: %w", revisionErr)
		}
		rows, _ = revisionResult.RowsAffected()
		updateBuildID = id
	}

	if rows == 0 {
		return nil, nil
	}

	return s.GetForOwner(ctx, updateBuildID, ownerUserID)
}

// SetReaction upserts a user's reaction on a published build.
func (s *BuildStore) SetReaction(ctx context.Context, id string, userID string, reaction models.BuildReaction) (*models.Build, error) {
	result, err := s.db.ExecContext(
		ctx,
		`
		INSERT INTO build_reactions (build_id, user_id, reaction)
		SELECT b.id, $2, $3
		FROM builds b
		WHERE b.id = $1
		  AND b.status = 'PUBLISHED'
		ON CONFLICT (build_id, user_id)
		DO UPDATE SET reaction = EXCLUDED.reaction, updated_at = NOW()
		`,
		id,
		userID,
		reaction,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set build reaction: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, nil
	}

	return s.GetPublic(ctx, id, userID)
}

// ClearReaction removes a user's reaction from a published build.
func (s *BuildStore) ClearReaction(ctx context.Context, id string, userID string) (*models.Build, error) {
	var exists bool
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM builds WHERE id = $1 AND status = 'PUBLISHED')`,
		id,
	).Scan(&exists); err != nil {
		return nil, fmt.Errorf("failed to verify build before clearing reaction: %w", err)
	}
	if !exists {
		return nil, nil
	}

	if _, err := s.db.ExecContext(
		ctx,
		`DELETE FROM build_reactions WHERE build_id = $1 AND user_id = $2`,
		id,
		userID,
	); err != nil {
		return nil, fmt.Errorf("failed to clear build reaction: %w", err)
	}

	return s.GetPublic(ctx, id, userID)
}

// SetImage stores a new approved image asset reference for a build.
// Returns any previous image asset ID so callers can clean up orphaned assets.
func (s *BuildStore) SetImage(ctx context.Context, id string, ownerUserID string, imageAssetID string) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to start image transaction: %w", err)
	}
	defer tx.Rollback()

	targetBuildID := id
	var currentStatus models.BuildStatus
	if err := tx.QueryRowContext(
		ctx,
		`SELECT status FROM builds WHERE id = $1 AND owner_user_id = $2 AND status IN ('DRAFT', 'PENDING_REVIEW', 'PUBLISHED', 'UNPUBLISHED')`,
		id,
		ownerUserID,
	).Scan(&currentStatus); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("build not found")
		}
		return "", fmt.Errorf("failed to load build before setting image: %w", err)
	}

	if currentStatus == models.BuildStatusPublished {
		var createErr error
		targetBuildID, createErr = s.ensurePublishedRevisionDraftTx(ctx, tx, id, ownerUserID)
		if createErr != nil {
			return "", createErr
		}
		if strings.TrimSpace(targetBuildID) == "" {
			return "", fmt.Errorf("build not found")
		}
	}

	var previousAssetID sql.NullString
	if err := tx.QueryRowContext(
		ctx,
		`SELECT image_asset_id FROM builds WHERE id = $1 AND owner_user_id = $2 AND status IN ('DRAFT', 'PENDING_REVIEW', 'UNPUBLISHED')`,
		targetBuildID,
		ownerUserID,
	).Scan(&previousAssetID); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("build not found")
		}
		return "", fmt.Errorf("failed to fetch existing build image reference: %w", err)
	}

	query := `
		UPDATE builds
		SET image_asset_id = $1,
		    updated_at = NOW()
		WHERE id = $2 AND owner_user_id = $3 AND status IN ('DRAFT', 'PENDING_REVIEW', 'UNPUBLISHED')
	`
	result, err := tx.ExecContext(ctx, query, imageAssetID, targetBuildID, ownerUserID)
	if err != nil {
		return "", fmt.Errorf("failed to set build image: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return "", fmt.Errorf("build not found")
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit build image update: %w", err)
	}

	if previousAssetID.Valid {
		return previousAssetID.String, nil
	}
	return "", nil
}

// GetImageForOwner loads approved build image bytes for an owner-visible build.
func (s *BuildStore) GetImageForOwner(ctx context.Context, id string, ownerUserID string) ([]byte, error) {
	query := `
		SELECT ia.image_bytes
		FROM builds b
		LEFT JOIN builds r
		  ON r.revision_of_build_id = b.id
		 AND r.owner_user_id = b.owner_user_id
		 AND r.status IN ('DRAFT', 'PENDING_REVIEW', 'UNPUBLISHED')
		JOIN image_assets ia
		  ON ia.id = COALESCE(
			CASE
				WHEN b.status = 'PUBLISHED' THEN r.image_asset_id
				ELSE NULL
			END,
			b.image_asset_id
		  )
		 AND ia.status = 'APPROVED'
		WHERE b.id = $1
		  AND b.owner_user_id = $2
		  AND b.status IN ('DRAFT', 'PENDING_REVIEW', 'PUBLISHED', 'UNPUBLISHED')
		  AND COALESCE(
			CASE
				WHEN b.status = 'PUBLISHED' THEN r.image_asset_id
				ELSE NULL
			END,
			b.image_asset_id
		  ) IS NOT NULL
	`

	var imageData []byte
	err := s.db.QueryRowContext(ctx, query, id, ownerUserID).Scan(&imageData)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get build image: %w", err)
	}
	return imageData, nil
}

// GetPublicImage loads approved build image bytes for a published build.
func (s *BuildStore) GetPublicImage(ctx context.Context, id string) ([]byte, error) {
	query := `
		SELECT ia.image_bytes
		FROM builds b
		JOIN image_assets ia ON ia.id = b.image_asset_id AND ia.status = 'APPROVED'
		WHERE b.id = $1
		  AND b.status = 'PUBLISHED'
		  AND b.image_asset_id IS NOT NULL
	`

	var imageData []byte
	err := s.db.QueryRowContext(ctx, query, id).Scan(&imageData)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get public build image: %w", err)
	}
	return imageData, nil
}

// DeleteImage removes a build image and returns any previous image asset ID.
func (s *BuildStore) DeleteImage(ctx context.Context, id string, ownerUserID string) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to start image delete transaction: %w", err)
	}
	defer tx.Rollback()

	targetBuildID := id
	var currentStatus models.BuildStatus
	if err := tx.QueryRowContext(
		ctx,
		`SELECT status FROM builds WHERE id = $1 AND owner_user_id = $2 AND status IN ('DRAFT', 'PENDING_REVIEW', 'PUBLISHED', 'UNPUBLISHED')`,
		id,
		ownerUserID,
	).Scan(&currentStatus); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("build not found")
		}
		return "", fmt.Errorf("failed to load build before deleting image: %w", err)
	}

	if currentStatus == models.BuildStatusPublished {
		var createErr error
		targetBuildID, createErr = s.ensurePublishedRevisionDraftTx(ctx, tx, id, ownerUserID)
		if createErr != nil {
			return "", createErr
		}
		if strings.TrimSpace(targetBuildID) == "" {
			return "", fmt.Errorf("build not found")
		}
	}

	var previousAssetID sql.NullString
	if err := tx.QueryRowContext(
		ctx,
		`SELECT image_asset_id FROM builds WHERE id = $1 AND owner_user_id = $2 AND status IN ('DRAFT', 'PENDING_REVIEW', 'UNPUBLISHED')`,
		targetBuildID,
		ownerUserID,
	).Scan(&previousAssetID); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("build not found")
		}
		return "", fmt.Errorf("failed to fetch existing build image reference: %w", err)
	}

	query := `
		UPDATE builds
		SET image_asset_id = NULL,
		    updated_at = NOW()
		WHERE id = $1 AND owner_user_id = $2 AND status IN ('DRAFT', 'PENDING_REVIEW', 'UNPUBLISHED')
	`
	result, err := tx.ExecContext(ctx, query, targetBuildID, ownerUserID)
	if err != nil {
		return "", fmt.Errorf("failed to delete build image: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return "", fmt.Errorf("build not found")
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit build image delete: %w", err)
	}
	if previousAssetID.Valid {
		return previousAssetID.String, nil
	}
	return "", nil
}

// Delete removes a non-temp build for the owner.
func (s *BuildStore) Delete(ctx context.Context, id string, ownerUserID string) (bool, error) {
	result, err := s.db.ExecContext(
		ctx,
		`DELETE FROM builds WHERE id = $1 AND owner_user_id = $2 AND status IN ('DRAFT', 'PENDING_REVIEW', 'PUBLISHED', 'UNPUBLISHED')`,
		id,
		ownerUserID,
	)
	if err != nil {
		return false, fmt.Errorf("failed to delete build: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	return rowsAffected > 0, nil
}

// DeleteExpiredTemp deletes temp builds expired at or before cutoff.
func (s *BuildStore) DeleteExpiredTemp(ctx context.Context, cutoff time.Time) (int64, error) {
	result, err := s.db.ExecContext(
		ctx,
		`DELETE FROM builds WHERE status = 'TEMP' AND expires_at IS NOT NULL AND expires_at <= $1`,
		cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired temp builds: %w", err)
	}
	rows, _ := result.RowsAffected()
	return rows, nil
}

// ListForModeration returns builds for content moderation workflows.
func (s *BuildStore) ListForModeration(ctx context.Context, params models.BuildModerationListParams) (*models.BuildListResponse, error) {
	if params.Limit <= 0 {
		params.Limit = 30
	}
	if params.Limit > 100 {
		params.Limit = 100
	}
	if params.Offset < 0 {
		params.Offset = 0
	}

	status := models.NormalizeBuildStatus(params.Status)
	if status == "" {
		status = models.BuildStatusPendingReview
	}

	conditions := []string{"b.status = $1"}
	args := []interface{}{status}
	argIdx := 2

	search := strings.TrimSpace(params.Query)
	if search != "" {
		conditions = append(conditions, fmt.Sprintf(`
			(
				LOWER(COALESCE(b.title, '')) LIKE LOWER($%d)
				OR LOWER(COALESCE(b.description, '')) LIKE LOWER($%d)
				OR LOWER(COALESCE(b.build_video_url, '')) LIKE LOWER($%d)
				OR LOWER(COALESCE(b.flight_video_url, '')) LIKE LOWER($%d)
				OR LOWER(COALESCE(u.call_sign, '')) LIKE LOWER($%d)
				OR LOWER(COALESCE(u.display_name, '')) LIKE LOWER($%d)
			)
		`, argIdx, argIdx, argIdx, argIdx, argIdx, argIdx))
		args = append(args, "%"+search+"%")
		argIdx++
	}

	switch params.DeclineFilter {
	case models.BuildModerationDeclineFilterDeclined:
		conditions = append(conditions, "NULLIF(TRIM(b.moderation_reason), '') IS NOT NULL")
	case models.BuildModerationDeclineFilterNotDeclined:
		conditions = append(conditions, "NULLIF(TRIM(b.moderation_reason), '') IS NULL")
	}

	whereClause := strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM builds b
		LEFT JOIN users u ON b.owner_user_id = u.id
		WHERE %s
	`, whereClause)

	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count moderation builds: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT
			b.id,
			b.owner_user_id,
			b.image_asset_id,
			b.status,
			b.token,
			b.expires_at,
			b.title,
			b.description,
			b.build_video_url,
			b.flight_video_url,
			b.source_aircraft_id,
			b.moderation_reason,
			b.created_at,
			b.updated_at,
			b.published_at,
			u.id,
			u.call_sign,
			COALESCE(NULLIF(u.display_name, ''), NULLIF(u.google_name, ''), NULLIF(u.call_sign, ''), 'Pilot'),
			COALESCE(u.profile_visibility, 'public') = 'public'
		FROM builds b
		LEFT JOIN users u ON b.owner_user_id = u.id
		WHERE %s
		ORDER BY b.updated_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	args = append(args, params.Limit, params.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list moderation builds: %w", err)
	}
	defer rows.Close()

	builds, err := scanBuildRows(rows)
	if err != nil {
		return nil, err
	}

	buildPtrs := make([]*models.Build, 0, len(builds))
	for i := range builds {
		buildPtrs = append(buildPtrs, &builds[i])
	}
	if err := s.attachParts(ctx, buildPtrs); err != nil {
		return nil, err
	}
	s.setAdminMainImageURLs(buildPtrs)
	if err := s.attachReactionSummary(ctx, buildPtrs, ""); err != nil {
		return nil, err
	}

	return &models.BuildListResponse{
		Builds:     builds,
		TotalCount: totalCount,
		Sort:       models.BuildSortNewest,
	}, nil
}

// GetForModeration returns a build for content moderation workflows.
func (s *BuildStore) GetForModeration(ctx context.Context, id string) (*models.Build, error) {
	query := baseBuildSelect + ` WHERE b.id = $1 AND b.status IN ('DRAFT', 'PENDING_REVIEW', 'PUBLISHED', 'UNPUBLISHED')`
	build, err := s.scanBuild(ctx, query, id)
	if err != nil || build == nil {
		return build, err
	}
	if err := s.attachParts(ctx, []*models.Build{build}); err != nil {
		return nil, err
	}
	s.setAdminMainImageURLs([]*models.Build{build})
	if err := s.attachReactionSummary(ctx, []*models.Build{build}, ""); err != nil {
		return nil, err
	}
	return build, nil
}

// UpdateForModeration updates build title/description and optionally parts by moderator.
func (s *BuildStore) UpdateForModeration(ctx context.Context, id string, params models.UpdateBuildParams) (*models.Build, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	setClauses := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIndex := 1

	if params.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIndex))
		args = append(args, strings.TrimSpace(*params.Title))
		argIndex++
	}
	if params.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, strings.TrimSpace(*params.Description))
		argIndex++
	}
	if params.YouTubeURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("build_video_url = $%d", argIndex))
		args = append(args, nullString(strings.TrimSpace(*params.YouTubeURL)))
		argIndex++
	}
	if params.FlightYouTubeURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("flight_video_url = $%d", argIndex))
		args = append(args, nullString(strings.TrimSpace(*params.FlightYouTubeURL)))
		argIndex++
	}

	query := fmt.Sprintf(`
		UPDATE builds
		SET %s
		WHERE id = $%d AND status IN ('DRAFT', 'PENDING_REVIEW', 'UNPUBLISHED')
	`, strings.Join(setClauses, ", "), argIndex)
	args = append(args, id)

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update moderation build: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, nil
	}

	if params.Parts != nil {
		if err := s.replacePartsTx(ctx, tx, id, params.Parts); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit moderation build update: %w", err)
	}

	return s.GetForModeration(ctx, id)
}

// SetImageForModeration stores a new approved image asset reference for a build.
func (s *BuildStore) SetImageForModeration(ctx context.Context, id string, imageAssetID string) (string, error) {
	var previousAssetID sql.NullString
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT image_asset_id FROM builds WHERE id = $1 AND status IN ('DRAFT', 'PENDING_REVIEW', 'PUBLISHED', 'UNPUBLISHED')`,
		id,
	).Scan(&previousAssetID); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("build not found")
		}
		return "", fmt.Errorf("failed to fetch existing build image reference: %w", err)
	}

	result, err := s.db.ExecContext(
		ctx,
		`UPDATE builds SET image_asset_id = $1, updated_at = NOW() WHERE id = $2 AND status IN ('DRAFT', 'PENDING_REVIEW', 'PUBLISHED', 'UNPUBLISHED')`,
		imageAssetID,
		id,
	)
	if err != nil {
		return "", fmt.Errorf("failed to set moderation build image: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return "", fmt.Errorf("build not found")
	}

	if previousAssetID.Valid {
		return previousAssetID.String, nil
	}
	return "", nil
}

// GetImageForModeration loads approved image bytes for admin moderation views.
func (s *BuildStore) GetImageForModeration(ctx context.Context, id string) ([]byte, error) {
	query := `
		SELECT ia.image_bytes
		FROM builds b
		JOIN image_assets ia ON ia.id = b.image_asset_id AND ia.status = 'APPROVED'
		WHERE b.id = $1
		  AND b.status IN ('DRAFT', 'PENDING_REVIEW', 'PUBLISHED', 'UNPUBLISHED')
		  AND b.image_asset_id IS NOT NULL
	`

	var imageData []byte
	err := s.db.QueryRowContext(ctx, query, id).Scan(&imageData)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get moderation build image: %w", err)
	}
	return imageData, nil
}

// DeleteImageForModeration removes a build image and returns any previous asset ID.
func (s *BuildStore) DeleteImageForModeration(ctx context.Context, id string) (string, error) {
	var previousAssetID sql.NullString
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT image_asset_id FROM builds WHERE id = $1 AND status IN ('DRAFT', 'PENDING_REVIEW', 'PUBLISHED', 'UNPUBLISHED')`,
		id,
	).Scan(&previousAssetID); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("build not found")
		}
		return "", fmt.Errorf("failed to fetch existing build image reference: %w", err)
	}

	result, err := s.db.ExecContext(
		ctx,
		`UPDATE builds SET image_asset_id = NULL, updated_at = NOW() WHERE id = $1 AND status IN ('DRAFT', 'PENDING_REVIEW', 'PUBLISHED', 'UNPUBLISHED')`,
		id,
	)
	if err != nil {
		return "", fmt.Errorf("failed to delete moderation build image: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return "", fmt.Errorf("build not found")
	}
	if previousAssetID.Valid {
		return previousAssetID.String, nil
	}
	return "", nil
}

// ApproveForModeration publishes a build from the pending moderation queue.
func (s *BuildStore) ApproveForModeration(ctx context.Context, id string) (*models.Build, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start moderation approval transaction: %w", err)
	}
	defer tx.Rollback()

	var revisionOfBuildID sql.NullString
	if err := tx.QueryRowContext(
		ctx,
		`SELECT revision_of_build_id FROM builds WHERE id = $1 AND status = 'PENDING_REVIEW'`,
		id,
	).Scan(&revisionOfBuildID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to load pending moderation build: %w", err)
	}

	approvedBuildID := id
	if strings.TrimSpace(revisionOfBuildID.String) != "" {
		approvedBuildID = strings.TrimSpace(revisionOfBuildID.String)

		result, err := tx.ExecContext(
			ctx,
			`
				UPDATE builds AS published
				SET title = pending.title,
				    description = pending.description,
				    build_video_url = pending.build_video_url,
				    flight_video_url = pending.flight_video_url,
				    source_aircraft_id = pending.source_aircraft_id,
				    image_asset_id = pending.image_asset_id,
				    moderation_reason = NULL,
				    updated_at = NOW()
				FROM builds AS pending
				WHERE pending.id = $1
				  AND pending.status = 'PENDING_REVIEW'
				  AND published.id = pending.revision_of_build_id
				  AND published.owner_user_id = pending.owner_user_id
				  AND published.status = 'PUBLISHED'
			`,
			id,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to merge approved build revision: %w", err)
		}
		rows, _ := result.RowsAffected()
		if rows == 0 {
			return nil, nil
		}

		if _, err := tx.ExecContext(ctx, `DELETE FROM build_parts WHERE build_id = $1`, approvedBuildID); err != nil {
			return nil, fmt.Errorf("failed to clear published build parts during approval: %w", err)
		}
		if _, err := tx.ExecContext(
			ctx,
			`
				INSERT INTO build_parts (build_id, gear_type, catalog_item_id, position, notes)
				SELECT $1, gear_type, catalog_item_id, position, notes
				FROM build_parts
				WHERE build_id = $2
			`,
			approvedBuildID,
			id,
		); err != nil {
			return nil, fmt.Errorf("failed to copy approved revision parts: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `DELETE FROM builds WHERE id = $1`, id); err != nil {
			return nil, fmt.Errorf("failed to clean up approved build revision: %w", err)
		}
	} else {
		result, err := tx.ExecContext(
			ctx,
			`UPDATE builds SET status = 'PUBLISHED', published_at = NOW(), moderation_reason = NULL, updated_at = NOW() WHERE id = $1 AND status = 'PENDING_REVIEW'`,
			id,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to approve moderation build: %w", err)
		}
		rows, _ := result.RowsAffected()
		if rows == 0 {
			return nil, nil
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit moderation approval: %w", err)
	}

	return s.GetForModeration(ctx, approvedBuildID)
}

// DeclineForModeration rejects a pending build and stores moderator feedback.
func (s *BuildStore) DeclineForModeration(ctx context.Context, id string, reason string) (*models.Build, error) {
	trimmedReason := strings.TrimSpace(reason)
	if trimmedReason == "" {
		return nil, fmt.Errorf("decline reason is required")
	}

	result, err := s.db.ExecContext(
		ctx,
		`
			UPDATE builds
			SET status = 'UNPUBLISHED',
			    published_at = NULL,
			    moderation_reason = $2,
			    updated_at = NOW()
			WHERE id = $1
			  AND status = 'PENDING_REVIEW'
		`,
		id,
		trimmedReason,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to decline moderation build: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, nil
	}

	return s.GetForModeration(ctx, id)
}

func (s *BuildStore) ensurePublishedRevisionDraftTx(ctx context.Context, tx *sql.Tx, publishedBuildID string, ownerUserID string) (string, error) {
	var revisionBuildID string
	if err := s.selectExistingRevisionDraftTx(ctx, tx, ownerUserID, publishedBuildID, &revisionBuildID); err == nil {
		return revisionBuildID, nil
	} else if err != sql.ErrNoRows {
		return "", fmt.Errorf("failed to check existing published build revision: %w", err)
	}

	if err := tx.QueryRowContext(
		ctx,
		`
			INSERT INTO builds (
				owner_user_id,
				image_asset_id,
				status,
				title,
				description,
				build_video_url,
				flight_video_url,
				source_aircraft_id,
				revision_of_build_id
			)
			SELECT
				owner_user_id,
				image_asset_id,
				'DRAFT',
				title,
				description,
				build_video_url,
				flight_video_url,
				source_aircraft_id,
				id
			FROM builds
			WHERE id = $1
			  AND owner_user_id = $2
			  AND status = 'PUBLISHED'
			RETURNING id
		`,
		publishedBuildID,
		ownerUserID,
	).Scan(&revisionBuildID); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && string(pqErr.Code) == "23505" {
			if retryErr := s.selectExistingRevisionDraftTx(ctx, tx, ownerUserID, publishedBuildID, &revisionBuildID); retryErr == nil {
				return revisionBuildID, nil
			} else if retryErr != sql.ErrNoRows {
				return "", fmt.Errorf("failed to recover existing revision draft after concurrent insert: %w", retryErr)
			}
		}
		return "", fmt.Errorf("failed to create published build revision draft: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`
			INSERT INTO build_parts (build_id, gear_type, catalog_item_id, position, notes)
			SELECT $1, gear_type, catalog_item_id, position, notes
			FROM build_parts
			WHERE build_id = $2
		`,
		revisionBuildID,
		publishedBuildID,
	); err != nil {
		return "", fmt.Errorf("failed to copy parts into revision draft: %w", err)
	}

	return revisionBuildID, nil
}

func (s *BuildStore) selectExistingRevisionDraftTx(
	ctx context.Context,
	tx *sql.Tx,
	ownerUserID string,
	publishedBuildID string,
	revisionBuildID *string,
) error {
	return tx.QueryRowContext(
		ctx,
		`
			SELECT id
			FROM builds
			WHERE owner_user_id = $1
			  AND revision_of_build_id = $2
			  AND status IN ('DRAFT', 'PENDING_REVIEW', 'UNPUBLISHED')
			ORDER BY updated_at DESC
			LIMIT 1
		`,
		ownerUserID,
		publishedBuildID,
	).Scan(revisionBuildID)
}

func (s *BuildStore) replacePartsTx(ctx context.Context, tx *sql.Tx, buildID string, parts []models.BuildPartInput) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM build_parts WHERE build_id = $1`, buildID); err != nil {
		return fmt.Errorf("failed to clear build parts: %w", err)
	}

	if len(parts) == 0 {
		return nil
	}

	query := `
		INSERT INTO build_parts (build_id, gear_type, catalog_item_id, position, notes)
		VALUES ($1, $2, $3, $4, $5)
	`

	for _, part := range parts {
		if part.GearType == "" {
			continue
		}
		if strings.TrimSpace(part.CatalogItemID) == "" {
			continue
		}
		position := part.Position
		if position < 0 {
			position = 0
		}
		if _, err := tx.ExecContext(
			ctx,
			query,
			buildID,
			part.GearType,
			nullString(strings.TrimSpace(part.CatalogItemID)),
			position,
			nullString(strings.TrimSpace(part.Notes)),
		); err != nil {
			return fmt.Errorf("failed to insert build part (%s): %w", part.GearType, err)
		}
	}

	return nil
}

func (s *BuildStore) scanBuild(ctx context.Context, query string, args ...interface{}) (*models.Build, error) {
	row := s.db.QueryRowContext(ctx, query, args...)
	build, err := scanBuildRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan build: %w", err)
	}
	return build, nil
}

func (s *BuildStore) scanOwnerBuild(ctx context.Context, query string, args ...interface{}) (*models.Build, error) {
	row := s.db.QueryRowContext(ctx, query, args...)
	build, err := scanOwnerBuildRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan owner build: %w", err)
	}
	return build, nil
}

func (s *BuildStore) attachParts(ctx context.Context, builds []*models.Build) error {
	if len(builds) == 0 {
		return nil
	}

	ids := make([]string, 0, len(builds))
	idToIndex := make(map[string]int, len(builds))
	for i := range builds {
		ids = append(ids, builds[i].ID)
		idToIndex[builds[i].ID] = i
	}

	query := `
		SELECT
			bp.id,
			bp.build_id,
			bp.gear_type,
			bp.catalog_item_id,
			bp.position,
			bp.notes,
			bp.created_at,
			bp.updated_at,
			gc.id,
			gc.gear_type,
			gc.brand,
			gc.model,
			gc.variant,
			gc.msrp,
			gc.status,
			COALESCE(
				NULLIF(TRIM(gc.image_url), ''),
				CASE
					WHEN (gc.image_asset_id IS NOT NULL OR gc.image_data IS NOT NULL) AND COALESCE(gc.image_status, 'missing') IN ('approved', 'scanned')
						THEN '/api/gear-catalog/' || gc.id || '/image?v=' || (EXTRACT(EPOCH FROM COALESCE(gc.image_curated_at, gc.updated_at))*1000)::bigint
					ELSE NULL
				END
			) AS image_url
		FROM build_parts bp
		LEFT JOIN gear_catalog gc ON gc.id = bp.catalog_item_id
		WHERE bp.build_id = ANY($1::uuid[])
		ORDER BY bp.build_id, bp.gear_type, bp.position
	`

	rows, err := s.db.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		return fmt.Errorf("failed to load build parts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var part models.BuildPart
		var catalogItemID sql.NullString
		var notes sql.NullString
		var catalogID sql.NullString
		var catalogGearType sql.NullString
		var catalogBrand sql.NullString
		var catalogModel sql.NullString
		var catalogVariant sql.NullString
		var catalogMSRP sql.NullFloat64
		var catalogStatus sql.NullString
		var catalogImageURL sql.NullString

		if err := rows.Scan(
			&part.ID,
			&part.BuildID,
			&part.GearType,
			&catalogItemID,
			&part.Position,
			&notes,
			&part.CreatedAt,
			&part.UpdatedAt,
			&catalogID,
			&catalogGearType,
			&catalogBrand,
			&catalogModel,
			&catalogVariant,
			&catalogMSRP,
			&catalogStatus,
			&catalogImageURL,
		); err != nil {
			return fmt.Errorf("failed to scan build part: %w", err)
		}

		part.CatalogItemID = catalogItemID.String
		part.Notes = notes.String

		if catalogID.Valid {
			part.CatalogItem = &models.BuildCatalogItem{
				ID:       catalogID.String,
				GearType: models.GearType(catalogGearType.String),
				Brand:    catalogBrand.String,
				Model:    catalogModel.String,
				Variant:  catalogVariant.String,
				Status:   models.NormalizeCatalogStatus(models.CatalogItemStatus(catalogStatus.String)),
				ImageURL: catalogImageURL.String,
			}
			if catalogMSRP.Valid {
				msrp := catalogMSRP.Float64
				part.CatalogItem.MSRP = &msrp
			}
		}

		idx, ok := idToIndex[part.BuildID]
		if !ok {
			continue
		}
		builds[idx].Parts = append(builds[idx].Parts, part)
	}

	for i := range builds {
		for _, part := range builds[i].Parts {
			if part.GearType == models.GearTypeFrame && part.CatalogItem != nil && part.CatalogItem.ImageURL != "" {
				builds[i].MainImageURL = part.CatalogItem.ImageURL
				break
			}
		}
	}

	return nil
}

func (s *BuildStore) attachStagedRevisionParts(ctx context.Context, builds []*models.Build) error {
	stagedIDs := make([]string, 0, len(builds))
	for _, build := range builds {
		if build == nil || strings.TrimSpace(build.StagedRevisionID) == "" {
			continue
		}
		stagedIDs = append(stagedIDs, build.StagedRevisionID)
	}
	if len(stagedIDs) == 0 {
		return nil
	}

	stagedBuildPtrs := make([]*models.Build, 0, len(stagedIDs))
	for _, stagedID := range stagedIDs {
		stagedBuildPtrs = append(stagedBuildPtrs, &models.Build{ID: stagedID})
	}
	if err := s.attachParts(ctx, stagedBuildPtrs); err != nil {
		return fmt.Errorf("failed to load staged revision parts: %w", err)
	}

	stagedByID := make(map[string]*models.Build, len(stagedBuildPtrs))
	for _, stagedBuild := range stagedBuildPtrs {
		if stagedBuild == nil || strings.TrimSpace(stagedBuild.ID) == "" {
			continue
		}
		stagedByID[stagedBuild.ID] = stagedBuild
	}

	for _, build := range builds {
		if build == nil || strings.TrimSpace(build.StagedRevisionID) == "" {
			continue
		}

		staged := stagedByID[build.StagedRevisionID]
		if staged == nil {
			continue
		}
		build.Parts = staged.Parts
		if strings.TrimSpace(build.ImageAssetID) == "" {
			build.MainImageURL = staged.MainImageURL
		}
	}
	return nil
}

func (s *BuildStore) attachReactionSummary(ctx context.Context, builds []*models.Build, viewerUserID string) error {
	if len(builds) == 0 {
		return nil
	}

	ids := make([]string, 0, len(builds))
	byID := make(map[string]*models.Build, len(builds))
	for _, build := range builds {
		if build == nil {
			continue
		}
		build.LikeCount = 0
		build.DislikeCount = 0
		build.ViewerReaction = ""
		ids = append(ids, build.ID)
		byID[build.ID] = build
	}

	if len(ids) == 0 {
		return nil
	}

	countRows, err := s.db.QueryContext(
		ctx,
		`
		SELECT
			build_id,
			COUNT(*) FILTER (WHERE reaction = 'LIKE') AS like_count,
			COUNT(*) FILTER (WHERE reaction = 'DISLIKE') AS dislike_count
		FROM build_reactions
		WHERE build_id = ANY($1::uuid[])
		GROUP BY build_id
		`,
		pq.Array(ids),
	)
	if err != nil {
		return fmt.Errorf("failed to load build reaction counts: %w", err)
	}
	defer countRows.Close()

	for countRows.Next() {
		var (
			buildID      string
			likeCount    int
			dislikeCount int
		)
		if err := countRows.Scan(&buildID, &likeCount, &dislikeCount); err != nil {
			return fmt.Errorf("failed to scan build reaction counts: %w", err)
		}
		build := byID[buildID]
		if build == nil {
			continue
		}
		build.LikeCount = likeCount
		build.DislikeCount = dislikeCount
	}

	if err := countRows.Err(); err != nil {
		return fmt.Errorf("failed to iterate build reaction counts: %w", err)
	}

	if strings.TrimSpace(viewerUserID) == "" {
		return nil
	}

	reactionRows, err := s.db.QueryContext(
		ctx,
		`
		SELECT build_id, reaction
		FROM build_reactions
		WHERE build_id = ANY($1::uuid[])
		  AND user_id = $2
		`,
		pq.Array(ids),
		viewerUserID,
	)
	if err != nil {
		return fmt.Errorf("failed to load viewer build reactions: %w", err)
	}
	defer reactionRows.Close()

	for reactionRows.Next() {
		var (
			buildID  string
			reaction string
		)
		if err := reactionRows.Scan(&buildID, &reaction); err != nil {
			return fmt.Errorf("failed to scan viewer build reaction: %w", err)
		}
		build := byID[buildID]
		if build == nil {
			continue
		}
		build.ViewerReaction = models.NormalizeBuildReaction(models.BuildReaction(reaction))
	}

	if err := reactionRows.Err(); err != nil {
		return fmt.Errorf("failed to iterate viewer build reactions: %w", err)
	}

	return nil
}

func (s *BuildStore) setMainImageURLs(builds []*models.Build, isPublic bool) {
	for _, build := range builds {
		if build == nil {
			continue
		}
		if strings.TrimSpace(build.ImageAssetID) == "" {
			continue
		}
		if isPublic {
			build.MainImageURL = fmt.Sprintf("/api/public/builds/%s/image?v=%d", build.ID, build.UpdatedAt.UnixMilli())
		} else {
			build.MainImageURL = fmt.Sprintf("/api/builds/%s/image?v=%d", build.ID, build.UpdatedAt.UnixMilli())
		}
	}
}

func (s *BuildStore) setAdminMainImageURLs(builds []*models.Build) {
	for _, build := range builds {
		if build == nil {
			continue
		}
		if strings.TrimSpace(build.ImageAssetID) == "" {
			continue
		}
		build.MainImageURL = fmt.Sprintf("/api/admin/builds/%s/image?v=%d", build.ID, build.UpdatedAt.UnixMilli())
	}
}

var baseBuildSelect = `
	SELECT
		b.id,
		b.owner_user_id,
		b.image_asset_id,
		b.status,
		b.token,
		b.expires_at,
		b.title,
		b.description,
		b.build_video_url,
		b.flight_video_url,
		b.source_aircraft_id,
		b.moderation_reason,
		b.created_at,
		b.updated_at,
		b.published_at,
		u.id,
		u.call_sign,
		COALESCE(NULLIF(u.display_name, ''), NULLIF(u.google_name, ''), NULLIF(u.call_sign, ''), 'Pilot'),
		COALESCE(u.profile_visibility, 'public') = 'public'
	FROM builds b
	LEFT JOIN users u ON b.owner_user_id = u.id
`

var ownerBuildSelect = `
	SELECT
		b.id,
		b.owner_user_id,
		COALESCE(
			CASE
				WHEN b.status = 'PUBLISHED' THEN r.image_asset_id
				ELSE NULL
			END,
			b.image_asset_id
		) AS image_asset_id,
		b.status,
		b.token,
		b.expires_at,
		COALESCE(
			CASE
				WHEN b.status = 'PUBLISHED' THEN r.title
				ELSE NULL
			END,
			b.title
		) AS title,
		COALESCE(
			CASE
				WHEN b.status = 'PUBLISHED' THEN r.description
				ELSE NULL
			END,
			b.description
		) AS description,
		COALESCE(
			CASE
				WHEN b.status = 'PUBLISHED' THEN r.build_video_url
				ELSE NULL
			END,
			b.build_video_url
		) AS build_video_url,
		COALESCE(
			CASE
				WHEN b.status = 'PUBLISHED' THEN r.flight_video_url
				ELSE NULL
			END,
			b.flight_video_url
		) AS flight_video_url,
		COALESCE(
			CASE
				WHEN b.status = 'PUBLISHED' THEN r.source_aircraft_id
				ELSE NULL
			END,
			b.source_aircraft_id
		) AS source_aircraft_id,
		COALESCE(
			CASE
				WHEN b.status = 'PUBLISHED' THEN r.moderation_reason
				ELSE NULL
			END,
			b.moderation_reason
		) AS moderation_reason,
		b.created_at,
		COALESCE(
			CASE
				WHEN b.status = 'PUBLISHED' THEN r.updated_at
				ELSE NULL
			END,
			b.updated_at
		) AS updated_at,
		b.published_at,
		u.id,
		u.call_sign,
		COALESCE(NULLIF(u.display_name, ''), NULLIF(u.google_name, ''), NULLIF(u.call_sign, ''), 'Pilot'),
		COALESCE(u.profile_visibility, 'public') = 'public',
		CASE
			WHEN b.status = 'PUBLISHED' THEN r.id
			ELSE NULL
		END AS staged_revision_id,
		CASE
			WHEN b.status = 'PUBLISHED' THEN r.status
			ELSE NULL
		END AS staged_revision_status
	FROM builds b
	LEFT JOIN builds r
	  ON r.revision_of_build_id = b.id
	 AND r.owner_user_id = b.owner_user_id
	 AND r.status IN ('DRAFT', 'PENDING_REVIEW', 'UNPUBLISHED')
	LEFT JOIN users u ON b.owner_user_id = u.id
`

func scanBuildRows(rows *sql.Rows) ([]models.Build, error) {
	items := make([]models.Build, 0)
	for rows.Next() {
		item, err := scanBuildRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan build row: %w", err)
		}
		items = append(items, *item)
	}
	return items, nil
}

func scanOwnerBuildRows(rows *sql.Rows) ([]models.Build, error) {
	items := make([]models.Build, 0)
	for rows.Next() {
		item, err := scanOwnerBuildRow(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan owner build row: %w", err)
		}
		items = append(items, *item)
	}
	return items, nil
}

func scanBuildRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.Build, error) {
	var item models.Build
	var ownerUserID sql.NullString
	var imageAssetID sql.NullString
	var token sql.NullString
	var expiresAt sql.NullTime
	var description sql.NullString
	var youtubeURL sql.NullString
	var flightYouTubeURL sql.NullString
	var sourceAircraftID sql.NullString
	var moderationReason sql.NullString
	var publishedAt sql.NullTime

	var pilotUserID sql.NullString
	var pilotCallSign sql.NullString
	var pilotDisplayName sql.NullString
	var pilotIsPublic sql.NullBool

	err := scanner.Scan(
		&item.ID,
		&ownerUserID,
		&imageAssetID,
		&item.Status,
		&token,
		&expiresAt,
		&item.Title,
		&description,
		&youtubeURL,
		&flightYouTubeURL,
		&sourceAircraftID,
		&moderationReason,
		&item.CreatedAt,
		&item.UpdatedAt,
		&publishedAt,
		&pilotUserID,
		&pilotCallSign,
		&pilotDisplayName,
		&pilotIsPublic,
	)
	if err != nil {
		return nil, err
	}

	item.OwnerUserID = ownerUserID.String
	item.ImageAssetID = imageAssetID.String
	item.Token = token.String
	item.Description = description.String
	item.YouTubeURL = youtubeURL.String
	item.FlightYouTubeURL = flightYouTubeURL.String
	item.SourceAircraftID = sourceAircraftID.String
	item.ModerationReason = moderationReason.String
	if expiresAt.Valid {
		item.ExpiresAt = &expiresAt.Time
	}
	if publishedAt.Valid {
		item.PublishedAt = &publishedAt.Time
	}

	if pilotUserID.Valid {
		pilot := &models.BuildPilot{
			UserID:          pilotUserID.String,
			CallSign:        pilotCallSign.String,
			DisplayName:     pilotDisplayName.String,
			IsProfilePublic: pilotIsPublic.Bool,
		}
		if pilot.IsProfilePublic && pilot.UserID != "" {
			pilot.ProfileURL = "/social/pilots/" + pilot.UserID
		}
		item.Pilot = pilot
	}

	return &item, nil
}

func scanOwnerBuildRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.Build, error) {
	var item models.Build
	var ownerUserID sql.NullString
	var imageAssetID sql.NullString
	var token sql.NullString
	var expiresAt sql.NullTime
	var description sql.NullString
	var youtubeURL sql.NullString
	var flightYouTubeURL sql.NullString
	var sourceAircraftID sql.NullString
	var moderationReason sql.NullString
	var publishedAt sql.NullTime
	var stagedRevisionID sql.NullString
	var stagedRevisionStatus sql.NullString

	var pilotUserID sql.NullString
	var pilotCallSign sql.NullString
	var pilotDisplayName sql.NullString
	var pilotIsPublic sql.NullBool

	err := scanner.Scan(
		&item.ID,
		&ownerUserID,
		&imageAssetID,
		&item.Status,
		&token,
		&expiresAt,
		&item.Title,
		&description,
		&youtubeURL,
		&flightYouTubeURL,
		&sourceAircraftID,
		&moderationReason,
		&item.CreatedAt,
		&item.UpdatedAt,
		&publishedAt,
		&pilotUserID,
		&pilotCallSign,
		&pilotDisplayName,
		&pilotIsPublic,
		&stagedRevisionID,
		&stagedRevisionStatus,
	)
	if err != nil {
		return nil, err
	}

	item.OwnerUserID = ownerUserID.String
	item.ImageAssetID = imageAssetID.String
	item.Token = token.String
	item.Description = description.String
	item.YouTubeURL = youtubeURL.String
	item.FlightYouTubeURL = flightYouTubeURL.String
	item.SourceAircraftID = sourceAircraftID.String
	item.ModerationReason = moderationReason.String
	item.StagedRevisionID = stagedRevisionID.String
	item.StagedRevisionStatus = models.NormalizeBuildStatus(models.BuildStatus(stagedRevisionStatus.String))
	if expiresAt.Valid {
		item.ExpiresAt = &expiresAt.Time
	}
	if publishedAt.Valid {
		item.PublishedAt = &publishedAt.Time
	}

	if pilotUserID.Valid {
		pilot := &models.BuildPilot{
			UserID:          pilotUserID.String,
			CallSign:        pilotCallSign.String,
			DisplayName:     pilotDisplayName.String,
			IsProfilePublic: pilotIsPublic.Bool,
		}
		if pilot.IsProfilePublic && pilot.UserID != "" {
			pilot.ProfileURL = "/social/pilots/" + pilot.UserID
		}
		item.Pilot = pilot
	}

	return &item, nil
}

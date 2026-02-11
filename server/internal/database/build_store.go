package database

import (
	"context"
	"database/sql"
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
		INSERT INTO builds (owner_user_id, status, token, expires_at, title, description, source_aircraft_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
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
		WHERE b.owner_user_id = $1 AND b.status != 'TEMP'
	`
	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, ownerUserID).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count owner builds: %w", err)
	}

	query := `
		SELECT
			b.id,
			b.owner_user_id,
			b.status,
			b.token,
			b.expires_at,
			b.title,
			b.description,
			b.source_aircraft_id,
			b.created_at,
			b.updated_at,
			b.published_at,
			u.id,
			u.call_sign,
			COALESCE(NULLIF(u.display_name, ''), NULLIF(u.google_name, ''), NULLIF(u.call_sign, ''), 'Pilot'),
			COALESCE(u.profile_visibility, 'public') = 'public'
		FROM builds b
		LEFT JOIN users u ON b.owner_user_id = u.id
		WHERE b.owner_user_id = $1 AND b.status != 'TEMP'
		ORDER BY b.updated_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := s.db.QueryContext(ctx, query, ownerUserID, params.Limit, params.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list owner builds: %w", err)
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

	return &models.BuildListResponse{
		Builds:     builds,
		TotalCount: totalCount,
		Sort:       models.BuildSortNewest,
	}, nil
}

// ListPublic returns published builds for browsing.
func (s *BuildStore) ListPublic(ctx context.Context, params models.BuildListParams) (*models.BuildListResponse, error) {
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
			b.status,
			b.token,
			b.expires_at,
			b.title,
			b.description,
			b.source_aircraft_id,
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

	return &models.BuildListResponse{
		Builds:      builds,
		TotalCount:  totalCount,
		Sort:        params.Sort,
		FrameFilter: strings.TrimSpace(params.FrameFilter),
	}, nil
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
	return build, nil
}

// GetForOwner returns a build that belongs to the supplied owner.
func (s *BuildStore) GetForOwner(ctx context.Context, id string, ownerUserID string) (*models.Build, error) {
	query := baseBuildSelect + ` WHERE b.id = $1 AND b.owner_user_id = $2`
	build, err := s.scanBuild(ctx, query, id, ownerUserID)
	if err != nil || build == nil {
		return build, err
	}
	if err := s.attachParts(ctx, []*models.Build{build}); err != nil {
		return nil, err
	}
	return build, nil
}

// GetPublic returns a published build.
func (s *BuildStore) GetPublic(ctx context.Context, id string) (*models.Build, error) {
	query := baseBuildSelect + ` WHERE b.id = $1 AND b.status = 'PUBLISHED'`
	build, err := s.scanBuild(ctx, query, id)
	if err != nil || build == nil {
		return build, err
	}
	if err := s.attachParts(ctx, []*models.Build{build}); err != nil {
		return nil, err
	}
	return build, nil
}

// GetTempByToken fetches an unexpired temp build by secret token.
func (s *BuildStore) GetTempByToken(ctx context.Context, token string) (*models.Build, error) {
	query := baseBuildSelect + ` WHERE b.token = $1 AND b.status = 'TEMP' AND (b.expires_at IS NULL OR b.expires_at > NOW())`
	build, err := s.scanBuild(ctx, query, token)
	if err != nil || build == nil {
		return build, err
	}
	if err := s.attachParts(ctx, []*models.Build{build}); err != nil {
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

	query := fmt.Sprintf(`
		UPDATE builds
		SET %s
		WHERE id = $%d AND owner_user_id = $%d AND status != 'TEMP'
	`, strings.Join(setClauses, ", "), argIndex, argIndex+1)
	args = append(args, id, ownerUserID)

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update build: %w", err)
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, nil
	}

	if params.Parts != nil {
		if err := s.replacePartsTx(ctx, tx, id, params.Parts); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit build update: %w", err)
	}

	return s.GetForOwner(ctx, id, ownerUserID)
}

// UpdateTempByToken updates title/description/parts for a temp build.
func (s *BuildStore) UpdateTempByToken(ctx context.Context, token string, params models.UpdateBuildParams) (*models.Build, error) {
	build, err := s.GetTempByToken(ctx, token)
	if err != nil || build == nil {
		return build, err
	}

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

	query := fmt.Sprintf(`
		UPDATE builds
		SET %s
		WHERE id = $%d AND status = 'TEMP'
	`, strings.Join(setClauses, ", "), argIndex)
	args = append(args, build.ID)

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("failed to update temp build: %w", err)
	}

	if params.Parts != nil {
		if err := s.replacePartsTx(ctx, tx, build.ID, params.Parts); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit temp build update: %w", err)
	}

	return s.GetTempByToken(ctx, token)
}

// SetStatus updates a build's publication status.
func (s *BuildStore) SetStatus(ctx context.Context, id string, ownerUserID string, status models.BuildStatus) (*models.Build, error) {
	status = models.NormalizeBuildStatus(status)
	var query string

	switch status {
	case models.BuildStatusPublished:
		query = `
			UPDATE builds
			SET status = 'PUBLISHED', published_at = NOW(), updated_at = NOW()
			WHERE id = $1 AND owner_user_id = $2 AND status != 'TEMP'
		`
	case models.BuildStatusUnpublished:
		query = `
			UPDATE builds
			SET status = 'UNPUBLISHED', published_at = NULL, updated_at = NOW()
			WHERE id = $1 AND owner_user_id = $2 AND status = 'PUBLISHED'
		`
	default:
		return nil, fmt.Errorf("unsupported status transition to %q", status)
	}

	result, err := s.db.ExecContext(ctx, query, id, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to update build status: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, nil
	}

	return s.GetForOwner(ctx, id, ownerUserID)
}

// Delete removes a non-temp build for the owner.
func (s *BuildStore) Delete(ctx context.Context, id string, ownerUserID string) (bool, error) {
	result, err := s.db.ExecContext(
		ctx,
		`DELETE FROM builds WHERE id = $1 AND owner_user_id = $2 AND status != 'TEMP'`,
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
			gc.status,
			CASE
				WHEN gc.image_url IS NOT NULL AND gc.image_url != '' THEN gc.image_url
				WHEN (gc.image_asset_id IS NOT NULL OR gc.image_data IS NOT NULL) AND COALESCE(gc.image_status, 'missing') IN ('approved', 'scanned')
					THEN '/api/gear-catalog/' || gc.id || '/image?v=' || (EXTRACT(EPOCH FROM COALESCE(gc.image_curated_at, gc.updated_at))*1000)::bigint
				ELSE NULL
			END AS image_url
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

var baseBuildSelect = `
	SELECT
		b.id,
		b.owner_user_id,
		b.status,
		b.token,
		b.expires_at,
		b.title,
		b.description,
		b.source_aircraft_id,
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

func scanBuildRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.Build, error) {
	var item models.Build
	var ownerUserID sql.NullString
	var token sql.NullString
	var expiresAt sql.NullTime
	var description sql.NullString
	var sourceAircraftID sql.NullString
	var publishedAt sql.NullTime

	var pilotUserID sql.NullString
	var pilotCallSign sql.NullString
	var pilotDisplayName sql.NullString
	var pilotIsPublic sql.NullBool

	err := scanner.Scan(
		&item.ID,
		&ownerUserID,
		&item.Status,
		&token,
		&expiresAt,
		&item.Title,
		&description,
		&sourceAircraftID,
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
	item.Token = token.String
	item.Description = description.String
	item.SourceAircraftID = sourceAircraftID.String
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

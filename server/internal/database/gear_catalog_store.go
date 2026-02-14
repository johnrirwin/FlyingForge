package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/johnrirwin/flyingforge/internal/models"
)

// GearCatalogStore handles gear catalog database operations
type GearCatalogStore struct {
	db *DB
}

var ErrCatalogItemNotFound = errors.New("catalog item not found")
var ErrCatalogImageAlreadyCurated = errors.New("catalog image already curated")
var ErrCatalogImageMissing = errors.New("catalog image missing")

// NewGearCatalogStore creates a new gear catalog store
func NewGearCatalogStore(db *DB) *GearCatalogStore {
	return &GearCatalogStore{db: db}
}

// Create inserts a new catalog item or returns existing if canonical_key matches
func (s *GearCatalogStore) Create(ctx context.Context, userID string, params models.CreateGearCatalogParams) (*models.GearCatalogCreateResponse, error) {
	// Build canonical key
	canonicalKey := models.BuildCanonicalKey(params.GearType, params.Brand, params.Model, params.Variant)

	// First, check if an item with this canonical key already exists
	existing, err := s.GetByCanonicalKey(ctx, canonicalKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing item: %w", err)
	}

	if existing != nil {
		return &models.GearCatalogCreateResponse{
			Item:     existing,
			Existing: true,
		}, nil
	}

	// No existing item, create new one
	specs := params.Specs
	if specs == nil {
		specs = json.RawMessage(`{}`)
	}

	query := `
		INSERT INTO gear_catalog (
			gear_type, brand, model, variant, specs, best_for, msrp, source,
			created_by_user_id, status, canonical_key, description,
			image_status, description_status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, created_at, updated_at
	`

	item := &models.GearCatalogItem{
		GearType:          params.GearType,
		Brand:             strings.TrimSpace(params.Brand),
		Model:             strings.TrimSpace(params.Model),
		Variant:           strings.TrimSpace(params.Variant),
		Specs:             specs,
		BestFor:           params.BestFor,
		MSRP:              params.MSRP,
		Source:            models.CatalogSourceUserSubmitted,
		CreatedByUserID:   userID,
		Status:            models.CatalogStatusPending,
		CanonicalKey:      canonicalKey,
		Description:       params.Description,
		ImageStatus:       models.ImageStatusMissing,
		DescriptionStatus: models.ImageStatusMissing,
	}

	// Set description status based on whether description was provided
	descriptionStatus := models.ImageStatusMissing
	if strings.TrimSpace(params.Description) != "" {
		descriptionStatus = models.ImageStatusApproved
		item.DescriptionStatus = descriptionStatus
	}

	var createdByUserIDPtr *string
	if userID != "" {
		createdByUserIDPtr = &userID
	}

	err = s.db.QueryRowContext(ctx, query,
		item.GearType, item.Brand, item.Model, nullString(item.Variant),
		item.Specs, pq.Array(item.BestFor), item.MSRP, item.Source, createdByUserIDPtr, item.Status,
		item.CanonicalKey, nullString(item.Description),
		item.ImageStatus, descriptionStatus,
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)

	if err != nil {
		// Handle unique constraint violation (race condition)
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") {
			existing, err2 := s.GetByCanonicalKey(ctx, canonicalKey)
			if err2 == nil && existing != nil {
				return &models.GearCatalogCreateResponse{
					Item:     existing,
					Existing: true,
				}, nil
			}
		}
		return nil, fmt.Errorf("failed to insert catalog item: %w", err)
	}

	return &models.GearCatalogCreateResponse{
		Item:     item,
		Existing: false,
	}, nil
}

// Get retrieves a catalog item by ID
func (s *GearCatalogStore) Get(ctx context.Context, id string) (*models.GearCatalogItem, error) {
	query := `
		SELECT id, gear_type, brand, model, variant, specs, best_for, msrp, source,
			   created_by_user_id, status, canonical_key,
			   CASE WHEN image_asset_id IS NOT NULL OR image_data IS NOT NULL THEN '/api/gear-catalog/' || id || '/image?v=' || (EXTRACT(EPOCH FROM COALESCE(image_curated_at, updated_at))*1000)::bigint ELSE NULL END as image_url,
			   description,
			   created_at, updated_at,
			   (SELECT COUNT(*) FROM inventory_items WHERE catalog_id = gear_catalog.id) as usage_count,
			   COALESCE(image_status, 'missing'), image_curated_by_user_id, image_curated_at,
			   COALESCE(description_status, 'missing'), description_curated_by_user_id, description_curated_at
		FROM gear_catalog
		WHERE id = $1
	`

	item := &models.GearCatalogItem{}
	var variant, imageURL, description, createdByUserID sql.NullString
	var imageCuratedByUserID, descriptionCuratedByUserID sql.NullString
	var imageCuratedAt, descriptionCuratedAt sql.NullTime
	var msrp sql.NullFloat64

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&item.ID, &item.GearType, &item.Brand, &item.Model, &variant,
		&item.Specs, pq.Array(&item.BestFor), &msrp, &item.Source, &createdByUserID, &item.Status,
		&item.CanonicalKey, &imageURL, &description,
		&item.CreatedAt, &item.UpdatedAt, &item.UsageCount,
		&item.ImageStatus, &imageCuratedByUserID, &imageCuratedAt,
		&item.DescriptionStatus, &descriptionCuratedByUserID, &descriptionCuratedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get catalog item: %w", err)
	}

	item.Variant = variant.String
	item.ImageURL = imageURL.String
	item.Description = description.String
	item.CreatedByUserID = createdByUserID.String
	if msrp.Valid {
		item.MSRP = &msrp.Float64
	}
	if imageCuratedByUserID.Valid {
		item.ImageCuratedByUserID = imageCuratedByUserID.String
	}
	if imageCuratedAt.Valid {
		item.ImageCuratedAt = &imageCuratedAt.Time
	}
	if descriptionCuratedByUserID.Valid {
		item.DescriptionCuratedByUserID = descriptionCuratedByUserID.String
	}
	if descriptionCuratedAt.Valid {
		item.DescriptionCuratedAt = &descriptionCuratedAt.Time
	}

	return item, nil
}

// GetByCanonicalKey retrieves a catalog item by its canonical key
func (s *GearCatalogStore) GetByCanonicalKey(ctx context.Context, canonicalKey string) (*models.GearCatalogItem, error) {
	query := `
		SELECT id, gear_type, brand, model, variant, specs, best_for, msrp, source,
			   created_by_user_id, status, canonical_key,
			   CASE WHEN image_asset_id IS NOT NULL OR image_data IS NOT NULL THEN '/api/gear-catalog/' || id || '/image?v=' || (EXTRACT(EPOCH FROM COALESCE(image_curated_at, updated_at))*1000)::bigint ELSE NULL END as image_url,
			   description,
			   created_at, updated_at,
			   (SELECT COUNT(*) FROM inventory_items WHERE catalog_id = gear_catalog.id) as usage_count,
			   COALESCE(image_status, 'missing'), image_curated_by_user_id, image_curated_at,
			   COALESCE(description_status, 'missing'), description_curated_by_user_id, description_curated_at
		FROM gear_catalog
		WHERE canonical_key = $1
	`

	item := &models.GearCatalogItem{}
	var variant, imageURL, description, createdByUserID sql.NullString
	var imageCuratedByUserID, descriptionCuratedByUserID sql.NullString
	var imageCuratedAt, descriptionCuratedAt sql.NullTime
	var msrp sql.NullFloat64

	err := s.db.QueryRowContext(ctx, query, canonicalKey).Scan(
		&item.ID, &item.GearType, &item.Brand, &item.Model, &variant,
		&item.Specs, pq.Array(&item.BestFor), &msrp, &item.Source, &createdByUserID, &item.Status,
		&item.CanonicalKey, &imageURL, &description,
		&item.CreatedAt, &item.UpdatedAt, &item.UsageCount,
		&item.ImageStatus, &imageCuratedByUserID, &imageCuratedAt,
		&item.DescriptionStatus, &descriptionCuratedByUserID, &descriptionCuratedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get catalog item by canonical key: %w", err)
	}

	item.Variant = variant.String
	item.ImageURL = imageURL.String
	item.Description = description.String
	item.CreatedByUserID = createdByUserID.String
	if msrp.Valid {
		item.MSRP = &msrp.Float64
	}
	if imageCuratedByUserID.Valid {
		item.ImageCuratedByUserID = imageCuratedByUserID.String
	}
	if imageCuratedAt.Valid {
		item.ImageCuratedAt = &imageCuratedAt.Time
	}
	if descriptionCuratedByUserID.Valid {
		item.DescriptionCuratedByUserID = descriptionCuratedByUserID.String
	}
	if descriptionCuratedAt.Valid {
		item.DescriptionCuratedAt = &descriptionCuratedAt.Time
	}

	return item, nil
}

// Search searches the catalog with various filters
func (s *GearCatalogStore) Search(ctx context.Context, params models.GearCatalogSearchParams) (*models.GearCatalogSearchResponse, error) {
	// Default limit
	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	// Build WHERE clauses
	whereClauses := []string{"status = 'published'"}
	args := []interface{}{}
	argIdx := 1

	if params.GearType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("gear_type = $%d", argIdx))
		args = append(args, params.GearType)
		argIdx++
	}

	if params.Brand != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("LOWER(brand) = LOWER($%d)", argIdx))
		args = append(args, params.Brand)
		argIdx++
	}

	if params.Status != "" {
		normalizedStatus := models.NormalizeCatalogStatus(params.Status)
		if !models.IsValidCatalogStatus(normalizedStatus) {
			return nil, fmt.Errorf("invalid catalog status %q", params.Status)
		}
		// Override the default published-only status filter
		whereClauses[0] = fmt.Sprintf("status = $%d", argIdx)
		args = append(args, normalizedStatus)
		argIdx++
	}

	// Text search
	var orderBy string
	if params.Query != "" {
		// Use both full-text search and ILIKE for flexibility
		searchClause := fmt.Sprintf(`(
			to_tsvector('english', brand || ' ' || model || ' ' || COALESCE(variant, '')) @@ plainto_tsquery('english', $%d)
			OR LOWER(brand || ' ' || model || ' ' || COALESCE(variant, '')) LIKE LOWER($%d)
		)`, argIdx, argIdx+1)
		whereClauses = append(whereClauses, searchClause)
		args = append(args, params.Query, "%"+params.Query+"%")
		argIdx += 2

		// Order by relevance when searching
		orderBy = fmt.Sprintf(`
			ts_rank(to_tsvector('english', brand || ' ' || model || ' ' || COALESCE(variant, '')), plainto_tsquery('english', $%d)) DESC,
			(SELECT COUNT(*) FROM inventory_items WHERE catalog_id = gear_catalog.id) DESC,
			brand, model
		`, argIdx)
		args = append(args, params.Query)
		argIdx++
	} else {
		// Default ordering: most used first, then alphabetical
		orderBy = "(SELECT COUNT(*) FROM inventory_items WHERE catalog_id = gear_catalog.id) DESC, brand, model"
	}

	whereClause := strings.Join(whereClauses, " AND ")

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM gear_catalog WHERE %s", whereClause)
	var totalCount int
	// Use only the non-orderBy args for count
	countArgs := args[:]
	if params.Query != "" {
		countArgs = args[:len(args)-1] // Exclude the last orderBy arg
	}
	if err := s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count catalog items: %w", err)
	}

	// Main query
	query := fmt.Sprintf(`
		SELECT id, gear_type, brand, model, variant, specs, best_for, msrp, source,
			   created_by_user_id, status, canonical_key,
			   CASE WHEN image_asset_id IS NOT NULL OR image_data IS NOT NULL THEN '/api/gear-catalog/' || id || '/image?v=' || (EXTRACT(EPOCH FROM COALESCE(image_curated_at, updated_at))*1000)::bigint ELSE NULL END as image_url,
			   description,
			   created_at, updated_at,
			   (SELECT COUNT(*) FROM inventory_items WHERE catalog_id = gear_catalog.id) as usage_count,
			   COALESCE(image_status, 'missing'), image_curated_by_user_id, image_curated_at,
			   COALESCE(description_status, 'missing'), description_curated_by_user_id, description_curated_at
		FROM gear_catalog
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, argIdx, argIdx+1)

	args = append(args, params.Limit, params.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search catalog: %w", err)
	}
	defer rows.Close()

	items := make([]models.GearCatalogItem, 0)
	for rows.Next() {
		var item models.GearCatalogItem
		var variant, imageURL, description, createdByUserID sql.NullString
		var imageCuratedByUserID, descriptionCuratedByUserID sql.NullString
		var imageCuratedAt, descriptionCuratedAt sql.NullTime
		var msrp sql.NullFloat64

		if err := rows.Scan(
			&item.ID, &item.GearType, &item.Brand, &item.Model, &variant,
			&item.Specs, pq.Array(&item.BestFor), &msrp, &item.Source, &createdByUserID, &item.Status,
			&item.CanonicalKey, &imageURL, &description,
			&item.CreatedAt, &item.UpdatedAt, &item.UsageCount,
			&item.ImageStatus, &imageCuratedByUserID, &imageCuratedAt,
			&item.DescriptionStatus, &descriptionCuratedByUserID, &descriptionCuratedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan catalog item: %w", err)
		}

		item.Variant = variant.String
		item.ImageURL = imageURL.String
		item.Description = description.String
		item.CreatedByUserID = createdByUserID.String
		if msrp.Valid {
			item.MSRP = &msrp.Float64
		}
		if imageCuratedByUserID.Valid {
			item.ImageCuratedByUserID = imageCuratedByUserID.String
		}
		if imageCuratedAt.Valid {
			item.ImageCuratedAt = &imageCuratedAt.Time
		}
		if descriptionCuratedByUserID.Valid {
			item.DescriptionCuratedByUserID = descriptionCuratedByUserID.String
		}
		if descriptionCuratedAt.Valid {
			item.DescriptionCuratedAt = &descriptionCuratedAt.Time
		}

		items = append(items, item)
	}

	return &models.GearCatalogSearchResponse{
		Items:      items,
		TotalCount: totalCount,
		Query:      params.Query,
	}, nil
}

// FindNearMatches finds potential duplicate items using similarity search
func (s *GearCatalogStore) FindNearMatches(ctx context.Context, gearType models.GearType, brand, model string, threshold float64) ([]models.NearMatch, error) {
	if threshold <= 0 {
		threshold = 0.3 // Default similarity threshold
	}

	// Try to use pg_trgm similarity if available, fall back to ILIKE
	query := `
		SELECT id, gear_type, brand, model, variant, specs, source,
			   created_by_user_id, status, canonical_key,
			   CASE WHEN image_asset_id IS NOT NULL OR image_data IS NOT NULL THEN '/api/gear-catalog/' || id || '/image?v=' || (EXTRACT(EPOCH FROM updated_at)*1000)::bigint ELSE NULL END as image_url,
			   description,
			   created_at, updated_at,
			   (SELECT COUNT(*) FROM inventory_items WHERE catalog_id = gear_catalog.id) as usage_count,
			   COALESCE(similarity(LOWER(brand || ' ' || model), LOWER($2 || ' ' || $3)), 0) as sim_score
		FROM gear_catalog
		WHERE gear_type = $1
		  AND status = 'published'
		  AND (
			COALESCE(similarity(LOWER(brand || ' ' || model), LOWER($2 || ' ' || $3)), 0) >= $4
			OR LOWER(brand) = LOWER($2)
			OR LOWER(model) LIKE LOWER('%' || $3 || '%')
		  )
		ORDER BY sim_score DESC
		LIMIT 10
	`

	rows, err := s.db.QueryContext(ctx, query, gearType, brand, model, threshold)
	if err != nil {
		// If pg_trgm is not available, fall back to simpler matching
		if strings.Contains(err.Error(), "function similarity") || strings.Contains(err.Error(), "does not exist") {
			return s.findNearMatchesFallback(ctx, gearType, brand, model)
		}
		return nil, fmt.Errorf("failed to find near matches: %w", err)
	}
	defer rows.Close()

	matches := make([]models.NearMatch, 0)
	for rows.Next() {
		var item models.GearCatalogItem
		var variant, imageURL, description, createdByUserID sql.NullString
		var simScore float64

		if err := rows.Scan(
			&item.ID, &item.GearType, &item.Brand, &item.Model, &variant,
			&item.Specs, &item.Source, &createdByUserID, &item.Status,
			&item.CanonicalKey, &imageURL, &description,
			&item.CreatedAt, &item.UpdatedAt, &item.UsageCount, &simScore,
		); err != nil {
			return nil, fmt.Errorf("failed to scan near match: %w", err)
		}

		item.Variant = variant.String
		item.ImageURL = imageURL.String
		item.Description = description.String
		item.CreatedByUserID = createdByUserID.String

		matches = append(matches, models.NearMatch{
			Item:       item,
			Similarity: simScore,
		})
	}

	return matches, nil
}

// findNearMatchesFallback is used when pg_trgm is not available
func (s *GearCatalogStore) findNearMatchesFallback(ctx context.Context, gearType models.GearType, brand, model string) ([]models.NearMatch, error) {
	query := `
		SELECT id, gear_type, brand, model, variant, specs, source,
			   created_by_user_id, status, canonical_key,
			   CASE WHEN image_asset_id IS NOT NULL OR image_data IS NOT NULL THEN '/api/gear-catalog/' || id || '/image?v=' || (EXTRACT(EPOCH FROM COALESCE(image_curated_at, updated_at))*1000)::bigint ELSE NULL END as image_url,
			   description,
			   created_at, updated_at,
			   (SELECT COUNT(*) FROM inventory_items WHERE catalog_id = gear_catalog.id) as usage_count
		FROM gear_catalog
		WHERE gear_type = $1
		  AND status = 'published'
		  AND (
			LOWER(brand) = LOWER($2)
			OR LOWER(model) LIKE LOWER('%' || $3 || '%')
			OR LOWER(brand || ' ' || model) LIKE LOWER('%' || $2 || '%' || $3 || '%')
		  )
		ORDER BY 
			CASE WHEN LOWER(brand) = LOWER($2) THEN 0 ELSE 1 END,
			brand, model
		LIMIT 10
	`

	rows, err := s.db.QueryContext(ctx, query, gearType, brand, model)
	if err != nil {
		return nil, fmt.Errorf("failed to find near matches (fallback): %w", err)
	}
	defer rows.Close()

	matches := make([]models.NearMatch, 0)
	for rows.Next() {
		var item models.GearCatalogItem
		var variant, imageURL, description, createdByUserID sql.NullString

		if err := rows.Scan(
			&item.ID, &item.GearType, &item.Brand, &item.Model, &variant,
			&item.Specs, &item.Source, &createdByUserID, &item.Status,
			&item.CanonicalKey, &imageURL, &description,
			&item.CreatedAt, &item.UpdatedAt, &item.UsageCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan near match: %w", err)
		}

		item.Variant = variant.String
		item.ImageURL = imageURL.String
		item.Description = description.String
		item.CreatedByUserID = createdByUserID.String

		// Estimate similarity based on string matching
		similarity := 0.5 // Base similarity for matching items
		if strings.EqualFold(item.Brand, brand) {
			similarity += 0.25
		}
		if strings.Contains(strings.ToLower(item.Model), strings.ToLower(model)) {
			similarity += 0.25
		}

		matches = append(matches, models.NearMatch{
			Item:       item,
			Similarity: similarity,
		})
	}

	return matches, nil
}

// FindNearMatchesAdmin finds potential duplicates across all catalog statuses.
// This is used for admin/import workflows so editors can see already-imported/pending/removed items.
func (s *GearCatalogStore) FindNearMatchesAdmin(ctx context.Context, gearType models.GearType, brand, model string, threshold float64) ([]models.NearMatch, error) {
	if threshold <= 0 {
		threshold = 0.3 // Default similarity threshold
	}

	// Try to use pg_trgm similarity if available, fall back to ILIKE.
	// NOTE: Unlike FindNearMatches, we do not filter by status.
	query := `
		SELECT id, gear_type, brand, model, variant, specs, best_for, msrp, source,
			   created_by_user_id, status, canonical_key,
			   CASE WHEN image_asset_id IS NOT NULL OR image_data IS NOT NULL THEN '/api/gear-catalog/' || id || '/image?v=' || (EXTRACT(EPOCH FROM COALESCE(image_curated_at, updated_at))*1000)::bigint ELSE NULL END as image_url,
			   description,
			   created_at, updated_at,
			   (SELECT COUNT(*) FROM inventory_items WHERE catalog_id = gear_catalog.id) as usage_count,
			   COALESCE(image_status, 'missing'), image_curated_by_user_id, image_curated_at,
			   COALESCE(description_status, 'missing'), description_curated_by_user_id, description_curated_at,
			   COALESCE(similarity(LOWER(brand || ' ' || model), LOWER($2 || ' ' || $3)), 0) as sim_score
		FROM gear_catalog
		WHERE gear_type = $1
		  AND (
			COALESCE(similarity(LOWER(brand || ' ' || model), LOWER($2 || ' ' || $3)), 0) >= $4
			OR LOWER(brand) = LOWER($2)
			OR LOWER(model) LIKE LOWER('%' || $3 || '%')
		  )
		ORDER BY sim_score DESC
		LIMIT 10
	`

	rows, err := s.db.QueryContext(ctx, query, gearType, brand, model, threshold)
	if err != nil {
		// If pg_trgm is not available, fall back to simpler matching
		if strings.Contains(err.Error(), "function similarity") || strings.Contains(err.Error(), "does not exist") {
			return s.findNearMatchesAdminFallback(ctx, gearType, brand, model)
		}
		return nil, fmt.Errorf("failed to find near matches (admin): %w", err)
	}
	defer rows.Close()

	matches := make([]models.NearMatch, 0)
	for rows.Next() {
		var item models.GearCatalogItem
		var variant, imageURL, description, createdByUserID sql.NullString
		var imageCuratedByUserID, descriptionCuratedByUserID sql.NullString
		var imageCuratedAt, descriptionCuratedAt sql.NullTime
		var msrp sql.NullFloat64
		var simScore float64

		if err := rows.Scan(
			&item.ID, &item.GearType, &item.Brand, &item.Model, &variant,
			&item.Specs, pq.Array(&item.BestFor), &msrp, &item.Source, &createdByUserID, &item.Status,
			&item.CanonicalKey, &imageURL, &description,
			&item.CreatedAt, &item.UpdatedAt, &item.UsageCount,
			&item.ImageStatus, &imageCuratedByUserID, &imageCuratedAt,
			&item.DescriptionStatus, &descriptionCuratedByUserID, &descriptionCuratedAt,
			&simScore,
		); err != nil {
			return nil, fmt.Errorf("failed to scan near match (admin): %w", err)
		}

		item.Variant = variant.String
		item.ImageURL = imageURL.String
		item.Description = description.String
		item.CreatedByUserID = createdByUserID.String
		if msrp.Valid {
			item.MSRP = &msrp.Float64
		}
		if imageCuratedByUserID.Valid {
			item.ImageCuratedByUserID = imageCuratedByUserID.String
		}
		if imageCuratedAt.Valid {
			item.ImageCuratedAt = &imageCuratedAt.Time
		}
		if descriptionCuratedByUserID.Valid {
			item.DescriptionCuratedByUserID = descriptionCuratedByUserID.String
		}
		if descriptionCuratedAt.Valid {
			item.DescriptionCuratedAt = &descriptionCuratedAt.Time
		}

		matches = append(matches, models.NearMatch{
			Item:       item,
			Similarity: simScore,
		})
	}

	return matches, nil
}

func (s *GearCatalogStore) findNearMatchesAdminFallback(ctx context.Context, gearType models.GearType, brand, model string) ([]models.NearMatch, error) {
	query := `
		SELECT id, gear_type, brand, model, variant, specs, best_for, msrp, source,
			   created_by_user_id, status, canonical_key,
			   CASE WHEN image_asset_id IS NOT NULL OR image_data IS NOT NULL THEN '/api/gear-catalog/' || id || '/image?v=' || (EXTRACT(EPOCH FROM COALESCE(image_curated_at, updated_at))*1000)::bigint ELSE NULL END as image_url,
			   description,
			   created_at, updated_at,
			   (SELECT COUNT(*) FROM inventory_items WHERE catalog_id = gear_catalog.id) as usage_count,
			   COALESCE(image_status, 'missing'), image_curated_by_user_id, image_curated_at,
			   COALESCE(description_status, 'missing'), description_curated_by_user_id, description_curated_at
		FROM gear_catalog
		WHERE gear_type = $1
		  AND (
			LOWER(brand) = LOWER($2)
			OR LOWER(model) LIKE LOWER('%' || $3 || '%')
			OR LOWER(brand || ' ' || model) LIKE LOWER('%' || $2 || '%' || $3 || '%')
		  )
		ORDER BY 
			CASE WHEN LOWER(brand) = LOWER($2) THEN 0 ELSE 1 END,
			brand, model
		LIMIT 10
	`

	rows, err := s.db.QueryContext(ctx, query, gearType, brand, model)
	if err != nil {
		return nil, fmt.Errorf("failed to find near matches (admin fallback): %w", err)
	}
	defer rows.Close()

	matches := make([]models.NearMatch, 0)
	for rows.Next() {
		var item models.GearCatalogItem
		var variant, imageURL, description, createdByUserID sql.NullString
		var imageCuratedByUserID, descriptionCuratedByUserID sql.NullString
		var imageCuratedAt, descriptionCuratedAt sql.NullTime
		var msrp sql.NullFloat64

		if err := rows.Scan(
			&item.ID, &item.GearType, &item.Brand, &item.Model, &variant,
			&item.Specs, pq.Array(&item.BestFor), &msrp, &item.Source, &createdByUserID, &item.Status,
			&item.CanonicalKey, &imageURL, &description,
			&item.CreatedAt, &item.UpdatedAt, &item.UsageCount,
			&item.ImageStatus, &imageCuratedByUserID, &imageCuratedAt,
			&item.DescriptionStatus, &descriptionCuratedByUserID, &descriptionCuratedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan near match (admin fallback): %w", err)
		}

		item.Variant = variant.String
		item.ImageURL = imageURL.String
		item.Description = description.String
		item.CreatedByUserID = createdByUserID.String
		if msrp.Valid {
			item.MSRP = &msrp.Float64
		}
		if imageCuratedByUserID.Valid {
			item.ImageCuratedByUserID = imageCuratedByUserID.String
		}
		if imageCuratedAt.Valid {
			item.ImageCuratedAt = &imageCuratedAt.Time
		}
		if descriptionCuratedByUserID.Valid {
			item.DescriptionCuratedByUserID = descriptionCuratedByUserID.String
		}
		if descriptionCuratedAt.Valid {
			item.DescriptionCuratedAt = &descriptionCuratedAt.Time
		}

		// Estimate similarity based on string matching (keep consistent with FindNearMatches fallback).
		similarity := 0.5 // Base similarity for matching items
		if strings.EqualFold(item.Brand, brand) {
			similarity += 0.25
		}
		if strings.Contains(strings.ToLower(item.Model), strings.ToLower(model)) {
			similarity += 0.25
		}

		matches = append(matches, models.NearMatch{
			Item:       item,
			Similarity: similarity,
		})
	}

	return matches, nil
}

// UpdateStatus updates the status of a catalog item (for moderation)
func (s *GearCatalogStore) UpdateStatus(ctx context.Context, id string, status models.CatalogItemStatus) error {
	query := `UPDATE gear_catalog SET status = $1, updated_at = NOW() WHERE id = $2`

	result, err := s.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update catalog item status: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("catalog item not found: %s", id)
	}

	return nil
}

// GetPopular returns the most used catalog items
func (s *GearCatalogStore) GetPopular(ctx context.Context, gearType models.GearType, limit int) ([]models.GearCatalogItem, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT id, gear_type, brand, model, variant, specs, best_for, msrp, source,
			   created_by_user_id, status, canonical_key,
			   CASE WHEN image_asset_id IS NOT NULL OR image_data IS NOT NULL THEN '/api/gear-catalog/' || id || '/image?v=' || (EXTRACT(EPOCH FROM COALESCE(image_curated_at, updated_at))*1000)::bigint ELSE NULL END as image_url,
			   description,
			   created_at, updated_at,
			   (SELECT COUNT(*) FROM inventory_items WHERE catalog_id = gear_catalog.id) as usage_count,
			   COALESCE(image_status, 'missing'), image_curated_by_user_id, image_curated_at,
			   COALESCE(description_status, 'missing'), description_curated_by_user_id, description_curated_at
		FROM gear_catalog
		WHERE status = 'published'
		  AND ($1 = '' OR gear_type = $1)
		ORDER BY usage_count DESC, brand, model
		LIMIT $2
	`

	gearTypeStr := ""
	if gearType != "" {
		gearTypeStr = string(gearType)
	}

	rows, err := s.db.QueryContext(ctx, query, gearTypeStr, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular catalog items: %w", err)
	}
	defer rows.Close()

	items := make([]models.GearCatalogItem, 0)
	for rows.Next() {
		var item models.GearCatalogItem
		var variant, imageURL, description, createdByUserID sql.NullString
		var imageCuratedByUserID, descriptionCuratedByUserID sql.NullString
		var imageCuratedAt, descriptionCuratedAt sql.NullTime
		var msrp sql.NullFloat64

		if err := rows.Scan(
			&item.ID, &item.GearType, &item.Brand, &item.Model, &variant,
			&item.Specs, pq.Array(&item.BestFor), &msrp, &item.Source, &createdByUserID, &item.Status,
			&item.CanonicalKey, &imageURL, &description,
			&item.CreatedAt, &item.UpdatedAt, &item.UsageCount,
			&item.ImageStatus, &imageCuratedByUserID, &imageCuratedAt,
			&item.DescriptionStatus, &descriptionCuratedByUserID, &descriptionCuratedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan popular item: %w", err)
		}

		item.Variant = variant.String
		item.ImageURL = imageURL.String
		item.Description = description.String
		item.CreatedByUserID = createdByUserID.String
		if msrp.Valid {
			item.MSRP = &msrp.Float64
		}
		if imageCuratedByUserID.Valid {
			item.ImageCuratedByUserID = imageCuratedByUserID.String
		}
		if imageCuratedAt.Valid {
			item.ImageCuratedAt = &imageCuratedAt.Time
		}
		if descriptionCuratedByUserID.Valid {
			item.DescriptionCuratedByUserID = descriptionCuratedByUserID.String
		}
		if descriptionCuratedAt.Valid {
			item.DescriptionCuratedAt = &descriptionCuratedAt.Time
		}

		items = append(items, item)
	}

	return items, nil
}

// MigrateInventoryItem creates a catalog entry from an existing inventory item
// and links the inventory item to it. Uses a transaction to ensure consistency.
// Note: Does NOT copy image data from inventory - catalog images require admin curation.
func (s *GearCatalogStore) MigrateInventoryItem(ctx context.Context, inventoryItemID, userID, name, manufacturer string, category models.EquipmentCategory, specs json.RawMessage) (*models.GearCatalogItem, error) {
	// Start a transaction to ensure atomic create+link
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Convert category to gear type
	gearType := models.GearTypeFromEquipmentCategory(category)

	// Extract brand, model, variant from the item name
	brand, model, variant := models.ExtractBrandModelFromName(name, manufacturer)
	if brand == "" {
		brand = "Unknown"
	}
	if model == "" {
		model = name
	}

	// Generate canonical key
	canonicalKey := models.BuildCanonicalKey(gearType, brand, model, variant)

	// First check if this catalog item already exists
	checkQuery := `SELECT id FROM gear_catalog WHERE canonical_key = $1`
	var catalogID string
	err = tx.QueryRowContext(ctx, checkQuery, canonicalKey).Scan(&catalogID)

	if err == sql.ErrNoRows {
		// Create new catalog entry - image_status='missing' enforces admin curation
		catalogID = uuid.New().String()
		insertQuery := `
			INSERT INTO gear_catalog (id, gear_type, brand, model, variant, specs, source, created_by_user_id, status, canonical_key, image_status, description_status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, 'user', $7, 'pending', $8, 'missing', 'missing', NOW(), NOW())
		`
		_, err = tx.ExecContext(ctx, insertQuery, catalogID, gearType, brand, model, variant, specs, userID, canonicalKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create catalog entry: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to check for existing catalog entry: %w", err)
	}

	// Update the inventory item to link to the catalog
	updateQuery := `UPDATE inventory_items SET catalog_id = $1, updated_at = NOW() WHERE id = $2`
	_, err = tx.ExecContext(ctx, updateQuery, catalogID, inventoryItemID)
	if err != nil {
		return nil, fmt.Errorf("failed to link inventory item to catalog: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Fetch the full catalog item to return
	return s.Get(ctx, catalogID)
}

// AdminSearch searches for gear items with admin-specific filters (like imageStatus)
func (s *GearCatalogStore) AdminSearch(ctx context.Context, params models.AdminGearSearchParams) (*models.GearCatalogSearchResponse, error) {
	// Default limit
	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Limit > 100 {
		params.Limit = 100
	}

	// Build WHERE clauses
	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if params.GearType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("gear_type = $%d", argIdx))
		args = append(args, params.GearType)
		argIdx++
	}

	if params.Brand != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("LOWER(brand) LIKE LOWER($%d)", argIdx))
		args = append(args, "%"+params.Brand+"%")
		argIdx++
	}

	if params.Status != "" {
		normalizedStatus := models.NormalizeCatalogStatus(params.Status)
		if !models.IsValidCatalogStatus(normalizedStatus) {
			return nil, fmt.Errorf("invalid catalog status %q", params.Status)
		}
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, normalizedStatus)
		argIdx++
	}

	if params.ImageStatus != "" {
		switch params.ImageStatus {
		case models.ImageStatusRecentlyCurated:
			// Special filter: items curated within last 24 hours
			whereClauses = append(whereClauses, "image_curated_at >= NOW() - INTERVAL '24 hours'")
		case models.ImageStatusAll:
			// Special filter: include all records (no curation-status WHERE clause)
		default:
			whereClauses = append(whereClauses, fmt.Sprintf("COALESCE(image_status, 'missing') = $%d", argIdx))
			args = append(args, params.ImageStatus)
			argIdx++
		}
	} else {
		// Default "Needs Work" view:
		// - items missing an image,
		// - items with a scanned (not-yet-curated) image,
		// - or items missing a description.
		whereClauses = append(whereClauses, "(COALESCE(image_status, 'missing') IN ('missing', 'scanned') OR COALESCE(description_status, 'missing') = 'missing')")
	}

	// Text search
	if params.Query != "" {
		searchClause := fmt.Sprintf(`(
			LOWER(brand || ' ' || model || ' ' || COALESCE(variant, '')) LIKE LOWER($%d)
		)`, argIdx)
		whereClauses = append(whereClauses, searchClause)
		args = append(args, "%"+params.Query+"%")
		argIdx++
	}

	whereClause := strings.Join(whereClauses, " AND ")

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM gear_catalog WHERE %s", whereClause)
	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count catalog items: %w", err)
	}

	// Order by: recently curated sorts by curation time, otherwise by creation time
	orderBy := "created_at DESC"
	if params.ImageStatus == models.ImageStatusRecentlyCurated {
		orderBy = "image_curated_at DESC"
	}

	// Main query - order by most recent first for admin review
	query := fmt.Sprintf(`
		SELECT id, gear_type, brand, model, variant, specs, best_for, msrp, source,
			   created_by_user_id, status, canonical_key,
			   CASE WHEN image_asset_id IS NOT NULL OR image_data IS NOT NULL THEN '/api/gear-catalog/' || id || '/image?v=' || (EXTRACT(EPOCH FROM COALESCE(image_curated_at, updated_at))*1000)::bigint ELSE NULL END as image_url,
			   description,
			   created_at, updated_at,
			   (SELECT COUNT(*) FROM inventory_items WHERE catalog_id = gear_catalog.id) as usage_count,
			   COALESCE(image_status, 'missing'), image_curated_by_user_id, image_curated_at,
			   COALESCE(description_status, 'missing'), description_curated_by_user_id, description_curated_at
		FROM gear_catalog
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, argIdx, argIdx+1)

	args = append(args, params.Limit, params.Offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to admin search catalog: %w", err)
	}
	defer rows.Close()

	items := make([]models.GearCatalogItem, 0)
	for rows.Next() {
		var item models.GearCatalogItem
		var variant, imageURL, description, createdByUserID sql.NullString
		var imageCuratedByUserID, descriptionCuratedByUserID sql.NullString
		var imageCuratedAt, descriptionCuratedAt sql.NullTime
		var msrp sql.NullFloat64

		if err := rows.Scan(
			&item.ID, &item.GearType, &item.Brand, &item.Model, &variant,
			&item.Specs, pq.Array(&item.BestFor), &msrp, &item.Source, &createdByUserID, &item.Status,
			&item.CanonicalKey, &imageURL, &description,
			&item.CreatedAt, &item.UpdatedAt, &item.UsageCount,
			&item.ImageStatus, &imageCuratedByUserID, &imageCuratedAt,
			&item.DescriptionStatus, &descriptionCuratedByUserID, &descriptionCuratedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan admin catalog item: %w", err)
		}

		item.Variant = variant.String
		item.ImageURL = imageURL.String
		item.Description = description.String
		item.CreatedByUserID = createdByUserID.String
		if msrp.Valid {
			item.MSRP = &msrp.Float64
		}
		if imageCuratedByUserID.Valid {
			item.ImageCuratedByUserID = imageCuratedByUserID.String
		}
		if imageCuratedAt.Valid {
			item.ImageCuratedAt = &imageCuratedAt.Time
		}
		if descriptionCuratedByUserID.Valid {
			item.DescriptionCuratedByUserID = descriptionCuratedByUserID.String
		}
		if descriptionCuratedAt.Valid {
			item.DescriptionCuratedAt = &descriptionCuratedAt.Time
		}

		items = append(items, item)
	}

	return &models.GearCatalogSearchResponse{
		Items:      items,
		TotalCount: totalCount,
		Query:      params.Query,
	}, nil
}

// AdminUpdate updates a gear catalog item with admin-provided values
func (s *GearCatalogStore) AdminUpdate(ctx context.Context, id string, adminUserID string, params models.AdminUpdateGearCatalogParams) (*models.GearCatalogItem, error) {
	// If brand/model/variant is changing, we need to recompute canonical_key
	needsCanonicalKeyUpdate := params.Brand != nil || params.Model != nil || params.Variant != nil

	var currentItem *models.GearCatalogItem
	var err error
	if needsCanonicalKeyUpdate {
		currentItem, err = s.Get(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get current item: %w", err)
		}
		if currentItem == nil {
			return nil, fmt.Errorf("catalog item not found: %s", id)
		}
	}

	var sets []string
	var args []interface{}
	argIdx := 1

	// Track effective values for canonical key computation
	var effectiveBrand, effectiveModel, effectiveVariant string
	var effectiveGearType models.GearType

	if needsCanonicalKeyUpdate {
		effectiveBrand = currentItem.Brand
		effectiveModel = currentItem.Model
		effectiveVariant = currentItem.Variant
		effectiveGearType = currentItem.GearType
	}

	if params.Brand != nil {
		sets = append(sets, fmt.Sprintf("brand = $%d", argIdx))
		args = append(args, *params.Brand)
		argIdx++
		effectiveBrand = *params.Brand
	}
	if params.Model != nil {
		sets = append(sets, fmt.Sprintf("model = $%d", argIdx))
		args = append(args, *params.Model)
		argIdx++
		effectiveModel = *params.Model
	}
	if params.Variant != nil {
		sets = append(sets, fmt.Sprintf("variant = $%d", argIdx))
		args = append(args, *params.Variant)
		argIdx++
		effectiveVariant = *params.Variant
	}

	if params.Specs != nil {
		sets = append(sets, fmt.Sprintf("specs = $%d", argIdx))
		args = append(args, params.Specs)
		argIdx++
	}

	// Recompute canonical_key if brand/model/variant changed
	if needsCanonicalKeyUpdate {
		newCanonicalKey := models.BuildCanonicalKey(effectiveGearType, effectiveBrand, effectiveModel, effectiveVariant)
		// Check if new canonical_key would conflict with another item
		if newCanonicalKey != currentItem.CanonicalKey {
			existing, err := s.GetByCanonicalKey(ctx, newCanonicalKey)
			if err != nil {
				return nil, fmt.Errorf("failed to check for canonical key conflict: %w", err)
			}
			if existing != nil {
				return nil, fmt.Errorf("cannot update: another item already exists with brand=%q model=%q variant=%q", effectiveBrand, effectiveModel, effectiveVariant)
			}
			sets = append(sets, fmt.Sprintf("canonical_key = $%d", argIdx))
			args = append(args, newCanonicalKey)
			argIdx++
		}
	}

	if params.Description != nil {
		sets = append(sets, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *params.Description)
		argIdx++
		// Update description curation fields
		if *params.Description != "" {
			sets = append(sets, fmt.Sprintf("description_status = $%d", argIdx))
			args = append(args, models.ImageStatusApproved)
			argIdx++
			sets = append(sets, fmt.Sprintf("description_curated_by_user_id = $%d", argIdx))
			args = append(args, adminUserID)
			argIdx++
			sets = append(sets, "description_curated_at = NOW()")
		} else {
			// Clearing description - reset curation status
			sets = append(sets, fmt.Sprintf("description_status = $%d", argIdx))
			args = append(args, models.ImageStatusMissing)
			argIdx++
			sets = append(sets, "description_curated_by_user_id = NULL")
			sets = append(sets, "description_curated_at = NULL")
		}
	}
	if params.ClearMSRP {
		sets = append(sets, "msrp = NULL")
	} else if params.MSRP != nil {
		sets = append(sets, fmt.Sprintf("msrp = $%d", argIdx))
		args = append(args, *params.MSRP)
		argIdx++
	}
	if params.BestFor != nil {
		sets = append(sets, fmt.Sprintf("best_for = $%d", argIdx))
		args = append(args, pq.Array(params.BestFor))
		argIdx++
	}
	if params.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *params.Status)
		argIdx++

		// Publishing a catalog item should also finalize any scanned image curation.
		// This keeps status and image curation state in sync for admin moderation UX.
		if *params.Status == models.CatalogStatusPublished && params.ImageStatus == nil {
			// If this row currently has a scanned image, promote it to approved and record curator metadata.
			sets = append(sets, fmt.Sprintf(`
				image_status = CASE
					WHEN COALESCE(image_status, 'missing') = $%d
					     AND (
					       image_asset_id IS NOT NULL
					       OR image_data IS NOT NULL
					     )
					THEN $%d
					ELSE COALESCE(image_status, 'missing')
				END
			`, argIdx, argIdx+1))
			args = append(args, models.ImageStatusScanned, models.ImageStatusApproved)
			argIdx += 2

			sets = append(sets, fmt.Sprintf(`
				image_curated_by_user_id = CASE
					WHEN COALESCE(image_status, 'missing') = $%d
					     AND (
					       image_asset_id IS NOT NULL
					       OR image_data IS NOT NULL
					     )
					THEN $%d
					ELSE image_curated_by_user_id
				END
			`, argIdx, argIdx+1))
			args = append(args, models.ImageStatusScanned, adminUserID)
			argIdx += 2

			sets = append(sets, fmt.Sprintf(`
				image_curated_at = CASE
					WHEN COALESCE(image_status, 'missing') = $%d
					     AND (
					       image_asset_id IS NOT NULL
					       OR image_data IS NOT NULL
					     )
					THEN NOW()
					ELSE image_curated_at
				END
			`, argIdx))
			args = append(args, models.ImageStatusScanned)
			argIdx++
		}
	}
	if params.ImageStatus != nil {
		sets = append(sets, fmt.Sprintf("image_status = $%d", argIdx))
		args = append(args, *params.ImageStatus)
		argIdx++

		switch *params.ImageStatus {
		case models.ImageStatusApproved:
			sets = append(sets, fmt.Sprintf("image_curated_by_user_id = $%d", argIdx))
			args = append(args, adminUserID)
			argIdx++
			sets = append(sets, "image_curated_at = NOW()")
		case models.ImageStatusScanned, models.ImageStatusMissing:
			sets = append(sets, "image_curated_by_user_id = NULL")
			sets = append(sets, "image_curated_at = NULL")
		}
	}
	if len(sets) == 0 {
		return s.Get(ctx, id)
	}

	sets = append(sets, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE gear_catalog SET %s
		WHERE id = $%d
	`, strings.Join(sets, ", "), argIdx)

	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to admin update catalog item: %w", err)
	}

	return s.Get(ctx, id)
}

// SetImage stores an approved image asset reference for a gear catalog item (admin only).
// Returns any previous image asset ID for cleanup.
func (s *GearCatalogStore) SetImage(ctx context.Context, id string, adminUserID string, imageType string, imageAssetID string) (string, error) {
	var previousAssetID sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT image_asset_id FROM gear_catalog WHERE id = $1`, id).Scan(&previousAssetID); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("gear catalog item not found")
		}
		return "", fmt.Errorf("failed to fetch existing gear image reference: %w", err)
	}

	query := `
		UPDATE gear_catalog 
		SET image_asset_id = $1,
		    image_data = NULL,
		    image_type = $2, 
		    image_status = $3,
		    image_curated_by_user_id = $4,
		    image_curated_at = NOW(),
		    updated_at = NOW()
		WHERE id = $5
	`
	result, err := s.db.ExecContext(ctx, query, imageAssetID, imageType, models.ImageStatusApproved, adminUserID, id)
	if err != nil {
		return "", fmt.Errorf("failed to set gear image: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return "", fmt.Errorf("gear catalog item not found")
	}

	if previousAssetID.Valid {
		return previousAssetID.String, nil
	}
	return "", nil
}

// SetUserSubmittedImage stores a moderated user-submitted image for a catalog item.
// This marks image_status as "scanned" so it remains in the admin moderation queue.
// Returns previous image asset ID for cleanup.
func (s *GearCatalogStore) SetUserSubmittedImage(ctx context.Context, id string, imageType string, imageAssetID string) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction for user-submitted gear image: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var (
		previousAssetID sql.NullString
		imageStatus     models.ImageStatus
	)

	// Lock the row to avoid TOCTOU races with concurrent admin curation.
	if err := tx.QueryRowContext(ctx, `
		SELECT image_asset_id, COALESCE(image_status, 'missing')
		FROM gear_catalog
		WHERE id = $1
		FOR UPDATE
	`, id).Scan(&previousAssetID, &imageStatus); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("%w: %s", ErrCatalogItemNotFound, id)
		}
		return "", fmt.Errorf("failed to fetch existing gear image reference: %w", err)
	}

	if imageStatus == models.ImageStatusApproved {
		return "", ErrCatalogImageAlreadyCurated
	}

	query := `
		UPDATE gear_catalog
		SET image_asset_id = $1,
		    image_data = NULL,
		    image_type = $2,
		    image_status = $3,
		    image_curated_by_user_id = NULL,
		    image_curated_at = NULL,
		    updated_at = NOW()
		WHERE id = $4
	`
	result, err := tx.ExecContext(ctx, query, imageAssetID, imageType, models.ImageStatusScanned, id)
	if err != nil {
		return "", fmt.Errorf("failed to set user-submitted gear image: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return "", fmt.Errorf("%w: %s", ErrCatalogItemNotFound, id)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit user-submitted gear image update: %w", err)
	}

	if previousAssetID.Valid {
		return previousAssetID.String, nil
	}
	return "", nil
}

// ApproveImage marks an existing catalog image as approved by an admin.
func (s *GearCatalogStore) ApproveImage(ctx context.Context, id string, adminUserID string) error {
	var (
		imageStatus models.ImageStatus
		hasImage    bool
	)

	if err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(image_status, 'missing'),
		       (
		         (image_asset_id IS NOT NULL)
		         OR (image_data IS NOT NULL)
		       ) AS has_image
		FROM gear_catalog
		WHERE id = $1
	`, id).Scan(&imageStatus, &hasImage); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("%w: %s", ErrCatalogItemNotFound, id)
		}
		return fmt.Errorf("failed to fetch gear image status: %w", err)
	}

	if !hasImage {
		return ErrCatalogImageMissing
	}
	if imageStatus == models.ImageStatusApproved {
		return nil
	}

	query := `
		UPDATE gear_catalog
		SET image_status = $1,
		    image_curated_by_user_id = $2,
		    image_curated_at = NOW(),
		    updated_at = NOW()
		WHERE id = $3
	`
	result, err := s.db.ExecContext(ctx, query, models.ImageStatusApproved, adminUserID, id)
	if err != nil {
		return fmt.Errorf("failed to approve gear image: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("%w: %s", ErrCatalogItemNotFound, id)
	}

	return nil
}

// GetImage retrieves the binary image data for a gear catalog item
func (s *GearCatalogStore) GetImage(ctx context.Context, id string) ([]byte, string, error) {
	query := `
		SELECT COALESCE(ia.image_bytes, gc.image_data), gc.image_type
		FROM gear_catalog gc
		LEFT JOIN image_assets ia ON ia.id = gc.image_asset_id AND ia.status = 'APPROVED'
		WHERE gc.id = $1 AND ((gc.image_asset_id IS NOT NULL AND ia.id IS NOT NULL) OR gc.image_data IS NOT NULL)
	`
	var imageData []byte
	var imageType sql.NullString
	err := s.db.QueryRowContext(ctx, query, id).Scan(&imageData, &imageType)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("failed to get gear image: %w", err)
	}

	return imageData, imageType.String, nil
}

// HasImage checks if a gear catalog item has an uploaded image
func (s *GearCatalogStore) HasImage(ctx context.Context, id string) (bool, error) {
	query := `
		SELECT (
			image_data IS NOT NULL
			OR EXISTS (
				SELECT 1
				FROM image_assets ia
				WHERE ia.id = gear_catalog.image_asset_id
				  AND ia.status = 'APPROVED'
			)
		)
		FROM gear_catalog
		WHERE id = $1
	`
	var hasImage bool
	err := s.db.QueryRowContext(ctx, query, id).Scan(&hasImage)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check for gear image: %w", err)
	}
	return hasImage, nil
}

// DeleteImage removes the image from a gear catalog item (admin only).
// Returns previous image asset ID for cleanup.
func (s *GearCatalogStore) DeleteImage(ctx context.Context, id string) (string, error) {
	var previousAssetID sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT image_asset_id FROM gear_catalog WHERE id = $1`, id).Scan(&previousAssetID); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("catalog item not found: %s", id)
		}
		return "", fmt.Errorf("failed to fetch existing gear image reference: %w", err)
	}

	query := `
		UPDATE gear_catalog 
		SET image_asset_id = NULL,
		    image_data = NULL, 
		    image_type = NULL, 
		    image_status = $1,
		    image_curated_by_user_id = NULL,
		    image_curated_at = NULL,
		    updated_at = NOW()
		WHERE id = $2
	`
	result, err := s.db.ExecContext(ctx, query, models.ImageStatusMissing, id)
	if err != nil {
		return "", fmt.Errorf("failed to delete gear image: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return "", fmt.Errorf("catalog item not found: %s", id)
	}

	if previousAssetID.Valid {
		return previousAssetID.String, nil
	}
	return "", nil
}

// AdminDelete permanently deletes a gear catalog item (admin only).
// Related inventory_items.catalog_id references are nulled via FK ON DELETE SET NULL.
func (s *GearCatalogStore) AdminDelete(ctx context.Context, id string) error {
	query := `DELETE FROM gear_catalog WHERE id = $1`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete gear catalog item: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("%w: %s", ErrCatalogItemNotFound, id)
	}

	return nil
}

func (s *GearCatalogStore) AdminBulkDelete(ctx context.Context, ids []string) ([]string, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query := `DELETE FROM gear_catalog WHERE id = ANY($1::uuid[]) RETURNING id`
	rows, err := s.db.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to bulk delete gear catalog items: %w", err)
	}
	defer rows.Close()

	deletedIDs := make([]string, 0, len(ids))
	for rows.Next() {
		var deletedID string
		if err := rows.Scan(&deletedID); err != nil {
			return nil, fmt.Errorf("failed to scan deleted gear catalog id: %w", err)
		}
		deletedIDs = append(deletedIDs, deletedID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to bulk delete gear catalog items: %w", err)
	}

	return deletedIDs, nil
}

package database

import (
	"context"
	"database/sql"
	"encoding/json"
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
			created_by_user_id, status, canonical_key, image_url, description
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at, updated_at
	`

	item := &models.GearCatalogItem{
		GearType:        params.GearType,
		Brand:           strings.TrimSpace(params.Brand),
		Model:           strings.TrimSpace(params.Model),
		Variant:         strings.TrimSpace(params.Variant),
		Specs:           specs,
		BestFor:         params.BestFor,
		MSRP:            params.MSRP,
		Source:          models.CatalogSourceUserSubmitted,
		CreatedByUserID: userID,
		Status:          models.CatalogStatusActive,
		CanonicalKey:    canonicalKey,
		ImageURL:        params.ImageURL,
		Description:     params.Description,
	}

	var createdByUserIDPtr *string
	if userID != "" {
		createdByUserIDPtr = &userID
	}

	err = s.db.QueryRowContext(ctx, query,
		item.GearType, item.Brand, item.Model, nullString(item.Variant),
		item.Specs, pq.Array(item.BestFor), item.MSRP, item.Source, createdByUserIDPtr, item.Status,
		item.CanonicalKey, nullString(item.ImageURL), nullString(item.Description),
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
			   created_by_user_id, status, canonical_key, image_url, description,
			   created_at, updated_at,
			   (SELECT COUNT(*) FROM inventory_items WHERE catalog_id = gear_catalog.id) as usage_count
		FROM gear_catalog
		WHERE id = $1
	`

	item := &models.GearCatalogItem{}
	var variant, imageURL, description, createdByUserID sql.NullString
	var msrp sql.NullFloat64

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&item.ID, &item.GearType, &item.Brand, &item.Model, &variant,
		&item.Specs, pq.Array(&item.BestFor), &msrp, &item.Source, &createdByUserID, &item.Status,
		&item.CanonicalKey, &imageURL, &description,
		&item.CreatedAt, &item.UpdatedAt, &item.UsageCount,
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

	return item, nil
}

// GetByCanonicalKey retrieves a catalog item by its canonical key
func (s *GearCatalogStore) GetByCanonicalKey(ctx context.Context, canonicalKey string) (*models.GearCatalogItem, error) {
	query := `
		SELECT id, gear_type, brand, model, variant, specs, best_for, msrp, source,
			   created_by_user_id, status, canonical_key, image_url, description,
			   created_at, updated_at,
			   (SELECT COUNT(*) FROM inventory_items WHERE catalog_id = gear_catalog.id) as usage_count
		FROM gear_catalog
		WHERE canonical_key = $1
	`

	item := &models.GearCatalogItem{}
	var variant, imageURL, description, createdByUserID sql.NullString
	var msrp sql.NullFloat64

	err := s.db.QueryRowContext(ctx, query, canonicalKey).Scan(
		&item.ID, &item.GearType, &item.Brand, &item.Model, &variant,
		&item.Specs, pq.Array(&item.BestFor), &msrp, &item.Source, &createdByUserID, &item.Status,
		&item.CanonicalKey, &imageURL, &description,
		&item.CreatedAt, &item.UpdatedAt, &item.UsageCount,
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
	whereClauses := []string{"status = 'active'"}
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
		// Override the default status filter
		whereClauses[0] = fmt.Sprintf("status = $%d", argIdx)
		args = append(args, params.Status)
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
			   created_by_user_id, status, canonical_key, image_url, description,
			   created_at, updated_at,
			   (SELECT COUNT(*) FROM inventory_items WHERE catalog_id = gear_catalog.id) as usage_count
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
		var msrp sql.NullFloat64

		if err := rows.Scan(
			&item.ID, &item.GearType, &item.Brand, &item.Model, &variant,
			&item.Specs, pq.Array(&item.BestFor), &msrp, &item.Source, &createdByUserID, &item.Status,
			&item.CanonicalKey, &imageURL, &description,
			&item.CreatedAt, &item.UpdatedAt, &item.UsageCount,
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
			   created_by_user_id, status, canonical_key, image_url, description,
			   created_at, updated_at,
			   (SELECT COUNT(*) FROM inventory_items WHERE catalog_id = gear_catalog.id) as usage_count,
			   COALESCE(similarity(LOWER(brand || ' ' || model), LOWER($2 || ' ' || $3)), 0) as sim_score
		FROM gear_catalog
		WHERE gear_type = $1
		  AND status = 'active'
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
			   created_by_user_id, status, canonical_key, image_url, description,
			   created_at, updated_at,
			   (SELECT COUNT(*) FROM inventory_items WHERE catalog_id = gear_catalog.id) as usage_count
		FROM gear_catalog
		WHERE gear_type = $1
		  AND status = 'active'
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
			   created_by_user_id, status, canonical_key, image_url, description,
			   created_at, updated_at,
			   (SELECT COUNT(*) FROM inventory_items WHERE catalog_id = gear_catalog.id) as usage_count
		FROM gear_catalog
		WHERE status = 'active'
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
		var msrp sql.NullFloat64

		if err := rows.Scan(
			&item.ID, &item.GearType, &item.Brand, &item.Model, &variant,
			&item.Specs, pq.Array(&item.BestFor), &msrp, &item.Source, &createdByUserID, &item.Status,
			&item.CanonicalKey, &imageURL, &description,
			&item.CreatedAt, &item.UpdatedAt, &item.UsageCount,
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

		items = append(items, item)
	}

	return items, nil
}

// MigrateInventoryItem creates a catalog entry from an existing inventory item
// and links the inventory item to it. Uses a transaction to ensure consistency.
func (s *GearCatalogStore) MigrateInventoryItem(ctx context.Context, inventoryItemID, userID, name, manufacturer string, category models.EquipmentCategory, specs json.RawMessage, imageURL string) (*models.GearCatalogItem, error) {
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
		// Create new catalog entry
		catalogID = uuid.New().String()
		insertQuery := `
			INSERT INTO gear_catalog (id, gear_type, brand, model, variant, specs, source, created_by_user_id, status, canonical_key, image_url, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, 'user', $7, 'active', $8, $9, NOW(), NOW())
		`
		_, err = tx.ExecContext(ctx, insertQuery, catalogID, gearType, brand, model, variant, specs, userID, canonicalKey, imageURL)
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

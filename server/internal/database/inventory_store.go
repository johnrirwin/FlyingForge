package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/johnrirwin/flyingforge/internal/models"
)

// InventoryStore handles inventory database operations
type InventoryStore struct {
	db *DB
}

// NewInventoryStore creates a new inventory store
func NewInventoryStore(db *DB) *InventoryStore {
	return &InventoryStore{db: db}
}

// Add creates a new inventory item
func (s *InventoryStore) Add(ctx context.Context, userID string, params models.AddInventoryParams) (*models.InventoryItem, error) {
	specs := params.Specs
	if specs == nil {
		specs = json.RawMessage(`{}`)
	}

	quantity := params.Quantity
	if quantity <= 0 {
		quantity = 1
	}

	query := `
		INSERT INTO inventory_items (
			user_id, name, category, manufacturer, quantity, notes,
			build_id, purchase_price, purchase_seller,
			product_url, specs, source_equipment_id, catalog_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at, updated_at
	`

	item := &models.InventoryItem{
		UserID:            userID,
		Name:              params.Name,
		Category:          params.Category,
		Manufacturer:      params.Manufacturer,
		Quantity:          quantity,
		Notes:             params.Notes,
		BuildID:           params.BuildID,
		PurchasePrice:     params.PurchasePrice,
		PurchaseSeller:    params.PurchaseSeller,
		ProductURL:        params.ProductURL,
		Specs:             specs,
		SourceEquipmentID: params.SourceEquipmentID,
		CatalogID:         params.CatalogID,
	}

	err := s.db.QueryRowContext(ctx, query,
		nullString(userID), item.Name, item.Category, item.Manufacturer, item.Quantity, item.Notes,
		nullString(item.BuildID), item.PurchasePrice, nullString(item.PurchaseSeller),
		nullString(item.ProductURL), item.Specs, nullString(item.SourceEquipmentID),
		nullString(item.CatalogID),
	).Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to insert inventory item: %w", err)
	}

	return item, nil
}

// AddOrIncrement atomically adds a new inventory item or increments quantity if one with the same
// catalog_id already exists for the user (uses UPSERT). Only works when catalog_id is provided.
func (s *InventoryStore) AddOrIncrement(ctx context.Context, userID string, params models.AddInventoryParams) (*models.InventoryItem, error) {
	if params.CatalogID == "" {
		return nil, fmt.Errorf("AddOrIncrement requires a catalog_id")
	}

	specs := params.Specs
	if specs == nil {
		specs = json.RawMessage(`{}`)
	}

	quantity := params.Quantity
	if quantity <= 0 {
		quantity = 1
	}

	// UPSERT: insert if not exists, otherwise increment quantity
	// The ON CONFLICT predicate must match the partial unique index exactly
	query := `
		INSERT INTO inventory_items (
			user_id, name, category, manufacturer, quantity, notes,
			build_id, purchase_price, purchase_seller,
			product_url, specs, source_equipment_id, catalog_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (user_id, catalog_id) WHERE user_id IS NOT NULL AND catalog_id IS NOT NULL
		DO UPDATE SET quantity = inventory_items.quantity + EXCLUDED.quantity, updated_at = NOW()
		RETURNING id, user_id, name, category, manufacturer, quantity, notes,
			build_id, purchase_price, purchase_seller,
			product_url, specs, source_equipment_id, catalog_id, created_at, updated_at
	`

	item := &models.InventoryItem{}
	var itemUserID sql.NullString
	var buildID, purchaseSeller, productURL, sourceEquipmentID, catalogID sql.NullString
	var purchasePriceNull sql.NullFloat64

	err := s.db.QueryRowContext(ctx, query,
		nullString(userID), params.Name, params.Category, params.Manufacturer, quantity, params.Notes,
		nullString(params.BuildID), params.PurchasePrice, nullString(params.PurchaseSeller),
		nullString(params.ProductURL), specs, nullString(params.SourceEquipmentID),
		nullString(params.CatalogID),
	).Scan(
		&item.ID, &itemUserID, &item.Name, &item.Category, &item.Manufacturer,
		&item.Quantity, &item.Notes,
		&buildID, &purchasePriceNull, &purchaseSeller,
		&productURL, &item.Specs, &sourceEquipmentID, &catalogID,
		&item.CreatedAt, &item.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert inventory item: %w", err)
	}

	if itemUserID.Valid {
		item.UserID = itemUserID.String
	}
	item.BuildID = buildID.String
	item.PurchaseSeller = purchaseSeller.String
	item.ProductURL = productURL.String
	item.SourceEquipmentID = sourceEquipmentID.String
	item.CatalogID = catalogID.String

	if purchasePriceNull.Valid {
		item.PurchasePrice = &purchasePriceNull.Float64
	}

	return item, nil
}

// Get retrieves an inventory item by ID (optionally scoped to user)
func (s *InventoryStore) Get(ctx context.Context, id string, userID string) (*models.InventoryItem, error) {
	query := `
		SELECT i.id, i.user_id, i.name, i.category, i.manufacturer, i.quantity, i.notes,
			   i.build_id, i.purchase_price, i.purchase_seller,
			   i.product_url, 
			   COALESCE(
			        NULLIF(TRIM(i.image_url), ''),
			        NULLIF(TRIM(gc.image_url), ''),
			        CASE
			             WHEN COALESCE(gc.image_status, 'missing') IN ('approved', 'scanned')
			                  AND (gc.image_asset_id IS NOT NULL OR gc.image_data IS NOT NULL)
			                  THEN '/api/gear-catalog/' || gc.id || '/image?v=' || (EXTRACT(EPOCH FROM COALESCE(gc.image_curated_at, gc.updated_at))*1000)::bigint
			             ELSE NULL
			        END
			   ) as image_url,
			   i.specs, i.source_equipment_id, i.catalog_id, i.created_at, i.updated_at
		FROM inventory_items i
		LEFT JOIN gear_catalog gc ON i.catalog_id = gc.id
		WHERE i.id = $1
	`
	args := []interface{}{id}

	// If userID is provided, scope the query
	if userID != "" {
		query = `
			SELECT i.id, i.user_id, i.name, i.category, i.manufacturer, i.quantity, i.notes,
				   i.build_id, i.purchase_price, i.purchase_seller,
				   i.product_url, 
				   COALESCE(
				        NULLIF(TRIM(i.image_url), ''),
				        NULLIF(TRIM(gc.image_url), ''),
				        CASE
				             WHEN COALESCE(gc.image_status, 'missing') IN ('approved', 'scanned')
				                  AND (gc.image_asset_id IS NOT NULL OR gc.image_data IS NOT NULL)
				                  THEN '/api/gear-catalog/' || gc.id || '/image?v=' || (EXTRACT(EPOCH FROM COALESCE(gc.image_curated_at, gc.updated_at))*1000)::bigint
				             ELSE NULL
				        END
				   ) as image_url,
				   i.specs, i.source_equipment_id, i.catalog_id, i.created_at, i.updated_at
			FROM inventory_items i
			LEFT JOIN gear_catalog gc ON i.catalog_id = gc.id
			WHERE i.id = $1 AND i.user_id = $2
		`
		args = append(args, userID)
	}

	item := &models.InventoryItem{}
	var itemUserID sql.NullString
	var buildID, purchaseSeller, productURL, imageURL, sourceEquipmentID, catalogID sql.NullString
	var purchasePrice sql.NullFloat64

	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&item.ID, &itemUserID, &item.Name, &item.Category, &item.Manufacturer,
		&item.Quantity, &item.Notes,
		&buildID, &purchasePrice, &purchaseSeller,
		&productURL, &imageURL, &item.Specs, &sourceEquipmentID, &catalogID,
		&item.CreatedAt, &item.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory item: %w", err)
	}

	if itemUserID.Valid {
		item.UserID = itemUserID.String
	}
	item.BuildID = buildID.String
	item.PurchaseSeller = purchaseSeller.String
	item.ProductURL = productURL.String
	item.ImageURL = imageURL.String
	item.SourceEquipmentID = sourceEquipmentID.String
	item.CatalogID = catalogID.String

	if purchasePrice.Valid {
		item.PurchasePrice = &purchasePrice.Float64
	}

	return item, nil
}

// List retrieves inventory items with optional filtering (scoped to user if userID provided)
func (s *InventoryStore) List(ctx context.Context, userID string, params models.InventoryFilterParams) (*models.InventoryResponse, error) {
	// Build WHERE clause
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Scope to user if userID is provided
	if userID != "" {
		conditions = append(conditions, fmt.Sprintf("i.user_id = $%d", argIndex))
		args = append(args, userID)
		argIndex++
	}

	if params.Category != "" {
		conditions = append(conditions, fmt.Sprintf("i.category = $%d", argIndex))
		args = append(args, params.Category)
		argIndex++
	}

	if params.BuildID != "" {
		conditions = append(conditions, fmt.Sprintf("i.build_id = $%d", argIndex))
		args = append(args, params.BuildID)
		argIndex++
	}

	if params.Query != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(i.name ILIKE $%d OR i.manufacturer ILIKE $%d)",
			argIndex, argIndex,
		))
		args = append(args, "%"+params.Query+"%")
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var totalCount int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM inventory_items i %s", whereClause)
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count inventory: %w", err)
	}

	// Get items
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := params.Offset

	query := fmt.Sprintf(`
		SELECT i.id, i.user_id, i.name, i.category, i.manufacturer, i.quantity, i.notes,
			   i.build_id, i.purchase_price, i.purchase_seller,
			   i.product_url, 
			   COALESCE(
			        NULLIF(TRIM(i.image_url), ''),
			        NULLIF(TRIM(gc.image_url), ''),
			        CASE
			             WHEN COALESCE(gc.image_status, 'missing') IN ('approved', 'scanned')
			                  AND (gc.image_asset_id IS NOT NULL OR gc.image_data IS NOT NULL)
			                  THEN '/api/gear-catalog/' || gc.id || '/image?v=' || (EXTRACT(EPOCH FROM COALESCE(gc.image_curated_at, gc.updated_at))*1000)::bigint
			             ELSE NULL
			        END
			   ) as image_url,
			   i.specs, i.source_equipment_id, i.catalog_id, i.created_at, i.updated_at
		FROM inventory_items i
		LEFT JOIN gear_catalog gc ON i.catalog_id = gc.id
		%s
		ORDER BY i.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list inventory: %w", err)
	}
	defer rows.Close()

	items := make([]models.InventoryItem, 0)
	categories := make(map[models.EquipmentCategory]int)

	for rows.Next() {
		var item models.InventoryItem
		var buildID, purchaseSeller, productURL, imageURL, sourceEquipmentID, catalogID sql.NullString
		var purchasePrice sql.NullFloat64

		if err := rows.Scan(
			&item.ID, &item.UserID, &item.Name, &item.Category, &item.Manufacturer,
			&item.Quantity, &item.Notes,
			&buildID, &purchasePrice, &purchaseSeller,
			&productURL, &imageURL, &item.Specs, &sourceEquipmentID, &catalogID,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan inventory item: %w", err)
		}

		item.BuildID = buildID.String
		item.PurchaseSeller = purchaseSeller.String
		item.ProductURL = productURL.String
		item.ImageURL = imageURL.String
		item.SourceEquipmentID = sourceEquipmentID.String
		item.CatalogID = catalogID.String

		if purchasePrice.Valid {
			item.PurchasePrice = &purchasePrice.Float64
		}

		items = append(items, item)
		categories[item.Category]++
	}

	return &models.InventoryResponse{
		Items:      items,
		TotalCount: totalCount,
		Categories: categories,
	}, nil
}

// Update updates an inventory item (scoped to user if userID provided)
func (s *InventoryStore) Update(ctx context.Context, userID string, params models.UpdateInventoryParams) (*models.InventoryItem, error) {
	// Build SET clause
	var sets []string
	var args []interface{}
	argIndex := 1

	if params.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *params.Name)
		argIndex++
	}

	if params.Category != nil {
		sets = append(sets, fmt.Sprintf("category = $%d", argIndex))
		args = append(args, *params.Category)
		argIndex++
	}

	if params.Manufacturer != nil {
		sets = append(sets, fmt.Sprintf("manufacturer = $%d", argIndex))
		args = append(args, *params.Manufacturer)
		argIndex++
	}

	if params.Quantity != nil {
		sets = append(sets, fmt.Sprintf("quantity = $%d", argIndex))
		args = append(args, *params.Quantity)
		argIndex++
	}

	if params.Notes != nil {
		sets = append(sets, fmt.Sprintf("notes = $%d", argIndex))
		args = append(args, *params.Notes)
		argIndex++
	}

	if params.BuildID != nil {
		sets = append(sets, fmt.Sprintf("build_id = $%d", argIndex))
		args = append(args, nullString(*params.BuildID))
		argIndex++
	}

	if params.PurchasePrice != nil {
		sets = append(sets, fmt.Sprintf("purchase_price = $%d", argIndex))
		args = append(args, *params.PurchasePrice)
		argIndex++
	}

	if params.PurchaseSeller != nil {
		sets = append(sets, fmt.Sprintf("purchase_seller = $%d", argIndex))
		args = append(args, nullString(*params.PurchaseSeller))
		argIndex++
	}

	if params.ProductURL != nil {
		sets = append(sets, fmt.Sprintf("product_url = $%d", argIndex))
		args = append(args, nullString(*params.ProductURL))
		argIndex++
	}

	if params.Specs != nil {
		sets = append(sets, fmt.Sprintf("specs = $%d", argIndex))
		args = append(args, params.Specs)
		argIndex++
	}

	if len(sets) == 0 {
		return s.Get(ctx, params.ID, userID)
	}

	sets = append(sets, "updated_at = NOW()")

	whereClause := fmt.Sprintf("id = $%d", argIndex)
	args = append(args, params.ID)
	argIndex++

	// Scope to user if userID provided
	if userID != "" {
		whereClause += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, userID)
	}

	query := fmt.Sprintf(`
		UPDATE inventory_items
		SET %s
		WHERE %s
	`, strings.Join(sets, ", "), whereClause)

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update inventory item: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, fmt.Errorf("inventory item not found: %s", params.ID)
	}

	return s.Get(ctx, params.ID, userID)
}

// Delete removes an inventory item (scoped to user if userID provided)
func (s *InventoryStore) Delete(ctx context.Context, id string, userID string) error {
	query := "DELETE FROM inventory_items WHERE id = $1"
	args := []interface{}{id}

	if userID != "" {
		query = "DELETE FROM inventory_items WHERE id = $1 AND user_id = $2"
		args = append(args, userID)
	}

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete inventory item: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("inventory item not found: %s", id)
	}

	return nil
}

// GetSummary returns a summary of the inventory (scoped to user if userID provided)
func (s *InventoryStore) GetSummary(ctx context.Context, userID string) (*models.InventorySummary, error) {
	summary := &models.InventorySummary{
		ByCategory: make(map[models.EquipmentCategory]int),
	}

	query := `SELECT category, quantity, purchase_price, specs FROM inventory_items`
	args := []interface{}{}
	if userID != "" {
		query += " WHERE user_id = $1"
		args = append(args, userID)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory summary: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var category models.EquipmentCategory
		var quantity int
		var purchasePrice sql.NullFloat64
		var specs json.RawMessage

		if err := rows.Scan(&category, &quantity, &purchasePrice, &specs); err != nil {
			return nil, fmt.Errorf("failed to scan inventory summary row: %w", err)
		}

		summary.TotalItems++
		summary.ByCategory[category]++

		var pricePtr *float64
		if purchasePrice.Valid {
			price := purchasePrice.Float64
			pricePtr = &price
		}

		summary.TotalValue += models.CalculateInventoryItemTotalValue(quantity, pricePtr, specs)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate inventory summary rows: %w", err)
	}

	return summary, nil
}

// GetByCatalogID finds an inventory item by catalog ID for a user
func (s *InventoryStore) GetByCatalogID(ctx context.Context, userID, catalogID string) (*models.InventoryItem, error) {
	if catalogID == "" {
		return nil, nil
	}

	query := `
		SELECT i.id, i.user_id, i.name, i.category, i.manufacturer, i.quantity, i.notes,
			   i.build_id, i.purchase_price, i.purchase_seller,
			   i.product_url, 
			   COALESCE(
			        NULLIF(TRIM(i.image_url), ''),
			        NULLIF(TRIM(gc.image_url), ''),
			        CASE
			             WHEN COALESCE(gc.image_status, 'missing') IN ('approved', 'scanned')
			                  AND (gc.image_asset_id IS NOT NULL OR gc.image_data IS NOT NULL)
			                  THEN '/api/gear-catalog/' || gc.id || '/image?v=' || (EXTRACT(EPOCH FROM COALESCE(gc.image_curated_at, gc.updated_at))*1000)::bigint
			             ELSE NULL
			        END
			   ) as image_url,
			   i.specs, i.source_equipment_id, i.catalog_id, i.created_at, i.updated_at
		FROM inventory_items i
		LEFT JOIN gear_catalog gc ON i.catalog_id = gc.id
		WHERE i.user_id = $1 AND i.catalog_id = $2
		ORDER BY i.created_at DESC
		LIMIT 1
	`

	item := &models.InventoryItem{}
	var itemUserID sql.NullString
	var buildID, purchaseSeller, productURL, imageURL, sourceEquipmentID, itemCatalogID sql.NullString
	var purchasePrice sql.NullFloat64

	err := s.db.QueryRowContext(ctx, query, userID, catalogID).Scan(
		&item.ID, &itemUserID, &item.Name, &item.Category, &item.Manufacturer,
		&item.Quantity, &item.Notes,
		&buildID, &purchasePrice, &purchaseSeller,
		&productURL, &imageURL, &item.Specs, &sourceEquipmentID, &itemCatalogID,
		&item.CreatedAt, &item.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory item by catalog ID: %w", err)
	}

	if itemUserID.Valid {
		item.UserID = itemUserID.String
	}
	item.BuildID = buildID.String
	item.PurchaseSeller = purchaseSeller.String
	item.ProductURL = productURL.String
	item.ImageURL = imageURL.String
	item.SourceEquipmentID = sourceEquipmentID.String
	item.CatalogID = itemCatalogID.String

	if purchasePrice.Valid {
		item.PurchasePrice = &purchasePrice.Float64
	}

	return item, nil
}

// IncrementQuantity increases the quantity of an existing inventory item
func (s *InventoryStore) IncrementQuantity(ctx context.Context, id string, userID string, amount int) (*models.InventoryItem, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("increment amount must be positive, got %d", amount)
	}

	query := `
		UPDATE inventory_items 
		SET quantity = quantity + $1, updated_at = NOW()
		WHERE id = $2 AND user_id = $3
		RETURNING id
	`

	var returnedID string
	err := s.db.QueryRowContext(ctx, query, amount, id, userID).Scan(&returnedID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("inventory item not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to increment inventory quantity: %w", err)
	}

	return s.Get(ctx, id, userID)
}

// Helper function for nullable strings
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

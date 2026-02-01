package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/johnrirwin/rotorlife/internal/models"
)

// BatteryStore handles battery database operations
type BatteryStore struct {
	db *DB
}

// NewBatteryStore creates a new battery store
func NewBatteryStore(db *DB) *BatteryStore {
	return &BatteryStore{db: db}
}

// Create creates a new battery
func (s *BatteryStore) Create(ctx context.Context, userID string, batteryCode string, params models.CreateBatteryParams) (*models.Battery, error) {
	query := `
		INSERT INTO batteries (user_id, battery_code, name, chemistry, cells, capacity_mah, c_rating, connector, weight_grams, brand, model, purchase_date, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, user_id, battery_code, name, chemistry, cells, capacity_mah, c_rating, connector, weight_grams, brand, model, purchase_date, notes, created_at, updated_at
	`

	battery := &models.Battery{}
	var (
		scanName, scanConnector, scanNotes, scanBrand, scanModel sql.NullString
		scanCRating, scanWeightGrams                             sql.NullInt32
		scanPurchaseDate                                         sql.NullTime
	)

	// Prepare nullable fields
	var name, connector, notes, brand, model sql.NullString
	var cRating, weightGrams sql.NullInt32
	var purchaseDate sql.NullTime

	if params.Name != "" {
		name = sql.NullString{String: params.Name, Valid: true}
	}
	if params.Connector != "" {
		connector = sql.NullString{String: params.Connector, Valid: true}
	}
	if params.Notes != "" {
		notes = sql.NullString{String: params.Notes, Valid: true}
	}
	if params.Brand != "" {
		brand = sql.NullString{String: params.Brand, Valid: true}
	}
	if params.Model != "" {
		model = sql.NullString{String: params.Model, Valid: true}
	}
	if params.CRating != nil {
		cRating = sql.NullInt32{Int32: int32(*params.CRating), Valid: true}
	}
	if params.WeightGrams != nil {
		weightGrams = sql.NullInt32{Int32: int32(*params.WeightGrams), Valid: true}
	}
	if params.PurchaseDate != nil {
		purchaseDate = sql.NullTime{Time: *params.PurchaseDate, Valid: true}
	}

	err := s.db.QueryRowContext(ctx, query,
		userID, batteryCode, name, string(params.Chemistry), params.Cells, params.CapacityMah,
		cRating, connector, weightGrams, brand, model, purchaseDate, notes,
	).Scan(
		&battery.ID, &battery.UserID, &battery.BatteryCode, &scanName,
		&battery.Chemistry, &battery.Cells, &battery.CapacityMah,
		&scanCRating, &scanConnector, &scanWeightGrams, &scanBrand, &scanModel, &scanPurchaseDate, &scanNotes,
		&battery.CreatedAt, &battery.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create battery: %w", err)
	}

	battery.Name = scanName.String
	battery.Connector = scanConnector.String
	battery.Notes = scanNotes.String
	battery.Brand = scanBrand.String
	battery.Model = scanModel.String
	if scanCRating.Valid {
		v := int(scanCRating.Int32)
		battery.CRating = &v
	}
	if scanWeightGrams.Valid {
		v := int(scanWeightGrams.Int32)
		battery.WeightGrams = &v
	}
	if scanPurchaseDate.Valid {
		battery.PurchaseDate = &scanPurchaseDate.Time
	}

	return battery, nil
}

// Get retrieves a battery by ID
func (s *BatteryStore) Get(ctx context.Context, id string, userID string) (*models.Battery, error) {
	query := `
		SELECT b.id, b.user_id, b.battery_code, b.name, b.chemistry, b.cells, b.capacity_mah,
		       b.c_rating, b.connector, b.weight_grams, b.brand, b.model, b.purchase_date, b.notes, b.created_at, b.updated_at,
		       COALESCE(SUM(l.cycle_delta), 0) as total_cycles,
		       MAX(l.logged_at) as last_logged
		FROM batteries b
		LEFT JOIN battery_logs l ON l.battery_id = b.id
		WHERE b.id = $1 AND b.user_id = $2
		GROUP BY b.id
	`

	battery := &models.Battery{}
	var (
		scanName, scanConnector, scanNotes, scanBrand, scanModel sql.NullString
		scanCRating, scanWeightGrams                             sql.NullInt32
		scanPurchaseDate, scanLastLogged                         sql.NullTime
	)

	err := s.db.QueryRowContext(ctx, query, id, userID).Scan(
		&battery.ID, &battery.UserID, &battery.BatteryCode, &scanName,
		&battery.Chemistry, &battery.Cells, &battery.CapacityMah,
		&scanCRating, &scanConnector, &scanWeightGrams, &scanBrand, &scanModel, &scanPurchaseDate, &scanNotes,
		&battery.CreatedAt, &battery.UpdatedAt,
		&battery.TotalCycles, &scanLastLogged,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get battery: %w", err)
	}

	battery.Name = scanName.String
	battery.Connector = scanConnector.String
	battery.Notes = scanNotes.String
	battery.Brand = scanBrand.String
	battery.Model = scanModel.String
	if scanCRating.Valid {
		v := int(scanCRating.Int32)
		battery.CRating = &v
	}
	if scanWeightGrams.Valid {
		v := int(scanWeightGrams.Int32)
		battery.WeightGrams = &v
	}
	if scanPurchaseDate.Valid {
		battery.PurchaseDate = &scanPurchaseDate.Time
	}
	if scanLastLogged.Valid {
		battery.LastLoggedDate = &scanLastLogged.Time
	}

	return battery, nil
}

// GetByCode retrieves a battery by battery code for a user
func (s *BatteryStore) GetByCode(ctx context.Context, batteryCode string, userID string) (*models.Battery, error) {
	query := `
		SELECT b.id, b.user_id, b.battery_code, b.name, b.chemistry, b.cells, b.capacity_mah,
		       b.c_rating, b.connector, b.weight_grams, b.brand, b.model, b.purchase_date, b.notes, b.created_at, b.updated_at,
		       COALESCE(SUM(l.cycle_delta), 0) as total_cycles,
		       MAX(l.logged_at) as last_logged
		FROM batteries b
		LEFT JOIN battery_logs l ON l.battery_id = b.id
		WHERE b.battery_code = $1 AND b.user_id = $2
		GROUP BY b.id
	`

	battery := &models.Battery{}
	var (
		scanName, scanConnector, scanNotes, scanBrand, scanModel sql.NullString
		scanCRating, scanWeightGrams                             sql.NullInt32
		scanPurchaseDate, scanLastLogged                         sql.NullTime
	)

	err := s.db.QueryRowContext(ctx, query, batteryCode, userID).Scan(
		&battery.ID, &battery.UserID, &battery.BatteryCode, &scanName,
		&battery.Chemistry, &battery.Cells, &battery.CapacityMah,
		&scanCRating, &scanConnector, &scanWeightGrams, &scanBrand, &scanModel, &scanPurchaseDate, &scanNotes,
		&battery.CreatedAt, &battery.UpdatedAt,
		&battery.TotalCycles, &scanLastLogged,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get battery by code: %w", err)
	}

	battery.Name = scanName.String
	battery.Connector = scanConnector.String
	battery.Notes = scanNotes.String
	battery.Brand = scanBrand.String
	battery.Model = scanModel.String
	if scanCRating.Valid {
		v := int(scanCRating.Int32)
		battery.CRating = &v
	}
	if scanWeightGrams.Valid {
		v := int(scanWeightGrams.Int32)
		battery.WeightGrams = &v
	}
	if scanPurchaseDate.Valid {
		battery.PurchaseDate = &scanPurchaseDate.Time
	}
	if scanLastLogged.Valid {
		battery.LastLoggedDate = &scanLastLogged.Time
	}

	return battery, nil
}

// Update updates a battery
func (s *BatteryStore) Update(ctx context.Context, userID string, params models.UpdateBatteryParams) (*models.Battery, error) {
	setClauses := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIndex := 1

	if params.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *params.Name)
		argIndex++
	}
	if params.Chemistry != nil {
		setClauses = append(setClauses, fmt.Sprintf("chemistry = $%d", argIndex))
		args = append(args, string(*params.Chemistry))
		argIndex++
	}
	if params.Cells != nil {
		setClauses = append(setClauses, fmt.Sprintf("cells = $%d", argIndex))
		args = append(args, *params.Cells)
		argIndex++
	}
	if params.CapacityMah != nil {
		setClauses = append(setClauses, fmt.Sprintf("capacity_mah = $%d", argIndex))
		args = append(args, *params.CapacityMah)
		argIndex++
	}
	if params.CRating != nil {
		setClauses = append(setClauses, fmt.Sprintf("c_rating = $%d", argIndex))
		args = append(args, *params.CRating)
		argIndex++
	}
	if params.Connector != nil {
		setClauses = append(setClauses, fmt.Sprintf("connector = $%d", argIndex))
		args = append(args, *params.Connector)
		argIndex++
	}
	if params.WeightGrams != nil {
		setClauses = append(setClauses, fmt.Sprintf("weight_grams = $%d", argIndex))
		args = append(args, *params.WeightGrams)
		argIndex++
	}
	if params.Brand != nil {
		setClauses = append(setClauses, fmt.Sprintf("brand = $%d", argIndex))
		args = append(args, *params.Brand)
		argIndex++
	}
	if params.Model != nil {
		setClauses = append(setClauses, fmt.Sprintf("model = $%d", argIndex))
		args = append(args, *params.Model)
		argIndex++
	}
	if params.PurchaseDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("purchase_date = $%d", argIndex))
		args = append(args, *params.PurchaseDate)
		argIndex++
	}
	if params.Notes != nil {
		setClauses = append(setClauses, fmt.Sprintf("notes = $%d", argIndex))
		args = append(args, *params.Notes)
		argIndex++
	}

	query := fmt.Sprintf(`
		UPDATE batteries SET %s
		WHERE id = $%d AND user_id = $%d
		RETURNING id, user_id, battery_code, name, chemistry, cells, capacity_mah, c_rating, connector, weight_grams, brand, model, purchase_date, notes, created_at, updated_at
	`, strings.Join(setClauses, ", "), argIndex, argIndex+1)

	args = append(args, params.ID, userID)

	battery := &models.Battery{}
	var (
		scanName, scanConnector, scanNotes, scanBrand, scanModel sql.NullString
		scanCRating, scanWeightGrams                             sql.NullInt32
		scanPurchaseDate                                         sql.NullTime
	)

	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&battery.ID, &battery.UserID, &battery.BatteryCode, &scanName,
		&battery.Chemistry, &battery.Cells, &battery.CapacityMah,
		&scanCRating, &scanConnector, &scanWeightGrams, &scanBrand, &scanModel, &scanPurchaseDate, &scanNotes,
		&battery.CreatedAt, &battery.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update battery: %w", err)
	}

	battery.Name = scanName.String
	battery.Connector = scanConnector.String
	battery.Notes = scanNotes.String
	battery.Brand = scanBrand.String
	battery.Model = scanModel.String
	if scanCRating.Valid {
		v := int(scanCRating.Int32)
		battery.CRating = &v
	}
	if scanWeightGrams.Valid {
		v := int(scanWeightGrams.Int32)
		battery.WeightGrams = &v
	}
	if scanPurchaseDate.Valid {
		battery.PurchaseDate = &scanPurchaseDate.Time
	}

	return battery, nil
}

// Delete deletes a battery
func (s *BatteryStore) Delete(ctx context.Context, id string, userID string) error {
	query := `DELETE FROM batteries WHERE id = $1 AND user_id = $2`
	result, err := s.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete battery: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("battery not found")
	}
	return nil
}

// List lists batteries for a user
func (s *BatteryStore) List(ctx context.Context, userID string, params models.BatteryListParams) (*models.BatteryListResponse, error) {
	// Build WHERE clause
	whereClauses := []string{"b.user_id = $1"}
	args := []interface{}{userID}
	argIndex := 2

	if params.Chemistry != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("b.chemistry = $%d", argIndex))
		args = append(args, string(params.Chemistry))
		argIndex++
	}
	if params.Cells > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("b.cells = $%d", argIndex))
		args = append(args, params.Cells)
		argIndex++
	}
	if params.MinCapacity > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("b.capacity_mah >= $%d", argIndex))
		args = append(args, params.MinCapacity)
		argIndex++
	}
	if params.MaxCapacity > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("b.capacity_mah <= $%d", argIndex))
		args = append(args, params.MaxCapacity)
		argIndex++
	}
	if params.Query != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(b.name ILIKE $%d OR b.battery_code ILIKE $%d)", argIndex, argIndex+1))
		searchPattern := "%" + params.Query + "%"
		args = append(args, searchPattern, searchPattern)
		argIndex += 2
	}

	whereClause := strings.Join(whereClauses, " AND ")

	// Count query
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM batteries b WHERE %s`, whereClause)
	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count batteries: %w", err)
	}

	// Normalize and validate sort direction from params.SortOrder, if available.
	// Only "ASC" and "DESC" are accepted; anything else falls back to the
	// existing default for each sort field to preserve current behavior.
	direction := strings.ToUpper(params.SortOrder)
	validDirection := direction == "ASC" || direction == "DESC"

	// Determine sort order
	orderBy := "b.updated_at DESC"
	switch params.Sort {
	case "name":
		if !validDirection {
			direction = "ASC"
		}
		orderBy = fmt.Sprintf("COALESCE(NULLIF(b.name, ''), b.battery_code) %s", direction)
	case "logged":
		if !validDirection {
			direction = "DESC"
		}
		if direction == "ASC" {
			orderBy = "last_logged ASC NULLS FIRST"
		} else {
			orderBy = "last_logged DESC NULLS LAST"
		}
	case "cycles":
		if !validDirection {
			direction = "DESC"
		}
		orderBy = fmt.Sprintf("total_cycles %s", direction)
	case "updated":
		if !validDirection {
			direction = "DESC"
		}
		orderBy = fmt.Sprintf("b.updated_at %s", direction)
	}

	// List query
	query := fmt.Sprintf(`
		SELECT b.id, b.user_id, b.battery_code, b.name, b.chemistry, b.cells, b.capacity_mah,
		       b.c_rating, b.connector, b.weight_grams, b.brand, b.model, b.purchase_date, b.notes, b.created_at, b.updated_at,
		       COALESCE(SUM(l.cycle_delta), 0) as total_cycles,
		       MAX(l.logged_at) as last_logged
		FROM batteries b
		LEFT JOIN battery_logs l ON l.battery_id = b.id
		WHERE %s
		GROUP BY b.id
		ORDER BY %s
	`, whereClause, orderBy)

	// Add pagination
	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query += fmt.Sprintf(" LIMIT $%d", argIndex)
	args = append(args, limit)
	argIndex++

	if params.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, params.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list batteries: %w", err)
	}
	defer rows.Close()

	batteries := []models.Battery{}
	for rows.Next() {
		battery := models.Battery{}
		var (
			scanName, scanConnector, scanNotes, scanBrand, scanModel sql.NullString
			scanCRating, scanWeightGrams                             sql.NullInt32
			scanPurchaseDate, scanLastLogged                         sql.NullTime
		)

		if err := rows.Scan(
			&battery.ID, &battery.UserID, &battery.BatteryCode, &scanName,
			&battery.Chemistry, &battery.Cells, &battery.CapacityMah,
			&scanCRating, &scanConnector, &scanWeightGrams, &scanBrand, &scanModel, &scanPurchaseDate, &scanNotes,
			&battery.CreatedAt, &battery.UpdatedAt,
			&battery.TotalCycles, &scanLastLogged,
		); err != nil {
			return nil, fmt.Errorf("failed to scan battery: %w", err)
		}

		battery.Name = scanName.String
		battery.Connector = scanConnector.String
		battery.Notes = scanNotes.String
		battery.Brand = scanBrand.String
		battery.Model = scanModel.String
		if scanCRating.Valid {
			v := int(scanCRating.Int32)
			battery.CRating = &v
		}
		if scanWeightGrams.Valid {
			v := int(scanWeightGrams.Int32)
			battery.WeightGrams = &v
		}
		if scanPurchaseDate.Valid {
			battery.PurchaseDate = &scanPurchaseDate.Time
		}
		if scanLastLogged.Valid {
			battery.LastLoggedDate = &scanLastLogged.Time
		}

		batteries = append(batteries, battery)
	}

	return &models.BatteryListResponse{
		Batteries:  batteries,
		TotalCount: totalCount,
	}, nil
}

// BatteryCodeExists checks if a battery code already exists for a user
func (s *BatteryStore) BatteryCodeExists(ctx context.Context, userID string, code string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM batteries WHERE user_id = $1 AND battery_code = $2)`
	var exists bool
	err := s.db.QueryRowContext(ctx, query, userID, code).Scan(&exists)
	return exists, err
}

// CreateLog creates a new battery log entry
func (s *BatteryStore) CreateLog(ctx context.Context, userID string, params models.CreateBatteryLogParams) (*models.BatteryLog, error) {
	// Verify battery belongs to user
	var batteryUserID string
	err := s.db.QueryRowContext(ctx, "SELECT user_id FROM batteries WHERE id = $1", params.BatteryID).Scan(&batteryUserID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("battery not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to verify battery: %w", err)
	}
	if batteryUserID != userID {
		return nil, fmt.Errorf("battery not found")
	}

	query := `
		INSERT INTO battery_logs (battery_id, user_id, logged_at, cycle_delta, ir_mohm_per_cell, min_cell_v, max_cell_v, storage_ok, notes)
		VALUES ($1, $2, COALESCE($3, NOW()), $4, $5, $6, $7, $8, $9)
		RETURNING id, battery_id, user_id, logged_at, cycle_delta, ir_mohm_per_cell, min_cell_v, max_cell_v, storage_ok, notes, created_at
	`

	var loggedAt sql.NullTime
	if params.LoggedAt != nil {
		loggedAt = sql.NullTime{Time: *params.LoggedAt, Valid: true}
	}

	var irJSON interface{}
	if params.IRMohmPerCell != nil {
		irJSON = params.IRMohmPerCell
	}

	log := &models.BatteryLog{}
	var (
		scanNotes              sql.NullString
		scanMinV, scanMaxV     sql.NullFloat64
		scanStorageOk          sql.NullBool
		scanIR                 []byte
	)

	err = s.db.QueryRowContext(ctx, query,
		params.BatteryID, userID, loggedAt, params.CycleDelta, irJSON,
		params.MinCellV, params.MaxCellV, params.StorageOk, params.Notes,
	).Scan(
		&log.ID, &log.BatteryID, &log.UserID, &log.LoggedAt, &log.CycleDelta,
		&scanIR, &scanMinV, &scanMaxV, &scanStorageOk, &scanNotes, &log.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create battery log: %w", err)
	}

	log.Notes = scanNotes.String
	if scanMinV.Valid {
		log.MinCellV = &scanMinV.Float64
	}
	if scanMaxV.Valid {
		log.MaxCellV = &scanMaxV.Float64
	}
	if scanStorageOk.Valid {
		log.StorageOk = &scanStorageOk.Bool
	}
	if scanIR != nil {
		log.IRMohmPerCell = json.RawMessage(scanIR)
	}

	return log, nil
}

// ListLogs lists logs for a battery
func (s *BatteryStore) ListLogs(ctx context.Context, batteryID string, userID string, limit int) (*models.BatteryLogListResponse, error) {
	// Verify battery belongs to user
	var batteryUserID string
	err := s.db.QueryRowContext(ctx, "SELECT user_id FROM batteries WHERE id = $1", batteryID).Scan(&batteryUserID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("battery not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to verify battery: %w", err)
	}
	if batteryUserID != userID {
		return nil, fmt.Errorf("battery not found")
	}

	// Count
	var totalCount int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM battery_logs WHERE battery_id = $1", batteryID).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count logs: %w", err)
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT id, battery_id, user_id, logged_at, cycle_delta, ir_mohm_per_cell, min_cell_v, max_cell_v, storage_ok, notes, created_at
		FROM battery_logs
		WHERE battery_id = $1
		ORDER BY logged_at DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, batteryID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list logs: %w", err)
	}
	defer rows.Close()

	logs := []models.BatteryLog{}
	for rows.Next() {
		log := models.BatteryLog{}
		var (
			scanNotes          sql.NullString
			scanMinV, scanMaxV sql.NullFloat64
			scanStorageOk      sql.NullBool
			scanIR             []byte
		)

		if err := rows.Scan(
			&log.ID, &log.BatteryID, &log.UserID, &log.LoggedAt, &log.CycleDelta,
			&scanIR, &scanMinV, &scanMaxV, &scanStorageOk, &scanNotes, &log.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan log: %w", err)
		}

		log.Notes = scanNotes.String
		if scanMinV.Valid {
			log.MinCellV = &scanMinV.Float64
		}
		if scanMaxV.Valid {
			log.MaxCellV = &scanMaxV.Float64
		}
		if scanStorageOk.Valid {
			log.StorageOk = &scanStorageOk.Bool
		}
		if scanIR != nil {
			log.IRMohmPerCell = json.RawMessage(scanIR)
		}

		logs = append(logs, log)
	}

	return &models.BatteryLogListResponse{
		Logs:       logs,
		TotalCount: totalCount,
	}, nil
}

// DeleteLog deletes a battery log entry
func (s *BatteryStore) DeleteLog(ctx context.Context, logID string, userID string) error {
	query := `DELETE FROM battery_logs WHERE id = $1 AND user_id = $2`
	result, err := s.db.ExecContext(ctx, query, logID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete log: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("log not found")
	}
	return nil
}

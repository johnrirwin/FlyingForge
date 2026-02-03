package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/johnrirwin/flyingforge/internal/models"
)

// FCConfigStore handles flight controller config database operations
type FCConfigStore struct {
	db *DB
}

// NewFCConfigStore creates a new FC config store
func NewFCConfigStore(db *DB) *FCConfigStore {
	return &FCConfigStore{db: db}
}

// SaveConfig creates a new flight controller config
func (s *FCConfigStore) SaveConfig(ctx context.Context, userID string, config *models.FlightControllerConfig) error {
	parseWarnings, err := json.Marshal(config.ParseWarnings)
	if err != nil {
		parseWarnings = []byte("[]")
	}

	var parsedTuning []byte
	if config.ParsedTuning != nil {
		parsedTuning, err = json.Marshal(config.ParsedTuning)
		if err != nil {
			parsedTuning = nil
		}
	}

	query := `
		INSERT INTO fc_configs (
			user_id, inventory_item_id, name, notes, raw_cli_dump,
			firmware_name, firmware_version, board_target, board_name, mcu_type,
			parse_status, parse_warnings, parsed_tuning
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, created_at, updated_at
	`

	err = s.db.QueryRowContext(ctx, query,
		userID,
		config.InventoryItemID,
		config.Name,
		nullString(config.Notes),
		config.RawCLIDump,
		config.FirmwareName,
		nullString(config.FirmwareVersion),
		nullString(config.BoardTarget),
		nullString(config.BoardName),
		nullString(config.MCUType),
		config.ParseStatus,
		parseWarnings,
		parsedTuning,
	).Scan(&config.ID, &config.CreatedAt, &config.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to save FC config: %w", err)
	}

	config.UserID = userID
	return nil
}

// GetConfig retrieves a config by ID
func (s *FCConfigStore) GetConfig(ctx context.Context, id string, userID string) (*models.FlightControllerConfig, error) {
	query := `
		SELECT id, user_id, inventory_item_id, name, notes, raw_cli_dump,
			   firmware_name, firmware_version, board_target, board_name, mcu_type,
			   parse_status, parse_warnings, parsed_tuning, created_at, updated_at
		FROM fc_configs
		WHERE id = $1 AND user_id = $2
	`

	config := &models.FlightControllerConfig{}
	var notes, firmwareVersion, boardTarget, boardName, mcuType sql.NullString
	var parseWarnings, parsedTuning []byte

	err := s.db.QueryRowContext(ctx, query, id, userID).Scan(
		&config.ID,
		&config.UserID,
		&config.InventoryItemID,
		&config.Name,
		&notes,
		&config.RawCLIDump,
		&config.FirmwareName,
		&firmwareVersion,
		&boardTarget,
		&boardName,
		&mcuType,
		&config.ParseStatus,
		&parseWarnings,
		&parsedTuning,
		&config.CreatedAt,
		&config.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get FC config: %w", err)
	}

	config.Notes = notes.String
	config.FirmwareVersion = firmwareVersion.String
	config.BoardTarget = boardTarget.String
	config.BoardName = boardName.String
	config.MCUType = mcuType.String

	if len(parseWarnings) > 0 {
		json.Unmarshal(parseWarnings, &config.ParseWarnings)
	}
	if len(parsedTuning) > 0 {
		config.ParsedTuning = &models.ParsedTuning{}
		json.Unmarshal(parsedTuning, config.ParsedTuning)
	}

	return config, nil
}

// ListConfigs lists configs for a user, optionally filtered by inventory item
func (s *FCConfigStore) ListConfigs(ctx context.Context, userID string, params models.FCConfigListParams) (*models.FCConfigListResponse, error) {
	countQuery := `SELECT COUNT(*) FROM fc_configs WHERE user_id = $1`
	listQuery := `
		SELECT id, user_id, inventory_item_id, name, notes,
			   firmware_name, firmware_version, board_target, board_name, mcu_type,
			   parse_status, parse_warnings, created_at, updated_at
		FROM fc_configs
		WHERE user_id = $1
	`
	args := []interface{}{userID}
	argIdx := 2

	if params.InventoryItemID != "" {
		countQuery += fmt.Sprintf(" AND inventory_item_id = $%d", argIdx)
		listQuery += fmt.Sprintf(" AND inventory_item_id = $%d", argIdx)
		args = append(args, params.InventoryItemID)
		argIdx++
	}

	listQuery += " ORDER BY created_at DESC"

	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	listQuery += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	// Get total count
	var totalCount int
	err := s.db.QueryRowContext(ctx, countQuery, args[:len(args)]...).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count FC configs: %w", err)
	}

	// Get configs
	rows, err := s.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list FC configs: %w", err)
	}
	defer rows.Close()

	configs := make([]models.FlightControllerConfig, 0)
	for rows.Next() {
		config := models.FlightControllerConfig{}
		var notes, firmwareVersion, boardTarget, boardName, mcuType sql.NullString
		var parseWarnings []byte

		err := rows.Scan(
			&config.ID,
			&config.UserID,
			&config.InventoryItemID,
			&config.Name,
			&notes,
			&config.FirmwareName,
			&firmwareVersion,
			&boardTarget,
			&boardName,
			&mcuType,
			&config.ParseStatus,
			&parseWarnings,
			&config.CreatedAt,
			&config.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan FC config: %w", err)
		}

		config.Notes = notes.String
		config.FirmwareVersion = firmwareVersion.String
		config.BoardTarget = boardTarget.String
		config.BoardName = boardName.String
		config.MCUType = mcuType.String

		if len(parseWarnings) > 0 {
			json.Unmarshal(parseWarnings, &config.ParseWarnings)
		}

		configs = append(configs, config)
	}

	return &models.FCConfigListResponse{
		Configs:    configs,
		TotalCount: totalCount,
	}, nil
}

// UpdateConfig updates a config's metadata (name, notes)
func (s *FCConfigStore) UpdateConfig(ctx context.Context, id string, userID string, params models.UpdateFCConfigParams) (*models.FlightControllerConfig, error) {
	// Build update query
	setFields := []string{}
	args := []interface{}{}
	argIdx := 1

	if params.Name != nil {
		setFields = append(setFields, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *params.Name)
		argIdx++
	}
	if params.Notes != nil {
		setFields = append(setFields, fmt.Sprintf("notes = $%d", argIdx))
		args = append(args, *params.Notes)
		argIdx++
	}

	if len(setFields) == 0 {
		return s.GetConfig(ctx, id, userID)
	}

	setFields = append(setFields, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now())
	argIdx++

	args = append(args, id, userID)

	query := fmt.Sprintf(`
		UPDATE fc_configs
		SET %s
		WHERE id = $%d AND user_id = $%d
	`, joinStrings(setFields, ", "), argIdx, argIdx+1)

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update FC config: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, nil
	}

	return s.GetConfig(ctx, id, userID)
}

// DeleteConfig deletes a config
func (s *FCConfigStore) DeleteConfig(ctx context.Context, id string, userID string) error {
	query := `DELETE FROM fc_configs WHERE id = $1 AND user_id = $2`
	result, err := s.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete FC config: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("config not found")
	}

	return nil
}

// SaveTuningSnapshot creates a new tuning snapshot for an aircraft
func (s *FCConfigStore) SaveTuningSnapshot(ctx context.Context, snapshot *models.AircraftTuningSnapshot) error {
	tuningData, err := json.Marshal(snapshot.TuningData)
	if err != nil {
		return fmt.Errorf("failed to marshal tuning data: %w", err)
	}

	parseWarnings, err := json.Marshal(snapshot.ParseWarnings)
	if err != nil {
		parseWarnings = []byte("[]")
	}

	query := `
		INSERT INTO aircraft_tuning_snapshots (
			aircraft_id, flight_controller_id, flight_controller_config_id,
			firmware_name, firmware_version, board_target, board_name,
			tuning_data, parse_status, parse_warnings, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`

	err = s.db.QueryRowContext(ctx, query,
		snapshot.AircraftID,
		nullString(snapshot.FlightControllerID),
		nullString(snapshot.FlightControllerConfigID),
		snapshot.FirmwareName,
		nullString(snapshot.FirmwareVersion),
		nullString(snapshot.BoardTarget),
		nullString(snapshot.BoardName),
		tuningData,
		snapshot.ParseStatus,
		parseWarnings,
		nullString(snapshot.Notes),
	).Scan(&snapshot.ID, &snapshot.CreatedAt, &snapshot.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to save tuning snapshot: %w", err)
	}

	return nil
}

// GetLatestTuningSnapshot gets the most recent tuning snapshot for an aircraft
func (s *FCConfigStore) GetLatestTuningSnapshot(ctx context.Context, aircraftID string, userID string) (*models.AircraftTuningSnapshot, error) {
	// Verify user owns the aircraft
	query := `
		SELECT ts.id, ts.aircraft_id, ts.flight_controller_id, ts.flight_controller_config_id,
			   ts.firmware_name, ts.firmware_version, ts.board_target, ts.board_name,
			   ts.tuning_data, ts.parse_status, ts.parse_warnings, ts.notes,
			   ts.created_at, ts.updated_at
		FROM aircraft_tuning_snapshots ts
		INNER JOIN aircraft a ON a.id = ts.aircraft_id
		WHERE ts.aircraft_id = $1 AND a.user_id = $2
		ORDER BY ts.created_at DESC
		LIMIT 1
	`

	snapshot := &models.AircraftTuningSnapshot{}
	var fcID, configID, firmwareVersion, boardTarget, boardName, notes sql.NullString
	var tuningData, parseWarnings []byte

	err := s.db.QueryRowContext(ctx, query, aircraftID, userID).Scan(
		&snapshot.ID,
		&snapshot.AircraftID,
		&fcID,
		&configID,
		&snapshot.FirmwareName,
		&firmwareVersion,
		&boardTarget,
		&boardName,
		&tuningData,
		&snapshot.ParseStatus,
		&parseWarnings,
		&notes,
		&snapshot.CreatedAt,
		&snapshot.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tuning snapshot: %w", err)
	}

	snapshot.FlightControllerID = fcID.String
	snapshot.FlightControllerConfigID = configID.String
	snapshot.FirmwareVersion = firmwareVersion.String
	snapshot.BoardTarget = boardTarget.String
	snapshot.BoardName = boardName.String
	snapshot.Notes = notes.String

	snapshot.TuningData = tuningData

	if len(parseWarnings) > 0 {
		json.Unmarshal(parseWarnings, &snapshot.ParseWarnings)
	}

	return snapshot, nil
}

// ListTuningSnapshots lists all tuning snapshots for an aircraft
func (s *FCConfigStore) ListTuningSnapshots(ctx context.Context, aircraftID string, userID string) ([]models.AircraftTuningSnapshot, error) {
	query := `
		SELECT ts.id, ts.aircraft_id, ts.flight_controller_id, ts.flight_controller_config_id,
			   ts.firmware_name, ts.firmware_version, ts.board_target, ts.board_name,
			   ts.parse_status, ts.parse_warnings, ts.notes,
			   ts.created_at, ts.updated_at
		FROM aircraft_tuning_snapshots ts
		INNER JOIN aircraft a ON a.id = ts.aircraft_id
		WHERE ts.aircraft_id = $1 AND a.user_id = $2
		ORDER BY ts.created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, aircraftID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tuning snapshots: %w", err)
	}
	defer rows.Close()

	snapshots := make([]models.AircraftTuningSnapshot, 0)
	for rows.Next() {
		snapshot := models.AircraftTuningSnapshot{}
		var fcID, configID, firmwareVersion, boardTarget, boardName, notes sql.NullString
		var parseWarnings []byte

		err := rows.Scan(
			&snapshot.ID,
			&snapshot.AircraftID,
			&fcID,
			&configID,
			&snapshot.FirmwareName,
			&firmwareVersion,
			&boardTarget,
			&boardName,
			&snapshot.ParseStatus,
			&parseWarnings,
			&notes,
			&snapshot.CreatedAt,
			&snapshot.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tuning snapshot: %w", err)
		}

		snapshot.FlightControllerID = fcID.String
		snapshot.FlightControllerConfigID = configID.String
		snapshot.FirmwareVersion = firmwareVersion.String
		snapshot.BoardTarget = boardTarget.String
		snapshot.BoardName = boardName.String
		snapshot.Notes = notes.String

		if len(parseWarnings) > 0 {
			json.Unmarshal(parseWarnings, &snapshot.ParseWarnings)
		}

		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

// GetAircraftByFC finds an aircraft that has the given FC (inventory item) assigned
func (s *FCConfigStore) GetAircraftByFC(ctx context.Context, userID string, inventoryItemID string) (*models.Aircraft, error) {
	query := `
		SELECT a.id, a.user_id, a.name, a.nickname, a.type, 
			   (a.image_data IS NOT NULL) as has_image, a.description,
			   a.created_at, a.updated_at
		FROM aircraft a
		INNER JOIN aircraft_components ac ON ac.aircraft_id = a.id
		WHERE a.user_id = $1 
		  AND ac.inventory_item_id = $2 
		  AND ac.category = 'fc'
		LIMIT 1
	`

	aircraft := &models.Aircraft{}
	var nickname, aircraftType, description sql.NullString

	err := s.db.QueryRowContext(ctx, query, userID, inventoryItemID).Scan(
		&aircraft.ID,
		&aircraft.UserID,
		&aircraft.Name,
		&nickname,
		&aircraftType,
		&aircraft.HasImage,
		&description,
		&aircraft.CreatedAt,
		&aircraft.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find aircraft by FC: %w", err)
	}

	aircraft.Nickname = nickname.String
	aircraft.Type = models.AircraftType(aircraftType.String)
	aircraft.Description = description.String

	return aircraft, nil
}

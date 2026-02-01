package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/johnrirwin/rotorlife/internal/models"
)

// RadioStore handles radio database operations
type RadioStore struct {
	db *DB
}

// NewRadioStore creates a new radio store
func NewRadioStore(db *DB) *RadioStore {
	return &RadioStore{db: db}
}

// CreateRadio creates a new radio
func (s *RadioStore) CreateRadio(ctx context.Context, userID string, params models.CreateRadioParams) (*models.Radio, error) {
	query := `
		INSERT INTO radios (user_id, manufacturer, model, firmware_family, notes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	radio := &models.Radio{
		UserID:         userID,
		Manufacturer:   params.Manufacturer,
		Model:          params.Model,
		FirmwareFamily: params.FirmwareFamily,
		Notes:          params.Notes,
	}

	err := s.db.QueryRowContext(ctx, query,
		nullString(userID),
		string(radio.Manufacturer),
		radio.Model,
		nullString(string(radio.FirmwareFamily)),
		nullString(radio.Notes),
	).Scan(&radio.ID, &radio.CreatedAt, &radio.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create radio: %w", err)
	}

	return radio, nil
}

// GetRadio retrieves a radio by ID (optionally scoped to user)
func (s *RadioStore) GetRadio(ctx context.Context, id string, userID string) (*models.Radio, error) {
	query := `
		SELECT id, user_id, manufacturer, model, firmware_family, notes, created_at, updated_at
		FROM radios
		WHERE id = $1
	`
	args := []interface{}{id}

	if userID != "" {
		query = `
			SELECT id, user_id, manufacturer, model, firmware_family, notes, created_at, updated_at
			FROM radios
			WHERE id = $1 AND user_id = $2
		`
		args = append(args, userID)
	}

	radio := &models.Radio{}
	var radioUserID, firmwareFamily, notes sql.NullString

	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&radio.ID,
		&radioUserID,
		&radio.Manufacturer,
		&radio.Model,
		&firmwareFamily,
		&notes,
		&radio.CreatedAt,
		&radio.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get radio: %w", err)
	}

	if radioUserID.Valid {
		radio.UserID = radioUserID.String
	}
	if firmwareFamily.Valid {
		radio.FirmwareFamily = models.FirmwareFamily(firmwareFamily.String)
	}
	if notes.Valid {
		radio.Notes = notes.String
	}

	return radio, nil
}

// ListRadios lists radios for a user
func (s *RadioStore) ListRadios(ctx context.Context, userID string, params models.RadioListParams) (*models.RadioListResponse, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM radios WHERE user_id = $1`
	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, userID).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count radios: %w", err)
	}

	// Set defaults
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, user_id, manufacturer, model, firmware_family, notes, created_at, updated_at
		FROM radios
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list radios: %w", err)
	}
	defer rows.Close()

	radios := []models.Radio{}
	for rows.Next() {
		radio := models.Radio{}
		var radioUserID, firmwareFamily, notes sql.NullString

		if err := rows.Scan(
			&radio.ID,
			&radioUserID,
			&radio.Manufacturer,
			&radio.Model,
			&firmwareFamily,
			&notes,
			&radio.CreatedAt,
			&radio.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan radio: %w", err)
		}

		if radioUserID.Valid {
			radio.UserID = radioUserID.String
		}
		if firmwareFamily.Valid {
			radio.FirmwareFamily = models.FirmwareFamily(firmwareFamily.String)
		}
		if notes.Valid {
			radio.Notes = notes.String
		}

		radios = append(radios, radio)
	}

	return &models.RadioListResponse{
		Radios:     radios,
		TotalCount: totalCount,
	}, nil
}

// UpdateRadio updates a radio
func (s *RadioStore) UpdateRadio(ctx context.Context, id string, userID string, params models.UpdateRadioParams) (*models.Radio, error) {
	// First check the radio exists and belongs to user
	existing, err := s.GetRadio(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("radio not found")
	}

	// Build update query dynamically
	setClauses := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argNum := 1

	if params.FirmwareFamily != nil {
		setClauses = append(setClauses, fmt.Sprintf("firmware_family = $%d", argNum))
		args = append(args, string(*params.FirmwareFamily))
		argNum++
	}

	if params.Notes != nil {
		setClauses = append(setClauses, fmt.Sprintf("notes = $%d", argNum))
		args = append(args, *params.Notes)
		argNum++
	}

	args = append(args, id, userID)

	query := fmt.Sprintf(`
		UPDATE radios
		SET %s
		WHERE id = $%d AND user_id = $%d
		RETURNING id, user_id, manufacturer, model, firmware_family, notes, created_at, updated_at
	`, joinStrings(setClauses, ", "), argNum, argNum+1)

	radio := &models.Radio{}
	var radioUserID, firmwareFamily, notes sql.NullString

	err = s.db.QueryRowContext(ctx, query, args...).Scan(
		&radio.ID,
		&radioUserID,
		&radio.Manufacturer,
		&radio.Model,
		&firmwareFamily,
		&notes,
		&radio.CreatedAt,
		&radio.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update radio: %w", err)
	}

	if radioUserID.Valid {
		radio.UserID = radioUserID.String
	}
	if firmwareFamily.Valid {
		radio.FirmwareFamily = models.FirmwareFamily(firmwareFamily.String)
	}
	if notes.Valid {
		radio.Notes = notes.String
	}

	return radio, nil
}

// DeleteRadio deletes a radio
func (s *RadioStore) DeleteRadio(ctx context.Context, id string, userID string) error {
	query := `DELETE FROM radios WHERE id = $1 AND user_id = $2`
	result, err := s.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete radio: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("radio not found")
	}

	return nil
}

// CreateBackup creates a new backup record
func (s *RadioStore) CreateBackup(ctx context.Context, radioID string, params models.CreateRadioBackupParams, storagePath string) (*models.RadioBackup, error) {
	query := `
		INSERT INTO radio_backups (radio_id, backup_name, backup_type, file_name, file_size, checksum, storage_path)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`

	backup := &models.RadioBackup{
		RadioID:     radioID,
		BackupName:  params.BackupName,
		BackupType:  params.BackupType,
		FileName:    params.FileName,
		FileSize:    params.FileSize,
		Checksum:    params.Checksum,
		StoragePath: storagePath,
	}

	err := s.db.QueryRowContext(ctx, query,
		backup.RadioID,
		backup.BackupName,
		string(backup.BackupType),
		backup.FileName,
		backup.FileSize,
		nullString(backup.Checksum),
		backup.StoragePath,
	).Scan(&backup.ID, &backup.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}

	return backup, nil
}

// GetBackup retrieves a backup by ID
func (s *RadioStore) GetBackup(ctx context.Context, id string, radioID string) (*models.RadioBackup, error) {
	query := `
		SELECT id, radio_id, backup_name, backup_type, file_name, file_size, checksum, storage_path, created_at
		FROM radio_backups
		WHERE id = $1 AND radio_id = $2
	`

	backup := &models.RadioBackup{}
	var checksum sql.NullString

	err := s.db.QueryRowContext(ctx, query, id, radioID).Scan(
		&backup.ID,
		&backup.RadioID,
		&backup.BackupName,
		&backup.BackupType,
		&backup.FileName,
		&backup.FileSize,
		&checksum,
		&backup.StoragePath,
		&backup.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get backup: %w", err)
	}

	if checksum.Valid {
		backup.Checksum = checksum.String
	}

	return backup, nil
}

// ListBackups lists backups for a radio
func (s *RadioStore) ListBackups(ctx context.Context, radioID string, params models.RadioBackupListParams) (*models.RadioBackupListResponse, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM radio_backups WHERE radio_id = $1`
	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, radioID).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count backups: %w", err)
	}

	// Set defaults
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, radio_id, backup_name, backup_type, file_name, file_size, checksum, storage_path, created_at
		FROM radio_backups
		WHERE radio_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.QueryContext(ctx, query, radioID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}
	defer rows.Close()

	backups := []models.RadioBackup{}
	for rows.Next() {
		backup := models.RadioBackup{}
		var checksum sql.NullString

		if err := rows.Scan(
			&backup.ID,
			&backup.RadioID,
			&backup.BackupName,
			&backup.BackupType,
			&backup.FileName,
			&backup.FileSize,
			&checksum,
			&backup.StoragePath,
			&backup.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan backup: %w", err)
		}

		if checksum.Valid {
			backup.Checksum = checksum.String
		}

		backups = append(backups, backup)
	}

	return &models.RadioBackupListResponse{
		Backups:    backups,
		TotalCount: totalCount,
	}, nil
}

// DeleteBackup deletes a backup record (caller should handle file deletion)
func (s *RadioStore) DeleteBackup(ctx context.Context, id string, radioID string) (*models.RadioBackup, error) {
	// Get backup first to return storage path
	backup, err := s.GetBackup(ctx, id, radioID)
	if err != nil {
		return nil, err
	}
	if backup == nil {
		return nil, fmt.Errorf("backup not found")
	}

	query := `DELETE FROM radio_backups WHERE id = $1 AND radio_id = $2`
	_, err = s.db.ExecContext(ctx, query, id, radioID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete backup: %w", err)
	}

	return backup, nil
}

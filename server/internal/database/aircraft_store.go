package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/johnrirwin/flyingforge/internal/crypto"
	"github.com/johnrirwin/flyingforge/internal/models"
)

// AircraftStore handles aircraft database operations
type AircraftStore struct {
	db        *DB
	encryptor *crypto.Encryptor
}

// NewAircraftStore creates a new aircraft store
func NewAircraftStore(db *DB, encryptor *crypto.Encryptor) *AircraftStore {
	return &AircraftStore{db: db, encryptor: encryptor}
}

// Create creates a new aircraft
func (s *AircraftStore) Create(ctx context.Context, userID string, params models.CreateAircraftParams) (*models.Aircraft, error) {
	query := `
		INSERT INTO aircraft (user_id, name, nickname, type, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, name, nickname, type, image_data IS NOT NULL as has_image, image_type, description, created_at, updated_at
	`

	aircraft := &models.Aircraft{}
	var userIDNull, nickname, aircraftType, description sql.NullString

	if userID != "" {
		userIDNull = sql.NullString{String: userID, Valid: true}
	}
	if params.Nickname != "" {
		nickname = sql.NullString{String: params.Nickname, Valid: true}
	}
	if params.Type != "" {
		aircraftType = sql.NullString{String: string(params.Type), Valid: true}
	}
	if params.Description != "" {
		description = sql.NullString{String: params.Description, Valid: true}
	}

	var scanUserID, scanNickname, scanType, scanImageType, scanDescription sql.NullString
	err := s.db.QueryRowContext(ctx, query,
		userIDNull, params.Name, nickname, aircraftType, description,
	).Scan(
		&aircraft.ID, &scanUserID, &aircraft.Name, &scanNickname,
		&scanType, &aircraft.HasImage, &scanImageType, &scanDescription,
		&aircraft.CreatedAt, &aircraft.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create aircraft: %w", err)
	}

	aircraft.UserID = scanUserID.String
	aircraft.Nickname = scanNickname.String
	aircraft.Type = models.AircraftType(scanType.String)
	aircraft.ImageType = scanImageType.String
	aircraft.Description = scanDescription.String

	return aircraft, nil
}

// Get retrieves an aircraft by ID
func (s *AircraftStore) Get(ctx context.Context, id string, userID string) (*models.Aircraft, error) {
	query := `
		SELECT id, user_id, name, nickname, type, image_data IS NOT NULL as has_image, image_type, description, created_at, updated_at
		FROM aircraft
		WHERE id = $1 AND (user_id = $2 OR user_id IS NULL)
	`

	aircraft := &models.Aircraft{}
	var scanUserID, scanNickname, scanType, scanImageType, scanDescription sql.NullString

	err := s.db.QueryRowContext(ctx, query, id, userID).Scan(
		&aircraft.ID, &scanUserID, &aircraft.Name, &scanNickname,
		&scanType, &aircraft.HasImage, &scanImageType, &scanDescription,
		&aircraft.CreatedAt, &aircraft.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get aircraft: %w", err)
	}

	aircraft.UserID = scanUserID.String
	aircraft.Nickname = scanNickname.String
	aircraft.Type = models.AircraftType(scanType.String)
	aircraft.ImageType = scanImageType.String
	aircraft.Description = scanDescription.String

	return aircraft, nil
}

// Update updates an aircraft
func (s *AircraftStore) Update(ctx context.Context, userID string, params models.UpdateAircraftParams) (*models.Aircraft, error) {
	// Build dynamic update query
	setClauses := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIndex := 1

	if params.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIndex))
		args = append(args, *params.Name)
		argIndex++
	}
	if params.Nickname != nil {
		setClauses = append(setClauses, fmt.Sprintf("nickname = $%d", argIndex))
		args = append(args, *params.Nickname)
		argIndex++
	}
	if params.Type != nil {
		setClauses = append(setClauses, fmt.Sprintf("type = $%d", argIndex))
		args = append(args, string(*params.Type))
		argIndex++
	}
	if params.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIndex))
		args = append(args, *params.Description)
		argIndex++
	}

	query := fmt.Sprintf(`
		UPDATE aircraft SET %s
		WHERE id = $%d AND user_id = $%d
		RETURNING id, user_id, name, nickname, type, image_data IS NOT NULL as has_image, image_type, description, created_at, updated_at
	`, joinStrings(setClauses, ", "), argIndex, argIndex+1)

	args = append(args, params.ID, userID)

	aircraft := &models.Aircraft{}
	var scanUserID, scanNickname, scanType, scanImageType, scanDescription sql.NullString

	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&aircraft.ID, &scanUserID, &aircraft.Name, &scanNickname,
		&scanType, &aircraft.HasImage, &scanImageType, &scanDescription,
		&aircraft.CreatedAt, &aircraft.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update aircraft: %w", err)
	}

	aircraft.UserID = scanUserID.String
	aircraft.Nickname = scanNickname.String
	aircraft.Type = models.AircraftType(scanType.String)
	aircraft.ImageType = scanImageType.String
	aircraft.Description = scanDescription.String

	return aircraft, nil
}

// Delete deletes an aircraft
func (s *AircraftStore) Delete(ctx context.Context, id string, userID string) error {
	query := `DELETE FROM aircraft WHERE id = $1 AND user_id = $2`
	result, err := s.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete aircraft: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("aircraft not found")
	}
	return nil
}

// List lists all aircraft for a user
func (s *AircraftStore) List(ctx context.Context, userID string, params models.AircraftListParams) (*models.AircraftListResponse, error) {
	// Count query
	countQuery := `SELECT COUNT(*) FROM aircraft WHERE user_id = $1`
	countArgs := []interface{}{userID}

	if params.Type != "" {
		countQuery += ` AND type = $2`
		countArgs = append(countArgs, string(params.Type))
	}

	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count aircraft: %w", err)
	}

	// List query
	query := `
		SELECT id, user_id, name, nickname, type, image_data IS NOT NULL as has_image, image_type, description, created_at, updated_at
		FROM aircraft
		WHERE user_id = $1
	`
	args := []interface{}{userID}
	argIndex := 2

	if params.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", argIndex)
		args = append(args, string(params.Type))
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	limit := params.Limit
	if limit <= 0 {
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
		return nil, fmt.Errorf("failed to list aircraft: %w", err)
	}
	defer rows.Close()

	aircraft := []models.Aircraft{}
	for rows.Next() {
		var a models.Aircraft
		var scanUserID, scanNickname, scanType, scanImageType, scanDescription sql.NullString

		if err := rows.Scan(
			&a.ID, &scanUserID, &a.Name, &scanNickname,
			&scanType, &a.HasImage, &scanImageType, &scanDescription,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan aircraft: %w", err)
		}

		a.UserID = scanUserID.String
		a.Nickname = scanNickname.String
		a.Type = models.AircraftType(scanType.String)
		a.ImageType = scanImageType.String
		a.Description = scanDescription.String

		aircraft = append(aircraft, a)
	}

	return &models.AircraftListResponse{
		Aircraft:   aircraft,
		TotalCount: totalCount,
	}, nil
}

// ListByUserID returns all aircraft for a user (simplified version for public profiles)
func (s *AircraftStore) ListByUserID(ctx context.Context, userID string) ([]*models.Aircraft, error) {
	query := `
		SELECT id, user_id, name, nickname, type, image_data IS NOT NULL as has_image, image_type, description, created_at, updated_at
		FROM aircraft
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list aircraft by user: %w", err)
	}
	defer rows.Close()

	var aircraft []*models.Aircraft
	for rows.Next() {
		a := &models.Aircraft{}
		var scanUserID, scanNickname, scanType, scanImageType, scanDescription sql.NullString

		if err := rows.Scan(
			&a.ID, &scanUserID, &a.Name, &scanNickname,
			&scanType, &a.HasImage, &scanImageType, &scanDescription,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan aircraft: %w", err)
		}

		a.UserID = scanUserID.String
		a.Nickname = scanNickname.String
		a.Type = models.AircraftType(scanType.String)
		a.ImageType = scanImageType.String
		a.Description = scanDescription.String

		aircraft = append(aircraft, a)
	}

	return aircraft, nil
}

// SetComponent sets or updates a component on an aircraft
func (s *AircraftStore) SetComponent(ctx context.Context, aircraftID string, category models.ComponentCategory, inventoryItemID string, notes string) (*models.AircraftComponent, error) {
	query := `
		INSERT INTO aircraft_components (aircraft_id, category, inventory_item_id, notes)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (aircraft_id, category) DO UPDATE SET
			inventory_item_id = EXCLUDED.inventory_item_id,
			notes = EXCLUDED.notes,
			updated_at = NOW()
		RETURNING id, aircraft_id, category, inventory_item_id, notes, created_at, updated_at
	`

	component := &models.AircraftComponent{}
	var scanInventoryItemID sql.NullString
	var scanNotes sql.NullString

	var invItemArg interface{}
	if inventoryItemID != "" {
		invItemArg = inventoryItemID
	}

	err := s.db.QueryRowContext(ctx, query, aircraftID, string(category), invItemArg, nullString(notes)).Scan(
		&component.ID, &component.AircraftID, &component.Category,
		&scanInventoryItemID, &scanNotes,
		&component.CreatedAt, &component.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set component: %w", err)
	}

	component.InventoryItemID = scanInventoryItemID.String
	component.Notes = scanNotes.String

	return component, nil
}

// GetComponents retrieves all components for an aircraft
func (s *AircraftStore) GetComponents(ctx context.Context, aircraftID string) ([]models.AircraftComponent, error) {
	query := `
		SELECT ac.id, ac.aircraft_id, ac.category, ac.inventory_item_id, ac.notes, ac.created_at, ac.updated_at,
			   ii.id, ii.name, ii.category, ii.manufacturer, ii.quantity, ii.condition, ii.notes,
			   ii.purchase_price, ii.image_url, ii.specs
		FROM aircraft_components ac
		LEFT JOIN inventory_items ii ON ac.inventory_item_id = ii.id
		WHERE ac.aircraft_id = $1
		ORDER BY ac.category
	`

	rows, err := s.db.QueryContext(ctx, query, aircraftID)
	if err != nil {
		return nil, fmt.Errorf("failed to get components: %w", err)
	}
	defer rows.Close()

	components := []models.AircraftComponent{}
	for rows.Next() {
		var c models.AircraftComponent
		var scanInventoryItemID, scanNotes sql.NullString
		var invID, invName, invCategory, invManufacturer, invCondition, invNotes, invImageURL sql.NullString
		var invQuantity sql.NullInt32
		var invPrice sql.NullFloat64
		var invSpecs json.RawMessage

		if err := rows.Scan(
			&c.ID, &c.AircraftID, &c.Category, &scanInventoryItemID, &scanNotes, &c.CreatedAt, &c.UpdatedAt,
			&invID, &invName, &invCategory, &invManufacturer, &invQuantity, &invCondition, &invNotes,
			&invPrice, &invImageURL, &invSpecs,
		); err != nil {
			return nil, fmt.Errorf("failed to scan component: %w", err)
		}

		c.InventoryItemID = scanInventoryItemID.String
		c.Notes = scanNotes.String

		if invID.Valid {
			c.InventoryItem = &models.InventoryItem{
				ID:           invID.String,
				Name:         invName.String,
				Category:     models.EquipmentCategory(invCategory.String),
				Manufacturer: invManufacturer.String,
				Quantity:     int(invQuantity.Int32),
				Condition:    models.ItemCondition(invCondition.String),
				Notes:        invNotes.String,
				ImageURL:     invImageURL.String,
				Specs:        invSpecs,
			}
			if invPrice.Valid {
				price := invPrice.Float64
				c.InventoryItem.PurchasePrice = &price
			}
		}

		components = append(components, c)
	}

	return components, nil
}

// RemoveComponent removes a component from an aircraft
func (s *AircraftStore) RemoveComponent(ctx context.Context, aircraftID string, category models.ComponentCategory) error {
	query := `DELETE FROM aircraft_components WHERE aircraft_id = $1 AND category = $2`
	_, err := s.db.ExecContext(ctx, query, aircraftID, string(category))
	if err != nil {
		return fmt.Errorf("failed to remove component: %w", err)
	}
	return nil
}

// SetReceiverSettings sets or updates receiver settings for an aircraft.
// SECURITY: Sensitive fields (BindPhrase, BindingPhrase, UID, WifiPassword) are encrypted before storage.
func (s *AircraftStore) SetReceiverSettings(ctx context.Context, aircraftID string, settings json.RawMessage) (*models.AircraftReceiverSettings, error) {
	// Encrypt sensitive fields before storing
	encryptedSettings, err := s.encryptReceiverSettings(settings)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt receiver settings: %w", err)
	}

	query := `
		INSERT INTO aircraft_receiver_settings (aircraft_id, settings_json)
		VALUES ($1, $2)
		ON CONFLICT (aircraft_id) DO UPDATE SET
			settings_json = EXCLUDED.settings_json,
			updated_at = NOW()
		RETURNING id, aircraft_id, settings_json, created_at, updated_at
	`

	rx := &models.AircraftReceiverSettings{}
	err = s.db.QueryRowContext(ctx, query, aircraftID, encryptedSettings).Scan(
		&rx.ID, &rx.AircraftID, &rx.Settings, &rx.CreatedAt, &rx.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to set receiver settings: %w", err)
	}

	// Decrypt the settings before returning so the caller gets plaintext
	decryptedSettings, err := s.decryptReceiverSettings(rx.Settings)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt receiver settings: %w", err)
	}
	rx.Settings = decryptedSettings

	return rx, nil
}

// GetReceiverSettings retrieves receiver settings for an aircraft.
// SECURITY: Sensitive fields are decrypted after retrieval from storage.
func (s *AircraftStore) GetReceiverSettings(ctx context.Context, aircraftID string) (*models.AircraftReceiverSettings, error) {
	query := `
		SELECT id, aircraft_id, settings_json, created_at, updated_at
		FROM aircraft_receiver_settings
		WHERE aircraft_id = $1
	`

	rx := &models.AircraftReceiverSettings{}
	err := s.db.QueryRowContext(ctx, query, aircraftID).Scan(
		&rx.ID, &rx.AircraftID, &rx.Settings, &rx.CreatedAt, &rx.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get receiver settings: %w", err)
	}

	// Decrypt sensitive fields before returning
	decryptedSettings, err := s.decryptReceiverSettings(rx.Settings)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt receiver settings: %w", err)
	}
	rx.Settings = decryptedSettings

	return rx, nil
}

// encryptReceiverSettings encrypts sensitive fields in receiver settings JSON.
// Fields encrypted: BindPhrase, BindingPhrase, UID, WifiPassword
func (s *AircraftStore) encryptReceiverSettings(settings json.RawMessage) (json.RawMessage, error) {
	if s.encryptor == nil || len(settings) == 0 {
		return settings, nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal(settings, &data); err != nil {
		return settings, nil // Return as-is if not valid JSON
	}

	// Encrypt sensitive string fields
	sensitiveFields := []string{"bindPhrase", "bindingPhrase", "uid", "wifiPassword"}
	for _, field := range sensitiveFields {
		if val, ok := data[field].(string); ok && val != "" {
			encrypted, err := s.encryptor.Encrypt(val)
			if err != nil {
				return nil, fmt.Errorf("failed to encrypt %s: %w", field, err)
			}
			data[field] = encrypted
		}
	}

	return json.Marshal(data)
}

// decryptReceiverSettings decrypts sensitive fields in receiver settings JSON.
func (s *AircraftStore) decryptReceiverSettings(settings json.RawMessage) (json.RawMessage, error) {
	if s.encryptor == nil || len(settings) == 0 {
		return settings, nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal(settings, &data); err != nil {
		return settings, nil // Return as-is if not valid JSON
	}

	// Decrypt sensitive string fields
	sensitiveFields := []string{"bindPhrase", "bindingPhrase", "uid", "wifiPassword"}
	for _, field := range sensitiveFields {
		if val, ok := data[field].(string); ok && val != "" {
			decrypted, err := s.encryptor.Decrypt(val)
			if err != nil {
				// If decryption fails, the data might be unencrypted (legacy) or corrupted
				// Log and continue with original value for backwards compatibility
				continue
			}
			data[field] = decrypted
		}
	}

	return json.Marshal(data)
}

// GetDetails retrieves full aircraft details including components and receiver settings
func (s *AircraftStore) GetDetails(ctx context.Context, id string, userID string) (*models.AircraftDetailsResponse, error) {
	aircraft, err := s.Get(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if aircraft == nil {
		return nil, nil
	}

	components, err := s.GetComponents(ctx, id)
	if err != nil {
		return nil, err
	}

	receiverSettings, err := s.GetReceiverSettings(ctx, id)
	if err != nil {
		return nil, err
	}

	return &models.AircraftDetailsResponse{
		Aircraft:         *aircraft,
		Components:       components,
		ReceiverSettings: receiverSettings,
	}, nil
}

// SetImage sets the image data for an aircraft
func (s *AircraftStore) SetImage(ctx context.Context, id string, userID string, imageType string, imageData []byte) error {
	query := `
		UPDATE aircraft SET image_data = $1, image_type = $2, updated_at = NOW()
		WHERE id = $3 AND user_id = $4
	`
	result, err := s.db.ExecContext(ctx, query, imageData, imageType, id, userID)
	if err != nil {
		return fmt.Errorf("failed to set aircraft image: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("aircraft not found")
	}
	return nil
}

// GetImage retrieves the image data for an aircraft
func (s *AircraftStore) GetImage(ctx context.Context, id string, userID string) ([]byte, string, error) {
	query := `
		SELECT image_data, image_type FROM aircraft
		WHERE id = $1 AND (user_id = $2 OR user_id IS NULL) AND image_data IS NOT NULL
	`
	var imageData []byte
	var imageType sql.NullString

	err := s.db.QueryRowContext(ctx, query, id, userID).Scan(&imageData, &imageType)
	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("failed to get aircraft image: %w", err)
	}

	return imageData, imageType.String, nil
}

// GetPublicImage retrieves image data for an aircraft if the owner allows it
// This is used for public pilot profiles - checks owner's social settings
func (s *AircraftStore) GetPublicImage(ctx context.Context, aircraftID string) ([]byte, string, error) {
	query := `
		SELECT a.image_data, a.image_type 
		FROM aircraft a
		JOIN users u ON a.user_id = u.id
		WHERE a.id = $1 
		  AND a.image_data IS NOT NULL
		  AND u.show_aircraft = true
		  AND u.profile_visibility = 'public'
	`
	var imageData []byte
	var imageType sql.NullString

	err := s.db.QueryRowContext(ctx, query, aircraftID).Scan(&imageData, &imageType)
	if err == sql.ErrNoRows {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("failed to get public aircraft image: %w", err)
	}

	return imageData, imageType.String, nil
}

// DeleteImage removes the image from an aircraft
func (s *AircraftStore) DeleteImage(ctx context.Context, id string, userID string) error {
	query := `
		UPDATE aircraft SET image_data = NULL, image_type = NULL, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
	`
	result, err := s.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete aircraft image: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("aircraft not found")
	}
	return nil
}

// Helper to join strings
func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

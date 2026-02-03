package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/johnrirwin/flyingforge/internal/models"
)

// OrderStore handles order database operations
type OrderStore struct {
	db *DB
}

// NewOrderStore creates a new order store
func NewOrderStore(db *DB) *OrderStore {
	return &OrderStore{db: db}
}

// Add creates a new order
func (s *OrderStore) Add(ctx context.Context, userID string, params models.AddOrderParams) (*models.Order, error) {
	query := `
		INSERT INTO orders (
			user_id, carrier, tracking_number, label, status
		) VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	order := &models.Order{
		UserID:         userID,
		Carrier:        params.Carrier,
		TrackingNumber: params.TrackingNumber,
		Label:          params.Label,
		Status:         models.StatusUnknown,
		Archived:       false,
	}

	err := s.db.QueryRowContext(ctx, query,
		nullString(userID), order.Carrier, order.TrackingNumber, nullString(order.Label), order.Status,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to add order: %w", err)
	}

	return order, nil
}

// GetByID retrieves a single order by ID
func (s *OrderStore) GetByID(ctx context.Context, id string) (*models.Order, error) {
	query := `
		SELECT id, user_id, carrier, tracking_number, label, status, status_details,
			   estimated_date, delivered_at, last_checked_at, archived, created_at, updated_at
		FROM orders
		WHERE id = $1
	`

	order := &models.Order{}
	var userID, label, statusDetails sql.NullString
	var estimatedDate, deliveredAt, lastCheckedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&order.ID, &userID, &order.Carrier, &order.TrackingNumber, &label,
		&order.Status, &statusDetails, &estimatedDate, &deliveredAt, &lastCheckedAt,
		&order.Archived, &order.CreatedAt, &order.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	order.UserID = userID.String
	order.Label = label.String
	order.StatusDetails = statusDetails.String
	if estimatedDate.Valid {
		order.EstimatedDate = &estimatedDate.Time
	}
	if deliveredAt.Valid {
		order.DeliveredAt = &deliveredAt.Time
	}
	if lastCheckedAt.Valid {
		order.LastCheckedAt = &lastCheckedAt.Time
	}

	return order, nil
}

// List retrieves all orders for a user
func (s *OrderStore) List(ctx context.Context, userID string, includeArchived bool, limit, offset int) (*models.OrderListResponse, error) {
	// Count query
	countQuery := `SELECT COUNT(*) FROM orders WHERE user_id = $1`
	args := []interface{}{userID}

	if !includeArchived {
		countQuery += ` AND archived = false`
	}

	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count orders: %w", err)
	}

	// List query - active first (non-delivered, non-archived), then by creation date
	listQuery := `
		SELECT id, user_id, carrier, tracking_number, label, status, status_details,
			   estimated_date, delivered_at, last_checked_at, archived, created_at, updated_at
		FROM orders
		WHERE user_id = $1
	`

	if !includeArchived {
		listQuery += ` AND archived = false`
	}

	listQuery += `
		ORDER BY 
			CASE WHEN status != 'delivered' AND archived = false THEN 0 ELSE 1 END,
			created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.QueryContext(ctx, listQuery, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list orders: %w", err)
	}
	defer rows.Close()

	orders := make([]models.Order, 0)
	for rows.Next() {
		var order models.Order
		var userIDNull, label, statusDetails sql.NullString
		var estimatedDate, deliveredAt, lastCheckedAt sql.NullTime

		err := rows.Scan(
			&order.ID, &userIDNull, &order.Carrier, &order.TrackingNumber, &label,
			&order.Status, &statusDetails, &estimatedDate, &deliveredAt, &lastCheckedAt,
			&order.Archived, &order.CreatedAt, &order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}

		order.UserID = userIDNull.String
		order.Label = label.String
		order.StatusDetails = statusDetails.String
		if estimatedDate.Valid {
			order.EstimatedDate = &estimatedDate.Time
		}
		if deliveredAt.Valid {
			order.DeliveredAt = &deliveredAt.Time
		}
		if lastCheckedAt.Valid {
			order.LastCheckedAt = &lastCheckedAt.Time
		}

		orders = append(orders, order)
	}

	return &models.OrderListResponse{
		Orders:     orders,
		TotalCount: totalCount,
	}, nil
}

// Update updates an existing order
func (s *OrderStore) Update(ctx context.Context, userID string, params models.UpdateOrderParams) (*models.Order, error) {
	// Build dynamic update query
	setClauses := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIndex := 1

	if params.Carrier != nil {
		setClauses = append(setClauses, fmt.Sprintf("carrier = $%d", argIndex))
		args = append(args, *params.Carrier)
		argIndex++
	}
	if params.TrackingNumber != nil {
		setClauses = append(setClauses, fmt.Sprintf("tracking_number = $%d", argIndex))
		args = append(args, *params.TrackingNumber)
		argIndex++
	}
	if params.Label != nil {
		setClauses = append(setClauses, fmt.Sprintf("label = $%d", argIndex))
		args = append(args, nullString(*params.Label))
		argIndex++
	}
	if params.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *params.Status)
		argIndex++
	}
	if params.StatusDetails != nil {
		setClauses = append(setClauses, fmt.Sprintf("status_details = $%d", argIndex))
		args = append(args, nullString(*params.StatusDetails))
		argIndex++
	}
	if params.EstimatedDate != nil {
		setClauses = append(setClauses, fmt.Sprintf("estimated_date = $%d", argIndex))
		args = append(args, params.EstimatedDate)
		argIndex++
	}
	if params.DeliveredAt != nil {
		setClauses = append(setClauses, fmt.Sprintf("delivered_at = $%d", argIndex))
		args = append(args, params.DeliveredAt)
		argIndex++
	}
	if params.Archived != nil {
		setClauses = append(setClauses, fmt.Sprintf("archived = $%d", argIndex))
		args = append(args, *params.Archived)
		argIndex++
	}

	// Add ID and user_id to args
	args = append(args, params.ID, userID)

	query := fmt.Sprintf(`
		UPDATE orders SET %s
		WHERE id = $%d AND user_id = $%d
		RETURNING id, user_id, carrier, tracking_number, label, status, status_details,
				  estimated_date, delivered_at, last_checked_at, archived, created_at, updated_at
	`, joinStrings(setClauses, ", "), argIndex, argIndex+1)

	order := &models.Order{}
	var userIDNull, label, statusDetails sql.NullString
	var estimatedDate, deliveredAt, lastCheckedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&order.ID, &userIDNull, &order.Carrier, &order.TrackingNumber, &label,
		&order.Status, &statusDetails, &estimatedDate, &deliveredAt, &lastCheckedAt,
		&order.Archived, &order.CreatedAt, &order.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	order.UserID = userIDNull.String
	order.Label = label.String
	order.StatusDetails = statusDetails.String
	if estimatedDate.Valid {
		order.EstimatedDate = &estimatedDate.Time
	}
	if deliveredAt.Valid {
		order.DeliveredAt = &deliveredAt.Time
	}
	if lastCheckedAt.Valid {
		order.LastCheckedAt = &lastCheckedAt.Time
	}

	return order, nil
}

// UpdateStatus updates just the status fields (for background sync)
func (s *OrderStore) UpdateStatus(ctx context.Context, id string, status models.ShipmentStatus, statusDetails string, estimatedDate, deliveredAt *time.Time) error {
	query := `
		UPDATE orders SET
			status = $1,
			status_details = $2,
			estimated_date = $3,
			delivered_at = $4,
			last_checked_at = NOW(),
			updated_at = NOW()
		WHERE id = $5
	`

	_, err := s.db.ExecContext(ctx, query, status, nullString(statusDetails), estimatedDate, deliveredAt, id)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return nil
}

// Delete removes an order
func (s *OrderStore) Delete(ctx context.Context, id, userID string) error {
	query := `DELETE FROM orders WHERE id = $1 AND user_id = $2`

	result, err := s.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete order: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("order not found or access denied")
	}

	return nil
}

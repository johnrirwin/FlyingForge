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

type AnnouncementStore struct {
	db *DB
}

func NewAnnouncementStore(db *DB) *AnnouncementStore {
	return &AnnouncementStore{db: db}
}

func (s *AnnouncementStore) ListActive(ctx context.Context, placement models.AnnouncementPlacement, audiences []models.AnnouncementAudience, now time.Time) ([]models.Announcement, error) {
	if s == nil || s.db == nil {
		return []models.Announcement{}, nil
	}

	placementValues := []string{strings.ToLower(strings.TrimSpace(string(placement)))}
	if placement != models.AnnouncementPlacementGlobal {
		placementValues = append(placementValues, string(models.AnnouncementPlacementGlobal))
	}

	audienceValues := make([]string, 0, len(audiences))
	for _, audience := range audiences {
		trimmed := strings.ToLower(strings.TrimSpace(string(audience)))
		if trimmed == "" {
			continue
		}
		audienceValues = append(audienceValues, trimmed)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id, title, body, status, priority, placements, audience,
			cta_label, cta_url, dismissible, starts_at, ends_at,
			created_at, updated_at
		FROM announcements
		WHERE LOWER(status) = 'published'
		  AND LOWER(audience) = ANY($1)
		  AND EXISTS (
			SELECT 1
			FROM unnest(placements) AS placement_value
			WHERE LOWER(placement_value) = ANY($2)
		  )
		  AND (starts_at IS NULL OR starts_at <= $3)
		  AND (ends_at IS NULL OR ends_at >= $3)
		ORDER BY priority DESC, updated_at DESC, created_at DESC
	`, pq.Array(audienceValues), pq.Array(placementValues), now.UTC())
	if err != nil {
		return nil, fmt.Errorf("list active announcements: %w", err)
	}
	defer rows.Close()

	return scanAnnouncements(rows)
}

func (s *AnnouncementStore) ListAdmin(ctx context.Context, params models.AdminAnnouncementListParams) (*models.AnnouncementListResponse, error) {
	if s == nil || s.db == nil {
		return &models.AnnouncementListResponse{Announcements: []models.Announcement{}}, nil
	}

	whereParts := []string{"TRUE"}
	args := make([]interface{}, 0, 4)
	argPos := 1

	if params.Query != "" {
		whereParts = append(whereParts, fmt.Sprintf("(title ILIKE $%d OR body ILIKE $%d)", argPos, argPos))
		args = append(args, "%"+params.Query+"%")
		argPos++
	}
	if params.Status != "" {
		whereParts = append(whereParts, fmt.Sprintf("LOWER(status) = $%d", argPos))
		args = append(args, strings.ToLower(string(params.Status)))
		argPos++
	}

	whereSQL := strings.Join(whereParts, " AND ")

	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM announcements WHERE "+whereSQL, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count announcements: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT
			id, title, body, status, priority, placements, audience,
			cta_label, cta_url, dismissible, starts_at, ends_at,
			created_at, updated_at
		FROM announcements
		WHERE `+whereSQL+`
		ORDER BY priority DESC, updated_at DESC, created_at DESC
		LIMIT $`+fmt.Sprintf("%d", argPos)+` OFFSET $`+fmt.Sprintf("%d", argPos+1),
		append(args, params.Limit, params.Offset)...,
	)
	if err != nil {
		return nil, fmt.Errorf("list admin announcements: %w", err)
	}
	defer rows.Close()

	items, err := scanAnnouncements(rows)
	if err != nil {
		return nil, err
	}

	return &models.AnnouncementListResponse{
		Announcements: items,
		TotalCount:    total,
	}, nil
}

func (s *AnnouncementStore) GetByID(ctx context.Context, id string) (*models.Announcement, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}

	row := s.db.QueryRowContext(ctx, `
		SELECT
			id, title, body, status, priority, placements, audience,
			cta_label, cta_url, dismissible, starts_at, ends_at,
			created_at, updated_at
		FROM announcements
		WHERE id = $1
	`, id)

	item, err := scanAnnouncement(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get announcement: %w", err)
	}
	return item, nil
}

func (s *AnnouncementStore) Create(ctx context.Context, actorUserID string, params models.SaveAnnouncementParams) (*models.Announcement, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("announcement store unavailable")
	}

	row := s.db.QueryRowContext(ctx, `
		INSERT INTO announcements (
			title, body, status, priority, placements, audience,
			cta_label, cta_url, dismissible, starts_at, ends_at,
			created_by_user_id, updated_by_user_id, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			NULLIF($12, '')::uuid, NULLIF($12, '')::uuid, NOW(), NOW()
		)
		RETURNING
			id, title, body, status, priority, placements, audience,
			cta_label, cta_url, dismissible, starts_at, ends_at,
			created_at, updated_at
	`,
		params.Title,
		params.Body,
		string(params.Status),
		params.Priority,
		pq.Array(placementsToStrings(params.Placements)),
		string(params.Audience),
		nullString(params.CTALabel),
		nullString(params.CTAURL),
		params.Dismissible,
		params.StartsAt,
		params.EndsAt,
		actorUserID,
	)

	item, err := scanAnnouncement(row)
	if err != nil {
		return nil, fmt.Errorf("create announcement: %w", err)
	}
	return item, nil
}

func (s *AnnouncementStore) Update(ctx context.Context, id string, actorUserID string, params models.SaveAnnouncementParams) (*models.Announcement, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("announcement store unavailable")
	}

	row := s.db.QueryRowContext(ctx, `
		UPDATE announcements
		SET
			title = $2,
			body = $3,
			status = $4,
			priority = $5,
			placements = $6,
			audience = $7,
			cta_label = $8,
			cta_url = $9,
			dismissible = $10,
			starts_at = $11,
			ends_at = $12,
			updated_by_user_id = NULLIF($13, '')::uuid,
			updated_at = NOW()
		WHERE id = $1
		RETURNING
			id, title, body, status, priority, placements, audience,
			cta_label, cta_url, dismissible, starts_at, ends_at,
			created_at, updated_at
	`,
		id,
		params.Title,
		params.Body,
		string(params.Status),
		params.Priority,
		pq.Array(placementsToStrings(params.Placements)),
		string(params.Audience),
		nullString(params.CTALabel),
		nullString(params.CTAURL),
		params.Dismissible,
		params.StartsAt,
		params.EndsAt,
		actorUserID,
	)

	item, err := scanAnnouncement(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("update announcement: %w", err)
	}
	return item, nil
}

func (s *AnnouncementStore) Delete(ctx context.Context, id string) (bool, error) {
	if s == nil || s.db == nil {
		return false, fmt.Errorf("announcement store unavailable")
	}

	res, err := s.db.ExecContext(ctx, `DELETE FROM announcements WHERE id = $1`, id)
	if err != nil {
		return false, fmt.Errorf("delete announcement: %w", err)
	}
	rows, _ := res.RowsAffected()
	return rows > 0, nil
}

type announcementScanner interface {
	Scan(dest ...interface{}) error
}

func scanAnnouncements(rows *sql.Rows) ([]models.Announcement, error) {
	items := make([]models.Announcement, 0)
	for rows.Next() {
		item, err := scanAnnouncement(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate announcements: %w", err)
	}
	return items, nil
}

func scanAnnouncement(scanner announcementScanner) (*models.Announcement, error) {
	var item models.Announcement
	var placements pq.StringArray
	var status string
	var audience string
	var ctaLabel, ctaURL sql.NullString
	var startsAt, endsAt sql.NullTime

	if err := scanner.Scan(
		&item.ID,
		&item.Title,
		&item.Body,
		&status,
		&item.Priority,
		&placements,
		&audience,
		&ctaLabel,
		&ctaURL,
		&item.Dismissible,
		&startsAt,
		&endsAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}

	item.Status = models.NormalizeAnnouncementStatus(models.AnnouncementStatus(status))
	item.Audience = models.NormalizeAnnouncementAudience(models.AnnouncementAudience(audience))
	item.Placements = placementsFromStrings([]string(placements))
	if ctaLabel.Valid {
		item.CTALabel = ctaLabel.String
	}
	if ctaURL.Valid {
		item.CTAURL = ctaURL.String
	}
	if startsAt.Valid {
		t := startsAt.Time.UTC()
		item.StartsAt = &t
	}
	if endsAt.Valid {
		t := endsAt.Time.UTC()
		item.EndsAt = &t
	}

	return &item, nil
}

func placementsToStrings(placements []models.AnnouncementPlacement) []string {
	if len(placements) == 0 {
		return []string{}
	}
	values := make([]string, 0, len(placements))
	for _, placement := range placements {
		values = append(values, string(placement))
	}
	return values
}

func placementsFromStrings(placements []string) []models.AnnouncementPlacement {
	if len(placements) == 0 {
		return []models.AnnouncementPlacement{}
	}
	values := make([]models.AnnouncementPlacement, 0, len(placements))
	for _, placement := range placements {
		normalized := models.NormalizeAnnouncementPlacement(models.AnnouncementPlacement(placement))
		if !models.IsValidAnnouncementPlacement(normalized) {
			continue
		}
		values = append(values, normalized)
	}
	if values == nil {
		return []models.AnnouncementPlacement{}
	}
	return values
}

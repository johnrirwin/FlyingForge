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

// FeedItemStore persists aggregated feed items in Postgres.
type FeedItemStore struct {
	db *DB
}

func NewFeedItemStore(db *DB) *FeedItemStore {
	return &FeedItemStore{db: db}
}

func (s *FeedItemStore) UpsertItems(ctx context.Context, items []models.FeedItem) error {
	if len(items) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO feed_items (
			id, title, url, source, source_type,
			author, summary, content,
			published_at, fetched_at,
			thumbnail, tags,
			upvotes, comments,
			media_type, media_image_url, media_video_url, media_duration,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8,
			$9, $10,
			$11, $12,
			$13, $14,
			$15, $16, $17, $18,
			NOW(), NOW()
		)
		-- The primary key is id, but we also enforce uniqueness on (lower(url), lower(source)).
		-- Use that unique index as the conflict target so we don't fail if ID generation ever changes.
		ON CONFLICT ((LOWER(url)), (LOWER(source))) DO UPDATE SET
			title = EXCLUDED.title,
			url = EXCLUDED.url,
			source = EXCLUDED.source,
			source_type = EXCLUDED.source_type,
			author = EXCLUDED.author,
			summary = EXCLUDED.summary,
			content = EXCLUDED.content,
			published_at = EXCLUDED.published_at,
			fetched_at = EXCLUDED.fetched_at,
			thumbnail = EXCLUDED.thumbnail,
			tags = EXCLUDED.tags,
			upvotes = EXCLUDED.upvotes,
			comments = EXCLUDED.comments,
			media_type = EXCLUDED.media_type,
			media_image_url = EXCLUDED.media_image_url,
			media_video_url = EXCLUDED.media_video_url,
			media_duration = EXCLUDED.media_duration,
			updated_at = NOW()
	`)
	if err != nil {
		return fmt.Errorf("prepare upsert: %w", err)
	}
	defer stmt.Close()

	for _, item := range items {
		var upvotes, comments sql.NullInt64
		if item.Engagement != nil {
			upvotes = sql.NullInt64{Int64: int64(item.Engagement.Upvotes), Valid: true}
			comments = sql.NullInt64{Int64: int64(item.Engagement.Comments), Valid: true}
		}

		var mediaType, mediaImageURL, mediaVideoURL, mediaDuration sql.NullString
		if item.Media != nil {
			mediaType = nullString(item.Media.Type)
			mediaImageURL = nullString(item.Media.ImageUrl)
			mediaVideoURL = nullString(item.Media.VideoUrl)
			mediaDuration = nullString(item.Media.Duration)
		}

		tags := item.Tags
		if tags == nil {
			tags = []string{}
		}

		if _, err := stmt.ExecContext(ctx,
			item.ID,
			item.Title,
			item.URL,
			item.Source,
			item.SourceType,
			nullString(item.Author),
			nullString(item.Summary),
			nullString(item.Content),
			item.PublishedAt,
			item.FetchedAt,
			nullString(item.Thumbnail),
			pq.Array(tags),
			upvotes,
			comments,
			mediaType,
			mediaImageURL,
			mediaVideoURL,
			mediaDuration,
		); err != nil {
			return fmt.Errorf("upsert feed item %s: %w", item.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}

func (s *FeedItemStore) DeleteItemsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM feed_items WHERE published_at < $1`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete old feed items: %w", err)
	}
	rows, _ := res.RowsAffected()
	return rows, nil
}

// QueryItems returns items + total matching count (before limit/offset).
// resolvedSources should contain normalized source names (lowercased) that map
// to FeedItem.Source values, not SourceInfo IDs.
func (s *FeedItemStore) QueryItems(ctx context.Context, params models.FilterParams, resolvedSources []string) ([]models.FeedItem, int, error) {
	whereParts := []string{"TRUE"}
	args := make([]interface{}, 0)
	argPos := 1

	// Filter by sources (case-insensitive): LOWER(source) = ANY($n)
	if len(resolvedSources) > 0 {
		lowered := make([]string, 0, len(resolvedSources))
		for _, src := range resolvedSources {
			trimmed := strings.TrimSpace(src)
			if trimmed == "" {
				continue
			}
			lowered = append(lowered, strings.ToLower(trimmed))
		}
		if len(lowered) > 0 {
			whereParts = append(whereParts, fmt.Sprintf("LOWER(source) = ANY($%d)", argPos))
			args = append(args, pq.Array(lowered))
			argPos++
		}
	}

	// Filter by source type (maps UI groupings to feed item types).
	if strings.TrimSpace(params.SourceType) != "" {
		st := strings.ToLower(strings.TrimSpace(params.SourceType))
		allowed := []string{st}
		switch st {
		case "community":
			allowed = []string{"reddit", "forum"}
		case "news":
			allowed = []string{"rss"}
		}

		whereParts = append(whereParts, fmt.Sprintf("LOWER(source_type) = ANY($%d)", argPos))
		args = append(args, pq.Array(allowed))
		argPos++
	}

	// Filter by tag (case-insensitive).
	if strings.TrimSpace(params.Tag) != "" {
		whereParts = append(whereParts, fmt.Sprintf("EXISTS (SELECT 1 FROM unnest(tags) t WHERE LOWER(t) = LOWER($%d))", argPos))
		args = append(args, strings.TrimSpace(params.Tag))
		argPos++
	}

	// Filter by query (case-insensitive search across multiple fields).
	if strings.TrimSpace(params.Query) != "" {
		placeholder := fmt.Sprintf("$%d", argPos)
		whereParts = append(whereParts, fmt.Sprintf("(title ILIKE %s OR summary ILIKE %s OR content ILIKE %s OR source ILIKE %s)", placeholder, placeholder, placeholder, placeholder))
		args = append(args, "%"+strings.TrimSpace(params.Query)+"%")
		argPos++
	}

	// Filter by date range.
	if fromTime, ok := models.ParseDateFilter(params.FromDate); ok {
		whereParts = append(whereParts, fmt.Sprintf("published_at >= $%d", argPos))
		args = append(args, fromTime)
		argPos++
	}
	if toTime, ok := models.ParseDateFilter(params.ToDate); ok {
		// End of day for inclusive filter.
		toTime = toTime.Add(24*time.Hour - time.Nanosecond)
		whereParts = append(whereParts, fmt.Sprintf("published_at <= $%d", argPos))
		args = append(args, toTime)
		argPos++
	}

	whereSQL := strings.Join(whereParts, " AND ")

	// Count query (no limit/offset).
	countQuery := "SELECT COUNT(*) FROM feed_items WHERE " + whereSQL
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count feed items: %w", err)
	}

	// Sort.
	orderSQL := "ORDER BY published_at DESC"
	switch strings.ToLower(strings.TrimSpace(params.Sort)) {
	case "score", "top":
		orderSQL = "ORDER BY (COALESCE(upvotes, 0) + COALESCE(comments, 0)) DESC, published_at DESC"
	}

	// Select query + pagination.
	selectQuery := `
		SELECT
			id, title, url, source, source_type,
			author, summary, content,
			published_at, fetched_at,
			thumbnail, tags,
			upvotes, comments,
			media_type, media_image_url, media_video_url, media_duration
		FROM feed_items
		WHERE ` + whereSQL + "\n\t\t" + orderSQL

	selectArgs := append([]interface{}{}, args...)
	if params.Limit > 0 {
		selectQuery += fmt.Sprintf("\n\t\tLIMIT $%d OFFSET $%d", argPos, argPos+1)
		selectArgs = append(selectArgs, params.Limit, params.Offset)
	}

	rows, err := s.db.QueryContext(ctx, selectQuery, selectArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query feed items: %w", err)
	}
	defer rows.Close()

	items := make([]models.FeedItem, 0)
	for rows.Next() {
		var item models.FeedItem
		var author, summary, content, thumbnail sql.NullString
		var tags pq.StringArray
		var upvotes, comments sql.NullInt64
		var mediaType, mediaImageURL, mediaVideoURL, mediaDuration sql.NullString

		if err := rows.Scan(
			&item.ID,
			&item.Title,
			&item.URL,
			&item.Source,
			&item.SourceType,
			&author,
			&summary,
			&content,
			&item.PublishedAt,
			&item.FetchedAt,
			&thumbnail,
			&tags,
			&upvotes,
			&comments,
			&mediaType,
			&mediaImageURL,
			&mediaVideoURL,
			&mediaDuration,
		); err != nil {
			return nil, 0, fmt.Errorf("scan feed item: %w", err)
		}

		if author.Valid {
			item.Author = author.String
		}
		if summary.Valid {
			item.Summary = summary.String
		}
		if content.Valid {
			item.Content = content.String
		}
		if thumbnail.Valid {
			item.Thumbnail = thumbnail.String
		}

		item.Tags = []string(tags)
		if item.Tags == nil {
			item.Tags = []string{}
		}

		if upvotes.Valid || comments.Valid {
			item.Engagement = &models.Engagement{
				Upvotes:  int(upvotes.Int64),
				Comments: int(comments.Int64),
			}
		}

		if mediaType.Valid || mediaImageURL.Valid || mediaVideoURL.Valid || mediaDuration.Valid {
			item.Media = &models.MediaInfo{
				Type:     mediaType.String,
				ImageUrl: mediaImageURL.String,
				VideoUrl: mediaVideoURL.String,
				Duration: mediaDuration.String,
			}
		}

		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate feed items: %w", err)
	}

	return items, total, nil
}

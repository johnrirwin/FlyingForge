package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/johnrirwin/flyingforge/internal/models"
)

// UserStore handles user database operations
type UserStore struct {
	db *DB
}

// NewUserStore creates a new user store
func NewUserStore(db *DB) *UserStore {
	return &UserStore{db: db}
}

// Create creates a new user
func (s *UserStore) Create(ctx context.Context, params models.CreateUserParams) (*models.User, error) {
	email := strings.ToLower(strings.TrimSpace(params.Email))
	status := params.Status
	if status == "" {
		status = models.UserStatusActive
	}

	query := `
		INSERT INTO users (email, display_name, call_sign, avatar_url, status, google_name, google_avatar_url, avatar_type)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, email, display_name, avatar_url, status, created_at, updated_at, last_login_at,
		          call_sign, google_name, google_avatar_url, avatar_type, custom_avatar_url
	`

	// Default avatar type to google
	avatarType := models.AvatarTypeGoogle

	user := &models.User{}
	var avatarURL, callSign, googleName, googleAvatarURL, customAvatarURL, avatarTypeStr sql.NullString
	var lastLoginAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query,
		email, params.DisplayName, nullString(params.CallSign), nullString(params.AvatarURL), status,
		nullString(params.GoogleName), nullString(params.GoogleAvatarURL), string(avatarType),
	).Scan(
		&user.ID, &user.Email, &user.DisplayName, &avatarURL,
		&user.Status, &user.CreatedAt, &user.UpdatedAt, &lastLoginAt,
		&callSign, &googleName, &googleAvatarURL, &avatarTypeStr, &customAvatarURL,
	)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, fmt.Errorf("user with email %s already exists", email)
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}
	if callSign.Valid {
		user.CallSign = callSign.String
	}
	if googleName.Valid {
		user.GoogleName = googleName.String
	}
	if googleAvatarURL.Valid {
		user.GoogleAvatarURL = googleAvatarURL.String
	}
	if avatarTypeStr.Valid {
		user.AvatarType = models.AvatarType(avatarTypeStr.String)
	}
	if customAvatarURL.Valid {
		user.CustomAvatarURL = customAvatarURL.String
	}

	return user, nil
}

// GetByID retrieves a user by ID
func (s *UserStore) GetByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT id, email, display_name, avatar_url, status, created_at, updated_at, last_login_at,
		       call_sign, google_name, google_avatar_url, avatar_type, custom_avatar_url,
		       profile_visibility, show_aircraft, allow_search
		FROM users
		WHERE id = $1
	`

	return s.scanUser(s.db.QueryRowContext(ctx, query, id))
}

// GetByEmail retrieves a user by email (case-insensitive)
func (s *UserStore) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	query := `
		SELECT id, email, display_name, avatar_url, status, created_at, updated_at, last_login_at,
		       call_sign, google_name, google_avatar_url, avatar_type, custom_avatar_url,
		       profile_visibility, show_aircraft, allow_search
		FROM users
		WHERE LOWER(email) = $1
	`

	return s.scanUser(s.db.QueryRowContext(ctx, query, email))
}

// GetByCallSign retrieves a user by callsign (case-insensitive)
func (s *UserStore) GetByCallSign(ctx context.Context, callSign string) (*models.User, error) {
	callSign = strings.ToLower(strings.TrimSpace(callSign))
	query := `
		SELECT id, email, display_name, avatar_url, status, created_at, updated_at, last_login_at,
		       call_sign, google_name, google_avatar_url, avatar_type, custom_avatar_url,
		       profile_visibility, show_aircraft, allow_search
		FROM users
		WHERE LOWER(call_sign) = $1
	`

	return s.scanUser(s.db.QueryRowContext(ctx, query, callSign))
}

// Update updates a user
func (s *UserStore) Update(ctx context.Context, id string, params models.UpdateUserParams) (*models.User, error) {
	var sets []string
	var args []interface{}
	argIdx := 1

	if params.DisplayName != nil {
		sets = append(sets, fmt.Sprintf("display_name = $%d", argIdx))
		args = append(args, *params.DisplayName)
		argIdx++
	}
	if params.AvatarURL != nil {
		sets = append(sets, fmt.Sprintf("avatar_url = $%d", argIdx))
		args = append(args, *params.AvatarURL)
		argIdx++
	}
	if params.Status != nil {
		sets = append(sets, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *params.Status)
		argIdx++
	}
	if params.CallSign != nil {
		sets = append(sets, fmt.Sprintf("call_sign = $%d", argIdx))
		args = append(args, *params.CallSign)
		argIdx++
	}
	if params.GoogleName != nil {
		sets = append(sets, fmt.Sprintf("google_name = $%d", argIdx))
		args = append(args, *params.GoogleName)
		argIdx++
	}
	if params.GoogleAvatarURL != nil {
		sets = append(sets, fmt.Sprintf("google_avatar_url = $%d", argIdx))
		args = append(args, *params.GoogleAvatarURL)
		argIdx++
	}
	if params.AvatarType != nil {
		sets = append(sets, fmt.Sprintf("avatar_type = $%d", argIdx))
		args = append(args, string(*params.AvatarType))
		argIdx++
	}
	if params.CustomAvatarURL != nil {
		sets = append(sets, fmt.Sprintf("custom_avatar_url = $%d", argIdx))
		args = append(args, *params.CustomAvatarURL)
		argIdx++
	}

	if len(sets) == 0 {
		return s.GetByID(ctx, id)
	}

	sets = append(sets, "updated_at = NOW()")
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE users SET %s
		WHERE id = $%d
		RETURNING id, email, display_name, avatar_url, status, created_at, updated_at, last_login_at,
		          call_sign, google_name, google_avatar_url, avatar_type, custom_avatar_url,
		          profile_visibility, show_aircraft, allow_search
	`, strings.Join(sets, ", "), argIdx)

	return s.scanUser(s.db.QueryRowContext(ctx, query, args...))
}

// UpdateLastLogin updates the last login timestamp
func (s *UserStore) UpdateLastLogin(ctx context.Context, id string) error {
	query := `UPDATE users SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// Delete soft-deletes a user by setting status to disabled
func (s *UserStore) Delete(ctx context.Context, id string) error {
	query := `UPDATE users SET status = 'disabled', updated_at = NOW() WHERE id = $1`
	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// SearchPilots searches for pilots by callsign or display name
// Only returns users who have a callsign set and allow_search = true
func (s *UserStore) SearchPilots(ctx context.Context, params models.PilotSearchParams) ([]models.PilotSearchResult, error) {
	query := params.Query
	if query == "" {
		return []models.PilotSearchResult{}, nil
	}

	limit := params.Limit
	if limit <= 0 || limit > 50 {
		limit = 50
	}

	searchTerm := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"

	// Search by callsign or display name (if set)
	// Require callsign to be set for social visibility
	sqlQuery := `
		SELECT id, call_sign, display_name, google_name, avatar_url, google_avatar_url, avatar_type, custom_avatar_url
		FROM users
		WHERE status = 'active' 
		  AND call_sign IS NOT NULL 
		  AND call_sign != ''
		  AND (allow_search IS NULL OR allow_search = true)
		  AND (LOWER(call_sign) LIKE $1 OR LOWER(display_name) LIKE $1)
		ORDER BY call_sign
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, sqlQuery, searchTerm, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.PilotSearchResult
	for rows.Next() {
		var callSign, displayName, googleName, avatarURL, googleAvatarURL, avatarType, customAvatarURL sql.NullString
		var id string

		if err := rows.Scan(&id, &callSign, &displayName, &googleName, &avatarURL, &googleAvatarURL, &avatarType, &customAvatarURL); err != nil {
			return nil, err
		}

		// Compute effective avatar URL
		effectiveAvatarURL := ""
		if avatarType.Valid && avatarType.String == string(models.AvatarTypeCustom) && customAvatarURL.Valid {
			effectiveAvatarURL = customAvatarURL.String
		} else if googleAvatarURL.Valid {
			effectiveAvatarURL = googleAvatarURL.String
		} else if avatarURL.Valid {
			effectiveAvatarURL = avatarURL.String
		}

		results = append(results, models.PilotSearchResult{
			ID:                 id,
			CallSign:           callSign.String,
			DisplayName:        displayName.String,
			GoogleName:         googleName.String,
			EffectiveAvatarURL: effectiveAvatarURL,
		})
	}

	if results == nil {
		results = []models.PilotSearchResult{}
	}

	return results, nil
}

// List retrieves users with optional filtering
func (s *UserStore) List(ctx context.Context, params models.UserFilterParams) (*models.UsersResponse, error) {
	var where []string
	var args []interface{}
	argIdx := 1

	if params.Status != "" {
		where = append(where, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, params.Status)
		argIdx++
	}

	if params.Query != "" {
		where = append(where, fmt.Sprintf("(LOWER(email) LIKE $%d OR LOWER(display_name) LIKE $%d)", argIdx, argIdx))
		args = append(args, "%"+strings.ToLower(params.Query)+"%")
		argIdx++
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users %s", whereClause)
	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, err
	}

	// Get users
	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	args = append(args, limit, offset)
	query := fmt.Sprintf(`
		SELECT id, email, display_name, avatar_url, status, created_at, updated_at, last_login_at,
		       call_sign, google_name, google_avatar_url, avatar_type, custom_avatar_url,
		       profile_visibility, show_aircraft, allow_search
		FROM users %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		user, err := s.scanUserFromRows(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *user)
	}

	if users == nil {
		users = []models.User{}
	}

	return &models.UsersResponse{
		Users:      users,
		TotalCount: totalCount,
	}, nil
}

func (s *UserStore) scanUser(row *sql.Row) (*models.User, error) {
	user := &models.User{}
	var avatarURL, callSign, googleName, googleAvatarURL, avatarType, customAvatarURL sql.NullString
	var profileVisibility sql.NullString
	var showAircraft, allowSearch sql.NullBool
	var lastLoginAt sql.NullTime

	err := row.Scan(
		&user.ID, &user.Email, &user.DisplayName, &avatarURL,
		&user.Status, &user.CreatedAt, &user.UpdatedAt, &lastLoginAt,
		&callSign, &googleName, &googleAvatarURL, &avatarType, &customAvatarURL,
		&profileVisibility, &showAircraft, &allowSearch,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}
	if callSign.Valid {
		user.CallSign = callSign.String
	}
	if googleName.Valid {
		user.GoogleName = googleName.String
	}
	if googleAvatarURL.Valid {
		user.GoogleAvatarURL = googleAvatarURL.String
	}
	if avatarType.Valid {
		user.AvatarType = models.AvatarType(avatarType.String)
	}
	if customAvatarURL.Valid {
		user.CustomAvatarURL = customAvatarURL.String
	}

	// Set social settings with defaults
	user.SocialSettings = models.DefaultSocialSettings()
	if profileVisibility.Valid {
		user.SocialSettings.ProfileVisibility = models.ProfileVisibility(profileVisibility.String)
	}
	if showAircraft.Valid {
		user.SocialSettings.ShowAircraft = showAircraft.Bool
	}
	if allowSearch.Valid {
		user.SocialSettings.AllowSearch = allowSearch.Bool
	}

	return user, nil
}

func (s *UserStore) scanUserFromRows(rows *sql.Rows) (*models.User, error) {
	user := &models.User{}
	var avatarURL, callSign, googleName, googleAvatarURL, avatarType, customAvatarURL sql.NullString
	var profileVisibility sql.NullString
	var showAircraft, allowSearch sql.NullBool
	var lastLoginAt sql.NullTime

	err := rows.Scan(
		&user.ID, &user.Email, &user.DisplayName, &avatarURL,
		&user.Status, &user.CreatedAt, &user.UpdatedAt, &lastLoginAt,
		&callSign, &googleName, &googleAvatarURL, &avatarType, &customAvatarURL,
		&profileVisibility, &showAircraft, &allowSearch,
	)

	if err != nil {
		return nil, err
	}

	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}
	if callSign.Valid {
		user.CallSign = callSign.String
	}
	if googleName.Valid {
		user.GoogleName = googleName.String
	}
	if googleAvatarURL.Valid {
		user.GoogleAvatarURL = googleAvatarURL.String
	}
	if avatarType.Valid {
		user.AvatarType = models.AvatarType(avatarType.String)
	}
	if customAvatarURL.Valid {
		user.CustomAvatarURL = customAvatarURL.String
	}

	// Set social settings with defaults
	user.SocialSettings = models.DefaultSocialSettings()
	if profileVisibility.Valid {
		user.SocialSettings.ProfileVisibility = models.ProfileVisibility(profileVisibility.String)
	}
	if showAircraft.Valid {
		user.SocialSettings.ShowAircraft = showAircraft.Bool
	}
	if allowSearch.Valid {
		user.SocialSettings.AllowSearch = allowSearch.Bool
	}

	return user, nil
}

// Identity operations

// CreateIdentity creates a new user identity
func (s *UserStore) CreateIdentity(ctx context.Context, userID string, provider models.AuthProvider, subject, email string) (*models.UserIdentity, error) {
	query := `
		INSERT INTO user_identities (user_id, provider, provider_subject, provider_email)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, provider, provider_subject, provider_email, created_at
	`

	identity := &models.UserIdentity{}
	var providerEmail sql.NullString

	err := s.db.QueryRowContext(ctx, query, userID, provider, subject, nullString(email)).Scan(
		&identity.ID, &identity.UserID, &identity.Provider, &identity.ProviderSubject,
		&providerEmail, &identity.CreatedAt,
	)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, fmt.Errorf("identity already linked to another account")
		}
		return nil, err
	}

	if providerEmail.Valid {
		identity.ProviderEmail = providerEmail.String
	}

	return identity, nil
}

// GetIdentityByProvider retrieves an identity by provider and subject
func (s *UserStore) GetIdentityByProvider(ctx context.Context, provider models.AuthProvider, subject string) (*models.UserIdentity, error) {
	query := `
		SELECT id, user_id, provider, provider_subject, provider_email, created_at
		FROM user_identities
		WHERE provider = $1 AND provider_subject = $2
	`

	identity := &models.UserIdentity{}
	var providerEmail sql.NullString

	err := s.db.QueryRowContext(ctx, query, provider, subject).Scan(
		&identity.ID, &identity.UserID, &identity.Provider, &identity.ProviderSubject,
		&providerEmail, &identity.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if providerEmail.Valid {
		identity.ProviderEmail = providerEmail.String
	}

	return identity, nil
}

// GetIdentitiesByUserID retrieves all identities for a user
func (s *UserStore) GetIdentitiesByUserID(ctx context.Context, userID string) ([]models.UserIdentity, error) {
	query := `
		SELECT id, user_id, provider, provider_subject, provider_email, created_at
		FROM user_identities
		WHERE user_id = $1
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var identities []models.UserIdentity
	for rows.Next() {
		var identity models.UserIdentity
		var providerEmail sql.NullString

		err := rows.Scan(
			&identity.ID, &identity.UserID, &identity.Provider, &identity.ProviderSubject,
			&providerEmail, &identity.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if providerEmail.Valid {
			identity.ProviderEmail = providerEmail.String
		}

		identities = append(identities, identity)
	}

	return identities, nil
}

// Refresh token operations

// CreateRefreshToken stores a new refresh token
func (s *UserStore) CreateRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*models.RefreshToken, error) {
	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, token_hash, expires_at, created_at, revoked_at
	`

	token := &models.RefreshToken{}
	var revokedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, userID, tokenHash, expiresAt).Scan(
		&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt,
		&token.CreatedAt, &revokedAt,
	)

	if err != nil {
		return nil, err
	}

	if revokedAt.Valid {
		token.RevokedAt = &revokedAt.Time
	}

	return token, nil
}

// GetRefreshTokenByHash retrieves a refresh token by its hash
func (s *UserStore) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, created_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`

	token := &models.RefreshToken{}
	var revokedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&token.ID, &token.UserID, &token.TokenHash, &token.ExpiresAt,
		&token.CreatedAt, &revokedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if revokedAt.Valid {
		token.RevokedAt = &revokedAt.Time
	}

	return token, nil
}

// RevokeRefreshToken revokes a refresh token
func (s *UserStore) RevokeRefreshToken(ctx context.Context, tokenID string) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, tokenID)
	return err
}

// RevokeAllUserRefreshTokens revokes all refresh tokens for a user
func (s *UserStore) RevokeAllUserRefreshTokens(ctx context.Context, userID string) error {
	query := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL`
	_, err := s.db.ExecContext(ctx, query, userID)
	return err
}

// UpdateSocialSettings updates a user's social settings
func (s *UserStore) UpdateSocialSettings(ctx context.Context, userID string, params models.UpdateSocialSettingsParams) error {
	var sets []string
	var args []interface{}
	argIdx := 1

	if params.ProfileVisibility != nil {
		sets = append(sets, fmt.Sprintf("profile_visibility = $%d", argIdx))
		args = append(args, string(*params.ProfileVisibility))
		argIdx++
	}
	if params.ShowAircraft != nil {
		sets = append(sets, fmt.Sprintf("show_aircraft = $%d", argIdx))
		args = append(args, *params.ShowAircraft)
		argIdx++
	}
	if params.AllowSearch != nil {
		sets = append(sets, fmt.Sprintf("allow_search = $%d", argIdx))
		args = append(args, *params.AllowSearch)
		argIdx++
	}

	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = NOW()")
	args = append(args, userID)

	query := fmt.Sprintf(`
		UPDATE users SET %s
		WHERE id = $%d
	`, strings.Join(sets, ", "), argIdx)

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// Follow operations

// CreateFollow creates a follow relationship between two users
func (s *UserStore) CreateFollow(ctx context.Context, followerUserID, followedUserID string) (*models.Follow, error) {
	if followerUserID == followedUserID {
		return nil, fmt.Errorf("cannot follow yourself")
	}

	query := `
		INSERT INTO follows (follower_user_id, followed_user_id)
		VALUES ($1, $2)
		ON CONFLICT (follower_user_id, followed_user_id) DO NOTHING
		RETURNING id, follower_user_id, followed_user_id, created_at
	`

	follow := &models.Follow{}
	err := s.db.QueryRowContext(ctx, query, followerUserID, followedUserID).Scan(
		&follow.ID, &follow.FollowerUserID, &follow.FollowedUserID, &follow.CreatedAt,
	)

	if err == sql.ErrNoRows {
		// Already following, return existing
		return s.GetFollow(ctx, followerUserID, followedUserID)
	}
	if err != nil {
		return nil, err
	}

	return follow, nil
}

// DeleteFollow removes a follow relationship
func (s *UserStore) DeleteFollow(ctx context.Context, followerUserID, followedUserID string) error {
	query := `DELETE FROM follows WHERE follower_user_id = $1 AND followed_user_id = $2`
	_, err := s.db.ExecContext(ctx, query, followerUserID, followedUserID)
	return err
}

// DeleteAllFollowsForUser removes all follow relationships for a user (both as follower and followed)
func (s *UserStore) DeleteAllFollowsForUser(ctx context.Context, userID string) error {
	query := `DELETE FROM follows WHERE follower_user_id = $1 OR followed_user_id = $1`
	_, err := s.db.ExecContext(ctx, query, userID)
	return err
}

// GetFollow checks if a follow relationship exists
func (s *UserStore) GetFollow(ctx context.Context, followerUserID, followedUserID string) (*models.Follow, error) {
	query := `
		SELECT id, follower_user_id, followed_user_id, created_at
		FROM follows
		WHERE follower_user_id = $1 AND followed_user_id = $2
	`

	follow := &models.Follow{}
	err := s.db.QueryRowContext(ctx, query, followerUserID, followedUserID).Scan(
		&follow.ID, &follow.FollowerUserID, &follow.FollowedUserID, &follow.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return follow, nil
}

// IsFollowing checks if followerUserID follows followedUserID
func (s *UserStore) IsFollowing(ctx context.Context, followerUserID, followedUserID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM follows WHERE follower_user_id = $1 AND followed_user_id = $2)`
	var exists bool
	err := s.db.QueryRowContext(ctx, query, followerUserID, followedUserID).Scan(&exists)
	return exists, err
}

// GetFollowerCount returns the number of followers for a user
// Only counts users with callsigns set (for privacy consistency with GetFollowers)
func (s *UserStore) GetFollowerCount(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM follows f JOIN users follower ON follower.id = f.follower_user_id WHERE f.followed_user_id = $1 AND follower.call_sign IS NOT NULL AND follower.call_sign != ''`
	var count int
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&count)
	return count, err
}

// GetFollowingCount returns the number of users a user is following
// Only counts users with callsigns set (for privacy consistency with GetFollowing)
func (s *UserStore) GetFollowingCount(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM follows f JOIN users followed ON followed.id = f.followed_user_id WHERE f.follower_user_id = $1 AND followed.call_sign IS NOT NULL AND followed.call_sign != ''`
	var count int
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&count)
	return count, err
}

// GetFollowers returns the list of users following the given user
func (s *UserStore) GetFollowers(ctx context.Context, userID string, limit, offset int) (*models.FollowListResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// Get total count - only count users with callsigns
	countQuery := `SELECT COUNT(*) FROM follows f JOIN users follower ON follower.id = f.follower_user_id WHERE f.followed_user_id = $1 AND follower.call_sign IS NOT NULL AND follower.call_sign != ''`
	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, userID).Scan(&totalCount); err != nil {
		return nil, err
	}

	// Get follower user details - only users with callsigns for privacy
	query := `
		SELECT follower.id, follower.call_sign, follower.display_name, follower.avatar_url, follower.google_avatar_url, follower.avatar_type, follower.custom_avatar_url
		FROM follows f
		JOIN users follower ON follower.id = f.follower_user_id
		WHERE f.followed_user_id = $1
		  AND follower.call_sign IS NOT NULL AND follower.call_sign != ''
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pilots []models.PilotSummary
	for rows.Next() {
		var id string
		var callSign, displayName, avatarURL, googleAvatarURL, avatarType, customAvatarURL sql.NullString

		if err := rows.Scan(&id, &callSign, &displayName, &avatarURL, &googleAvatarURL, &avatarType, &customAvatarURL); err != nil {
			return nil, err
		}

		// Compute effective avatar URL
		effectiveAvatarURL := ""
		if avatarType.Valid && avatarType.String == string(models.AvatarTypeCustom) && customAvatarURL.Valid {
			effectiveAvatarURL = customAvatarURL.String
		} else if googleAvatarURL.Valid {
			effectiveAvatarURL = googleAvatarURL.String
		} else if avatarURL.Valid {
			effectiveAvatarURL = avatarURL.String
		}

		pilots = append(pilots, models.PilotSummary{
			ID:                 id,
			CallSign:           callSign.String,
			DisplayName:        displayName.String,
			EffectiveAvatarURL: effectiveAvatarURL,
		})
	}

	if pilots == nil {
		pilots = []models.PilotSummary{}
	}

	return &models.FollowListResponse{
		Pilots:     pilots,
		TotalCount: totalCount,
	}, nil
}

// GetFollowing returns the list of users the given user is following
func (s *UserStore) GetFollowing(ctx context.Context, userID string, limit, offset int) (*models.FollowListResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	// Get total count - only count users with callsigns
	countQuery := `SELECT COUNT(*) FROM follows f JOIN users followed ON followed.id = f.followed_user_id WHERE f.follower_user_id = $1 AND followed.call_sign IS NOT NULL AND followed.call_sign != ''`
	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, userID).Scan(&totalCount); err != nil {
		return nil, err
	}

	// Get following user details - only users with callsigns for privacy
	query := `
		SELECT followed.id, followed.call_sign, followed.display_name, followed.avatar_url, followed.google_avatar_url, followed.avatar_type, followed.custom_avatar_url
		FROM follows f
		JOIN users followed ON followed.id = f.followed_user_id
		WHERE f.follower_user_id = $1
		  AND followed.call_sign IS NOT NULL AND followed.call_sign != ''
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pilots []models.PilotSummary
	for rows.Next() {
		var id string
		var callSign, displayName, avatarURL, googleAvatarURL, avatarType, customAvatarURL sql.NullString

		if err := rows.Scan(&id, &callSign, &displayName, &avatarURL, &googleAvatarURL, &avatarType, &customAvatarURL); err != nil {
			return nil, err
		}

		// Compute effective avatar URL
		effectiveAvatarURL := ""
		if avatarType.Valid && avatarType.String == string(models.AvatarTypeCustom) && customAvatarURL.Valid {
			effectiveAvatarURL = customAvatarURL.String
		} else if googleAvatarURL.Valid {
			effectiveAvatarURL = googleAvatarURL.String
		} else if avatarURL.Valid {
			effectiveAvatarURL = avatarURL.String
		}

		pilots = append(pilots, models.PilotSummary{
			ID:                 id,
			CallSign:           callSign.String,
			DisplayName:        displayName.String,
			EffectiveAvatarURL: effectiveAvatarURL,
		})
	}

	if pilots == nil {
		pilots = []models.PilotSummary{}
	}

	return &models.FollowListResponse{
		Pilots:     pilots,
		TotalCount: totalCount,
	}, nil
}

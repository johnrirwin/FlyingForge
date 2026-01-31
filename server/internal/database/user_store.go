package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/johnrirwin/mcp-news-feed/internal/models"
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
		INSERT INTO users (email, password_hash, display_name, avatar_url, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, email, display_name, avatar_url, status, created_at, updated_at, last_login_at
	`

	var passwordHash sql.NullString
	if params.Password != "" {
		passwordHash = sql.NullString{String: params.Password, Valid: true}
	}

	user := &models.User{}
	var avatarURL sql.NullString
	var lastLoginAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query,
		email, passwordHash, params.DisplayName, nullString(params.AvatarURL), status,
	).Scan(
		&user.ID, &user.Email, &user.DisplayName, &avatarURL,
		&user.Status, &user.CreatedAt, &user.UpdatedAt, &lastLoginAt,
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

	return user, nil
}

// GetByID retrieves a user by ID
func (s *UserStore) GetByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, display_name, avatar_url, status, created_at, updated_at, last_login_at
		FROM users
		WHERE id = $1
	`

	return s.scanUser(s.db.QueryRowContext(ctx, query, id))
}

// GetByEmail retrieves a user by email (case-insensitive)
func (s *UserStore) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	query := `
		SELECT id, email, password_hash, display_name, avatar_url, status, created_at, updated_at, last_login_at
		FROM users
		WHERE LOWER(email) = $1
	`

	return s.scanUser(s.db.QueryRowContext(ctx, query, email))
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
	if params.Password != nil {
		sets = append(sets, fmt.Sprintf("password_hash = $%d", argIdx))
		args = append(args, *params.Password)
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
		RETURNING id, email, password_hash, display_name, avatar_url, status, created_at, updated_at, last_login_at
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
		SELECT id, email, password_hash, display_name, avatar_url, status, created_at, updated_at, last_login_at
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
	var passwordHash, avatarURL sql.NullString
	var lastLoginAt sql.NullTime

	err := row.Scan(
		&user.ID, &user.Email, &passwordHash, &user.DisplayName, &avatarURL,
		&user.Status, &user.CreatedAt, &user.UpdatedAt, &lastLoginAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	user.PasswordHash = passwordHash.String
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
	}

	return user, nil
}

func (s *UserStore) scanUserFromRows(rows *sql.Rows) (*models.User, error) {
	user := &models.User{}
	var passwordHash, avatarURL sql.NullString
	var lastLoginAt sql.NullTime

	err := rows.Scan(
		&user.ID, &user.Email, &passwordHash, &user.DisplayName, &avatarURL,
		&user.Status, &user.CreatedAt, &user.UpdatedAt, &lastLoginAt,
	)

	if err != nil {
		return nil, err
	}

	user.PasswordHash = passwordHash.String
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if lastLoginAt.Valid {
		user.LastLoginAt = &lastLoginAt.Time
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

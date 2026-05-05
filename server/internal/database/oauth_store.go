package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/johnrirwin/flyingforge/internal/models"
)

// OAuthStore manages self-hosted OAuth persistence.
type OAuthStore struct {
	db *DB
}

func NewOAuthStore(db *DB) *OAuthStore {
	return &OAuthStore{db: db}
}

func (s *OAuthStore) CreateClient(ctx context.Context, clientID, clientName string, redirectURIs, grantTypes, responseTypes []string, tokenEndpointAuthMethod, scope string) (*models.OAuthClient, error) {
	redirectsJSON, err := json.Marshal(redirectURIs)
	if err != nil {
		return nil, fmt.Errorf("marshal redirect uris: %w", err)
	}
	grantTypesJSON, err := json.Marshal(grantTypes)
	if err != nil {
		return nil, fmt.Errorf("marshal grant types: %w", err)
	}
	responseTypesJSON, err := json.Marshal(responseTypes)
	if err != nil {
		return nil, fmt.Errorf("marshal response types: %w", err)
	}

	const query = `
		INSERT INTO oauth_clients (
			client_id, client_name, redirect_uris, grant_types, response_types,
			token_endpoint_auth_method, scope
		)
		VALUES ($1, $2, $3::jsonb, $4::jsonb, $5::jsonb, $6, $7)
		RETURNING id, client_id, client_name, redirect_uris, grant_types, response_types,
			token_endpoint_auth_method, scope, created_at
	`

	row := s.db.QueryRowContext(ctx, query,
		clientID,
		nullString(clientName),
		string(redirectsJSON),
		string(grantTypesJSON),
		string(responseTypesJSON),
		tokenEndpointAuthMethod,
		nullString(scope),
	)

	return scanOAuthClient(row)
}

func (s *OAuthStore) GetClientByClientID(ctx context.Context, clientID string) (*models.OAuthClient, error) {
	const query = `
		SELECT id, client_id, client_name, redirect_uris, grant_types, response_types,
			token_endpoint_auth_method, scope, created_at
		FROM oauth_clients
		WHERE client_id = $1
	`

	row := s.db.QueryRowContext(ctx, query, clientID)
	client, err := scanOAuthClient(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (s *OAuthStore) CreateAuthorizationCode(ctx context.Context, codeHash, clientID, userID, redirectURI, scope, resource, codeChallenge, codeChallengeMethod string, expiresAt time.Time) (*models.OAuthAuthorizationCode, error) {
	const query = `
		INSERT INTO oauth_authorization_codes (
			code_hash, client_id, user_id, redirect_uri, scope, resource,
			code_challenge, code_challenge_method, expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, client_id, user_id, redirect_uri, scope, resource,
			code_challenge, code_challenge_method, expires_at, created_at, consumed_at
	`

	row := s.db.QueryRowContext(ctx, query,
		codeHash,
		clientID,
		userID,
		redirectURI,
		scope,
		nullString(resource),
		codeChallenge,
		codeChallengeMethod,
		expiresAt,
	)

	return scanOAuthAuthorizationCode(row)
}

func (s *OAuthStore) ConsumeAuthorizationCode(ctx context.Context, codeHash string) (*models.OAuthAuthorizationCode, error) {
	const query = `
		UPDATE oauth_authorization_codes
		SET consumed_at = NOW()
		WHERE code_hash = $1
		  AND consumed_at IS NULL
		  AND expires_at > NOW()
		RETURNING id, client_id, user_id, redirect_uri, scope, resource,
			code_challenge, code_challenge_method, expires_at, created_at, consumed_at
	`

	row := s.db.QueryRowContext(ctx, query, codeHash)
	code, err := scanOAuthAuthorizationCode(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return code, nil
}

func (s *OAuthStore) CreateRefreshToken(ctx context.Context, userID, clientID, tokenHash, scope, resource string, expiresAt time.Time) (*models.OAuthRefreshToken, error) {
	const query = `
		INSERT INTO oauth_refresh_tokens (
			user_id, client_id, token_hash, scope, resource, expires_at
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, client_id, scope, resource, expires_at, created_at, revoked_at
	`

	row := s.db.QueryRowContext(ctx, query,
		userID,
		clientID,
		tokenHash,
		scope,
		nullString(resource),
		expiresAt,
	)

	return scanOAuthRefreshToken(row)
}

func (s *OAuthStore) ConsumeRefreshToken(ctx context.Context, tokenHash string) (*models.OAuthRefreshToken, error) {
	const query = `
		UPDATE oauth_refresh_tokens
		SET revoked_at = NOW()
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
		RETURNING id, user_id, client_id, scope, resource, expires_at, created_at, revoked_at
	`

	row := s.db.QueryRowContext(ctx, query, tokenHash)
	refreshToken, err := scanOAuthRefreshToken(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return refreshToken, nil
}

func scanOAuthClient(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.OAuthClient, error) {
	var client models.OAuthClient
	var redirectURIsJSON string
	var grantTypesJSON string
	var responseTypesJSON string
	var clientName sql.NullString
	var scope sql.NullString

	err := scanner.Scan(
		&client.ID,
		&client.ClientID,
		&clientName,
		&redirectURIsJSON,
		&grantTypesJSON,
		&responseTypesJSON,
		&client.TokenEndpointAuthMethod,
		&scope,
		&client.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if clientName.Valid {
		client.ClientName = clientName.String
	}
	if scope.Valid {
		client.Scope = scope.String
	}

	if err := json.Unmarshal([]byte(redirectURIsJSON), &client.RedirectURIs); err != nil {
		return nil, fmt.Errorf("decode redirect uris: %w", err)
	}
	if err := json.Unmarshal([]byte(grantTypesJSON), &client.GrantTypes); err != nil {
		return nil, fmt.Errorf("decode grant types: %w", err)
	}
	if err := json.Unmarshal([]byte(responseTypesJSON), &client.ResponseTypes); err != nil {
		return nil, fmt.Errorf("decode response types: %w", err)
	}

	return &client, nil
}

func scanOAuthAuthorizationCode(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.OAuthAuthorizationCode, error) {
	var code models.OAuthAuthorizationCode
	var resource sql.NullString
	var consumedAt sql.NullTime

	err := scanner.Scan(
		&code.ID,
		&code.ClientID,
		&code.UserID,
		&code.RedirectURI,
		&code.Scope,
		&resource,
		&code.CodeChallenge,
		&code.CodeChallengeMethod,
		&code.ExpiresAt,
		&code.CreatedAt,
		&consumedAt,
	)
	if err != nil {
		return nil, err
	}

	if resource.Valid {
		code.Resource = resource.String
	}
	if consumedAt.Valid {
		code.ConsumedAt = &consumedAt.Time
	}

	return &code, nil
}

func scanOAuthRefreshToken(scanner interface {
	Scan(dest ...interface{}) error
}) (*models.OAuthRefreshToken, error) {
	var refreshToken models.OAuthRefreshToken
	var resource sql.NullString
	var revokedAt sql.NullTime

	err := scanner.Scan(
		&refreshToken.ID,
		&refreshToken.UserID,
		&refreshToken.ClientID,
		&refreshToken.Scope,
		&resource,
		&refreshToken.ExpiresAt,
		&refreshToken.CreatedAt,
		&revokedAt,
	)
	if err != nil {
		return nil, err
	}

	if resource.Valid {
		refreshToken.Resource = resource.String
	}
	if revokedAt.Valid {
		refreshToken.RevokedAt = &revokedAt.Time
	}

	return &refreshToken, nil
}

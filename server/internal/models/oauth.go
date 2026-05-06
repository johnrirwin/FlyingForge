package models

import "time"

const (
	OAuthTokenEndpointAuthMethodNone = "none"
	OAuthCodeChallengeMethodS256     = "S256"
	OAuthGrantTypeAuthorizationCode  = "authorization_code"
	OAuthGrantTypeRefreshToken       = "refresh_token"
	OAuthResponseTypeCode            = "code"
)

// OAuthClient stores dynamic client registration metadata for MCP clients.
type OAuthClient struct {
	ID                      string    `json:"id"`
	ClientID                string    `json:"clientId"`
	ClientName              string    `json:"clientName,omitempty"`
	RedirectURIs            []string  `json:"redirectUris"`
	GrantTypes              []string  `json:"grantTypes"`
	ResponseTypes           []string  `json:"responseTypes"`
	TokenEndpointAuthMethod string    `json:"tokenEndpointAuthMethod"`
	Scope                   string    `json:"scope,omitempty"`
	CreatedAt               time.Time `json:"createdAt"`
}

// OAuthAuthorizationCode stores a pending authorization code exchange.
type OAuthAuthorizationCode struct {
	ID                  string     `json:"id"`
	ClientID            string     `json:"clientId"`
	UserID              string     `json:"userId"`
	RedirectURI         string     `json:"redirectUri"`
	Scope               string     `json:"scope"`
	Resource            string     `json:"resource,omitempty"`
	CodeChallenge       string     `json:"codeChallenge"`
	CodeChallengeMethod string     `json:"codeChallengeMethod"`
	ExpiresAt           time.Time  `json:"expiresAt"`
	CreatedAt           time.Time  `json:"createdAt"`
	ConsumedAt          *time.Time `json:"consumedAt,omitempty"`
}

// OAuthRefreshToken stores refresh-token metadata for self-hosted OAuth grants.
type OAuthRefreshToken struct {
	ID        string     `json:"id"`
	UserID    string     `json:"userId"`
	ClientID  string     `json:"clientId"`
	Scope     string     `json:"scope"`
	Resource  string     `json:"resource,omitempty"`
	ExpiresAt time.Time  `json:"expiresAt"`
	CreatedAt time.Time  `json:"createdAt"`
	RevokedAt *time.Time `json:"revokedAt,omitempty"`
}

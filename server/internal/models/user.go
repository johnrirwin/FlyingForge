package models

import (
	"time"
)

// UserStatus represents the status of a user account
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
	UserStatusPending  UserStatus = "pending"
)

// AuthProvider represents an identity provider
type AuthProvider string

const (
	AuthProviderGoogle AuthProvider = "google"
	AuthProviderEmail  AuthProvider = "email"
)

// User represents a user in the system
type User struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"` // Never expose password hash
	DisplayName  string     `json:"displayName"`
	AvatarURL    string     `json:"avatarUrl,omitempty"`
	Status       UserStatus `json:"status"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	LastLoginAt  *time.Time `json:"lastLoginAt,omitempty"`
}

// UserIdentity represents a linked identity provider (Google, etc.)
type UserIdentity struct {
	ID              string       `json:"id"`
	UserID          string       `json:"userId"`
	Provider        AuthProvider `json:"provider"`
	ProviderSubject string       `json:"providerSubject"` // e.g., Google 'sub' claim
	ProviderEmail   string       `json:"providerEmail"`
	CreatedAt       time.Time    `json:"createdAt"`
}

// AuthTokens represents the tokens returned after authentication
type AuthTokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken,omitempty"`
	TokenType    string `json:"tokenType"`
	ExpiresIn    int    `json:"expiresIn"` // seconds
}

// AuthResponse represents the response after successful authentication
type AuthResponse struct {
	User      *User       `json:"user"`
	Tokens    *AuthTokens `json:"tokens"`
	IsNewUser bool        `json:"isNewUser,omitempty"`
	IsLinked  bool        `json:"isLinked,omitempty"`
}

// SignupParams represents email/password signup parameters
type SignupParams struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

// LoginParams represents email/password login parameters
type LoginParams struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// GoogleLoginParams represents Google OAuth login parameters
type GoogleLoginParams struct {
	// Either IDToken (from Google Identity Services) or Code (from auth code flow)
	IDToken     string `json:"idToken,omitempty"`
	Code        string `json:"code,omitempty"`
	RedirectURI string `json:"redirectUri,omitempty"`
}

// GoogleClaims represents the claims from a Google ID token
type GoogleClaims struct {
	Subject       string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
}

// CreateUserParams represents parameters for creating a user
type CreateUserParams struct {
	Email        string     `json:"email"`
	Password     string     `json:"password,omitempty"`
	DisplayName  string     `json:"displayName"`
	AvatarURL    string     `json:"avatarUrl,omitempty"`
	Status       UserStatus `json:"status,omitempty"`
}

// UpdateUserParams represents parameters for updating a user
type UpdateUserParams struct {
	DisplayName *string     `json:"displayName,omitempty"`
	AvatarURL   *string     `json:"avatarUrl,omitempty"`
	Status      *UserStatus `json:"status,omitempty"`
	Password    *string     `json:"password,omitempty"`
}

// UserFilterParams represents parameters for filtering users
type UserFilterParams struct {
	Query  string     `json:"query,omitempty"`
	Status UserStatus `json:"status,omitempty"`
	Limit  int        `json:"limit,omitempty"`
	Offset int        `json:"offset,omitempty"`
}

// UsersResponse represents a paginated list of users
type UsersResponse struct {
	Users      []User `json:"users"`
	TotalCount int    `json:"totalCount"`
}

// RefreshToken represents a stored refresh token
type RefreshToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	TokenHash string    `json:"-"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
	RevokedAt *time.Time `json:"revokedAt,omitempty"`
}

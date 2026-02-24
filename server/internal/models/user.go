package models

import (
	"regexp"
	"strings"
	"time"

	goaway "github.com/TwiN/go-away"
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
)

// AvatarType represents which avatar to use
type AvatarType string

const (
	AvatarTypeGoogle AvatarType = "google"
	AvatarTypeCustom AvatarType = "custom"
)

// ProfileVisibility represents who can see a user's profile
type ProfileVisibility string

const (
	ProfileVisibilityPublic  ProfileVisibility = "public"
	ProfileVisibilityPrivate ProfileVisibility = "private"
)

// SocialSettings contains user's social/privacy preferences
type SocialSettings struct {
	ProfileVisibility ProfileVisibility `json:"profileVisibility"` // public or private
	ShowAircraft      bool              `json:"showAircraft"`      // whether aircraft are visible to others
	AllowSearch       bool              `json:"allowSearch"`       // whether user appears in search
}

// DefaultSocialSettings returns the default social settings for new users
func DefaultSocialSettings() SocialSettings {
	return SocialSettings{
		ProfileVisibility: ProfileVisibilityPublic,
		ShowAircraft:      true,
		AllowSearch:       true,
	}
}

// User represents a user in the system
type User struct {
	ID             string     `json:"id"`
	Email          string     `json:"email"`
	DisplayName    string     `json:"displayName"`
	AvatarURL      string     `json:"avatarUrl,omitempty"` // Legacy field, kept for compatibility
	Status         UserStatus `json:"status"`
	IsAdmin        bool       `json:"isAdmin"`               // Full admin access (content moderation + user admin)
	IsContentAdmin bool       `json:"isContentAdmin"`        // Content moderation access only (unless also admin)
	IsGearAdmin    bool       `json:"isGearAdmin,omitempty"` // Deprecated alias kept for compatibility
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	LastLoginAt    *time.Time `json:"lastLoginAt,omitempty"`

	// Profile fields
	CallSign        string     `json:"callSign,omitempty"`
	GoogleName      string     `json:"googleName,omitempty"`
	GoogleAvatarURL string     `json:"googleAvatarUrl,omitempty"`
	AvatarType      AvatarType `json:"avatarType,omitempty"`
	CustomAvatarURL string     `json:"customAvatarUrl,omitempty"`
	AvatarImageID   string     `json:"avatarImageAssetId,omitempty"`

	// Social settings
	SocialSettings SocialSettings `json:"socialSettings"`
}

// EffectiveAvatarURL returns the avatar URL to use based on AvatarType
func (u *User) EffectiveAvatarURL() string {
	if u.AvatarType == AvatarTypeCustom && u.AvatarImageID != "" {
		return "/api/images/" + u.AvatarImageID
	}
	if u.AvatarType == AvatarTypeCustom && u.CustomAvatarURL != "" {
		return u.CustomAvatarURL
	}
	if u.GoogleAvatarURL != "" {
		return u.GoogleAvatarURL
	}
	return u.AvatarURL // Fallback to legacy field
}

// EffectiveDisplayName returns the display name to use
func (u *User) EffectiveDisplayName() string {
	if u.DisplayName != "" {
		return u.DisplayName
	}
	if u.GoogleName != "" {
		return u.GoogleName
	}
	return u.Email
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
	Email           string     `json:"email"`
	DisplayName     string     `json:"displayName"`
	CallSign        string     `json:"callSign,omitempty"`
	AvatarURL       string     `json:"avatarUrl,omitempty"`
	Status          UserStatus `json:"status,omitempty"`
	GoogleName      string     `json:"googleName,omitempty"`
	GoogleAvatarURL string     `json:"googleAvatarUrl,omitempty"`
}

// UpdateUserParams represents parameters for updating a user
type UpdateUserParams struct {
	DisplayName     *string     `json:"displayName,omitempty"`
	AvatarURL       *string     `json:"avatarUrl,omitempty"`
	Status          *UserStatus `json:"status,omitempty"`
	CallSign        *string     `json:"callSign,omitempty"`
	GoogleName      *string     `json:"googleName,omitempty"`
	GoogleAvatarURL *string     `json:"googleAvatarUrl,omitempty"`
	AvatarType      *AvatarType `json:"avatarType,omitempty"`
	CustomAvatarURL *string     `json:"customAvatarUrl,omitempty"`
	AvatarImageID   *string     `json:"avatarImageAssetId,omitempty"`
}

// AdminUpdateUserParams represents admin-only user updates
type AdminUpdateUserParams struct {
	Status         *UserStatus `json:"status,omitempty"`
	IsAdmin        *bool       `json:"isAdmin,omitempty"`
	IsContentAdmin *bool       `json:"isContentAdmin,omitempty"`
	IsGearAdmin    *bool       `json:"isGearAdmin,omitempty"` // Deprecated alias accepted for compatibility
}

// UpdateProfileParams represents parameters for updating user profile
type UpdateProfileParams struct {
	CallSign    *string     `json:"callSign,omitempty"`
	DisplayName *string     `json:"displayName,omitempty"`
	AvatarType  *AvatarType `json:"avatarType,omitempty"`
}

// UpdateSocialSettingsParams represents parameters for updating social settings
type UpdateSocialSettingsParams struct {
	ProfileVisibility *ProfileVisibility `json:"profileVisibility,omitempty"`
	ShowAircraft      *bool              `json:"showAircraft,omitempty"`
	AllowSearch       *bool              `json:"allowSearch,omitempty"`
}

// UserProfile represents the public profile response
type UserProfile struct {
	ID                 string     `json:"id"`
	CallSign           string     `json:"callSign,omitempty"`
	DisplayName        string     `json:"displayName,omitempty"`
	GoogleName         string     `json:"googleName,omitempty"`
	EffectiveAvatarURL string     `json:"effectiveAvatarUrl"`
	AvatarType         AvatarType `json:"avatarType,omitempty"`
	CreatedAt          time.Time  `json:"createdAt"`
}

// ToProfile converts a User to a UserProfile (public view)
func (u *User) ToProfile() *UserProfile {
	return &UserProfile{
		ID:                 u.ID,
		CallSign:           u.CallSign,
		DisplayName:        u.DisplayName,
		GoogleName:         u.GoogleName,
		EffectiveAvatarURL: u.EffectiveAvatarURL(),
		AvatarType:         u.AvatarType,
		CreatedAt:          u.CreatedAt,
	}
}

// PilotSearchResult represents a pilot in search results
type PilotSearchResult struct {
	ID                 string `json:"id"`
	CallSign           string `json:"callSign,omitempty"`
	DisplayName        string `json:"displayName,omitempty"`
	GoogleName         string `json:"googleName,omitempty"`
	EffectiveAvatarURL string `json:"effectiveAvatarUrl"`
}

// PilotSearchParams represents search parameters for pilots
type PilotSearchParams struct {
	Query  string `json:"q"`
	Limit  int    `json:"limit,omitempty"`
	Offset int    `json:"offset,omitempty"`
}

// PilotProfile represents a pilot's public profile with their aircraft
type PilotProfile struct {
	ID                 string           `json:"id"`
	CallSign           string           `json:"callSign,omitempty"`
	DisplayName        string           `json:"displayName,omitempty"`
	GoogleName         string           `json:"googleName,omitempty"`
	EffectiveAvatarURL string           `json:"effectiveAvatarUrl"`
	CreatedAt          time.Time        `json:"createdAt"`
	Aircraft           []AircraftPublic `json:"aircraft"`
	PublishedBuilds    []Build          `json:"publishedBuilds"`
	IsFollowing        bool             `json:"isFollowing"`    // Whether current user follows this pilot
	FollowerCount      int              `json:"followerCount"`  // Number of followers
	FollowingCount     int              `json:"followingCount"` // Number of users this pilot follows
}

// PilotSummary represents minimal pilot info for follower/following lists
type PilotSummary struct {
	ID                 string `json:"id"`
	CallSign           string `json:"callSign,omitempty"`
	DisplayName        string `json:"displayName,omitempty"`
	EffectiveAvatarURL string `json:"effectiveAvatarUrl"`
}

// PilotSummaryWithFollowers extends PilotSummary with follower count for discovery
type PilotSummaryWithFollowers struct {
	PilotSummary
	FollowerCount int `json:"followerCount"`
}

// Follow represents a follow relationship between two users
type Follow struct {
	ID             string    `json:"id"`
	FollowerUserID string    `json:"followerUserId"` // The user who is following
	FollowedUserID string    `json:"followedUserId"` // The user being followed
	CreatedAt      time.Time `json:"createdAt"`
}

// FollowListResponse represents a paginated list of followers or following
type FollowListResponse struct {
	Pilots     []PilotSummary `json:"pilots"`
	TotalCount int            `json:"totalCount"`
}

// AircraftPublic represents aircraft info for public pilot profiles
type AircraftPublic struct {
	ID               string                     `json:"id"`
	Name             string                     `json:"name"`
	Nickname         string                     `json:"nickname,omitempty"`
	Type             AircraftType               `json:"type,omitempty"`
	HasImage         bool                       `json:"hasImage"`
	Description      string                     `json:"description,omitempty"`
	CreatedAt        time.Time                  `json:"createdAt"`
	Components       []AircraftComponentPublic  `json:"components,omitempty"`
	ReceiverSettings *ReceiverSanitizedSettings `json:"receiverSettings,omitempty"` // Sanitized receiver data
	Tuning           *AircraftTuningPublic      `json:"tuning,omitempty"`           // Public tuning data
}

// AircraftTuningPublic represents tuning data for public view (PIDs, rates, filters)
type AircraftTuningPublic struct {
	FirmwareName    FCConfigFirmware `json:"firmwareName,omitempty"`
	FirmwareVersion string           `json:"firmwareVersion,omitempty"`
	BoardTarget     string           `json:"boardTarget,omitempty"`
	BoardName       string           `json:"boardName,omitempty"`
	ParsedTuning    *ParsedTuning    `json:"parsedTuning,omitempty"`
	SnapshotDate    time.Time        `json:"snapshotDate,omitempty"`
}

// AircraftComponentPublic represents component info for public view
// NOTE: Intentionally omits purchase price, seller, notes, and other private details
type AircraftComponentPublic struct {
	Category     ComponentCategory `json:"category"`
	Name         string            `json:"name,omitempty"`
	Manufacturer string            `json:"manufacturer,omitempty"`
	ImageURL     string            `json:"imageUrl,omitempty"`
	CatalogID    string            `json:"catalogId,omitempty"`
}

// ReceiverSanitizedSettings contains only safe-to-share receiver configuration
// CRITICAL: This struct MUST NOT contain BindPhrase, ModelMatch, UID, or any secrets
// Uses the SAME field names as the frontend ReceiverConfig for simplicity
type ReceiverSanitizedSettings struct {
	Rate       *int   `json:"rate,omitempty"`       // Packet rate in Hz (e.g., 250, 500)
	Tlm        *int   `json:"tlm,omitempty"`        // Telemetry ratio denominator (e.g., 8 for 1:8, 0 for off)
	Power      *int   `json:"power,omitempty"`      // TX power in mW (e.g., 100, 250, 500)
	DeviceName string `json:"deviceName,omitempty"` // Device name
}

// CallSign validation
var callSignRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,20}$`)

type callSignProfanityChecker interface {
	IsProfane(s string) bool
}

var defaultCallSignProfanityChecker callSignProfanityChecker = goaway.NewProfanityDetector()

// ValidateCallSign validates a callsign
func ValidateCallSign(callSign string) error {
	return validateCallSignWithChecker(callSign, defaultCallSignProfanityChecker)
}

func validateCallSignWithChecker(callSign string, profanityChecker callSignProfanityChecker) error {
	callSign = strings.TrimSpace(callSign)
	if callSign == "" {
		return &ValidationError{Field: "callSign", Message: "callsign is required"}
	}
	if len(callSign) < 3 {
		return &ValidationError{Field: "callSign", Message: "callsign must be at least 3 characters"}
	}
	if len(callSign) > 20 {
		return &ValidationError{Field: "callSign", Message: "callsign must be at most 20 characters"}
	}
	if !callSignRegex.MatchString(callSign) {
		return &ValidationError{Field: "callSign", Message: "callsign can only contain letters, numbers, underscores, and hyphens"}
	}
	if profanityChecker != nil && profanityChecker.IsProfane(callSign) {
		return &ValidationError{Field: "callSign", Message: "callsign contains inappropriate language"}
	}
	return nil
}

// ValidationError represents a field validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Message
}

// IsValidUserStatus checks if a user status is one of the supported values.
func IsValidUserStatus(status UserStatus) bool {
	switch status {
	case UserStatusActive, UserStatusDisabled, UserStatusPending:
		return true
	default:
		return false
	}
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
	ID        string     `json:"id"`
	UserID    string     `json:"userId"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expiresAt"`
	CreatedAt time.Time  `json:"createdAt"`
	RevokedAt *time.Time `json:"revokedAt,omitempty"`
}

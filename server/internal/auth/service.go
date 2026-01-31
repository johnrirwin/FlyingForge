package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/johnrirwin/mcp-news-feed/internal/config"
	"github.com/johnrirwin/mcp-news-feed/internal/database"
	"github.com/johnrirwin/mcp-news-feed/internal/logging"
	"github.com/johnrirwin/mcp-news-feed/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// Service handles authentication operations
type Service struct {
	config    config.AuthConfig
	userStore *database.UserStore
	logger    *logging.Logger
}

// NewService creates a new auth service
func NewService(userStore *database.UserStore, cfg config.AuthConfig, logger *logging.Logger) *Service {
	return &Service{
		config:    cfg,
		userStore: userStore,
		logger:    logger,
	}
}

// SignupWithEmail creates a new user with email/password
func (s *Service) SignupWithEmail(ctx context.Context, params models.SignupParams) (*models.AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(params.Email))

	// Validate input
	if email == "" {
		return nil, &AuthError{Code: "invalid_input", Message: "email is required"}
	}
	if params.Password == "" {
		return nil, &AuthError{Code: "invalid_input", Message: "password is required"}
	}
	if len(params.Password) < 8 {
		return nil, &AuthError{Code: "invalid_input", Message: "password must be at least 8 characters"}
	}
	if params.DisplayName == "" {
		params.DisplayName = strings.Split(email, "@")[0]
	}

	// Check if user exists
	existing, err := s.userStore.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existing != nil {
		return nil, &AuthError{Code: "user_exists", Message: "a user with this email already exists"}
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(params.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user, err := s.userStore.Create(ctx, models.CreateUserParams{
		Email:       email,
		Password:    string(passwordHash),
		DisplayName: params.DisplayName,
		Status:      models.UserStatusActive,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate tokens
	tokens, err := s.generateTokens(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	s.logger.Info("User signed up with email", logging.WithFields(map[string]interface{}{
		"userId": user.ID,
		"email":  user.Email,
	}))

	return &models.AuthResponse{
		User:      user,
		Tokens:    tokens,
		IsNewUser: true,
	}, nil
}

// LoginWithEmail authenticates a user with email/password
func (s *Service) LoginWithEmail(ctx context.Context, params models.LoginParams) (*models.AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(params.Email))

	if email == "" || params.Password == "" {
		return nil, &AuthError{Code: "invalid_input", Message: "email and password are required"}
	}

	// Get user
	user, err := s.userStore.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, &AuthError{Code: "invalid_credentials", Message: "invalid email or password"}
	}

	// Check status
	if user.Status != models.UserStatusActive {
		return nil, &AuthError{Code: "account_disabled", Message: "account is disabled"}
	}

	// Check password
	if user.PasswordHash == "" {
		return nil, &AuthError{Code: "invalid_credentials", Message: "this account uses social login"}
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(params.Password)); err != nil {
		return nil, &AuthError{Code: "invalid_credentials", Message: "invalid email or password"}
	}

	// Update last login
	if err := s.userStore.UpdateLastLogin(ctx, user.ID); err != nil {
		s.logger.Warn("Failed to update last login", logging.WithField("error", err.Error()))
	}

	// Generate tokens
	tokens, err := s.generateTokens(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	s.logger.Info("User logged in with email", logging.WithFields(map[string]interface{}{
		"userId": user.ID,
		"email":  user.Email,
	}))

	return &models.AuthResponse{
		User:   user,
		Tokens: tokens,
	}, nil
}

// LoginWithGoogle authenticates a user with Google OAuth
func (s *Service) LoginWithGoogle(ctx context.Context, params models.GoogleLoginParams) (*models.AuthResponse, error) {
	var claims *models.GoogleClaims
	var err error

	if params.IDToken != "" {
		claims, err = s.validateGoogleIDToken(ctx, params.IDToken)
	} else if params.Code != "" {
		claims, err = s.exchangeGoogleCode(ctx, params.Code, params.RedirectURI)
	} else {
		return nil, &AuthError{Code: "invalid_input", Message: "id_token or code is required"}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to validate Google credentials: %w", err)
	}

	email := strings.ToLower(strings.TrimSpace(claims.Email))
	isNewUser := false
	isLinked := false

	// Check if identity already exists
	identity, err := s.userStore.GetIdentityByProvider(ctx, models.AuthProviderGoogle, claims.Subject)
	if err != nil {
		return nil, fmt.Errorf("failed to check identity: %w", err)
	}

	var user *models.User

	if identity != nil {
		// Identity exists - get the user
		user, err = s.userStore.GetByID(ctx, identity.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}
		if user == nil {
			return nil, &AuthError{Code: "user_not_found", Message: "user not found"}
		}
	} else {
		// Identity doesn't exist - check if user exists by email
		user, err = s.userStore.GetByEmail(ctx, email)
		if err != nil {
			return nil, fmt.Errorf("failed to check user: %w", err)
		}

		if user != nil {
			// User exists - link the Google identity
			_, err = s.userStore.CreateIdentity(ctx, user.ID, models.AuthProviderGoogle, claims.Subject, email)
			if err != nil {
				return nil, fmt.Errorf("failed to link identity: %w", err)
			}
			isLinked = true
			s.logger.Info("Linked Google identity to existing user", logging.WithFields(map[string]interface{}{
				"userId":    user.ID,
				"googleSub": claims.Subject,
			}))
		} else {
			// Create new user
			displayName := claims.Name
			if displayName == "" {
				displayName = strings.Split(email, "@")[0]
			}

			user, err = s.userStore.Create(ctx, models.CreateUserParams{
				Email:       email,
				DisplayName: displayName,
				AvatarURL:   claims.Picture,
				Status:      models.UserStatusActive,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create user: %w", err)
			}

			// Link Google identity
			_, err = s.userStore.CreateIdentity(ctx, user.ID, models.AuthProviderGoogle, claims.Subject, email)
			if err != nil {
				return nil, fmt.Errorf("failed to create identity: %w", err)
			}

			isNewUser = true
			s.logger.Info("Created new user via Google", logging.WithFields(map[string]interface{}{
				"userId":    user.ID,
				"googleSub": claims.Subject,
			}))
		}
	}

	// Check status
	if user.Status != models.UserStatusActive {
		return nil, &AuthError{Code: "account_disabled", Message: "account is disabled"}
	}

	// Update last login
	if err := s.userStore.UpdateLastLogin(ctx, user.ID); err != nil {
		s.logger.Warn("Failed to update last login", logging.WithField("error", err.Error()))
	}

	// Generate tokens
	tokens, err := s.generateTokens(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &models.AuthResponse{
		User:      user,
		Tokens:    tokens,
		IsNewUser: isNewUser,
		IsLinked:  isLinked,
	}, nil
}

// RefreshTokens refreshes the access token using a refresh token
func (s *Service) RefreshTokens(ctx context.Context, refreshToken string) (*models.AuthTokens, error) {
	tokenHash := hashToken(refreshToken)

	storedToken, err := s.userStore.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}
	if storedToken == nil {
		return nil, &AuthError{Code: "invalid_token", Message: "invalid or expired refresh token"}
	}

	user, err := s.userStore.GetByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil || user.Status != models.UserStatusActive {
		return nil, &AuthError{Code: "invalid_token", Message: "user not found or disabled"}
	}

	// Revoke old token
	if err := s.userStore.RevokeRefreshToken(ctx, storedToken.ID); err != nil {
		s.logger.Warn("Failed to revoke old refresh token", logging.WithField("error", err.Error()))
	}

	// Generate new tokens
	tokens, err := s.generateTokens(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return tokens, nil
}

// Logout revokes all refresh tokens for a user
func (s *Service) Logout(ctx context.Context, userID string) error {
	return s.userStore.RevokeAllUserRefreshTokens(ctx, userID)
}

// ValidateAccessToken validates a JWT access token and returns the user ID
func (s *Service) ValidateAccessToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWTSecret), nil
	})

	if err != nil {
		return "", &AuthError{Code: "invalid_token", Message: "invalid or expired token"}
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", &AuthError{Code: "invalid_token", Message: "invalid token claims"}
	}

	// Validate issuer and audience
	if iss, _ := claims["iss"].(string); iss != s.config.JWTIssuer {
		return "", &AuthError{Code: "invalid_token", Message: "invalid token issuer"}
	}
	if aud, _ := claims["aud"].(string); aud != s.config.JWTAudience {
		return "", &AuthError{Code: "invalid_token", Message: "invalid token audience"}
	}

	userID, ok := claims["sub"].(string)
	if !ok || userID == "" {
		return "", &AuthError{Code: "invalid_token", Message: "invalid token subject"}
	}

	return userID, nil
}

// GetUser retrieves a user by ID
func (s *Service) GetUser(ctx context.Context, userID string) (*models.User, error) {
	return s.userStore.GetByID(ctx, userID)
}

// generateTokens generates access and refresh tokens
func (s *Service) generateTokens(ctx context.Context, user *models.User) (*models.AuthTokens, error) {
	now := time.Now()

	// Generate access token
	accessClaims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"name":  user.DisplayName,
		"iss":   s.config.JWTIssuer,
		"aud":   s.config.JWTAudience,
		"iat":   now.Unix(),
		"exp":   now.Add(s.config.AccessTokenTTL).Unix(),
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(s.config.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Generate refresh token
	refreshTokenBytes := make([]byte, 32)
	if _, err := rand.Read(refreshTokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	refreshTokenString := base64.URLEncoding.EncodeToString(refreshTokenBytes)

	// Store refresh token hash
	refreshTokenHash := hashToken(refreshTokenString)
	expiresAt := now.Add(s.config.RefreshTokenTTL)

	_, err = s.userStore.CreateRefreshToken(ctx, user.ID, refreshTokenHash, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &models.AuthTokens{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.config.AccessTokenTTL.Seconds()),
	}, nil
}

// validateGoogleIDToken validates a Google ID token
func (s *Service) validateGoogleIDToken(ctx context.Context, idToken string) (*models.GoogleClaims, error) {
	// Validate with Google's tokeninfo endpoint
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + url.QueryEscape(idToken))
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token validation failed: %s", string(body))
	}

	var tokenInfo struct {
		Aud           string `json:"aud"`
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		return nil, fmt.Errorf("failed to decode token info: %w", err)
	}

	// Validate audience
	if tokenInfo.Aud != s.config.GoogleClientID {
		return nil, fmt.Errorf("invalid token audience")
	}

	return &models.GoogleClaims{
		Subject:       tokenInfo.Sub,
		Email:         tokenInfo.Email,
		EmailVerified: tokenInfo.EmailVerified == "true",
		Name:          tokenInfo.Name,
		Picture:       tokenInfo.Picture,
		GivenName:     tokenInfo.GivenName,
		FamilyName:    tokenInfo.FamilyName,
	}, nil
}

// exchangeGoogleCode exchanges an authorization code for tokens
func (s *Service) exchangeGoogleCode(ctx context.Context, code, redirectURI string) (*models.GoogleClaims, error) {
	if s.config.GoogleClientSecret == "" {
		return nil, fmt.Errorf("Google client secret not configured")
	}

	if redirectURI == "" {
		redirectURI = s.config.GoogleRedirectURI
	}

	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", s.config.GoogleClientID)
	data.Set("client_secret", s.config.GoogleClientSecret)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	resp, err := http.PostForm("https://oauth2.googleapis.com/token", data)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("code exchange failed: %s", string(body))
	}

	var tokenResp struct {
		IDToken string `json:"id_token"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return s.validateGoogleIDToken(ctx, tokenResp.IDToken)
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// AuthError represents an authentication error
type AuthError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AuthError) Error() string {
	return e.Message
}

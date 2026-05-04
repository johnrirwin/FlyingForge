package auth

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/johnrirwin/flyingforge/internal/config"
	"github.com/johnrirwin/flyingforge/internal/database"
	"github.com/johnrirwin/flyingforge/internal/models"
	"github.com/johnrirwin/flyingforge/internal/testutil"
)

// setupTestAuthService creates a test auth service with a test database
func setupTestAuthService(t *testing.T) *Service {
	t.Helper()

	testDB := testutil.NewTestDB(t)
	t.Cleanup(func() { testDB.Close() })

	db := &database.DB{DB: testDB.DB}
	userStore := database.NewUserStore(db)
	logger := testutil.NullLogger()
	cfg := config.AuthConfig{
		JWTSecret:         "test-secret-key-minimum-32-chars-long",
		JWTIssuer:         "flyingforge-test",
		JWTAudience:       "flyingforge-users",
		AccessTokenTTL:    15 * time.Minute,
		RefreshTokenTTL:   7 * 24 * time.Hour,
		GoogleClientID:    "test-client-id",
		GoogleRedirectURI: "http://localhost:3000/auth/callback",
	}
	return NewService(userStore, cfg, logger)
}

func TestAuthError(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		message  string
		expected string
	}{
		{
			name:     "invalid_input error",
			code:     "invalid_input",
			message:  "email is required",
			expected: "email is required",
		},
		{
			name:     "user_exists error",
			code:     "user_exists",
			message:  "a user with this email already exists",
			expected: "a user with this email already exists",
		},
		{
			name:     "invalid_credentials error",
			code:     "invalid_credentials",
			message:  "invalid email or password",
			expected: "invalid email or password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &AuthError{Code: tt.code, Message: tt.message}
			if err.Error() != tt.expected {
				t.Errorf("AuthError.Error() = %s, want %s", err.Error(), tt.expected)
			}
		})
	}
}

func TestServiceCreation(t *testing.T) {
	service := setupTestAuthService(t)
	if service == nil {
		t.Error("Expected service to be created, got nil")
	}
}

func TestValidateAccessToken_Invalid(t *testing.T) {
	service := setupTestAuthService(t)

	// Test with invalid token
	_, err := service.ValidateAccessToken("invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token, got nil")
	}
}

func TestValidateAccessToken_Empty(t *testing.T) {
	service := setupTestAuthService(t)

	// Test with empty token
	_, err := service.ValidateAccessToken("")
	if err == nil {
		t.Error("Expected error for empty token, got nil")
	}
}

func TestResolveGoogleUser_RequiresVerifiedEmailForEmailFallback(t *testing.T) {
	service := setupTestAuthService(t)

	_, _, _, err := service.resolveGoogleUser(context.Background(), &models.GoogleClaims{
		Subject:       "google-subject-1",
		Email:         "pilot@example.com",
		EmailVerified: false,
		Name:          "Pilot",
	})
	if err == nil {
		t.Fatal("expected unverified Google email to be rejected")
	}

	authErr, ok := err.(*AuthError)
	if !ok {
		t.Fatalf("expected AuthError, got %T", err)
	}
	if authErr.Code != "invalid_credentials" {
		t.Fatalf("expected invalid_credentials, got %q", authErr.Code)
	}
	if !strings.Contains(authErr.Message, "verified email") {
		t.Fatalf("expected verified email message, got %q", authErr.Message)
	}
}

func TestResolveGoogleUser_LinksExistingUserByVerifiedEmail(t *testing.T) {
	service := setupTestAuthService(t)

	user, err := service.userStore.Create(context.Background(), models.CreateUserParams{
		Email:       "pilot@example.com",
		DisplayName: "Pilot",
		Status:      models.UserStatusActive,
	})
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	resolvedUser, isNewUser, isLinked, err := service.resolveGoogleUser(context.Background(), &models.GoogleClaims{
		Subject:       "google-subject-2",
		Email:         "pilot@example.com",
		EmailVerified: true,
		Name:          "Pilot",
	})
	if err != nil {
		t.Fatalf("expected verified email to link existing user, got %v", err)
	}
	if isNewUser {
		t.Fatal("expected existing user link, not new user")
	}
	if !isLinked {
		t.Fatal("expected Google identity to be linked to existing user")
	}
	if resolvedUser.ID != user.ID {
		t.Fatalf("expected resolved user %s, got %s", user.ID, resolvedUser.ID)
	}

	identity, err := service.userStore.GetIdentityByProvider(context.Background(), models.AuthProviderGoogle, "google-subject-2")
	if err != nil {
		t.Fatalf("failed to load Google identity: %v", err)
	}
	if identity == nil || identity.UserID != user.ID {
		t.Fatalf("expected linked Google identity for user %s, got %+v", user.ID, identity)
	}
}

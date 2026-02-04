package auth

import (
	"testing"
	"time"

	"github.com/johnrirwin/flyingforge/internal/config"
	"github.com/johnrirwin/flyingforge/internal/database"
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

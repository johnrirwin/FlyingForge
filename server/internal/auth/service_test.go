package auth

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/johnrirwin/rotorlife/internal/config"
	"github.com/johnrirwin/rotorlife/internal/database"
	"github.com/johnrirwin/rotorlife/internal/models"
	"github.com/johnrirwin/rotorlife/internal/testutil"
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
		JWTIssuer:         "rotorlife-test",
		JWTAudience:       "rotorlife-users",
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

func TestPasswordValidation(t *testing.T) {
	service := setupTestAuthService(t)

	tests := []struct {
		name     string
		password string
		wantErr  bool
		errCode  string
	}{
		{
			name:     "valid password",
			password: "securepassword123",
			wantErr:  false,
		},
		{
			name:     "exactly 8 characters",
			password: "12345678",
			wantErr:  false,
		},
		{
			name:     "too short",
			password: "1234567",
			wantErr:  true,
			errCode:  "invalid_input",
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
			errCode:  "invalid_input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test password validation through the actual SignupWithEmail method
			ctx := context.Background()
			email := fmt.Sprintf("test-%s@example.com", tt.name)

			_, err := service.SignupWithEmail(ctx, models.SignupParams{
				Email:       email,
				Password:    tt.password,
				DisplayName: "Test User",
			})

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
					return
				}
				authErr, ok := err.(*AuthError)
				if !ok {
					t.Errorf("Expected AuthError but got %T: %v", err, err)
					return
				}
				if authErr.Code != tt.errCode {
					t.Errorf("Expected error code %s but got %s", tt.errCode, authErr.Code)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDefaultDisplayName(t *testing.T) {
	service := setupTestAuthService(t)

	tests := []struct {
		name         string
		email        string
		displayName  string
		expectedName string
	}{
		{
			name:         "uses provided display name",
			email:        "test1@example.com",
			displayName:  "Test User",
			expectedName: "Test User",
		},
		{
			name:         "derives from email when empty",
			email:        "john@example.com",
			displayName:  "",
			expectedName: "john",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Test through actual SignupWithEmail method
			authResp, err := service.SignupWithEmail(ctx, models.SignupParams{
				Email:       tt.email,
				Password:    "validpassword123",
				DisplayName: tt.displayName,
			})

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if authResp.User.DisplayName != tt.expectedName {
				t.Errorf("Display name = %s, want %s", authResp.User.DisplayName, tt.expectedName)
			}
		})
	}
}

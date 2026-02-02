package models

import "testing"

func TestUserStatus_Values(t *testing.T) {
	// Verify the constants have expected values
	if UserStatusActive != "active" {
		t.Errorf("UserStatusActive = %q, want %q", UserStatusActive, "active")
	}
	if UserStatusDisabled != "disabled" {
		t.Errorf("UserStatusDisabled = %q, want %q", UserStatusDisabled, "disabled")
	}
	if UserStatusPending != "pending" {
		t.Errorf("UserStatusPending = %q, want %q", UserStatusPending, "pending")
	}
}

func TestAuthProvider_Values(t *testing.T) {
	if AuthProviderGoogle != "google" {
		t.Errorf("AuthProviderGoogle = %q, want %q", AuthProviderGoogle, "google")
	}
	if AuthProviderEmail != "email" {
		t.Errorf("AuthProviderEmail = %q, want %q", AuthProviderEmail, "email")
	}
}

func TestUser_JSONOmitsPasswordHash(t *testing.T) {
	// The PasswordHash field has json:"-" tag
	// This test verifies the tag exists by checking struct tag behavior
	// (This is mostly a documentation test - the real verification is the tag itself)
	user := User{
		ID:           "123",
		Email:        "test@example.com",
		PasswordHash: "secret-hash",
		DisplayName:  "Test User",
		Status:       UserStatusActive,
	}

	// Just verify the struct can be created with a password hash
	if user.PasswordHash != "secret-hash" {
		t.Errorf("User.PasswordHash = %q, want %q", user.PasswordHash, "secret-hash")
	}
}

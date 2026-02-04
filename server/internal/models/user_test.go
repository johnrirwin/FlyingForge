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
}

func TestUser_Creation(t *testing.T) {
	// Verify a user can be created with basic fields
	user := User{
		ID:          "123",
		Email:       "test@example.com",
		DisplayName: "Test User",
		Status:      UserStatusActive,
	}

	if user.ID != "123" {
		t.Errorf("User.ID = %q, want %q", user.ID, "123")
	}
	if user.Email != "test@example.com" {
		t.Errorf("User.Email = %q, want %q", user.Email, "test@example.com")
	}
}

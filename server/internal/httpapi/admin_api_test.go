package httpapi

import (
	"testing"

	"github.com/johnrirwin/flyingforge/internal/models"
)

func TestCanModerateContent(t *testing.T) {
	tests := []struct {
		name string
		user *models.User
		want bool
	}{
		{
			name: "nil user",
			user: nil,
			want: false,
		},
		{
			name: "regular user",
			user: &models.User{},
			want: false,
		},
		{
			name: "content admin",
			user: &models.User{IsContentAdmin: true},
			want: true,
		},
		{
			name: "full admin",
			user: &models.User{IsAdmin: true},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canModerateContent(tt.user)
			if got != tt.want {
				t.Fatalf("canModerateContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanManageUsers(t *testing.T) {
	tests := []struct {
		name string
		user *models.User
		want bool
	}{
		{
			name: "nil user",
			user: nil,
			want: false,
		},
		{
			name: "regular user",
			user: &models.User{},
			want: false,
		},
		{
			name: "content admin",
			user: &models.User{IsContentAdmin: true},
			want: false,
		},
		{
			name: "full admin",
			user: &models.User{IsAdmin: true},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canManageUsers(tt.user)
			if got != tt.want {
				t.Fatalf("canManageUsers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsJSONContentType(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        bool
	}{
		{
			name:        "plain json",
			contentType: "application/json",
			want:        true,
		},
		{
			name:        "json with charset",
			contentType: "application/json; charset=utf-8",
			want:        true,
		},
		{
			name:        "mixed case json",
			contentType: "Application/JSON",
			want:        true,
		},
		{
			name:        "multipart",
			contentType: "multipart/form-data; boundary=abc123",
			want:        false,
		},
		{
			name:        "empty",
			contentType: "",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isJSONContentType(tt.contentType)
			if got != tt.want {
				t.Fatalf("isJSONContentType() = %v, want %v", got, tt.want)
			}
		})
	}
}

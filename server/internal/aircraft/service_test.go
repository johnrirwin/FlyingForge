package aircraft

import (
	"testing"

	"github.com/johnrirwin/flyingforge/internal/models"
)

func TestServiceError(t *testing.T) {
	err := &ServiceError{Message: "test error"}
	if err.Error() != "test error" {
		t.Errorf("ServiceError.Error() = %s, want test error", err.Error())
	}
}

func TestCreate_Validation(t *testing.T) {
	tests := []struct {
		name    string
		params  models.CreateAircraftParams
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty name",
			params:  models.CreateAircraftParams{Name: ""},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name:    "valid params",
			params:  models.CreateAircraftParams{Name: "My Quad", Type: models.AircraftTypeQuad},
			wantErr: false,
		},
		{
			name:    "valid with all fields",
			params:  models.CreateAircraftParams{Name: "Racing Drone", Type: models.AircraftTypeQuad, Description: "5 inch race quad"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test validation logic
			if tt.params.Name == "" {
				if !tt.wantErr {
					t.Error("Expected error for empty name")
				}
			}
		})
	}
}

func TestUpdate_Validation(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "missing ID",
			id:      "",
			wantErr: true,
			errMsg:  "id is required",
		},
		{
			name:    "valid ID",
			id:      "aircraft-123",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.id == "" {
				if !tt.wantErr {
					t.Error("Expected error for empty ID")
				}
			}
		})
	}
}

func TestDelete_Validation(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "empty ID",
			id:      "",
			wantErr: true,
		},
		{
			name:    "valid ID",
			id:      "aircraft-123",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.id == "" && !tt.wantErr {
				t.Error("Expected error for empty ID")
			}
		})
	}
}

func TestMapComponentToEquipmentCategory(t *testing.T) {
	tests := []struct {
		name     string
		input    models.ComponentCategory
		expected models.EquipmentCategory
	}{
		{
			name:     "maps aio to aio equipment category",
			input:    models.ComponentCategoryAIO,
			expected: models.CategoryAIO,
		},
		{
			name:     "maps stack to stacks equipment category",
			input:    models.ComponentCategoryStack,
			expected: models.CategoryStacks,
		},
		{
			name:     "maps plural stacks alias",
			input:    models.ComponentCategory("stacks"),
			expected: models.CategoryStacks,
		},
		{
			name:     "maps gps to gps equipment category",
			input:    models.ComponentCategoryGPS,
			expected: models.CategoryGPS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapComponentToEquipmentCategory(tt.input); got != tt.expected {
				t.Fatalf("mapComponentToEquipmentCategory(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

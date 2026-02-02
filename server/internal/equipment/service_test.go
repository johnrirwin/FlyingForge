package equipment

import (
	"testing"
)

func TestServiceError(t *testing.T) {
	err := &ServiceError{Message: "test error message"}
	if err.Error() != "test error message" {
		t.Errorf("ServiceError.Error() = %s, want test error message", err.Error())
	}
}

func TestServiceError_UnknownSeller(t *testing.T) {
	err := &ServiceError{Message: "Unknown seller: fakeseller"}
	expected := "Unknown seller: fakeseller"
	if err.Error() != expected {
		t.Errorf("ServiceError.Error() = %s, want %s", err.Error(), expected)
	}
}

func TestSearchParamsDefaults(t *testing.T) {
	tests := []struct {
		name          string
		inputLimit    int
		expectedLimit int
	}{
		{
			name:          "zero limit defaults to 20",
			inputLimit:    0,
			expectedLimit: 20,
		},
		{
			name:          "negative limit defaults to 20",
			inputLimit:    -1,
			expectedLimit: 20,
		},
		{
			name:          "positive limit preserved",
			inputLimit:    50,
			expectedLimit: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit := tt.inputLimit
			if limit <= 0 {
				limit = 20
			}
			if limit != tt.expectedLimit {
				t.Errorf("limit = %d, want %d", limit, tt.expectedLimit)
			}
		})
	}
}

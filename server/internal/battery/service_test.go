package battery

import (
	"context"
	"testing"

	"github.com/johnrirwin/flyingforge/internal/models"
	"github.com/johnrirwin/flyingforge/internal/testutil"
)

// mockStore implements the Store interface for testing
type mockStore struct {
	codeExists bool
}

func (m *mockStore) BatteryCodeExists(ctx context.Context, userID, code string) (bool, error) {
	return m.codeExists, nil
}

func (m *mockStore) Create(ctx context.Context, userID, code string, params models.CreateBatteryParams) (*models.Battery, error) {
	return &models.Battery{
		ID:          "bat-test-1",
		UserID:      userID,
		BatteryCode: code,
		Name:        params.Name,
		Chemistry:   params.Chemistry,
		Cells:       params.Cells,
		CapacityMah: params.CapacityMah,
		CRating:     params.CRating,
		Connector:   params.Connector,
		Brand:       params.Brand,
		Model:       params.Model,
		Notes:       params.Notes,
	}, nil
}

func (m *mockStore) Get(ctx context.Context, id, userID string) (*models.Battery, error) {
	return nil, nil
}

func (m *mockStore) GetByCode(ctx context.Context, code, userID string) (*models.Battery, error) {
	return nil, nil
}

func (m *mockStore) Update(ctx context.Context, userID string, params models.UpdateBatteryParams) (*models.Battery, error) {
	return nil, nil
}

func (m *mockStore) Delete(ctx context.Context, id, userID string) error {
	return nil
}

func (m *mockStore) List(ctx context.Context, userID string, params models.BatteryListParams) (*models.BatteryListResponse, error) {
	return &models.BatteryListResponse{}, nil
}

func (m *mockStore) CreateLog(ctx context.Context, userID string, params models.CreateBatteryLogParams) (*models.BatteryLog, error) {
	return nil, nil
}

func (m *mockStore) ListLogs(ctx context.Context, batteryID, userID string, limit int) (*models.BatteryLogListResponse, error) {
	return &models.BatteryLogListResponse{}, nil
}

func (m *mockStore) DeleteLog(ctx context.Context, logID, userID string) error {
	return nil
}

func newTestService() *Service {
	return &Service{
		store:  &mockStore{},
		logger: testutil.NullLogger(),
	}
}

func TestService_Create(t *testing.T) {
	tests := []struct {
		name        string
		params      models.CreateBatteryParams
		wantErr     bool
		errContains string
	}{
		{
			name: "valid LIPO battery",
			params: models.CreateBatteryParams{
				Name:        "Race Pack 1",
				Chemistry:   models.ChemistryLIPO,
				Cells:       4,
				CapacityMah: 1500,
				Brand:       "CNHL",
			},
			wantErr: false,
		},
		{
			name: "valid LIPO HV battery",
			params: models.CreateBatteryParams{
				Chemistry:   models.ChemistryLIPOHV,
				Cells:       6,
				CapacityMah: 2200,
			},
			wantErr: false,
		},
		{
			name: "invalid chemistry",
			params: models.CreateBatteryParams{
				Chemistry:   "NIMH",
				Cells:       4,
				CapacityMah: 1500,
			},
			wantErr:     true,
			errContains: "invalid chemistry",
		},
		{
			name: "cells too low",
			params: models.CreateBatteryParams{
				Chemistry:   models.ChemistryLIPO,
				Cells:       0,
				CapacityMah: 1500,
			},
			wantErr:     true,
			errContains: "cells must be between 1 and 8",
		},
		{
			name: "cells too high",
			params: models.CreateBatteryParams{
				Chemistry:   models.ChemistryLIPO,
				Cells:       9,
				CapacityMah: 1500,
			},
			wantErr:     true,
			errContains: "cells must be between 1 and 8",
		},
		{
			name: "capacity zero",
			params: models.CreateBatteryParams{
				Chemistry:   models.ChemistryLIPO,
				Cells:       4,
				CapacityMah: 0,
			},
			wantErr:     true,
			errContains: "capacity must be greater than 0",
		},
		{
			name: "capacity negative",
			params: models.CreateBatteryParams{
				Chemistry:   models.ChemistryLIPO,
				Cells:       4,
				CapacityMah: -100,
			},
			wantErr:     true,
			errContains: "capacity must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService()
			ctx := context.Background()
			battery, err := svc.Create(ctx, "user-123", tt.params)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Create() expected error containing %q, got nil", tt.errContains)
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("Create() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("Create() unexpected error = %v", err)
				return
			}

			if battery == nil {
				t.Error("Create() returned nil battery")
				return
			}

			if battery.Chemistry != tt.params.Chemistry {
				t.Errorf("Create() battery.Chemistry = %v, want %v", battery.Chemistry, tt.params.Chemistry)
			}
			if battery.Cells != tt.params.Cells {
				t.Errorf("Create() battery.Cells = %v, want %v", battery.Cells, tt.params.Cells)
			}
			if battery.CapacityMah != tt.params.CapacityMah {
				t.Errorf("Create() battery.CapacityMah = %v, want %v", battery.CapacityMah, tt.params.CapacityMah)
			}
			if battery.BatteryCode == "" {
				t.Error("Create() battery.BatteryCode is empty")
			}
			if battery.UserID != "user-123" {
				t.Errorf("Create() battery.UserID = %v, want %v", battery.UserID, "user-123")
			}
		})
	}
}

func TestService_Update_Validation(t *testing.T) {
	tests := []struct {
		name        string
		params      models.UpdateBatteryParams
		wantErr     bool
		errContains string
	}{
		{
			name: "missing ID",
			params: models.UpdateBatteryParams{
				ID: "",
			},
			wantErr:     true,
			errContains: "id is required",
		},
		{
			name: "invalid chemistry",
			params: models.UpdateBatteryParams{
				ID:        "bat-123",
				Chemistry: ptr(models.BatteryChemistry("NIMH")),
			},
			wantErr:     true,
			errContains: "invalid chemistry",
		},
		{
			name: "cells too low",
			params: models.UpdateBatteryParams{
				ID:    "bat-123",
				Cells: intPtr(0),
			},
			wantErr:     true,
			errContains: "cells must be between 1 and 8",
		},
		{
			name: "capacity zero",
			params: models.UpdateBatteryParams{
				ID:          "bat-123",
				CapacityMah: intPtr(0),
			},
			wantErr:     true,
			errContains: "capacity must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestService()
			ctx := context.Background()
			_, err := svc.Update(ctx, "user-123", tt.params)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Update() expected error containing %q, got nil", tt.errContains)
					return
				}
				if tt.errContains != "" && !containsString(err.Error(), tt.errContains) {
					t.Errorf("Update() error = %v, want error containing %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestValidateCreateParams(t *testing.T) {
	svc := newTestService()

	tests := []struct {
		name    string
		params  models.CreateBatteryParams
		wantErr bool
	}{
		{
			name: "valid params",
			params: models.CreateBatteryParams{
				Chemistry:   models.ChemistryLIPO,
				Cells:       4,
				CapacityMah: 1500,
			},
			wantErr: false,
		},
		{
			name: "valid LIION",
			params: models.CreateBatteryParams{
				Chemistry:   models.ChemistryLIION,
				Cells:       2,
				CapacityMah: 3000,
			},
			wantErr: false,
		},
		{
			name: "1S battery valid",
			params: models.CreateBatteryParams{
				Chemistry:   models.ChemistryLIPO,
				Cells:       1,
				CapacityMah: 500,
			},
			wantErr: false,
		},
		{
			name: "8S battery valid",
			params: models.CreateBatteryParams{
				Chemistry:   models.ChemistryLIPO,
				Cells:       8,
				CapacityMah: 5000,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.validateCreateParams(tt.params)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateCreateParams() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("validateCreateParams() unexpected error = %v", err)
				}
			}
		})
	}
}

// Helper functions
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func ptr[T any](v T) *T {
	return &v
}

func intPtr(v int) *int {
	return &v
}

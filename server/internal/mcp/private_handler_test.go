package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/johnrirwin/flyingforge/internal/models"
	"github.com/johnrirwin/flyingforge/internal/testutil"
)

type stubAircraftReader struct {
	listResponse     *models.AircraftListResponse
	listErr          error
	detailsResponse  *models.AircraftDetailsResponse
	detailsErr       error
	receiverSettings *models.AircraftReceiverSettings
	receiverErr      error
}

func (s *stubAircraftReader) List(_ context.Context, _ string, _ models.AircraftListParams) (*models.AircraftListResponse, error) {
	return s.listResponse, s.listErr
}

func (s *stubAircraftReader) GetDetails(_ context.Context, _ string, _ string) (*models.AircraftDetailsResponse, error) {
	return s.detailsResponse, s.detailsErr
}

func (s *stubAircraftReader) GetReceiverSettings(_ context.Context, _ string, _ string) (*models.AircraftReceiverSettings, error) {
	return s.receiverSettings, s.receiverErr
}

type stubTuningReader struct {
	snapshot *models.AircraftTuningSnapshot
	err      error
}

func (s *stubTuningReader) GetLatestTuningSnapshot(_ context.Context, _ string, _ string) (*models.AircraftTuningSnapshot, error) {
	return s.snapshot, s.err
}

type stubRadioReader struct {
	radioListResponse  *models.RadioListResponse
	radioListErr       error
	radio              *models.Radio
	radioErr           error
	backupListResponse *models.RadioBackupListResponse
	backupListErr      error
}

func (s *stubRadioReader) ListRadios(_ context.Context, _ string, _ models.RadioListParams) (*models.RadioListResponse, error) {
	return s.radioListResponse, s.radioListErr
}

func (s *stubRadioReader) GetRadio(_ context.Context, _ string, _ string) (*models.Radio, error) {
	return s.radio, s.radioErr
}

func (s *stubRadioReader) ListBackups(_ context.Context, _ string, _ string, _ models.RadioBackupListParams) (*models.RadioBackupListResponse, error) {
	return s.backupListResponse, s.backupListErr
}

func authenticatedContext() context.Context {
	return WithRequestAuth(context.Background(), RequestAuth{UserID: "user-1"})
}

func TestGetAircraftDetailsOmitsRawReceiverSettings(t *testing.T) {
	handler := NewHandler(
		nil,
		nil,
		&stubAircraftReader{
			detailsResponse: &models.AircraftDetailsResponse{
				Aircraft: models.Aircraft{
					ID:        "air-1",
					Name:      "Demo Quad",
					Type:      models.AircraftTypeQuad,
					CreatedAt: time.Unix(100, 0).UTC(),
					UpdatedAt: time.Unix(200, 0).UTC(),
				},
				Components: []models.AircraftComponent{
					{
						ID:              "comp-1",
						AircraftID:      "air-1",
						Category:        models.ComponentCategoryReceiver,
						InventoryItemID: "inv-1",
						CreatedAt:       time.Unix(100, 0).UTC(),
						UpdatedAt:       time.Unix(200, 0).UTC(),
						InventoryItem: &models.InventoryItem{
							ID:           "inv-1",
							Name:         "Receiver",
							Category:     models.CategoryReceivers,
							Manufacturer: "Acme",
						},
					},
				},
				ReceiverSettings: &models.AircraftReceiverSettings{
					AircraftID: "air-1",
					Settings:   json.RawMessage(`{"bindPhrase":"secret-phrase"}`),
				},
			},
		},
		nil,
		nil,
		[]string{"flyingforge.read"},
		testutil.NullLogger(),
	)

	result, err := handler.HandleToolCall(authenticatedContext(), "get_aircraft_details", json.RawMessage(`{"aircraftId":"air-1"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	toolResult := result.(ToolResultData)
	payload := toolResult.StructuredContent.(aircraftDetailsToolResponse)
	if !payload.HasReceiverSettings {
		t.Fatal("expected receiver settings presence to be preserved")
	}

	serialized, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	text := string(serialized)
	if strings.Contains(text, "receiverSettings") {
		t.Fatalf("expected raw receiver settings to be omitted, got %s", text)
	}
	if strings.Contains(text, "bindPhrase") {
		t.Fatalf("expected sensitive receiver fields to be omitted, got %s", text)
	}
}

func TestGetAircraftReceiverSummarySanitizesSensitiveFields(t *testing.T) {
	handler := NewHandler(
		nil,
		nil,
		&stubAircraftReader{
			receiverSettings: &models.AircraftReceiverSettings{
				AircraftID: "air-1",
				Settings: json.RawMessage(`{
					"bindPhrase":"secret",
					"modelMatch":12,
					"uid":"abc123",
					"wifiPassword":"pw",
					"wifiSSID":"ssid",
					"rate":500,
					"tlm":64,
					"power":250,
					"deviceName":"EP1"
				}`),
			},
		},
		nil,
		nil,
		[]string{"flyingforge.read"},
		testutil.NullLogger(),
	)

	result, err := handler.HandleToolCall(authenticatedContext(), "get_aircraft_receiver_summary", json.RawMessage(`{"aircraftId":"air-1"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	toolResult := result.(ToolResultData)
	payload := toolResult.StructuredContent.(aircraftReceiverSummaryResponse)
	if !payload.HasReceiverConfig {
		t.Fatal("expected receiver config to be reported as present")
	}
	if payload.ReceiverSettings == nil {
		t.Fatal("expected sanitized receiver settings")
	}
	if payload.ReceiverSettings.DeviceName != "EP1" {
		t.Fatalf("expected safe device name to be preserved, got %+v", payload.ReceiverSettings)
	}

	serialized, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	text := string(serialized)
	for _, sensitive := range []string{"bindPhrase", "modelMatch", "uid", "wifiPassword", "wifiSSID"} {
		if strings.Contains(text, sensitive) {
			t.Fatalf("expected %q to be removed from sanitized payload: %s", sensitive, text)
		}
	}
}

func TestGetAircraftTuningOmitsDiffBackup(t *testing.T) {
	handler := NewHandler(
		nil,
		nil,
		nil,
		nil,
		&stubTuningReader{
			snapshot: &models.AircraftTuningSnapshot{
				ID:              "snap-1",
				AircraftID:      "air-1",
				FirmwareName:    models.FirmwareBetaflight,
				FirmwareVersion: "4.5.0",
				BoardTarget:     "STM32F7X2",
				BoardName:       "Matek",
				TuningData:      json.RawMessage(`{}`),
				DiffBackup:      "diff all secret",
				ParseStatus:     models.ParseStatusSuccess,
				CreatedAt:       time.Unix(500, 0).UTC(),
			},
		},
		[]string{"flyingforge.read"},
		testutil.NullLogger(),
	)

	result, err := handler.HandleToolCall(authenticatedContext(), "get_aircraft_tuning", json.RawMessage(`{"aircraftId":"air-1"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	toolResult := result.(ToolResultData)
	payload := toolResult.StructuredContent.(aircraftTuningToolResponse)
	if !payload.HasTuning {
		t.Fatal("expected tuning data to be present")
	}
	if !payload.HasDiffBackup {
		t.Fatal("expected hasDiffBackup to reflect stored metadata")
	}

	serialized, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	text := string(serialized)
	if strings.Contains(text, `"diffBackup":`) {
		t.Fatalf("expected raw diff backup to be omitted, got %s", text)
	}
	if strings.Contains(text, "rawCliDump") {
		t.Fatalf("expected raw CLI dump data to be omitted, got %s", text)
	}
}

func TestListRadioBackupsOmitsStoragePath(t *testing.T) {
	handler := NewHandler(
		nil,
		nil,
		nil,
		&stubRadioReader{
			radio: &models.Radio{
				ID:             "radio-1",
				Manufacturer:   models.ManufacturerRadioMaster,
				Model:          "Boxer",
				FirmwareFamily: models.FirmwareFamilyEdgeTX,
				CreatedAt:      time.Unix(100, 0).UTC(),
				UpdatedAt:      time.Unix(200, 0).UTC(),
			},
			backupListResponse: &models.RadioBackupListResponse{
				Backups: []models.RadioBackup{
					{
						ID:          "backup-1",
						RadioID:     "radio-1",
						BackupName:  "May backup",
						BackupType:  models.BackupTypeFullBackup,
						FileName:    "backup.zip",
						FileSize:    1024,
						StoragePath: "/private/backups/secret.zip",
						CreatedAt:   time.Unix(300, 0).UTC(),
					},
				},
				TotalCount: 1,
			},
		},
		nil,
		[]string{"flyingforge.read"},
		testutil.NullLogger(),
	)

	result, err := handler.HandleToolCall(authenticatedContext(), "list_radio_backups", json.RawMessage(`{"radioId":"radio-1"}`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	toolResult := result.(ToolResultData)
	payload := toolResult.StructuredContent.(listRadioBackupsResponse)
	if payload.TotalCount != 1 {
		t.Fatalf("expected one backup, got %+v", payload)
	}

	serialized, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	text := string(serialized)
	if strings.Contains(text, "storagePath") {
		t.Fatalf("expected storage paths to be omitted, got %s", text)
	}
}

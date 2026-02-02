package models

import (
	"encoding/json"
	"testing"
)

// TestReceiverSanitizationStripsSensitiveData verifies that bind phrase, model match,
// and other sensitive fields are NEVER present in sanitized receiver settings.
// This is a CRITICAL security test.
func TestReceiverSanitizationStripsSensitiveData(t *testing.T) {
	// Create receiver settings with sensitive data
	fullSettings := &ReceiverSettingsData{
		// SENSITIVE fields that must be stripped
		BindPhrase:    "super-secret-bind-phrase",
		BindingPhrase: "super-secret-bind-phrase",
		ModelMatch:    intPtr(42),
		ModelMatchNum: intPtr(42),
		ModelID:       intPtr(42),
		UID:           "rx-unique-id-12345",
		WifiPassword:  "secret-wifi-pass",
		WifiSSID:      "MyHomeNetwork",

		// SAFE fields that should be preserved (using frontend field names)
		Rate:       intPtr(500),
		Tlm:        intPtr(64),
		Power:      intPtr(250),
		DeviceName: "MyReceiver",
	}

	// Sanitize the settings
	sanitized := fullSettings.Sanitize()

	if sanitized == nil {
		t.Fatal("Sanitize() returned nil")
	}

	// CRITICAL: Verify sensitive fields are NOT present in sanitized output
	// We need to check the JSON output to ensure nothing leaks
	jsonBytes, err := json.Marshal(sanitized)
	if err != nil {
		t.Fatalf("Failed to marshal sanitized settings: %v", err)
	}
	jsonStr := string(jsonBytes)

	// These strings should NEVER appear in sanitized output
	sensitiveValues := []string{
		"super-secret-bind-phrase",
		"bindPhrase",
		"bindingPhrase",
		"modelMatch",
		"modelId",
		"rx-unique-id-12345",
		"uid",
		"secret-wifi-pass",
		"wifiPassword",
		"MyHomeNetwork",
		"wifiSSID",
	}

	for _, sensitive := range sensitiveValues {
		if containsIgnoreCase(jsonStr, sensitive) {
			t.Errorf("SECURITY VIOLATION: Sanitized output contains sensitive value '%s': %s", sensitive, jsonStr)
		}
	}

	// Verify safe fields ARE present
	if sanitized.Rate == nil || *sanitized.Rate != 500 {
		t.Errorf("Expected Rate 500, got %v", sanitized.Rate)
	}
	if sanitized.Tlm == nil || *sanitized.Tlm != 64 {
		t.Errorf("Expected Tlm 64, got %v", sanitized.Tlm)
	}
	if sanitized.Power == nil || *sanitized.Power != 250 {
		t.Errorf("Expected Power 250, got %v", sanitized.Power)
	}
	if sanitized.DeviceName != "MyReceiver" {
		t.Errorf("Expected DeviceName 'MyReceiver', got '%s'", sanitized.DeviceName)
	}
}

// TestSanitizeReceiverSettingsFromRawJSON tests the SanitizeReceiverSettings function
// which parses raw JSON and returns sanitized settings.
func TestSanitizeReceiverSettingsFromRawJSON(t *testing.T) {
	// Create a raw JSON receiver settings object (using frontend field names)
	rawJSON := `{
		"bindingPhrase": "my-secret-phrase",
		"modelMatch": 7,
		"uid": "receiver-uid-xyz",
		"rate": 250,
		"tlm": 128,
		"power": 100,
		"deviceName": "TestDevice"
	}`

	aircraftSettings := &AircraftReceiverSettings{
		ID:         "test-id",
		AircraftID: "aircraft-id",
		Settings:   json.RawMessage(rawJSON),
	}

	sanitized := SanitizeReceiverSettings(aircraftSettings)

	if sanitized == nil {
		t.Fatal("SanitizeReceiverSettings returned nil")
	}

	// Verify JSON output doesn't contain sensitive data
	jsonBytes, err := json.Marshal(sanitized)
	if err != nil {
		t.Fatalf("Failed to marshal sanitized settings: %v", err)
	}
	jsonStr := string(jsonBytes)

	sensitiveValues := []string{
		"my-secret-phrase",
		"receiver-uid-xyz",
		"bindPhrase",
		"bindingPhrase",
		"modelMatch",
		"modelId",
		"uid",
	}

	for _, sensitive := range sensitiveValues {
		if containsIgnoreCase(jsonStr, sensitive) {
			t.Errorf("SECURITY VIOLATION: Sanitized output contains sensitive value '%s': %s", sensitive, jsonStr)
		}
	}

	// Verify safe fields are preserved
	if sanitized.Rate == nil || *sanitized.Rate != 250 {
		t.Errorf("Expected Rate 250, got %v", sanitized.Rate)
	}
	if sanitized.Tlm == nil || *sanitized.Tlm != 128 {
		t.Errorf("Expected Tlm 128, got %v", sanitized.Tlm)
	}
	if sanitized.Power == nil || *sanitized.Power != 100 {
		t.Errorf("Expected Power 100, got %v", sanitized.Power)
	}
	if sanitized.DeviceName != "TestDevice" {
		t.Errorf("Expected DeviceName 'TestDevice', got '%s'", sanitized.DeviceName)
	}
}

// TestSanitizeNilSettings verifies that sanitizing nil settings returns nil
func TestSanitizeNilSettings(t *testing.T) {
	var settings *ReceiverSettingsData = nil
	sanitized := settings.Sanitize()
	if sanitized != nil {
		t.Error("Expected nil result for nil settings")
	}

	result := SanitizeReceiverSettings(nil)
	if result != nil {
		t.Error("Expected nil result for nil AircraftReceiverSettings")
	}
}

// TestSanitizeEmptySettings verifies that sanitizing empty settings works
func TestSanitizeEmptySettings(t *testing.T) {
	settings := &ReceiverSettingsData{}
	sanitized := settings.Sanitize()

	if sanitized == nil {
		t.Fatal("Expected non-nil result for empty settings")
	}

	// All fields should be nil/empty
	if sanitized.Rate != nil || sanitized.Tlm != nil || sanitized.Power != nil || sanitized.DeviceName != "" {
		t.Error("Expected empty/nil fields for empty settings")
	}
}

// TestReceiverSanitizedSettingsHasNoSensitiveFields verifies that the
// ReceiverSanitizedSettings struct itself has no sensitive fields defined.
// This is a compile-time-like check through reflection.
func TestReceiverSanitizedSettingsHasNoSensitiveFields(t *testing.T) {
	sanitized := &ReceiverSanitizedSettings{}
	
	// Marshal to JSON and check structure
	jsonBytes, _ := json.Marshal(sanitized)
	
	// Unmarshal into a map to check field names
	var fieldMap map[string]interface{}
	json.Unmarshal(jsonBytes, &fieldMap)

	// List of fields that should NEVER exist in sanitized settings
	forbiddenFields := []string{
		"bindPhrase",
		"modelMatch",
		"modelId",
		"uid",
		"extra",
	}

	// Note: The check here is structural - the type itself doesn't have these fields
	// This test documents the expectation
	for _, field := range forbiddenFields {
		if _, exists := fieldMap[field]; exists {
			t.Errorf("SECURITY VIOLATION: ReceiverSanitizedSettings should not have field '%s'", field)
		}
	}
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func containsIgnoreCase(s, substr string) bool {
	return contains(toLower(s), toLower(substr))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		result[i] = c
	}
	return string(result)
}

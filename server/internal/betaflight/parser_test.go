package betaflight

import (
	"testing"

	"github.com/johnrirwin/flyingforge/internal/models"
)

func TestParseStatus_EmptyInput(t *testing.T) {
	parser := NewParser()
	result := parser.Parse("")

	if result.ParseStatus != models.ParseStatusFailed {
		t.Errorf("Expected ParseStatusFailed for empty input, got %v", result.ParseStatus)
	}

	if len(result.ParseWarnings) == 0 {
		t.Error("Expected warnings for empty input")
	}
}

func TestParseStatus_NoTuningData(t *testing.T) {
	parser := NewParser()
	// Input with no PID or filter data
	cliDump := `# Betaflight / STM32F405 (S405) 4.4.2 Jun 1 2023 / 12:34:56 (1234567) MSP API: 1.45
# board_name MATEKF405
# mcu STM32F405
set some_random_setting = 123
`
	result := parser.Parse(cliDump)

	if result.ParseStatus != models.ParseStatusFailed {
		t.Errorf("Expected ParseStatusFailed for no tuning data, got %v", result.ParseStatus)
	}

	if result.ParsedTuning.PIDs != nil {
		t.Error("Expected PIDs to be nil when no PID data found")
	}

	if result.ParsedTuning.Filters != nil {
		t.Error("Expected Filters to be nil when no filter data found")
	}

	if result.ParsedTuning.MotorMixer != nil {
		t.Error("Expected MotorMixer to be nil when no motor/mixer data found")
	}

	if result.ParsedTuning.Features != nil {
		t.Error("Expected Features to be nil when no feature data found")
	}

	if result.ParsedTuning.Misc != nil {
		t.Error("Expected Misc to be nil when no misc data found")
	}
}

func TestParseStatus_PartialWithFiltersOnly(t *testing.T) {
	parser := NewParser()
	// Input with only filter data
	cliDump := `# Betaflight / STM32F405 (S405) 4.4.2 Jun 1 2023 / 12:34:56 (1234567) MSP API: 1.45
set gyro_lpf1_static_hz = 250
set dterm_lpf1_static_hz = 100
`
	result := parser.Parse(cliDump)

	if result.ParseStatus != models.ParseStatusPartial {
		t.Errorf("Expected ParseStatusPartial for filters only, got %v", result.ParseStatus)
	}

	if result.ParsedTuning.Filters == nil {
		t.Error("Expected Filters to be set when filter data found")
	}

	if result.ParsedTuning.PIDs != nil {
		t.Error("Expected PIDs to be nil when no PID data found")
	}
}

func TestParseStatus_PartialWithPIDsOnly(t *testing.T) {
	parser := NewParser()
	// Input with only PID data (no firmware info)
	cliDump := `set p_roll = 45
set i_roll = 50
set d_roll = 35
set p_pitch = 48
set i_pitch = 52
set d_pitch = 38
set p_yaw = 65
set i_yaw = 70
set d_yaw = 0
`
	result := parser.Parse(cliDump)

	if result.ParseStatus != models.ParseStatusPartial {
		t.Errorf("Expected ParseStatusPartial for PIDs without firmware, got %v", result.ParseStatus)
	}

	if result.ParsedTuning.PIDs == nil {
		t.Error("Expected PIDs to be set when PID data found")
	} else {
		if result.ParsedTuning.PIDs.Roll.P != 45 {
			t.Errorf("Expected Roll P=45, got %d", result.ParsedTuning.PIDs.Roll.P)
		}
	}
}

func TestParseStatus_Success(t *testing.T) {
	parser := NewParser()
	// Input with firmware info and PID data
	cliDump := `# Betaflight / STM32F405 (S405) 4.4.2 Jun 1 2023 / 12:34:56 (1234567) MSP API: 1.45
# board_name MATEKF405
set p_roll = 45
set i_roll = 50
set d_roll = 35
set p_pitch = 48
set i_pitch = 52
set d_pitch = 38
set p_yaw = 65
set i_yaw = 70
set d_yaw = 0
`
	result := parser.Parse(cliDump)

	if result.ParseStatus != models.ParseStatusSuccess {
		t.Errorf("Expected ParseStatusSuccess for firmware + PIDs, got %v", result.ParseStatus)
	}

	if result.FirmwareName != models.FirmwareBetaflight {
		t.Errorf("Expected FirmwareBetaflight, got %v", result.FirmwareName)
	}

	if result.ParsedTuning.PIDs == nil {
		t.Error("Expected PIDs to be set")
	}
}

func TestParseStatus_OnlyMotorMixerSettings(t *testing.T) {
	parser := NewParser()
	// Input with only motor mixer settings
	cliDump := `set motor_pwm_protocol = DSHOT600
set gyro_sync_denom = 1
set pid_process_denom = 2
`
	result := parser.Parse(cliDump)

	if result.ParseStatus != models.ParseStatusFailed {
		t.Errorf("Expected ParseStatusFailed (motor mixer not counted for parse status), got %v", result.ParseStatus)
	}

	if result.ParsedTuning.MotorMixer == nil {
		t.Error("Expected MotorMixer to be set when motor/mixer data found")
	}

	// MotorMixer is set but doesn't count toward success/partial status
	if result.ParsedTuning.PIDs != nil {
		t.Error("Expected PIDs to be nil")
	}
	if result.ParsedTuning.Filters != nil {
		t.Error("Expected Filters to be nil")
	}
}

func TestParseStatus_OnlyFeatures(t *testing.T) {
	parser := NewParser()
	// Input with only features
	cliDump := `feature GPS
feature TELEMETRY
feature OSD
`
	result := parser.Parse(cliDump)

	if result.ParseStatus != models.ParseStatusFailed {
		t.Errorf("Expected ParseStatusFailed (features not counted for parse status), got %v", result.ParseStatus)
	}

	if result.ParsedTuning.Features == nil {
		t.Error("Expected Features to be set when feature data found")
	}

	// Features are set but don't count toward success/partial status
	if result.ParsedTuning.PIDs != nil {
		t.Error("Expected PIDs to be nil")
	}
	if result.ParsedTuning.Filters != nil {
		t.Error("Expected Filters to be nil")
	}
}

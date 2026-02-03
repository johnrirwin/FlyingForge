package betaflight

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/johnrirwin/flyingforge/internal/models"
)

// Parser parses Betaflight CLI dump output
type Parser struct {
	// Regex patterns for parsing
	versionPattern     *regexp.Regexp
	boardPattern       *regexp.Regexp
	mcuPattern         *regexp.Regexp
	pidPattern         *regexp.Regexp
	ratePattern        *regexp.Regexp
	filterPattern      *regexp.Regexp
	featurePattern     *regexp.Regexp
	setPattern         *regexp.Regexp
	profilePattern     *regexp.Regexp
	rateprofilePattern *regexp.Regexp
}

// ParseResult contains the results of parsing a CLI dump
type ParseResult struct {
	FirmwareName    models.FCConfigFirmware
	FirmwareVersion string
	BoardTarget     string
	BoardName       string
	MCUType         string
	ParseStatus     models.ParseStatus
	ParseWarnings   []string
	ParsedTuning    *models.ParsedTuning
}

// NewParser creates a new Betaflight CLI parser
func NewParser() *Parser {
	return &Parser{
		versionPattern:     regexp.MustCompile(`#\s*version\s*$|#\s*Betaflight\s*/\s*(\S+)\s+(\d+\.\d+\.\d+)`),
		boardPattern:       regexp.MustCompile(`#\s*board_name\s+(\S+)|board_name\s+(\S+)`),
		mcuPattern:         regexp.MustCompile(`#\s*mcu\s+(\S+)|mcu\s+(\S+)`),
		pidPattern:         regexp.MustCompile(`set\s+(p_|i_|d_|f_)?(roll|pitch|yaw)\s*=\s*(\d+)`),
		ratePattern:        regexp.MustCompile(`set\s+(rc_rates?|rates?|rc_expo|super_rate)\s*=\s*(.+)`),
		filterPattern:      regexp.MustCompile(`set\s+(gyro_|dterm_|dyn_)?(\S*filter\S*|notch\S*)\s*=\s*(.+)`),
		featurePattern:     regexp.MustCompile(`feature\s+(-?)(\S+)`),
		setPattern:         regexp.MustCompile(`set\s+(\S+)\s*=\s*(.+)`),
		profilePattern:     regexp.MustCompile(`profile\s+(\d+)`),
		rateprofilePattern: regexp.MustCompile(`rateprofile\s+(\d+)`),
	}
}

// Parse parses a Betaflight CLI dump and returns structured data
func (p *Parser) Parse(cliDump string) *ParseResult {
	result := &ParseResult{
		FirmwareName:  models.FirmwareUnknown,
		ParseStatus:   models.ParseStatusPartial,
		ParseWarnings: []string{},
		ParsedTuning: &models.ParsedTuning{
			PIDProfiles:  make([]models.PIDProfile, 0),
			RateProfiles: make([]models.RateProfile, 0),
		},
	}

	if cliDump == "" {
		result.ParseStatus = models.ParseStatusFailed
		result.ParseWarnings = append(result.ParseWarnings, "Empty CLI dump")
		return result
	}

	lines := strings.Split(cliDump, "\n")

	// Parse metadata first
	p.parseMetadata(lines, result)

	// Parse tuning data
	p.parsePIDProfiles(lines, result)
	p.parseRateProfiles(lines, result)
	p.parseFilters(lines, result)
	p.parseMotorMixer(lines, result)
	p.parseFeatures(lines, result)
	p.parseMiscSettings(lines, result)

	// Set active profile data as the main PIDs/Rates
	if len(result.ParsedTuning.PIDProfiles) > 0 {
		idx := result.ParsedTuning.ActivePIDProfile
		if idx >= 0 && idx < len(result.ParsedTuning.PIDProfiles) {
			profile := result.ParsedTuning.PIDProfiles[idx]
			result.ParsedTuning.PIDs = &profile
		}
	}

	if len(result.ParsedTuning.RateProfiles) > 0 {
		idx := result.ParsedTuning.ActiveRateProfile
		if idx >= 0 && idx < len(result.ParsedTuning.RateProfiles) {
			profile := result.ParsedTuning.RateProfiles[idx]
			result.ParsedTuning.Rates = &profile
		}
	}

	// Determine overall parse status
	if result.FirmwareName != models.FirmwareUnknown && result.ParsedTuning.PIDs != nil {
		result.ParseStatus = models.ParseStatusSuccess
	} else if result.ParsedTuning.PIDs != nil || result.ParsedTuning.Filters != nil {
		result.ParseStatus = models.ParseStatusPartial
	} else {
		result.ParseStatus = models.ParseStatusFailed
		result.ParseWarnings = append(result.ParseWarnings, "Could not parse any tuning data")
	}

	return result
}

// parseMetadata extracts firmware info, board details, etc.
func (p *Parser) parseMetadata(lines []string, result *ParseResult) {
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for Betaflight version
		if strings.Contains(line, "Betaflight") {
			result.FirmwareName = models.FirmwareBetaflight
			matches := regexp.MustCompile(`Betaflight\s*/\s*(\S+)\s+(\d+\.\d+\.\d+)`).FindStringSubmatch(line)
			if len(matches) >= 3 {
				result.BoardTarget = matches[1]
				result.FirmwareVersion = matches[2]
			}
		}

		// Check for INAV
		if strings.Contains(line, "INAV") {
			result.FirmwareName = models.FirmwareINAV
			matches := regexp.MustCompile(`INAV\s*/\s*(\S+)\s+(\d+\.\d+\.\d+)`).FindStringSubmatch(line)
			if len(matches) >= 3 {
				result.BoardTarget = matches[1]
				result.FirmwareVersion = matches[2]
			}
		}

		// Board name
		if strings.HasPrefix(line, "board_name") || strings.HasPrefix(line, "# board_name") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				result.BoardName = parts[len(parts)-1]
			}
		}

		// MCU type
		if strings.Contains(line, "# mcu ") || strings.HasPrefix(line, "mcu ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				result.MCUType = parts[len(parts)-1]
			}
		}
	}
}

// parsePIDProfiles extracts PID tuning values
func (p *Parser) parsePIDProfiles(lines []string, result *ParseResult) {
	// Initialize default profile
	profiles := make(map[int]*models.PIDProfile)
	currentProfile := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Track profile switches
		if matches := p.profilePattern.FindStringSubmatch(line); matches != nil {
			if idx, err := strconv.Atoi(matches[1]); err == nil {
				currentProfile = idx
			}
		}

		// Initialize profile if needed
		if profiles[currentProfile] == nil {
			profiles[currentProfile] = &models.PIDProfile{
				ProfileIndex: currentProfile,
			}
		}

		// Parse PID set commands
		if strings.HasPrefix(line, "set ") {
			p.parsePIDLine(line, profiles[currentProfile])
		}
	}

	// Convert map to slice
	for i := 0; i <= 3; i++ {
		if profile, ok := profiles[i]; ok {
			result.ParsedTuning.PIDProfiles = append(result.ParsedTuning.PIDProfiles, *profile)
		}
	}
}

// parsePIDLine parses a single PID set command
func (p *Parser) parsePIDLine(line string, profile *models.PIDProfile) {
	// set p_roll = 45
	// set d_pitch = 32
	// set i_yaw = 85
	// set f_roll = 120

	pidRe := regexp.MustCompile(`set\s+(p|i|d|f)_(roll|pitch|yaw)\s*=\s*(\d+)`)
	if matches := pidRe.FindStringSubmatch(line); len(matches) >= 4 {
		value, _ := strconv.Atoi(matches[3])
		switch matches[2] {
		case "roll":
			switch matches[1] {
			case "p":
				profile.Roll.P = value
			case "i":
				profile.Roll.I = value
			case "d":
				profile.Roll.D = value
			case "f":
				profile.Roll.FF = value
			}
		case "pitch":
			switch matches[1] {
			case "p":
				profile.Pitch.P = value
			case "i":
				profile.Pitch.I = value
			case "d":
				profile.Pitch.D = value
			case "f":
				profile.Pitch.FF = value
			}
		case "yaw":
			switch matches[1] {
			case "p":
				profile.Yaw.P = value
			case "i":
				profile.Yaw.I = value
			case "d":
				profile.Yaw.D = value
			case "f":
				profile.Yaw.FF = value
			}
		}
	}

	// Parse additional PID settings
	p.parseAdditionalPIDSettings(line, profile)
}

// parseAdditionalPIDSettings parses settings like anti_gravity, d_min, etc.
func (p *Parser) parseAdditionalPIDSettings(line string, profile *models.PIDProfile) {
	if val := p.extractSetInt(line, "anti_gravity_gain"); val != nil {
		profile.AntiGravityGain = *val
	}
	if val := p.extractSetString(line, "anti_gravity_mode"); val != "" {
		profile.AntiGravityMode = val
	}
	if val := p.extractSetInt(line, "d_min_roll"); val != nil {
		profile.DMinRoll = *val
	}
	if val := p.extractSetInt(line, "d_min_pitch"); val != nil {
		profile.DMinPitch = *val
	}
	if val := p.extractSetInt(line, "d_min_yaw"); val != nil {
		profile.DMinYaw = *val
	}
	if val := p.extractSetInt(line, "d_min_gain"); val != nil {
		profile.DMinGain = *val
	}
	if val := p.extractSetInt(line, "d_min_advance"); val != nil {
		profile.DMinAdvance = *val
	}
	if val := p.extractSetString(line, "iterm_relax"); val != "" {
		profile.ITermRelax = val
	}
	if val := p.extractSetString(line, "iterm_relax_type"); val != "" {
		profile.ITermRelaxType = val
	}
	if val := p.extractSetInt(line, "iterm_relax_cutoff"); val != nil {
		profile.ITermRelaxCutoff = *val
	}
	if val := p.extractSetInt(line, "tpa_rate"); val != nil {
		profile.TPARate = *val
	}
	if val := p.extractSetInt(line, "tpa_breakpoint"); val != nil {
		profile.TPABreakpoint = *val
	}
	if val := p.extractSetString(line, "tpa_mode"); val != "" {
		profile.TPAMode = val
	}
	if val := p.extractSetInt(line, "feedforward_transition"); val != nil {
		profile.FeedforwardTransition = *val
	}
	if val := p.extractSetInt(line, "feedforward_averaging"); val != nil {
		profile.FeedforwardAveraging = *val
	}
	if val := p.extractSetInt(line, "feedforward_smooth_factor"); val != nil {
		profile.FeedforwardSmooth = *val
	}
	if val := p.extractSetInt(line, "feedforward_jitter_factor"); val != nil {
		profile.FeedforwardJitterFactor = *val
	}
	if val := p.extractSetInt(line, "feedforward_boost"); val != nil {
		profile.FeedforwardBoost = *val
	}
}

// parseRateProfiles extracts rate settings
func (p *Parser) parseRateProfiles(lines []string, result *ParseResult) {
	profiles := make(map[int]*models.RateProfile)
	currentProfile := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Track rateprofile switches
		if matches := p.rateprofilePattern.FindStringSubmatch(line); matches != nil {
			if idx, err := strconv.Atoi(matches[1]); err == nil {
				currentProfile = idx
			}
		}

		// Initialize profile if needed
		if profiles[currentProfile] == nil {
			profiles[currentProfile] = &models.RateProfile{
				ProfileIndex: currentProfile,
			}
		}

		// Parse rate set commands
		if strings.HasPrefix(line, "set ") {
			p.parseRateLine(line, profiles[currentProfile])
		}
	}

	// Convert map to slice
	for i := 0; i <= 5; i++ {
		if profile, ok := profiles[i]; ok {
			result.ParsedTuning.RateProfiles = append(result.ParsedTuning.RateProfiles, *profile)
		}
	}
}

// parseRateLine parses a single rate set command
func (p *Parser) parseRateLine(line string, profile *models.RateProfile) {
	if val := p.extractSetString(line, "rates_type"); val != "" {
		profile.RateType = val
	}

	// RC rates (comma separated: roll,pitch,yaw)
	if val := p.extractSetString(line, "roll_rc_rate"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			profile.RCRates.Roll = v
		}
	}
	if val := p.extractSetString(line, "pitch_rc_rate"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			profile.RCRates.Pitch = v
		}
	}
	if val := p.extractSetString(line, "yaw_rc_rate"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			profile.RCRates.Yaw = v
		}
	}

	// Super rates
	if val := p.extractSetString(line, "roll_srate"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			profile.SuperRates.Roll = v
		}
	}
	if val := p.extractSetString(line, "pitch_srate"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			profile.SuperRates.Pitch = v
		}
	}
	if val := p.extractSetString(line, "yaw_srate"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			profile.SuperRates.Yaw = v
		}
	}

	// RC expo
	if val := p.extractSetString(line, "roll_expo"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			profile.RCExpo.Roll = v
		}
	}
	if val := p.extractSetString(line, "pitch_expo"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			profile.RCExpo.Pitch = v
		}
	}
	if val := p.extractSetString(line, "yaw_expo"); val != "" {
		if v, err := strconv.Atoi(val); err == nil {
			profile.RCExpo.Yaw = v
		}
	}

	// Throttle settings
	if val := p.extractSetInt(line, "thr_mid"); val != nil {
		profile.ThrottleMid = *val
	}
	if val := p.extractSetInt(line, "thr_expo"); val != nil {
		profile.ThrottleExpo = *val
	}
	if val := p.extractSetString(line, "throttle_limit_type"); val != "" {
		profile.ThrottleLimitType = val
	}
	if val := p.extractSetInt(line, "throttle_limit_percent"); val != nil {
		profile.ThrottleLimitPercent = *val
	}
}

// parseFilters extracts filter settings
func (p *Parser) parseFilters(lines []string, result *ParseResult) {
	filters := &models.FilterSettings{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "set ") {
			continue
		}

		// Gyro lowpass 1
		if val := p.extractSetInt(line, "gyro_lpf1_static_hz"); val != nil {
			filters.GyroLowpassHz = *val
			filters.GyroLowpassEnabled = *val > 0
		}
		if val := p.extractSetString(line, "gyro_lpf1_type"); val != "" {
			filters.GyroLowpassType = val
		}

		// Gyro lowpass 2
		if val := p.extractSetInt(line, "gyro_lpf2_static_hz"); val != nil {
			filters.GyroLowpass2Hz = *val
			filters.GyroLowpass2Enabled = *val > 0
		}
		if val := p.extractSetString(line, "gyro_lpf2_type"); val != "" {
			filters.GyroLowpass2Type = val
		}

		// Dynamic gyro lowpass
		if val := p.extractSetInt(line, "gyro_lpf1_dyn_min_hz"); val != nil {
			filters.GyroDynLowpassMinHz = *val
			filters.GyroDynLowpassEnabled = *val > 0
		}
		if val := p.extractSetInt(line, "gyro_lpf1_dyn_max_hz"); val != nil {
			filters.GyroDynLowpassMaxHz = *val
		}

		// Gyro notch 1
		if val := p.extractSetInt(line, "gyro_notch1_hz"); val != nil {
			filters.GyroNotch1Hz = *val
			filters.GyroNotch1Enabled = *val > 0
		}
		if val := p.extractSetInt(line, "gyro_notch1_cutoff"); val != nil {
			filters.GyroNotch1Cutoff = *val
		}

		// Gyro notch 2
		if val := p.extractSetInt(line, "gyro_notch2_hz"); val != nil {
			filters.GyroNotch2Hz = *val
			filters.GyroNotch2Enabled = *val > 0
		}
		if val := p.extractSetInt(line, "gyro_notch2_cutoff"); val != nil {
			filters.GyroNotch2Cutoff = *val
		}

		// D-term lowpass 1
		if val := p.extractSetInt(line, "dterm_lpf1_static_hz"); val != nil {
			filters.DTermLowpassHz = *val
			filters.DTermLowpassEnabled = *val > 0
		}
		if val := p.extractSetString(line, "dterm_lpf1_type"); val != "" {
			filters.DTermLowpassType = val
		}

		// D-term lowpass 2
		if val := p.extractSetInt(line, "dterm_lpf2_static_hz"); val != nil {
			filters.DTermLowpass2Hz = *val
			filters.DTermLowpass2Enabled = *val > 0
		}
		if val := p.extractSetString(line, "dterm_lpf2_type"); val != "" {
			filters.DTermLowpass2Type = val
		}

		// Dynamic D-term lowpass
		if val := p.extractSetInt(line, "dterm_lpf1_dyn_min_hz"); val != nil {
			filters.DTermDynLowpassMinHz = *val
			filters.DTermDynLowpassEnabled = *val > 0
		}
		if val := p.extractSetInt(line, "dterm_lpf1_dyn_max_hz"); val != nil {
			filters.DTermDynLowpassMaxHz = *val
		}

		// D-term notch
		if val := p.extractSetInt(line, "dterm_notch_hz"); val != nil {
			filters.DTermNotchHz = *val
			filters.DTermNotchEnabled = *val > 0
		}
		if val := p.extractSetInt(line, "dterm_notch_cutoff"); val != nil {
			filters.DTermNotchCutoff = *val
		}

		// RPM filter
		if val := p.extractSetInt(line, "rpm_filter_harmonics"); val != nil {
			filters.RPMFilterHarmonics = *val
			filters.RPMFilterEnabled = *val > 0
		}
		if val := p.extractSetInt(line, "rpm_filter_min_hz"); val != nil {
			filters.RPMFilterMinHz = *val
		}
		if val := p.extractSetInt(line, "rpm_filter_fade_range_hz"); val != nil {
			filters.RPMFilterFadeRange = *val
		}
		if val := p.extractSetInt(line, "rpm_filter_q"); val != nil {
			filters.RPMFilterQFactor = *val
		}

		// Dynamic notch
		if val := p.extractSetInt(line, "dyn_notch_count"); val != nil {
			filters.DynNotchCount = *val
			filters.DynNotchEnabled = *val > 0
		}
		if val := p.extractSetInt(line, "dyn_notch_q"); val != nil {
			filters.DynNotchQ = *val
		}
		if val := p.extractSetInt(line, "dyn_notch_min_hz"); val != nil {
			filters.DynNotchMinHz = *val
		}
		if val := p.extractSetInt(line, "dyn_notch_max_hz"); val != nil {
			filters.DynNotchMaxHz = *val
		}
	}

	result.ParsedTuning.Filters = filters
}

// parseMotorMixer extracts motor and mixer settings
func (p *Parser) parseMotorMixer(lines []string, result *ParseResult) {
	mixer := &models.MotorMixerConfig{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "set ") {
			continue
		}

		if val := p.extractSetString(line, "motor_pwm_protocol"); val != "" {
			mixer.MotorProtocol = val
		}
		if val := p.extractSetInt(line, "motor_pwm_rate"); val != nil {
			mixer.MotorPWMRate = *val
		}
		if val := p.extractSetInt(line, "dshot_idle_value"); val != nil {
			mixer.DigitalIdlePercent = *val
		}
		if val := p.extractSetInt(line, "motor_poles"); val != nil {
			mixer.MotorPoles = *val
		}
		if val := p.extractSetInt(line, "gyro_sync_denom"); val != nil {
			mixer.GyroSyncDenom = *val
		}
		if val := p.extractSetInt(line, "pid_process_denom"); val != nil {
			mixer.PIDLoopDenom = *val
		}
		if val := p.extractSetString(line, "dshot_bidir"); val != "" {
			mixer.DShotBidir = val == "ON" || val == "1"
		}
		if val := p.extractSetString(line, "dshot_bitbang"); val != "" {
			mixer.DShotBitbang = val
		}
	}

	// Calculate Hz values
	baseGyroHz := 8000 // Most modern FCs run at 8kHz
	if mixer.GyroSyncDenom > 0 {
		mixer.GyroHz = baseGyroHz / mixer.GyroSyncDenom
	}
	if mixer.PIDLoopDenom > 0 && mixer.GyroHz > 0 {
		mixer.PIDHz = mixer.GyroHz / mixer.PIDLoopDenom
	}

	result.ParsedTuning.MotorMixer = mixer
}

// parseFeatures extracts enabled features
func (p *Parser) parseFeatures(lines []string, result *ParseResult) {
	features := &models.FeatureFlags{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "feature ") {
			continue
		}

		// feature GPS or feature -GPS
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		featureName := parts[1]
		enabled := true
		if strings.HasPrefix(featureName, "-") {
			enabled = false
			featureName = featureName[1:]
		}

		switch strings.ToUpper(featureName) {
		case "GPS":
			features.GPS = enabled
		case "TELEMETRY":
			features.Telemetry = enabled
		case "OSD":
			features.OSD = enabled
		case "LED_STRIP":
			features.LED_STRIP = enabled
		case "AIRMODE":
			features.Airmode = enabled
		case "ANTI_GRAVITY":
			features.AntiGravity = enabled
		case "DYNAMIC_FILTER":
			features.DynamicFilter = enabled
		case "RPM_FILTER":
			features.RPMFilter = enabled
		}
	}

	result.ParsedTuning.Features = features
}

// parseMiscSettings extracts misc settings
func (p *Parser) parseMiscSettings(lines []string, result *ParseResult) {
	misc := &models.MiscSettings{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "set ") {
			continue
		}

		if val := p.extractSetString(line, "name"); val != "" && val != "-" {
			misc.Name = val
		}
		if val := p.extractSetString(line, "crash_recovery"); val != "" {
			misc.CrashRecovery = val
		}
		if val := p.extractSetInt(line, "gyro_calib_noise_limit"); val != nil {
			misc.GyroCalibNoise = *val
		}
		if val := p.extractSetInt(line, "vbat_min_cell_voltage"); val != nil {
			misc.VBatMinCellVoltage = *val
		}
		if val := p.extractSetInt(line, "vbat_max_cell_voltage"); val != nil {
			misc.VBatMaxCellVoltage = *val
		}
		if val := p.extractSetInt(line, "vbat_warning_cell_voltage"); val != nil {
			misc.VBatWarningCellVoltage = *val
		}
	}

	result.ParsedTuning.Misc = misc
}

// extractSetInt extracts an integer value from a set command
func (p *Parser) extractSetInt(line, key string) *int {
	pattern := regexp.MustCompile(`set\s+` + regexp.QuoteMeta(key) + `\s*=\s*(\d+)`)
	if matches := pattern.FindStringSubmatch(line); len(matches) >= 2 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			return &val
		}
	}
	return nil
}

// extractSetString extracts a string value from a set command
func (p *Parser) extractSetString(line, key string) string {
	pattern := regexp.MustCompile(`set\s+` + regexp.QuoteMeta(key) + `\s*=\s*(\S+)`)
	if matches := pattern.FindStringSubmatch(line); len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

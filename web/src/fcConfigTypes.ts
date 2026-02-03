// FC Config Types - Matching server/internal/models/fc_config.go

export type FCConfigFirmware = 'betaflight' | 'inav' | 'ardupilot' | 'unknown';
export type ParseStatus = 'success' | 'partial' | 'failed';

// PID values for a single axis
export interface AxisPID {
  p: number;
  i: number;
  d: number;
  ff?: number;
}

// Rate values for all axes
export interface RateAxisValues {
  roll: number;
  pitch: number;
  yaw: number;
}

// PID Profile
export interface PIDProfile {
  profileIndex: number;
  profileName?: string;
  roll: AxisPID;
  pitch: AxisPID;
  yaw: AxisPID;
  level?: AxisPID;
  
  // Additional PID settings
  antiGravityGain?: number;
  antiGravityMode?: string;
  feedforwardTransition?: number;
  feedforwardAveraging?: number;
  feedforwardSmooth?: number;
  feedforwardJitterFactor?: number;
  feedforwardBoost?: number;
  
  // D-term settings
  dMinRoll?: number;
  dMinPitch?: number;
  dMinYaw?: number;
  dMinGain?: number;
  dMinAdvance?: number;
  
  // I-term settings
  iTermRelax?: string;
  iTermRelaxType?: string;
  iTermRelaxCutoff?: number;
  
  // TPA
  tpaRate?: number;
  tpaBreakpoint?: number;
  tpaMode?: string;
}

// Rate Profile
export interface RateProfile {
  profileIndex: number;
  profileName?: string;
  rateType?: string;
  
  rcRates: RateAxisValues;
  superRates: RateAxisValues;
  rcExpo: RateAxisValues;
  
  // For ACTUAL rate type
  centerSensitivity?: RateAxisValues;
  maxRate?: RateAxisValues;
  
  // Throttle
  throttleMid?: number;
  throttleExpo?: number;
  throttleLimitType?: string;
  throttleLimitPercent?: number;
}

// Filter Settings
export interface FilterSettings {
  // Gyro lowpass
  gyroLowpassEnabled: boolean;
  gyroLowpassHz?: number;
  gyroLowpassType?: string;
  gyroLowpass2Enabled: boolean;
  gyroLowpass2Hz?: number;
  gyroLowpass2Type?: string;
  
  // Dynamic gyro lowpass
  gyroDynLowpassEnabled: boolean;
  gyroDynLowpassMinHz?: number;
  gyroDynLowpassMaxHz?: number;
  
  // Gyro notch
  gyroNotch1Enabled: boolean;
  gyroNotch1Hz?: number;
  gyroNotch1Cutoff?: number;
  gyroNotch2Enabled: boolean;
  gyroNotch2Hz?: number;
  gyroNotch2Cutoff?: number;
  
  // D-term lowpass
  dtermLowpassEnabled: boolean;
  dtermLowpassHz?: number;
  dtermLowpassType?: string;
  dtermLowpass2Enabled: boolean;
  dtermLowpass2Hz?: number;
  dtermLowpass2Type?: string;
  
  // Dynamic D-term
  dtermDynLowpassEnabled: boolean;
  dtermDynLowpassMinHz?: number;
  dtermDynLowpassMaxHz?: number;
  
  // D-term notch
  dtermNotchEnabled: boolean;
  dtermNotchHz?: number;
  dtermNotchCutoff?: number;
  
  // RPM filter
  rpmFilterEnabled: boolean;
  rpmFilterHarmonics?: number;
  rpmFilterMinHz?: number;
  rpmFilterFadeRange?: number;
  rpmFilterQFactor?: number;
  
  // Dynamic notch
  dynNotchEnabled: boolean;
  dynNotchCount?: number;
  dynNotchQ?: number;
  dynNotchMinHz?: number;
  dynNotchMaxHz?: number;
}

// Motor/Mixer config
export interface MotorMixerConfig {
  motorProtocol?: string;
  motorPwmRate?: number;
  motorIdlePercent?: number;
  digitalIdlePercent?: number;
  motorPoles?: number;
  mixerType?: string;
  gyroSyncDenom?: number;
  pidLoopDenom?: number;
  gyroHz?: number;
  pidHz?: number;
  dshotBidir?: boolean;
  dshotBitbang?: string;
}

// Feature flags
export interface FeatureFlags {
  gps: boolean;
  telemetry: boolean;
  osd: boolean;
  ledStrip: boolean;
  airmode: boolean;
  antiGravity: boolean;
  dynamicFilter: boolean;
  rpmFilter: boolean;
}

// Misc settings
export interface MiscSettings {
  name?: string;
  crashRecovery?: string;
  gyroCalibNoise?: number;
  accCalibX?: number;
  accCalibY?: number;
  accCalibZ?: number;
  vbatMinCellVoltage?: number;
  vbatMaxCellVoltage?: number;
  vbatWarningCellVoltage?: number;
}

// Parsed tuning data
export interface ParsedTuning {
  pids?: PIDProfile;
  rates?: RateProfile;
  filters?: FilterSettings;
  motorMixer?: MotorMixerConfig;
  features?: FeatureFlags;
  misc?: MiscSettings;
  pidProfiles?: PIDProfile[];
  rateProfiles?: RateProfile[];
  activePidProfile: number;
  activeRateProfile: number;
}

// Flight Controller Config (saved CLI dump)
export interface FlightControllerConfig {
  id: string;
  userId?: string;
  inventoryItemId: string;
  name: string;
  notes?: string;
  rawCliDump: string;
  firmwareName: FCConfigFirmware;
  firmwareVersion?: string;
  boardTarget?: string;
  boardName?: string;
  mcuType?: string;
  parseStatus: ParseStatus;
  parseWarnings?: string[];
  parsedTuning?: ParsedTuning;
  createdAt: string;
  updatedAt: string;
}

// Aircraft Tuning Snapshot
export interface AircraftTuningSnapshot {
  id: string;
  aircraftId: string;
  flightControllerId?: string;
  flightControllerConfigId?: string;
  firmwareName: FCConfigFirmware;
  firmwareVersion?: string;
  boardTarget?: string;
  boardName?: string;
  tuningData: string; // JSON string
  parseStatus: ParseStatus;
  parseWarnings?: string[];
  notes?: string;
  createdAt: string;
  updatedAt: string;
}

// API Request/Response types
export interface SaveFCConfigParams {
  inventoryItemId: string;
  name?: string;
  notes?: string;
  rawCliDump: string;
}

export interface UpdateFCConfigParams {
  name?: string;
  notes?: string;
}

export interface FCConfigListParams {
  inventoryItemId?: string;
  limit?: number;
  offset?: number;
}

export interface FCConfigListResponse {
  configs: FlightControllerConfig[];
  totalCount: number;
}

export interface AircraftTuningResponse {
  aircraftId: string;
  hasTuning: boolean;
  firmwareName?: FCConfigFirmware;
  firmwareVersion?: string;
  boardTarget?: string;
  boardName?: string;
  tuning?: ParsedTuning;
  snapshotId?: string;
  snapshotDate?: string;
  parseStatus?: ParseStatus;
  parseWarnings?: string[];
  hasDiffBackup?: boolean;
  diffBackup?: string;
}

export interface CreateTuningSnapshotParams {
  rawCliDump: string;
  diffBackup?: string;
  notes?: string;
}

export interface TuningSnapshotsListResponse {
  snapshots: AircraftTuningSnapshot[];
  total_count: number;
}

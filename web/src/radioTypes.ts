// Radio types matching the Go server schema

// Radio manufacturer
export type RadioManufacturer = 'RadioMaster' | 'FrSky' | 'Jumper' | 'TBS';

// Firmware family
export type FirmwareFamily = 'EdgeTX' | 'OpenTX';

// Backup type
export type BackupType = 'edgetx-models' | 'radio-firmware' | 'sd-card-pack' | 'full-backup' | 'other';

export const BACKUP_TYPES: { value: BackupType; label: string; description: string }[] = [
  { value: 'edgetx-models', label: 'EdgeTX Models', description: 'Model configurations and settings' },
  { value: 'radio-firmware', label: 'Radio Firmware', description: 'Firmware binary file' },
  { value: 'sd-card-pack', label: 'SD Card Pack', description: 'Full SD card contents' },
  { value: 'full-backup', label: 'Full Backup', description: 'Complete radio backup' },
  { value: 'other', label: 'Other', description: 'Other backup type' },
];

export const FIRMWARE_FAMILIES: { value: FirmwareFamily; label: string }[] = [
  { value: 'EdgeTX', label: 'EdgeTX' },
  { value: 'OpenTX', label: 'OpenTX' },
];

// Radio model (from the predefined list)
export interface RadioModel {
  id: string;
  manufacturer: RadioManufacturer;
  model: string;
  displayName: string;
}

// User's radio
export interface Radio {
  id: string;
  userId?: string;
  manufacturer: RadioManufacturer;
  model: string;
  firmwareFamily?: FirmwareFamily;
  notes?: string;
  createdAt: string;
  updatedAt: string;
}

// Radio backup
export interface RadioBackup {
  id: string;
  radioId: string;
  backupName: string;
  backupType: BackupType;
  fileName: string;
  fileSize: number;
  checksum?: string;
  createdAt: string;
}

// Create radio params
export interface CreateRadioParams {
  manufacturer: RadioManufacturer;
  model: string;
  firmwareFamily?: FirmwareFamily;
  notes?: string;
}

// Update radio params
export interface UpdateRadioParams {
  firmwareFamily?: FirmwareFamily;
  notes?: string;
}

// Create backup params
export interface CreateRadioBackupParams {
  backupName: string;
  backupType: BackupType;
  file: File;
}

// Radio list params
export interface RadioListParams {
  limit?: number;
  offset?: number;
}

// Radio list response
export interface RadioListResponse {
  radios: Radio[];
  totalCount: number;
}

// Radio models response
export interface RadioModelsResponse {
  models: RadioModel[];
}

// Backup list params
export interface RadioBackupListParams {
  limit?: number;
  offset?: number;
}

// Backup list response
export interface RadioBackupListResponse {
  backups: RadioBackup[];
  totalCount: number;
}

// Helper function to format file size
export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

// Helper function to get backup type label
export function getBackupTypeLabel(type: BackupType): string {
  const found = BACKUP_TYPES.find(t => t.value === type);
  return found?.label || type;
}

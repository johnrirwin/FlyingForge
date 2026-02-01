// Aircraft types matching the Go server schema
import { EquipmentCategory, InventoryItem, AddInventoryParams } from './equipmentTypes';

// Aircraft types
export type AircraftType = 'racing' | 'freestyle' | 'long_range' | 'cinematic' | 'tiny_whoop' | 'fixed_wing' | 'other';

export const AIRCRAFT_TYPES: { value: AircraftType; label: string; icon: string }[] = [
  { value: 'racing', label: 'Racing', icon: '🏎️' },
  { value: 'freestyle', label: 'Freestyle', icon: '🪂' },
  { value: 'long_range', label: 'Long Range', icon: '📡' },
  { value: 'cinematic', label: 'Cinematic', icon: '🎬' },
  { value: 'tiny_whoop', label: 'Tiny Whoop', icon: '🐝' },
  { value: 'fixed_wing', label: 'Fixed Wing', icon: '✈️' },
  { value: 'other', label: 'Other', icon: '🚁' },
];

// Component categories for aircraft
export type ComponentCategory = 'fc' | 'esc' | 'elrs_module' | 'vtx' | 'motors' | 'camera' | 'frame' | 'props' | 'battery' | 'antenna';

export const COMPONENT_CATEGORIES: { value: ComponentCategory; label: string; equipmentCategory: EquipmentCategory }[] = [
  { value: 'fc', label: 'Flight Controller', equipmentCategory: 'flight_controllers' },
  { value: 'esc', label: 'ESC', equipmentCategory: 'esc' },
  { value: 'elrs_module', label: 'ELRS Receiver', equipmentCategory: 'receivers' },
  { value: 'vtx', label: 'Video Transmitter', equipmentCategory: 'vtx' },
  { value: 'motors', label: 'Motors', equipmentCategory: 'motors' },
  { value: 'camera', label: 'Camera', equipmentCategory: 'cameras' },
  { value: 'frame', label: 'Frame', equipmentCategory: 'frames' },
  { value: 'props', label: 'Propellers', equipmentCategory: 'propellers' },
  { value: 'battery', label: 'Battery', equipmentCategory: 'batteries' },
  { value: 'antenna', label: 'Antenna', equipmentCategory: 'antennas' },
];

// Aircraft model
export interface Aircraft {
  id: string;
  userId: string;
  name: string;
  nickname?: string;
  type: AircraftType;
  hasImage: boolean;
  imageType?: string;
  description?: string;
  createdAt: string;
  updatedAt: string;
}

// Aircraft component
export interface AircraftComponent {
  aircraftId: string;
  category: ComponentCategory;
  inventoryItemId: string;
  notes?: string;
  createdAt: string;
  updatedAt: string;
  // Populated inventory item details
  inventoryItem?: InventoryItem;
}

// ELRS settings for an aircraft
export interface AircraftELRSSettings {
  aircraftId: string;
  settings: ELRSConfig;
  updatedAt: string;
}

// ELRS configuration structure
export interface ELRSConfig {
  bindingPhrase?: string;
  rate?: number;
  tlm?: number;
  power?: number;
  isAirFCUART?: boolean;
  isAirFCHALFDuplex?: boolean;
  isAirFCSerial?: boolean;
  deviceName?: string;
  wifiPassword?: string;
  wifiSSID?: string;
  customConfig?: Record<string, unknown>;
}

// Create aircraft params
export interface CreateAircraftParams {
  name: string;
  nickname?: string;
  type: AircraftType;
  description?: string;
}

// Update aircraft params
export interface UpdateAircraftParams {
  name?: string;
  nickname?: string;
  type?: AircraftType;
  description?: string;
}

// Set component params - supports auto-add gear
export interface SetComponentParams {
  category: ComponentCategory;
  inventoryItemId?: string;
  notes?: string;
  // For auto-add gear: create a new inventory item and assign it
  newGear?: AddInventoryParams;
}

// Set ELRS settings params
export interface SetELRSSettingsParams {
  settings: ELRSConfig;
}

// Aircraft list params
export interface AircraftListParams {
  type?: AircraftType;
  limit?: number;
  offset?: number;
}

// Aircraft list response
export interface AircraftListResponse {
  aircraft: Aircraft[];
  totalCount: number;
}

// Aircraft details response (full with components and ELRS)
export interface AircraftDetailsResponse {
  aircraft: Aircraft;
  components: AircraftComponent[];
  elrsSettings?: AircraftELRSSettings;
}

// Components response
export interface ComponentsResponse {
  components: AircraftComponent[];
  count: number;
}

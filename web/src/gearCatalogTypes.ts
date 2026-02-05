// Gear Catalog types for crowd-sourced gear definitions

// Gear types matching the Go backend
export type GearType =
  | 'motor'
  | 'esc'
  | 'fc'
  | 'aio'
  | 'frame'
  | 'vtx'
  | 'receiver'
  | 'antenna'
  | 'battery'
  | 'prop'
  | 'radio'
  | 'camera'
  | 'other';

export const GEAR_TYPES: { value: GearType; label: string }[] = [
  { value: 'motor', label: 'Motors' },
  { value: 'esc', label: 'ESCs' },
  { value: 'fc', label: 'Flight Controllers' },
  { value: 'aio', label: 'AIO (FC/ESC)' },
  { value: 'frame', label: 'Frames' },
  { value: 'vtx', label: 'Video Transmitters' },
  { value: 'receiver', label: 'Receivers' },
  { value: 'antenna', label: 'Antennas' },
  { value: 'battery', label: 'Batteries' },
  { value: 'prop', label: 'Propellers' },
  { value: 'radio', label: 'Radios' },
  { value: 'camera', label: 'Cameras' },
  { value: 'other', label: 'Other' },
];

// Catalog item status
export type CatalogItemStatus = 'active' | 'pending' | 'flagged' | 'rejected';

// Catalog item source
export type CatalogItemSource = 'user-submitted' | 'admin' | 'import' | 'migration';

// Drone types for "Best For" field
export type DroneType = 
  | 'freestyle'
  | 'long-range'
  | 'cinematic'
  | 'racing'
  | 'tiny-whoop'
  | 'cinewhoop'
  | 'micro'
  | 'toothpick'
  | 'x-class'
  | 'other';

export const DRONE_TYPES: { value: DroneType; label: string }[] = [
  { value: 'freestyle', label: 'Freestyle' },
  { value: 'long-range', label: 'Long Range' },
  { value: 'cinematic', label: 'Cinematic' },
  { value: 'racing', label: 'Racing' },
  { value: 'tiny-whoop', label: 'Tiny Whoop' },
  { value: 'cinewhoop', label: 'Cinewhoop' },
  { value: 'micro', label: 'Micro (2-3")' },
  { value: 'toothpick', label: 'Toothpick' },
  { value: 'x-class', label: 'X-Class' },
  { value: 'other', label: 'Other' },
];

// Gear catalog item - canonical gear definition
export interface GearCatalogItem {
  id: string;
  gearType: GearType;
  brand: string;
  model: string;
  variant?: string;
  specs?: Record<string, unknown>;
  bestFor?: DroneType[]; // What drone types this gear is best suited for
  source: CatalogItemSource;
  createdByUserId?: string;
  status: CatalogItemStatus;
  canonicalKey: string;
  imageUrl?: string;
  description?: string;
  usageCount: number;
  createdAt: string;
  updatedAt: string;
}

// Parameters for creating a catalog item
export interface CreateGearCatalogParams {
  gearType: GearType;
  brand: string;
  model: string;
  variant?: string;
  specs?: Record<string, unknown>;
  bestFor?: DroneType[]; // What drone types this gear is best suited for
  imageUrl?: string;
  description?: string;
}

// Search parameters for the catalog
export interface GearCatalogSearchParams {
  query?: string;
  gearType?: GearType;
  brand?: string;
  status?: CatalogItemStatus;
  limit?: number;
  offset?: number;
}

// Search response from the catalog
export interface GearCatalogSearchResponse {
  items: GearCatalogItem[];
  totalCount: number;
  query?: string;
}

// Response when creating a catalog item (may return existing)
export interface GearCatalogCreateResponse {
  item: GearCatalogItem;
  existing: boolean;
}

// Near match for duplicate detection
export interface NearMatch {
  item: GearCatalogItem;
  similarity: number;
}

// Response for near match detection
export interface NearMatchResponse {
  matches: NearMatch[];
}

// Parameters for checking near matches
export interface NearMatchParams {
  gearType: GearType;
  brand: string;
  model: string;
  threshold?: number;
}

// Helper to convert GearType to EquipmentCategory
import type { EquipmentCategory } from './equipmentTypes';

export function gearTypeToEquipmentCategory(gearType: GearType): EquipmentCategory {
  const mapping: Record<GearType, EquipmentCategory> = {
    motor: 'motors',
    esc: 'esc',
    fc: 'flight_controllers',
    aio: 'aio',
    frame: 'frames',
    vtx: 'vtx',
    receiver: 'receivers',
    antenna: 'antennas',
    battery: 'accessories', // Note: batteries is handled differently in existing code
    prop: 'propellers',
    radio: 'accessories',
    camera: 'cameras',
    other: 'accessories',
  };
  return mapping[gearType] || 'accessories';
}

// Helper to convert EquipmentCategory to GearType
export function equipmentCategoryToGearType(category: EquipmentCategory): GearType {
  const mapping: Record<EquipmentCategory, GearType> = {
    motors: 'motor',
    esc: 'esc',
    flight_controllers: 'fc',
    aio: 'aio',
    frames: 'frame',
    vtx: 'vtx',
    receivers: 'receiver',
    antennas: 'antenna',
    propellers: 'prop',
    cameras: 'camera',
    accessories: 'other',
  };
  return mapping[category] || 'other';
}

// Get display name for a catalog item
export function getCatalogItemDisplayName(item: GearCatalogItem): string {
  let name = `${item.brand} ${item.model}`;
  if (item.variant) {
    name += ` ${item.variant}`;
  }
  return name.trim();
}

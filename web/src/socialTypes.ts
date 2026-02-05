// Social/Pilot Directory types

import type { AircraftType } from './aircraftTypes';

// Profile visibility settings
export type ProfileVisibility = 'public' | 'private';

// Social settings for a user
export interface SocialSettings {
  profileVisibility: ProfileVisibility;
  showAircraft: boolean;
  allowSearch: boolean;
}

// Pilot search result (from /api/pilots/search)
export interface PilotSearchResult {
  id: string;
  callSign?: string;
  displayName?: string;
  googleName?: string;
  effectiveAvatarUrl: string;
}

// Pilot search response
export interface PilotSearchResponse {
  pilots: PilotSearchResult[];
  total: number;
}

// Sanitized receiver settings (safe for public display)
// CRITICAL: This type intentionally OMITS sensitive fields like bindingPhrase, modelMatch, uid
// Uses the SAME field names as ReceiverConfig for simplicity
export interface ReceiverSanitizedSettings {
  rate?: number;             // Packet rate in Hz (e.g., 250, 500)
  tlm?: number;              // Telemetry ratio denominator (e.g., 8 for 1:8, 0 for off)
  power?: number;            // TX power in mW (e.g., 100, 250, 500)
  deviceName?: string;       // Device name
}

// Component category types
export type ComponentCategory = 
  | 'fc' 
  | 'esc' 
  | 'aio'
  | 'receiver' 
  | 'vtx' 
  | 'motors' 
  | 'camera' 
  | 'frame' 
  | 'propellers' 
  | 'antenna';

// Public component info (no purchase details)
// NOTE: Purchase price, seller, and notes are intentionally omitted for privacy
export interface AircraftComponentPublic {
  category: ComponentCategory;
  name?: string;
  manufacturer?: string;
  imageUrl?: string;
}

// Public aircraft info shown on pilot profiles
export interface AircraftPublic {
  id: string;
  name: string;
  nickname?: string;
  type?: AircraftType;
  hasImage: boolean;
  description?: string;
  createdAt: string;
  components?: AircraftComponentPublic[];
  receiverSettings?: ReceiverSanitizedSettings; // Sanitized receiver data
  tuning?: AircraftTuningPublic; // Public tuning data (PIDs, rates, etc)
}

// Public tuning data for pilot profiles
export interface AircraftTuningPublic {
  firmwareName?: string;
  firmwareVersion?: string;
  boardTarget?: string;
  boardName?: string;
  parsedTuning?: ParsedTuningPublic;
  snapshotDate?: string;
}

// Simplified parsed tuning for public display
export interface ParsedTuningPublic {
  pids?: PIDProfilePublic;
  rates?: RateProfilePublic;
  filters?: FilterSettingsPublic;
  activePidProfile?: number;
  activeRateProfile?: number;
}

export interface PIDProfilePublic {
  profileIndex?: number;
  roll?: { p?: number; i?: number; d?: number; ff?: number };
  pitch?: { p?: number; i?: number; d?: number; ff?: number };
  yaw?: { p?: number; i?: number; d?: number; ff?: number };
}

export interface RateAxisValues {
  roll?: number;
  pitch?: number;
  yaw?: number;
}

export interface RateProfilePublic {
  profileIndex?: number;
  rateType?: string;
  // Backend uses this structure
  rcRates?: RateAxisValues;
  superRates?: RateAxisValues;
  rcExpo?: RateAxisValues;
}

export interface FilterSettingsPublic {
  gyroLowpassHz?: number;
  gyroLowpass2Hz?: number;
  dtermLowpassHz?: number;
  dtermLowpass2Hz?: number;
  dynNotchEnabled?: boolean;
  rpmFilterEnabled?: boolean;
}

// Full pilot profile (from /api/pilots/:id)
export interface PilotProfile {
  id: string;
  callSign?: string;
  displayName?: string;
  googleName?: string;
  effectiveAvatarUrl: string;
  createdAt: string;
  aircraft: AircraftPublic[];
  isFollowing: boolean;
  followerCount: number;
  followingCount: number;
}

// Pilot summary (for follower/following lists)
export interface PilotSummary {
  id: string;
  callSign?: string;
  displayName?: string;
  effectiveAvatarUrl: string;
}

// Pilot summary with follower count (for discovery)
export interface PilotSummaryWithFollowers extends PilotSummary {
  followerCount: number;
}

// Featured pilots response (from /api/pilots/discover)
export interface FeaturedPilotsResponse {
  popular: PilotSummaryWithFollowers[];
  recent: PilotSummary[];
}

// Follow list response
export interface FollowListResponse {
  pilots: PilotSummary[];
  totalCount: number;
}

// Follow/unfollow response
export interface FollowResponse {
  success: boolean;
  following: boolean;
  followId?: string;
}

// Avatar upload response
export interface AvatarUploadResponse {
  avatarUrl: string;
  avatarType?: 'google' | 'custom';
  effectiveAvatar?: string;
}


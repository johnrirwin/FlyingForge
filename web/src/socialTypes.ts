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
}


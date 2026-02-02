// Social/Pilot Directory types

import type { AircraftType } from './aircraftTypes';

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

// Public aircraft info shown on pilot profiles
export interface AircraftPublic {
  id: string;
  name: string;
  nickname?: string;
  type?: AircraftType;
  hasImage: boolean;
  description?: string;
  createdAt: string;
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
}

// Avatar upload response
export interface AvatarUploadResponse {
  effectiveAvatarUrl: string;
  avatarType: 'google' | 'custom';
  customAvatarUrl: string;
}

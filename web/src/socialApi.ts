// Social API client for follow/unfollow and social settings

import type { 
  SocialSettings, 
  FollowResponse, 
  FollowListResponse
} from './socialTypes';
import { getStoredTokens } from './authApi';

const API_BASE = '/api';

// Helper to get auth headers
function getAuthHeaders(): HeadersInit {
  const tokens = getStoredTokens();
  return {
    'Content-Type': 'application/json',
    ...(tokens?.accessToken && { 'Authorization': `Bearer ${tokens.accessToken}` }),
  };
}

// Get public aircraft image URL (for pilot profiles)
export function getPublicAircraftImageUrl(aircraftId: string): string {
  const tokens = getStoredTokens();
  const timestamp = Date.now();
  const baseUrl = `${API_BASE}/pilots/aircraft/${aircraftId}/image`;
  if (tokens?.accessToken) {
    return `${baseUrl}?token=${encodeURIComponent(tokens.accessToken)}&t=${timestamp}`;
  }
  return `${baseUrl}?t=${timestamp}`;
}

// Follow a pilot
export async function followPilot(userId: string): Promise<FollowResponse> {
  const response = await fetch(`${API_BASE}/social/follow/${userId}`, {
    method: 'POST',
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(error.message || 'Failed to follow pilot');
  }

  return response.json();
}

// Unfollow a pilot
export async function unfollowPilot(userId: string): Promise<FollowResponse> {
  const response = await fetch(`${API_BASE}/social/follow/${userId}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(error.message || 'Failed to unfollow pilot');
  }

  return response.json();
}

// Get a user's followers
export async function getFollowers(
  userId: string, 
  limit = 20, 
  offset = 0
): Promise<FollowListResponse> {
  const params = new URLSearchParams({
    limit: limit.toString(),
    offset: offset.toString(),
  });

  const response = await fetch(`${API_BASE}/social/${userId}/followers?${params}`, {
    method: 'GET',
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(error.message || 'Failed to get followers');
  }

  return response.json();
}

// Get users a user is following
export async function getFollowing(
  userId: string, 
  limit = 20, 
  offset = 0
): Promise<FollowListResponse> {
  const params = new URLSearchParams({
    limit: limit.toString(),
    offset: offset.toString(),
  });

  const response = await fetch(`${API_BASE}/social/${userId}/following?${params}`, {
    method: 'GET',
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(error.message || 'Failed to get following');
  }

  return response.json();
}

// Get current user's social settings
export async function getSocialSettings(): Promise<SocialSettings> {
  const response = await fetch(`${API_BASE}/me/social-settings`, {
    method: 'GET',
    headers: getAuthHeaders(),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(error.message || 'Failed to get social settings');
  }

  return response.json();
}

// Update current user's social settings
export async function updateSocialSettings(
  settings: Partial<SocialSettings>
): Promise<SocialSettings> {
  const response = await fetch(`${API_BASE}/me/social-settings`, {
    method: 'PUT',
    headers: {
      ...getAuthHeaders(),
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(settings),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(error.message || 'Failed to update social settings');
  }

  return response.json();
}

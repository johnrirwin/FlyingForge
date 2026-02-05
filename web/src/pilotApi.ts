import type { PilotSearchResponse, PilotProfile, FeaturedPilotsResponse } from './socialTypes';
import { getStoredTokens } from './authApi';

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

// Get authorization header
function getAuthHeader(): Record<string, string> {
  const tokens = getStoredTokens();
  if (!tokens) {
    throw new Error('Not authenticated');
  }
  return {
    Authorization: `Bearer ${tokens.accessToken}`,
  };
}

// Discover pilots - returns featured/popular pilots
export async function discoverPilots(limit?: number): Promise<FeaturedPilotsResponse> {
  const params = new URLSearchParams();
  if (limit) {
    params.set('limit', limit.toString());
  }
  
  const url = params.toString() 
    ? `${API_BASE}/api/pilots/discover?${params}` 
    : `${API_BASE}/api/pilots/discover`;
  
  const response = await fetch(url, {
    method: 'GET',
    headers: {
      ...getAuthHeader(),
    },
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to discover pilots' }));
    throw new Error(error.message || 'Failed to discover pilots');
  }

  return response.json();
}

// Search pilots by callsign or name
export async function searchPilots(query: string): Promise<PilotSearchResponse> {
  const params = new URLSearchParams({ q: query });
  
  const response = await fetch(`${API_BASE}/api/pilots/search?${params}`, {
    method: 'GET',
    headers: {
      ...getAuthHeader(),
    },
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to search pilots' }));
    throw new Error(error.message || 'Failed to search pilots');
  }

  return response.json();
}

// Get a pilot's public profile
export async function getPilotProfile(pilotId: string): Promise<PilotProfile> {
  const response = await fetch(`${API_BASE}/api/pilots/${pilotId}`, {
    method: 'GET',
    headers: {
      ...getAuthHeader(),
    },
  });

  if (!response.ok) {
    if (response.status === 404) {
      throw new Error('Pilot not found');
    }
    const error = await response.json().catch(() => ({ message: 'Failed to get pilot profile' }));
    throw new Error(error.message || 'Failed to get pilot profile');
  }

  return response.json();
}

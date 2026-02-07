import type { UserProfile, UpdateProfileParams } from './authTypes';
import type { AvatarUploadResponse } from './socialTypes';
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

// Get current user's profile
export async function getProfile(): Promise<UserProfile> {
  const response = await fetch(`${API_BASE}/api/me/profile`, {
    method: 'GET',
    headers: {
      ...getAuthHeader(),
    },
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to get profile' }));
    throw new Error(error.message || 'Failed to get profile');
  }

  return response.json();
}

// Update current user's profile
export async function updateProfile(params: UpdateProfileParams): Promise<UserProfile> {
  const response = await fetch(`${API_BASE}/api/me/profile`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeader(),
    },
    body: JSON.stringify(params),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to update profile' }));
    throw new Error(error.message || 'Failed to update profile');
  }

  return response.json();
}

// Upload custom avatar
export async function uploadAvatar(file: File): Promise<AvatarUploadResponse> {
  const formData = new FormData();
  formData.append('avatar', file);

  const response = await fetch(`${API_BASE}/api/me/avatar`, {
    method: 'POST',
    headers: {
      ...getAuthHeader(),
    },
    body: formData,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to upload avatar' }));
    throw new Error(error.message || 'Failed to upload avatar');
  }

  return response.json();
}

// Validate callsign format (client-side)
export function validateCallSign(callSign: string): string | null {
  const trimmed = callSign.trim();
  if (trimmed === '') {
    return null; // Empty is allowed (to clear callsign)
  }
  if (trimmed.length < 3) {
    return 'Callsign must be at least 3 characters';
  }
  if (trimmed.length > 20) {
    return 'Callsign must be at most 20 characters';
  }
  if (!/^[a-zA-Z0-9_-]+$/.test(trimmed)) {
    return 'Callsign can only contain letters, numbers, underscores, and hyphens';
  }
  return null;
}

// Delete current user's account and all associated data
export async function deleteAccount(): Promise<void> {
  const response = await fetch(`${API_BASE}/api/me/profile`, {
    method: 'DELETE',
    headers: {
      ...getAuthHeader(),
    },
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to delete account' }));
    throw new Error(error.message || 'Failed to delete account');
  }

  // No content returned on success (204)
}

import BadWordsNext from 'bad-words-next';
import en from 'bad-words-next/lib/en';
import type { UserProfile, UpdateProfileParams } from './authTypes';
import type { AvatarUploadResponse } from './socialTypes';
import { getStoredTokens } from './authApi';
import type { ImageModerationResponse } from './imageTypes';
export type { ModerationStatus, ImageModerationResponse } from './imageTypes';

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

const callSignProfanityFilter = new BadWordsNext({ data: en });

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

// Upload image for moderation (does not persist avatar yet)
export async function moderateImageUpload(
  file: File,
  entityType: 'avatar' | 'aircraft' | 'gear' | 'other' = 'avatar'
): Promise<ImageModerationResponse> {
  const formData = new FormData();
  formData.append('image', file);
  formData.append('entityType', entityType);

  const response = await fetch(`${API_BASE}/api/images/upload`, {
    method: 'POST',
    headers: {
      ...getAuthHeader(),
    },
    body: formData,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to moderate image' }));
    throw new Error(error.message || error.reason || 'Failed to moderate image');
  }

  return response.json();
}

// Persist custom avatar after moderation returned APPROVED
export async function uploadAvatar(uploadId: string): Promise<AvatarUploadResponse> {
  if (!uploadId) {
    throw new Error('uploadId is required');
  }

  const response = await fetch(`${API_BASE}/api/users/avatar`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeader(),
    },
    body: JSON.stringify({ uploadId }),
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
  if (containsBlockedCallSignTerm(trimmed)) {
    return 'Callsign contains inappropriate language';
  }
  return null;
}

function containsBlockedCallSignTerm(callSign: string): boolean {
  return callSignProfanityFilter.check(callSign);
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

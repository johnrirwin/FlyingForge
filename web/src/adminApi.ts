// Admin API for gear moderation

import type {
  GearCatalogItem,
  GearCatalogSearchResponse,
  AdminGearSearchParams,
  AdminUpdateGearCatalogParams,
} from './gearCatalogTypes';
import { getStoredTokens } from './authApi';

const API_BASE = '/api/admin';

// Get auth token from stored tokens
function getAuthToken(): string | null {
  const tokens = getStoredTokens();
  return tokens?.accessToken || null;
}

// Admin search for gear items (with imageStatus filter)
export async function adminSearchGear(
  params: AdminGearSearchParams
): Promise<GearCatalogSearchResponse> {
  const token = getAuthToken();
  if (!token) {
    throw new Error('Authentication required');
  }

  const searchParams = new URLSearchParams();
  if (params.query) searchParams.set('query', params.query);
  if (params.gearType) searchParams.set('gearType', params.gearType);
  if (params.brand) searchParams.set('brand', params.brand);
  if (params.imageStatus) searchParams.set('imageStatus', params.imageStatus);
  if (params.limit) searchParams.set('limit', params.limit.toString());
  if (params.offset) searchParams.set('offset', params.offset.toString());

  const response = await fetch(`${API_BASE}/gear?${searchParams.toString()}`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: 'Request failed' }));
    if (response.status === 403) {
      throw new Error('Admin access required');
    }
    throw new Error(data.error || 'Failed to search gear');
  }

  return response.json();
}

// Get a single gear item by ID
export async function adminGetGear(id: string): Promise<GearCatalogItem> {
  const token = getAuthToken();
  if (!token) {
    throw new Error('Authentication required');
  }

  const response = await fetch(`${API_BASE}/gear/${id}`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: 'Request failed' }));
    if (response.status === 403) {
      throw new Error('Admin access required');
    }
    if (response.status === 404) {
      throw new Error('Gear item not found');
    }
    throw new Error(data.error || 'Failed to get gear item');
  }

  return response.json();
}

// Update a gear item (admin only)
export async function adminUpdateGear(
  id: string,
  params: AdminUpdateGearCatalogParams
): Promise<GearCatalogItem> {
  const token = getAuthToken();
  if (!token) {
    throw new Error('Authentication required');
  }

  const response = await fetch(`${API_BASE}/gear/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify(params),
  });

  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: 'Request failed' }));
    if (response.status === 403) {
      throw new Error('Admin access required');
    }
    if (response.status === 404) {
      throw new Error('Gear item not found');
    }
    throw new Error(data.error || 'Failed to update gear item');
  }

  return response.json();
}

// Upload an image for a gear item (admin only)
// Max file size: 1MB, accepts JPEG/PNG/WebP
export async function adminUploadGearImage(
  id: string,
  imageFile: File
): Promise<void> {
  const token = getAuthToken();
  if (!token) {
    throw new Error('Authentication required');
  }

  // Validate file size (1MB max)
  if (imageFile.size > 1024 * 1024) {
    throw new Error('Image file is too large. Maximum size is 1MB.');
  }

  // Validate file type
  const validTypes = ['image/jpeg', 'image/png', 'image/webp'];
  if (!validTypes.includes(imageFile.type)) {
    throw new Error('Invalid image type. Please use JPEG, PNG, or WebP.');
  }

  const formData = new FormData();
  formData.append('image', imageFile);

  const response = await fetch(`${API_BASE}/gear/${id}/image`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
      // Don't set Content-Type - browser will set it with boundary for FormData
    },
    body: formData,
  });

  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: 'Request failed' }));
    if (response.status === 403) {
      throw new Error('Admin access required');
    }
    if (response.status === 400) {
      throw new Error(data.error || 'Invalid image');
    }
    throw new Error(data.error || 'Failed to upload image');
  }
}

// Delete the image for a gear item (admin only)
export async function adminDeleteGearImage(id: string): Promise<void> {
  const token = getAuthToken();
  if (!token) {
    throw new Error('Authentication required');
  }

  const response = await fetch(`${API_BASE}/gear/${id}/image`, {
    method: 'DELETE',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: 'Request failed' }));
    throw new Error(data.error || 'Failed to delete image');
  }
}

// Get the URL for a gear catalog image (public endpoint)
export function getGearImageUrl(gearId: string): string {
  return `/api/gear-catalog/${gearId}/image`;
}

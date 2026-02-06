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

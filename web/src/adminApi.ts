// Admin API for content moderation and user administration

import type {
  GearCatalogItem,
  GearCatalogSearchResponse,
  AdminGearSearchParams,
  AdminUpdateGearCatalogParams,
} from './gearCatalogTypes';
import type { Build, BuildListResponse, BuildPublishResponse, BuildStatus, UpdateBuildParams } from './buildTypes';
import type {
  AdminUser,
  AdminUserSearchParams,
  AdminUsersResponse,
  AdminUpdateUserParams,
} from './adminUserTypes';
import { getStoredTokens } from './authApi';

const API_BASE = '/api/admin';

// Get auth token from stored tokens
function getAuthToken(): string | null {
  const tokens = getStoredTokens();
  return tokens?.accessToken || null;
}

function withAdminImageAuth(url: string, cacheBuster?: number): string {
  const params = new URLSearchParams();
  const token = getAuthToken();
  if (token) {
    params.set('token', token);
  }
  if (typeof cacheBuster === 'number') {
    params.set('v', cacheBuster.toString());
  }
  const query = params.toString();
  return query ? `${url}?${query}` : url;
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
  if (params.status) searchParams.set('status', params.status);
  if (params.imageStatus) searchParams.set('imageStatus', params.imageStatus);
  if (params.limit) searchParams.set('limit', params.limit.toString());
  if (params.offset) searchParams.set('offset', params.offset.toString());

  const response = await fetch(`${API_BASE}/gear?${searchParams.toString()}`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
    // Prevent browser from returning cached responses
    cache: 'no-store',
  });

  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: 'Request failed' }));
    if (response.status === 403) {
      throw new Error('Admin or content-admin access required');
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
    cache: 'no-store',
  });

  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: 'Request failed' }));
    if (response.status === 403) {
      throw new Error('Admin or content-admin access required');
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
      throw new Error('Admin or content-admin access required');
    }
    if (response.status === 404) {
      throw new Error('Gear item not found');
    }
    throw new Error(data.error || 'Failed to update gear item');
  }

  return response.json();
}

// Upload an image for a gear item (admin only)
// Max file size: 2MB, accepts JPEG/PNG/WebP
export async function adminUploadGearImage(
  id: string,
  imageFile: File
): Promise<void> {
  const token = getAuthToken();
  if (!token) {
    throw new Error('Authentication required');
  }

  // Validate file size (2MB max)
  if (imageFile.size > 2 * 1024 * 1024) {
    throw new Error('Image file is too large. Maximum size is 2MB.');
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
      throw new Error('Admin or content-admin access required');
    }
    if (response.status === 400) {
      throw new Error(data.error || 'Invalid image');
    }
    throw new Error(data.error || 'Failed to upload image');
  }
}

// Persist an approved moderated upload token as a curated gear image (admin only).
export async function adminSaveGearImageUpload(id: string, uploadId: string): Promise<void> {
  const token = getAuthToken();
  if (!token) {
    throw new Error('Authentication required');
  }
  if (!uploadId) {
    throw new Error('uploadId is required');
  }

  const response = await fetch(`${API_BASE}/gear/${id}/image`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ uploadId }),
  });

  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: 'Request failed' }));
    if (response.status === 403) {
      throw new Error('Admin or content-admin access required');
    }
    if (response.status === 404) {
      throw new Error('Gear item not found');
    }
    if (response.status === 422) {
      throw new Error(data.error || 'Image approval token expired or not approved');
    }
    throw new Error(data.error || 'Failed to save approved image');
  }
}

// Approve an existing scanned image for a gear item (admin only)
export async function adminApproveGearImage(id: string): Promise<void> {
  const token = getAuthToken();
  if (!token) {
    throw new Error('Authentication required');
  }

  const response = await fetch(`${API_BASE}/gear/${id}/image/approve`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: 'Request failed' }));
    if (response.status === 403) {
      throw new Error('Admin or content-admin access required');
    }
    if (response.status === 404) {
      throw new Error('Gear item not found');
    }
    if (response.status === 422) {
      throw new Error(data.error || 'No image available to approve');
    }
    throw new Error(data.error || 'Failed to approve image');
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

// Delete a gear item (admin only)
export async function adminDeleteGear(id: string): Promise<void> {
  const token = getAuthToken();
  if (!token) {
    throw new Error('Authentication required');
  }

  const response = await fetch(`${API_BASE}/gear/${id}`, {
    method: 'DELETE',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: 'Request failed' }));
    if (response.status === 403) {
      throw new Error('Admin or content-admin access required');
    }
    if (response.status === 404) {
      throw new Error('Gear item not found');
    }
    throw new Error(data.error || 'Failed to delete gear item');
  }
}

// Get the URL for a gear catalog image (public endpoint)
// Optional timestamp parameter for cache-busting after uploads
export function getGearImageUrl(gearId: string, cacheBuster?: number): string {
  const url = `/api/gear-catalog/${gearId}/image`;
  return cacheBuster ? `${url}?v=${cacheBuster}` : url;
}

// Get the URL for a gear catalog image via admin endpoint (no caching)
// Use this in admin UI to always see latest image
export function getAdminGearImageUrl(gearId: string, cacheBuster?: number): string {
  const url = `/api/admin/gear/${gearId}/image`;
  return withAdminImageAuth(url, cacheBuster);
}

export interface AdminBuildSearchParams {
  query?: string;
  status?: BuildStatus;
  limit?: number;
  offset?: number;
}

export async function adminSearchBuilds(params: AdminBuildSearchParams): Promise<BuildListResponse> {
  const token = getAuthToken();
  if (!token) {
    throw new Error('Authentication required');
  }

  const searchParams = new URLSearchParams();
  if (params.query) searchParams.set('query', params.query);
  if (params.status) searchParams.set('status', params.status);
  if (params.limit) searchParams.set('limit', params.limit.toString());
  if (params.offset) searchParams.set('offset', params.offset.toString());

  const response = await fetch(`${API_BASE}/builds?${searchParams.toString()}`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
    cache: 'no-store',
  });

  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: 'Request failed' }));
    if (response.status === 403) {
      throw new Error('Admin or content-admin access required');
    }
    throw new Error(data.error || 'Failed to search builds');
  }

  return response.json();
}

export async function adminGetBuild(id: string): Promise<Build> {
  const token = getAuthToken();
  if (!token) {
    throw new Error('Authentication required');
  }

  const response = await fetch(`${API_BASE}/builds/${id}`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
    cache: 'no-store',
  });

  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: 'Request failed' }));
    if (response.status === 403) {
      throw new Error('Admin or content-admin access required');
    }
    if (response.status === 404) {
      throw new Error('Build not found');
    }
    throw new Error(data.error || 'Failed to get build');
  }

  return response.json();
}

export async function adminUpdateBuild(id: string, params: UpdateBuildParams): Promise<Build> {
  const token = getAuthToken();
  if (!token) {
    throw new Error('Authentication required');
  }

  const response = await fetch(`${API_BASE}/builds/${id}`, {
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
      throw new Error('Admin or content-admin access required');
    }
    if (response.status === 404) {
      throw new Error('Build not found');
    }
    throw new Error(data.error || 'Failed to update build');
  }

  return response.json();
}

export async function adminPublishBuild(id: string): Promise<BuildPublishResponse> {
  const token = getAuthToken();
  if (!token) {
    throw new Error('Authentication required');
  }

  const response = await fetch(`${API_BASE}/builds/${id}/publish`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (response.status === 400) {
    const data = await response.json().catch(() => ({ error: 'Request failed' }));
    if (data.validation) {
      return data as BuildPublishResponse;
    }
    throw new Error(data.error || 'Failed to publish build');
  }

  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: 'Request failed' }));
    if (response.status === 403) {
      throw new Error('Admin or content-admin access required');
    }
    if (response.status === 404) {
      throw new Error('Build not found');
    }
    throw new Error(data.error || 'Failed to publish build');
  }

  return response.json();
}

export async function adminUploadBuildImage(id: string, imageFile: File): Promise<void> {
  const token = getAuthToken();
  if (!token) {
    throw new Error('Authentication required');
  }

  if (imageFile.size > 2 * 1024 * 1024) {
    throw new Error('Image file is too large. Maximum size is 2MB.');
  }

  const validTypes = ['image/jpeg', 'image/png', 'image/webp'];
  if (!validTypes.includes(imageFile.type)) {
    throw new Error('Invalid image type. Please use JPEG, PNG, or WebP.');
  }

  const formData = new FormData();
  formData.append('image', imageFile);

  const response = await fetch(`${API_BASE}/builds/${id}/image`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${token}`,
    },
    body: formData,
  });

  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: 'Request failed' }));
    if (response.status === 403) {
      throw new Error('Admin or content-admin access required');
    }
    if (response.status === 404) {
      throw new Error('Build not found');
    }
    throw new Error(data.error || 'Failed to upload build image');
  }
}

export async function adminDeleteBuildImage(id: string): Promise<void> {
  const token = getAuthToken();
  if (!token) {
    throw new Error('Authentication required');
  }

  const response = await fetch(`${API_BASE}/builds/${id}/image`, {
    method: 'DELETE',
    headers: {
      Authorization: `Bearer ${token}`,
    },
  });

  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: 'Request failed' }));
    if (response.status === 403) {
      throw new Error('Admin or content-admin access required');
    }
    if (response.status === 404) {
      throw new Error('Build not found');
    }
    throw new Error(data.error || 'Failed to delete build image');
  }
}

export function getAdminBuildImageUrl(buildId: string, cacheBuster?: number): string {
  const url = `/api/admin/builds/${buildId}/image`;
  return withAdminImageAuth(url, cacheBuster);
}

// Admin search for users (admin only)
export async function adminSearchUsers(
  params: AdminUserSearchParams
): Promise<AdminUsersResponse> {
  const token = getAuthToken();
  if (!token) {
    throw new Error('Authentication required');
  }

  const searchParams = new URLSearchParams();
  if (params.query) searchParams.set('query', params.query);
  if (params.status) searchParams.set('status', params.status);
  if (params.limit) searchParams.set('limit', params.limit.toString());
  if (params.offset) searchParams.set('offset', params.offset.toString());

  const response = await fetch(`${API_BASE}/users?${searchParams.toString()}`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
    cache: 'no-store',
  });

  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: 'Request failed' }));
    if (response.status === 403) {
      throw new Error('Admin access required');
    }
    throw new Error(data.error || 'Failed to search users');
  }

  return response.json();
}

// Get a single user by ID (admin only)
export async function adminGetUser(id: string): Promise<AdminUser> {
  const token = getAuthToken();
  if (!token) {
    throw new Error('Authentication required');
  }

  const response = await fetch(`${API_BASE}/users/${id}`, {
    headers: {
      Authorization: `Bearer ${token}`,
    },
    cache: 'no-store',
  });

  if (!response.ok) {
    const data = await response.json().catch(() => ({ error: 'Request failed' }));
    if (response.status === 403) {
      throw new Error('Admin access required');
    }
    if (response.status === 404) {
      throw new Error('User not found');
    }
    throw new Error(data.error || 'Failed to get user');
  }

  return response.json();
}

// Update a user as admin (status/roles)
export async function adminUpdateUser(
  id: string,
  params: AdminUpdateUserParams
): Promise<AdminUser> {
  const token = getAuthToken();
  if (!token) {
    throw new Error('Authentication required');
  }

  const response = await fetch(`${API_BASE}/users/${id}`, {
    method: 'PATCH',
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
      throw new Error('User not found');
    }
    throw new Error(data.error || 'Failed to update user');
  }

  return response.json();
}

// Delete a user account (admin only)
export async function adminDeleteUser(id: string): Promise<void> {
  const token = getAuthToken();
  if (!token) {
    throw new Error('Authentication required');
  }

  const response = await fetch(`${API_BASE}/users/${id}`, {
    method: 'DELETE',
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
      throw new Error('User not found');
    }
    throw new Error(data.error || 'Failed to delete user');
  }
}

// Remove a user's profile picture (admin only)
export async function adminDeleteUserAvatar(id: string): Promise<AdminUser> {
  const token = getAuthToken();
  if (!token) {
    throw new Error('Authentication required');
  }

  const response = await fetch(`${API_BASE}/users/${id}/avatar`, {
    method: 'DELETE',
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
      throw new Error('User not found');
    }
    throw new Error(data.error || 'Failed to remove profile picture');
  }

  return response.json();
}

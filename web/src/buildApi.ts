import type {
  Build,
  BuildListParams,
  BuildListResponse,
  BuildPublishResponse,
  CreateBuildParams,
  TempBuildCreateResponse,
  UpdateBuildParams,
} from './buildTypes';

const API_BASE = import.meta.env.VITE_API_BASE_URL || '';

function getAccessToken(): string | null {
  return localStorage.getItem('access_token');
}

async function fetchJSON<T>(endpoint: string, options?: RequestInit, includeAuth = true): Promise<T> {
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...options?.headers,
  };

  if (includeAuth) {
    const token = getAccessToken();
    if (token) {
      (headers as Record<string, string>).Authorization = `Bearer ${token}`;
    }
  }

  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Request failed' }));
    throw new Error(error.message || error.error || `HTTP ${response.status}`);
  }

  if (response.status === 204) {
    return {} as T;
  }

  return response.json() as Promise<T>;
}

function buildQuery(params?: BuildListParams): string {
  if (!params) return '';
  const query = new URLSearchParams();
  if (params.sort) query.set('sort', params.sort);
  if (params.frameFilter) query.set('frameFilter', params.frameFilter);
  if (params.limit !== undefined) query.set('limit', String(params.limit));
  if (params.offset !== undefined) query.set('offset', String(params.offset));
  const q = query.toString();
  return q ? `?${q}` : '';
}

// Public endpoints
export async function listPublicBuilds(params?: BuildListParams): Promise<BuildListResponse> {
  return fetchJSON<BuildListResponse>(`/api/public/builds${buildQuery(params)}`, undefined, false);
}

export async function getPublicBuild(id: string): Promise<Build> {
  return fetchJSON<Build>(`/api/public/builds/${id}`, undefined, false);
}

// Temporary build endpoints
export async function createTempBuild(params?: CreateBuildParams): Promise<TempBuildCreateResponse> {
  const token = getAccessToken();
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
  };
  if (token) {
    (headers as Record<string, string>).Authorization = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE}/api/builds/temp`, {
    method: 'POST',
    headers,
    body: JSON.stringify(params ?? {}),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Request failed' }));
    throw new Error(error.message || error.error || `HTTP ${response.status}`);
  }

  return response.json() as Promise<TempBuildCreateResponse>;
}

export async function getTempBuild(token: string): Promise<Build> {
  return fetchJSON<Build>(`/api/builds/temp/${token}`, undefined, false);
}

export async function updateTempBuild(token: string, params: UpdateBuildParams): Promise<Build> {
  return fetchJSON<Build>(`/api/builds/temp/${token}`, {
    method: 'PUT',
    body: JSON.stringify(params),
  }, false);
}

export async function shareTempBuild(token: string): Promise<Build> {
  return fetchJSON<Build>(`/api/builds/temp/${token}/share`, {
    method: 'POST',
  }, false);
}

// Authenticated build management
export async function listMyBuilds(params?: BuildListParams): Promise<BuildListResponse> {
  return fetchJSON<BuildListResponse>(`/api/builds${buildQuery(params)}`);
}

export async function createDraftBuild(params?: CreateBuildParams): Promise<Build> {
  return fetchJSON<Build>('/api/builds', {
    method: 'POST',
    body: JSON.stringify(params ?? {}),
  });
}

export async function createBuildFromAircraft(aircraftId: string): Promise<Build> {
  return fetchJSON<Build>(`/api/builds/from-aircraft/${aircraftId}`, {
    method: 'POST',
  });
}

export async function getMyBuild(id: string): Promise<Build> {
  return fetchJSON<Build>(`/api/builds/${id}`);
}

export async function updateMyBuild(id: string, params: UpdateBuildParams): Promise<Build> {
  return fetchJSON<Build>(`/api/builds/${id}`, {
    method: 'PUT',
    body: JSON.stringify(params),
  });
}

export async function deleteMyBuild(id: string): Promise<void> {
  await fetchJSON<void>(`/api/builds/${id}`, {
    method: 'DELETE',
  });
}

export async function publishMyBuild(id: string): Promise<BuildPublishResponse> {
  const token = getAccessToken();
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
  };
  if (token) {
    (headers as Record<string, string>).Authorization = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE}/api/builds/${id}/publish`, {
    method: 'POST',
    headers,
  });

  if (response.status === 400) {
    return response.json() as Promise<BuildPublishResponse>;
  }

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Request failed' }));
    throw new Error(error.message || error.error || `HTTP ${response.status}`);
  }

  return response.json() as Promise<BuildPublishResponse>;
}

export async function unpublishMyBuild(id: string): Promise<Build> {
  return fetchJSON<Build>(`/api/builds/${id}/unpublish`, {
    method: 'POST',
  });
}

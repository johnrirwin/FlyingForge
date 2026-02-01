import type {
  Aircraft,
  AircraftComponent,
  AircraftDetailsResponse,
  AircraftELRSSettings,
  AircraftListParams,
  AircraftListResponse,
  ComponentsResponse,
  CreateAircraftParams,
  SetComponentParams,
  SetELRSSettingsParams,
  UpdateAircraftParams,
} from './aircraftTypes';

const API_BASE = import.meta.env.VITE_API_BASE_URL || '';

// Get access token from localStorage
function getAccessToken(): string | null {
  return localStorage.getItem('access_token');
}

async function fetchAPI<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const token = getAccessToken();
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...options?.headers,
  };
  
  if (token) {
    (headers as Record<string, string>)['Authorization'] = `Bearer ${token}`;
  }
  
  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Request failed' }));
    throw new Error(error.message || error.error || `HTTP ${response.status}`);
  }

  // Handle 204 No Content
  if (response.status === 204) {
    return {} as T;
  }

  return response.json();
}

// Aircraft CRUD

export async function listAircraft(params?: AircraftListParams): Promise<AircraftListResponse> {
  const searchParams = new URLSearchParams();
  
  if (params?.type) searchParams.set('type', params.type);
  if (params?.limit) searchParams.set('limit', params.limit.toString());
  if (params?.offset) searchParams.set('offset', params.offset.toString());

  const query = searchParams.toString();
  return fetchAPI<AircraftListResponse>(`/api/aircraft${query ? `?${query}` : ''}`);
}

export async function getAircraft(id: string): Promise<Aircraft> {
  return fetchAPI<Aircraft>(`/api/aircraft/${id}`);
}

export async function getAircraftDetails(id: string): Promise<AircraftDetailsResponse> {
  return fetchAPI<AircraftDetailsResponse>(`/api/aircraft/${id}/details`);
}

export async function createAircraft(params: CreateAircraftParams): Promise<Aircraft> {
  return fetchAPI<Aircraft>('/api/aircraft', {
    method: 'POST',
    body: JSON.stringify(params),
  });
}

export async function updateAircraft(id: string, params: UpdateAircraftParams): Promise<Aircraft> {
  return fetchAPI<Aircraft>(`/api/aircraft/${id}`, {
    method: 'PUT',
    body: JSON.stringify(params),
  });
}

export async function deleteAircraft(id: string): Promise<void> {
  await fetchAPI<void>(`/api/aircraft/${id}`, {
    method: 'DELETE',
  });
}

// Aircraft Components

export async function getAircraftComponents(aircraftId: string): Promise<ComponentsResponse> {
  return fetchAPI<ComponentsResponse>(`/api/aircraft/${aircraftId}/components`);
}

export async function setAircraftComponent(
  aircraftId: string,
  params: SetComponentParams
): Promise<AircraftComponent | null> {
  const response = await fetch(`${API_BASE}/api/aircraft/${aircraftId}/components`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${getAccessToken()}`,
    },
    body: JSON.stringify(params),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Request failed' }));
    throw new Error(error.message || error.error || `HTTP ${response.status}`);
  }

  // 204 means component was removed
  if (response.status === 204) {
    return null;
  }

  return response.json();
}

export async function removeAircraftComponent(
  aircraftId: string,
  category: string
): Promise<void> {
  await fetchAPI<void>(`/api/aircraft/${aircraftId}/components?category=${category}`, {
    method: 'DELETE',
  });
}

// ELRS Settings

export async function getELRSSettings(aircraftId: string): Promise<AircraftELRSSettings> {
  return fetchAPI<AircraftELRSSettings>(`/api/aircraft/${aircraftId}/elrs`);
}

export async function setELRSSettings(
  aircraftId: string,
  params: SetELRSSettingsParams
): Promise<AircraftELRSSettings> {
  return fetchAPI<AircraftELRSSettings>(`/api/aircraft/${aircraftId}/elrs`, {
    method: 'POST',
    body: JSON.stringify({ settings: JSON.stringify(params.settings) }),
  });
}

// Aircraft Image

export function getAircraftImageUrl(aircraftId: string): string {
  return `${API_BASE}/api/aircraft/${aircraftId}/image`;
}

export async function uploadAircraftImage(aircraftId: string, imageFile: File): Promise<void> {
  const token = getAccessToken();
  const formData = new FormData();
  formData.append('image', imageFile);

  const response = await fetch(`${API_BASE}/api/aircraft/${aircraftId}/image`, {
    method: 'POST',
    headers: token ? { 'Authorization': `Bearer ${token}` } : {},
    body: formData,
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Upload failed' }));
    throw new Error(error.message || error.error || `HTTP ${response.status}`);
  }
}

export async function deleteAircraftImage(aircraftId: string): Promise<void> {
  await fetchAPI<void>(`/api/aircraft/${aircraftId}/image`, {
    method: 'DELETE',
  });
}

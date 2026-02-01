import type {
  Radio,
  RadioBackup,
  RadioListParams,
  RadioListResponse,
  RadioModelsResponse,
  RadioBackupListParams,
  RadioBackupListResponse,
  CreateRadioParams,
  UpdateRadioParams,
  CreateRadioBackupParams,
} from './radioTypes';

const API_BASE = import.meta.env.VITE_API_BASE_URL || '';

// Get access token from localStorage
function getAccessToken(): string | null {
  return localStorage.getItem('access_token');
}

async function fetchAPI<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const token = getAccessToken();
  const headers: HeadersInit = {
    ...options?.headers,
  };

  // Only set Content-Type for non-FormData requests
  if (!(options?.body instanceof FormData)) {
    (headers as Record<string, string>)['Content-Type'] = 'application/json';
  }

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

// Radio Models (public endpoint)

export async function getRadioModels(): Promise<RadioModelsResponse> {
  return fetchAPI<RadioModelsResponse>('/api/radio/models');
}

// Radios CRUD

export async function listRadios(params?: RadioListParams): Promise<RadioListResponse> {
  const searchParams = new URLSearchParams();

  if (params?.limit) searchParams.set('limit', params.limit.toString());
  if (params?.offset) searchParams.set('offset', params.offset.toString());

  const query = searchParams.toString();
  return fetchAPI<RadioListResponse>(`/api/radios${query ? `?${query}` : ''}`);
}

export async function getRadio(id: string): Promise<Radio> {
  return fetchAPI<Radio>(`/api/radios/${id}`);
}

export async function createRadio(params: CreateRadioParams): Promise<Radio> {
  return fetchAPI<Radio>('/api/radios', {
    method: 'POST',
    body: JSON.stringify(params),
  });
}

export async function updateRadio(id: string, params: UpdateRadioParams): Promise<Radio> {
  return fetchAPI<Radio>(`/api/radios/${id}`, {
    method: 'PUT',
    body: JSON.stringify(params),
  });
}

export async function deleteRadio(id: string): Promise<void> {
  await fetchAPI<void>(`/api/radios/${id}`, {
    method: 'DELETE',
  });
}

// Backups CRUD

export async function listBackups(radioId: string, params?: RadioBackupListParams): Promise<RadioBackupListResponse> {
  const searchParams = new URLSearchParams();

  if (params?.limit) searchParams.set('limit', params.limit.toString());
  if (params?.offset) searchParams.set('offset', params.offset.toString());

  const query = searchParams.toString();
  return fetchAPI<RadioBackupListResponse>(`/api/radios/${radioId}/backups${query ? `?${query}` : ''}`);
}

export async function getBackup(radioId: string, backupId: string): Promise<RadioBackup> {
  return fetchAPI<RadioBackup>(`/api/radios/${radioId}/backups/${backupId}`);
}

export async function createBackup(radioId: string, params: CreateRadioBackupParams): Promise<RadioBackup> {
  const formData = new FormData();
  formData.append('backupName', params.backupName);
  formData.append('backupType', params.backupType);
  formData.append('file', params.file);

  return fetchAPI<RadioBackup>(`/api/radios/${radioId}/backups`, {
    method: 'POST',
    body: formData,
  });
}

export async function deleteBackup(radioId: string, backupId: string): Promise<void> {
  await fetchAPI<void>(`/api/radios/${radioId}/backups/${backupId}`, {
    method: 'DELETE',
  });
}

// Download backup (returns blob URL)
export async function downloadBackup(radioId: string, backupId: string, fileName: string): Promise<void> {
  const token = getAccessToken();
  const headers: HeadersInit = {};

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE}/api/radios/${radioId}/backups/${backupId}/download`, {
    headers,
  });

  if (!response.ok) {
    throw new Error('Failed to download backup');
  }

  // Create blob and trigger download
  const blob = await response.blob();
  const url = window.URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = fileName;
  document.body.appendChild(a);
  a.click();
  window.URL.revokeObjectURL(url);
  document.body.removeChild(a);
}

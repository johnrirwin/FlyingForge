import type {
  Battery,
  BatteryLog,
  BatteryListParams,
  BatteryListResponse,
  CreateBatteryParams,
  UpdateBatteryParams,
  CreateBatteryLogParams,
  LabelSize,
} from './batteryTypes';

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

// List batteries with optional filters
export async function getBatteries(params?: BatteryListParams): Promise<BatteryListResponse> {
  const searchParams = new URLSearchParams();
  
  if (params?.chemistry) searchParams.set('chemistry', params.chemistry);
  if (params?.cells) searchParams.set('cells', params.cells.toString());
  if (params?.min_capacity) searchParams.set('min_capacity', params.min_capacity.toString());
  if (params?.max_capacity) searchParams.set('max_capacity', params.max_capacity.toString());
  if (params?.query) searchParams.set('query', params.query);
  if (params?.sort_by) searchParams.set('sort_by', params.sort_by);
  if (params?.sort_order) searchParams.set('sort_order', params.sort_order);
  if (params?.limit) searchParams.set('limit', params.limit.toString());
  if (params?.offset) searchParams.set('offset', params.offset.toString());

  const query = searchParams.toString();
  const endpoint = query ? `/api/batteries?${query}` : '/api/batteries';
  
  return fetchAPI<BatteryListResponse>(endpoint);
}

// Get a single battery by ID
export async function getBattery(id: string): Promise<Battery> {
  return fetchAPI<Battery>(`/api/batteries/${id}`);
}

// Get a battery by its code (e.g., BAT-XXXX)
export async function getBatteryByCode(code: string): Promise<Battery> {
  return fetchAPI<Battery>(`/api/batteries/code/${code}`);
}

// Create a new battery
export async function createBattery(params: CreateBatteryParams): Promise<Battery> {
  return fetchAPI<Battery>('/api/batteries', {
    method: 'POST',
    body: JSON.stringify(params),
  });
}

// Update an existing battery
export async function updateBattery(id: string, params: UpdateBatteryParams): Promise<Battery> {
  return fetchAPI<Battery>(`/api/batteries/${id}`, {
    method: 'PUT',
    body: JSON.stringify(params),
  });
}

// Delete a battery
export async function deleteBattery(id: string): Promise<void> {
  await fetchAPI<void>(`/api/batteries/${id}`, {
    method: 'DELETE',
  });
}

// Battery Logs

interface BatteryLogListResponse {
  logs: BatteryLog[];
  totalCount: number;
}

// List logs for a battery
export async function getBatteryLogs(batteryId: string): Promise<BatteryLog[]> {
  const response = await fetchAPI<BatteryLogListResponse>(`/api/batteries/${batteryId}/logs`);
  return response.logs || [];
}

// Create a new log entry
export async function createBatteryLog(
  batteryId: string,
  params: CreateBatteryLogParams
): Promise<BatteryLog> {
  return fetchAPI<BatteryLog>(`/api/batteries/${batteryId}/logs`, {
    method: 'POST',
    body: JSON.stringify(params),
  });
}

// Delete a log entry
export async function deleteBatteryLog(batteryId: string, logId: string): Promise<void> {
  await fetchAPI<void>(`/api/batteries/${batteryId}/logs/${logId}`, {
    method: 'DELETE',
  });
}

// Battery Label

// Get label URL for printing
export function getBatteryLabelUrl(batteryId: string, size: LabelSize = 'standard'): string {
  return `${API_BASE}/api/batteries/${batteryId}/label?size=${size}`;
}

// Open label in new window for printing
export async function printBatteryLabel(batteryId: string, size: LabelSize = 'standard'): Promise<void> {
  const url = getBatteryLabelUrl(batteryId, size);
  const token = getAccessToken();
  const response = await fetch(url, {
    headers: token ? { 'Authorization': `Bearer ${token}` } : {},
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Request failed' }));
    throw new Error(error.message || error.error || `HTTP ${response.status}`);
  }
  const blob = await response.blob();
  const blobUrl = window.URL.createObjectURL(blob);
  window.open(blobUrl, '_blank', 'width=400,height=300,menubar=no,toolbar=no');
}

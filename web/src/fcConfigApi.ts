import type {
  FlightControllerConfig,
  FCConfigListResponse,
  SaveFCConfigParams,
  UpdateFCConfigParams,
  AircraftTuningResponse,
  CreateTuningSnapshotParams,
  AircraftTuningSnapshot,
  TuningSnapshotsListResponse,
} from './fcConfigTypes';
import { getStoredTokens } from './authApi';

const API_BASE = import.meta.env.VITE_API_BASE_URL || '';

function getAuthHeaders(): HeadersInit {
  const tokens = getStoredTokens();
  return {
    'Content-Type': 'application/json',
    ...(tokens?.accessToken && { Authorization: `Bearer ${tokens.accessToken}` }),
  };
}

async function fetchAPI<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers: {
      ...getAuthHeaders(),
      ...options?.headers,
    },
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Request failed' }));
    throw new Error(error.error || error.message || `HTTP ${response.status}`);
  }

  // Handle 204 No Content
  if (response.status === 204) {
    return {} as T;
  }

  return response.json();
}

// FC Config CRUD operations

/**
 * Create a new FC config from a CLI dump
 */
export async function createFCConfig(params: SaveFCConfigParams): Promise<FlightControllerConfig> {
  return fetchAPI<FlightControllerConfig>('/api/fc-configs', {
    method: 'POST',
    body: JSON.stringify(params),
  });
}

/**
 * Get a single FC config by ID
 */
export async function getFCConfig(id: string): Promise<FlightControllerConfig> {
  return fetchAPI<FlightControllerConfig>(`/api/fc-configs/${id}`);
}

/**
 * List FC configs, optionally filtered by inventory item
 */
export async function listFCConfigs(params?: {
  inventoryItemId?: string;
  limit?: number;
  offset?: number;
}): Promise<FCConfigListResponse> {
  const searchParams = new URLSearchParams();
  if (params?.inventoryItemId) searchParams.set('inventory_item_id', params.inventoryItemId);
  if (params?.limit) searchParams.set('limit', params.limit.toString());
  if (params?.offset) searchParams.set('offset', params.offset.toString());
  
  const query = searchParams.toString();
  return fetchAPI<FCConfigListResponse>(`/api/fc-configs${query ? `?${query}` : ''}`);
}

/**
 * Update an FC config's metadata (name, notes)
 */
export async function updateFCConfig(id: string, params: UpdateFCConfigParams): Promise<FlightControllerConfig> {
  return fetchAPI<FlightControllerConfig>(`/api/fc-configs/${id}`, {
    method: 'PUT',
    body: JSON.stringify(params),
  });
}

/**
 * Delete an FC config
 */
export async function deleteFCConfig(id: string): Promise<void> {
  await fetchAPI<void>(`/api/fc-configs/${id}`, {
    method: 'DELETE',
  });
}

// Aircraft Tuning operations

/**
 * Get the latest tuning data for an aircraft
 */
export async function getAircraftTuning(aircraftId: string): Promise<AircraftTuningResponse> {
  return fetchAPI<AircraftTuningResponse>(`/api/tuning/aircraft/${aircraftId}`);
}

/**
 * Create a tuning snapshot for an aircraft from a CLI dump
 */
export async function createTuningSnapshot(
  aircraftId: string,
  params: CreateTuningSnapshotParams
): Promise<AircraftTuningSnapshot> {
  return fetchAPI<AircraftTuningSnapshot>(`/api/tuning/aircraft/${aircraftId}/snapshots`, {
    method: 'POST',
    body: JSON.stringify(params),
  });
}

/**
 * List all tuning snapshots for an aircraft
 */
export async function listTuningSnapshots(aircraftId: string): Promise<TuningSnapshotsListResponse> {
  return fetchAPI<TuningSnapshotsListResponse>(`/api/tuning/aircraft/${aircraftId}/snapshots`);
}

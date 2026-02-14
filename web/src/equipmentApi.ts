import type {
  EquipmentSearchParams,
  EquipmentSearchResponse,
  SellersResponse,
  EquipmentCategory,
  InventoryResponse,
  InventoryFilterParams,
  AddInventoryParams,
  UpdateInventoryParams,
  InventoryItem,
  InventorySummary,
} from './equipmentTypes';

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

// Equipment API

export async function searchEquipment(params: EquipmentSearchParams): Promise<EquipmentSearchResponse> {
  const searchParams = new URLSearchParams();
  
  if (params.query) searchParams.set('q', params.query);
  if (params.category) searchParams.set('category', params.category);
  if (params.seller) searchParams.set('seller', params.seller);
  if (params.minPrice !== undefined) searchParams.set('minPrice', params.minPrice.toString());
  if (params.maxPrice !== undefined) searchParams.set('maxPrice', params.maxPrice.toString());
  if (params.inStockOnly) searchParams.set('inStock', 'true');
  if (params.limit) searchParams.set('limit', params.limit.toString());
  if (params.offset) searchParams.set('offset', params.offset.toString());
  if (params.sort) searchParams.set('sort', params.sort);

  const query = searchParams.toString();
  return fetchAPI<EquipmentSearchResponse>(`/api/equipment/search${query ? `?${query}` : ''}`);
}

export async function getEquipmentByCategory(
  category: EquipmentCategory,
  limit?: number,
  offset?: number
): Promise<EquipmentSearchResponse> {
  const searchParams = new URLSearchParams();
  
  if (limit) searchParams.set('limit', limit.toString());
  if (offset) searchParams.set('offset', offset.toString());

  const query = searchParams.toString();
  return fetchAPI<EquipmentSearchResponse>(`/api/equipment/category/${category}${query ? `?${query}` : ''}`);
}

export async function getSellers(): Promise<SellersResponse> {
  return fetchAPI<SellersResponse>('/api/equipment/sellers');
}

export async function syncSellerProducts(seller: string, category: EquipmentCategory): Promise<{ status: string; synced: number }> {
  return fetchAPI('/api/equipment/sync', {
    method: 'POST',
    body: JSON.stringify({ seller, category }),
  });
}

// Inventory API

export async function getInventory(params?: InventoryFilterParams): Promise<InventoryResponse> {
  const searchParams = new URLSearchParams();
  
  if (params?.category) searchParams.set('category', params.category);
  if (params?.buildId) searchParams.set('buildId', params.buildId);
  if (params?.query) searchParams.set('q', params.query);
  if (params?.limit) searchParams.set('limit', params.limit.toString());
  if (params?.offset) searchParams.set('offset', params.offset.toString());

  const query = searchParams.toString();
  return fetchAPI<InventoryResponse>(`/api/inventory${query ? `?${query}` : ''}`);
}

export async function getInventoryItem(id: string): Promise<InventoryItem> {
  return fetchAPI<InventoryItem>(`/api/inventory/${id}`);
}

export async function addInventoryItem(params: AddInventoryParams): Promise<InventoryItem> {
  return fetchAPI<InventoryItem>('/api/inventory', {
    method: 'POST',
    body: JSON.stringify(params),
  });
}

export async function updateInventoryItem(id: string, params: UpdateInventoryParams): Promise<InventoryItem> {
  return fetchAPI<InventoryItem>(`/api/inventory/${id}`, {
    method: 'PUT',
    body: JSON.stringify(params),
  });
}

export async function deleteInventoryItem(id: string): Promise<void> {
  await fetchAPI<void>(`/api/inventory/${id}`, {
    method: 'DELETE',
  });
}

export async function getInventorySummary(): Promise<InventorySummary> {
  return fetchAPI<InventorySummary>('/api/inventory/summary');
}

// Helper to add equipment item directly to inventory
export async function addEquipmentToInventory(
  equipmentId: string,
  equipmentName: string,
  category: EquipmentCategory,
  manufacturer: string,
  price: number,
  seller: string,
  productUrl: string,
  specs?: Record<string, unknown>,
  quantity = 1,
  notes?: string
): Promise<InventoryItem> {
  return addInventoryItem({
    name: equipmentName,
    category,
    manufacturer,
    quantity,
    notes,
    purchasePrice: price,
    purchaseSeller: seller,
    productUrl,
    specs,
    sourceEquipmentId: equipmentId,
  });
}

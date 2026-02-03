import type {
  Order,
  AddOrderParams,
  UpdateOrderParams,
  OrderListResponse,
} from './orderTypes';

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

export interface OrderListParams {
  limit?: number;
  offset?: number;
  includeArchived?: boolean;
}

// List orders with optional filters
export async function getOrders(params?: OrderListParams): Promise<OrderListResponse> {
  const searchParams = new URLSearchParams();

  if (params?.limit) searchParams.set('limit', params.limit.toString());
  if (params?.offset) searchParams.set('offset', params.offset.toString());
  if (params?.includeArchived) searchParams.set('includeArchived', 'true');

  const query = searchParams.toString();
  const endpoint = query ? `/api/orders?${query}` : '/api/orders';

  return fetchAPI<OrderListResponse>(endpoint);
}

// Get a single order by ID
export async function getOrder(id: string): Promise<Order> {
  return fetchAPI<Order>(`/api/orders/${id}`);
}

// Create a new order
export async function createOrder(params: AddOrderParams): Promise<Order> {
  return fetchAPI<Order>('/api/orders', {
    method: 'POST',
    body: JSON.stringify(params),
  });
}

// Update an existing order
export async function updateOrder(id: string, params: UpdateOrderParams): Promise<Order> {
  return fetchAPI<Order>(`/api/orders/${id}`, {
    method: 'PUT',
    body: JSON.stringify(params),
  });
}

// Delete an order
export async function deleteOrder(id: string): Promise<void> {
  await fetchAPI<void>(`/api/orders/${id}`, {
    method: 'DELETE',
  });
}

// Archive an order
export async function archiveOrder(id: string): Promise<Order> {
  return updateOrder(id, { archived: true });
}

// Unarchive an order
export async function unarchiveOrder(id: string): Promise<Order> {
  return updateOrder(id, { archived: false });
}

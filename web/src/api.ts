import type { AggregatedResponse, FeedItem, FilterParams, SourcesResponse } from './types';

const API_BASE = import.meta.env.VITE_API_BASE_URL || '';

// Custom error class for rate limiting
export class RateLimitError extends Error {
  constructor(message: string) {
    super(message);
    this.name = 'RateLimitError';
  }
}

async function fetchAPI<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Request failed' }));
    
    // Throw specific error for rate limiting
    if (response.status === 429) {
      throw new RateLimitError(error.message || 'Rate limit exceeded');
    }
    
    throw new Error(error.message || `HTTP ${response.status}`);
  }

  return response.json();
}

export async function getItems(params?: FilterParams): Promise<AggregatedResponse> {
  const searchParams = new URLSearchParams();
  
  if (params?.limit) searchParams.set('limit', params.limit.toString());
  if (params?.sources?.length) searchParams.set('sources', params.sources.join(','));
  if (params?.sourceType) searchParams.set('sourceType', params.sourceType);
  if (params?.query) searchParams.set('q', params.query);
  if (params?.sort) searchParams.set('sort', params.sort);
  if (params?.fromDate) searchParams.set('fromDate', params.fromDate);
  if (params?.toDate) searchParams.set('toDate', params.toDate);

  const query = searchParams.toString();
  return fetchAPI<AggregatedResponse>(`/api/items${query ? `?${query}` : ''}`);
}

export async function getItem(id: string): Promise<FeedItem> {
  return fetchAPI<FeedItem>(`/api/items/${id}`);
}

export async function getSources(): Promise<SourcesResponse> {
  return fetchAPI<SourcesResponse>('/api/sources');
}

export async function refreshFeeds(sources?: string[]): Promise<AggregatedResponse> {
  return fetchAPI<AggregatedResponse>('/api/refresh', {
    method: 'POST',
    body: JSON.stringify({ sources }),
  });
}

export async function checkHealth(): Promise<{ status: string; timestamp: string }> {
  return fetchAPI('/health');
}

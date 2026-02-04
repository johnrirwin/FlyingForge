import type {
  AuthResponse,
  AuthTokens,
  GoogleLoginParams,
  RefreshParams,
  User,
} from './authTypes';

const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';

// Token storage keys
const ACCESS_TOKEN_KEY = 'access_token';
const REFRESH_TOKEN_KEY = 'refresh_token';

// Get stored tokens
export function getStoredTokens(): AuthTokens | null {
  const accessToken = localStorage.getItem(ACCESS_TOKEN_KEY);
  const refreshToken = localStorage.getItem(REFRESH_TOKEN_KEY);
  
  if (!accessToken || !refreshToken) {
    return null;
  }
  
  return {
    accessToken,
    refreshToken,
    tokenType: 'Bearer',
    expiresIn: 0, // Unknown from storage
  };
}

// Store tokens
export function storeTokens(tokens: AuthTokens): void {
  localStorage.setItem(ACCESS_TOKEN_KEY, tokens.accessToken);
  localStorage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken);
}

// Clear stored tokens
export function clearStoredTokens(): void {
  localStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
}

// Check if token is expired (with 60s buffer)
export function isTokenExpired(token: string): boolean {
  try {
    const payload = JSON.parse(atob(token.split('.')[1]));
    const exp = payload.exp * 1000; // Convert to milliseconds
    return Date.now() >= exp - 60000; // 60 second buffer
  } catch {
    return true;
  }
}

// Make authenticated request
async function authFetch(
  endpoint: string,
  options: RequestInit = {}
): Promise<Response> {
  const tokens = getStoredTokens();
  
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...options.headers,
  };
  
  if (tokens?.accessToken) {
    (headers as Record<string, string>)['Authorization'] = `Bearer ${tokens.accessToken}`;
  }
  
  return fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  });
}

// Handle API response
async function handleResponse<T>(response: Response): Promise<T> {
  const data = await response.json();
  
  if (!response.ok) {
    throw {
      code: data.code || 'unknown_error',
      message: data.message || data.error || 'An error occurred',
    };
  }
  
  return data as T;
}

// Auth API functions

export async function loginWithGoogle(params: GoogleLoginParams): Promise<AuthResponse> {
  const response = await authFetch('/api/auth/google', {
    method: 'POST',
    body: JSON.stringify(params),
  });
  
  const data = await handleResponse<AuthResponse>(response);
  storeTokens(data.tokens);
  return data;
}

export async function refreshTokens(): Promise<AuthTokens> {
  const tokens = getStoredTokens();
  
  if (!tokens?.refreshToken) {
    throw { code: 'no_refresh_token', message: 'No refresh token available' };
  }
  
  const params: RefreshParams = { refreshToken: tokens.refreshToken };
  
  const response = await fetch(`${API_BASE}/api/auth/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(params),
  });
  
  const data = await handleResponse<AuthTokens>(response);
  storeTokens(data);
  return data;
}

export async function logout(): Promise<void> {
  try {
    await authFetch('/api/auth/logout', { method: 'POST' });
  } finally {
    clearStoredTokens();
  }
}

export async function getCurrentUser(): Promise<User> {
  const response = await authFetch('/api/auth/me');
  return handleResponse<User>(response);
}

// Auto-refresh tokens if needed before making a request
export async function ensureValidToken(): Promise<string | null> {
  const tokens = getStoredTokens();
  
  if (!tokens?.accessToken) {
    return null;
  }
  
  if (isTokenExpired(tokens.accessToken)) {
    try {
      const newTokens = await refreshTokens();
      return newTokens.accessToken;
    } catch {
      clearStoredTokens();
      return null;
    }
  }
  
  return tokens.accessToken;
}

// Make an authenticated API call (auto-refreshes tokens)
export async function authenticatedFetch(
  endpoint: string,
  options: RequestInit = {}
): Promise<Response> {
  const token = await ensureValidToken();
  
  if (!token) {
    throw { code: 'not_authenticated', message: 'Not authenticated' };
  }
  
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${token}`,
    ...options.headers,
  };
  
  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  });
  
  // If we get 401, try to refresh and retry once
  if (response.status === 401) {
    try {
      const newTokens = await refreshTokens();
      (headers as Record<string, string>)['Authorization'] = `Bearer ${newTokens.accessToken}`;
      
      return fetch(`${API_BASE}${endpoint}`, {
        ...options,
        headers,
      });
    } catch {
      clearStoredTokens();
      throw { code: 'session_expired', message: 'Session expired, please log in again' };
    }
  }
  
  return response;
}

import { createContext, useReducer, useEffect, useCallback, useRef } from 'react';
import type { ReactNode } from 'react';
import type {
  AuthState,
  AuthAction,
  GoogleLoginParams,
  AuthError,
  User,
} from '../authTypes';
import * as authApi from '../authApi';
import { trackEvent } from '../hooks/useGoogleAnalytics';
import { dispatchAuthExpired, getCurrentPathWithSearchAndHash } from '../authRouting';

type Auth401Listener = () => void;

function requestHasAuthorizationHeader(
  input: RequestInfo | URL,
  init?: RequestInit,
): boolean {
  const initHeaders = new Headers(init?.headers ?? {});
  if (initHeaders.has('Authorization')) {
    return true;
  }

  if (input instanceof Request) {
    return input.headers.has('Authorization');
  }

  return false;
}

function getRequestPathname(input: RequestInfo | URL): string | null {
  const rawUrl = typeof input === 'string'
    ? input
    : input instanceof URL
      ? input.toString()
      : input.url;

  try {
    return new URL(rawUrl, window.location.origin).pathname;
  } catch {
    return null;
  }
}

class SessionExpiredError extends Error {
  code = 'session_expired' as const;

  constructor() {
    super('Session expired');
    this.name = 'SessionExpiredError';
  }
}

const auth401Listeners = new Set<Auth401Listener>();
let authFetchWrapperRefCount = 0;
let originalWindowFetch: typeof window.fetch | null = null;

function notifyAuth401Listeners() {
  for (const listener of auth401Listeners) {
    try {
      listener();
    } catch {
      // Keep processing remaining listeners.
    }
  }
}

function shouldHandleAuth401(
  input: RequestInfo | URL,
  init: RequestInit | undefined,
  response: Response,
): boolean {
  if (response.status !== 401 || !requestHasAuthorizationHeader(input, init)) {
    return false;
  }

  const requestPathname = getRequestPathname(input);
  return requestPathname !== '/api/auth/logout';
}

function installAuthFetchWrapper() {
  if (authFetchWrapperRefCount === 0) {
    originalWindowFetch = window.fetch;

    const wrappedFetch: typeof window.fetch = async (input, init) => {
      const response = await originalWindowFetch!(input, init);

      if (shouldHandleAuth401(input, init, response)) {
        notifyAuth401Listeners();
        throw new SessionExpiredError();
      }

      return response;
    };

    window.fetch = wrappedFetch;
  }

  authFetchWrapperRefCount += 1;
}

function uninstallAuthFetchWrapper() {
  authFetchWrapperRefCount = Math.max(0, authFetchWrapperRefCount - 1);

  if (authFetchWrapperRefCount === 0 && originalWindowFetch) {
    window.fetch = originalWindowFetch;
    originalWindowFetch = null;
  }
}

function subscribeToAuth401(listener: Auth401Listener): () => void {
  installAuthFetchWrapper();
  auth401Listeners.add(listener);

  return () => {
    auth401Listeners.delete(listener);
    uninstallAuthFetchWrapper();
  };
}

// Initial state
const initialState: AuthState = {
  user: null,
  tokens: null,
  isLoading: true,
  isAuthenticated: false,
  error: null,
};

// Reducer
function authReducer(state: AuthState, action: AuthAction): AuthState {
  switch (action.type) {
    case 'AUTH_START':
      return { ...state, isLoading: true, error: null };
    case 'AUTH_SUCCESS':
      return {
        ...state,
        user: action.payload.user,
        tokens: action.payload.tokens,
        isLoading: false,
        isAuthenticated: true,
        error: null,
      };
    case 'AUTH_ERROR':
      return {
        ...state,
        isLoading: false,
        error: action.payload,
      };
    case 'AUTH_LOGOUT':
      return {
        ...state,
        user: null,
        tokens: null,
        isLoading: false,
        isAuthenticated: false,
        error: null,
      };
    case 'REFRESH_TOKENS':
      return {
        ...state,
        tokens: action.payload,
      };
    case 'UPDATE_USER':
      return {
        ...state,
        user: state.user ? { ...state.user, ...action.payload } : null,
      };
    case 'CLEAR_ERROR':
      return { ...state, error: null };
    default:
      return state;
  }
}

// Context type - exported for use by useAuth hook
export interface AuthContextType extends AuthState {
  loginWithGoogle: (params: GoogleLoginParams) => Promise<void>;
  logout: () => Promise<void>;
  updateUser: (updates: Partial<User>) => void;
  clearError: () => void;
}

// Create context - exported for use by useAuth hook
export const AuthContext = createContext<AuthContextType | null>(null);

// Provider component
interface AuthProviderProps {
  children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [state, dispatch] = useReducer(authReducer, initialState);
  const authFailureHandledRef = useRef(false);

  const handleSessionExpired = useCallback(() => {
    if (authFailureHandledRef.current) {
      return;
    }

    authFailureHandledRef.current = true;
    authApi.clearStoredTokens();
    dispatch({ type: 'AUTH_LOGOUT' });

    if (window.location.pathname !== '/login' && window.location.pathname !== '/auth/callback') {
      dispatchAuthExpired(getCurrentPathWithSearchAndHash());
    }
  }, []);

  useEffect(() => {
    return subscribeToAuth401(() => {
      handleSessionExpired();
    });
  }, [handleSessionExpired]);

  useEffect(() => {
    if (state.isAuthenticated) {
      authFailureHandledRef.current = false;
    }
  }, [state.isAuthenticated]);

  // Check for existing session on mount
  useEffect(() => {
    async function checkAuth() {
      const tokens = authApi.getStoredTokens();
      
      if (!tokens) {
        dispatch({ type: 'AUTH_LOGOUT' });
        return;
      }

      try {
        // Try to get current user
        const user = await authApi.getCurrentUser();
        dispatch({
          type: 'AUTH_SUCCESS',
          payload: { user, tokens },
        });
      } catch {
        // Token is invalid, clear it
        authApi.clearStoredTokens();
        dispatch({ type: 'AUTH_LOGOUT' });
      }
    }

    checkAuth();
  }, []);

  const loginWithGoogle = useCallback(async (params: GoogleLoginParams) => {
    dispatch({ type: 'AUTH_START' });
    try {
      const response = await authApi.loginWithGoogle(params);
      dispatch({
        type: 'AUTH_SUCCESS',
        payload: { user: response.user, tokens: response.tokens },
      });
      // Track login/signup for GA4 conversions
      // Note: response.isNewUser would distinguish signup from login if available
      trackEvent('login', { method: 'google' });
    } catch (error) {
      dispatch({ type: 'AUTH_ERROR', payload: error as AuthError });
      throw error;
    }
  }, []);

  const logout = useCallback(async () => {
    try {
      await authApi.logout();
    } finally {
      dispatch({ type: 'AUTH_LOGOUT' });
    }
  }, []);

  const clearError = useCallback(() => {
    dispatch({ type: 'CLEAR_ERROR' });
  }, []);

  const updateUser = useCallback((updates: Partial<User>) => {
    dispatch({ type: 'UPDATE_USER', payload: updates });
  }, []);

  const value: AuthContextType = {
    ...state,
    loginWithGoogle,
    logout,
    updateUser,
    clearError,
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
}

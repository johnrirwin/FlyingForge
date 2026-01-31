import { createContext, useContext, useReducer, useEffect, useCallback } from 'react';
import type { ReactNode } from 'react';
import type {
  AuthState,
  AuthAction,
  SignupParams,
  LoginParams,
  GoogleLoginParams,
  AuthError,
} from '../authTypes';
import * as authApi from '../authApi';

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
    case 'CLEAR_ERROR':
      return { ...state, error: null };
    default:
      return state;
  }
}

// Context type
interface AuthContextType extends AuthState {
  signup: (params: SignupParams) => Promise<void>;
  login: (params: LoginParams) => Promise<void>;
  loginWithGoogle: (params: GoogleLoginParams) => Promise<void>;
  logout: () => Promise<void>;
  clearError: () => void;
}

// Create context
const AuthContext = createContext<AuthContextType | null>(null);

// Provider component
interface AuthProviderProps {
  children: ReactNode;
}

export function AuthProvider({ children }: AuthProviderProps) {
  const [state, dispatch] = useReducer(authReducer, initialState);

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

  const signup = useCallback(async (params: SignupParams) => {
    dispatch({ type: 'AUTH_START' });
    try {
      const response = await authApi.signup(params);
      dispatch({
        type: 'AUTH_SUCCESS',
        payload: { user: response.user, tokens: response.tokens },
      });
    } catch (error) {
      dispatch({ type: 'AUTH_ERROR', payload: error as AuthError });
      throw error;
    }
  }, []);

  const login = useCallback(async (params: LoginParams) => {
    dispatch({ type: 'AUTH_START' });
    try {
      const response = await authApi.login(params);
      dispatch({
        type: 'AUTH_SUCCESS',
        payload: { user: response.user, tokens: response.tokens },
      });
    } catch (error) {
      dispatch({ type: 'AUTH_ERROR', payload: error as AuthError });
      throw error;
    }
  }, []);

  const loginWithGoogle = useCallback(async (params: GoogleLoginParams) => {
    dispatch({ type: 'AUTH_START' });
    try {
      const response = await authApi.loginWithGoogle(params);
      dispatch({
        type: 'AUTH_SUCCESS',
        payload: { user: response.user, tokens: response.tokens },
      });
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

  const value: AuthContextType = {
    ...state,
    signup,
    login,
    loginWithGoogle,
    logout,
    clearError,
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
}

// Hook for using auth context
export function useAuth(): AuthContextType {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}

// Export context for advanced usage
export { AuthContext };

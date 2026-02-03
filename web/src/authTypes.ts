// Auth types for the frontend

export type AvatarType = 'google' | 'custom';

export interface User {
  id: string;
  email: string;
  displayName: string;
  avatarUrl?: string;
  status: 'active' | 'suspended' | 'deleted';
  emailVerified: boolean;
  createdAt: string;
  lastLoginAt?: string;
  // Profile fields
  callSign?: string;
  googleName?: string;
  googleAvatarUrl?: string;
  avatarType?: AvatarType;
  customAvatarUrl?: string;
}

// Extended user profile response from /api/me/profile
export interface UserProfile extends User {
  effectiveAvatarUrl: string;
  updatedAt: string;
}

// Parameters for updating profile
export interface UpdateProfileParams {
  callSign?: string;
  displayName?: string;
  avatarType?: AvatarType;
}

export interface AuthTokens {
  accessToken: string;
  refreshToken: string;
  tokenType: string;
  expiresIn: number;
}

export interface AuthResponse {
  user: User;
  tokens: AuthTokens;
  isNewUser?: boolean;
  isLinked?: boolean;
}

export interface SignupParams {
  email: string;
  password: string;
  displayName?: string;
  callSign?: string;
}

export interface LoginParams {
  email: string;
  password: string;
}

export interface GoogleLoginParams {
  idToken?: string;
  code?: string;
  redirectUri?: string;
}

export interface RefreshParams {
  refreshToken: string;
}

export interface AuthError {
  code: string;
  message: string;
}

export interface AuthState {
  user: User | null;
  tokens: AuthTokens | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  error: AuthError | null;
}

// Context actions
export type AuthAction =
  | { type: 'AUTH_START' }
  | { type: 'AUTH_SUCCESS'; payload: { user: User; tokens: AuthTokens } }
  | { type: 'AUTH_ERROR'; payload: AuthError }
  | { type: 'AUTH_LOGOUT' }
  | { type: 'REFRESH_TOKENS'; payload: AuthTokens }
  | { type: 'UPDATE_USER'; payload: Partial<User> }
  | { type: 'CLEAR_ERROR' };

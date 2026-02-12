export type AdminUserStatus = 'active' | 'disabled' | 'pending';

export interface AdminUser {
  id: string;
  email: string;
  displayName: string;
  callSign?: string;
  status: AdminUserStatus;
  isAdmin: boolean;
  isContentAdmin: boolean;
  isGearAdmin?: boolean;
  avatarUrl?: string;
  googleAvatarUrl?: string;
  customAvatarUrl?: string;
  avatarType?: 'google' | 'custom';
  createdAt: string;
  updatedAt: string;
  lastLoginAt?: string;
}

export interface AdminUserSearchParams {
  query?: string;
  status?: AdminUserStatus;
  limit?: number;
  offset?: number;
}

export interface AdminUsersResponse {
  users: AdminUser[];
  totalCount: number;
}

export interface AdminUpdateUserParams {
  status?: AdminUserStatus;
  isAdmin?: boolean;
  isContentAdmin?: boolean;
  isGearAdmin?: boolean;
}

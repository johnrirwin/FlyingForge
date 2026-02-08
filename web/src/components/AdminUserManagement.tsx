import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { AdminUser, AdminUserStatus } from '../adminUserTypes';
import { adminDeleteUser, adminDeleteUserAvatar, adminGetUser, adminSearchUsers, adminUpdateUser } from '../adminApi';

interface AdminUserManagementProps {
  isAdmin: boolean;
  currentUserId?: string;
  authLoading?: boolean;
}

const PAGE_SIZE = 25;

function formatDate(value?: string): string {
  if (!value) return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '—';
  return date.toLocaleString();
}

function getUserCallSign(user: Pick<AdminUser, 'callSign'>): string {
  const callSign = user.callSign?.trim();
  return callSign ? callSign : 'No Callsign';
}

function getUserDisplayName(user: Pick<AdminUser, 'displayName'>): string {
  const displayName = user.displayName?.trim();
  return displayName ? displayName : 'Unnamed User';
}

function getRoleLabel(user: Pick<AdminUser, 'isAdmin' | 'isGearAdmin'>): string {
  if (user.isAdmin) return 'Admin';
  if (user.isGearAdmin) return 'Gear Admin';
  return 'User';
}

function getStatusLabel(status: AdminUserStatus): string {
  switch (status) {
    case 'active':
      return 'Active';
    case 'disabled':
      return 'Disabled';
    case 'pending':
      return 'Pending';
    default:
      return status;
  }
}

function getStatusBadgeClass(status: AdminUserStatus): string {
  switch (status) {
    case 'active':
      return 'bg-emerald-500/20 text-emerald-300 border-emerald-500/40';
    case 'disabled':
      return 'bg-red-500/20 text-red-300 border-red-500/40';
    case 'pending':
      return 'bg-amber-500/20 text-amber-300 border-amber-500/40';
    default:
      return 'bg-slate-700 text-slate-200 border-slate-600';
  }
}

function getUserAvatarURL(user: AdminUser | null): string | null {
  if (!user) return null;
  if (user.avatarType === 'custom' && user.customAvatarUrl) return user.customAvatarUrl;
  if (user.googleAvatarUrl) return user.googleAvatarUrl;
  if (user.avatarUrl) return user.avatarUrl;
  return null;
}

export function AdminUserManagement({ isAdmin, currentUserId, authLoading }: AdminUserManagementProps) {
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [query, setQuery] = useState('');
  const [appliedQuery, setAppliedQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<AdminUserStatus | ''>('');
  const [hasMore, setHasMore] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);

  const [selectedUserID, setSelectedUserID] = useState<string | null>(null);
  const [profileUser, setProfileUser] = useState<AdminUser | null>(null);
  const [profileStatus, setProfileStatus] = useState<AdminUserStatus>('active');
  const [profileIsAdmin, setProfileIsAdmin] = useState(false);
  const [profileIsGearAdmin, setProfileIsGearAdmin] = useState(false);
  const [isProfileLoading, setIsProfileLoading] = useState(false);
  const [isSavingProfile, setIsSavingProfile] = useState(false);
  const [isRemovingAvatar, setIsRemovingAvatar] = useState(false);
  const [profileError, setProfileError] = useState<string | null>(null);
  const [toastMessage, setToastMessage] = useState<string | null>(null);

  const [deleteTargetUser, setDeleteTargetUser] = useState<AdminUser | null>(null);
  const [deleteConfirmText, setDeleteConfirmText] = useState('');
  const [isDeletingUser, setIsDeletingUser] = useState(false);
  const [showRemoveAvatarModal, setShowRemoveAvatarModal] = useState(false);
  const [removeAvatarConfirmText, setRemoveAvatarConfirmText] = useState('');

  const loadMoreRef = useRef<HTMLDivElement>(null);
  const currentOffsetRef = useRef(0);
  const isLoadingRef = useRef(false);
  const latestLoadRequestRef = useRef(0);

  const isProfileSelf = profileUser?.id === currentUserId;
  const profileAvatarURL = useMemo(() => getUserAvatarURL(profileUser), [profileUser]);

  const loadUsers = useCallback(async (reset = false, forceRefresh = false) => {
    if (!isAdmin) return;

    // Prevent concurrent loads by default; allow forced resets to supersede in-flight loads.
    if (isLoadingRef.current && !(reset && forceRefresh)) return;
    isLoadingRef.current = true;
    const requestID = ++latestLoadRequestRef.current;

    if (reset) {
      setIsLoading(true);
      currentOffsetRef.current = 0;
      setHasMore(true);
    } else {
      setIsLoadingMore(true);
    }
    setError(null);

    const offset = currentOffsetRef.current;

    try {
      const response = await adminSearchUsers({
        query: appliedQuery || undefined,
        status: statusFilter || undefined,
        limit: PAGE_SIZE,
        offset,
      });

      // Ignore stale responses from superseded requests.
      if (requestID !== latestLoadRequestRef.current) {
        return;
      }

      if (reset) {
        setUsers(response.users || []);
      } else {
        setUsers((prev) => [...prev, ...(response.users || [])]);
      }

      const fetchedCount = response.users?.length || 0;
      currentOffsetRef.current = offset + fetchedCount;
      setTotalCount(response.totalCount || 0);
      setHasMore(fetchedCount === PAGE_SIZE && currentOffsetRef.current < (response.totalCount || 0));
    } catch (err) {
      if (requestID !== latestLoadRequestRef.current) {
        return;
      }
      setError(err instanceof Error ? err.message : 'Failed to load users');
      if (reset) {
        setUsers([]);
        setTotalCount(0);
      }
      setHasMore(false);
    } finally {
      if (requestID === latestLoadRequestRef.current) {
        setIsLoading(false);
        setIsLoadingMore(false);
        isLoadingRef.current = false;
      }
    }
  }, [appliedQuery, isAdmin, statusFilter]);

  useEffect(() => {
    if (isAdmin) {
      void loadUsers(true, true);
    }
  }, [isAdmin, loadUsers]);

  useEffect(() => {
    if (typeof IntersectionObserver === 'undefined') return;
    const node = loadMoreRef.current;
    if (!node || !hasMore) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting) {
          void loadUsers(false);
        }
      },
      { threshold: 0.1 }
    );

    observer.observe(node);
    return () => observer.disconnect();
  }, [hasMore, loadUsers]);

  useEffect(() => {
    if (!selectedUserID) {
      setProfileUser(null);
      setProfileError(null);
      setShowRemoveAvatarModal(false);
      setRemoveAvatarConfirmText('');
      return;
    }

    let cancelled = false;
    setIsProfileLoading(true);
    setProfileError(null);

    void adminGetUser(selectedUserID)
      .then((user) => {
        if (cancelled) return;
        setProfileUser(user);
        setProfileStatus(user.status);
        setProfileIsAdmin(user.isAdmin);
        setProfileIsGearAdmin(user.isGearAdmin);
      })
      .catch((err) => {
        if (cancelled) return;
        setProfileError(err instanceof Error ? err.message : 'Failed to load user profile');
      })
      .finally(() => {
        if (cancelled) return;
        setIsProfileLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [selectedUserID]);

  useEffect(() => {
    if (!toastMessage) return;
    const timer = window.setTimeout(() => {
      setToastMessage(null);
    }, 2500);
    return () => window.clearTimeout(timer);
  }, [toastMessage]);

  const handleSearch = useCallback(() => {
    const trimmed = query.trim();
    setAppliedQuery((prev) => {
      if (prev === trimmed) {
        void loadUsers(true, true);
      }
      return trimmed;
    });
  }, [loadUsers, query]);

  const handleOpenProfile = useCallback((user: AdminUser) => {
    setSelectedUserID(user.id);
    setProfileUser(user);
    setProfileStatus(user.status);
    setProfileIsAdmin(user.isAdmin);
    setProfileIsGearAdmin(user.isGearAdmin);
    setProfileError(null);
  }, []);

  const handleCloseProfile = useCallback(() => {
    if (isSavingProfile || isRemovingAvatar) return;
    setSelectedUserID(null);
    setProfileUser(null);
    setProfileError(null);
    setShowRemoveAvatarModal(false);
    setRemoveAvatarConfirmText('');
  }, [isRemovingAvatar, isSavingProfile]);

  const applyUserUpdateToList = useCallback((updated: AdminUser) => {
    setUsers((prev) => prev.map((item) => (item.id === updated.id ? { ...item, ...updated } : item)));
  }, []);

  const handleSaveProfile = useCallback(async () => {
    if (!profileUser) return;

    setIsSavingProfile(true);
    setProfileError(null);
    try {
      const updated = await adminUpdateUser(profileUser.id, {
        status: profileStatus,
        isAdmin: profileIsAdmin,
        isGearAdmin: profileIsGearAdmin,
      });
      setProfileUser(updated);
      setProfileStatus(updated.status);
      setProfileIsAdmin(updated.isAdmin);
      setProfileIsGearAdmin(updated.isGearAdmin);
      applyUserUpdateToList(updated);
      setToastMessage('User was updated');
    } catch (err) {
      setProfileError(err instanceof Error ? err.message : 'Failed to save user profile');
    } finally {
      setIsSavingProfile(false);
    }
  }, [applyUserUpdateToList, profileIsAdmin, profileIsGearAdmin, profileStatus, profileUser]);

  const handleOpenRemoveAvatarModal = useCallback(() => {
    if (!profileUser || !profileAvatarURL || isRemovingAvatar) return;
    setShowRemoveAvatarModal(true);
    setRemoveAvatarConfirmText('');
  }, [isRemovingAvatar, profileAvatarURL, profileUser]);

  const handleCancelRemoveAvatar = useCallback(() => {
    if (isRemovingAvatar) return;
    setShowRemoveAvatarModal(false);
    setRemoveAvatarConfirmText('');
  }, [isRemovingAvatar]);

  const handleConfirmRemoveAvatar = useCallback(async () => {
    if (!profileUser) return;
    if (removeAvatarConfirmText.trim().toLowerCase() !== 'delete') return;

    setIsRemovingAvatar(true);
    setProfileError(null);
    try {
      const updated = await adminDeleteUserAvatar(profileUser.id);
      setProfileUser(updated);
      applyUserUpdateToList(updated);
      setShowRemoveAvatarModal(false);
      setRemoveAvatarConfirmText('');
      setToastMessage('Profile picture removed');
    } catch (err) {
      setProfileError(err instanceof Error ? err.message : 'Failed to remove profile picture');
    } finally {
      setIsRemovingAvatar(false);
    }
  }, [applyUserUpdateToList, profileUser, removeAvatarConfirmText]);

  const handleDeleteClick = useCallback((user: AdminUser) => {
    setDeleteTargetUser(user);
    setDeleteConfirmText('');
    setError(null);
    setShowRemoveAvatarModal(false);
    setRemoveAvatarConfirmText('');
  }, []);

  const handleCancelDelete = useCallback(() => {
    if (isDeletingUser) return;
    setDeleteTargetUser(null);
    setDeleteConfirmText('');
  }, [isDeletingUser]);

  const handleConfirmDelete = useCallback(async () => {
    if (!deleteTargetUser) return;
    if (deleteConfirmText.trim().toLowerCase() !== 'delete') return;

    setIsDeletingUser(true);
    setError(null);
    try {
      await adminDeleteUser(deleteTargetUser.id);
      if (selectedUserID === deleteTargetUser.id) {
        setSelectedUserID(null);
        setProfileUser(null);
      }
      setDeleteTargetUser(null);
      setDeleteConfirmText('');
      setToastMessage('User deleted');
      await loadUsers(true, true);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete user');
    } finally {
      setIsDeletingUser(false);
    }
  }, [deleteConfirmText, deleteTargetUser, loadUsers, selectedUserID]);

  // Show loading while auth state is being determined
  if (authLoading) {
    return (
      <div className="p-8 text-center">
        <div className="w-8 h-8 border-2 border-primary-500/30 border-t-primary-500 rounded-full animate-spin mx-auto" />
        <p className="text-slate-400 mt-4">Loading...</p>
      </div>
    );
  }

  if (!isAdmin) {
    return (
      <div className="p-8 text-center">
        <h1 className="text-2xl font-bold text-red-400 mb-4">Access Denied</h1>
        <p className="text-slate-400">You must be an admin to access this page.</p>
      </div>
    );
  }

  return (
    <>
      <div className="flex-1 flex flex-col min-h-0">
        <div className="bg-slate-800 border-b border-slate-700 px-4 md:px-6 py-4">
          <h1 className="text-lg md:text-2xl font-bold text-white">User Admin</h1>
          <p className="text-slate-400 mt-1 text-sm">Search users, review profile details, manage roles, and delete accounts.</p>

          <div className="mt-4 flex flex-col md:flex-row gap-3">
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  handleSearch();
                }
              }}
              placeholder="Search by email, display name, or callsign..."
              className="flex-1 px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
            <select
              value={statusFilter}
              onChange={(e) => {
                setStatusFilter(e.target.value as AdminUserStatus | '');
              }}
              className="px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              <option value="">All statuses</option>
              <option value="active">Active</option>
              <option value="disabled">Disabled</option>
              <option value="pending">Pending</option>
            </select>
            <button
              onClick={handleSearch}
              className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-lg font-medium transition-colors"
            >
              Search
            </button>
          </div>
        </div>

        <div className="flex-1 overflow-auto p-4 md:p-6">
          {error && (
            <div className="mb-4 p-3 rounded-lg border border-red-500/40 bg-red-500/10 text-red-300 text-sm">
              {error}
            </div>
          )}

          <div className="mb-3 text-sm text-slate-400">
            {isLoading ? 'Loading users...' : `${totalCount} user${totalCount === 1 ? '' : 's'} found`}
          </div>

          {/* Desktop table */}
          <div className="hidden md:block border border-slate-800 rounded-xl overflow-hidden">
            <table className="w-full table-fixed text-sm">
              <thead className="bg-slate-900 text-slate-400">
                <tr>
                  <th className="w-1/4 text-left px-4 py-3 font-medium">User</th>
                  <th className="w-1/4 text-center px-4 py-3 font-medium">Status</th>
                  <th className="w-1/4 text-center px-4 py-3 font-medium">Roles</th>
                  <th className="w-1/4 text-right px-4 py-3 font-medium">Last Login</th>
                </tr>
              </thead>
              <tbody>
                {users.map((user) => {
                  const isSelf = user.id === currentUserId;
                  const isSelected = selectedUserID === user.id;
                  return (
                    <tr
                      key={user.id}
                      onClick={() => handleOpenProfile(user)}
                      onKeyDown={(event) => {
                        if (event.key === 'Enter' || event.key === ' ') {
                          event.preventDefault();
                          handleOpenProfile(user);
                        }
                      }}
                      role="button"
                      tabIndex={0}
                      aria-label={`Open profile for ${getUserCallSign(user)}`}
                      className={`border-t border-slate-800 transition-colors cursor-pointer ${
                        isSelected ? 'bg-primary-600/10' : 'bg-slate-900/40 hover:bg-slate-800/50'
                      }`}
                    >
                      <td className="w-1/4 px-4 py-3 align-middle">
                        <div className="font-medium text-white truncate">{getUserCallSign(user)}</div>
                        <div className="text-slate-300 break-all">{user.email}</div>
                        <div className="text-xs text-slate-500 truncate">{getUserDisplayName(user)}</div>
                        {isSelf && <div className="text-[11px] text-slate-500 mt-1">Current user</div>}
                      </td>
                      <td className="w-1/4 px-4 py-3 text-center align-middle">
                        <span className={`inline-flex items-center px-2.5 py-1 rounded-full border text-xs font-medium ${getStatusBadgeClass(user.status)}`}>
                          {getStatusLabel(user.status)}
                        </span>
                      </td>
                      <td className="w-1/4 px-4 py-3 text-center text-slate-300 align-middle">{getRoleLabel(user)}</td>
                      <td className="w-1/4 px-4 py-3 text-right text-slate-400 align-middle whitespace-nowrap">{formatDate(user.lastLoginAt)}</td>
                    </tr>
                  );
                })}
                {!isLoading && users.length === 0 && (
                  <tr>
                    <td colSpan={4} className="px-4 py-8 text-center text-slate-400">
                      No users found.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>

          {/* Mobile cards */}
          <div className="md:hidden space-y-3">
            {users.map((user) => {
              const isSelf = user.id === currentUserId;
              const isSelected = selectedUserID === user.id;

              return (
                <button
                  key={user.id}
                  onClick={() => handleOpenProfile(user)}
                  className={`w-full text-left border rounded-xl p-4 transition-colors ${
                    isSelected
                      ? 'border-primary-500/50 bg-primary-600/10'
                      : 'border-slate-800 bg-slate-900/40 hover:bg-slate-800/50'
                  }`}
                >
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <p className="font-medium text-white truncate">{getUserCallSign(user)}</p>
                      <p className="text-slate-300 break-all">{user.email}</p>
                      <p className="text-xs text-slate-500 truncate">{getUserDisplayName(user)}</p>
                      {isSelf && <p className="text-[11px] text-slate-500 mt-1">Current user</p>}
                    </div>
                    <span className={`inline-flex items-center px-2.5 py-1 rounded-full border text-xs font-medium shrink-0 ${getStatusBadgeClass(user.status)}`}>
                      {getStatusLabel(user.status)}
                    </span>
                  </div>

                  <div className="mt-3 grid grid-cols-2 gap-3 text-xs">
                    <div>
                      <p className="text-slate-500 uppercase tracking-wide">Role</p>
                      <p className="text-slate-300 mt-0.5">{getRoleLabel(user)}</p>
                    </div>
                    <div className="text-right">
                      <p className="text-slate-500 uppercase tracking-wide">Last Login</p>
                      <p className="text-slate-300 mt-0.5">{formatDate(user.lastLoginAt)}</p>
                    </div>
                  </div>
                </button>
              );
            })}

            {!isLoading && users.length === 0 && (
              <div className="border border-slate-800 rounded-xl p-6 text-center text-slate-400 bg-slate-900/40">
                No users found.
              </div>
            )}
          </div>

          {hasMore && (
            <div ref={loadMoreRef} className="py-4 flex justify-center">
              {isLoadingMore ? (
                <div className="flex items-center gap-2 text-slate-400 text-sm">
                  <div className="w-4 h-4 border-2 border-slate-600 border-t-primary-500 rounded-full animate-spin" />
                  Loading more users...
                </div>
              ) : (
                <span className="text-xs text-slate-500">Scroll to load more</span>
              )}
            </div>
          )}

          {!isLoading && !hasMore && users.length > 0 && (
            <div className="pt-1 text-center text-xs text-slate-500">All users loaded</div>
          )}
        </div>
      </div>

      {/* User Profile Modal */}
      {selectedUserID && (
        <div
          className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4"
          onClick={(e) => {
            if (e.currentTarget === e.target) {
              handleCloseProfile();
            }
          }}
        >
          <div className="bg-slate-800 rounded-xl w-full max-w-2xl max-h-[90vh] overflow-y-auto border border-slate-700 shadow-2xl">
            <div className="px-6 py-4 border-b border-slate-700 flex items-center justify-between">
              <h3 className="text-lg font-semibold text-white">User Profile</h3>
              <button
                onClick={handleCloseProfile}
                disabled={isSavingProfile || isRemovingAvatar}
                className="text-slate-400 hover:text-white transition-colors"
                aria-label="Close profile modal"
              >
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <div className="p-6">
              {isProfileLoading ? (
                <div className="py-12 text-center">
                  <div className="w-8 h-8 border-2 border-primary-500/30 border-t-primary-500 rounded-full animate-spin mx-auto" />
                  <p className="text-slate-400 mt-4">Loading user profile...</p>
                </div>
              ) : profileUser ? (
                <>
                  <div className="flex items-center gap-4 mb-6">
                    {profileAvatarURL ? (
                      <img src={profileAvatarURL} alt={getUserDisplayName(profileUser)} className="w-20 h-20 rounded-full object-cover border border-slate-600" />
                    ) : (
                      <div className="w-20 h-20 rounded-full bg-primary-600 flex items-center justify-center border border-slate-600">
                        <span className="text-white text-2xl font-semibold">
                          {getUserCallSign(profileUser)[0].toUpperCase()}
                        </span>
                      </div>
                    )}

                    <div className="min-w-0">
                      <p className="text-xl font-semibold text-white truncate">{getUserCallSign(profileUser)}</p>
                      <p className="text-slate-300 truncate">{profileUser.email}</p>
                      <p className="text-sm text-slate-500 truncate">{getUserDisplayName(profileUser)}</p>
                    </div>
                  </div>

                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-5">
                    <div>
                      <label htmlFor="profile-status" className="block text-sm font-medium text-slate-300 mb-2">
                        Account Status
                      </label>
                      <select
                        id="profile-status"
                        value={profileStatus}
                        onChange={(e) => setProfileStatus(e.target.value as AdminUserStatus)}
                        disabled={isSavingProfile || isProfileSelf}
                        className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-primary-500 disabled:opacity-60"
                      >
                        <option value="active">Active</option>
                        <option value="disabled">Disabled</option>
                        <option value="pending">Pending</option>
                      </select>
                    </div>

                    <div>
                      <p className="block text-sm font-medium text-slate-300 mb-2">Roles</p>
                      <label className="flex items-center gap-2 text-slate-200">
                        <input
                          type="checkbox"
                          checked={profileIsAdmin}
                          disabled={isSavingProfile || (isProfileSelf && profileIsAdmin)}
                          onChange={(e) => setProfileIsAdmin(e.target.checked)}
                        />
                        Admin
                      </label>
                      <label className="flex items-center gap-2 text-slate-200 mt-2">
                        <input
                          type="checkbox"
                          checked={profileIsGearAdmin}
                          disabled={isSavingProfile}
                          onChange={(e) => setProfileIsGearAdmin(e.target.checked)}
                        />
                        Gear Admin
                      </label>
                    </div>
                  </div>

                  <div className="text-xs text-slate-500 space-y-1 mb-5">
                    <p>Joined: {formatDate(profileUser.createdAt)}</p>
                    <p>Last login: {formatDate(profileUser.lastLoginAt)}</p>
                  </div>

                  <div className="border-t border-slate-700 pt-4 mb-5">
                    <button
                      onClick={handleOpenRemoveAvatarModal}
                      disabled={isRemovingAvatar || !profileAvatarURL}
                      className="px-4 py-2 bg-red-600/80 hover:bg-red-600 text-white rounded-lg disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      {profileAvatarURL ? 'Remove Profile Picture' : 'No Profile Picture'}
                    </button>
                  </div>

                  {profileError && (
                    <div className="mb-4 p-3 rounded-lg border border-red-500/40 bg-red-500/10 text-red-300 text-sm">
                      {profileError}
                    </div>
                  )}

                  <div className="flex gap-3">
                    <button
                      onClick={() => handleDeleteClick(profileUser)}
                      disabled={isSavingProfile || isRemovingAvatar || isDeletingUser || isProfileSelf}
                      className="flex-1 px-4 py-2 bg-red-600/80 hover:bg-red-600 text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                      Delete User
                    </button>
                    <button
                      onClick={() => void handleSaveProfile()}
                      disabled={isSavingProfile || isRemovingAvatar}
                      className="flex-1 px-4 py-2 bg-primary-600 hover:bg-primary-500 text-white rounded-lg font-medium transition-colors disabled:opacity-50"
                    >
                      {isSavingProfile ? 'Saving...' : 'Save Changes'}
                    </button>
                  </div>
                </>
              ) : (
                <p className="text-slate-400">Unable to load user profile.</p>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Remove Profile Picture Confirmation Modal */}
      {showRemoveAvatarModal && profileUser && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-[60] p-4">
          <div className="bg-slate-800 rounded-xl p-6 max-w-md w-full shadow-2xl border border-red-500/50">
            <div className="flex items-start justify-between gap-3 mb-4">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 bg-red-500/20 rounded-full flex items-center justify-center">
                  <svg className="w-5 h-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 10l4.553-4.553a1 1 0 00-1.414-1.414L13.586 8.586M9 14l-4.553 4.553a1 1 0 001.414 1.414L10.414 15.414M15 14a6 6 0 10-6-6 6 6 0 006 6z" />
                  </svg>
                </div>
                <h3 className="text-lg font-semibold text-white">Remove Profile Picture?</h3>
              </div>
              <button
                onClick={handleCancelRemoveAvatar}
                disabled={isRemovingAvatar}
                aria-label="Close remove profile picture modal"
                className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors disabled:opacity-50"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <p className="text-slate-300 mb-3">
              <strong className="text-red-400">This action cannot be undone.</strong> The selected user image will be removed immediately.
            </p>
            <div className="text-sm text-slate-400 space-y-1 mb-4">
              <p><span className="text-slate-200">Callsign:</span> {getUserCallSign(profileUser)}</p>
              <p><span className="text-slate-200">Email:</span> {profileUser.email}</p>
            </div>

            <p className="text-sm text-slate-400 mb-2">
              Type <span className="font-mono text-red-400 bg-red-400/10 px-1.5 py-0.5 rounded">delete</span> to confirm:
            </p>
            <input
              type="text"
              value={removeAvatarConfirmText}
              onChange={(e) => setRemoveAvatarConfirmText(e.target.value)}
              placeholder="Type 'delete' to confirm"
              className="w-full px-4 py-2 bg-slate-700 border border-red-500/50 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-red-500 focus:border-transparent mb-4"
              autoFocus
              disabled={isRemovingAvatar}
            />

            <div className="flex">
              <button
                onClick={() => void handleConfirmRemoveAvatar()}
                disabled={isRemovingAvatar || removeAvatarConfirmText.trim().toLowerCase() !== 'delete'}
                className="w-full px-4 py-2 bg-red-600 hover:bg-red-500 text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isRemovingAvatar ? 'Removing...' : 'Remove Picture'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete User Confirmation Modal */}
      {deleteTargetUser && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-[60] p-4">
          <div className="bg-slate-800 rounded-xl p-6 max-w-md w-full shadow-2xl border border-red-500/50">
            <div className="flex items-start justify-between gap-3 mb-4">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 bg-red-500/20 rounded-full flex items-center justify-center">
                  <svg className="w-5 h-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </div>
                <h3 className="text-lg font-semibold text-white">Delete User Account?</h3>
              </div>
              <button
                onClick={handleCancelDelete}
                disabled={isDeletingUser}
                aria-label="Close delete user modal"
                className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors disabled:opacity-50"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <p className="text-slate-300 mb-3">
              <strong className="text-red-400">This action cannot be undone.</strong> Deleting this user permanently removes account data and related records.
            </p>
            <div className="text-sm text-slate-400 space-y-1 mb-4">
              <p><span className="text-slate-200">Callsign:</span> {getUserCallSign(deleteTargetUser)}</p>
              <p><span className="text-slate-200">Email:</span> {deleteTargetUser.email}</p>
              <p><span className="text-slate-200">Display Name:</span> {getUserDisplayName(deleteTargetUser)}</p>
            </div>

            <p className="text-sm text-slate-400 mb-2">
              Type <span className="font-mono text-red-400 bg-red-400/10 px-1.5 py-0.5 rounded">delete</span> to confirm:
            </p>
            <input
              type="text"
              value={deleteConfirmText}
              onChange={(e) => setDeleteConfirmText(e.target.value)}
              placeholder="Type 'delete' to confirm"
              className="w-full px-4 py-2 bg-slate-700 border border-red-500/50 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-red-500 focus:border-transparent mb-4"
              autoFocus
              disabled={isDeletingUser}
            />

            <div className="flex">
              <button
                onClick={() => void handleConfirmDelete()}
                disabled={isDeletingUser || deleteConfirmText.trim().toLowerCase() !== 'delete'}
                className="w-full px-4 py-2 bg-red-600 hover:bg-red-500 text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isDeletingUser ? 'Deleting...' : 'Delete User'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Success Toast */}
      {toastMessage && (
        <div className="fixed top-4 right-4 z-[70]">
          <div className="px-4 py-3 rounded-lg bg-emerald-600 text-white shadow-2xl border border-emerald-400/50">
            {toastMessage}
          </div>
        </div>
      )}
    </>
  );
}

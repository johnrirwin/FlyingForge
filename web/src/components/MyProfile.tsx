import { useState, useEffect, useRef } from 'react';
import { useAuth } from '../hooks/useAuth';
import { getProfile, updateProfile, uploadAvatar, validateCallSign, deleteProfile } from '../profileApi';
import type { UserProfile, UpdateProfileParams } from '../authTypes';

interface ProfileFormData {
  callSign: string;
  displayName: string;
}

interface PendingAvatar {
  file: File;
  previewUrl: string;
}

export function MyProfile() {
  const { updateUser } = useAuth();
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [formData, setFormData] = useState<ProfileFormData>({
    callSign: '',
    displayName: '',
  });
  const [validationError, setValidationError] = useState<string | null>(null);
  const [pendingAvatar, setPendingAvatar] = useState<PendingAvatar | null>(null);
  const [showClearCallSignModal, setShowClearCallSignModal] = useState(false);
  const [confirmText, setConfirmText] = useState('');
  const [showDeleteAccountModal, setShowDeleteAccountModal] = useState(false);
  const [deleteConfirmText, setDeleteConfirmText] = useState('');
  const [isDeleting, setIsDeleting] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    loadProfile();
  }, []);

  // Cleanup pending avatar preview URL on unmount or state change
  useEffect(() => {
    const avatarToCleanup = pendingAvatar;
    return () => {
      if (avatarToCleanup) {
        URL.revokeObjectURL(avatarToCleanup.previewUrl);
      }
    };
  }, [pendingAvatar]);

  const loadProfile = async () => {
    try {
      setIsLoading(true);
      setError(null);
      const data = await getProfile();
      setProfile(data);
      setFormData({
        callSign: data.callSign || '',
        displayName: data.displayName || '',
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load profile');
    } finally {
      setIsLoading(false);
    }
  };

  const handleCallSignChange = (value: string) => {
    setFormData(prev => ({ ...prev, callSign: value }));
    const error = validateCallSign(value);
    setValidationError(error);
  };

  // Check if there are unsaved changes
  const hasChanges = () => {
    if (pendingAvatar) return true;
    if (!profile) return false;
    return (
      formData.callSign !== (profile.callSign || '') ||
      formData.displayName !== (profile.displayName || '')
    );
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    // Validate before submit - only validate if callsign is not empty
    const trimmedCallSign = formData.callSign.trim();
    if (trimmedCallSign) {
      const callSignError = validateCallSign(trimmedCallSign);
      if (callSignError) {
        setValidationError(callSignError);
        return;
      }
    }

    // Check if user is clearing their callsign (had one before, now removing it)
    if (!trimmedCallSign && profile?.callSign) {
      setShowClearCallSignModal(true);
      return;
    }

    await saveProfile();
  };

  const saveProfile = async () => {
    const trimmedCallSign = formData.callSign.trim();

    try {
      setIsSaving(true);
      setError(null);
      setSuccess(null);
      
      // Upload avatar first if there's a pending one
      if (pendingAvatar) {
        const avatarResult = await uploadAvatar(pendingAvatar.file);
        
        // Use effectiveAvatar if available, otherwise fall back to avatarUrl
        const newAvatarUrl = avatarResult.effectiveAvatar || avatarResult.avatarUrl;
        
        // Update local state
        setProfile(prev => prev ? {
          ...prev,
          customAvatarUrl: avatarResult.avatarUrl,
          avatarType: avatarResult.avatarType || 'custom',
          effectiveAvatarUrl: newAvatarUrl,
        } : null);
        
        // Update auth context so sidebar updates
        if (updateUser) {
          updateUser({
            avatarUrl: newAvatarUrl,
          });
        }
        
        // Revoke the preview URL to free memory
        URL.revokeObjectURL(pendingAvatar.previewUrl);
        setPendingAvatar(null);
      }
      
      // Build update params - include fields that have changed from original
      const params: UpdateProfileParams = {};
      const trimmedDisplayName = formData.displayName.trim();
      
      // Always send callSign if it changed (including clearing it)
      if (trimmedCallSign !== (profile?.callSign || '')) {
        params.callSign = trimmedCallSign;
      }
      // Always send displayName if it changed (including clearing it)
      if (trimmedDisplayName !== (profile?.displayName || '')) {
        params.displayName = trimmedDisplayName;
      }
      
      const updatedProfile = await updateProfile(params);
      setProfile(updatedProfile);
      
      // Update auth context user
      if (updateUser) {
        updateUser({
          displayName: updatedProfile.displayName,
          callSign: updatedProfile.callSign,
        });
      }
      
      setSuccess('Profile updated successfully!');
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update profile');
    } finally {
      setIsSaving(false);
    }
  };

  const handleConfirmClearCallSign = async () => {
    setShowClearCallSignModal(false);
    setConfirmText('');
    await saveProfile();
  };

  const handleCancelClearCallSign = () => {
    setShowClearCallSignModal(false);
    setConfirmText('');
    // Restore the original callsign
    setFormData(prev => ({ ...prev, callSign: profile?.callSign || '' }));
  };

  const handleAvatarSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    // Validate file type
    const allowedTypes = ['image/jpeg', 'image/jpg', 'image/png', 'image/webp'];
    if (!allowedTypes.includes(file.type)) {
      setError('Only JPEG, PNG, and WebP images are allowed');
      return;
    }

    // Validate file size (2MB)
    if (file.size > 2 * 1024 * 1024) {
      setError('Image must be smaller than 2MB');
      return;
    }

    // Revoke old preview URL if exists
    if (pendingAvatar) {
      URL.revokeObjectURL(pendingAvatar.previewUrl);
    }

    // Create preview URL for local display
    const previewUrl = URL.createObjectURL(file);
    setPendingAvatar({ file, previewUrl });
    setError(null);
    
    // Reset file input
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const handleCancelAvatar = () => {
    if (pendingAvatar) {
      URL.revokeObjectURL(pendingAvatar.previewUrl);
      setPendingAvatar(null);
    }
  };

  const handleDeleteAccount = async () => {
    try {
      setIsDeleting(true);
      setError(null);
      await deleteProfile();
      // Account deleted - logout and redirect to home
      // The logout function should clear tokens and redirect
      window.location.href = '/';
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete account');
      setIsDeleting(false);
      setShowDeleteAccountModal(false);
      setDeleteConfirmText('');
    }
  };

  const handleCancelDeleteAccount = () => {
    setShowDeleteAccountModal(false);
    setDeleteConfirmText('');
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary-500"></div>
      </div>
    );
  }

  const effectiveAvatarUrl = pendingAvatar?.previewUrl || profile?.effectiveAvatarUrl || profile?.avatarUrl;

  return (
    <div className="flex-1 overflow-y-auto">
      <div className="max-w-2xl mx-auto p-6 pb-24">
        <h1 className="text-2xl font-bold text-white mb-6">My Profile</h1>

      {error && (
        <div className="mb-4 p-4 bg-red-500/10 border border-red-500/50 rounded-lg text-red-400">
          {error}
        </div>
      )}

      {success && (
        <div className="mb-4 p-4 bg-green-500/10 border border-green-500/50 rounded-lg text-green-400">
          {success}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-6">
        {/* Avatar Section */}
        <div className="bg-slate-800 rounded-lg p-6">
          <h2 className="text-lg font-semibold text-white mb-4">Profile Picture</h2>
          
          <div className="flex items-start gap-6">
            {/* Current Avatar */}
            <div className="flex-shrink-0 relative">
              {effectiveAvatarUrl ? (
                <img
                  src={effectiveAvatarUrl}
                  alt="Profile"
                  className={`w-24 h-24 rounded-full object-cover border-2 ${pendingAvatar ? 'border-primary-500' : 'border-slate-600'}`}
                />
              ) : (
                <div className="w-24 h-24 rounded-full bg-slate-700 flex items-center justify-center border-2 border-slate-600">
                  <svg className="w-12 h-12 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                  </svg>
                </div>
              )}
              {pendingAvatar && (
                <span className="absolute -top-1 -right-1 px-1.5 py-0.5 bg-primary-500 text-white text-xs rounded-full">
                  New
                </span>
              )}
            </div>

            {/* Upload Button */}
            <div className="flex-1 space-y-2">
              <input
                ref={fileInputRef}
                type="file"
                accept="image/jpeg,image/jpg,image/png,image/webp"
                onChange={handleAvatarSelect}
                className="hidden"
              />
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => fileInputRef.current?.click()}
                  className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-lg text-sm transition-colors"
                >
                  {pendingAvatar ? 'Choose Different' : 'Change Avatar'}
                </button>
                {pendingAvatar && (
                  <button
                    type="button"
                    onClick={handleCancelAvatar}
                    className="px-3 py-2 text-slate-400 hover:text-white text-sm transition-colors"
                  >
                    Cancel
                  </button>
                )}
              </div>
              <p className="text-xs text-slate-500">
                JPEG, PNG, or WebP. Max 2MB.
                {pendingAvatar && <span className="text-primary-400 ml-1">Click "Save Changes" to apply.</span>}
              </p>
            </div>
          </div>
        </div>

        {/* Profile Info Section */}
        <div className="bg-slate-800 rounded-lg p-6 space-y-4">
          <h2 className="text-lg font-semibold text-white mb-4">Profile Information</h2>

          {/* CallSign */}
          <div>
            <label htmlFor="callSign" className="block text-sm font-medium text-slate-400 mb-1">
              Callsign
            </label>
            <input
              type="text"
              id="callSign"
              value={formData.callSign}
              onChange={(e) => handleCallSignChange(e.target.value)}
              placeholder="e.g., FPV_Pilot_123"
              className={`w-full px-4 py-2 bg-slate-700 border ${
                validationError ? 'border-red-500' : 'border-slate-600'
              } rounded-lg text-white focus:outline-none focus:border-primary-500`}
            />
            {validationError && (
              <p className="mt-1 text-sm text-red-400">{validationError}</p>
            )}
            <p className="mt-1 text-xs text-slate-500">
              3-20 characters. Letters, numbers, underscores, and hyphens only. This is how other pilots will find you.
            </p>
          </div>

          {/* Display Name */}
          <div>
            <label htmlFor="displayName" className="block text-sm font-medium text-slate-400 mb-1">
              Display Name
            </label>
            <input
              type="text"
              id="displayName"
              value={formData.displayName}
              onChange={(e) => setFormData(prev => ({ ...prev, displayName: e.target.value }))}
              placeholder="Your display name"
              className="w-full px-4 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white focus:outline-none focus:border-primary-500"
            />
            <p className="mt-1 text-xs text-slate-500">
              Optional. If set, will be visible and searchable on the social section.
            </p>
          </div>

          {/* Email (read-only) */}
          <div>
            <label className="block text-sm font-medium text-slate-400 mb-1">
              Email
            </label>
            <div className="px-4 py-2 bg-slate-900 border border-slate-700 rounded-lg text-slate-400">
              {profile?.email}
            </div>
            <p className="mt-1 text-xs text-slate-500">
              Email cannot be changed.
            </p>
          </div>
        </div>

        {/* Submit Button */}
        <div className="flex justify-end gap-3">
          {hasChanges() && (
            <span className="self-center text-sm text-slate-400">
              You have unsaved changes
            </span>
          )}
          <button
            type="submit"
            disabled={isSaving || !!validationError || !hasChanges()}
            className="px-6 py-2 bg-primary-600 hover:bg-primary-500 text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isSaving ? 'Saving...' : 'Save Changes'}
          </button>
        </div>
      </form>

      {/* Account Info */}
      <div className="mt-8 p-4 bg-slate-800/50 rounded-lg">
        <h3 className="text-sm font-medium text-slate-400 mb-2">Account Information</h3>
        <div className="text-xs text-slate-500 space-y-1">
          <p>Member since: {profile?.createdAt ? new Date(profile.createdAt).toLocaleDateString() : 'N/A'}</p>
          <p>Last updated: {profile?.updatedAt ? new Date(profile.updatedAt).toLocaleDateString() : 'N/A'}</p>
        </div>
      </div>

      {/* Danger Zone */}
      <div className="mt-8 p-4 bg-red-900/20 border border-red-500/30 rounded-lg">
        <h3 className="text-sm font-semibold text-red-400 mb-2">Danger Zone</h3>
        <p className="text-xs text-slate-400 mb-4">
          Once you delete your account, there is no going back. All your data will be permanently removed.
        </p>
        <button
          type="button"
          onClick={() => setShowDeleteAccountModal(true)}
          className="px-4 py-2 bg-red-600/20 hover:bg-red-600/30 text-red-400 border border-red-500/50 rounded-lg text-sm font-medium transition-colors"
        >
          Delete Account
        </button>
      </div>

      {/* Clear Call Sign Confirmation Modal */}
      {showClearCallSignModal && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
          <div className="bg-slate-800 rounded-xl p-6 max-w-md w-full shadow-2xl border border-slate-700">
            <div className="flex items-center gap-3 mb-4">
              <div className="w-10 h-10 bg-red-500/20 rounded-full flex items-center justify-center">
                <svg className="w-5 h-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-white">Remove Call Sign?</h3>
            </div>
            
            <div className="mb-4">
              <p className="text-slate-300 mb-3">
                Removing your call sign will:
              </p>
              <ul className="text-sm text-slate-400 space-y-2 mb-4">
                <li className="flex items-start gap-2">
                  <span className="text-red-400 mt-0.5">•</span>
                  <span>Remove all pilots you are following</span>
                </li>
                <li className="flex items-start gap-2">
                  <span className="text-red-400 mt-0.5">•</span>
                  <span>Remove all pilots following you</span>
                </li>
                <li className="flex items-start gap-2">
                  <span className="text-red-400 mt-0.5">•</span>
                  <span>Hide your profile from the pilot directory</span>
                </li>
              </ul>
              <p className="text-sm text-slate-400">
                Type <span className="font-mono text-red-400 bg-red-400/10 px-1.5 py-0.5 rounded">confirm</span> to proceed:
              </p>
            </div>
            
            <input
              type="text"
              value={confirmText}
              onChange={(e) => setConfirmText(e.target.value)}
              placeholder="Type 'confirm' to proceed"
              className="w-full px-4 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-red-500 focus:border-transparent mb-4"
              autoFocus
            />
            
            <div className="flex gap-3">
              <button
                onClick={handleCancelClearCallSign}
                className="flex-1 px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-lg font-medium transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleConfirmClearCallSign}
                disabled={confirmText.toLowerCase() !== 'confirm'}
                className="flex-1 px-4 py-2 bg-red-600 hover:bg-red-500 text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Remove Call Sign
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Account Confirmation Modal */}
      {showDeleteAccountModal && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
          <div className="bg-slate-800 rounded-xl p-6 max-w-md w-full shadow-2xl border border-red-500/50">
            <div className="flex items-center gap-3 mb-4">
              <div className="w-10 h-10 bg-red-500/20 rounded-full flex items-center justify-center">
                <svg className="w-5 h-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-white">Delete Account?</h3>
            </div>
            
            <div className="mb-4">
              <p className="text-slate-300 mb-3">
                <strong className="text-red-400">This action cannot be undone.</strong> Deleting your account will permanently remove:
              </p>
              <ul className="text-sm text-slate-400 space-y-2 mb-4">
                <li className="flex items-start gap-2">
                  <span className="text-red-400 mt-0.5">•</span>
                  <span>Your profile and account information</span>
                </li>
                <li className="flex items-start gap-2">
                  <span className="text-red-400 mt-0.5">•</span>
                  <span>All your aircraft and configurations</span>
                </li>
                <li className="flex items-start gap-2">
                  <span className="text-red-400 mt-0.5">•</span>
                  <span>Your inventory items</span>
                </li>
                <li className="flex items-start gap-2">
                  <span className="text-red-400 mt-0.5">•</span>
                  <span>Battery logs and radio backups</span>
                </li>
                <li className="flex items-start gap-2">
                  <span className="text-red-400 mt-0.5">•</span>
                  <span>All followers and following relationships</span>
                </li>
              </ul>
              <p className="text-sm text-slate-400">
                Type <span className="font-mono text-red-400 bg-red-400/10 px-1.5 py-0.5 rounded">delete my account</span> to confirm:
              </p>
            </div>
            
            <input
              type="text"
              value={deleteConfirmText}
              onChange={(e) => setDeleteConfirmText(e.target.value)}
              placeholder="Type 'delete my account' to confirm"
              className="w-full px-4 py-2 bg-slate-700 border border-red-500/50 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-red-500 focus:border-transparent mb-4"
              autoFocus
              disabled={isDeleting}
            />
            
            <div className="flex gap-3">
              <button
                onClick={handleCancelDeleteAccount}
                disabled={isDeleting}
                className="flex-1 px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-lg font-medium transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                onClick={handleDeleteAccount}
                disabled={deleteConfirmText.toLowerCase() !== 'delete my account' || isDeleting}
                className="flex-1 px-4 py-2 bg-red-600 hover:bg-red-500 text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isDeleting ? 'Deleting...' : 'Delete Account'}
              </button>
            </div>
          </div>
        </div>
      )}
      </div>
    </div>
  );
}

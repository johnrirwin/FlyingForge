import { useState, useEffect, useRef } from 'react';
import { useAuth } from '../hooks/useAuth';
import { getProfile, updateProfile, uploadAvatar, validateCallSign } from '../profileApi';
import type { UserProfile, UpdateProfileParams, AvatarType } from '../authTypes';

interface ProfileFormData {
  callSign: string;
  displayName: string;
  avatarType: AvatarType;
}

export function MyProfile() {
  const { updateUser } = useAuth();
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [formData, setFormData] = useState<ProfileFormData>({
    callSign: '',
    displayName: '',
    avatarType: 'google',
  });
  const [validationError, setValidationError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    loadProfile();
  }, []);

  const loadProfile = async () => {
    try {
      setIsLoading(true);
      setError(null);
      const data = await getProfile();
      setProfile(data);
      setFormData({
        callSign: data.callSign || '',
        displayName: data.displayName || '',
        avatarType: data.avatarType || 'google',
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

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    // Validate before submit
    const callSignError = validateCallSign(formData.callSign);
    if (callSignError) {
      setValidationError(callSignError);
      return;
    }

    try {
      setIsSaving(true);
      setError(null);
      setSuccess(null);
      
      const params: UpdateProfileParams = {
        callSign: formData.callSign.trim() || undefined,
        displayName: formData.displayName.trim() || undefined,
        avatarType: formData.avatarType,
      };
      
      const updatedProfile = await updateProfile(params);
      setProfile(updatedProfile);
      
      // Update auth context user
      if (updateUser) {
        updateUser({
          displayName: updatedProfile.displayName,
          callSign: updatedProfile.callSign,
          avatarType: updatedProfile.avatarType,
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

  const handleAvatarUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
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

    try {
      setIsUploading(true);
      setError(null);
      
      const result = await uploadAvatar(file);
      
      // Update local state
      setProfile(prev => prev ? {
        ...prev,
        customAvatarUrl: result.customAvatarUrl,
        avatarType: result.avatarType,
        effectiveAvatarUrl: result.effectiveAvatarUrl,
      } : null);
      
      setFormData(prev => ({ ...prev, avatarType: 'custom' }));
      
      setSuccess('Avatar uploaded successfully!');
      setTimeout(() => setSuccess(null), 3000);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to upload avatar');
    } finally {
      setIsUploading(false);
      // Reset file input
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary-500"></div>
      </div>
    );
  }

  const effectiveAvatarUrl = profile?.effectiveAvatarUrl;
  const googleAvatarUrl = profile?.googleAvatarUrl;
  const customAvatarUrl = profile?.customAvatarUrl;

  return (
    <div className="max-w-2xl mx-auto p-6">
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
            <div className="flex-shrink-0">
              {effectiveAvatarUrl ? (
                <img
                  src={effectiveAvatarUrl}
                  alt="Profile"
                  className="w-24 h-24 rounded-full object-cover border-2 border-slate-600"
                />
              ) : (
                <div className="w-24 h-24 rounded-full bg-slate-700 flex items-center justify-center border-2 border-slate-600">
                  <svg className="w-12 h-12 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                  </svg>
                </div>
              )}
            </div>

            {/* Avatar Options */}
            <div className="flex-1 space-y-4">
              {/* Avatar Type Toggle */}
              <div className="space-y-2">
                <label className="block text-sm font-medium text-slate-400">Avatar Source</label>
                <div className="flex gap-4">
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="radio"
                      name="avatarType"
                      value="google"
                      checked={formData.avatarType === 'google'}
                      onChange={() => setFormData(prev => ({ ...prev, avatarType: 'google' }))}
                      className="text-primary-500 focus:ring-primary-500"
                      disabled={!googleAvatarUrl}
                    />
                    <span className={`text-sm ${!googleAvatarUrl ? 'text-slate-600' : 'text-slate-300'}`}>
                      Google Avatar
                    </span>
                  </label>
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="radio"
                      name="avatarType"
                      value="custom"
                      checked={formData.avatarType === 'custom'}
                      onChange={() => setFormData(prev => ({ ...prev, avatarType: 'custom' }))}
                      className="text-primary-500 focus:ring-primary-500"
                      disabled={!customAvatarUrl}
                    />
                    <span className={`text-sm ${!customAvatarUrl ? 'text-slate-600' : 'text-slate-300'}`}>
                      Custom Avatar
                    </span>
                  </label>
                </div>
              </div>

              {/* Upload Button */}
              <div>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="image/jpeg,image/jpg,image/png,image/webp"
                  onChange={handleAvatarUpload}
                  className="hidden"
                />
                <button
                  type="button"
                  onClick={() => fileInputRef.current?.click()}
                  disabled={isUploading}
                  className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-lg text-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {isUploading ? 'Uploading...' : 'Upload Custom Avatar'}
                </button>
                <p className="mt-1 text-xs text-slate-500">
                  JPEG, PNG, or WebP. Max 2MB.
                </p>
              </div>
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
              Optional. Shown alongside your callsign.
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
        <div className="flex justify-end">
          <button
            type="submit"
            disabled={isSaving || !!validationError}
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
    </div>
  );
}

import { useState, useEffect, useCallback } from 'react';
import { getPilotProfile } from '../pilotApi';
import { followPilot, unfollowPilot, getPublicAircraftImageUrl } from '../socialApi';
import type { PilotProfile as PilotProfileType, AircraftPublic } from '../socialTypes';
import { useAuth } from '../hooks/useAuth';

interface PilotProfileProps {
  pilotId: string;
  onBack: () => void;
}

export function PilotProfile({ pilotId, onBack }: PilotProfileProps) {
  const { user } = useAuth();
  const [profile, setProfile] = useState<PilotProfileType | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isFollowing, setIsFollowing] = useState(false);
  const [isFollowLoading, setIsFollowLoading] = useState(false);

  const isOwnProfile = user?.id === pilotId;

  const loadProfile = useCallback(async () => {
    try {
      setIsLoading(true);
      setError(null);
      const data = await getPilotProfile(pilotId);
      setProfile(data);
      setIsFollowing(data.isFollowing);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load pilot profile');
    } finally {
      setIsLoading(false);
    }
  }, [pilotId]);

  useEffect(() => {
    loadProfile();
  }, [loadProfile]);

  const handleFollowToggle = async () => {
    if (isFollowLoading || isOwnProfile) return;
    
    try {
      setIsFollowLoading(true);
      if (isFollowing) {
        await unfollowPilot(pilotId);
        setIsFollowing(false);
        if (profile) {
          setProfile({ ...profile, followerCount: profile.followerCount - 1 });
        }
      } else {
        await followPilot(pilotId);
        setIsFollowing(true);
        if (profile) {
          setProfile({ ...profile, followerCount: profile.followerCount + 1 });
        }
      }
    } catch (err) {
      console.error('Failed to toggle follow:', err);
    } finally {
      setIsFollowLoading(false);
    }
  };

  const getDisplayName = () => {
    if (!profile) return 'Unknown Pilot';
    if (profile.callSign) return profile.callSign;
    if (profile.displayName) return profile.displayName;
    if (profile.googleName) return profile.googleName;
    return 'Unknown Pilot';
  };

  const getSecondaryName = () => {
    if (!profile) return null;
    if (profile.callSign && profile.displayName) return profile.displayName;
    if (profile.callSign && profile.googleName) return profile.googleName;
    return null;
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary-500"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <button
          onClick={onBack}
          className="flex items-center gap-2 text-slate-400 hover:text-white mb-6 transition-colors"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
          Back to Search
        </button>
        <div className="p-4 bg-red-500/10 border border-red-500/50 rounded-lg text-red-400 max-w-2xl">
          {error}
        </div>
      </div>
    );
  }

  if (!profile) {
    return (
      <div className="p-6">
        <button
          onClick={onBack}
          className="flex items-center gap-2 text-slate-400 hover:text-white mb-6 transition-colors"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
          </svg>
          Back to Search
        </button>
        <div className="text-slate-500">Pilot not found</div>
      </div>
    );
  }

  return (
    <div className="p-6">
      {/* Back Button */}
      <button
        onClick={onBack}
        className="flex items-center gap-2 text-slate-400 hover:text-white mb-6 transition-colors"
      >
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
        </svg>
        Back to Search
      </button>

      {/* Profile Header */}
      <div className="bg-slate-800 rounded-lg p-6 mb-6 max-w-2xl">
        <div className="flex items-center gap-6">
          {/* Avatar */}
          {profile.effectiveAvatarUrl ? (
            <img
              src={profile.effectiveAvatarUrl}
              alt=""
              className="w-24 h-24 rounded-full object-cover border-2 border-slate-600"
            />
          ) : (
            <div className="w-24 h-24 rounded-full bg-slate-700 flex items-center justify-center border-2 border-slate-600">
              <svg className="w-12 h-12 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
              </svg>
            </div>
          )}

          {/* Info */}
          <div className="flex-1">
            <h1 className="text-2xl font-bold text-white">{getDisplayName()}</h1>
            {getSecondaryName() && (
              <p className="text-slate-400">{getSecondaryName()}</p>
            )}
            <div className="flex items-center gap-4 mt-2 text-sm text-slate-500">
              <span>{profile.followerCount} followers</span>
              <span>{profile.followingCount} following</span>
            </div>
            <p className="text-sm text-slate-500 mt-1">
              Member since {new Date(profile.createdAt).toLocaleDateString()}
            </p>
          </div>

          {/* Follow Button */}
          {!isOwnProfile && (
            <button
              onClick={handleFollowToggle}
              disabled={isFollowLoading}
              className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                isFollowing
                  ? 'bg-slate-700 text-white hover:bg-slate-600'
                  : 'bg-primary-500 text-white hover:bg-primary-600'
              } disabled:opacity-50`}
            >
              {isFollowLoading ? '...' : isFollowing ? 'Following' : 'Follow'}
            </button>
          )}
        </div>
      </div>

      {/* Aircraft Section */}
      <div className="bg-slate-800 rounded-lg p-6 max-w-2xl">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-white">Aircraft</h2>
          <span className="text-sm text-slate-400">
            {profile.aircraft.length} aircraft
          </span>
        </div>

        {profile.aircraft.length === 0 ? (
          <div className="text-center py-8 text-slate-500">
            <svg className="w-12 h-12 mx-auto mb-3 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
            </svg>
            <p>No aircraft to display</p>
          </div>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2">
            {profile.aircraft.map((aircraft) => (
              <AircraftCard key={aircraft.id} aircraft={aircraft} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function AircraftCard({ aircraft }: { aircraft: AircraftPublic }) {
  const [isExpanded, setIsExpanded] = useState(false);
  const [imageError, setImageError] = useState(false);
  
  const formatType = (type?: string) => {
    if (!type) return 'Unknown Type';
    return type.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
  };

  const formatCategory = (category: string) => {
    const categoryLabels: Record<string, string> = {
      fc: 'Flight Controller',
      esc: 'ESC',
      elrs_module: 'ELRS RX',
      vtx: 'VTX',
      motors: 'Motors',
      camera: 'Camera',
      frame: 'Frame',
      propellers: 'Props',
      antenna: 'Antenna',
    };
    return categoryLabels[category] || category;
  };

  const hasElrsSettings = aircraft.elrsSettings && Object.values(aircraft.elrsSettings).some(v => v);
  const hasComponents = aircraft.components && aircraft.components.length > 0;
  const hasDetails = hasComponents || hasElrsSettings;

  return (
    <div 
      className={`bg-slate-900 rounded-lg overflow-hidden border border-slate-700 transition-all ${hasDetails ? 'cursor-pointer hover:border-slate-600' : ''}`}
      onClick={() => hasDetails && setIsExpanded(!isExpanded)}
    >
      {/* Image */}
      <div className="aspect-video bg-slate-800 flex items-center justify-center relative">
        {aircraft.hasImage && !imageError ? (
          <img
            src={getPublicAircraftImageUrl(aircraft.id)}
            alt={aircraft.name}
            className="w-full h-full object-cover"
            onError={() => setImageError(true)}
          />
        ) : (
          <svg className="w-12 h-12 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
          </svg>
        )}
        {hasDetails && (
          <div className="absolute bottom-2 right-2 bg-slate-900/80 px-2 py-1 rounded text-[10px] text-slate-400">
            {isExpanded ? 'Click to collapse' : 'Click for details'}
          </div>
        )}
      </div>

      {/* Info */}
      <div className="p-4">
        <h3 className="font-medium text-white truncate">{aircraft.name}</h3>
        {aircraft.nickname && (
          <p className="text-sm text-slate-400 truncate mt-0.5">"{aircraft.nickname}"</p>
        )}
        {aircraft.type && (
          <div className="mt-2">
            <span className="px-2 py-0.5 bg-slate-700 rounded text-xs text-slate-300">
              {formatType(aircraft.type)}
            </span>
          </div>
        )}
        {aircraft.description && (
          <p className="text-sm text-slate-500 mt-2 line-clamp-2">{aircraft.description}</p>
        )}

        {/* Expandable Details */}
        {isExpanded && (
          <>
            {/* Components */}
            {hasComponents && (
              <div className="mt-3 pt-3 border-t border-slate-700">
                <div className="flex items-center gap-1.5 mb-2">
                  <span className="text-xs font-medium text-slate-400">Components</span>
                </div>
                <div className="space-y-1.5">
                  {aircraft.components?.map((component, idx) => (
                    <div key={idx} className="flex items-start gap-2 text-xs">
                      <span className="text-slate-500 w-20 shrink-0">{formatCategory(component.category)}</span>
                      <span className="text-slate-300">
                        {component.manufacturer && <span className="text-slate-500">{component.manufacturer} </span>}
                        {component.name || 'Unspecified'}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* ELRS Settings (Sanitized/Safe View) */}
            {hasElrsSettings && (
              <div className="mt-3 pt-3 border-t border-slate-700">
                <div className="flex items-center gap-1.5 mb-2">
                  <span className="text-xs font-medium text-slate-400">ELRS Config</span>
                  <span className="px-1.5 py-0.5 bg-green-500/20 text-green-400 text-[10px] rounded">
                    Safe View
                  </span>
                </div>
                <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
                  {aircraft.elrsSettings?.receiverModel && (
                    <div className="text-slate-500">
                      <span className="text-slate-400">RX:</span> {aircraft.elrsSettings.receiverModel}
                    </div>
                  )}
                  {aircraft.elrsSettings?.packetRate && (
                    <div className="text-slate-500">
                      <span className="text-slate-400">Rate:</span> {aircraft.elrsSettings.packetRate}
                    </div>
                  )}
                  {aircraft.elrsSettings?.outputPower && (
                    <div className="text-slate-500">
                      <span className="text-slate-400">Power:</span> {aircraft.elrsSettings.outputPower}
                    </div>
                  )}
                  {aircraft.elrsSettings?.switchMode && (
                    <div className="text-slate-500">
                      <span className="text-slate-400">Switch:</span> {aircraft.elrsSettings.switchMode}
                    </div>
                  )}
                  {aircraft.elrsSettings?.telemetryRatio && (
                    <div className="text-slate-500">
                      <span className="text-slate-400">Telem:</span> {aircraft.elrsSettings.telemetryRatio}
                    </div>
                  )}
                  {aircraft.elrsSettings?.firmwareVersion && (
                    <div className="text-slate-500">
                      <span className="text-slate-400">FW:</span> {aircraft.elrsSettings.firmwareVersion}
                    </div>
                  )}
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}

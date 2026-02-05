import { useState, useEffect, useCallback } from 'react';
import { getPilotProfile } from '../pilotApi';
import { followPilot, unfollowPilot, ApiError } from '../socialApi';
import { updateProfile } from '../profileApi';
import type { PilotProfile as PilotProfileType, AircraftPublic } from '../socialTypes';
import { useAuth } from '../hooks/useAuth';
import { PublicAircraftModal } from './PublicAircraftModal';
import { FollowListModal } from './FollowListModal';
import { CallSignPromptModal } from './SocialPage';
import { AircraftImage } from './AircraftImage';
import { trackEvent } from '../hooks/useGoogleAnalytics';

type FollowListType = 'followers' | 'following' | null;

interface PilotProfileProps {
  pilotId: string;
  onBack: () => void;
  onSelectPilot?: (pilotId: string) => void;
}

export function PilotProfile({ pilotId, onBack, onSelectPilot }: PilotProfileProps) {
  const { user, updateUser } = useAuth();
  const [profile, setProfile] = useState<PilotProfileType | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isFollowing, setIsFollowing] = useState(false);
  const [isFollowLoading, setIsFollowLoading] = useState(false);
  const [selectedAircraft, setSelectedAircraft] = useState<AircraftPublic | null>(null);
  const [showFollowList, setShowFollowList] = useState<FollowListType>(null);
  const [showCallSignPrompt, setShowCallSignPrompt] = useState(false);

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
        // Track unfollow action
        trackEvent('social_unfollow');
      } else {
        await followPilot(pilotId);
        setIsFollowing(true);
        if (profile) {
          setProfile({ ...profile, followerCount: profile.followerCount + 1 });
        }
        // Track follow action
        trackEvent('social_follow');
      }
    } catch (err) {
      // Check if this is a callsign required error - show the modal instead
      // Use both instanceof and property check for robustness with bundlers
      const isCallSignRequired = 
        (err instanceof ApiError && err.code === 'callsign_required') ||
        (err && typeof err === 'object' && 'code' in err && (err as { code: string }).code === 'callsign_required');
      
      if (isCallSignRequired) {
        setShowCallSignPrompt(true);
      } else {
        // Log other errors but don't show inline - they're usually transient
        console.error('Failed to toggle follow:', err);
      }
    } finally {
      setIsFollowLoading(false);
    }
  };

  // Handle saving callsign and then following
  const handleSaveCallSignAndFollow = async (callSign: string) => {
    // First save the callsign - let errors propagate to the modal's error handling
    await updateProfile({ callSign });
    // Update local state after backend succeeded
    updateUser({ callSign });
    
    // Now try to follow - keep modal open until this succeeds
    try {
      setIsFollowLoading(true);
      await followPilot(pilotId);
      setIsFollowing(true);
      if (profile) {
        setProfile({ ...profile, followerCount: profile.followerCount + 1 });
      }
      trackEvent('social_follow');
      // Only close modal after entire flow succeeds
      setShowCallSignPrompt(false);
    } catch (err) {
      // Re-throw with clearer message since callsign was saved but follow failed
      const message = err instanceof Error ? err.message : 'Failed to follow';
      throw new Error(`Call sign saved! But follow failed: ${message}. Click "Continue" to try again.`);
    } finally {
      setIsFollowLoading(false);
    }
  };

  const getDisplayName = () => {
    if (!profile) return 'Unknown Pilot';
    return profile.callSign || 'Unknown Pilot';
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
        <div className="flex flex-col sm:flex-row sm:items-center gap-4 sm:gap-6">
          {/* Avatar */}
          <div className="flex items-center gap-4 sm:block">
            {profile.effectiveAvatarUrl ? (
              <img
                src={profile.effectiveAvatarUrl}
                alt=""
                className="w-20 h-20 sm:w-24 sm:h-24 rounded-full object-cover border-2 border-slate-600 flex-shrink-0"
              />
            ) : (
              <div className="w-20 h-20 sm:w-24 sm:h-24 rounded-full bg-slate-700 flex items-center justify-center border-2 border-slate-600 flex-shrink-0">
                <svg className="w-10 h-10 sm:w-12 sm:h-12 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                </svg>
              </div>
            )}
            {/* Mobile: Name next to avatar */}
            <div className="sm:hidden flex-1 min-w-0">
              <div className="flex items-center gap-2 flex-wrap">
                <h1 className="text-xl font-bold text-white truncate">{getDisplayName()}</h1>
                {isOwnProfile && (
                  <span className="px-2 py-0.5 text-xs font-medium bg-slate-700 text-slate-300 rounded whitespace-nowrap flex-shrink-0">
                    You
                  </span>
                )}
              </div>
              {profile.displayName && (
                <p className="text-slate-400 text-sm mt-0.5">{profile.displayName}</p>
              )}
            </div>
          </div>

          {/* Info - Desktop */}
          <div className="hidden sm:block flex-1 min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <h1 className="text-2xl font-bold text-white truncate">{getDisplayName()}</h1>
              {isOwnProfile && (
                <span className="px-2 py-0.5 text-xs font-medium bg-slate-700 text-slate-300 rounded whitespace-nowrap flex-shrink-0">
                  You
                </span>
              )}
            </div>
            {profile.displayName && (
              <p className="text-slate-400 mt-1">{profile.displayName}</p>
            )}
            <div className="flex items-center gap-4 mt-2 text-sm">
              <button 
                onClick={() => setShowFollowList('followers')}
                className="text-slate-400 hover:text-white transition-colors"
              >
                <span className="font-medium text-white">{profile.followerCount}</span> followers
              </button>
              <button 
                onClick={() => setShowFollowList('following')}
                className="text-slate-400 hover:text-white transition-colors"
              >
                <span className="font-medium text-white">{profile.followingCount}</span> following
              </button>
            </div>
            <p className="text-sm text-slate-500 mt-1">
              Member since {new Date(profile.createdAt).toLocaleDateString()}
            </p>
          </div>

          {/* Follow Button - Desktop */}
          {!isOwnProfile && (
            <button
              onClick={handleFollowToggle}
              disabled={isFollowLoading}
              className={`hidden sm:block px-4 py-2 rounded-lg font-medium transition-colors flex-shrink-0 ${
                isFollowing
                  ? 'bg-slate-700 text-white hover:bg-slate-600'
                  : 'bg-primary-500 text-white hover:bg-primary-600'
              } disabled:opacity-50`}
            >
              {isFollowLoading ? '...' : isFollowing ? 'Following' : 'Follow'}
            </button>
          )}
        </div>

        {/* Mobile: Stats and Follow Button Row */}
        <div className="sm:hidden mt-4 flex items-center justify-between gap-4">
          <div className="flex items-center gap-4 text-sm">
            <button 
              onClick={() => setShowFollowList('followers')}
              className="text-slate-400 hover:text-white transition-colors"
            >
              <span className="font-medium text-white">{profile.followerCount}</span> followers
            </button>
            <button 
              onClick={() => setShowFollowList('following')}
              className="text-slate-400 hover:text-white transition-colors"
            >
              <span className="font-medium text-white">{profile.followingCount}</span> following
            </button>
          </div>
          {!isOwnProfile && (
            <button
              onClick={handleFollowToggle}
              disabled={isFollowLoading}
              className={`px-4 py-2 rounded-lg font-medium transition-colors flex-shrink-0 ${
                isFollowing
                  ? 'bg-slate-700 text-white hover:bg-slate-600'
                  : 'bg-primary-500 text-white hover:bg-primary-600'
              } disabled:opacity-50`}
            >
              {isFollowLoading ? '...' : isFollowing ? 'Following' : 'Follow'}
            </button>
          )}
        </div>

        {/* Mobile: Member since */}
        <p className="sm:hidden text-sm text-slate-500 mt-2">
          Member since {new Date(profile.createdAt).toLocaleDateString()}
        </p>
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
              <AircraftCard 
                key={aircraft.id} 
                aircraft={aircraft} 
                onClick={() => setSelectedAircraft(aircraft)}
              />
            ))}
          </div>
        )}
      </div>

      {/* Aircraft Detail Modal */}
      {selectedAircraft && (
        <PublicAircraftModal 
          aircraft={selectedAircraft} 
          onClose={() => setSelectedAircraft(null)} 
        />
      )}

      {/* Follow List Modal */}
      {showFollowList && profile && (
        <FollowListModal
          userId={pilotId}
          userName={getDisplayName()}
          type={showFollowList}
          onClose={() => setShowFollowList(null)}
          onSelectPilot={(newPilotId) => {
            setShowFollowList(null);
            if (newPilotId !== pilotId && onSelectPilot) {
              onSelectPilot(newPilotId);
            }
          }}
        />
      )}

      {/* Call Sign Prompt Modal */}
      {showCallSignPrompt && (
        <CallSignPromptModal
          onClose={() => setShowCallSignPrompt(false)}
          onSave={handleSaveCallSignAndFollow}
          title="Set Your Call Sign"
          subtitle="Required to follow other pilots"
          description="To follow other pilots in the community, you need to set up your call sign first. This helps build a trusted community where pilots can connect with each other."
          initialCallSign={user?.callSign || ''}
        />
      )}
    </div>
  );
}

function AircraftCard({ aircraft, onClick }: { aircraft: AircraftPublic; onClick: () => void }) {
  const formatType = (type?: string) => {
    if (!type) return 'Unknown Type';
    return type.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
  };

  const hasReceiverSettings = aircraft.receiverSettings && Object.values(aircraft.receiverSettings).some(v => v);
  const componentCount = aircraft.components?.length || 0;

  return (
    <div 
      className="bg-slate-900 rounded-lg overflow-hidden border border-slate-700 transition-all cursor-pointer hover:border-slate-500 hover:shadow-lg"
      onClick={onClick}
    >
      {/* Image */}
      <div className="aspect-video bg-slate-800 flex items-center justify-center relative">
        <AircraftImage
          aircraftId={aircraft.id}
          aircraftName={aircraft.name}
          hasImage={aircraft.hasImage}
          className="w-full h-full object-cover"
          fallbackIcon={
            <svg className="w-12 h-12 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
            </svg>
          }
        />
        {/* Click hint overlay */}
        <div className="absolute inset-0 bg-black/0 hover:bg-black/20 transition-colors flex items-center justify-center opacity-0 hover:opacity-100">
          <span className="bg-slate-900/90 px-3 py-1.5 rounded-lg text-sm text-white">
            View Details
          </span>
        </div>
      </div>

      {/* Info */}
      <div className="p-4">
        <h3 className="font-medium text-white truncate">{aircraft.name}</h3>
        {aircraft.nickname && (
          <p className="text-sm text-slate-400 truncate mt-0.5">"{aircraft.nickname}"</p>
        )}
        <div className="flex items-center gap-2 mt-2 flex-wrap">
          {aircraft.type && (
            <span className="px-2 py-0.5 bg-slate-700 rounded text-xs text-slate-300">
              {formatType(aircraft.type)}
            </span>
          )}
          {componentCount > 0 && (
            <span className="px-2 py-0.5 bg-slate-700 rounded text-xs text-slate-400">
              {componentCount} component{componentCount !== 1 ? 's' : ''}
            </span>
          )}
          {hasReceiverSettings && (
            <span className="px-2 py-0.5 bg-green-500/20 text-green-400 rounded text-xs">
              Receiver
            </span>
          )}
        </div>
        {aircraft.description && (
          <p className="text-sm text-slate-500 mt-2 line-clamp-2">{aircraft.description}</p>
        )}
      </div>
    </div>
  );
}
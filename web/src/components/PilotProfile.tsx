import { useState, useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { getPilotProfile } from '../pilotApi';
import { getPublicBuild } from '../buildApi';
import { followPilot, unfollowPilot, ApiError } from '../socialApi';
import { updateProfile } from '../profileApi';
import type { PilotProfile as PilotProfileType, AircraftPublic } from '../socialTypes';
import type { GearCatalogItem } from '../gearCatalogTypes';
import type { Build } from '../buildTypes';
import { findPart, getBuildPartDisplayName } from '../buildTypes';
import { useAuth } from '../hooks/useAuth';
import { PublicAircraftModal } from './PublicAircraftModal';
import { FollowListModal } from './FollowListModal';
import { CallSignPromptModal } from './SocialPage';
import { AircraftImage } from './AircraftImage';
import { trackEvent } from '../hooks/useGoogleAnalytics';
import { getYouTubeEmbedURL } from '../youtube';

type FollowListType = 'followers' | 'following' | null;

interface PilotProfileProps {
  pilotId: string;
  onBack: () => void;
  onSelectPilot?: (pilotId: string) => void;
  isModal?: boolean;
  onAddToInventory?: (item: GearCatalogItem) => void;
}

export function PilotProfile({ pilotId, onBack, onSelectPilot, isModal = false, onAddToInventory }: PilotProfileProps) {
  const { user, updateUser } = useAuth();
  const [profile, setProfile] = useState<PilotProfileType | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [isFollowing, setIsFollowing] = useState(false);
  const [isFollowLoading, setIsFollowLoading] = useState(false);
  const [selectedAircraft, setSelectedAircraft] = useState<AircraftPublic | null>(null);
  const [selectedBuildID, setSelectedBuildID] = useState<string | null>(null);
  const [selectedBuild, setSelectedBuild] = useState<Build | null>(null);
  const [isBuildPreviewLoading, setIsBuildPreviewLoading] = useState(false);
  const [buildPreviewError, setBuildPreviewError] = useState<string | null>(null);
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

  const handleOpenBuildPreview = useCallback(async (buildID: string) => {
    const nextBuildID = buildID.trim();
    if (!nextBuildID) return;

    setSelectedBuildID(nextBuildID);
    setSelectedBuild(null);
    setBuildPreviewError(null);
    setIsBuildPreviewLoading(true);

    try {
      const build = await getPublicBuild(nextBuildID);
      setSelectedBuild(build);
    } catch (err) {
      setBuildPreviewError(err instanceof Error ? err.message : 'Failed to load build');
    } finally {
      setIsBuildPreviewLoading(false);
    }
  }, []);

  const handleCloseBuildPreview = useCallback(() => {
    setSelectedBuildID(null);
    setSelectedBuild(null);
    setBuildPreviewError(null);
    setIsBuildPreviewLoading(false);
  }, []);

  const getDisplayName = () => {
    if (!profile) return 'Unknown Pilot';
    return profile.callSign || 'Unknown Pilot';
  };

  if (isLoading) {
    return (
      <div className={`flex items-center justify-center ${isModal ? 'py-12' : 'h-full'}`}>
        <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary-500"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        {!isModal && (
          <button
            onClick={onBack}
            className="flex items-center gap-2 text-slate-400 hover:text-white mb-6 transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
            </svg>
            Back to Search
          </button>
        )}
        <div className="p-4 bg-red-500/10 border border-red-500/50 rounded-lg text-red-400 max-w-2xl">
          {error}
        </div>
      </div>
    );
  }

  if (!profile) {
    return (
      <div className="p-6">
        {!isModal && (
          <button
            onClick={onBack}
            className="flex items-center gap-2 text-slate-400 hover:text-white mb-6 transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
            </svg>
            Back to Search
          </button>
        )}
        <div className="text-slate-500">Pilot not found</div>
      </div>
    );
  }

  const profileCard = (
    <div className="bg-slate-800 rounded-lg p-6">
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
  );

  const publishedBuilds = profile.publishedBuilds ?? [];

  const aircraftSection = (
    <div className="bg-slate-800 rounded-lg p-6">
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
  );

  const publishedBuildsSection = (
    <div className="bg-slate-800 rounded-lg p-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-white">Published Builds</h2>
        <span className="text-sm text-slate-400">
          {publishedBuilds.length} build{publishedBuilds.length === 1 ? '' : 's'}
        </span>
      </div>

      {publishedBuilds.length === 0 ? (
        <div className="text-center py-8 text-slate-500">
          <svg className="w-12 h-12 mx-auto mb-3 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7h18M7 12h10M9 17h6" />
          </svg>
          <p>No published builds yet</p>
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {publishedBuilds.map((build) => (
            <button
              key={build.id}
              type="button"
              onClick={() => {
                void handleOpenBuildPreview(build.id);
              }}
              className="w-full text-left bg-slate-900 rounded-lg overflow-hidden border border-slate-700 transition-all hover:border-primary-500 hover:shadow-lg"
            >
              <div className="aspect-video bg-slate-800 flex items-center justify-center">
                {build.mainImageUrl ? (
                  <img src={build.mainImageUrl} alt={build.title} className="w-full h-full object-cover" />
                ) : (
                  <span className="text-sm text-slate-500">No build image</span>
                )}
              </div>
              <div className="p-4">
                <h3 className="font-medium text-white line-clamp-2">{build.title}</h3>
                {build.description && (
                  <p className="text-sm text-slate-400 mt-2 line-clamp-2">{build.description}</p>
                )}
                <p className="text-xs text-slate-500 mt-3">
                  Published {new Date(build.publishedAt || build.createdAt).toLocaleDateString()}
                </p>
                <p className="text-xs text-primary-300 mt-1">View build details</p>
              </div>
            </button>
          ))}
        </div>
      )}
    </div>
  );

  return (
    <div className={`${isModal ? 'h-full flex flex-col' : 'flex-1 overflow-y-auto'}`}>
      {isModal ? (
        <div className="px-6 pt-6 pb-6 flex-1 min-h-0">
          <div className="max-w-4xl mx-auto w-full h-full flex flex-col">
            <div>
              {profileCard}
            </div>
            <div className="mt-4 flex-1 min-h-0 overflow-y-auto overscroll-y-contain pr-1 space-y-4">
              {publishedBuildsSection}
              {aircraftSection}
            </div>
          </div>
        </div>
      ) : (
        <div className="p-6 pb-24">
          <button
            onClick={onBack}
            className="flex items-center gap-2 text-slate-400 hover:text-white mb-6 transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 19l-7-7m0 0l7-7m-7 7h18" />
            </svg>
            Back to Search
          </button>
          <div className="max-w-2xl mb-6">
            {profileCard}
          </div>
          <div className="max-w-2xl mb-6">
            {publishedBuildsSection}
          </div>
          <div className="max-w-2xl">
            {aircraftSection}
          </div>
        </div>
      )}

      {/* Aircraft Detail Modal */}
      {selectedAircraft && (
        <PublicAircraftModal 
          aircraft={selectedAircraft} 
          onClose={() => setSelectedAircraft(null)}
          onAddToInventory={onAddToInventory}
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

      {(selectedBuildID || selectedBuild || buildPreviewError || isBuildPreviewLoading) && (
        <PublishedBuildPreviewModal
          build={selectedBuild}
          isLoading={isBuildPreviewLoading}
          error={buildPreviewError}
          onClose={handleCloseBuildPreview}
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

function PublishedBuildPreviewModal({
  build,
  isLoading,
  error,
  onClose,
}: {
  build: Build | null;
  isLoading: boolean;
  error: string | null;
  onClose: () => void;
}) {
  const parts = build?.parts ?? [];
  const frame = findPart(parts, 'frame');
  const motors = findPart(parts, 'motor');
  const receiver = findPart(parts, 'receiver');
  const vtx = findPart(parts, 'vtx');
  const aio = findPart(parts, 'aio');
  const stack = findPart(parts, 'stack');
  const fc = findPart(parts, 'fc');
  const esc = findPart(parts, 'esc');
  const uniqueCatalogParts = new Map<string, Build['parts'][number]>();
  for (const part of parts) {
    const catalogItemID = part.catalogItemId?.trim() || part.catalogItem?.id?.trim();
    if (!catalogItemID) continue;

    const existingPart = uniqueCatalogParts.get(catalogItemID);
    const existingMsrp = existingPart?.catalogItem?.msrp;
    const currentMsrp = part.catalogItem?.msrp;

    if (
      !existingPart
      || (typeof existingMsrp !== 'number' || existingMsrp <= 0)
        && typeof currentMsrp === 'number'
        && currentMsrp > 0
    ) {
      uniqueCatalogParts.set(catalogItemID, part);
    }
  }

  const msrpTotal = Array.from(uniqueCatalogParts.values()).reduce((sum, part) => {
    const msrp = part.catalogItem?.msrp;
    return typeof msrp === 'number' && msrp > 0 ? sum + msrp : sum;
  }, 0);
  const msrpLabel = msrpTotal > 0
    ? new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(msrpTotal)
    : 'N/A';
  const buildVideoEmbedURL = getYouTubeEmbedURL(build?.youtubeUrl);
  const flightVideoEmbedURL = getYouTubeEmbedURL(build?.flightYoutubeUrl);

  return (
    <div
      className="fixed inset-0 z-[80] flex items-center justify-center p-4"
      onClick={(event) => event.target === event.currentTarget && onClose()}
    >
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" />
      <div className="relative w-full max-w-3xl max-h-[90vh] overflow-hidden rounded-2xl border border-slate-700 bg-slate-900 flex flex-col">
        <div className="flex items-center justify-between px-5 py-4 border-b border-slate-700">
          <h3 className="text-lg font-semibold text-white">Build Details</h3>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close build details modal"
            className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-5 space-y-4">
          {isLoading && (
            <div className="rounded-lg border border-slate-700 bg-slate-800/60 p-6 text-center text-slate-400">
              Loading build details...
            </div>
          )}
          {!isLoading && error && (
            <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-4 text-sm text-red-300">
              {error}
            </div>
          )}
          {!isLoading && !error && build && (
            <>
              <div className="space-y-2">
                <h4 className="text-xl font-semibold text-white">{build.title}</h4>
                {build.description && <p className="text-sm text-slate-300">{build.description}</p>}
              </div>

              {build.mainImageUrl && (
                <div className="overflow-hidden rounded-xl border border-slate-700 bg-slate-800/60">
                  <img src={build.mainImageUrl} alt={build.title} className="max-h-[320px] w-full object-cover" />
                </div>
              )}

              {(buildVideoEmbedURL || flightVideoEmbedURL) && (
                <div className="rounded-xl border border-slate-700 bg-slate-800/60 p-4">
                  <h5 className="text-sm font-semibold uppercase tracking-wide text-slate-400 mb-3">Videos</h5>
                  <div className="space-y-4">
                    {buildVideoEmbedURL && (
                      <div>
                        <p className="mb-2 text-sm font-medium text-slate-300">Build Video</p>
                        <div className="aspect-video overflow-hidden rounded-lg border border-slate-700 bg-slate-900/60">
                          <iframe
                            src={buildVideoEmbedURL}
                            title={`${build.title} - Build Video`}
                            className="h-full w-full"
                            loading="lazy"
                            referrerPolicy="strict-origin-when-cross-origin"
                            allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
                            allowFullScreen
                          />
                        </div>
                      </div>
                    )}
                    {flightVideoEmbedURL && (
                      <div>
                        <p className="mb-2 text-sm font-medium text-slate-300">Flight Video</p>
                        <div className="aspect-video overflow-hidden rounded-lg border border-slate-700 bg-slate-900/60">
                          <iframe
                            src={flightVideoEmbedURL}
                            title={`${build.title} - Flight Video`}
                            className="h-full w-full"
                            loading="lazy"
                            referrerPolicy="strict-origin-when-cross-origin"
                            allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
                            allowFullScreen
                          />
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              )}

              <div className="rounded-lg border border-slate-700 bg-slate-800/60 p-4">
                <div className="flex items-center justify-between">
                  <h5 className="text-sm font-semibold uppercase tracking-wide text-slate-400">Estimated MSRP</h5>
                  <span className="text-lg font-semibold text-primary-300">{msrpLabel}</span>
                </div>
              </div>

              <div className="rounded-lg border border-slate-700 bg-slate-800/60 p-4">
                <h5 className="text-sm font-semibold uppercase tracking-wide text-slate-400 mb-3">Core Components</h5>
                <ul className="space-y-2 text-sm text-slate-300">
                  <li>Frame: {frame ? getBuildPartDisplayName(frame) : '—'}</li>
                  <li>Motors: {motors ? getBuildPartDisplayName(motors) : '—'}</li>
                  <li>
                    Power: {aio?.catalogItem
                      ? `AIO — ${getBuildPartDisplayName(aio)}`
                      : stack?.catalogItem
                        ? `FC/ESC Stack — ${getBuildPartDisplayName(stack)}`
                      : fc?.catalogItem || esc?.catalogItem
                        ? `${fc?.catalogItem ? getBuildPartDisplayName(fc) : 'FC'} + ${esc?.catalogItem ? getBuildPartDisplayName(esc) : 'ESC'}`
                        : '—'}
                  </li>
                  <li>Receiver: {receiver ? getBuildPartDisplayName(receiver) : '—'}</li>
                  <li>VTX: {vtx ? getBuildPartDisplayName(vtx) : '—'}</li>
                </ul>
              </div>
            </>
          )}
        </div>

        <div className="px-5 py-4 border-t border-slate-700 bg-slate-900/80 flex items-center justify-end gap-2">
          {build && (
            <Link
              to={`/builds/${build.id}`}
              className="rounded-lg bg-primary-600 px-3 py-2 text-sm font-medium text-white hover:bg-primary-500 transition-colors"
            >
              Open Build Page
            </Link>
          )}
        </div>
      </div>
    </div>
  );
}

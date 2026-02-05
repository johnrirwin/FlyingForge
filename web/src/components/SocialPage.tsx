import { useState, useCallback, useEffect, useMemo } from 'react';
import { searchPilots, getPilotProfile, discoverPilots } from '../pilotApi';
import { getFollowing, getFollowers } from '../socialApi';
import { updateProfile, validateCallSign } from '../profileApi';
import type { PilotSearchResult, PilotSummary, PilotProfile, PilotSummaryWithFollowers } from '../socialTypes';
import { useDebounce } from '../hooks';
import { useAuth } from '../hooks/useAuth';
import { trackEvent } from '../hooks/useGoogleAnalytics';

type SocialTab = 'search' | 'following' | 'followers';

interface SocialPageProps {
  onSelectPilot: (pilotId: string) => void;
}

interface CallSignPromptModalProps {
  onClose: () => void;
  onSave: (callSign: string) => Promise<void>;
  title?: string;
  subtitle?: string;
  description?: string;
  initialCallSign?: string;
}

// Modal component for prompting callsign setup
export function CallSignPromptModal({ 
  onClose, 
  onSave,
  title = 'Set Your Call Sign',
  subtitle = 'Required to appear in search results',
  description = 'To protect your privacy, you need a call sign to be visible to other pilots in the community. Only your call sign will be shown publicly.',
  initialCallSign = '',
}: CallSignPromptModalProps) {
  const [callSign, setCallSign] = useState(initialCallSign);
  const [error, setError] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    const trimmedCallSign = callSign.trim();
    if (!trimmedCallSign) {
      setError('Call sign is required');
      return;
    }

    const validationError = validateCallSign(trimmedCallSign);
    if (validationError) {
      setError(validationError);
      return;
    }

    try {
      setIsSaving(true);
      setError(null);
      await onSave(trimmedCallSign);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save call sign');
      setIsSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
      <div className="bg-slate-800 rounded-xl p-6 w-full max-w-md shadow-2xl">
        <div className="flex items-center gap-3 mb-4">
          <div className="w-10 h-10 rounded-full bg-emerald-600/20 flex items-center justify-center">
            <svg className="w-5 h-5 text-emerald-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
            </svg>
          </div>
          <div>
            <h2 className="text-lg font-bold text-white">{title}</h2>
            <p className="text-sm text-slate-400">{subtitle}</p>
          </div>
        </div>

        <p className="text-slate-300 text-sm mb-4">
          {description}
        </p>

        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label htmlFor="callSign" className="block text-sm font-medium text-slate-300 mb-2">
              Call Sign
            </label>
            <input
              type="text"
              id="callSign"
              value={callSign}
              onChange={(e) => {
                setCallSign(e.target.value);
                setError(null);
              }}
              className={`w-full px-4 py-3 bg-slate-700 border rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-emerald-500 ${
                error ? 'border-red-500' : 'border-slate-600'
              }`}
              placeholder="Enter your call sign"
              autoFocus
              disabled={isSaving}
            />
            {error && (
              <p className="mt-2 text-sm text-red-400">{error}</p>
            )}
            <p className="mt-2 text-xs text-slate-500">
              3-20 characters, letters, numbers, underscores, and hyphens only
            </p>
          </div>

          <div className="flex gap-3">
            <button
              type="button"
              onClick={onClose}
              disabled={isSaving}
              className="flex-1 px-4 py-2.5 bg-slate-700 hover:bg-slate-600 disabled:bg-slate-700 text-slate-300 rounded-lg font-medium transition-colors"
            >
              Skip for now
            </button>
            <button
              type="submit"
              disabled={isSaving || !callSign.trim()}
              className="flex-1 px-4 py-2.5 bg-emerald-600 hover:bg-emerald-500 disabled:bg-emerald-600/50 text-white rounded-lg font-medium transition-colors"
            >
              {isSaving ? 'Saving...' : 'Save Call Sign'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

export function SocialPage({ onSelectPilot }: SocialPageProps) {
  const { user, isAuthenticated, updateUser } = useAuth();
  const [activeTab, setActiveTab] = useState<SocialTab>('search');
  
  // My profile state
  const [myProfile, setMyProfile] = useState<PilotProfile | null>(null);
  const [isLoadingProfile, setIsLoadingProfile] = useState(false);
  
  // Call sign prompt state
  const [showCallSignPrompt, setShowCallSignPrompt] = useState(false);
  const [hasShownPrompt, setHasShownPrompt] = useState(false);
  
  // Search state
  const [query, setQuery] = useState('');
  const [searchResults, setSearchResults] = useState<PilotSearchResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [searchError, setSearchError] = useState<string | null>(null);
  const [hasSearched, setHasSearched] = useState(false);

  // Discovery state (featured pilots)
  const [popularPilots, setPopularPilots] = useState<PilotSummaryWithFollowers[]>([]);
  const [recentPilots, setRecentPilots] = useState<PilotSummary[]>([]);
  const [isLoadingDiscovery, setIsLoadingDiscovery] = useState(false);
  const [discoveryError, setDiscoveryError] = useState<string | null>(null);

  // Filter state for following/followers lists
  const [filterQuery, setFilterQuery] = useState('');

  // Following/Followers state
  const [following, setFollowing] = useState<PilotSummary[]>([]);
  const [followers, setFollowers] = useState<PilotSummary[]>([]);
  const [followingCount, setFollowingCount] = useState(0);
  const [followersCount, setFollowersCount] = useState(0);
  const [isLoadingFollowing, setIsLoadingFollowing] = useState(false);
  const [isLoadingFollowers, setIsLoadingFollowers] = useState(false);
  const [followingError, setFollowingError] = useState<string | null>(null);
  const [followersError, setFollowersError] = useState<string | null>(null);

  const debouncedQuery = useDebounce(query, 300);

  // Filter following list based on filterQuery
  const filteredFollowing = useMemo(() => {
    if (!filterQuery.trim()) return following;
    const lowerQuery = filterQuery.toLowerCase();
    return following.filter(pilot => 
      pilot.callSign?.toLowerCase().includes(lowerQuery) ||
      pilot.displayName?.toLowerCase().includes(lowerQuery)
    );
  }, [following, filterQuery]);

  // Filter followers list based on filterQuery
  const filteredFollowers = useMemo(() => {
    if (!filterQuery.trim()) return followers;
    const lowerQuery = filterQuery.toLowerCase();
    return followers.filter(pilot => 
      pilot.callSign?.toLowerCase().includes(lowerQuery) ||
      pilot.displayName?.toLowerCase().includes(lowerQuery)
    );
  }, [followers, filterQuery]);

  // Clear filter when changing tabs
  useEffect(() => {
    setFilterQuery('');
  }, [activeTab]);

  // Search when debounced query changes
  const handleSearch = useCallback(async (searchQuery: string) => {
    if (searchQuery.length < 2) {
      setSearchResults([]);
      setHasSearched(false);
      return;
    }

    try {
      setIsSearching(true);
      setSearchError(null);
      const response = await searchPilots(searchQuery);
      setSearchResults(response.pilots);
      setHasSearched(true);
      // Track pilot search (don't include query for privacy)
      trackEvent('social_pilot_search', { result_count: response.pilots.length });
    } catch (err) {
      setSearchError(err instanceof Error ? err.message : 'Search failed');
      setSearchResults([]);
    } finally {
      setIsSearching(false);
    }
  }, []);

  // Effect to trigger search on debounced query change
  useEffect(() => {
    if (activeTab === 'search') {
      if (debouncedQuery) {
        handleSearch(debouncedQuery);
      } else {
        setSearchResults([]);
        setHasSearched(false);
      }
    }
  }, [debouncedQuery, handleSearch, activeTab]);

  // Load discovery data (popular and recent pilots)
  const loadDiscovery = useCallback(async () => {
    if (!isAuthenticated) return;
    
    try {
      setIsLoadingDiscovery(true);
      setDiscoveryError(null);
      const response = await discoverPilots(10);
      setPopularPilots(response.popular);
      setRecentPilots(response.recent);
      trackEvent('social_view_discover');
    } catch (err) {
      setDiscoveryError(err instanceof Error ? err.message : 'Failed to load pilots');
    } finally {
      setIsLoadingDiscovery(false);
    }
  }, [isAuthenticated]);

  // Load discovery on mount
  useEffect(() => {
    if (isAuthenticated) {
      loadDiscovery();
    }
  }, [isAuthenticated, loadDiscovery]);

  // Load following list
  const loadFollowing = useCallback(async () => {
    if (!user?.id) return;
    
    try {
      setIsLoadingFollowing(true);
      setFollowingError(null);
      const response = await getFollowing(user.id, 50, 0);
      setFollowing(response.pilots);
      setFollowingCount(response.totalCount);
    } catch (err) {
      setFollowingError(err instanceof Error ? err.message : 'Failed to load following');
    } finally {
      setIsLoadingFollowing(false);
    }
  }, [user?.id]);

  // Load followers list
  const loadFollowers = useCallback(async () => {
    if (!user?.id) return;
    
    try {
      setIsLoadingFollowers(true);
      setFollowersError(null);
      const response = await getFollowers(user.id, 50, 0);
      setFollowers(response.pilots);
      setFollowersCount(response.totalCount);
    } catch (err) {
      setFollowersError(err instanceof Error ? err.message : 'Failed to load followers');
    } finally {
      setIsLoadingFollowers(false);
    }
  }, [user?.id]);

  // Load my profile on mount
  useEffect(() => {
    if (user?.id && isAuthenticated) {
      setIsLoadingProfile(true);
      getPilotProfile(user.id)
        .then(profile => {
          setMyProfile(profile);
          setFollowersCount(profile.followerCount);
          setFollowingCount(profile.followingCount);
        })
        .catch(() => {
          // Silently fail - user can still use other features
        })
        .finally(() => setIsLoadingProfile(false));
    }
  }, [user?.id, isAuthenticated]);

  // Show call sign prompt if user doesn't have one
  useEffect(() => {
    if (isAuthenticated && user && !user.callSign && !hasShownPrompt && !isLoadingProfile) {
      setShowCallSignPrompt(true);
      setHasShownPrompt(true);
    }
  }, [isAuthenticated, user, hasShownPrompt, isLoadingProfile]);

  // Handler for saving call sign from prompt
  const handleSaveCallSign = async (callSign: string) => {
    await updateProfile({ callSign });
    updateUser({ callSign });
    setMyProfile(prev => prev ? { ...prev, callSign } : prev);
    setShowCallSignPrompt(false);
  };

  // Load data when tab changes
  useEffect(() => {
    if (activeTab === 'following' && isAuthenticated) {
      loadFollowing();
      // Track viewing following list
      trackEvent('social_view_following');
    } else if (activeTab === 'followers' && isAuthenticated) {
      loadFollowers();
      // Track viewing followers list
      trackEvent('social_view_followers');
    }
  }, [activeTab, isAuthenticated, loadFollowing, loadFollowers]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && query.length >= 2) {
      handleSearch(query);
    }
  };

  const getDisplayName = (pilot: PilotSearchResult | PilotSummary) => {
    return pilot.callSign || 'Unknown Pilot';
  };

  const PilotCard = ({ pilot }: { pilot: PilotSearchResult | PilotSummary }) => (
    <button
      onClick={() => {
        onSelectPilot(pilot.id);
        // Track viewing pilot profile
        trackEvent('social_view_profile');
      }}
      className="flex items-center gap-3 p-4 bg-slate-800 hover:bg-slate-700 rounded-lg transition-colors text-left group"
    >
      {/* Avatar */}
      {pilot.effectiveAvatarUrl ? (
        <img
          src={pilot.effectiveAvatarUrl}
          alt=""
          className="w-12 h-12 rounded-full object-cover flex-shrink-0"
        />
      ) : (
        <div className="w-12 h-12 rounded-full bg-emerald-600 flex items-center justify-center flex-shrink-0">
          <span className="text-lg font-semibold text-white">
            {getDisplayName(pilot).charAt(0).toUpperCase()}
          </span>
        </div>
      )}

      {/* Info */}
      <div className="flex-1 min-w-0">
        <div className="font-medium text-white truncate">
          {getDisplayName(pilot)}
        </div>
        {pilot.displayName && (
          <div className="text-sm text-slate-400 truncate">
            {pilot.displayName}
          </div>
        )}
      </div>

      {/* Arrow */}
      <svg
        className="w-5 h-5 text-slate-500 group-hover:text-slate-400 transition-colors flex-shrink-0"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M9 5l7 7-7 7"
        />
      </svg>
    </button>
  );

  const EmptyState = ({ icon, title, subtitle }: { icon: React.ReactNode; title: string; subtitle: string }) => (
    <div className="text-center py-12 text-slate-500">
      {icon}
      <p className="text-lg">{title}</p>
      <p className="text-sm mt-1">{subtitle}</p>
    </div>
  );

  const getMyDisplayName = () => {
    return myProfile?.callSign || 'Set your callsign';
  };

  return (
    <div className="w-full p-6 flex flex-col items-center">
      {/* Call Sign Prompt Modal */}
      {showCallSignPrompt && (
        <CallSignPromptModal
          onClose={() => setShowCallSignPrompt(false)}
          onSave={handleSaveCallSign}
        />
      )}

      {/* Centered header section - always centered */}
      <div className="w-full max-w-xl">
        {/* My Profile Header */}
        {isAuthenticated && (
          <div className="bg-slate-800 rounded-lg p-6 mb-6">
          {isLoadingProfile ? (
            <div className="flex items-center gap-4">
              <div className="w-16 h-16 rounded-full bg-slate-700 animate-pulse" />
              <div className="flex-1">
                <div className="h-6 w-32 bg-slate-700 rounded animate-pulse mb-2" />
                <div className="h-4 w-24 bg-slate-700 rounded animate-pulse" />
              </div>
            </div>
          ) : (
            <div className="flex items-center gap-4">
              {/* Avatar */}
              <button
                onClick={() => user?.id && onSelectPilot(user.id)}
                className="focus:outline-none focus:ring-2 focus:ring-primary-500 rounded-full"
              >
                {myProfile?.effectiveAvatarUrl ? (
                  <img
                    src={myProfile.effectiveAvatarUrl}
                    alt=""
                    className="w-16 h-16 rounded-full object-cover border-2 border-slate-600 hover:border-primary-500 transition-colors"
                  />
                ) : (
                  <div className="w-16 h-16 rounded-full bg-slate-700 flex items-center justify-center border-2 border-slate-600 hover:border-primary-500 transition-colors">
                    <svg className="w-8 h-8 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                    </svg>
                  </div>
                )}
              </button>

              {/* Info */}
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <button
                    onClick={() => user?.id && onSelectPilot(user.id)}
                    className="text-xl font-bold text-white hover:text-primary-400 transition-colors"
                  >
                    {getMyDisplayName()}
                  </button>
                </div>
                {myProfile?.displayName && (
                  <p className="text-slate-400 mt-1">{myProfile.displayName}</p>
                )}
                <div className="flex items-center gap-4 mt-2 text-sm">
                  <button
                    onClick={() => setActiveTab('followers')}
                    className="text-slate-400 hover:text-white transition-colors"
                  >
                    <span className="font-medium text-white">{followersCount}</span> followers
                  </button>
                  <button
                    onClick={() => setActiveTab('following')}
                    className="text-slate-400 hover:text-white transition-colors"
                  >
                    <span className="font-medium text-white">{followingCount}</span> following
                  </button>
                </div>
              </div>

              {/* View Profile Button */}
              <button
                onClick={() => user?.id && onSelectPilot(user.id)}
                className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-lg font-medium transition-colors"
              >
                View Profile
              </button>
            </div>
          )}
        </div>
      )}

      {/* Section Title */}
      <div className="mb-6">
        <h2 className="text-lg font-semibold text-white">Find Pilots</h2>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 mb-6 bg-slate-800 p-1 rounded-lg w-full overflow-hidden">
        <button
          onClick={() => setActiveTab('search')}
          className={`flex-1 min-w-0 px-2 sm:px-4 py-2 rounded-md font-medium transition-colors text-sm sm:text-base ${
            activeTab === 'search'
              ? 'bg-primary-500 text-white'
              : 'text-slate-400 hover:text-white hover:bg-slate-700'
          }`}
        >
          <span className="flex items-center justify-center gap-1 sm:gap-2">
            <svg className="w-4 h-4 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            <span className="truncate">Search</span>
          </span>
        </button>
        {isAuthenticated && (
          <>
            <button
              onClick={() => setActiveTab('following')}
              className={`flex-1 min-w-0 px-2 sm:px-4 py-2 rounded-md font-medium transition-colors text-sm sm:text-base ${
                activeTab === 'following'
                  ? 'bg-primary-500 text-white'
                  : 'text-slate-400 hover:text-white hover:bg-slate-700'
              }`}
            >
              <span className="flex items-center justify-center gap-1 sm:gap-2">
                <svg className="w-4 h-4 flex-shrink-0 hidden sm:block" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                </svg>
                <span className="truncate">Following</span>
                {followingCount > 0 && (
                  <span className="text-xs bg-slate-600 px-1.5 py-0.5 rounded-full flex-shrink-0">
                    {followingCount}
                  </span>
                )}
              </span>
            </button>
            <button
              onClick={() => setActiveTab('followers')}
              className={`flex-1 min-w-0 px-2 sm:px-4 py-2 rounded-md font-medium transition-colors text-sm sm:text-base ${
                activeTab === 'followers'
                  ? 'bg-primary-500 text-white'
                  : 'text-slate-400 hover:text-white hover:bg-slate-700'
              }`}
            >
              <span className="flex items-center justify-center gap-1 sm:gap-2">
                <svg className="w-4 h-4 flex-shrink-0 hidden sm:block" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
                </svg>
                <span className="truncate">Followers</span>
                {followersCount > 0 && (
                  <span className="text-xs bg-slate-600 px-1.5 py-0.5 rounded-full flex-shrink-0">
                    {followersCount}
                  </span>
                )}
              </span>
            </button>
          </>
        )}
      </div>

        {/* Search Input - shown on all tabs */}
        <div className="mb-6">
          <div className="relative">
            <input
              type="text"
              value={activeTab === 'search' ? query : filterQuery}
              onChange={(e) => activeTab === 'search' ? setQuery(e.target.value) : setFilterQuery(e.target.value)}
              onKeyDown={activeTab === 'search' ? handleKeyDown : undefined}
              placeholder={
                activeTab === 'search' 
                  ? "Search by callsign or name..." 
                  : activeTab === 'following'
                    ? "Filter following..."
                    : "Filter followers..."
              }
              className="w-full px-4 py-3 pl-12 bg-slate-800 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:border-primary-500"
            />
            <svg
              className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-500"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
              />
            </svg>
            {activeTab === 'search' && isSearching && (
              <div className="absolute right-4 top-1/2 -translate-y-1/2">
                <div className="animate-spin rounded-full h-5 w-5 border-t-2 border-b-2 border-primary-500"></div>
              </div>
            )}
          </div>
          {activeTab === 'search' && (
            <p className="mt-2 text-xs text-slate-500">
              Enter at least 2 characters to search
            </p>
          )}
        </div>
      </div>

      {/* Full width results section */}
      <div className="w-full">
        {/* Search Tab Results */}
        {activeTab === 'search' && (
          <>
            {/* Error */}
            {searchError && (
              <div className="max-w-xl mx-auto mb-4 p-4 bg-red-500/10 border border-red-500/50 rounded-lg text-red-400">
                {searchError}
              </div>
            )}

            {/* Results */}
            {hasSearched && searchResults.length === 0 && !isSearching && (
              <EmptyState
                icon={
                  <svg className="w-16 h-16 mx-auto mb-4 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                  </svg>
                }
                title="No pilots found"
                subtitle="Try a different search term"
              />
            )}

            {searchResults.length > 0 && (
              <div>
                <p className="text-sm text-slate-400 mb-4">
                  Found {searchResults.length} pilot{searchResults.length !== 1 ? 's' : ''}
                </p>
                <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
                  {searchResults.map((pilot) => (
                    <PilotCard key={pilot.id} pilot={pilot} />
                  ))}
                </div>
              </div>
            )}

            {/* Initial state - show discovery */}
            {!hasSearched && !isSearching && (
              <div className="space-y-8">
                {/* Loading state */}
                {isLoadingDiscovery && (
                  <div className="flex items-center justify-center py-12">
                    <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary-500"></div>
                  </div>
                )}

                {/* Error state */}
                {discoveryError && (
                  <div className="max-w-xl mx-auto mb-4 p-4 bg-red-500/10 border border-red-500/50 rounded-lg text-red-400">
                    {discoveryError}
                  </div>
                )}

                {/* Discovery content */}
                {!isLoadingDiscovery && !discoveryError && (
                  <>
                    {/* Popular Pilots */}
                    {popularPilots.length > 0 && (
                      <div>
                        <div className="flex items-center gap-2 mb-4">
                          <svg className="w-5 h-5 text-yellow-500" fill="currentColor" viewBox="0 0 24 24">
                            <path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z" />
                          </svg>
                          <h3 className="text-lg font-semibold text-white">Popular Pilots</h3>
                        </div>
                        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
                          {popularPilots.map((pilot) => (
                            <div
                              key={pilot.id}
                              onClick={() => onSelectPilot(pilot.id)}
                              className="bg-slate-800 rounded-lg p-4 hover:bg-slate-700/80 transition-colors cursor-pointer border border-slate-700"
                            >
                              <div className="flex items-center gap-3">
                                {pilot.effectiveAvatarUrl ? (
                                  <img 
                                    src={pilot.effectiveAvatarUrl} 
                                    alt={pilot.callSign || 'Avatar'} 
                                    className="w-12 h-12 rounded-full object-cover"
                                  />
                                ) : (
                                  <div className="w-12 h-12 rounded-full bg-slate-600 flex items-center justify-center">
                                    <svg className="w-6 h-6 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                                    </svg>
                                  </div>
                                )}
                                <div className="flex-1 min-w-0">
                                  <p className="font-medium text-white truncate">{pilot.callSign || 'Unknown'}</p>
                                  {pilot.displayName && pilot.displayName !== pilot.callSign && (
                                    <p className="text-sm text-slate-400 truncate">{pilot.displayName}</p>
                                  )}
                                  <p className="text-xs text-slate-500 mt-1">
                                    {pilot.followerCount} follower{pilot.followerCount !== 1 ? 's' : ''}
                                  </p>
                                </div>
                              </div>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}

                    {/* Recently Joined */}
                    {recentPilots.length > 0 && (
                      <div>
                        <div className="flex items-center gap-2 mb-4">
                          <svg className="w-5 h-5 text-emerald-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18 9v3m0 0v3m0-3h3m-3 0h-3m-2-5a4 4 0 11-8 0 4 4 0 018 0zM3 20a6 6 0 0112 0v1H3v-1z" />
                          </svg>
                          <h3 className="text-lg font-semibold text-white">Recently Joined</h3>
                        </div>
                        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
                          {recentPilots.map((pilot) => (
                            <div
                              key={pilot.id}
                              onClick={() => onSelectPilot(pilot.id)}
                              className="bg-slate-800 rounded-lg p-4 hover:bg-slate-700/80 transition-colors cursor-pointer border border-slate-700"
                            >
                              <div className="flex items-center gap-3">
                                {pilot.effectiveAvatarUrl ? (
                                  <img 
                                    src={pilot.effectiveAvatarUrl} 
                                    alt={pilot.callSign || 'Avatar'} 
                                    className="w-12 h-12 rounded-full object-cover"
                                  />
                                ) : (
                                  <div className="w-12 h-12 rounded-full bg-slate-600 flex items-center justify-center">
                                    <svg className="w-6 h-6 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
                                    </svg>
                                  </div>
                                )}
                                <div className="flex-1 min-w-0">
                                  <p className="font-medium text-white truncate">{pilot.callSign || 'Unknown'}</p>
                                  {pilot.displayName && pilot.displayName !== pilot.callSign && (
                                    <p className="text-sm text-slate-400 truncate">{pilot.displayName}</p>
                                  )}
                                </div>
                              </div>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}

                    {/* Fallback empty state if no pilots */}
                    {popularPilots.length === 0 && recentPilots.length === 0 && (
                      <EmptyState
                        icon={
                          <svg className="w-16 h-16 mx-auto mb-4 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                          </svg>
                        }
                        title="Search for pilots"
                        subtitle="Find other FPV enthusiasts by their callsign"
                      />
                    )}
                  </>
                )}
              </div>
            )}
          </>
        )}

        {/* Following Tab */}
        {activeTab === 'following' && (
          <>
            {isLoadingFollowing && (
              <div className="flex items-center justify-center py-12">
                <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary-500"></div>
              </div>
            )}

            {followingError && (
              <div className="max-w-xl mx-auto mb-4 p-4 bg-red-500/10 border border-red-500/50 rounded-lg text-red-400">
                {followingError}
              </div>
            )}

            {!isLoadingFollowing && following.length === 0 && (
              <EmptyState
                icon={
                  <svg className="w-16 h-16 mx-auto mb-4 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18 9v3m0 0v3m0-3h3m-3 0h-3m-2-5a4 4 0 11-8 0 4 4 0 018 0zM3 20a6 6 0 0112 0v1H3v-1z" />
                  </svg>
                }
                title="Not following anyone yet"
                subtitle="Search for pilots and follow them to see them here"
              />
            )}

            {!isLoadingFollowing && following.length > 0 && filteredFollowing.length === 0 && filterQuery && (
              <EmptyState
                icon={
                  <svg className="w-16 h-16 mx-auto mb-4 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                  </svg>
                }
                title="No matches found"
                subtitle="Try a different filter term"
              />
            )}

            {!isLoadingFollowing && filteredFollowing.length > 0 && (
              <div>
                <p className="text-sm text-slate-400 mb-4">
                  {filterQuery 
                    ? `Showing ${filteredFollowing.length} of ${followingCount} pilot${followingCount !== 1 ? 's' : ''}`
                    : `Following ${followingCount} pilot${followingCount !== 1 ? 's' : ''}`
                  }
                </p>
                <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
                  {filteredFollowing.map((pilot) => (
                    <PilotCard key={pilot.id} pilot={pilot} />
                  ))}
                </div>
              </div>
            )}
          </>
        )}

        {/* Followers Tab */}
        {activeTab === 'followers' && (
          <>
            {isLoadingFollowers && (
              <div className="flex items-center justify-center py-12">
                <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary-500"></div>
              </div>
            )}

            {followersError && (
              <div className="max-w-xl mx-auto mb-4 p-4 bg-red-500/10 border border-red-500/50 rounded-lg text-red-400">
                {followersError}
              </div>
            )}

            {!isLoadingFollowers && followers.length === 0 && (
              <EmptyState
                icon={
                  <svg className="w-16 h-16 mx-auto mb-4 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
                  </svg>
                }
                title="No followers yet"
                subtitle="Share your profile to get followers"
              />
            )}

            {!isLoadingFollowers && followers.length > 0 && filteredFollowers.length === 0 && filterQuery && (
              <EmptyState
                icon={
                  <svg className="w-16 h-16 mx-auto mb-4 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                  </svg>
                }
                title="No matches found"
                subtitle="Try a different filter term"
              />
            )}

            {!isLoadingFollowers && filteredFollowers.length > 0 && (
              <div>
                <p className="text-sm text-slate-400 mb-4">
                  {filterQuery 
                    ? `Showing ${filteredFollowers.length} of ${followersCount} follower${followersCount !== 1 ? 's' : ''}`
                    : `${followersCount} follower${followersCount !== 1 ? 's' : ''}`
                  }
                </p>
                <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-4">
                  {filteredFollowers.map((pilot) => (
                    <PilotCard key={pilot.id} pilot={pilot} />
                  ))}
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}

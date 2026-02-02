import { useState, useCallback, useEffect } from 'react';
import { searchPilots, getPilotProfile } from '../pilotApi';
import { getFollowing, getFollowers } from '../socialApi';
import type { PilotSearchResult, PilotSummary, PilotProfile } from '../socialTypes';
import { useDebounce } from '../hooks';
import { useAuth } from '../hooks/useAuth';

type SocialTab = 'search' | 'following' | 'followers';

interface SocialPageProps {
  onSelectPilot: (pilotId: string) => void;
}

export function SocialPage({ onSelectPilot }: SocialPageProps) {
  const { user, isAuthenticated } = useAuth();
  const [activeTab, setActiveTab] = useState<SocialTab>('search');
  
  // My profile state
  const [myProfile, setMyProfile] = useState<PilotProfile | null>(null);
  const [isLoadingProfile, setIsLoadingProfile] = useState(false);
  
  // Search state
  const [query, setQuery] = useState('');
  const [searchResults, setSearchResults] = useState<PilotSearchResult[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const [searchError, setSearchError] = useState<string | null>(null);
  const [hasSearched, setHasSearched] = useState(false);

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

  // Load data when tab changes
  useEffect(() => {
    if (activeTab === 'following' && isAuthenticated) {
      loadFollowing();
    } else if (activeTab === 'followers' && isAuthenticated) {
      loadFollowers();
    }
  }, [activeTab, isAuthenticated, loadFollowing, loadFollowers]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && query.length >= 2) {
      handleSearch(query);
    }
  };

  const getDisplayName = (pilot: PilotSearchResult | PilotSummary) => {
    if (pilot.callSign) return pilot.callSign;
    if (pilot.displayName) return pilot.displayName;
    if ('googleName' in pilot && pilot.googleName) return pilot.googleName;
    return 'Unknown Pilot';
  };

  const getSecondaryInfo = (pilot: PilotSearchResult | PilotSummary) => {
    if (pilot.callSign && pilot.displayName) return pilot.displayName;
    if ('googleName' in pilot && pilot.callSign && pilot.googleName) return pilot.googleName;
    return null;
  };

  const PilotCard = ({ pilot }: { pilot: PilotSearchResult | PilotSummary }) => (
    <button
      onClick={() => onSelectPilot(pilot.id)}
      className="w-full flex items-center gap-4 p-4 bg-slate-800 hover:bg-slate-700 rounded-lg transition-colors text-left"
    >
      {/* Avatar */}
      {pilot.effectiveAvatarUrl ? (
        <img
          src={pilot.effectiveAvatarUrl}
          alt=""
          className="w-12 h-12 rounded-full object-cover"
        />
      ) : (
        <div className="w-12 h-12 rounded-full bg-slate-700 flex items-center justify-center">
          <svg
            className="w-6 h-6 text-slate-500"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"
            />
          </svg>
        </div>
      )}

      {/* Info */}
      <div className="flex-1 min-w-0">
        <div className="font-medium text-white truncate">
          {getDisplayName(pilot)}
        </div>
        {getSecondaryInfo(pilot) && (
          <div className="text-sm text-slate-400 truncate">
            {getSecondaryInfo(pilot)}
          </div>
        )}
      </div>

      {/* Arrow */}
      <svg
        className="w-5 h-5 text-slate-500"
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
    if (myProfile?.callSign) return myProfile.callSign;
    if (myProfile?.displayName) return myProfile.displayName;
    if (myProfile?.googleName) return myProfile.googleName;
    return user?.displayName || 'Pilot';
  };

  const getMySecondaryName = () => {
    if (myProfile?.callSign && myProfile?.displayName) return myProfile.displayName;
    if (myProfile?.callSign && myProfile?.googleName) return myProfile.googleName;
    return null;
  };

  return (
    <div className="max-w-4xl mx-auto p-6">
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
                {getMySecondaryName() && (
                  <p className="text-slate-400 text-sm">{getMySecondaryName()}</p>
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
      <div className="flex gap-1 mb-6 bg-slate-800 p-1 rounded-lg w-fit">
        <button
          onClick={() => setActiveTab('search')}
          className={`px-4 py-2 rounded-md font-medium transition-colors ${
            activeTab === 'search'
              ? 'bg-primary-500 text-white'
              : 'text-slate-400 hover:text-white hover:bg-slate-700'
          }`}
        >
          <span className="flex items-center gap-2">
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
            Search
          </span>
        </button>
        {isAuthenticated && (
          <>
            <button
              onClick={() => setActiveTab('following')}
              className={`px-4 py-2 rounded-md font-medium transition-colors ${
                activeTab === 'following'
                  ? 'bg-primary-500 text-white'
                  : 'text-slate-400 hover:text-white hover:bg-slate-700'
              }`}
            >
              <span className="flex items-center gap-2">
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                </svg>
                Following
                {followingCount > 0 && (
                  <span className="text-xs bg-slate-600 px-1.5 py-0.5 rounded-full">
                    {followingCount}
                  </span>
                )}
              </span>
            </button>
            <button
              onClick={() => setActiveTab('followers')}
              className={`px-4 py-2 rounded-md font-medium transition-colors ${
                activeTab === 'followers'
                  ? 'bg-primary-500 text-white'
                  : 'text-slate-400 hover:text-white hover:bg-slate-700'
              }`}
            >
              <span className="flex items-center gap-2">
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z" />
                </svg>
                Followers
                {followersCount > 0 && (
                  <span className="text-xs bg-slate-600 px-1.5 py-0.5 rounded-full">
                    {followersCount}
                  </span>
                )}
              </span>
            </button>
          </>
        )}
      </div>

      {/* Search Tab */}
      {activeTab === 'search' && (
        <>
          {/* Search Input */}
          <div className="mb-6">
            <div className="relative">
              <input
                type="text"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onKeyDown={handleKeyDown}
                placeholder="Search by callsign or name..."
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
              {isSearching && (
                <div className="absolute right-4 top-1/2 -translate-y-1/2">
                  <div className="animate-spin rounded-full h-5 w-5 border-t-2 border-b-2 border-primary-500"></div>
                </div>
              )}
            </div>
            <p className="mt-2 text-xs text-slate-500">
              Enter at least 2 characters to search
            </p>
          </div>

          {/* Error */}
          {searchError && (
            <div className="mb-4 p-4 bg-red-500/10 border border-red-500/50 rounded-lg text-red-400">
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
            <div className="space-y-2">
              <p className="text-sm text-slate-400 mb-4">
                Found {searchResults.length} pilot{searchResults.length !== 1 ? 's' : ''}
              </p>
              {searchResults.map((pilot) => (
                <PilotCard key={pilot.id} pilot={pilot} />
              ))}
            </div>
          )}

          {/* Initial state */}
          {!hasSearched && !isSearching && (
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

      {/* Following Tab */}
      {activeTab === 'following' && (
        <>
          {isLoadingFollowing && (
            <div className="flex items-center justify-center py-12">
              <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary-500"></div>
            </div>
          )}

          {followingError && (
            <div className="mb-4 p-4 bg-red-500/10 border border-red-500/50 rounded-lg text-red-400">
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

          {!isLoadingFollowing && following.length > 0 && (
            <div className="space-y-2">
              <p className="text-sm text-slate-400 mb-4">
                Following {followingCount} pilot{followingCount !== 1 ? 's' : ''}
              </p>
              {following.map((pilot) => (
                <PilotCard key={pilot.id} pilot={pilot} />
              ))}
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
            <div className="mb-4 p-4 bg-red-500/10 border border-red-500/50 rounded-lg text-red-400">
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

          {!isLoadingFollowers && followers.length > 0 && (
            <div className="space-y-2">
              <p className="text-sm text-slate-400 mb-4">
                {followersCount} follower{followersCount !== 1 ? 's' : ''}
              </p>
              {followers.map((pilot) => (
                <PilotCard key={pilot.id} pilot={pilot} />
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
}

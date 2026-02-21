import { useState, useEffect } from 'react';
import type { Aircraft } from '../aircraftTypes';
import type { FeedItem, SourceInfo } from '../types';
import type { PilotSummary } from '../socialTypes';
import type { InventoryItem, EquipmentCategory } from '../equipmentTypes';
import { getAircraftImageUrl } from '../aircraftApi';
import { getFollowers } from '../socialApi';
import { getInventory } from '../equipmentApi';
import { useAuth } from '../hooks/useAuth';

interface DashboardProps {
  // Data
  recentAircraft: Aircraft[];
  recentNews: FeedItem[];
  sources: SourceInfo[];
  // Loading states
  isAircraftLoading: boolean;
  isNewsLoading: boolean;
  // Actions
  onViewAllNews: () => void;
  onViewAllAircraft: () => void;
  onViewAllGear: () => void;
  onSelectAircraft: (aircraft: Aircraft) => void;
  onSelectNewsItem: (item: FeedItem) => void;
  onSelectPilot: (pilotId: string) => void;
  onGoToSocial: () => void;
}

// Skeleton loader component
function SkeletonCard({ className = '' }: { className?: string }) {
  return (
    <div className={`bg-slate-800 border border-slate-700 rounded-xl p-4 animate-pulse ${className}`}>
      <div className="flex gap-4">
        <div className="w-16 h-16 bg-slate-700 rounded-lg flex-shrink-0" />
        <div className="flex-1 space-y-2">
          <div className="h-4 bg-slate-700 rounded w-3/4" />
          <div className="h-3 bg-slate-700 rounded w-1/2" />
        </div>
      </div>
    </div>
  );
}

// Empty state component
function EmptyState({
  icon,
  title,
  description,
  actionLabel,
  onAction,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
  actionLabel?: string;
  onAction?: () => void;
}) {
  return (
    <div className="bg-slate-800/50 border border-slate-700/50 border-dashed rounded-xl p-6 text-center">
      <div className="w-12 h-12 bg-slate-700 rounded-full flex items-center justify-center mx-auto mb-3">
        {icon}
      </div>
      <h4 className="text-sm font-medium text-white mb-1">{title}</h4>
      <p className="text-xs text-slate-400 mb-3">{description}</p>
      {actionLabel && onAction && (
        <button
          onClick={onAction}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-primary-600 hover:bg-primary-700 text-white text-xs font-medium rounded-lg transition-colors"
        >
          <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          {actionLabel}
        </button>
      )}
    </div>
  );
}

// Aircraft card for dashboard
function DashboardAircraftCard({
  aircraft,
  onClick,
}: {
  aircraft: Aircraft;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className="w-full bg-slate-800 border border-slate-700 rounded-xl p-4 hover:bg-slate-700/50 hover:border-slate-600 transition-colors text-left group"
    >
      <div className="flex gap-4">
        {aircraft.hasImage ? (
          <img
            src={getAircraftImageUrl(aircraft.id)}
            alt={aircraft.name}
            className="w-16 h-16 rounded-lg object-cover flex-shrink-0"
          />
        ) : (
          <div className="w-16 h-16 bg-slate-700 rounded-lg flex items-center justify-center flex-shrink-0">
            <svg className="w-8 h-8 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
            </svg>
          </div>
        )}
        <div className="flex-1 min-w-0">
          <h4 className="text-sm font-medium text-white truncate group-hover:text-primary-400 transition-colors">
            {aircraft.name}
          </h4>
          {aircraft.nickname && (
            <p className="text-xs text-slate-400 truncate">"{aircraft.nickname}"</p>
          )}
          <span className="inline-block mt-1 px-2 py-0.5 bg-slate-700 rounded text-xs text-slate-300 capitalize">
            {aircraft.type.replace('_', ' ')}
          </span>
        </div>
      </div>
    </button>
  );
}

// Category display helper
const categoryDisplayNames: Record<EquipmentCategory, string> = {
  frames: 'Frame',
  vtx: 'VTX',
  flight_controllers: 'Flight Controller',
  esc: 'ESC',
  aio: 'AIO',
  stacks: 'FC/ESC Stack',
  motors: 'Motor',
  propellers: 'Propeller',
  receivers: 'Receiver',
  batteries: 'Battery',
  cameras: 'Camera',
  antennas: 'Antenna',
  accessories: 'Accessory',
};

// Gear card for dashboard
function DashboardGearCard({ 
  item,
  onClick,
}: { 
  item: InventoryItem;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className="w-full bg-slate-800 border border-slate-700 rounded-xl p-4 hover:bg-slate-700/50 hover:border-slate-600 transition-colors text-left group"
    >
      <div className="flex gap-4">
        {item.imageUrl ? (
          <img
            src={item.imageUrl}
            alt={item.name}
            className="w-12 h-12 rounded-lg object-cover flex-shrink-0"
          />
        ) : (
          <div className="w-12 h-12 bg-slate-700 rounded-lg flex items-center justify-center flex-shrink-0">
            <svg className="w-6 h-6 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
            </svg>
          </div>
        )}
        <div className="flex-1 min-w-0">
          <h4 className="text-sm font-medium text-white truncate group-hover:text-primary-400 transition-colors">
            {item.name}
          </h4>
          <p className="text-xs text-slate-400 truncate">
            {categoryDisplayNames[item.category]}
            {item.quantity > 1 && ` × ${item.quantity}`}
          </p>
          {item.manufacturer && (
            <p className="text-xs text-slate-500 mt-1 truncate">
              {item.manufacturer}
            </p>
          )}
        </div>
      </div>
    </button>
  );
}

// Recent followers component
function RecentFollowers({ 
  followers, 
  isLoading, 
  onSelectPilot,
  onViewAll 
}: { 
  followers: PilotSummary[]; 
  isLoading: boolean;
  onSelectPilot: (pilotId: string) => void;
  onViewAll: () => void;
}) {
  if (isLoading) {
    return (
      <div className="space-y-3">
        {[1, 2, 3].map(i => (
          <div key={i} className="flex items-center gap-3 animate-pulse">
            <div className="w-10 h-10 rounded-full bg-slate-700" />
            <div className="flex-1">
              <div className="h-4 bg-slate-700 rounded w-24" />
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (followers.length === 0) {
    return (
      <EmptyState
        icon={
          <svg className="w-6 h-6 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
          </svg>
        }
        title="No Followers Yet"
        description="Share your profile to get followers"
        actionLabel="Find Pilots"
        onAction={onViewAll}
      />
    );
  }

  const getDisplayName = (pilot: PilotSummary) => {
    if (pilot.callSign) return pilot.callSign;
    if (pilot.displayName) return pilot.displayName;
    return 'Pilot';
  };

  return (
    <div className="space-y-2">
      {followers.slice(0, 4).map(follower => (
        <button
          key={follower.id}
          onClick={() => onSelectPilot(follower.id)}
          className="w-full flex items-center gap-3 p-2 rounded-lg hover:bg-slate-700/50 transition-colors text-left"
        >
          {follower.effectiveAvatarUrl ? (
            <img
              src={follower.effectiveAvatarUrl}
              alt=""
              className="w-10 h-10 rounded-full object-cover"
            />
          ) : (
            <div className="w-10 h-10 rounded-full bg-slate-700 flex items-center justify-center">
              <svg className="w-5 h-5 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
              </svg>
            </div>
          )}
          <div className="flex-1 min-w-0">
            <div className="text-sm font-medium text-white truncate">
              {getDisplayName(follower)}
            </div>
            {follower.callSign && follower.displayName && (
              <div className="text-xs text-slate-400 truncate">
                {follower.displayName}
              </div>
            )}
          </div>
          <svg className="w-4 h-4 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
          </svg>
        </button>
      ))}
    </div>
  );
}

// News preview card (smaller than main feed)
function DashboardNewsCard({
  item,
  source,
  onClick,
}: {
  item: FeedItem;
  source?: SourceInfo;
  onClick: () => void;
}) {
  const imageUrl = item.media?.imageUrl;
  
  return (
    <button
      onClick={onClick}
      className="w-full bg-slate-800 border border-slate-700 rounded-xl p-3 hover:bg-slate-700/50 hover:border-slate-600 transition-colors text-left group"
    >
      <div className="flex gap-3">
        {imageUrl && (
          <img
            src={imageUrl}
            alt=""
            className="w-14 h-14 rounded-lg object-cover flex-shrink-0"
          />
        )}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            {source && (
              <span className="text-xs text-slate-500">{source.name}</span>
            )}
            {item.publishedAt && (
              <span className="text-xs text-slate-600">
                {new Date(item.publishedAt).toLocaleDateString()}
              </span>
            )}
          </div>
          <h4 className="text-sm font-medium text-white line-clamp-2 group-hover:text-primary-400 transition-colors">
            {item.title}
          </h4>
        </div>
      </div>
    </button>
  );
}

export function Dashboard({
  recentAircraft,
  recentNews,
  sources,
  isAircraftLoading,
  isNewsLoading,
  onViewAllNews,
  onViewAllAircraft,
  onViewAllGear,
  onSelectAircraft,
  onSelectNewsItem,
  onSelectPilot,
  onGoToSocial,
}: DashboardProps) {
  const { user, isAuthenticated } = useAuth();
  const [recentFollowers, setRecentFollowers] = useState<PilotSummary[]>([]);
  const [isFollowersLoading, setIsFollowersLoading] = useState(false);
  const [gearItems, setGearItems] = useState<InventoryItem[]>([]);
  const [isGearLoading, setIsGearLoading] = useState(false);
  const sourceMap = new Map(sources.map(s => [s.id, s]));

  // Load recent followers
  useEffect(() => {
    if (user?.id && isAuthenticated) {
      setIsFollowersLoading(true);
      getFollowers(user.id, 4, 0)
        .then(response => setRecentFollowers(response.pilots))
        .catch(() => setRecentFollowers([]))
        .finally(() => setIsFollowersLoading(false));
    }
  }, [user?.id, isAuthenticated]);

  // Load gear items
  useEffect(() => {
    if (isAuthenticated) {
      setIsGearLoading(true);
      getInventory({ limit: 3 })
        .then(response => setGearItems(response.items || []))
        .catch(() => setGearItems([]))
        .finally(() => setIsGearLoading(false));
    }
  }, [isAuthenticated]);

  return (
    <div className="flex-1 overflow-y-auto">
      <div className="p-4 md:p-6 max-w-6xl mx-auto">
        {/* Header */}
        <div className="mb-6 md:mb-8">
          <h1 className="text-2xl font-bold text-white mb-2">Dashboard</h1>
          <p className="text-slate-400">Welcome back! Here's your FlyingForge overview.</p>
        </div>

        {/* Main Grid */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 md:gap-6">
          {/* My Aircraft */}
          <section className="bg-slate-800/30 border border-slate-700/50 rounded-2xl p-4 md:p-5">
            <div className="flex items-center justify-between mb-3 md:mb-4">
              <h2 className="text-base md:text-lg font-semibold text-white flex items-center gap-2">
                <svg className="w-5 h-5 text-primary-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
                </svg>
                My Aircraft
              </h2>
              <button
                onClick={onViewAllAircraft}
                className="text-xs text-primary-400 hover:text-primary-300 transition-colors"
              >
                View All →
              </button>
            </div>
            <div className="space-y-3">
              {isAircraftLoading ? (
                <>
                  <SkeletonCard />
                  <SkeletonCard />
                </>
              ) : recentAircraft.length === 0 ? (
                <EmptyState
                  icon={
                    <svg className="w-6 h-6 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
                    </svg>
                  }
                  title="No Aircraft Yet"
                  description="Add your first drone to start tracking builds and settings"
                  actionLabel="View Aircraft"
                  onAction={onViewAllAircraft}
                />
              ) : (
                recentAircraft.slice(0, 3).map(aircraft => (
                  <DashboardAircraftCard
                    key={aircraft.id}
                    aircraft={aircraft}
                    onClick={() => onSelectAircraft(aircraft)}
                  />
                ))
              )}
            </div>
          </section>

          {/* My Inventory */}
          <section className="bg-slate-800/30 border border-slate-700/50 rounded-2xl p-4 md:p-5">
            <div className="flex items-center justify-between mb-3 md:mb-4">
              <h2 className="text-base md:text-lg font-semibold text-white flex items-center gap-2">
                <svg className="w-5 h-5 text-primary-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
                </svg>
                My Inventory
              </h2>
              <button
                onClick={onViewAllGear}
                className="text-xs text-primary-400 hover:text-primary-300 transition-colors"
              >
                View All →
              </button>
            </div>
            <div className="space-y-3">
              {isGearLoading ? (
                <>
                  <SkeletonCard />
                  <SkeletonCard />
                </>
              ) : gearItems.length === 0 ? (
                <EmptyState
                  icon={
                    <svg className="w-6 h-6 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
                    </svg>
                  }
                  title="No Gear Yet"
                  description="Add your equipment to track your inventory"
                  actionLabel="Add Gear"
                  onAction={onViewAllGear}
                />
              ) : (
                gearItems.map(item => (
                  <DashboardGearCard 
                    key={item.id} 
                    item={item}
                    onClick={onViewAllGear}
                  />
                ))
              )}
            </div>
          </section>

          {/* Recent Followers */}
          <section className="bg-slate-800/30 border border-slate-700/50 rounded-2xl p-4 md:p-5">
            <div className="flex items-center justify-between mb-3 md:mb-4">
              <h2 className="text-base md:text-lg font-semibold text-white flex items-center gap-2">
                <svg className="w-5 h-5 text-primary-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                </svg>
                Recent Followers
              </h2>
              {recentFollowers.length > 0 && (
                <button
                  onClick={onGoToSocial}
                  className="text-xs text-primary-400 hover:text-primary-300 transition-colors"
                >
                  View All →
                </button>
              )}
            </div>
            <RecentFollowers 
              followers={recentFollowers} 
              isLoading={isFollowersLoading}
              onSelectPilot={onSelectPilot}
              onViewAll={onGoToSocial}
            />
          </section>

          {/* Quick News Peek */}
          <section className="bg-slate-800/30 border border-slate-700/50 rounded-2xl p-4 md:p-5">
            <div className="flex items-center justify-between mb-3 md:mb-4">
              <h2 className="text-base md:text-lg font-semibold text-white flex items-center gap-2">
                <svg className="w-5 h-5 text-primary-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
                </svg>
                Latest News
              </h2>
              <button
                onClick={onViewAllNews}
                className="text-xs text-primary-400 hover:text-primary-300 transition-colors"
              >
                View All →
              </button>
            </div>
            <div className="space-y-2">
              {isNewsLoading ? (
                <>
                  <SkeletonCard />
                  <SkeletonCard />
                  <SkeletonCard />
                </>
              ) : recentNews.length === 0 ? (
                <div className="text-center py-4 text-slate-500 text-sm">
                  No news available
                </div>
              ) : (
                recentNews.slice(0, 4).map(item => (
                  <DashboardNewsCard
                    key={item.id}
                    item={item}
                    source={sourceMap.get(item.source)}
                    onClick={() => onSelectNewsItem(item)}
                  />
                ))
              )}
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}

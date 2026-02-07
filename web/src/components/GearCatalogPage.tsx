import { useState, useEffect, useCallback } from 'react';
import { searchGearCatalog, getPopularGear } from '../gearCatalogApi';
import type { GearCatalogItem, GearType } from '../gearCatalogTypes';
import { GEAR_TYPES, DRONE_TYPES, getCatalogItemDisplayName } from '../gearCatalogTypes';
import { useAuth } from '../hooks/useAuth';
import { GearDetailModal } from './GearDetailModal';

interface GearCatalogPageProps {
  onAddToInventory?: (item: GearCatalogItem) => void;
}

// Gear type tab component
function GearTypeTab({ 
  label, 
  isActive, 
  onClick 
}: { 
  label: string; 
  isActive: boolean; 
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className={`px-4 py-2 text-sm font-medium rounded-lg transition-colors whitespace-nowrap ${
        isActive
          ? 'bg-primary-600 text-white'
          : 'text-slate-400 hover:text-white hover:bg-slate-700'
      }`}
    >
      {label}
    </button>
  );
}

// Gear card for the catalog
function GearCard({ 
  item, 
  onAddToInventory,
  onOpenDetail,
  isAuthenticated,
}: { 
  item: GearCatalogItem; 
  onAddToInventory?: (item: GearCatalogItem) => void;
  onOpenDetail: (item: GearCatalogItem) => void;
  isAuthenticated: boolean;
}) {
  const typeLabel = GEAR_TYPES.find(t => t.value === item.gearType)?.label || item.gearType;

  const handleCardClick = () => {
    onOpenDetail(item);
  };

  const handleAddClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    onAddToInventory?.(item);
  };
  
  return (
    <div 
      className="bg-slate-800 border border-slate-700 rounded-xl p-4 hover:border-slate-600 transition-colors cursor-pointer"
      onClick={handleCardClick}
    >
      <div className="flex gap-4">
        {/* Image */}
        {item.imageUrl ? (
          <img
            src={item.imageUrl}
            alt={getCatalogItemDisplayName(item)}
            className="w-20 h-20 rounded-lg object-cover flex-shrink-0"
          />
        ) : (
          <div className="w-20 h-20 bg-slate-700 rounded-lg flex items-center justify-center flex-shrink-0">
            <svg className="w-8 h-8 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
            </svg>
          </div>
        )}

        {/* Content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-start justify-between gap-2">
            <div>
              <h3 className="text-white font-medium truncate">
                {getCatalogItemDisplayName(item)}
              </h3>
              <p className="text-sm text-slate-400">{item.brand}</p>
            </div>
            <span className="px-2 py-0.5 bg-slate-700 text-slate-300 text-xs rounded-full flex-shrink-0">
              {typeLabel}
            </span>
          </div>

          {/* Description */}
          {item.description && (
            <p className="text-sm text-slate-500 mt-2 line-clamp-2">
              {item.description}
            </p>
          )}

          {/* Best For badges */}
          {item.bestFor && item.bestFor.length > 0 && (
            <div className="flex flex-wrap gap-1 mt-2">
              {item.bestFor.map(droneType => {
                const label = DRONE_TYPES.find(t => t.value === droneType)?.label || droneType;
                return (
                  <span 
                    key={droneType}
                    className="px-2 py-0.5 bg-primary-600/20 text-primary-400 text-xs rounded-full"
                  >
                    {label}
                  </span>
                );
              })}
            </div>
          )}

          {/* Stats & Actions */}
          <div className="flex items-center justify-between mt-3">
            <div className="flex items-center gap-3 text-xs text-slate-500">
              <span className="flex items-center gap-1">
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                </svg>
                {item.usageCount} {item.usageCount === 1 ? 'pilot' : 'pilots'}
              </span>
            </div>

            {onAddToInventory && (
              <button
                onClick={handleAddClick}
                disabled={!isAuthenticated}
                title={isAuthenticated ? 'Add to your inventory' : 'Sign in to add to inventory'}
                className="px-3 py-1.5 bg-primary-600 hover:bg-primary-700 disabled:bg-slate-700 disabled:text-slate-500 disabled:cursor-not-allowed text-white text-sm font-medium rounded-lg transition-colors flex items-center gap-1"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                Add
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

export function GearCatalogPage({ onAddToInventory }: GearCatalogPageProps) {
  const { isAuthenticated } = useAuth();
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedType, setSelectedType] = useState<GearType | null>(null);
  const [items, setItems] = useState<GearCatalogItem[]>([]);
  const [popularItems, setPopularItems] = useState<GearCatalogItem[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isLoadingPopular, setIsLoadingPopular] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [totalCount, setTotalCount] = useState(0);
  const [hasSearched, setHasSearched] = useState(false);
  const [selectedItem, setSelectedItem] = useState<GearCatalogItem | null>(null);
  const [isDetailModalOpen, setIsDetailModalOpen] = useState(false);

  const handleOpenDetail = (item: GearCatalogItem) => {
    setSelectedItem(item);
    setIsDetailModalOpen(true);
  };

  const handleCloseDetail = () => {
    setIsDetailModalOpen(false);
    setSelectedItem(null);
  };

  // Load popular items on mount
  useEffect(() => {
    setIsLoadingPopular(true);
    getPopularGear(undefined, 12)
      .then(response => setPopularItems(response.items))
      .catch(() => setPopularItems([]))
      .finally(() => setIsLoadingPopular(false));
  }, []);

  // Search handler
  const handleSearch = useCallback(async () => {
    if (!searchQuery.trim() && !selectedType) {
      setHasSearched(false);
      setItems([]);
      return;
    }

    setIsLoading(true);
    setError(null);
    setHasSearched(true);

    try {
      const response = await searchGearCatalog({
        query: searchQuery.trim() || undefined,
        gearType: selectedType || undefined,
        limit: 50,
      });
      setItems(response.items);
      setTotalCount(response.totalCount);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search failed');
      setItems([]);
    } finally {
      setIsLoading(false);
    }
  }, [searchQuery, selectedType]);

  // Auto-search when type changes
  useEffect(() => {
    if (selectedType) {
      handleSearch();
    }
  }, [selectedType, handleSearch]);

  // Handle enter key in search
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSearch();
    }
  };

  // Clear search and show popular
  const handleClearSearch = () => {
    setSearchQuery('');
    setSelectedType(null);
    setHasSearched(false);
    setItems([]);
  };

  // Handle selecting "All Types" tab
  const handleSelectAllTypes = useCallback(async () => {
    setSelectedType(null);
    // If there's a search query, search with no type filter
    if (searchQuery.trim()) {
      setIsLoading(true);
      setError(null);
      setHasSearched(true);
      try {
        const response = await searchGearCatalog({
          query: searchQuery.trim(),
          gearType: undefined,
          limit: 50,
        });
        setItems(response.items);
        setTotalCount(response.totalCount);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Search failed');
        setItems([]);
      } finally {
        setIsLoading(false);
      }
    } else {
      // No search query, reset to show popular items
      setHasSearched(false);
      setItems([]);
    }
  }, [searchQuery]);

  const displayItems = hasSearched ? items : popularItems;
  const showingPopular = !hasSearched;

  return (
    <div className="flex-1 overflow-y-auto">
      {/* Header */}
      <div className="sticky top-0 z-10 bg-slate-900/95 backdrop-blur-sm border-b border-slate-800">
        <div className="px-6 py-4">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h1 className="text-xl font-semibold text-white">Gear Catalog</h1>
              <p className="text-sm text-slate-400">
                Browse community-contributed FPV gear • Like PCPartPicker for drones
              </p>
            </div>
            {!isAuthenticated && (
              <div className="flex items-center gap-2 text-sm text-slate-400">
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                Sign in to add gear to your inventory
              </div>
            )}
          </div>

          {/* Search bar */}
          <div className="flex gap-3">
            <div className="flex-1 relative">
              <svg
                className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400"
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
              <input
                type="text"
                placeholder="Search by brand, model, or keyword..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onKeyDown={handleKeyDown}
                className="w-full pl-10 pr-4 py-2.5 bg-slate-800 border border-slate-700 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500"
              />
            </div>
            <button
              onClick={handleSearch}
              className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white font-medium rounded-lg transition-colors"
            >
              Search
            </button>
            {hasSearched && (
              <button
                onClick={handleClearSearch}
                className="px-4 py-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
              >
                Clear
              </button>
            )}
          </div>

          {/* Type filters */}
          <div className="mt-4 flex gap-2 overflow-x-auto pb-2 scrollbar-thin scrollbar-thumb-slate-700">
            <GearTypeTab
              label="All Types"
              isActive={selectedType === null}
              onClick={handleSelectAllTypes}
            />
            {GEAR_TYPES.map(type => (
              <GearTypeTab
                key={type.value}
                label={type.label}
                isActive={selectedType === type.value}
                onClick={() => setSelectedType(type.value)}
              />
            ))}
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="p-6">
        {/* Section header */}
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-medium text-white">
            {showingPopular ? (
              <>
                <span className="text-primary-400">Popular Gear</span>
                <span className="text-slate-400 font-normal ml-2 text-sm">
                  Browse what other pilots are using
                </span>
              </>
            ) : (
              <>
                Search Results
                <span className="text-slate-400 font-normal ml-2 text-sm">
                  {totalCount} {totalCount === 1 ? 'item' : 'items'} found
                </span>
              </>
            )}
          </h2>
        </div>

        {/* Error state */}
        {error && (
          <div className="mb-4 p-4 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400">
            {error}
          </div>
        )}

        {/* Loading state */}
        {(isLoading || isLoadingPopular) && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {[1, 2, 3, 4, 5, 6].map(i => (
              <div key={i} className="bg-slate-800 border border-slate-700 rounded-xl p-4 animate-pulse">
                <div className="flex gap-4">
                  <div className="w-20 h-20 bg-slate-700 rounded-lg" />
                  <div className="flex-1 space-y-3">
                    <div className="h-5 bg-slate-700 rounded w-3/4" />
                    <div className="h-4 bg-slate-700 rounded w-1/2" />
                    <div className="h-4 bg-slate-700 rounded w-1/4" />
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Empty state */}
        {!isLoading && !isLoadingPopular && displayItems.length === 0 && (
          <div className="text-center py-12">
            <div className="w-16 h-16 mx-auto mb-4 rounded-full bg-slate-800 flex items-center justify-center">
              <svg className="w-8 h-8 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
              </svg>
            </div>
            <h3 className="text-lg font-medium text-white mb-2">
              {hasSearched ? 'No gear found' : 'No popular gear yet'}
            </h3>
            <p className="text-slate-400 max-w-md mx-auto">
              {hasSearched
                ? 'Try adjusting your search terms or filters'
                : 'Be the first to contribute to the gear catalog!'}
            </p>
          </div>
        )}

        {/* Results grid */}
        {!isLoading && !isLoadingPopular && displayItems.length > 0 && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {displayItems.map(item => (
              <GearCard
                key={item.id}
                item={item}
                onAddToInventory={onAddToInventory}
                onOpenDetail={handleOpenDetail}
                isAuthenticated={isAuthenticated}
              />
            ))}
          </div>
        )}

        {/* Community contribution note */}
        <div className="mt-8 p-4 bg-slate-800/50 border border-slate-700 rounded-xl">
          <div className="flex gap-3">
            <div className="w-10 h-10 rounded-full bg-primary-600/20 flex items-center justify-center flex-shrink-0">
              <svg className="w-5 h-5 text-primary-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
              </svg>
            </div>
            <div>
              <h3 className="text-white font-medium">Contribute to the Catalog</h3>
              <p className="text-sm text-slate-400 mt-1">
                Don't see your gear? When you add items to your inventory, they're automatically 
                added to the community catalog for others to find. Help grow the database!
              </p>
            </div>
          </div>
        </div>
      </div>

      {/* Gear Detail Modal */}
      {selectedItem && (
        <GearDetailModal
          item={selectedItem}
          isOpen={isDetailModalOpen}
          onClose={handleCloseDetail}
          onAddToInventory={onAddToInventory}
          isAuthenticated={isAuthenticated}
        />
      )}
    </div>
  );
}

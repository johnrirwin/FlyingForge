import type { FeedItem, SourceInfo } from '../types';
import { useInfiniteScroll } from '../hooks';

interface FeedListProps {
  items: FeedItem[];
  sources: SourceInfo[];
  isLoading: boolean;
  error: string | null;
  onItemClick: (item: FeedItem) => void;
  hasMore?: boolean;
  onLoadMore?: () => void;
}

export function FeedList({ items, sources, isLoading, error, onItemClick, hasMore = false, onLoadMore }: FeedListProps) {
  const sourceMap = new Map(sources.map(s => [s.id, s]));

  const { setLoadMoreRef } = useInfiniteScroll(
    () => onLoadMore?.(),
    { hasMore, isLoading }
  );

  if (error) {
    return (
      <div className="flex-1 flex items-center justify-center p-8">
        <div className="text-center max-w-md">
          <div className="w-16 h-16 bg-red-500/20 rounded-full flex items-center justify-center mx-auto mb-4">
            <svg className="w-8 h-8 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <h3 className="text-lg font-medium text-white mb-2">Failed to Load Feed</h3>
          <p className="text-slate-400 text-sm">{error}</p>
        </div>
      </div>
    );
  }

  if (isLoading && items.length === 0) {
    return (
      <div className="flex-1 p-6 space-y-4">
        {[...Array(6)].map((_, i) => (
          <div key={i} className="bg-slate-800 border border-slate-700 rounded-xl p-4 animate-pulse">
            <div className="flex gap-4">
              <div className="w-24 h-24 bg-slate-700 rounded-lg" />
              <div className="flex-1 space-y-3">
                <div className="flex gap-2">
                  <div className="w-20 h-5 bg-slate-700 rounded" />
                  <div className="w-24 h-5 bg-slate-700 rounded" />
                </div>
                <div className="h-6 bg-slate-700 rounded w-3/4" />
                <div className="h-4 bg-slate-700 rounded w-full" />
                <div className="h-4 bg-slate-700 rounded w-2/3" />
              </div>
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center p-8">
        <div className="text-center max-w-md">
          <div className="w-16 h-16 bg-slate-700 rounded-full flex items-center justify-center mx-auto mb-4">
            <svg className="w-8 h-8 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
            </svg>
          </div>
          <h3 className="text-lg font-medium text-white mb-2">No Items Found</h3>
          <p className="text-slate-400 text-sm">
            Try adjusting your filters or search query, or refresh to fetch new items.
          </p>
        </div>
      </div>
    );
  }

  // Import FeedCard dynamically to avoid circular imports
  return (
    <div className="flex-1 p-4 md:p-6">
      <div className="space-y-3 md:space-y-4 max-w-4xl mx-auto">
        {items.map(item => (
          <FeedCardWrapper
            key={item.id}
            item={item}
            source={sourceMap.get(item.source)}
            onClick={() => onItemClick(item)}
          />
        ))}
        
        {/* Infinite scroll trigger */}
        {hasMore && (
          <div ref={setLoadMoreRef} className="h-4" />
        )}
        
        {/* Loading indicator for infinite scroll */}
        {isLoading && items.length > 0 && (
          <div className="flex items-center justify-center py-8">
            <div className="w-8 h-8 border-2 border-primary-500 border-t-transparent rounded-full animate-spin" />
          </div>
        )}
        
        {/* End of list indicator */}
        {!hasMore && items.length > 0 && (
          <div className="text-center py-6 text-slate-500 text-sm">
            You've reached the end • {items.length} items loaded
          </div>
        )}
      </div>
    </div>
  );
}

// Wrapper to use FeedCard
import { FeedCard } from './FeedCard';

function FeedCardWrapper({ item, source, onClick }: { item: FeedItem; source?: SourceInfo; onClick: () => void }) {
  return <FeedCard item={item} source={source} onClick={onClick} />;
}

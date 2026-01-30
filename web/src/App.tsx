import { useState, useEffect, useCallback } from 'react';
import { Sidebar, TopBar, FeedList, ItemDetail } from './components';
import { getItems, getSources, refreshFeeds } from './api';
import { useFilters, useDebounce } from './hooks';
import type { FeedItem, SourceInfo, SourceType, FilterParams } from './types';

function App() {
  // State
  const [items, setItems] = useState<FeedItem[]>([]);
  const [sources, setSources] = useState<SourceInfo[]>([]);
  const [selectedItem, setSelectedItem] = useState<FeedItem | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [totalCount, setTotalCount] = useState(0);

  // Filters
  const { filters, updateFilter, toggleSource } = useFilters();
  const debouncedQuery = useDebounce(filters.query, 300);

  // Load sources on mount
  useEffect(() => {
    getSources()
      .then(response => {
        setSources(response.sources);
      })
      .catch(err => {
        console.error('Failed to load sources:', err);
      });
  }, []);

  // Load items when filters change
  useEffect(() => {
    const loadItems = async () => {
      setIsLoading(true);
      setError(null);

      try {
        const params: FilterParams = {
          limit: 50,
          sort: filters.sort,
        };

        if (filters.sources.length > 0) {
          params.sources = filters.sources;
        }

        if (filters.sourceType !== 'all') {
          params.sourceType = filters.sourceType;
        }

        if (debouncedQuery) {
          params.query = debouncedQuery;
        }

        if (filters.fromDate) {
          params.fromDate = filters.fromDate;
        }

        if (filters.toDate) {
          params.toDate = filters.toDate;
        }

        const response = await getItems(params);
        setItems(response.items || []);
        setTotalCount(response.totalCount || 0);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load items');
        setItems([]);
      } finally {
        setIsLoading(false);
      }
    };

    loadItems();
  }, [
    filters.sources,
    filters.sourceType,
    filters.sort,
    filters.fromDate,
    filters.toDate,
    debouncedQuery,
  ]);

  // Refresh handler
  const handleRefresh = useCallback(async () => {
    setIsRefreshing(true);
    setError(null);

    try {
      // First, trigger the backend to refresh feeds from sources
      await refreshFeeds(
        filters.sources.length > 0 ? filters.sources : undefined
      );
      
      // Then re-fetch items with current filters
      const params: FilterParams = {
        limit: 50,
        sort: filters.sort,
      };

      if (filters.sources.length > 0) {
        params.sources = filters.sources;
      }

      if (filters.sourceType !== 'all') {
        params.sourceType = filters.sourceType;
      }

      if (debouncedQuery) {
        params.query = debouncedQuery;
      }

      const response = await getItems(params);
      setItems(response.items || []);
      setTotalCount(response.totalCount || 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to refresh feeds');
    } finally {
      setIsRefreshing(false);
    }
  }, [filters.sources, filters.sourceType, filters.sort, debouncedQuery]);

  // Handle source type change
  const handleSourceTypeChange = useCallback((type: SourceType | 'all') => {
    updateFilter('sourceType', type);
    // Clear selected sources when changing type
    if (type !== 'all' && filters.sources.length > 0) {
      const validSources = sources
        .filter(s => s.sourceType === type)
        .map(s => s.id);
      const newSelected = filters.sources.filter(id => validSources.includes(id));
      if (newSelected.length !== filters.sources.length) {
        updateFilter('sources', []);
      }
    }
  }, [updateFilter, filters.sources, sources]);

  // Handle keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && selectedItem) {
        setSelectedItem(null);
      }
      if (e.key === '/' && !selectedItem) {
        e.preventDefault();
        const searchInput = document.querySelector('input[type="text"]') as HTMLInputElement;
        searchInput?.focus();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [selectedItem]);

  const sourceMap = new Map(sources.map(s => [s.id, s]));

  return (
    <div className="flex h-screen bg-slate-900 text-white overflow-hidden">
      {/* Sidebar */}
      <Sidebar
        sources={sources}
        selectedSources={filters.sources}
        sourceType={filters.sourceType}
        onToggleSource={toggleSource}
        onSourceTypeChange={handleSourceTypeChange}
        isLoading={sources.length === 0}
      />

      {/* Main Content */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Top Bar */}
        <TopBar
          query={filters.query}
          onQueryChange={q => updateFilter('query', q)}
          fromDate={filters.fromDate}
          toDate={filters.toDate}
          onFromDateChange={d => updateFilter('fromDate', d)}
          onToDateChange={d => updateFilter('toDate', d)}
          sort={filters.sort}
          onSortChange={s => updateFilter('sort', s)}
          onRefresh={handleRefresh}
          isRefreshing={isRefreshing}
          totalCount={totalCount}
        />

        {/* Feed List */}
        <FeedList
          items={items}
          sources={sources}
          isLoading={isLoading}
          error={error}
          onItemClick={setSelectedItem}
        />
      </div>

      {/* Item Detail Modal */}
      {selectedItem && (
        <ItemDetail
          item={selectedItem}
          source={sourceMap.get(selectedItem.source)}
          onClose={() => setSelectedItem(null)}
        />
      )}
    </div>
  );
}

export default App;

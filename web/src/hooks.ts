import { useState, useEffect, useCallback, useRef } from 'react';
import type { FiltersState } from './types';

const STORAGE_KEY = 'flyingforge-filters';

const defaultFilters: FiltersState = {
  sources: [],
  sourceType: 'all',
  query: '',
  sort: 'newest',
  fromDate: '',
  toDate: '',
};

export function useFilters() {
  const [filters, setFilters] = useState<FiltersState>(() => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored) {
        return { ...defaultFilters, ...JSON.parse(stored) };
      }
    } catch {
      // Ignore parse errors
    }
    return defaultFilters;
  });

  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(filters));
    } catch {
      // Ignore storage errors
    }
  }, [filters]);

  const updateFilter = useCallback(<K extends keyof FiltersState>(
    key: K,
    value: FiltersState[K]
  ) => {
    setFilters(prev => ({ ...prev, [key]: value }));
  }, []);

  const toggleSource = useCallback((sourceId: string) => {
    setFilters(prev => {
      const sources = prev.sources.includes(sourceId)
        ? prev.sources.filter(s => s !== sourceId)
        : [...prev.sources, sourceId];
      return { ...prev, sources };
    });
  }, []);

  const resetFilters = useCallback(() => {
    setFilters(defaultFilters);
  }, []);

  return {
    filters,
    setFilters,
    updateFilter,
    toggleSource,
    resetFilters,
  };
}

export function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState(value);

  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    return () => {
      clearTimeout(timer);
    };
  }, [value, delay]);

  return debouncedValue;
}

export function useInfiniteScroll(
  onLoadMore: () => void,
  options: {
    hasMore: boolean;
    isLoading: boolean;
    threshold?: number;
  }
) {
  const { hasMore, isLoading, threshold = 200 } = options;
  const observerRef = useRef<IntersectionObserver | null>(null);
  const loadMoreRef = useRef<HTMLDivElement | null>(null);

  const setLoadMoreRef = useCallback((node: HTMLDivElement | null) => {
    // Cleanup previous observer
    if (observerRef.current) {
      observerRef.current.disconnect();
    }

    if (!node || !hasMore || isLoading) {
      loadMoreRef.current = null;
      return;
    }

    loadMoreRef.current = node;

    observerRef.current = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting && hasMore && !isLoading) {
          onLoadMore();
        }
      },
      {
        rootMargin: `${threshold}px`,
      }
    );

    observerRef.current.observe(node);
  }, [hasMore, isLoading, onLoadMore, threshold]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (observerRef.current) {
        observerRef.current.disconnect();
      }
    };
  }, []);

  return { setLoadMoreRef };
}

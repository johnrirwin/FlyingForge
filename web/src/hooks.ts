import { useState, useEffect, useCallback } from 'react';
import type { FiltersState } from './types';

const STORAGE_KEY = 'drone-news-filters';

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

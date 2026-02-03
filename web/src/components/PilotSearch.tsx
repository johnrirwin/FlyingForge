import { useState, useCallback, useEffect } from 'react';
import { searchPilots } from '../pilotApi';
import type { PilotSearchResult } from '../socialTypes';
import { useDebounce } from '../hooks';

interface PilotSearchProps {
  onSelectPilot: (pilotId: string) => void;
}

export function PilotSearch({ onSelectPilot }: PilotSearchProps) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<PilotSearchResult[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hasSearched, setHasSearched] = useState(false);

  const debouncedQuery = useDebounce(query, 300);

  // Search when debounced query changes
  const handleSearch = useCallback(async (searchQuery: string) => {
    if (searchQuery.length < 2) {
      setResults([]);
      setHasSearched(false);
      return;
    }

    try {
      setIsLoading(true);
      setError(null);
      const response = await searchPilots(searchQuery);
      setResults(response.pilots);
      setHasSearched(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search failed');
      setResults([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Effect to trigger search on debounced query change
  useEffect(() => {
    if (debouncedQuery) {
      handleSearch(debouncedQuery);
    } else {
      setResults([]);
      setHasSearched(false);
    }
  }, [debouncedQuery, handleSearch]);

  // Manual search handler
  const handleSearchClick = () => {
    if (query.length >= 2) {
      handleSearch(query);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSearchClick();
    }
  };

  const getDisplayName = (pilot: PilotSearchResult) => {
    return pilot.callSign || 'Unknown Pilot';
  };

  return (
    <div className="max-w-4xl mx-auto p-6">
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-white mb-2">Pilot Directory</h1>
        <p className="text-slate-400">
          Search for other pilots by their callsign to view their profiles and aircraft.
        </p>
      </div>

      {/* Search Input */}
      <div className="mb-6">
        <div className="relative">
          <input
            type="text"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Search by callsign..."
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
          {isLoading && (
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
      {error && (
        <div className="mb-4 p-4 bg-red-500/10 border border-red-500/50 rounded-lg text-red-400">
          {error}
        </div>
      )}

      {/* Results */}
      {hasSearched && results.length === 0 && !isLoading && (
        <div className="text-center py-12 text-slate-500">
          <svg
            className="w-16 h-16 mx-auto mb-4 text-slate-600"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"
            />
          </svg>
          <p className="text-lg">No pilots found</p>
          <p className="text-sm mt-1">Try a different search term</p>
        </div>
      )}

      {results.length > 0 && (
        <div className="space-y-2">
          <p className="text-sm text-slate-400 mb-4">
            Found {results.length} pilot{results.length !== 1 ? 's' : ''}
          </p>
          {results.map((pilot) => (
            <button
              key={pilot.id}
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
                {pilot.displayName && (
                  <div className="text-sm text-slate-400 truncate">
                    {pilot.displayName}
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
          ))}
        </div>
      )}

      {/* Initial state */}
      {!hasSearched && !isLoading && (
        <div className="text-center py-12 text-slate-500">
          <svg
            className="w-16 h-16 mx-auto mb-4 text-slate-600"
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
          <p className="text-lg">Search for pilots</p>
          <p className="text-sm mt-1">Find other FPV enthusiasts by their callsign</p>
        </div>
      )}
    </div>
  );
}

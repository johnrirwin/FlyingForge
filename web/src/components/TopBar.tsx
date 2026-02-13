import { useRef } from 'react';
import type { SourceType } from '../types';

interface TopBarProps {
  query: string;
  onQueryChange: (query: string) => void;
  onSearch: () => void;
  fromDate: string;
  toDate: string;
  onFromDateChange: (date: string) => void;
  onToDateChange: (date: string) => void;
  sort: 'newest' | 'score';
  onSortChange: (sort: 'newest' | 'score') => void;
  sourceType: SourceType | 'all';
  onSourceTypeChange: (type: SourceType | 'all') => void;
  totalCount: number;
  isCollapsed?: boolean; // External control for mobile collapse
}

export function TopBar({
  query,
  onQueryChange,
  onSearch,
  fromDate,
  toDate,
  onFromDateChange,
  onToDateChange,
  sort,
  onSortChange,
  sourceType,
  onSourceTypeChange,
  totalCount,
  isCollapsed = false,
}: TopBarProps) {
  const headerRef = useRef<HTMLElement>(null);

  // Show filters when not collapsed (at top of scroll)
  const showFilters = !isCollapsed;

  // Handle enter key in search
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      onSearch();
    }
  };

  // Handle clearing search
  const handleClearSearch = () => {
    onQueryChange('');
    onSearch();
  };

  return (
    <header ref={headerRef} className="bg-slate-800 border-b border-slate-700 px-4 md:px-6 py-3 md:py-4">
      <div className="flex flex-col gap-3 md:flex-row md:items-center md:gap-4">
        {/* Search - full width on mobile */}
        <div className="flex gap-2 flex-1 min-w-0 md:min-w-[200px] md:max-w-md">
          <div className="relative flex-1">
            <svg
              className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400"
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
              placeholder="Search news..."
              value={query}
              onChange={e => onQueryChange(e.target.value)}
              onKeyDown={handleKeyDown}
              className="w-full pl-10 pr-4 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
            />
          </div>
          <button
            onClick={onSearch}
            className="px-3 py-2 bg-primary-600 hover:bg-primary-700 text-white text-sm font-medium rounded-lg transition-colors"
          >
            Search
          </button>
          {query && (
            <button
              onClick={handleClearSearch}
              className="px-3 py-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
            >
              Clear
            </button>
          )}
        </div>

        {/* Filters row - collapsible on mobile, always visible on desktop */}
        <div className={`flex flex-wrap items-center gap-2 md:gap-4 transition-all duration-200 ease-in-out md:!max-h-none md:!opacity-100 md:!overflow-visible ${
          showFilters 
            ? 'max-h-[200px] opacity-100' 
            : 'max-h-0 opacity-0 overflow-hidden md:max-h-none md:opacity-100'
        }`}>
          {/* Date Range - collapsible on mobile */}
          <div className="flex items-center gap-2 flex-wrap">
            <label className="text-xs md:text-sm text-slate-400">From:</label>
            <input
              type="date"
              value={fromDate}
              onChange={e => onFromDateChange(e.target.value)}
              className="px-2 md:px-3 py-1.5 md:py-2 bg-slate-900 border border-slate-700 rounded-lg text-white text-xs md:text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
            <label className="text-xs md:text-sm text-slate-400">To:</label>
            <input
              type="date"
              value={toDate}
              onChange={e => onToDateChange(e.target.value)}
              className="px-2 md:px-3 py-1.5 md:py-2 bg-slate-900 border border-slate-700 rounded-lg text-white text-xs md:text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
          {(fromDate || toDate) && (
            <button
              onClick={() => {
                onFromDateChange('');
                onToDateChange('');
              }}
              className="text-slate-400 hover:text-white"
              title="Clear dates"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          )}
        </div>

        {/* Source Type Filter */}
        <div className="flex items-center gap-2">
          <span className="text-xs md:text-sm text-slate-400 hidden sm:inline">Type:</span>
          <div className="flex gap-1 bg-slate-900 rounded-lg p-1">
            <button
              onClick={() => onSourceTypeChange('all')}
              className={`px-2 md:px-3 py-1 md:py-1.5 text-xs md:text-sm rounded-md transition-colors ${
                sourceType === 'all'
                  ? 'bg-primary-600 text-white'
                  : 'text-slate-400 hover:text-white'
              }`}
            >
              All
            </button>
            <button
              onClick={() => onSourceTypeChange('rss')}
              className={`px-2 md:px-3 py-1 md:py-1.5 text-xs md:text-sm rounded-md transition-colors ${
                sourceType === 'rss'
                  ? 'bg-primary-600 text-white'
                  : 'text-slate-400 hover:text-white'
              }`}
            >
              News
            </button>
            <button
              onClick={() => onSourceTypeChange('youtube')}
              className={`px-2 md:px-3 py-1 md:py-1.5 text-xs md:text-sm rounded-md transition-colors ${
                sourceType === 'youtube'
                  ? 'bg-primary-600 text-white'
                  : 'text-slate-400 hover:text-white'
              }`}
            >
              YouTube
            </button>
          </div>
        </div>

        {/* Sort */}
        <div className="flex items-center gap-2">
          <span className="text-xs md:text-sm text-slate-400 hidden sm:inline">Sort:</span>
          <div className="flex gap-1 bg-slate-900 rounded-lg p-1">
            <button
              onClick={() => onSortChange('newest')}
              className={`px-2 md:px-3 py-1 md:py-1.5 text-xs md:text-sm rounded-md transition-colors ${
                sort === 'newest'
                  ? 'bg-primary-600 text-white'
                  : 'text-slate-400 hover:text-white'
              }`}
            >
              Newest
            </button>
            <button
              onClick={() => onSortChange('score')}
              className={`px-2 md:px-3 py-1 md:py-1.5 text-xs md:text-sm rounded-md transition-colors ${
                sort === 'score'
                  ? 'bg-primary-600 text-white'
                  : 'text-slate-400 hover:text-white'
              }`}
            >
              Top
            </button>
          </div>
        </div>

        {/* Count */}
        <div className="text-xs md:text-sm text-slate-400 ml-auto md:ml-0">
          {totalCount.toLocaleString()} items
        </div>
        </div>
      </div>
    </header>
  );
}

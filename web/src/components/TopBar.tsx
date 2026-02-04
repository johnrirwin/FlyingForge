interface TopBarProps {
  query: string;
  onQueryChange: (query: string) => void;
  fromDate: string;
  toDate: string;
  onFromDateChange: (date: string) => void;
  onToDateChange: (date: string) => void;
  sort: 'newest' | 'score';
  onSortChange: (sort: 'newest' | 'score') => void;
  onRefresh: () => void;
  isRefreshing: boolean;
  refreshCooldown: number; // seconds remaining
  totalCount: number;
}

export function TopBar({
  query,
  onQueryChange,
  fromDate,
  toDate,
  onFromDateChange,
  onToDateChange,
  sort,
  onSortChange,
  onRefresh,
  isRefreshing,
  refreshCooldown,
  totalCount,
}: TopBarProps) {
  // Format cooldown time as M:SS
  const formatCooldown = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const isRefreshDisabled = isRefreshing || refreshCooldown > 0;

  const getRefreshButtonText = () => {
    if (isRefreshing) return 'Refreshing...';
    if (refreshCooldown > 0) return `Refresh (${formatCooldown(refreshCooldown)})`;
    return 'Refresh';
  };

  return (
    <header className="bg-slate-800 border-b border-slate-700 px-6 py-4">
      <div className="flex items-center gap-4 flex-wrap">
        {/* Search */}
        <div className="flex-1 min-w-[200px] max-w-md">
          <div className="relative">
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
              className="w-full pl-10 pr-4 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
            />
            {query && (
              <button
                onClick={() => onQueryChange('')}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-white"
                aria-label="Clear search"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            )}
          </div>
        </div>

        {/* Date Range */}
        <div className="flex items-center gap-2">
          <label className="text-sm text-slate-400">From:</label>
          <input
            type="date"
            value={fromDate}
            onChange={e => onFromDateChange(e.target.value)}
            className="px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
          />
          <label className="text-sm text-slate-400">To:</label>
          <input
            type="date"
            value={toDate}
            onChange={e => onToDateChange(e.target.value)}
            className="px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
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

        {/* Sort */}
        <div className="flex items-center gap-2">
          <span className="text-sm text-slate-400">Sort:</span>
          <div className="flex gap-1 bg-slate-900 rounded-lg p-1">
            <button
              onClick={() => onSortChange('newest')}
              className={`px-3 py-1.5 text-sm rounded-md transition-colors ${
                sort === 'newest'
                  ? 'bg-primary-600 text-white'
                  : 'text-slate-400 hover:text-white'
              }`}
            >
              Newest
            </button>
            <button
              onClick={() => onSortChange('score')}
              className={`px-3 py-1.5 text-sm rounded-md transition-colors ${
                sort === 'score'
                  ? 'bg-primary-600 text-white'
                  : 'text-slate-400 hover:text-white'
              }`}
            >
              Top Score
            </button>
          </div>
        </div>

        {/* Refresh */}
        <button
          onClick={onRefresh}
          disabled={isRefreshDisabled}
          className="flex items-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 disabled:bg-primary-800 disabled:cursor-not-allowed text-white rounded-lg transition-colors"
        >
          <svg
            className={`w-4 h-4 ${isRefreshing ? 'animate-spin' : ''}`}
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
            />
          </svg>
          {getRefreshButtonText()}
        </button>

        {/* Count */}
        <div className="text-sm text-slate-400">
          {totalCount.toLocaleString()} items
        </div>
      </div>
    </header>
  );
}

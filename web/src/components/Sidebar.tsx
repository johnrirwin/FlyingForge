import type { SourceInfo, SourceType as ST } from '../types';

interface SidebarProps {
  sources: SourceInfo[];
  selectedSources: string[];
  sourceType: ST | 'all';
  onToggleSource: (sourceId: string) => void;
  onSourceTypeChange: (type: ST | 'all') => void;
  isLoading: boolean;
}

export function Sidebar({
  sources,
  selectedSources,
  sourceType,
  onToggleSource,
  onSourceTypeChange,
  isLoading,
}: SidebarProps) {
  const newsSources = sources.filter(s => s.sourceType === 'news');
  const communitySources = sources.filter(s => s.sourceType === 'community');

  const filteredSources = sourceType === 'all' 
    ? sources 
    : sources.filter(s => s.sourceType === sourceType);

  return (
    <aside className="w-64 bg-slate-800 border-r border-slate-700 flex flex-col h-full overflow-hidden">
      <div className="p-4 border-b border-slate-700">
        <h2 className="text-lg font-semibold text-white flex items-center gap-2">
          <svg className="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="12" cy="12" r="3"/>
            <path d="M12 1v4M12 19v4M1 12h4M19 12h4"/>
            <path d="M4.22 4.22l2.83 2.83M16.95 16.95l2.83 2.83"/>
            <path d="M4.22 19.78l2.83-2.83M16.95 7.05l2.83-2.83"/>
          </svg>
          Drone News
        </h2>
      </div>

      {/* Source Type Toggle */}
      <div className="p-4 border-b border-slate-700">
        <div className="flex gap-1 bg-slate-900 rounded-lg p-1">
          {(['all', 'news', 'community'] as const).map(type => (
            <button
              key={type}
              onClick={() => onSourceTypeChange(type)}
              className={`flex-1 px-3 py-1.5 text-sm rounded-md transition-colors ${
                sourceType === type
                  ? 'bg-primary-600 text-white'
                  : 'text-slate-400 hover:text-white hover:bg-slate-700'
              }`}
            >
              {type === 'all' ? 'All' : type === 'news' ? 'News' : 'Community'}
            </button>
          ))}
        </div>
        <div className="mt-2 text-xs text-slate-500">
          {newsSources.length} news • {communitySources.length} community
        </div>
      </div>

      {/* Sources List */}
      <div className="flex-1 overflow-y-auto p-4">
        <div className="space-y-1">
          {isLoading ? (
            <div className="space-y-2">
              {[...Array(8)].map((_, i) => (
                <div key={i} className="h-8 bg-slate-700 rounded animate-pulse" />
              ))}
            </div>
          ) : (
            filteredSources.map(source => (
              <button
                key={source.id}
                onClick={() => onToggleSource(source.id)}
                className={`w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm transition-colors text-left ${
                  selectedSources.includes(source.id)
                    ? 'bg-primary-600/20 text-primary-400 border border-primary-500/30'
                    : selectedSources.length === 0
                    ? 'text-slate-300 hover:bg-slate-700'
                    : 'text-slate-500 hover:bg-slate-700 hover:text-slate-300'
                }`}
                title={source.description}
              >
                <span className={`w-2 h-2 rounded-full flex-shrink-0 ${
                  source.sourceType === 'news' ? 'bg-blue-500' : 'bg-green-500'
                }`} />
                <span className="truncate">{source.name}</span>
                <span className="ml-auto text-xs text-slate-500 flex-shrink-0">
                  {source.feedType}
                </span>
              </button>
            ))
          )}
        </div>

        {selectedSources.length > 0 && (
          <button
            onClick={() => selectedSources.forEach(onToggleSource)}
            className="mt-4 w-full text-sm text-slate-500 hover:text-slate-300 transition-colors"
          >
            Clear selection ({selectedSources.length})
          </button>
        )}
      </div>

      {/* Footer */}
      <div className="p-4 border-t border-slate-700 text-xs text-slate-500">
        <div className="flex items-center justify-between">
          <span>MCP News Feed</span>
          <span className="flex items-center gap-1">
            <span className="w-2 h-2 rounded-full bg-green-500" />
            Connected
          </span>
        </div>
      </div>
    </aside>
  );
}

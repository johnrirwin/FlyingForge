import type { FeedItem, SourceInfo } from '../types';
import { stripHtmlToText } from '../textUtils';

interface ItemDetailProps {
  item: FeedItem;
  source?: SourceInfo;
  onClose: () => void;
}

export function ItemDetail({ item, source, onClose }: ItemDetailProps) {
  const publishedAt = item.publishedAt ? new Date(item.publishedAt) : null;
  const summaryText = stripHtmlToText(item.summary);
  const contentText = stripHtmlToText(item.contentText);
  const isVideoItem =
    item.media?.type?.toLowerCase() === 'video' ||
    Boolean(item.media?.videoUrl) ||
    item.sourceType === 'youtube' ||
    item.tags.some(tag => tag.toLowerCase() === 'video');

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm">
      <div
        className="absolute inset-0"
        onClick={onClose}
        aria-hidden
      />
      
      <div className="relative w-full max-w-2xl max-h-[90vh] bg-slate-800 border border-slate-700 rounded-2xl shadow-xl overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-slate-700">
          <div className="flex items-center gap-2">
            <span
              className={`px-2 py-0.5 rounded-full text-xs font-medium ${
                item.sourceType === 'youtube'
                  ? 'bg-red-500/20 text-red-400'
                  : item.sourceType === 'rss'
                  ? 'bg-blue-500/20 text-blue-400'
                  : 'bg-green-500/20 text-green-400'
              }`}
            >
              {source?.name || item.source}
            </span>
            {item.author && (
              <span className="text-sm text-slate-400">by {item.author}</span>
            )}
          </div>
          <button
            onClick={onClose}
            className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-6">
          {/* Image */}
          {item.media?.imageUrl && (
            <div className="mb-6 rounded-lg overflow-hidden bg-slate-700">
              <img
                src={item.media.imageUrl}
                alt=""
                className="w-full max-h-64 object-cover"
                onError={e => {
                  (e.target as HTMLImageElement).style.display = 'none';
                }}
              />
            </div>
          )}

          {/* Title */}
          <h2 className="text-xl font-semibold text-white mb-4 leading-tight">
            {item.title}
          </h2>

          {/* Meta */}
          <div className="flex items-center gap-4 mb-6 text-sm text-slate-400">
            {publishedAt && (
              <span>
                {publishedAt.toLocaleDateString('en-US', {
                  year: 'numeric',
                  month: 'long',
                  day: 'numeric',
                  hour: '2-digit',
                  minute: '2-digit',
                })}
              </span>
            )}
            {item.score !== undefined && item.score !== null && (
              <span className="flex items-center gap-1 text-orange-400">
                <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M2 10.5a1.5 1.5 0 113 0v6a1.5 1.5 0 01-3 0v-6zM6 10.333v5.43a2 2 0 001.106 1.79l.05.025A4 4 0 008.943 18h5.416a2 2 0 001.962-1.608l1.2-6A2 2 0 0015.56 8H12V4a2 2 0 00-2-2 1 1 0 00-1 1v.667a4 4 0 01-.8 2.4L6.8 7.933a4 4 0 00-.8 2.4z" />
                </svg>
                {item.score} points
              </span>
            )}
          </div>

          {/* Tags */}
          {item.tags.length > 0 && (
            <div className="flex flex-wrap gap-2 mb-6">
              {item.tags.map(tag => (
                <span
                  key={tag}
                  className="px-3 py-1 bg-slate-700 text-slate-300 rounded-full text-sm"
                >
                  {tag}
                </span>
              ))}
            </div>
          )}

          {/* Summary */}
          {summaryText && (
            <div className="mb-6">
              <h3 className="text-sm font-medium text-slate-300 mb-2">Summary</h3>
              <p className="text-slate-400 leading-relaxed">{summaryText}</p>
            </div>
          )}

          {/* Content */}
          {contentText && contentText !== summaryText && (
            <div className="mb-6">
              <h3 className="text-sm font-medium text-slate-300 mb-2">Content</h3>
              <div className="text-slate-400 leading-relaxed whitespace-pre-wrap">
                {contentText}
              </div>
            </div>
          )}

          {/* Source Info */}
          {source && (
            <div className="p-4 bg-slate-900 rounded-lg">
              <h3 className="text-sm font-medium text-slate-300 mb-2">About {source.name}</h3>
              <p className="text-sm text-slate-400">{source.description}</p>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center gap-3 p-4 border-t border-slate-700 bg-slate-900">
          <a
            href={item.url}
            target="_blank"
            rel="noopener noreferrer"
            className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded-lg transition-colors"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
            </svg>
            {isVideoItem ? 'View Video' : 'Read Original'}
          </a>
          {item.commentsUrl && item.commentsUrl !== item.url && (
            <a
              href={item.commentsUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center justify-center gap-2 px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-lg transition-colors"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
              </svg>
              Comments
            </a>
          )}
        </div>
      </div>
    </div>
  );
}

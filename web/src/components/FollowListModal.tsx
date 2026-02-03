import { useState, useEffect, useCallback } from 'react';
import { getFollowers, getFollowing } from '../socialApi';
import type { PilotSummary } from '../socialTypes';

type ListType = 'followers' | 'following';

interface FollowListModalProps {
  userId: string;
  userName: string;
  type: ListType;
  onClose: () => void;
  onSelectPilot: (pilotId: string) => void;
}

export function FollowListModal({ userId, userName, type, onClose, onSelectPilot }: FollowListModalProps) {
  const [pilots, setPilots] = useState<PilotSummary[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadList = useCallback(async () => {
    try {
      setIsLoading(true);
      setError(null);
      const response = type === 'followers' 
        ? await getFollowers(userId, 50, 0)
        : await getFollowing(userId, 50, 0);
      setPilots(response.pilots);
      setTotalCount(response.totalCount);
    } catch (err) {
      setError(err instanceof Error ? err.message : `Failed to load ${type}`);
    } finally {
      setIsLoading(false);
    }
  }, [userId, type]);

  useEffect(() => {
    loadList();
  }, [loadList]);

  const getDisplayName = (pilot: PilotSummary) => {
    return pilot.callSign || 'Unknown Pilot';
  };

  const handlePilotClick = (pilotId: string) => {
    onClose();
    onSelectPilot(pilotId);
  };

  return (
    <div 
      className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div className="bg-slate-800 rounded-xl w-full max-w-md max-h-[80vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-slate-700">
          <div>
            <h2 className="text-lg font-semibold text-white">
              {type === 'followers' ? 'Followers' : 'Following'}
            </h2>
            <p className="text-sm text-slate-400">
              {type === 'followers' 
                ? `People following ${userName}`
                : `People ${userName} follows`}
            </p>
          </div>
          <button
            onClick={onClose}
            className="p-2 text-slate-400 hover:text-white transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4">
          {isLoading && (
            <div className="flex items-center justify-center py-12">
              <div className="animate-spin rounded-full h-8 w-8 border-t-2 border-b-2 border-primary-500"></div>
            </div>
          )}

          {error && (
            <div className="p-4 bg-red-500/10 border border-red-500/50 rounded-lg text-red-400">
              {error}
            </div>
          )}

          {!isLoading && pilots.length === 0 && (
            <div className="text-center py-12 text-slate-500">
              <svg className="w-12 h-12 mx-auto mb-3 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
              </svg>
              <p>{type === 'followers' ? 'No followers yet' : 'Not following anyone'}</p>
            </div>
          )}

          {!isLoading && pilots.length > 0 && (
            <div className="space-y-2">
              <p className="text-sm text-slate-400 mb-3">
                {totalCount} {type === 'followers' ? 'follower' : 'following'}{totalCount !== 1 ? 's' : ''}
              </p>
              {pilots.map((pilot) => (
                <button
                  key={pilot.id}
                  onClick={() => handlePilotClick(pilot.id)}
                  className="w-full flex items-center gap-3 p-3 bg-slate-700/50 hover:bg-slate-700 rounded-lg transition-colors text-left"
                >
                  {/* Avatar */}
                  {pilot.effectiveAvatarUrl ? (
                    <img
                      src={pilot.effectiveAvatarUrl}
                      alt=""
                      className="w-10 h-10 rounded-full object-cover"
                    />
                  ) : (
                    <div className="w-10 h-10 rounded-full bg-slate-600 flex items-center justify-center">
                      <svg
                        className="w-5 h-5 text-slate-400"
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
                    className="w-4 h-4 text-slate-500"
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
        </div>
      </div>
    </div>
  );
}

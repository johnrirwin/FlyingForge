import { type MouseEvent, useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { clearBuildReaction, createTempBuild, listPublicBuilds, setBuildReaction } from '../buildApi';
import type { Build, BuildReaction } from '../buildTypes';
import { findPart, getBuildPartDisplayName } from '../buildTypes';
import { useAuth } from '../hooks/useAuth';
import { MobileFloatingControls } from './MobileFloatingControls';

export function PublicBuildsPage() {
  const navigate = useNavigate();
  const { isAuthenticated } = useAuth();

  const [builds, setBuilds] = useState<Build[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isCreating, setIsCreating] = useState(false);
  const [isMobileControlsOpen, setIsMobileControlsOpen] = useState(false);
  const [reactionPendingById, setReactionPendingById] = useState<Record<string, boolean>>({});
  const [error, setError] = useState<string | null>(null);
  const [frameFilter, setFrameFilter] = useState('');

  const loadBuilds = useCallback(async () => {
    setIsLoading(true);
    setError(null);
    try {
      const response = await listPublicBuilds({
        sort: 'newest',
        frameFilter: frameFilter.trim() || undefined,
        limit: 60,
      });
      setBuilds(response.builds ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load builds');
    } finally {
      setIsLoading(false);
    }
  }, [frameFilter]);

  useEffect(() => {
    loadBuilds();
  }, [loadBuilds]);

  const handleBuildYourOwn = useCallback(async () => {
    if (isAuthenticated) {
      setIsMobileControlsOpen(false);
      navigate('/me/builds?new=1');
      return;
    }

    setIsCreating(true);
    setError(null);
    try {
      const created = await createTempBuild({ title: 'Temporary Build' });
      setIsMobileControlsOpen(false);
      navigate(created.url);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create temporary build');
    } finally {
      setIsCreating(false);
    }
  }, [isAuthenticated, navigate]);

  const handleReaction = useCallback(async (
    event: MouseEvent<HTMLButtonElement>,
    build: Build,
    reaction: BuildReaction,
  ) => {
    event.preventDefault();
    event.stopPropagation();

    if (!isAuthenticated) {
      setError('Sign in to like or dislike builds.');
      return;
    }

    if (reactionPendingById[build.id]) {
      return;
    }

    setReactionPendingById((prev) => ({ ...prev, [build.id]: true }));
    setError(null);

    try {
      const updatedBuild = build.viewerReaction === reaction
        ? await clearBuildReaction(build.id)
        : await setBuildReaction(build.id, reaction);

      setBuilds((prev) => prev.map((item) => (
        item.id === build.id
          ? {
              ...item,
              likeCount: updatedBuild.likeCount,
              dislikeCount: updatedBuild.dislikeCount,
              viewerReaction: updatedBuild.viewerReaction,
            }
          : item
      )));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save build reaction');
    } finally {
      setReactionPendingById((prev) => {
        const next = { ...prev };
        delete next[build.id];
        return next;
      });
    }
  }, [isAuthenticated, reactionPendingById]);

  const emptyMessage = useMemo(() => {
    if (frameFilter.trim()) {
      return `No published builds match "${frameFilter.trim()}" yet.`;
    }
    return 'No public builds are published yet. Be the first to share one.';
  }, [frameFilter]);

  const controls = (
    <div className="border-b border-slate-800 bg-slate-900">
      <div className="px-4 md:px-6 py-4">
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 className="text-2xl font-semibold text-white">Public Builds</h1>
            <p className="mt-1 text-sm text-slate-400">
              Browse pilot builds, compare parts, and start your own setup.
            </p>
          </div>
          <button
            type="button"
            disabled={isCreating}
            onClick={handleBuildYourOwn}
            className="rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-primary-500 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {isCreating ? 'Creating...' : 'Build Your Own'}
          </button>
        </div>

        <div className="mt-4 grid gap-3 sm:grid-cols-2">
          <label className="text-sm text-slate-300">
            <span className="mb-1 block text-xs uppercase tracking-wide text-slate-400">Sort</span>
            <select
              value="newest"
              disabled
              className="h-11 w-full rounded-lg border border-slate-600 bg-slate-700 px-3 text-white"
            >
              <option value="newest">Newest</option>
            </select>
          </label>
          <label className="text-sm text-slate-300">
            <span className="mb-1 block text-xs uppercase tracking-wide text-slate-400">Frame filter</span>
            <input
              value={frameFilter}
              onChange={(event) => setFrameFilter(event.target.value)}
              placeholder="Example: 5, whoop, 7 inch"
              className="h-11 w-full rounded-lg border border-slate-600 bg-slate-700 px-3 text-white placeholder:text-slate-500 focus:border-primary-500 focus:outline-none"
            />
          </label>
        </div>
      </div>
    </div>
  );

  return (
    <div className="relative flex-1 min-h-0 flex flex-col overflow-hidden">
      <div className="hidden md:block flex-shrink-0">{controls}</div>

      <div
        className="flex-1 min-h-0 overflow-y-auto p-4 md:p-6 pt-24 md:pt-6"
        onScroll={(event) => {
          setIsMobileControlsOpen((prev) => (prev ? false : prev));

          if (typeof window === 'undefined') return;
          if (!window.matchMedia || !window.matchMedia('(pointer: coarse)').matches) return;

          const activeElement = document.activeElement;
          if (!(activeElement instanceof HTMLElement) || activeElement === document.body) return;

          const scrollContainer = event.currentTarget;
          if (!scrollContainer.contains(activeElement)) return;

          const tagName = activeElement.tagName;
          if (tagName === 'INPUT' || tagName === 'TEXTAREA' || tagName === 'SELECT') {
            activeElement.blur();
          }
        }}
      >
        <div className="space-y-6">
          {error && (
            <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-300">
              {error}
            </div>
          )}

          {isLoading ? (
            <div className="rounded-xl border border-slate-700 bg-slate-800/60 p-8 text-center text-slate-400">Loading public builds...</div>
          ) : builds.length === 0 ? (
            <div className="rounded-xl border border-slate-700 bg-slate-800/60 p-8 text-center text-slate-400">{emptyMessage}</div>
          ) : (
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
              {builds.map((build) => {
                const frame = findPart(build.parts, 'frame');
                const motors = findPart(build.parts, 'motor');
                const receiver = findPart(build.parts, 'receiver');
                const vtx = findPart(build.parts, 'vtx');
                const aio = findPart(build.parts, 'aio');
                const stack = findPart(build.parts, 'stack');
                const fc = findPart(build.parts, 'fc');
                const esc = findPart(build.parts, 'esc');
                const pilotName = build.pilot?.callSign || build.pilot?.displayName || 'Pilot';
                const likeCount = build.likeCount ?? 0;
                const dislikeCount = build.dislikeCount ?? 0;
                const isReactionPending = !!reactionPendingById[build.id];

                return (
                  <div
                    key={build.id}
                    className="overflow-hidden rounded-xl border border-slate-700 bg-slate-800/60 transition hover:border-primary-500/40 hover:bg-slate-800"
                  >
                    <Link to={`/builds/${build.id}`} className="block">
                      <div className="aspect-[16/9] w-full bg-slate-900">
                        {build.mainImageUrl ? (
                          <img src={build.mainImageUrl} alt={build.title} className="h-full w-full object-cover" />
                        ) : (
                          <div className="flex h-full items-center justify-center text-sm text-slate-500">No build image</div>
                        )}
                      </div>
                      <div className="space-y-3 p-4">
                        <div>
                          <h2 className="line-clamp-2 text-lg font-semibold text-white">{build.title}</h2>
                          <p className="text-sm text-slate-400">by {pilotName}</p>
                        </div>
                        <ul className="space-y-1 text-sm text-slate-300">
                          <li>Frame: {frame ? getBuildPartDisplayName(frame) : '—'}</li>
                          <li>Motors: {motors ? getBuildPartDisplayName(motors) : '—'}</li>
                          <li>
                            Power: {aio?.catalogItem
                              ? `AIO — ${getBuildPartDisplayName(aio)}`
                              : stack?.catalogItem
                                ? `FC/ESC Stack — ${getBuildPartDisplayName(stack)}`
                              : fc?.catalogItem || esc?.catalogItem
                                ? `${fc?.catalogItem ? getBuildPartDisplayName(fc) : 'FC'} + ${esc?.catalogItem ? getBuildPartDisplayName(esc) : 'ESC'}`
                                : '—'}
                          </li>
                          <li>Receiver: {receiver ? getBuildPartDisplayName(receiver) : '—'}</li>
                          <li>VTX: {vtx ? getBuildPartDisplayName(vtx) : '—'}</li>
                        </ul>
                        <div className="flex items-center justify-between text-xs text-slate-400">
                          <span>{build.verified ? 'Verified parts' : 'Unverified parts'}</span>
                          {build.publishedAt && <span>{new Date(build.publishedAt).toLocaleDateString()}</span>}
                        </div>
                      </div>
                    </Link>
                    <div className="px-4 pb-4">
                      <div className="flex items-center gap-2 text-xs text-slate-400">
                        <button
                          type="button"
                          aria-label={`Like ${build.title}`}
                          disabled={isReactionPending}
                          onClick={(event) => handleReaction(event, build, 'LIKE')}
                          className={`inline-flex items-center gap-1 rounded-full border px-2 py-1 text-xs transition ${
                            build.viewerReaction === 'LIKE'
                              ? 'border-emerald-400/60 bg-emerald-500/20 text-emerald-200'
                              : 'border-slate-600 text-slate-300 hover:border-emerald-500/50 hover:text-emerald-200'
                          } ${isReactionPending ? 'cursor-not-allowed opacity-60' : ''}`}
                        >
                          <svg
                            aria-hidden="true"
                            className="h-3.5 w-3.5"
                            fill="none"
                            viewBox="0 0 24 24"
                            stroke="currentColor"
                            strokeWidth={1.75}
                          >
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              d="M14 9V5a3 3 0 00-3-3l-1 7H5a2 2 0 00-2 2v7a2 2 0 002 2h9.28a2 2 0 001.98-1.69l1.2-7A2 2 0 0015.5 9H14zM7 11v8"
                            />
                          </svg>
                          <span>{likeCount}</span>
                        </button>
                        <button
                          type="button"
                          aria-label={`Dislike ${build.title}`}
                          disabled={isReactionPending}
                          onClick={(event) => handleReaction(event, build, 'DISLIKE')}
                          className={`inline-flex items-center gap-1 rounded-full border px-2 py-1 text-xs transition ${
                            build.viewerReaction === 'DISLIKE'
                              ? 'border-rose-400/60 bg-rose-500/20 text-rose-200'
                              : 'border-slate-600 text-slate-300 hover:border-rose-500/50 hover:text-rose-200'
                          } ${isReactionPending ? 'cursor-not-allowed opacity-60' : ''}`}
                        >
                          <svg
                            aria-hidden="true"
                            className="h-3.5 w-3.5"
                            fill="none"
                            viewBox="0 0 24 24"
                            stroke="currentColor"
                            strokeWidth={1.75}
                          >
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              d="M10 15v4a3 3 0 003 3l1-7h5a2 2 0 002-2v-1a2 2 0 00-.03-.35l-1.17-6.65A2 2 0 0017.83 3H8.72a2 2 0 00-1.98 1.69l-1.2 7A2 2 0 006.5 14H10zM17 13V5"
                            />
                          </svg>
                          <span>{dislikeCount}</span>
                        </button>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>

      <MobileFloatingControls
        label="Build Filters"
        isOpen={isMobileControlsOpen}
        onToggle={() => setIsMobileControlsOpen((prev) => !prev)}
      >
        {controls}
      </MobileFloatingControls>
    </div>
  );
}

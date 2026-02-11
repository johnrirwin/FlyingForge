import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { createTempBuild, listPublicBuilds } from '../buildApi';
import type { Build } from '../buildTypes';
import { findPart, getBuildPartDisplayName } from '../buildTypes';
import { useAuth } from '../hooks/useAuth';

export function PublicBuildsPage() {
  const navigate = useNavigate();
  const { isAuthenticated } = useAuth();

  const [builds, setBuilds] = useState<Build[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isCreating, setIsCreating] = useState(false);
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
      navigate('/me/builds?new=1');
      return;
    }

    setIsCreating(true);
    setError(null);
    try {
      const created = await createTempBuild({ title: 'Temporary Build' });
      navigate(created.url);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create temporary build');
    } finally {
      setIsCreating(false);
    }
  }, [isAuthenticated, navigate]);

  const emptyMessage = useMemo(() => {
    if (frameFilter.trim()) {
      return `No published builds match "${frameFilter.trim()}" yet.`;
    }
    return 'No public builds are published yet. Be the first to share one.';
  }, [frameFilter]);

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <div className="mx-auto w-full max-w-6xl space-y-6">
        <header className="rounded-2xl border border-slate-700 bg-slate-800/70 p-5">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <h1 className="text-2xl font-semibold text-white">Public Builds</h1>
              <p className="mt-1 text-sm text-slate-400">
                Browse pilot builds, compare parts, and start your own PCPartPicker-style setup.
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
                className="w-full rounded-lg border border-slate-600 bg-slate-700 px-3 py-2 text-white"
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
                className="w-full rounded-lg border border-slate-600 bg-slate-700 px-3 py-2 text-white placeholder:text-slate-500 focus:border-primary-500 focus:outline-none"
              />
            </label>
          </div>
        </header>

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
              const fc = findPart(build.parts, 'fc');
              const esc = findPart(build.parts, 'esc');
              const pilotName = build.pilot?.callSign || build.pilot?.displayName || 'Pilot';

              return (
                <Link
                  key={build.id}
                  to={`/builds/${build.id}`}
                  className="overflow-hidden rounded-xl border border-slate-700 bg-slate-800/60 transition hover:border-primary-500/40 hover:bg-slate-800"
                >
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
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { createTempBuild, getPublicBuild } from '../buildApi';
import type { Build, BuildPart } from '../buildTypes';
import { getBuildPartDisplayName } from '../buildTypes';
import { useAuth } from '../hooks/useAuth';

interface SectionPart {
  label: string;
  part?: BuildPart;
}

export function PublicBuildDetailsPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { isAuthenticated } = useAuth();

  const [build, setBuild] = useState<Build | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isCreatingTemp, setIsCreatingTemp] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    setIsLoading(true);
    setError(null);

    getPublicBuild(id)
      .then((response) => setBuild(response))
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load build'))
      .finally(() => setIsLoading(false));
  }, [id]);

  const partsByType = useMemo(() => {
    const map = new Map<string, BuildPart>();
    for (const part of build?.parts ?? []) {
      map.set(part.gearType, part);
    }
    return map;
  }, [build?.parts]);

  const coreParts: SectionPart[] = [
    { label: 'Frame', part: partsByType.get('frame') },
    { label: 'Motors', part: partsByType.get('motor') },
    {
      label: 'Power Stack',
      part: partsByType.get('aio') ?? partsByType.get('stack') ?? undefined,
    },
    { label: 'Flight Controller', part: partsByType.get('fc') },
    { label: 'ESC', part: partsByType.get('esc') },
    { label: 'Receiver', part: partsByType.get('receiver') },
    { label: 'VTX', part: partsByType.get('vtx') },
  ];

  const optionalParts: SectionPart[] = [
    { label: 'Camera', part: partsByType.get('camera') },
    { label: 'Propellers', part: partsByType.get('prop') },
    { label: 'Antenna', part: partsByType.get('antenna') },
    { label: 'GPS', part: partsByType.get('gps') },
    { label: 'Other', part: partsByType.get('other') },
  ];

  const handleBuildYourOwn = useCallback(async () => {
    if (!build) return;

    const clonedParts = (build.parts || [])
      .filter((part) => part.catalogItemId)
      .map((part) => ({
        gearType: part.gearType,
        catalogItemId: part.catalogItemId,
        position: part.position,
        notes: part.notes,
      }));

    if (isAuthenticated) {
      // TODO: support pre-populating authenticated drafts from public builds.
      navigate('/me/builds?new=1');
      return;
    }

    setIsCreatingTemp(true);
    setError(null);
    try {
      const temp = await createTempBuild({
        title: build.title ? `${build.title} Copy` : 'Temporary Build',
        description: build.description || '',
        parts: clonedParts,
      });
      navigate(temp.url);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create temporary build');
    } finally {
      setIsCreatingTemp(false);
    }
  }, [build, isAuthenticated, navigate]);

  if (isLoading) {
    return (
      <div className="flex-1 overflow-y-auto p-6">
        <div className="mx-auto w-full max-w-4xl rounded-xl border border-slate-700 bg-slate-800/60 p-8 text-center text-slate-400">
          Loading build...
        </div>
      </div>
    );
  }

  if (error || !build) {
    return (
      <div className="flex-1 overflow-y-auto p-6">
        <div className="mx-auto w-full max-w-4xl rounded-xl border border-red-500/30 bg-red-500/10 p-6 text-sm text-red-300">
          {error || 'Build not found'}
        </div>
      </div>
    );
  }

  const pilotName = build.pilot?.callSign || build.pilot?.displayName || 'Pilot';

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <div className="mx-auto w-full max-w-4xl space-y-6">
        <header className="rounded-2xl border border-slate-700 bg-slate-800/70 p-5">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div className="space-y-2">
              <Link to="/builds" className="text-xs uppercase tracking-wide text-primary-400 hover:text-primary-300">
                ← Back to Builds
              </Link>
              <h1 className="text-2xl font-semibold text-white">{build.title}</h1>
              <div className="flex flex-wrap items-center gap-2 text-sm text-slate-400">
                <span>
                  Pilot:{' '}
                  {build.pilot?.isProfilePublic && build.pilot?.userId ? (
                    <Link className="text-primary-400 hover:text-primary-300" to={`/social/pilots/${build.pilot.userId}`}>
                      {pilotName}
                    </Link>
                  ) : (
                    <span>{pilotName}</span>
                  )}
                </span>
                <span>•</span>
                <span>{build.verified ? 'Verified' : 'Unverified'}</span>
              </div>
            </div>
            <button
              type="button"
              disabled={isCreatingTemp}
              onClick={handleBuildYourOwn}
              className="rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-primary-500 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {isCreatingTemp ? 'Creating...' : 'Build Your Own'}
            </button>
          </div>
          {build.description && <p className="mt-4 text-sm text-slate-300">{build.description}</p>}
        </header>

        {build.mainImageUrl && (
          <div className="overflow-hidden rounded-xl border border-slate-700 bg-slate-800/70">
            <img src={build.mainImageUrl} alt={build.title} className="max-h-[420px] w-full object-cover" />
          </div>
        )}

        <section className="rounded-xl border border-slate-700 bg-slate-800/60 p-5">
          <h2 className="text-lg font-semibold text-white">Core Components</h2>
          <div className="mt-4 grid gap-3 md:grid-cols-2">
            {coreParts.map((entry) => (
              <PartRow key={entry.label} label={entry.label} part={entry.part} />
            ))}
          </div>
        </section>

        <section className="rounded-xl border border-slate-700 bg-slate-800/60 p-5">
          <h2 className="text-lg font-semibold text-white">Optional Components</h2>
          <div className="mt-4 grid gap-3 md:grid-cols-2">
            {optionalParts.map((entry) => (
              <PartRow key={entry.label} label={entry.label} part={entry.part} />
            ))}
          </div>
        </section>
      </div>
    </div>
  );
}

function PartRow({ label, part }: { label: string; part?: BuildPart }) {
  return (
    <div className="rounded-lg border border-slate-700 bg-slate-800/80 p-3">
      <p className="text-xs uppercase tracking-wide text-slate-500">{label}</p>
      <p className="mt-1 text-sm text-slate-200">{part ? getBuildPartDisplayName(part) : 'Not specified'}</p>
    </div>
  );
}

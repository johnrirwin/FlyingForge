import { useEffect, useMemo, useRef, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { getTempBuild, shareTempBuild, updateTempBuild } from '../buildApi';
import type { Build, BuildPart } from '../buildTypes';
import { BuildBuilder } from './BuildBuilder';

export function TempBuildPage() {
  const { token } = useParams<{ token: string }>();
  const navigate = useNavigate();
  const routeToken = token ?? '';
  const [activeToken, setActiveToken] = useState(routeToken);

  const [build, setBuild] = useState<Build | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isAutoSaving, setIsAutoSaving] = useState(false);
  const [isCopying, setIsCopying] = useState(false);
  const [autoSaveMessage, setAutoSaveMessage] = useState<string | null>(null);
  const [copyMessage, setCopyMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const lastSavedPayloadRef = useRef<string>('');
  const lastSharedPayloadRef = useRef<string>('');
  const hydratedTokenRef = useRef<string>('');
  const buildRef = useRef<Build | null>(null);
  const buildTokenRef = useRef<string>('');

  useEffect(() => {
    setActiveToken(routeToken);
  }, [routeToken]);

  useEffect(() => {
    buildRef.current = build;
  }, [build]);

  useEffect(() => {
    if (!routeToken) return;
    if (routeToken === hydratedTokenRef.current && buildRef.current) {
      setIsLoading(false);
      return;
    }
    setIsLoading(true);
    setError(null);
    setBuild(null);
    buildTokenRef.current = '';

    getTempBuild(routeToken)
      .then((response) => {
        const normalized = normalizeTempBuild(response);
        setBuild(normalized);
        hydratedTokenRef.current = routeToken;
        lastSavedPayloadRef.current = buildPayloadKey(normalized);
        buildTokenRef.current = routeToken;
      })
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load temporary build'))
      .finally(() => setIsLoading(false));
  }, [routeToken]);

  const shareUrl = useMemo(() => {
    if (!activeToken) return '';
    if (typeof window === 'undefined') return `/builds/temp/${activeToken}`;
    return `${window.location.origin}/builds/temp/${activeToken}`;
  }, [activeToken]);

  const hasUnsharedChanges = useMemo(() => {
    if (!build || !lastSharedPayloadRef.current) return false;
    return buildPayloadKey(build) !== lastSharedPayloadRef.current;
  }, [build]);

  useEffect(() => {
    if (!activeToken || !build || build.status === 'SHARED' || isLoading) return;
    if (hydratedTokenRef.current !== activeToken) return;
    if (buildTokenRef.current !== activeToken) return;

    const payloadKey = buildPayloadKey(build);
    if (payloadKey === lastSavedPayloadRef.current) return;

    const timeout = window.setTimeout(async () => {
      setIsAutoSaving(true);
      setAutoSaveMessage(null);
      setError(null);
      try {
        const updated = await updateTempBuild(activeToken, {
          title: build.title,
          description: build.description,
          parts: toPartInputs(build.parts),
        });
        const updatedBuild = normalizeTempBuild(updated.build);
        const canonicalPayloadKey = buildPayloadKey(updatedBuild);
        lastSavedPayloadRef.current = canonicalPayloadKey;
        setBuild(updatedBuild);
        const nextToken = updated.token || activeToken;
        buildTokenRef.current = nextToken;
        if (nextToken !== activeToken) {
          setActiveToken(nextToken);
          hydratedTokenRef.current = nextToken;
          navigate(`/builds/temp/${nextToken}`, { replace: true });
        }
        setAutoSaveMessage('Changes saved');
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to save temporary build');
      } finally {
        setIsAutoSaving(false);
      }
    }, 250);

    return () => window.clearTimeout(timeout);
  }, [activeToken, build, isLoading, navigate]);

  const handleCopy = async () => {
    if (!activeToken || !build || !shareUrl) return;
    if (isLoading || isAutoSaving || hydratedTokenRef.current !== activeToken || buildTokenRef.current !== activeToken) {
      setCopyMessage('Please wait for the latest build URL to finish updating, then copy again.');
      window.setTimeout(() => setCopyMessage(null), 3000);
      return;
    }

    if (build.status === 'SHARED') {
      try {
        await navigator.clipboard.writeText(shareUrl);
        setCopyMessage('Share URL copied to clipboard.');
      } catch {
        setCopyMessage('Unable to copy automatically. Copy the URL manually.');
      }
      window.setTimeout(() => setCopyMessage(null), 3000);
      return;
    }

    const payloadKey = buildPayloadKey(build);
    setIsCopying(true);
    setError(null);
    try {
      let tokenToShare = activeToken;
      let sharedPayloadKey = payloadKey;
      if (payloadKey !== lastSavedPayloadRef.current) {
        const updated = await updateTempBuild(activeToken, {
          title: build.title,
          description: build.description,
          parts: toPartInputs(build.parts),
        });
        const normalizedUpdatedBuild = normalizeTempBuild(updated.build);
        const normalizedPayloadKey = buildPayloadKey(normalizedUpdatedBuild);
        lastSavedPayloadRef.current = normalizedPayloadKey;
        setBuild(normalizedUpdatedBuild);
        sharedPayloadKey = normalizedPayloadKey;
        tokenToShare = updated.token || activeToken;
        buildTokenRef.current = tokenToShare;
        if (tokenToShare !== activeToken) {
          setActiveToken(tokenToShare);
          hydratedTokenRef.current = tokenToShare;
          navigate(`/builds/temp/${tokenToShare}`, { replace: true });
        }
      }

      const shared = await shareTempBuild(tokenToShare);
      const copiedUrl = toAbsoluteTempBuildUrl(shared.url || `/builds/temp/${shared.token}`);
      lastSharedPayloadRef.current = sharedPayloadKey;

      try {
        await navigator.clipboard.writeText(copiedUrl);
        setCopyMessage('Share URL copied to clipboard. This link is saved permanently.');
      } catch {
        setCopyMessage('Share URL saved. Copy the URL manually if clipboard access is blocked.');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to copy share URL');
    } finally {
      setIsCopying(false);
      window.setTimeout(() => setCopyMessage(null), 3000);
    }
  };

  if (isLoading) {
    return (
      <div className="flex-1 overflow-y-auto p-6">
        <div className="mx-auto w-full max-w-4xl rounded-xl border border-slate-700 bg-slate-800/60 p-8 text-center text-slate-400">
          Loading temporary build...
        </div>
      </div>
    );
  }

  if (!build || !routeToken) {
    return (
      <div className="flex-1 overflow-y-auto p-6">
        <div className="mx-auto w-full max-w-4xl rounded-xl border border-red-500/30 bg-red-500/10 p-6 text-sm text-red-300">
          {error || 'Temporary build not found or expired.'}
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <div className="mx-auto w-full max-w-4xl space-y-6">
        <header className="rounded-2xl border border-slate-700 bg-slate-800/70 p-5">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div className="space-y-2">
              <Link to="/builds" className="text-xs uppercase tracking-wide text-primary-400 hover:text-primary-300">
                ← Back to Public Builds
              </Link>
              <h1 className="text-2xl font-semibold text-white">Temporary Build</h1>
              {build.status === 'SHARED' ? (
                <p className="text-sm text-emerald-300">This link is a saved snapshot that will not expire.</p>
              ) : (
                <p className="text-sm text-slate-400">
                  This build link expires on {build.expiresAt ? new Date(build.expiresAt).toLocaleString() : 'unknown date'}.
                </p>
              )}
            </div>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={handleCopy}
                disabled={isCopying}
                className="rounded-lg bg-emerald-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {isCopying ? 'Copying...' : 'Copy Share URL'}
              </button>
            </div>
          </div>

          <div className="mt-4 rounded-lg border border-slate-700 bg-slate-900/60 p-3 text-xs text-slate-300">
            <p className="break-all">{shareUrl}</p>
          </div>
          <p className="mt-2 text-xs text-slate-400">This URL rotates when the build changes. Copy Share URL saves a permanent snapshot link.</p>
          {hasUnsharedChanges && <p className="mt-2 text-xs text-amber-300">Build changed since last copy. Copy again to generate a new share URL.</p>}

          {copyMessage && <p className="mt-2 text-xs text-emerald-300">{copyMessage}</p>}
          {isAutoSaving && <p className="mt-2 text-xs text-slate-300">Saving changes...</p>}
          {!isAutoSaving && autoSaveMessage && <p className="mt-2 text-xs text-emerald-300">{autoSaveMessage}</p>}
          {error && <p className="mt-2 text-xs text-red-300">{error}</p>}
        </header>

        <BuildBuilder
          title={build.title}
          description={build.description || ''}
          parts={build.parts || []}
          readOnly={build.status === 'SHARED'}
          onTitleChange={(value) => setBuild((prev) => (prev ? { ...prev, title: value } : prev))}
          onDescriptionChange={(value) => setBuild((prev) => (prev ? { ...prev, description: value } : prev))}
          onPartsChange={(parts) => setBuild((prev) => (prev ? { ...prev, parts: parts ?? [] } : prev))}
        />
      </div>
    </div>
  );
}

function normalizeTempBuild(build: Build): Build {
  return {
    ...build,
    parts: build.parts ?? [],
  };
}

function toAbsoluteTempBuildUrl(url: string) {
  const trimmed = url.trim();
  if (!trimmed) return '';
  if (typeof window === 'undefined') return trimmed;
  try {
    return new URL(trimmed, window.location.origin).toString();
  } catch {
    return trimmed;
  }
}

function toPartInputs(parts?: BuildPart[]) {
  return (parts ?? [])
    .filter((part) => part.catalogItemId)
    .map((part) => ({
      gearType: part.gearType,
      catalogItemId: part.catalogItemId,
      position: part.position,
      notes: part.notes,
    }));
}

function buildPayloadKey(build: Build) {
  const sortedParts = [...toPartInputs(build.parts)].sort((a, b) => {
    if (a.gearType !== b.gearType) {
      return a.gearType.localeCompare(b.gearType);
    }
    if ((a.position ?? 0) !== (b.position ?? 0)) {
      return (a.position ?? 0) - (b.position ?? 0);
    }
    if (a.catalogItemId !== b.catalogItemId) {
      return a.catalogItemId.localeCompare(b.catalogItemId);
    }
    return (a.notes ?? '').localeCompare(b.notes ?? '');
  });

  return JSON.stringify({
    title: build.title ?? '',
    description: build.description ?? '',
    parts: sortedParts,
  });
}

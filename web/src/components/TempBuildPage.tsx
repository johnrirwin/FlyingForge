import { useEffect, useMemo, useRef, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { getTempBuild, shareTempBuild, updateTempBuild } from '../buildApi';
import type { Build, BuildPart } from '../buildTypes';
import { BuildBuilder } from './BuildBuilder';

export function TempBuildPage() {
  const { token } = useParams<{ token: string }>();

  const [build, setBuild] = useState<Build | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isAutoSaving, setIsAutoSaving] = useState(false);
  const [isSharing, setIsSharing] = useState(false);
  const [autoSaveMessage, setAutoSaveMessage] = useState<string | null>(null);
  const [copyMessage, setCopyMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const lastSavedPayloadRef = useRef<string>('');

  useEffect(() => {
    if (!token) return;
    setIsLoading(true);
    setError(null);

    getTempBuild(token)
      .then((response) => {
        const normalized = {
          ...response,
          parts: response.parts ?? [],
        };
        setBuild(normalized);
        lastSavedPayloadRef.current = buildPayloadKey(normalized);
      })
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load temporary build'))
      .finally(() => setIsLoading(false));
  }, [token]);

  const shareUrl = useMemo(() => {
    if (!token) return '';
    if (typeof window === 'undefined') return `/builds/temp/${token}`;
    return `${window.location.origin}/builds/temp/${token}`;
  }, [token]);

  useEffect(() => {
    if (!token || !build) return;

    const payloadKey = buildPayloadKey(build);
    if (payloadKey === lastSavedPayloadRef.current) return;

    const timeout = window.setTimeout(async () => {
      setIsAutoSaving(true);
      setAutoSaveMessage(null);
      setError(null);
      try {
        const updated = await updateTempBuild(token, {
          title: build.title,
          description: build.description,
          parts: toPartInputs(build.parts),
        });
        lastSavedPayloadRef.current = payloadKey;
        setBuild((prev) => {
          if (!prev) return prev;
          return {
            ...prev,
            updatedAt: updated.updatedAt,
            expiresAt: updated.expiresAt ?? prev.expiresAt,
          };
        });
        setAutoSaveMessage('Changes saved');
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to save temporary build');
      } finally {
        setIsAutoSaving(false);
      }
    }, 250);

    return () => window.clearTimeout(timeout);
  }, [build, token]);

  const handleCopy = async () => {
    if (!shareUrl) return;

    try {
      await navigator.clipboard.writeText(shareUrl);
      setCopyMessage('Share URL copied to clipboard');
    } catch {
      setCopyMessage('Unable to copy automatically. Copy the URL manually.');
    }

    window.setTimeout(() => setCopyMessage(null), 2500);
  };

  const handleShare = async () => {
    if (!token || !build || build.status === 'SHARED') return;

    setIsSharing(true);
    setError(null);
    try {
      const shared = await shareTempBuild(token);
      setBuild((prev) => (prev ? { ...prev, ...shared } : shared));
      try {
        await navigator.clipboard.writeText(shareUrl);
        setCopyMessage('Shared link copied to clipboard. This link will not expire.');
      } catch {
        setCopyMessage('Link shared. Copy the URL manually if clipboard access is blocked.');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to share temporary build');
    } finally {
      setIsSharing(false);
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

  if (!build || !token) {
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
                <p className="text-sm text-emerald-300">This link has been shared and will not expire.</p>
              ) : (
                <p className="text-sm text-slate-400">
                  This build link expires on {build.expiresAt ? new Date(build.expiresAt).toLocaleString() : 'unknown date'}.
                </p>
              )}
            </div>
            <div className="flex gap-2">
              <button
                type="button"
                disabled={isSharing || build.status === 'SHARED'}
                onClick={handleShare}
                className="rounded-lg bg-emerald-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {build.status === 'SHARED' ? 'Shared' : isSharing ? 'Sharing...' : 'Share'}
              </button>
              <button
                type="button"
                onClick={handleCopy}
                className="rounded-lg border border-slate-600 px-4 py-2 text-sm font-medium text-slate-200 transition hover:border-slate-500 hover:text-white"
              >
                Copy Share URL
              </button>
            </div>
          </div>

          <div className="mt-4 rounded-lg border border-slate-700 bg-slate-900/60 p-3 text-xs text-slate-300">
            <p className="break-all">{shareUrl}</p>
          </div>
          {build.status !== 'SHARED' && (
            <p className="mt-2 text-xs text-amber-300">
              Copying this URL manually keeps it temporary. Click Share to make the link permanent.
            </p>
          )}

          {copyMessage && <p className="mt-2 text-xs text-emerald-300">{copyMessage}</p>}
          {isAutoSaving && <p className="mt-2 text-xs text-slate-300">Saving changes...</p>}
          {!isAutoSaving && autoSaveMessage && <p className="mt-2 text-xs text-emerald-300">{autoSaveMessage}</p>}
          {error && <p className="mt-2 text-xs text-red-300">{error}</p>}
        </header>

        <BuildBuilder
          title={build.title}
          description={build.description || ''}
          parts={build.parts || []}
          onTitleChange={(value) => setBuild((prev) => (prev ? { ...prev, title: value } : prev))}
          onDescriptionChange={(value) => setBuild((prev) => (prev ? { ...prev, description: value } : prev))}
          onPartsChange={(parts) => setBuild((prev) => (prev ? { ...prev, parts: parts ?? [] } : prev))}
        />
      </div>
    </div>
  );
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

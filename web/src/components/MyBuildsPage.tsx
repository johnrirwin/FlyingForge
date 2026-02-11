import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import {
  createBuildFromAircraft,
  createDraftBuild,
  getMyBuild,
  listMyBuilds,
  publishMyBuild,
  unpublishMyBuild,
  updateMyBuild,
} from '../buildApi';
import type { Build, BuildValidationError } from '../buildTypes';
import type { Aircraft } from '../aircraftTypes';
import { listAircraft } from '../aircraftApi';
import { BuildBuilder } from './BuildBuilder';

export function MyBuildsPage() {
  const location = useLocation();
  const navigate = useNavigate();
  const autoCreateHandledRef = useRef(false);

  const [builds, setBuilds] = useState<Build[]>([]);
  const [selectedBuildId, setSelectedBuildId] = useState<string | null>(null);
  const [editorBuild, setEditorBuild] = useState<Build | null>(null);
  const [aircraft, setAircraft] = useState<Aircraft[]>([]);
  const [selectedAircraftId, setSelectedAircraftId] = useState<string>('');

  const [isLoadingList, setIsLoadingList] = useState(true);
  const [isLoadingBuild, setIsLoadingBuild] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [validationErrors, setValidationErrors] = useState<BuildValidationError[]>([]);

  const loadBuildList = useCallback(async () => {
    setIsLoadingList(true);
    try {
      const response = await listMyBuilds({ sort: 'newest', limit: 100 });
      setBuilds(response.builds ?? []);
      if (!selectedBuildId && response.builds?.length) {
        setSelectedBuildId(response.builds[0].id);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load builds');
    } finally {
      setIsLoadingList(false);
    }
  }, [selectedBuildId]);

  useEffect(() => {
    loadBuildList();
    listAircraft({ limit: 100 })
      .then((response) => setAircraft(response.aircraft ?? []))
      .catch(() => setAircraft([]));
  }, [loadBuildList]);

  useEffect(() => {
    const shouldCreate = new URLSearchParams(location.search).get('new') === '1';
    if (!shouldCreate || autoCreateHandledRef.current) return;

    autoCreateHandledRef.current = true;
    createDraftBuild({ title: 'Untitled Build' })
      .then((created) => {
        setSelectedBuildId(created.id);
        setEditorBuild(created);
        setBuilds((prev) => [created, ...prev.filter((item) => item.id !== created.id)]);
      })
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to create draft'))
      .finally(() => {
        navigate('/me/builds', { replace: true });
      });
  }, [location.search, navigate]);

  useEffect(() => {
    if (!selectedBuildId) {
      setEditorBuild(null);
      return;
    }

    setIsLoadingBuild(true);
    setError(null);
    setValidationErrors([]);

    getMyBuild(selectedBuildId)
      .then((build) => setEditorBuild(build))
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load build'))
      .finally(() => setIsLoadingBuild(false));
  }, [selectedBuildId]);

  const handleCreateDraft = async () => {
    setError(null);
    try {
      const created = await createDraftBuild({ title: 'Untitled Build' });
      setBuilds((prev) => [created, ...prev.filter((item) => item.id !== created.id)]);
      setSelectedBuildId(created.id);
      setEditorBuild(created);
      setValidationErrors([]);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create draft build');
    }
  };

  const handleCreateFromAircraft = async () => {
    if (!selectedAircraftId) return;

    setError(null);
    try {
      const created = await createBuildFromAircraft(selectedAircraftId);
      setBuilds((prev) => [created, ...prev.filter((item) => item.id !== created.id)]);
      setSelectedBuildId(created.id);
      setEditorBuild(created);
      setValidationErrors([]);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create build from aircraft');
    }
  };

  const handleSave = async () => {
    if (!editorBuild) return;

    setIsSaving(true);
    setError(null);
    try {
      const updated = await updateMyBuild(editorBuild.id, {
        title: editorBuild.title,
        description: editorBuild.description,
        parts: toPartInputs(editorBuild.parts),
      });
      setEditorBuild(updated);
      setBuilds((prev) => [updated, ...prev.filter((item) => item.id !== updated.id)]);
      setValidationErrors([]);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save build');
    } finally {
      setIsSaving(false);
    }
  };

  const handlePublish = async () => {
    if (!editorBuild) return;

    setIsSaving(true);
    setError(null);
    try {
      const response = await publishMyBuild(editorBuild.id);
      if (!response.validation.valid) {
        setValidationErrors(response.validation.errors ?? []);
        return;
      }
      if (response.build) {
        setEditorBuild(response.build);
        setBuilds((prev) => [response.build!, ...prev.filter((item) => item.id !== response.build!.id)]);
      }
      setValidationErrors([]);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to publish build');
    } finally {
      setIsSaving(false);
    }
  };

  const handleUnpublish = async () => {
    if (!editorBuild) return;

    setIsSaving(true);
    setError(null);
    try {
      const updated = await unpublishMyBuild(editorBuild.id);
      setEditorBuild(updated);
      setBuilds((prev) => [updated, ...prev.filter((item) => item.id !== updated.id)]);
      setValidationErrors([]);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to unpublish build');
    } finally {
      setIsSaving(false);
    }
  };

  const selectedStatusLabel = useMemo(() => {
    if (!editorBuild) return '';
    switch (editorBuild.status) {
      case 'PUBLISHED':
        return 'Published';
      case 'UNPUBLISHED':
        return 'Unpublished';
      case 'DRAFT':
        return 'Draft';
      case 'TEMP':
        return 'Temporary';
      default:
        return editorBuild.status;
    }
  }, [editorBuild]);

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <div className="mx-auto w-full max-w-7xl space-y-6">
        <header className="rounded-2xl border border-slate-700 bg-slate-800/70 p-5">
          <h1 className="text-2xl font-semibold text-white">My Builds</h1>
          <p className="mt-1 text-sm text-slate-400">
            Manage private drafts, build from an existing aircraft, and publish to the public builds feed.
          </p>

          <div className="mt-4 flex flex-wrap gap-3">
            <button
              type="button"
              onClick={handleCreateDraft}
              className="rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-primary-500"
            >
              New Draft
            </button>

            <div className="flex items-center gap-2 rounded-lg border border-slate-700 bg-slate-900/70 p-1">
              <select
                value={selectedAircraftId}
                onChange={(event) => setSelectedAircraftId(event.target.value)}
                className="rounded-md border border-slate-600 bg-slate-700 px-3 py-1.5 text-sm text-white focus:border-primary-500 focus:outline-none"
              >
                <option value="">Create from aircraft...</option>
                {aircraft.map((item) => (
                  <option key={item.id} value={item.id}>{item.name}</option>
                ))}
              </select>
              <button
                type="button"
                disabled={!selectedAircraftId}
                onClick={handleCreateFromAircraft}
                className="rounded-md bg-slate-700 px-3 py-1.5 text-sm font-medium text-slate-200 transition hover:bg-slate-600 disabled:cursor-not-allowed disabled:opacity-50"
              >
                Create
              </button>
            </div>
          </div>
        </header>

        {error && (
          <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-300">
            {error}
          </div>
        )}

        <div className="grid gap-6 lg:grid-cols-[320px,minmax(0,1fr)]">
          <aside className="space-y-3 rounded-xl border border-slate-700 bg-slate-800/60 p-4">
            <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-400">Builds</h2>
            {isLoadingList ? (
              <p className="text-sm text-slate-400">Loading builds...</p>
            ) : builds.length === 0 ? (
              <p className="text-sm text-slate-400">No builds yet.</p>
            ) : (
              <div className="space-y-2">
                {builds.map((item) => (
                  <button
                    key={item.id}
                    type="button"
                    onClick={() => setSelectedBuildId(item.id)}
                    className={`w-full rounded-lg border px-3 py-2 text-left transition ${
                      selectedBuildId === item.id
                        ? 'border-primary-500/50 bg-primary-500/10'
                        : 'border-slate-700 bg-slate-900/60 hover:border-slate-600'
                    }`}
                  >
                    <p className="truncate text-sm font-medium text-white">{item.title || 'Untitled Build'}</p>
                    <p className="mt-1 text-xs text-slate-400">
                      {item.status} • {new Date(item.updatedAt).toLocaleDateString()}
                    </p>
                  </button>
                ))}
              </div>
            )}
          </aside>

          <section className="space-y-4 rounded-xl border border-slate-700 bg-slate-800/60 p-4">
            {!selectedBuildId ? (
              <p className="text-sm text-slate-400">Select a build to edit.</p>
            ) : isLoadingBuild || !editorBuild ? (
              <p className="text-sm text-slate-400">Loading selected build...</p>
            ) : (
              <>
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div>
                    <h2 className="text-lg font-semibold text-white">{editorBuild.title || 'Untitled Build'}</h2>
                    <p className="text-sm text-slate-400">
                      Status: {selectedStatusLabel} • {editorBuild.verified ? 'Verified catalog parts' : 'Needs verification'}
                    </p>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <button
                      type="button"
                      disabled={isSaving}
                      onClick={handleSave}
                      className="rounded-lg border border-slate-600 px-3 py-2 text-sm font-medium text-slate-200 transition hover:border-slate-500 hover:text-white disabled:cursor-not-allowed disabled:opacity-60"
                    >
                      {isSaving ? 'Saving...' : 'Save Draft'}
                    </button>
                    {editorBuild.status === 'PUBLISHED' ? (
                      <button
                        type="button"
                        disabled={isSaving}
                        onClick={handleUnpublish}
                        className="rounded-lg bg-amber-600 px-3 py-2 text-sm font-medium text-white transition hover:bg-amber-500 disabled:cursor-not-allowed disabled:opacity-60"
                      >
                        Unpublish
                      </button>
                    ) : (
                      <button
                        type="button"
                        disabled={isSaving}
                        onClick={handlePublish}
                        className="rounded-lg bg-emerald-600 px-3 py-2 text-sm font-medium text-white transition hover:bg-emerald-500 disabled:cursor-not-allowed disabled:opacity-60"
                      >
                        Publish
                      </button>
                    )}
                  </div>
                </div>

                {validationErrors.length > 0 && (
                  <div className="rounded-lg border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-200">
                    <p className="font-medium">Publishing is blocked:</p>
                    <ul className="mt-1 list-inside list-disc space-y-1 text-xs">
                      {validationErrors.map((validation) => (
                        <li key={`${validation.category}-${validation.code}-${validation.message}`}>{validation.message}</li>
                      ))}
                    </ul>
                  </div>
                )}

                <BuildBuilder
                  title={editorBuild.title}
                  description={editorBuild.description || ''}
                  parts={editorBuild.parts || []}
                  validationErrors={validationErrors}
                  onTitleChange={(value) => setEditorBuild((prev) => (prev ? { ...prev, title: value } : prev))}
                  onDescriptionChange={(value) => setEditorBuild((prev) => (prev ? { ...prev, description: value } : prev))}
                  onPartsChange={(parts) => setEditorBuild((prev) => (prev ? { ...prev, parts } : prev))}
                />
              </>
            )}
          </section>
        </div>
      </div>
    </div>
  );
}

function toPartInputs(parts: Build['parts']) {
  return (parts || [])
    .filter((part) => part.catalogItemId)
    .map((part) => ({
      gearType: part.gearType,
      catalogItemId: part.catalogItemId,
      position: part.position,
      notes: part.notes,
    }));
}

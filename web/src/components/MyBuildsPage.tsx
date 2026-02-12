import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import {
  createBuildFromAircraft,
  createDraftBuild,
  deleteMyBuild,
  getMyBuildImageUrl,
  getMyBuild,
  listMyBuilds,
  moderateBuildImageUpload,
  publishMyBuild,
  saveBuildImageUpload,
  type ModerationStatus,
  unpublishMyBuild,
  updateMyBuild,
} from '../buildApi';
import type { Build, BuildValidationError } from '../buildTypes';
import type { Aircraft } from '../aircraftTypes';
import { listAircraft } from '../aircraftApi';
import { BuildBuilder } from './BuildBuilder';
import { ImageUploadModal, type UploadStatusTone } from './ImageUploadModal';

interface PendingBuildImage {
  previewUrl: string;
  uploadId?: string;
  moderationStatus?: ModerationStatus;
  moderationReason?: string;
}

function revokeBlobUrl(url?: string | null) {
  if (url && url.startsWith('blob:')) {
    URL.revokeObjectURL(url);
  }
}

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
  const [showDeleteConfirmModal, setShowDeleteConfirmModal] = useState(false);
  const [deleteTargetBuildId, setDeleteTargetBuildId] = useState<string | null>(null);
  const [showImageModal, setShowImageModal] = useState(false);
  const [modalImage, setModalImage] = useState<PendingBuildImage | null>(null);
  const [imageStatusText, setImageStatusText] = useState<string | null>(null);
  const [imageStatusTone, setImageStatusTone] = useState<UploadStatusTone>('neutral');
  const [isImageUploading, setIsImageUploading] = useState(false);
  const [isImageSaving, setIsImageSaving] = useState(false);
  const [imageError, setImageError] = useState<string | null>(null);
  const modalPreviewRef = useRef<string | null>(null);

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

  useEffect(() => {
    if (modalImage?.previewUrl) {
      revokeBlobUrl(modalImage.previewUrl);
    }
    setShowImageModal(false);
    setModalImage(null);
    setImageStatusText(null);
    setImageStatusTone('neutral');
    setImageError(null);
    setIsImageUploading(false);
    setIsImageSaving(false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedBuildId]);

  useEffect(() => {
    modalPreviewRef.current = modalImage?.previewUrl ?? null;
  }, [modalImage?.previewUrl]);

  useEffect(() => () => {
    revokeBlobUrl(modalPreviewRef.current);
  }, []);

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

  const closeImageModal = () => {
    if (modalImage?.previewUrl) {
      revokeBlobUrl(modalImage.previewUrl);
    }
    setShowImageModal(false);
    setModalImage(null);
    setImageStatusText(null);
    setImageStatusTone('neutral');
    setImageError(null);
    setIsImageUploading(false);
    setIsImageSaving(false);
  };

  const handleOpenImageModal = () => {
    if (isImageSaving) return;
    setShowImageModal(true);
    setImageError(null);
    setImageStatusText(null);
    setImageStatusTone('neutral');
    setModalImage(null);
  };

  const handleImageFileSelect = async (file: File) => {
    const allowedTypes = ['image/jpeg', 'image/jpg', 'image/png', 'image/webp'];
    if (!allowedTypes.includes(file.type)) {
      setImageError('Only JPEG, PNG, and WebP images are allowed');
      return;
    }
    if (file.size > 2 * 1024 * 1024) {
      setImageError('Image must be less than 2MB');
      return;
    }

    const previewUrl = URL.createObjectURL(file);
    if (modalImage?.previewUrl) {
      revokeBlobUrl(modalImage.previewUrl);
    }

    setModalImage({ previewUrl });
    setImageError(null);

    try {
      setIsImageUploading(true);
      setImageStatusTone('neutral');
      setImageStatusText('Checking image for safety…');

      const moderation = await moderateBuildImageUpload(file);
      if (moderation.status === 'APPROVED' && moderation.uploadId) {
        setModalImage({
          previewUrl,
          uploadId: moderation.uploadId,
          moderationStatus: moderation.status,
          moderationReason: moderation.reason,
        });
        setImageStatusTone('success');
        setImageStatusText('Approved');
      } else if (moderation.status === 'REJECTED') {
        setModalImage({
          previewUrl,
          moderationStatus: moderation.status,
          moderationReason: moderation.reason,
        });
        setImageStatusTone('error');
        setImageStatusText('Not allowed');
      } else {
        setModalImage({
          previewUrl,
          moderationStatus: moderation.status,
          moderationReason: moderation.reason,
        });
        setImageStatusTone('error');
        setImageStatusText('Unable to verify right now');
      }
    } catch (err) {
      setModalImage({
        previewUrl,
        moderationStatus: 'PENDING_REVIEW',
      });
      setImageStatusTone('error');
      setImageStatusText('Unable to verify right now');
      setImageError(err instanceof Error ? err.message : 'Unable to verify image right now');
    } finally {
      setIsImageUploading(false);
    }
  };

  const refreshBuildAfterImageChange = async (buildId: string) => {
    const refreshed = await getMyBuild(buildId);
    setEditorBuild(refreshed);
    setBuilds((prev) => [refreshed, ...prev.filter((item) => item.id !== refreshed.id)]);
  };

  const handleSaveImage = async () => {
    if (isImageSaving) return;
    if (!editorBuild) return;
    if (!modalImage?.uploadId || modalImage.moderationStatus !== 'APPROVED') return;

    setIsImageSaving(true);
    setImageError(null);
    try {
      await saveBuildImageUpload(editorBuild.id, modalImage.uploadId);
      await refreshBuildAfterImageChange(editorBuild.id);
      closeImageModal();
    } catch (err) {
      setImageError(err instanceof Error ? err.message : 'Failed to upload build image');
    } finally {
      setIsImageSaving(false);
    }
  };

  const handleDelete = async (buildId: string) => {
    setIsSaving(true);
    setError(null);
    try {
      await deleteMyBuild(buildId);
      const remaining = builds.filter((item) => item.id !== buildId);
      setBuilds(remaining);
      setSelectedBuildId(remaining[0]?.id ?? null);
      setEditorBuild(null);
      setValidationErrors([]);
      setShowDeleteConfirmModal(false);
      setDeleteTargetBuildId(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete build');
    } finally {
      setIsSaving(false);
    }
  };

  const handleOpenDeleteConfirm = () => {
    if (!editorBuild || isSaving) return;
    setDeleteTargetBuildId(editorBuild.id);
    setShowDeleteConfirmModal(true);
  };

  const handleCancelDelete = () => {
    if (isSaving) return;
    setShowDeleteConfirmModal(false);
    setDeleteTargetBuildId(null);
  };

  const handleConfirmDelete = async () => {
    if (!deleteTargetBuildId) return;
    await handleDelete(deleteTargetBuildId);
  };

  useEffect(() => {
    if (!showDeleteConfirmModal) return;

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        if (isSaving) return;
        setShowDeleteConfirmModal(false);
        setDeleteTargetBuildId(null);
      }
    };

    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [isSaving, showDeleteConfirmModal]);

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

  const buildImagePreviewUrl = useMemo(() => {
    if (!editorBuild?.mainImageUrl) {
      return null;
    }
    if (editorBuild.mainImageUrl.startsWith('/api/builds/')) {
      return getMyBuildImageUrl(editorBuild.id);
    }
    return editorBuild.mainImageUrl;
  }, [editorBuild?.id, editorBuild?.mainImageUrl]);

  return (
    <div className="flex-1 overflow-y-auto overflow-x-hidden p-4 sm:p-6">
      <div className="mx-auto w-full max-w-7xl min-w-0 space-y-6">
        <header className="rounded-2xl border border-slate-700 bg-slate-800/70 p-5">
          <h1 className="text-2xl font-semibold text-white">My Builds</h1>
          <p className="mt-1 text-sm text-slate-400">
            Manage private drafts, build from an existing aircraft, and publish to the public builds feed.
          </p>

          <div className="mt-4 flex flex-wrap gap-3">
            <button
              type="button"
              onClick={handleCreateDraft}
              className="h-10 rounded-lg bg-primary-600 px-4 text-sm font-medium text-white transition hover:bg-primary-500"
            >
              New Draft
            </button>

            <div className="flex w-full min-w-0 items-center gap-2 sm:w-auto">
              <select
                value={selectedAircraftId}
                onChange={(event) => setSelectedAircraftId(event.target.value)}
                className="h-10 min-w-0 flex-1 rounded-md border border-slate-600 bg-slate-700 px-3 text-sm text-white focus:border-primary-500 focus:outline-none sm:w-56"
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
                className={`h-10 shrink-0 rounded-md px-3 text-sm font-medium transition disabled:cursor-not-allowed ${
                  selectedAircraftId
                    ? 'bg-primary-600 text-white hover:bg-primary-500'
                    : 'bg-slate-700 text-slate-400 disabled:opacity-70'
                }`}
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

        <div className="grid min-w-0 gap-6 lg:grid-cols-[320px,minmax(0,1fr)]">
          <aside className="min-w-0 space-y-3 rounded-xl border border-slate-700 bg-slate-800/60 p-4">
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

          <section className="min-w-0 space-y-4 rounded-xl border border-slate-700 bg-slate-800/60 p-4">
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
                      onClick={handleOpenDeleteConfirm}
                      className="rounded-lg bg-red-600 px-3 py-2 text-sm font-medium text-white transition hover:bg-red-500 disabled:cursor-not-allowed disabled:opacity-60"
                    >
                      Delete
                    </button>
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
                  imagePreviewUrl={buildImagePreviewUrl}
                  onImageAction={handleOpenImageModal}
                  imageActionLabel={buildImagePreviewUrl ? 'Change Image' : 'Upload Image'}
                  imageHelperText="JPEG, PNG, or WebP. Max 2MB."
                  onTitleChange={(value) => setEditorBuild((prev) => (prev ? { ...prev, title: value } : prev))}
                  onDescriptionChange={(value) => setEditorBuild((prev) => (prev ? { ...prev, description: value } : prev))}
                  onPartsChange={(parts) => setEditorBuild((prev) => (prev ? { ...prev, parts } : prev))}
                />
              </>
            )}
          </section>
        </div>
      </div>

      {showDeleteConfirmModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/70" onClick={handleCancelDelete} />
          <div
            role="dialog"
            aria-modal="true"
            aria-labelledby="delete-build-modal-title"
            className="relative w-full max-w-md rounded-xl border border-red-500/40 bg-slate-800 p-6 shadow-2xl"
          >
            <div className="mb-4 flex items-start justify-between gap-3">
              <h3 id="delete-build-modal-title" className="text-lg font-semibold text-white">Delete build?</h3>
              <button
                onClick={handleCancelDelete}
                disabled={isSaving}
                aria-label="Close delete build modal"
                className="rounded-lg p-2 text-slate-400 transition-colors hover:bg-slate-700 hover:text-white disabled:opacity-50"
              >
                <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <p className="mb-6 text-sm text-slate-300">
              Delete{' '}
              <span className="font-semibold text-white">
                {editorBuild?.title || 'this build'}
              </span>
              ? This cannot be undone.
            </p>

            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={handleCancelDelete}
                disabled={isSaving}
                className="rounded-lg border border-slate-600 px-4 py-2 text-sm font-medium text-slate-200 transition hover:border-slate-500 hover:text-white disabled:cursor-not-allowed disabled:opacity-60"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={() => void handleConfirmDelete()}
                disabled={isSaving}
                className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white transition hover:bg-red-500 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {isSaving ? 'Deleting...' : 'Delete'}
              </button>
            </div>
          </div>
        </div>
      )}

      <ImageUploadModal
        isOpen={showImageModal}
        title={buildImagePreviewUrl ? 'Update Build Image' : 'Upload Build Image'}
        previewUrl={modalImage?.previewUrl ?? buildImagePreviewUrl}
        previewAlt={editorBuild?.title || 'Build image preview'}
        placeholder="🚁"
        accept="image/jpeg,image/jpg,image/png,image/webp"
        helperText="JPEG, PNG, or WebP. Max 2MB."
        selectButtonLabel={modalImage?.previewUrl ? 'Choose Different' : 'Select Image'}
        onSelectFile={handleImageFileSelect}
        onClose={closeImageModal}
        onSave={() => { void handleSaveImage(); }}
        disableSelect={isImageUploading || isImageSaving}
        disableClose={isImageSaving}
        disableSave={
          isImageSaving ||
          isImageUploading ||
          !modalImage?.uploadId ||
          modalImage.moderationStatus !== 'APPROVED'
        }
        saveLabel={isImageSaving ? 'Saving...' : 'Save Image'}
        statusText={imageStatusText}
        statusTone={imageStatusTone}
        statusReason={modalImage?.moderationReason}
        errorMessage={imageError}
      />
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

import { useEffect, useMemo, useState } from 'react';
import type {
  Announcement,
  AnnouncementStatus,
  AnnouncementPlacement,
  SaveAnnouncementParams,
} from '../announcementTypes';
import {
  adminCreateAnnouncement,
  adminDeleteAnnouncement,
  adminListAnnouncements,
  adminUpdateAnnouncement,
} from '../adminApi';

const ALL_STATUSES = [
  { value: '', label: 'All statuses' },
  { value: 'draft', label: 'Draft' },
  { value: 'published', label: 'Published' },
  { value: 'archived', label: 'Archived' },
] as const;

const PLACEMENT_OPTIONS: Array<{ value: AnnouncementPlacement; label: string }> = [
  { value: 'global', label: 'Global' },
  { value: 'home', label: 'Home' },
  { value: 'dashboard', label: 'Dashboard' },
  { value: 'news', label: 'News' },
];

function formatDateTime(value?: string): string {
  if (!value) return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '—';
  return date.toLocaleString();
}

function toDateTimeLocalValue(value?: string): string {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '';

  const offsetMs = date.getTimezoneOffset() * 60_000;
  return new Date(date.getTime() - offsetMs).toISOString().slice(0, 16);
}

function fromDateTimeLocalValue(value: string): string | undefined {
  if (!value.trim()) return undefined;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return undefined;
  return date.toISOString();
}

function placementLabel(placement: AnnouncementPlacement): string {
  return PLACEMENT_OPTIONS.find((option) => option.value === placement)?.label ?? placement;
}

function statusBadgeClass(status: AnnouncementStatus): string {
  switch (status) {
    case 'published':
      return 'bg-green-500/20 text-green-400';
    case 'archived':
      return 'bg-slate-500/20 text-slate-300';
    default:
      return 'bg-amber-500/20 text-amber-300';
  }
}

function buildInitialFormState(item?: Announcement | null): SaveAnnouncementParams {
  return {
    title: item?.title ?? '',
    body: item?.body ?? '',
    status: item?.status ?? 'draft',
    priority: item?.priority ?? 0,
    placements: item?.placements?.length ? item.placements : ['global'],
    audience: item?.audience ?? 'all',
    ctaLabel: item?.ctaLabel ?? '',
    ctaUrl: item?.ctaUrl ?? '',
    dismissible: item?.dismissible ?? true,
    startsAt: item?.startsAt,
    endsAt: item?.endsAt,
  };
}

function hasCallToAction(item?: Announcement | null): boolean {
  return Boolean(item?.ctaLabel?.trim() && item?.ctaUrl?.trim());
}

interface AnnouncementEditorModalProps {
  announcement: Announcement | null;
  isSaving: boolean;
  error: string | null;
  onClose: () => void;
  onSave: (params: SaveAnnouncementParams, existingId?: string) => void;
  onRequestDelete: (announcement: Announcement) => void;
}

function AnnouncementEditorModal({
  announcement,
  isSaving,
  error,
  onClose,
  onSave,
  onRequestDelete,
}: AnnouncementEditorModalProps) {
  const [form, setForm] = useState<SaveAnnouncementParams>(() => buildInitialFormState(announcement));
  const [callToActionEnabled, setCallToActionEnabled] = useState<boolean>(() => hasCallToAction(announcement));

  useEffect(() => {
    setForm(buildInitialFormState(announcement));
    setCallToActionEnabled(hasCallToAction(announcement));
  }, [announcement]);

  const isEditing = Boolean(announcement);

  return (
    <div className="fixed inset-0 z-30 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/70" onClick={onClose} aria-hidden="true" />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="announcement-editor-title"
        className="relative z-10 flex max-h-[90vh] w-full max-w-3xl flex-col overflow-hidden rounded-2xl border border-slate-700 bg-slate-800 shadow-2xl"
      >
        <div className="flex items-center justify-between border-b border-slate-700 px-6 py-4">
          <div>
            <h2 id="announcement-editor-title" className="text-lg font-semibold text-white">
              {isEditing ? 'Edit Announcement' : 'Create Announcement'}
            </h2>
            <p className="text-sm text-slate-400">Manage a first-party message shown across FlyingForge.</p>
          </div>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close announcement editor"
            className="rounded-lg p-2 text-slate-400 transition-colors hover:bg-slate-700 hover:text-white"
          >
            <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="space-y-4 overflow-y-auto px-6 py-5">
          {error && (
            <div className="rounded-lg border border-red-500/20 bg-red-500/10 px-4 py-3 text-sm text-red-300">
              {error}
            </div>
          )}

          <div className="grid gap-4 md:grid-cols-2">
            <label className="block">
              <span className="mb-1 block text-sm font-medium text-slate-200">Title</span>
              <input
                type="text"
                value={form.title}
                onChange={(event) => setForm((current) => ({ ...current, title: event.target.value }))}
                className="h-11 w-full rounded-lg border border-slate-700 bg-slate-900 px-3 text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
            </label>

            <label className="block">
              <span className="mb-1 block text-sm font-medium text-slate-200">Status</span>
              <select
                value={form.status}
                onChange={(event) => setForm((current) => ({ ...current, status: event.target.value as AnnouncementStatus }))}
                className="h-11 w-full rounded-lg border border-slate-700 bg-slate-900 px-3 text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
              >
                <option value="draft">Draft</option>
                <option value="published">Published</option>
                <option value="archived">Archived</option>
              </select>
            </label>
          </div>

          <label className="block">
            <span className="mb-1 block text-sm font-medium text-slate-200">Body</span>
            <textarea
              value={form.body}
              onChange={(event) => setForm((current) => ({ ...current, body: event.target.value }))}
              rows={5}
              className="w-full rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
            />
          </label>

          <div>
            <span className="mb-2 block text-sm font-medium text-slate-200">Placements</span>
            <div className="grid gap-2 sm:grid-cols-2">
              {PLACEMENT_OPTIONS.map((option) => {
                const checked = form.placements.includes(option.value);
                return (
                  <label key={option.value} className="flex items-center gap-3 rounded-lg border border-slate-700 bg-slate-900 px-3 py-2 text-sm text-slate-200">
                    <input
                      type="checkbox"
                      checked={checked}
                      onChange={(event) => {
                        setForm((current) => ({
                          ...current,
                          placements: event.target.checked
                            ? [...current.placements, option.value]
                            : current.placements.filter((placement) => placement !== option.value),
                        }));
                      }}
                      className="h-4 w-4 rounded border-slate-500 bg-slate-800 text-primary-500 focus:ring-primary-500"
                    />
                    {option.label}
                  </label>
                );
              })}
            </div>
          </div>

          <div className="grid gap-4 md:grid-cols-3">
            <label className="block">
              <span className="mb-1 block text-sm font-medium text-slate-200">Audience</span>
              <select
                value={form.audience}
                onChange={(event) => setForm((current) => ({ ...current, audience: event.target.value as SaveAnnouncementParams['audience'] }))}
                className="h-11 w-full rounded-lg border border-slate-700 bg-slate-900 px-3 text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
              >
                <option value="all">All visitors</option>
                <option value="signed_in">Signed in only</option>
                <option value="signed_out">Signed out only</option>
              </select>
            </label>

            <label className="block">
              <span className="mb-1 block text-sm font-medium text-slate-200">Priority</span>
              <input
                type="number"
                value={form.priority}
                onChange={(event) => setForm((current) => ({ ...current, priority: Number(event.target.value || 0) }))}
                className="h-11 w-full rounded-lg border border-slate-700 bg-slate-900 px-3 text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
            </label>

            <label className="flex items-center gap-3 rounded-lg border border-slate-700 bg-slate-900 px-3 py-3 text-sm text-slate-200">
              <input
                type="checkbox"
                checked={form.dismissible}
                onChange={(event) => setForm((current) => ({ ...current, dismissible: event.target.checked }))}
                className="h-4 w-4 rounded border-slate-500 bg-slate-800 text-primary-500 focus:ring-primary-500"
              />
              Allow users to dismiss
            </label>
          </div>

          <label className="flex items-center gap-3 rounded-lg border border-slate-700 bg-slate-900 px-3 py-3 text-sm text-slate-200">
            <input
              type="checkbox"
              checked={callToActionEnabled}
              onChange={(event) => setCallToActionEnabled(event.target.checked)}
              className="h-4 w-4 rounded border-slate-500 bg-slate-800 text-primary-500 focus:ring-primary-500"
            />
            Enable Call to Action
          </label>

          {callToActionEnabled && (
            <div className="grid gap-4 md:grid-cols-2">
              <label className="block">
                <span className="mb-1 block text-sm font-medium text-slate-200">Call to Action Label</span>
                <input
                  type="text"
                  value={form.ctaLabel ?? ''}
                  onChange={(event) => setForm((current) => ({ ...current, ctaLabel: event.target.value }))}
                  className="h-11 w-full rounded-lg border border-slate-700 bg-slate-900 px-3 text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
              </label>

              <label className="block">
                <span className="mb-1 block text-sm font-medium text-slate-200">Call to Action Link</span>
                <input
                  type="text"
                  value={form.ctaUrl ?? ''}
                  onChange={(event) => setForm((current) => ({ ...current, ctaUrl: event.target.value }))}
                  placeholder="/news or https://example.com"
                  className="h-11 w-full rounded-lg border border-slate-700 bg-slate-900 px-3 text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
                />
              </label>
            </div>
          )}

          <div className="grid gap-4 md:grid-cols-2">
            <label className="block">
              <span className="mb-1 block text-sm font-medium text-slate-200">Start Time</span>
              <input
                type="datetime-local"
                value={toDateTimeLocalValue(form.startsAt)}
                onChange={(event) => setForm((current) => ({ ...current, startsAt: fromDateTimeLocalValue(event.target.value) }))}
                className="h-11 w-full rounded-lg border border-slate-700 bg-slate-900 px-3 text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
            </label>

            <label className="block">
              <span className="mb-1 block text-sm font-medium text-slate-200">End Time</span>
              <input
                type="datetime-local"
                value={toDateTimeLocalValue(form.endsAt)}
                onChange={(event) => setForm((current) => ({ ...current, endsAt: fromDateTimeLocalValue(event.target.value) }))}
                className="h-11 w-full rounded-lg border border-slate-700 bg-slate-900 px-3 text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
            </label>
          </div>
        </div>

        <div className="flex items-center justify-between border-t border-slate-700 px-6 py-4">
          <div>
            {isEditing && announcement && (
              <button
                type="button"
                onClick={() => onRequestDelete(announcement)}
                className="rounded-lg border border-red-500/40 px-4 py-2 text-sm font-medium text-red-300 transition-colors hover:bg-red-500/10"
              >
                Delete
              </button>
            )}
          </div>
          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={onClose}
              className="rounded-lg px-4 py-2 text-sm font-medium text-slate-300 transition-colors hover:bg-slate-700 hover:text-white"
            >
              Cancel
            </button>
            <button
              type="button"
              disabled={isSaving}
              onClick={() =>
                onSave(
                  {
                    ...form,
                    ctaLabel: callToActionEnabled ? form.ctaLabel : '',
                    ctaUrl: callToActionEnabled ? form.ctaUrl : '',
                  },
                  announcement?.id,
                )
              }
              className="rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-700 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {isSaving ? 'Saving...' : isEditing ? 'Save Changes' : 'Create Announcement'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

export function AdminAnnouncementsPanel() {
  const [items, setItems] = useState<Announcement[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [query, setQuery] = useState('');
  const [appliedQuery, setAppliedQuery] = useState('');
  const [status, setStatus] = useState<AnnouncementStatus | ''>('');
  const [editingAnnouncement, setEditingAnnouncement] = useState<Announcement | null>(null);
  const [isCreating, setIsCreating] = useState(false);
  const [deleteCandidate, setDeleteCandidate] = useState<Announcement | null>(null);

  const loadAnnouncements = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const response = await adminListAnnouncements({
        query: appliedQuery || undefined,
        status: status || undefined,
        limit: 100,
        offset: 0,
      });
      setItems(response.announcements ?? []);
      setTotalCount(response.totalCount ?? response.announcements?.length ?? 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load announcements');
      setItems([]);
      setTotalCount(0);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    void loadAnnouncements();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [appliedQuery, status]);

  const openCreateModal = () => {
    setError(null);
    setEditingAnnouncement(null);
    setIsCreating(true);
  };

  const openEditModal = (announcement: Announcement) => {
    setError(null);
    setEditingAnnouncement(announcement);
    setIsCreating(false);
  };

  const closeModal = () => {
    setEditingAnnouncement(null);
    setIsCreating(false);
    setError(null);
  };

  const closeDeleteDialog = () => {
    setDeleteCandidate(null);
  };

  const handleSave = async (params: SaveAnnouncementParams, existingId?: string) => {
    setIsSaving(true);
    setError(null);
    try {
      const trimmedParams: SaveAnnouncementParams = {
        ...params,
        ctaLabel: params.ctaLabel?.trim() || undefined,
        ctaUrl: params.ctaUrl?.trim() || undefined,
      };

      if (existingId) {
        await adminUpdateAnnouncement(existingId, trimmedParams);
      } else {
        await adminCreateAnnouncement(trimmedParams);
      }
      await loadAnnouncements();
      closeModal();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save announcement');
    } finally {
      setIsSaving(false);
    }
  };

  const handleDelete = async (announcement: Announcement) => {
    setIsSaving(true);
    setError(null);
    try {
      await adminDeleteAnnouncement(announcement.id);
      await loadAnnouncements();
      closeDeleteDialog();
      closeModal();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete announcement');
    } finally {
      setIsSaving(false);
    }
  };

  const activeModalAnnouncement = isCreating ? null : editingAnnouncement;
  const visibleModal = isCreating || editingAnnouncement !== null;

  const countLabel = useMemo(
    () => `${totalCount} announcement${totalCount === 1 ? '' : 's'} found`,
    [totalCount],
  );

  return (
    <>
      <div className="rounded-xl border border-slate-800 bg-slate-900/40">
        <div className="border-b border-slate-800 px-4 py-4 md:px-5">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h2 className="text-lg font-semibold text-white">Announcements</h2>
              <p className="text-sm text-slate-400">Create site-wide or section-specific product updates.</p>
            </div>
            <button
              type="button"
              onClick={openCreateModal}
              className="inline-flex items-center justify-center gap-2 rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-700"
            >
              <span aria-hidden="true">+</span>
              New Announcement
            </button>
          </div>

          <div className="mt-4 flex flex-col gap-3 lg:flex-row lg:items-center">
            <div className="flex flex-1 gap-2">
              <input
                type="text"
                placeholder="Search announcements..."
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') {
                    setAppliedQuery(query.trim());
                  }
                }}
                className="h-11 w-full rounded-lg border border-slate-700 bg-slate-900 px-3 text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-500"
              />
              <button
                type="button"
                onClick={() => setAppliedQuery(query.trim())}
                className="rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-700"
              >
                Search
              </button>
              {(query || appliedQuery) && (
                <button
                  type="button"
                  onClick={() => {
                    setQuery('');
                    setAppliedQuery('');
                  }}
                  className="rounded-lg px-4 py-2 text-sm font-medium text-slate-300 transition-colors hover:bg-slate-700 hover:text-white"
                >
                  Clear
                </button>
              )}
            </div>

            <select
              value={status}
              onChange={(event) => setStatus(event.target.value as AnnouncementStatus | '')}
              className="h-11 min-w-[180px] rounded-lg border border-slate-700 bg-slate-900 px-3 text-white focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              {ALL_STATUSES.map((option) => (
                <option key={option.label} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </div>

          <p className="mt-3 text-sm text-slate-400">{countLabel}</p>
          {error && <div className="mt-3 rounded-lg border border-red-500/20 bg-red-500/10 px-4 py-3 text-sm text-red-300">{error}</div>}
        </div>

        <div className="p-4 md:p-5">
          {isLoading ? (
            <div className="p-8 text-center">
              <div className="mx-auto h-8 w-8 animate-spin rounded-full border-2 border-primary-500/30 border-t-primary-500" />
              <p className="mt-4 text-slate-400">Loading announcements...</p>
            </div>
          ) : items.length === 0 ? (
            <div className="rounded-xl border border-slate-800 bg-slate-900/40 p-8 text-center">
              <p className="text-slate-400">No announcements found</p>
            </div>
          ) : (
            <>
              <div className="hidden overflow-hidden rounded-xl border border-slate-800 md:block">
                <table className="w-full text-sm">
                  <thead className="bg-slate-900 text-slate-400">
                    <tr className="border-b border-slate-800">
                      <th className="px-4 py-3 text-left font-medium">Title</th>
                      <th className="px-4 py-3 text-left font-medium">Placements</th>
                      <th className="px-4 py-3 text-left font-medium">Audience</th>
                      <th className="px-4 py-3 text-left font-medium">Status</th>
                      <th className="px-4 py-3 text-left font-medium">Schedule</th>
                      <th className="px-4 py-3 text-left font-medium">Priority</th>
                    </tr>
                  </thead>
                  <tbody>
                    {items.map((announcement) => (
                      <tr
                        key={announcement.id}
                        role="button"
                        tabIndex={0}
                        aria-label={`Open editor for ${announcement.title}`}
                        onClick={() => openEditModal(announcement)}
                        onKeyDown={(event) => {
                          if (event.key === 'Enter' || event.key === ' ') {
                            event.preventDefault();
                            openEditModal(announcement);
                          }
                        }}
                        className="cursor-pointer border-t border-slate-800 bg-slate-900/40 transition-colors hover:bg-slate-800/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-inset"
                      >
                        <td className="px-4 py-3 text-white">
                          <div className="font-medium">{announcement.title}</div>
                          <div className="line-clamp-1 text-xs text-slate-400">{announcement.body}</div>
                        </td>
                        <td className="px-4 py-3 text-slate-300">{announcement.placements.map(placementLabel).join(', ')}</td>
                        <td className="px-4 py-3 text-slate-300">{announcement.audience.replace('_', ' ')}</td>
                        <td className="px-4 py-3">
                          <span className={`rounded px-2 py-0.5 text-xs ${statusBadgeClass(announcement.status)}`}>{announcement.status}</span>
                        </td>
                        <td className="px-4 py-3 text-slate-300">
                          <div>{formatDateTime(announcement.startsAt)}</div>
                          <div className="text-xs text-slate-500">to {formatDateTime(announcement.endsAt)}</div>
                        </td>
                        <td className="px-4 py-3 text-slate-300">{announcement.priority}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>

              <div className="space-y-3 md:hidden">
                {items.map((announcement) => (
                  <button
                    key={announcement.id}
                    type="button"
                    onClick={() => openEditModal(announcement)}
                    className="w-full rounded-xl border border-slate-700 bg-slate-800/50 p-4 text-left transition hover:border-primary-500/50 hover:bg-slate-800"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <h3 className="truncate text-base font-semibold text-white">{announcement.title}</h3>
                        <p className="mt-1 line-clamp-2 text-sm text-slate-300">{announcement.body}</p>
                      </div>
                      <span className={`shrink-0 rounded-full px-2 py-1 text-xs font-medium ${statusBadgeClass(announcement.status)}`}>
                        {announcement.status}
                      </span>
                    </div>
                    <div className="mt-3 text-xs text-slate-400">
                      {announcement.placements.map(placementLabel).join(', ')} • {announcement.audience.replace('_', ' ')}
                    </div>
                    <div className="mt-1 text-xs text-slate-500">
                      {formatDateTime(announcement.startsAt)} → {formatDateTime(announcement.endsAt)}
                    </div>
                  </button>
                ))}
              </div>
            </>
          )}
        </div>
      </div>

      {visibleModal && (
        <AnnouncementEditorModal
          announcement={activeModalAnnouncement}
          isSaving={isSaving}
          error={error}
          onClose={closeModal}
          onSave={handleSave}
          onRequestDelete={setDeleteCandidate}
        />
      )}

      {deleteCandidate && (
        <div className="fixed inset-0 z-40 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/70" onClick={closeDeleteDialog} aria-hidden="true" />
          <div
            role="dialog"
            aria-modal="true"
            aria-labelledby="delete-announcement-dialog-title"
            aria-describedby="delete-announcement-dialog-description"
            className="relative z-10 w-full max-w-md rounded-2xl border border-red-500/40 bg-slate-800 p-6 shadow-2xl"
          >
            <div className="mb-4 flex items-start justify-between gap-3">
              <div>
                <h3 id="delete-announcement-dialog-title" className="text-lg font-semibold text-white">
                  Delete Announcement?
                </h3>
                <p id="delete-announcement-dialog-description" className="mt-2 text-sm text-slate-300">
                  This will permanently delete <span className="font-medium text-white">{deleteCandidate.title}</span>.
                  This action cannot be undone.
                </p>
              </div>
              <button
                type="button"
                onClick={closeDeleteDialog}
                disabled={isSaving}
                aria-label="Close delete announcement modal"
                className="rounded-lg p-2 text-slate-400 transition-colors hover:bg-slate-700 hover:text-white disabled:opacity-50"
              >
                <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <div className="flex items-center justify-end gap-3">
              <button
                type="button"
                onClick={closeDeleteDialog}
                disabled={isSaving}
                className="rounded-lg px-4 py-2 text-sm font-medium text-slate-300 transition-colors hover:bg-slate-700 hover:text-white disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={() => void handleDelete(deleteCandidate)}
                disabled={isSaving}
                className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-red-500 disabled:cursor-not-allowed disabled:opacity-60"
              >
                {isSaving ? 'Deleting...' : 'Delete Announcement'}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

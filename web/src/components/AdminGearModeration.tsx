import { useState, useEffect, useCallback, useRef, type FormEvent } from 'react';
import type { GearCatalogItem, GearType, ImageStatusFilter, AdminUpdateGearCatalogParams, DroneType, CatalogItemStatus } from '../gearCatalogTypes';
import { GEAR_TYPES, DRONE_TYPES } from '../gearCatalogTypes';
import type { Build, BuildStatus, BuildValidationError } from '../buildTypes';
import {
  adminSearchGear,
  adminUpdateGear,
  adminSaveGearImageUpload,
  adminDeleteGearImage,
  adminDeleteGear,
  adminBulkDeleteGear,
  adminGetGear,
  adminSearchBuilds,
  adminGetBuild,
  adminUpdateBuild,
  adminPublishBuild,
  adminUploadBuildImage,
  adminDeleteBuildImage,
  getAdminGearImageUrl,
  getAdminBuildImageUrl,
} from '../adminApi';
import { moderateGearCatalogImageUpload } from '../gearCatalogApi';
import { CatalogSearchModal } from './CatalogSearchModal';
import { MobileFloatingControls } from './MobileFloatingControls';
import { ImageUploadModal } from './ImageUploadModal';

interface AdminGearModerationProps {
  hasContentAdminAccess: boolean;
  authLoading?: boolean;
}

type ModerationTab = 'gear' | 'builds';
type BuildModerationStatus = 'PENDING_REVIEW' | 'DRAFT' | 'PUBLISHED' | 'UNPUBLISHED';

function formatDate(value?: string): string {
  if (!value) return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '—';
  return date.toLocaleDateString();
}

function formatDateTime(value?: string): string {
  if (!value) return '—';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '—';
  return date.toLocaleString();
}

function getImageStatusLabel(status: GearCatalogItem['imageStatus']): string {
  switch (status) {
    case 'approved':
      return 'Approved';
    case 'scanned':
      return 'Scanned';
    default:
      return 'Missing';
  }
}

function getImageStatusClass(status: GearCatalogItem['imageStatus']): string {
  switch (status) {
    case 'approved':
      return 'bg-green-500/20 text-green-400';
    case 'scanned':
      return 'bg-blue-500/20 text-blue-400';
    default:
      return 'bg-yellow-500/20 text-yellow-400';
  }
}

function getImageStatusTextClass(status: GearCatalogItem['imageStatus']): string {
  switch (status) {
    case 'approved':
      return 'text-green-400';
    case 'scanned':
      return 'text-blue-400';
    default:
      return 'text-yellow-400';
  }
}

function getCatalogStatusLabel(status: CatalogItemStatus): string {
  switch (status) {
    case 'published':
      return 'Published';
    case 'pending':
      return 'Pending';
    case 'removed':
      return 'Removed';
    default:
      return status;
  }
}

function getCatalogStatusClass(status: CatalogItemStatus): string {
  switch (status) {
    case 'published':
      return 'bg-green-500/20 text-green-400';
    case 'pending':
      return 'bg-amber-500/20 text-amber-300';
    case 'removed':
      return 'bg-red-500/20 text-red-400';
    default:
      return 'bg-slate-500/20 text-slate-300';
  }
}

function getCatalogStatusTextClass(status: CatalogItemStatus): string {
  switch (status) {
    case 'published':
      return 'text-green-400';
    case 'pending':
      return 'text-amber-300';
    case 'removed':
      return 'text-red-400';
    default:
      return 'text-slate-300';
  }
}

function getBuildStatusLabel(status: BuildStatus): string {
  switch (status) {
    case 'PENDING_REVIEW':
      return 'Pending Review';
    case 'PUBLISHED':
      return 'Published';
    case 'UNPUBLISHED':
      return 'Unpublished';
    case 'DRAFT':
      return 'Draft';
    case 'SHARED':
      return 'Shared';
    case 'TEMP':
      return 'Temp';
    default:
      return status;
  }
}

function getBuildStatusClass(status: BuildStatus): string {
  switch (status) {
    case 'PENDING_REVIEW':
      return 'bg-amber-500/20 text-amber-300';
    case 'PUBLISHED':
      return 'bg-green-500/20 text-green-400';
    case 'UNPUBLISHED':
      return 'bg-red-500/20 text-red-400';
    case 'DRAFT':
      return 'bg-slate-500/20 text-slate-300';
    case 'SHARED':
      return 'bg-blue-500/20 text-blue-300';
    case 'TEMP':
      return 'bg-slate-500/20 text-slate-300';
    default:
      return 'bg-slate-500/20 text-slate-300';
  }
}

export function AdminGearModeration({ hasContentAdminAccess, authLoading }: AdminGearModerationProps) {
  const [activeTab, setActiveTab] = useState<ModerationTab>('gear');

  const [items, setItems] = useState<GearCatalogItem[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedItemIds, setSelectedItemIds] = useState<Set<string>>(() => new Set());
  const [isBulkEditMode, setIsBulkEditMode] = useState(false);

  // Bulk delete state
  const [showBulkDeleteConfirm, setShowBulkDeleteConfirm] = useState(false);
  const [bulkDeleteConfirmText, setBulkDeleteConfirmText] = useState('');
  const [bulkDeleteError, setBulkDeleteError] = useState<string | null>(null);
  const [bulkDeleteStatus, setBulkDeleteStatus] = useState<string | null>(null);
  const [isBulkDeleting, setIsBulkDeleting] = useState(false);

  // Filters
  const [query, setQuery] = useState('');
  const [appliedQuery, setAppliedQuery] = useState('');
  const [gearType, setGearType] = useState<GearType | ''>('');
  const [catalogStatus, setCatalogStatus] = useState<CatalogItemStatus | ''>('pending');
  const [imageStatus, setImageStatus] = useState<ImageStatusFilter | ''>('all'); // Default to all records
  const pageSize = 30;
  const [hasMore, setHasMore] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const loadMoreRef = useRef<HTMLDivElement>(null);
  
  // Use refs to track current offset and prevent race conditions
  const currentOffsetRef = useRef(0);
  const isLoadingRef = useRef(false);
  const latestLoadRequestRef = useRef(0);

  // Edit modal state - modalKey forces remount to fetch fresh data
  const [editingItemId, setEditingItemId] = useState<string | null>(null);
  const [modalKey, setModalKey] = useState(0);
  const [showAddGearModal, setShowAddGearModal] = useState(false);
  const [isMobileControlsOpen, setIsMobileControlsOpen] = useState(false);

  // Build moderation list.
  const [builds, setBuilds] = useState<Build[]>([]);
  const [buildTotalCount, setBuildTotalCount] = useState(0);
  const [isLoadingBuilds, setIsLoadingBuilds] = useState(false);
  const [buildError, setBuildError] = useState<string | null>(null);
  const [buildQuery, setBuildQuery] = useState('');
  const [appliedBuildQuery, setAppliedBuildQuery] = useState('');
  const [buildStatus, setBuildStatus] = useState<BuildModerationStatus>('PENDING_REVIEW');
  const [editingBuildId, setEditingBuildId] = useState<string | null>(null);
  const [buildModalKey, setBuildModalKey] = useState(0);

  const selectedCount = selectedItemIds.size;
  const selectedItems = items.filter((item) => selectedItemIds.has(item.id));
  const hasSelectedInUse = selectedItems.some((item) => item.usageCount > 0);
  const selectedInUseCount = selectedItems.filter((item) => item.usageCount > 0).length;
  const isAllSelected = items.length > 0 && items.every((item) => selectedItemIds.has(item.id));

  const clearSelection = useCallback(() => {
    setSelectedItemIds(new Set());
  }, []);

  const toggleSelection = useCallback((id: string) => {
    setSelectedItemIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  const selectAllLoaded = useCallback(() => {
    setSelectedItemIds(new Set(items.map((item) => item.id)));
  }, [items]);

  const handleEnterBulkEditMode = useCallback(() => {
    setIsBulkEditMode(true);
    setSelectedItemIds(new Set());
  }, []);

  const handleExitBulkEditMode = useCallback(() => {
    if (isBulkDeleting) return;
    setIsBulkEditMode(false);
    setSelectedItemIds(new Set());
    setShowBulkDeleteConfirm(false);
    setBulkDeleteConfirmText('');
  }, [isBulkDeleting]);

  const loadItems = useCallback(async (reset = false, forceRefresh = false) => {
    if (!hasContentAdminAccess) return;
    
    // Prevent concurrent loads by default; allow forced resets to supersede in-flight loads.
    if (isLoadingRef.current && !(reset && forceRefresh)) return;
    isLoadingRef.current = true;
    const requestId = ++latestLoadRequestRef.current;

    if (reset) {
      setIsLoading(true);
      currentOffsetRef.current = 0;
      setSelectedItemIds(new Set());
      setShowBulkDeleteConfirm(false);
      setBulkDeleteConfirmText('');
      setBulkDeleteError(null);
      setBulkDeleteStatus(null);
    } else {
      setIsLoadingMore(true);
    }
    setError(null);

    const offset = currentOffsetRef.current;

    try {
      const response = await adminSearchGear({
        query: appliedQuery || undefined,
        gearType: gearType || undefined,
        status: catalogStatus || undefined,
        imageStatus: imageStatus || undefined,
        limit: pageSize,
        offset: offset,
      });

      // Ignore stale responses from superseded requests.
      if (requestId !== latestLoadRequestRef.current) {
        return;
      }
      
      if (reset) {
        setItems(response.items);
      } else {
        setItems(prev => [...prev, ...response.items]);
      }
      currentOffsetRef.current = offset + response.items.length;
      setTotalCount(response.totalCount);
      setHasMore(response.items.length === pageSize && currentOffsetRef.current < response.totalCount);
    } catch (err) {
      if (requestId !== latestLoadRequestRef.current) {
        return;
      }
      setError(err instanceof Error ? err.message : 'Failed to load gear items');
    } finally {
      if (requestId === latestLoadRequestRef.current) {
        setIsLoading(false);
        setIsLoadingMore(false);
        isLoadingRef.current = false;
      }
    }
  }, [hasContentAdminAccess, appliedQuery, gearType, catalogStatus, imageStatus]);

  const loadBuilds = useCallback(async () => {
    if (!hasContentAdminAccess) return;
    setIsLoadingBuilds(true);
    setBuildError(null);
    try {
      const response = await adminSearchBuilds({
        query: appliedBuildQuery || undefined,
        status: buildStatus,
        limit: 100,
        offset: 0,
      });
      setBuilds(response.builds ?? []);
      setBuildTotalCount(response.totalCount ?? response.builds?.length ?? 0);
    } catch (err) {
      setBuildError(err instanceof Error ? err.message : 'Failed to load builds');
    } finally {
      setIsLoadingBuilds(false);
    }
  }, [hasContentAdminAccess, appliedBuildQuery, buildStatus]);

  useEffect(() => {
    if (activeTab !== 'gear') {
      setIsBulkEditMode(false);
      setSelectedItemIds(new Set());
      setShowBulkDeleteConfirm(false);
      setBulkDeleteConfirmText('');
    }
  }, [activeTab]);

  const handleOpenBulkDeleteConfirm = useCallback(() => {
    if (selectedCount === 0 || isBulkDeleting || isLoading || isLoadingMore) return;
    setBulkDeleteError(null);
    setBulkDeleteStatus(null);
    setBulkDeleteConfirmText('');
    setShowBulkDeleteConfirm(true);
  }, [selectedCount, isBulkDeleting, isLoading, isLoadingMore]);

  const handleCancelBulkDelete = useCallback(() => {
    if (isBulkDeleting) return;
    setShowBulkDeleteConfirm(false);
    setBulkDeleteConfirmText('');
  }, [isBulkDeleting]);

  const handleConfirmBulkDelete = useCallback(async () => {
    if (isBulkDeleting) return;

    const trimmed = bulkDeleteConfirmText.trim().toUpperCase();
    if (trimmed !== 'DELETE') return;

    const idsToDelete = Array.from(selectedItemIds);
    if (idsToDelete.length === 0) {
      setShowBulkDeleteConfirm(false);
      return;
    }

    setIsBulkDeleting(true);
    setBulkDeleteError(null);
    setBulkDeleteStatus(null);

    try {
      const response = await adminBulkDeleteGear(idsToDelete);
      setShowBulkDeleteConfirm(false);
      setBulkDeleteConfirmText('');
      setSelectedItemIds(new Set());

      await loadItems(true, true);

      setBulkDeleteStatus(
        `Deleted ${response.deletedCount} item${response.deletedCount === 1 ? '' : 's'}${response.notFoundCount ? ` (${response.notFoundCount} not found)` : ''}.`
      );
    } catch (err) {
      setBulkDeleteError(err instanceof Error ? err.message : 'Failed to bulk delete items');
    } finally {
      setIsBulkDeleting(false);
    }
  }, [isBulkDeleting, bulkDeleteConfirmText, selectedItemIds, loadItems]);

  // Initial load and auto-search when gear filters change.
  useEffect(() => {
    if (!hasContentAdminAccess) return;
    void loadItems(true);
  }, [hasContentAdminAccess, loadItems]);

  // Initial load and auto-search when build filters change.
  useEffect(() => {
    if (!hasContentAdminAccess) return;
    void loadBuilds();
  }, [hasContentAdminAccess, loadBuilds]);

  const handleGearSearch = useCallback(() => {
    setIsMobileControlsOpen(false);
    setAppliedQuery(query);
  }, [query]);

  const handleBuildSearch = useCallback(() => {
    setIsMobileControlsOpen(false);
    setAppliedBuildQuery(buildQuery);
  }, [buildQuery]);

  // Handle enter key in search input
  const handleGearKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleGearSearch();
    }
  };

  const handleBuildKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleBuildSearch();
    }
  };

  // Handle clearing search
  const handleGearClearSearch = () => {
    setQuery('');
    setAppliedQuery('');
  };

  const handleBuildClearSearch = () => {
    setBuildQuery('');
    setAppliedBuildQuery('');
  };

  // Infinite scroll observer
  // Note: loadItems prevents concurrent calls by default (except forced reset refreshes),
  // so we don't need to check loading state here - just trigger on intersection.
  useEffect(() => {
    if (activeTab !== 'gear') return;

    const element = loadMoreRef.current;
    if (!element || !hasMore || isLoading) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting) {
          loadItems(false);
        }
      },
      { threshold: 0.1 }
    );

    observer.observe(element);
    return () => observer.disconnect();
  }, [activeTab, hasMore, isLoading, loadItems]);

  const handleEditClick = (item: GearCatalogItem) => {
    setModalKey(k => k + 1); // Force modal remount to fetch fresh data
    setEditingItemId(item.id);
  };

  const handleEditClose = () => {
    setEditingItemId(null);
  };

  const handleEditSave = () => {
    // Refresh the list after saving
    setEditingItemId(null);
    loadItems(true);
  };

  const handleEditDelete = useCallback(() => {
    setEditingItemId(null);
    void loadItems(true, true);
  }, [loadItems]);

  const handleAddGearClick = () => {
    setShowAddGearModal(true);
  };

  const handleAddGearClose = () => {
    setShowAddGearModal(false);
  };

  const handleAddGearSelect = useCallback(() => {
    // Close create modal after successful add/select and refresh the list.
    setShowAddGearModal(false);
    void loadItems(true);
  }, [loadItems]);

  const handleBuildEditClick = useCallback((build: Build) => {
    setBuildModalKey((prev) => prev + 1);
    setEditingBuildId(build.id);
  }, []);

  const handleBuildEditClose = useCallback(() => {
    setEditingBuildId(null);
  }, []);

  const handleBuildEditSaved = useCallback(() => {
    setEditingBuildId(null);
    void loadBuilds();
  }, [loadBuilds]);

  const handleBuildPublished = useCallback(() => {
    setEditingBuildId(null);
    void loadBuilds();
  }, [loadBuilds]);

  // Show loading while auth state is being determined
  if (authLoading) {
    return (
      <div className="p-8 text-center">
        <div className="w-8 h-8 border-2 border-primary-500/30 border-t-primary-500 rounded-full animate-spin mx-auto" />
        <p className="text-slate-400 mt-4">Loading...</p>
      </div>
    );
  }

  if (!hasContentAdminAccess) {
    return (
      <div className="p-8 text-center">
        <h1 className="text-2xl font-bold text-red-400 mb-4">Access Denied</h1>
        <p className="text-slate-400">You must be an admin or content admin to access this page.</p>
      </div>
    );
  }

  const gearControls = (
    <>
      <div className="flex flex-col gap-3">
        <div className="flex flex-col sm:flex-row gap-2">
          <div className="relative flex-1 min-w-0">
            <svg
              className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
              />
            </svg>
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={handleGearKeyDown}
              placeholder="Search brand or model..."
              className="w-full h-11 pl-10 pr-4 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
            />
          </div>
          <button
            onClick={handleGearSearch}
            className="w-full sm:w-auto shrink-0 px-3 py-2 bg-primary-600 hover:bg-primary-700 text-white text-sm font-medium rounded-lg transition-colors"
          >
            Search
          </button>
          {appliedQuery && (
            <button
              onClick={handleGearClearSearch}
              className="w-full sm:w-auto shrink-0 px-3 py-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
            >
              Clear
            </button>
          )}
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2">
          <select
            value={gearType}
            onChange={(e) => setGearType(e.target.value as GearType | '')}
            className="w-full min-w-0 h-11 px-3 bg-slate-900 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
          >
            <option value="">All Types</option>
            {GEAR_TYPES.map((type) => (
              <option key={type.value} value={type.value}>
                {type.label}
              </option>
            ))}
          </select>

          <select
            value={catalogStatus}
            onChange={(e) => setCatalogStatus(e.target.value as CatalogItemStatus | '')}
            className="w-full min-w-0 h-11 px-3 bg-slate-900 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
          >
            <option value="">All Statuses</option>
            <option value="pending">Pending</option>
            <option value="published">Published</option>
            <option value="removed">Removed</option>
          </select>

          <select
            value={imageStatus}
            onChange={(e) => setImageStatus(e.target.value as ImageStatusFilter | '')}
            className="w-full min-w-0 h-11 px-3 bg-slate-900 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
          >
            <option value="">Needs Work</option>
            <option value="all">All Records</option>
            <option value="missing">Needs Image</option>
            <option value="scanned">Scanned (Needs Review)</option>
            <option value="approved">Has Image</option>
            <option value="recently-curated">Recently Updated (24h)</option>
          </select>
        </div>

        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
          <p className="text-slate-400 text-sm">
            {totalCount} item{totalCount !== 1 ? 's' : ''} found
            {isBulkEditMode && (
              <span className="ml-2 text-slate-500">• {selectedCount} selected</span>
            )}
          </p>
          <div className="flex flex-col sm:flex-row gap-2">
            {!isBulkEditMode ? (
              <>
                <button
                  type="button"
                  onClick={handleEnterBulkEditMode}
                  disabled={isLoading || items.length === 0}
                  className="w-full sm:w-auto px-3 py-2 bg-slate-700 hover:bg-slate-600 border border-slate-600 rounded-lg text-sm text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Bulk Edit
                </button>
                <button
                  type="button"
                  onClick={handleAddGearClick}
                  className="w-full sm:w-auto px-3 py-2 bg-primary-600 hover:bg-primary-700 text-white text-sm font-medium rounded-lg transition-colors flex items-center justify-center gap-2"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                  </svg>
                  Add Gear
                </button>
              </>
            ) : (
              <>
                <button
                  type="button"
                  onClick={selectAllLoaded}
                  disabled={isBulkDeleting || items.length === 0 || isAllSelected}
                  className="w-full sm:w-auto px-3 py-2 bg-slate-700 hover:bg-slate-600 border border-slate-600 rounded-lg text-sm text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Select All
                </button>
                <button
                  type="button"
                  onClick={clearSelection}
                  disabled={isBulkDeleting || selectedCount === 0}
                  className="w-full sm:w-auto px-3 py-2 text-slate-300 hover:text-white hover:bg-slate-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Clear Selection
                </button>
                <button
                  type="button"
                  onClick={handleOpenBulkDeleteConfirm}
                  disabled={isBulkDeleting || selectedCount === 0}
                  className="w-full sm:w-auto px-3 py-2 bg-red-600/80 hover:bg-red-600 disabled:opacity-50 disabled:cursor-not-allowed text-white text-sm font-medium rounded-lg transition-colors"
                >
                  Delete Selected ({selectedCount})
                </button>
                <button
                  type="button"
                  onClick={handleExitBulkEditMode}
                  disabled={isBulkDeleting}
                  className="w-full sm:w-auto px-3 py-2 bg-slate-700 hover:bg-slate-600 border border-slate-600 rounded-lg text-sm text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Done
                </button>
              </>
            )}
          </div>
        </div>
      </div>

      {bulkDeleteStatus && (
        <div className="mt-3 p-3 bg-green-500/10 border border-green-500/20 rounded-lg text-green-300 text-sm">
          {bulkDeleteStatus}
        </div>
      )}

      {bulkDeleteError && (
        <div className="mt-3 p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">
          {bulkDeleteError}
        </div>
      )}

      {error && (
        <div className="mt-3 p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">
          {error}
        </div>
      )}
    </>
  );

  const buildControls = (
    <>
      <div className="flex flex-col gap-3">
        <div className="flex flex-col sm:flex-row gap-2">
          <div className="relative flex-1 min-w-0">
            <svg
              className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
              />
            </svg>
            <input
              type="text"
              value={buildQuery}
              onChange={(e) => setBuildQuery(e.target.value)}
              onKeyDown={handleBuildKeyDown}
              placeholder="Search build title, description, or pilot..."
              className="w-full h-11 pl-10 pr-4 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
            />
          </div>
          <button
            onClick={handleBuildSearch}
            className="w-full sm:w-auto shrink-0 px-3 py-2 bg-primary-600 hover:bg-primary-700 text-white text-sm font-medium rounded-lg transition-colors"
          >
            Search
          </button>
          {appliedBuildQuery && (
            <button
              onClick={handleBuildClearSearch}
              className="w-full sm:w-auto shrink-0 px-3 py-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
            >
              Clear
            </button>
          )}
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
          <select
            value={buildStatus}
            onChange={(e) => setBuildStatus(e.target.value as BuildModerationStatus)}
            className="w-full min-w-0 h-11 px-3 bg-slate-900 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
          >
            <option value="PENDING_REVIEW">Pending Review</option>
            <option value="PUBLISHED">Published</option>
            <option value="DRAFT">Draft</option>
            <option value="UNPUBLISHED">Unpublished</option>
          </select>
        </div>

        <p className="text-slate-400 text-sm">
          {buildTotalCount} build{buildTotalCount !== 1 ? 's' : ''} found
        </p>
      </div>

      {buildError && (
        <div className="mt-3 p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">
          {buildError}
        </div>
      )}
    </>
  );

  const controls = (
    <div className="bg-slate-800 border-b border-slate-700 px-4 md:px-6 py-3 md:py-4">
      <h1 className="text-lg md:text-2xl font-bold text-white mb-3">Content Moderation</h1>
      <div className="mb-3 inline-flex rounded-lg border border-slate-700 bg-slate-900/60 p-1">
        <button
          type="button"
          onClick={() => setActiveTab('gear')}
          className={`rounded-md px-4 py-2 text-sm font-medium transition-colors ${
            activeTab === 'gear'
              ? 'bg-primary-600 text-white'
              : 'text-slate-300 hover:bg-slate-700 hover:text-white'
          }`}
        >
          Gear
        </button>
        <button
          type="button"
          onClick={() => setActiveTab('builds')}
          className={`rounded-md px-4 py-2 text-sm font-medium transition-colors ${
            activeTab === 'builds'
              ? 'bg-primary-600 text-white'
              : 'text-slate-300 hover:bg-slate-700 hover:text-white'
          }`}
        >
          Builds
        </button>
      </div>
      {activeTab === 'gear' ? gearControls : buildControls}
    </div>
  );

  return (
    <>
      {/* Main flex container - matches news section pattern */}
      <div className="relative flex-1 flex flex-col min-h-0 overflow-hidden">
        <div className="hidden md:block flex-shrink-0 z-10 bg-slate-900">
          {controls}
        </div>

      {/* Scrollable list */}
      <div
        className="flex-1 overflow-y-auto min-h-0 px-4 md:px-6 pt-24 md:pt-6 pb-20"
        onScroll={() => {
          setIsMobileControlsOpen((prev) => (prev ? false : prev));

          // Dismiss keyboard on scroll for mobile
          if (document.activeElement instanceof HTMLElement) {
            document.activeElement.blur();
          }
        }}
      >
        {activeTab === 'gear' ? (
          <>
            {/* Gear table - desktop */}
            <div className="hidden md:block border border-slate-800 rounded-xl overflow-hidden bg-slate-900/40">
              {isLoading ? (
                <div className="p-8 text-center">
                  <div className="w-8 h-8 border-2 border-primary-500/30 border-t-primary-500 rounded-full animate-spin mx-auto" />
                  <p className="text-slate-400 mt-4">Loading...</p>
                </div>
              ) : items.length === 0 ? (
                <div className="p-8 text-center">
                  <p className="text-slate-400">No items found</p>
                </div>
              ) : (
                <table className="w-full text-sm">
                  <thead className="sticky top-0 z-10 bg-slate-900 text-slate-400">
                    <tr className="border-b border-slate-800">
                      <th className="px-4 py-3 text-left font-medium">Upload Date</th>
                      <th className="px-4 py-3 text-left font-medium">Last Edit</th>
                      <th className="px-4 py-3 text-left font-medium">Type</th>
                      <th className="px-4 py-3 text-left font-medium">Brand</th>
                      <th className="px-4 py-3 text-left font-medium">Model</th>
                      <th className="px-4 py-3 text-left font-medium">Variant</th>
                      <th className="px-4 py-3 text-left font-medium">Image</th>
                      <th className="px-4 py-3 text-left font-medium">Status</th>
                    </tr>
                  </thead>
                  <tbody>
                    {items.map((item) => {
                      const displayName = `${item.brand} ${item.model}${item.variant ? ` ${item.variant}` : ''}`.trim();
                      const isEditing = editingItemId === item.id;
                      const isSelectedForBulkDelete = isBulkEditMode && selectedItemIds.has(item.id);
                      const ariaLabel = isBulkEditMode ? `Select ${displayName}` : `Open editor for ${displayName}`;
                      return (
                        <tr
                          key={item.id}
                          onClick={() => {
                            if (isBulkEditMode) {
                              toggleSelection(item.id);
                            } else {
                              handleEditClick(item);
                            }
                          }}
                          onKeyDown={(event) => {
                            if (event.target instanceof HTMLElement && event.target.tagName === 'INPUT') {
                              return;
                            }
                            if (event.key === 'Enter' || event.key === ' ') {
                              event.preventDefault();
                              if (isBulkEditMode) {
                                toggleSelection(item.id);
                              } else {
                                handleEditClick(item);
                              }
                            }
                          }}
                          role="button"
                          tabIndex={0}
                          aria-label={ariaLabel}
                          aria-pressed={isBulkEditMode ? isSelectedForBulkDelete : undefined}
                          className={`border-t border-slate-800 transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-inset focus-visible:bg-primary-600/20 ${
                            isEditing
                              ? 'bg-primary-600/10'
                              : isSelectedForBulkDelete
                                ? 'bg-red-500/10 hover:bg-red-500/20'
                                : 'bg-slate-900/40 hover:bg-slate-800/50'
                          }`}
                        >
                          <td className="px-4 py-3 text-sm text-slate-400">{formatDate(item.createdAt)}</td>
                          <td className="px-4 py-3 text-sm text-slate-400">{formatDate(item.updatedAt)}</td>
                          <td className="px-4 py-3 text-sm text-slate-300">
                            <span className="px-2 py-0.5 bg-slate-700/70 text-slate-300 rounded text-xs">
                              {item.gearType}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-sm text-white font-medium">{item.brand}</td>
                          <td className="px-4 py-3 text-sm text-slate-300">{item.model}</td>
                          <td className="px-4 py-3 text-sm text-slate-400">{item.variant || '—'}</td>
                          <td className="px-4 py-3 text-sm">
                            <span className={`px-2 py-0.5 rounded text-xs ${getImageStatusClass(item.imageStatus)}`}>
                              {getImageStatusLabel(item.imageStatus)}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-sm">
                            <span className={`px-2 py-0.5 rounded text-xs ${getCatalogStatusClass(item.status)}`}>
                              {getCatalogStatusLabel(item.status)}
                            </span>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              )}
            </div>

            {/* Gear cards - mobile */}
            <div className="md:hidden space-y-3">
              {isLoading ? (
                <div className="p-8 text-center border border-slate-800 bg-slate-900/40 rounded-xl">
                  <div className="w-8 h-8 border-2 border-primary-500/30 border-t-primary-500 rounded-full animate-spin mx-auto" />
                  <p className="text-slate-400 mt-4">Loading...</p>
                </div>
              ) : items.length === 0 ? (
                <div className="p-8 text-center border border-slate-800 bg-slate-900/40 rounded-xl">
                  <p className="text-slate-400">No items found</p>
                </div>
              ) : (
                items.map((item) => (
                  <div
                    key={item.id}
                    role="button"
                    tabIndex={0}
                    onClick={() => {
                      if (isBulkEditMode) {
                        toggleSelection(item.id);
                      } else {
                        handleEditClick(item);
                      }
                    }}
                    onKeyDown={(event) => {
                      if (event.key === 'Enter' || event.key === ' ') {
                        event.preventDefault();
                        if (isBulkEditMode) {
                          toggleSelection(item.id);
                        } else {
                          handleEditClick(item);
                        }
                      }
                    }}
                    aria-pressed={isBulkEditMode ? selectedItemIds.has(item.id) : undefined}
                    className={`w-full text-left rounded-xl p-4 transition-colors border ${
                      editingItemId === item.id
                        ? 'border-primary-500/50 bg-primary-600/10'
                        : isBulkEditMode && selectedItemIds.has(item.id)
                          ? 'border-red-500/50 bg-red-500/10'
                          : 'border-slate-800 bg-slate-900/40 hover:bg-slate-800/50'
                    }`}
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <span className="px-2 py-0.5 bg-slate-700 rounded text-xs text-slate-300">
                            {item.gearType}
                          </span>
                          <span className={`px-2 py-0.5 rounded text-xs ${getCatalogStatusClass(item.status)}`}>
                            {getCatalogStatusLabel(item.status)}
                          </span>
                          <span className={`px-2 py-0.5 rounded text-xs ${getImageStatusClass(item.imageStatus)}`}>
                            {getImageStatusLabel(item.imageStatus)}
                          </span>
                        </div>
                        <h3 className="text-white font-medium truncate">
                          {item.brand} {item.model}
                        </h3>
                        {item.variant && (
                          <p className="text-sm text-slate-400 truncate">{item.variant}</p>
                        )}
                        <div className="grid grid-cols-2 gap-3 mt-3 text-xs">
                          <div>
                            <p className="text-slate-500 uppercase tracking-wide">Upload</p>
                            <p className="text-slate-300 mt-0.5">{formatDate(item.createdAt)}</p>
                          </div>
                          <div className="text-right">
                            <p className="text-slate-500 uppercase tracking-wide">Last Edit</p>
                            <p className="text-slate-300 mt-0.5">{formatDate(item.updatedAt)}</p>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                ))
              )}
            </div>

            {/* Infinite scroll loading indicator */}
            {hasMore && !isLoading && (
              <div ref={loadMoreRef} className="flex items-center justify-center py-6">
                {isLoadingMore ? (
                  <div className="flex items-center gap-3">
                    <div className="w-5 h-5 border-2 border-primary-500/30 border-t-primary-500 rounded-full animate-spin" />
                    <span className="text-slate-400">Loading more...</span>
                  </div>
                ) : (
                  <span className="text-slate-500 text-sm">Scroll for more</span>
                )}
              </div>
            )}

            {/* End of list indicator */}
            {!hasMore && items.length > 0 && (
              <div className="text-center py-4 text-slate-500 text-sm">
                Showing all {items.length} of {totalCount} items
              </div>
            )}
          </>
        ) : (
          <>
            {/* Build table - desktop */}
            <div className="hidden md:block border border-slate-800 rounded-xl overflow-hidden bg-slate-900/40">
              {isLoadingBuilds ? (
                <div className="p-8 text-center">
                  <div className="w-8 h-8 border-2 border-primary-500/30 border-t-primary-500 rounded-full animate-spin mx-auto" />
                  <p className="text-slate-400 mt-4">Loading builds...</p>
                </div>
              ) : builds.length === 0 ? (
                <div className="p-8 text-center">
                  <p className="text-slate-400">No builds found</p>
                </div>
              ) : (
                <table className="w-full text-sm">
                  <thead className="sticky top-0 z-10 bg-slate-900 text-slate-400">
                    <tr className="border-b border-slate-800">
                      <th className="px-4 py-3 text-left font-medium">Last Edit</th>
                      <th className="px-4 py-3 text-left font-medium">Status</th>
                      <th className="px-4 py-3 text-left font-medium">Title</th>
                      <th className="px-4 py-3 text-left font-medium">Pilot</th>
                      <th className="px-4 py-3 text-left font-medium">Description</th>
                    </tr>
                  </thead>
                  <tbody>
                    {builds.map((build) => {
                      const displayName = build.title || 'Untitled Build';
                      const isSelected = editingBuildId === build.id;
                      return (
                        <tr
                          key={build.id}
                          onClick={() => handleBuildEditClick(build)}
                          onKeyDown={(event) => {
                            if (event.key === 'Enter' || event.key === ' ') {
                              event.preventDefault();
                              handleBuildEditClick(build);
                            }
                          }}
                          role="button"
                          tabIndex={0}
                          aria-label={`Open editor for ${displayName}`}
                          className={`border-t border-slate-800 transition-colors cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-inset focus-visible:bg-primary-600/20 ${
                            isSelected ? 'bg-primary-600/10' : 'bg-slate-900/40 hover:bg-slate-800/50'
                          }`}
                        >
                          <td className="px-4 py-3 text-sm text-slate-400">{formatDateTime(build.updatedAt)}</td>
                          <td className="px-4 py-3 text-sm">
                            <span className={`px-2 py-0.5 rounded text-xs ${getBuildStatusClass(build.status)}`}>
                              {getBuildStatusLabel(build.status)}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-sm text-white font-medium">{displayName}</td>
                          <td className="px-4 py-3 text-sm text-slate-300">
                            {build.pilot?.callSign || build.pilot?.displayName || 'Pilot'}
                          </td>
                          <td className="px-4 py-3 text-sm text-slate-300 max-w-md truncate">
                            {build.description?.trim() || 'No description provided'}
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              )}
            </div>

            {/* Build cards - mobile */}
            <div className="md:hidden space-y-3">
              {isLoadingBuilds ? (
                <div className="p-8 text-center border border-slate-800 bg-slate-900/40 rounded-xl">
                  <div className="w-8 h-8 border-2 border-primary-500/30 border-t-primary-500 rounded-full animate-spin mx-auto" />
                  <p className="text-slate-400 mt-4">Loading builds...</p>
                </div>
              ) : builds.length === 0 ? (
                <div className="p-8 text-center border border-slate-800 bg-slate-900/40 rounded-xl">
                  <p className="text-slate-400">No builds found</p>
                </div>
              ) : (
                builds.map((build) => (
                  <button
                    key={build.id}
                    type="button"
                    onClick={() => handleBuildEditClick(build)}
                    className={`group w-full rounded-xl border p-4 text-left transition ${
                      editingBuildId === build.id
                        ? 'border-primary-500/50 bg-primary-600/10'
                        : 'border-slate-700 bg-slate-800/50 hover:border-primary-500/50 hover:bg-slate-800'
                    }`}
                  >
                    <div className="mb-2 flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <p className="truncate text-base font-semibold text-white">{build.title || 'Untitled Build'}</p>
                        <p className="text-sm text-slate-400">
                          by {build.pilot?.callSign || build.pilot?.displayName || 'Pilot'}
                        </p>
                      </div>
                      <span className={`rounded-full px-2 py-1 text-xs font-medium ${getBuildStatusClass(build.status)}`}>
                        {getBuildStatusLabel(build.status)}
                      </span>
                    </div>
                    <p className="line-clamp-2 text-sm text-slate-300">
                      {build.description?.trim() || 'No description provided'}
                    </p>
                    <div className="mt-3 text-xs text-slate-500">
                      Updated {formatDateTime(build.updatedAt)}
                    </div>
                  </button>
                ))
              )}
            </div>
          </>
        )}
      </div>

      <MobileFloatingControls
        label={activeTab === 'gear' ? 'Gear Filters' : 'Build Filters'}
        isOpen={isMobileControlsOpen}
        onToggle={() => setIsMobileControlsOpen((prev) => !prev)}
      >
        {controls}
      </MobileFloatingControls>
      </div>

      {/* Edit Modal */}
      {editingItemId && (
        <AdminGearEditModal
          key={modalKey}
          itemId={editingItemId}
          onClose={handleEditClose}
          onSave={handleEditSave}
          onDelete={handleEditDelete}
        />
      )}

      <CatalogSearchModal
        isOpen={showAddGearModal}
        onClose={handleAddGearClose}
        onSelectItem={handleAddGearSelect}
        startInCreateMode
        enableJsonImport
        onModerateCatalogImage={moderateGearCatalogImageUpload}
        onSaveCatalogImageUpload={adminSaveGearImageUpload}
      />

      {editingBuildId && (
        <AdminBuildEditModal
          key={buildModalKey}
          buildId={editingBuildId}
          onClose={handleBuildEditClose}
          onSave={handleBuildEditSaved}
          onPublished={handleBuildPublished}
        />
      )}

      {showBulkDeleteConfirm && (
        <div className="fixed inset-0 z-30 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/70" onClick={handleCancelBulkDelete} />
          <div
            role="dialog"
            aria-modal="true"
            aria-labelledby="gear-bulk-delete-title"
            aria-describedby="gear-bulk-delete-description"
            className="relative bg-slate-800 rounded-xl p-6 max-w-md w-full shadow-2xl border border-red-500/50"
          >
            <div className="flex items-start justify-between gap-3 mb-4">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 bg-red-500/20 rounded-full flex items-center justify-center">
                  <svg className="w-5 h-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </div>
                <h3 id="gear-bulk-delete-title" className="text-lg font-semibold text-white">
                  Delete Selected Gear Items?
                </h3>
              </div>
              <button
                type="button"
                onClick={handleCancelBulkDelete}
                disabled={isBulkDeleting}
                aria-label="Close bulk delete modal"
                className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors disabled:opacity-50"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <div id="gear-bulk-delete-description" className="mb-4 text-sm text-slate-300 space-y-2">
              <p>
                <strong className="text-red-400">This action cannot be undone.</strong> This will permanently delete{' '}
                <span className="text-white font-medium">{selectedCount}</span> gear catalog item{selectedCount === 1 ? '' : 's'}.
              </p>
              <p className="text-slate-400">
                Any linked inventory items and build parts will be kept, but their catalog link will be removed.
              </p>
              {hasSelectedInUse && (
                <p className="text-amber-300">
                  Warning: {selectedInUseCount} selected item{selectedInUseCount === 1 ? '' : 's'} are linked to inventory records.
                </p>
              )}
            </div>

            <label className="block text-sm font-medium text-slate-300 mb-2">
              Type <span className="text-white font-semibold">DELETE</span> to confirm
            </label>
            <input
              type="text"
              value={bulkDeleteConfirmText}
              onChange={(e) => setBulkDeleteConfirmText(e.target.value)}
              disabled={isBulkDeleting}
              className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-red-400 disabled:opacity-50"
              placeholder="DELETE"
              autoFocus
            />

            <div className="mt-5 flex flex-col sm:flex-row gap-2">
              <button
                type="button"
                onClick={handleCancelBulkDelete}
                disabled={isBulkDeleting}
                className="w-full px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-lg font-medium transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={() => void handleConfirmBulkDelete()}
                disabled={isBulkDeleting || bulkDeleteConfirmText.trim().toUpperCase() !== 'DELETE'}
                className="w-full px-4 py-2 bg-red-600 hover:bg-red-500 text-white rounded-lg font-medium transition-colors disabled:opacity-50"
              >
                {isBulkDeleting ? 'Deleting…' : `Delete ${selectedCount}`}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

interface AdminBuildEditModalProps {
  buildId: string;
  onClose: () => void;
  onSave: () => void;
  onPublished: () => void;
}

function AdminBuildEditModal({ buildId, onClose, onSave, onPublished }: AdminBuildEditModalProps) {
  const [build, setBuild] = useState<Build | null>(null);
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [imageFile, setImageFile] = useState<File | null>(null);
  const [imagePreview, setImagePreview] = useState<string | null>(null);
  const [showImageModal, setShowImageModal] = useState(false);
  const [modalImageFile, setModalImageFile] = useState<File | null>(null);
  const [modalImagePreview, setModalImagePreview] = useState<string | null>(null);
  const [imageModalError, setImageModalError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [isPublishing, setIsPublishing] = useState(false);
  const [isDeletingImage, setIsDeletingImage] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [validationErrors, setValidationErrors] = useState<BuildValidationError[]>([]);
  const [imageCacheBuster, setImageCacheBuster] = useState(() => Date.now());

  useEffect(() => {
    let cancelled = false;

    const loadBuild = async () => {
      setIsLoading(true);
      setError(null);
      try {
        const loaded = await adminGetBuild(buildId);
        if (cancelled) return;
        setBuild(loaded);
        setTitle(loaded.title || '');
        setDescription(loaded.description || '');
      } catch (err) {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : 'Failed to load build');
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    };

    void loadBuild();
    return () => {
      cancelled = true;
      if (imagePreview?.startsWith('blob:')) {
        URL.revokeObjectURL(imagePreview);
      }
      if (modalImagePreview?.startsWith('blob:')) {
        URL.revokeObjectURL(modalImagePreview);
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [buildId]);

  const hasExistingImage = Boolean(build?.mainImageUrl);
  const existingImageUrl = build ? getAdminBuildImageUrl(build.id, imageCacheBuster) : null;
  const currentPreview = imagePreview || (hasExistingImage ? existingImageUrl : null);

  const refreshBuild = useCallback(async () => {
    const refreshed = await adminGetBuild(buildId);
    setBuild(refreshed);
    setTitle(refreshed.title || '');
    setDescription(refreshed.description || '');
    setImageFile(null);
    if (imagePreview?.startsWith('blob:')) {
      URL.revokeObjectURL(imagePreview);
    }
    setImagePreview(null);
    setModalImageFile(null);
    if (modalImagePreview?.startsWith('blob:')) {
      URL.revokeObjectURL(modalImagePreview);
    }
    setModalImagePreview(null);
    setImageCacheBuster(Date.now());
  }, [buildId, imagePreview, modalImagePreview]);

  const handleOpenImageModal = () => {
    setImageModalError(null);
    setShowImageModal(true);
    setModalImageFile(imageFile);
    setModalImagePreview(imagePreview);
  };

  const handleCloseImageModal = () => {
    setShowImageModal(false);
    if (modalImagePreview?.startsWith('blob:') && modalImagePreview !== imagePreview) {
      URL.revokeObjectURL(modalImagePreview);
    }
    setModalImageFile(null);
    setModalImagePreview(null);
    setImageModalError(null);
  };

  const handleSelectImage = (file: File) => {
    if (file.size > 2 * 1024 * 1024) {
      setImageModalError('Image file is too large. Maximum size is 2MB.');
      return;
    }
    const validTypes = ['image/jpeg', 'image/jpg', 'image/png'];
    if (!validTypes.includes(file.type)) {
      setImageModalError('Invalid image type. Please use JPEG or PNG.');
      return;
    }

    const previewUrl = URL.createObjectURL(file);
    if (modalImagePreview?.startsWith('blob:')) {
      URL.revokeObjectURL(modalImagePreview);
    }
    setModalImageFile(file);
    setModalImagePreview(previewUrl);
    setImageModalError(null);
  };

  const handleSaveImageSelection = () => {
    if (!modalImageFile || !modalImagePreview) return;
    if (imagePreview?.startsWith('blob:')) {
      URL.revokeObjectURL(imagePreview);
    }
    setImageFile(modalImageFile);
    setImagePreview(modalImagePreview);
    setModalImageFile(null);
    setModalImagePreview(null);
    setShowImageModal(false);
  };

  const saveChanges = useCallback(async (publishAfterSave: boolean) => {
    if (!build) return;

    const updatePayload = {
      title: title.trim(),
      description: description.trim(),
    };

    if (publishAfterSave) {
      setIsPublishing(true);
    } else {
      setIsSaving(true);
    }
    setError(null);
    setValidationErrors([]);

    try {
      let updated = await adminUpdateBuild(build.id, updatePayload);

      if (imageFile) {
        await adminUploadBuildImage(build.id, imageFile);
        updated = await adminGetBuild(build.id);
      }

      if (publishAfterSave) {
        const publishResponse = await adminPublishBuild(build.id);
        if (!publishResponse.validation.valid) {
          setValidationErrors(publishResponse.validation.errors ?? []);
          if (publishResponse.build) {
            setBuild(publishResponse.build);
          } else {
            setBuild(updated);
          }
          return;
        }
        onPublished();
        return;
      }

      setBuild(updated);
      setImageFile(null);
      if (imagePreview?.startsWith('blob:')) {
        URL.revokeObjectURL(imagePreview);
      }
      setImagePreview(null);
      setImageCacheBuster(Date.now());
      onSave();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save build');
    } finally {
      setIsSaving(false);
      setIsPublishing(false);
    }
  }, [build, description, imageFile, imagePreview, onPublished, onSave, title]);

  const handleDeleteImage = async () => {
    if (!build || isDeletingImage) return;
    setIsDeletingImage(true);
    setError(null);
    try {
      await adminDeleteBuildImage(build.id);
      await refreshBuild();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete build image');
    } finally {
      setIsDeletingImage(false);
    }
  };

  if (isLoading) {
    return (
      <div className="fixed inset-0 z-[70] flex items-center justify-center bg-black/70 p-4">
        <div className="rounded-xl border border-slate-700 bg-slate-800 p-6 text-slate-300">
          Loading build...
        </div>
      </div>
    );
  }

  if (!build) {
    return (
      <div className="fixed inset-0 z-[70] flex items-center justify-center bg-black/70 p-4">
        <div className="w-full max-w-md rounded-xl border border-slate-700 bg-slate-800 p-6">
          <p className="text-slate-300">Build not found.</p>
          <button
            type="button"
            onClick={onClose}
            className="mt-4 rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white hover:bg-primary-500"
          >
            Close
          </button>
        </div>
      </div>
    );
  }

  return (
    <>
      <div className="fixed inset-0 z-[65] bg-black/70" onClick={onClose} />
      <div className="fixed inset-0 z-[70] flex items-center justify-center p-4">
        <div className="max-h-[92vh] w-full max-w-3xl overflow-y-auto rounded-xl border border-slate-700 bg-slate-800 p-5 shadow-2xl">
          <div className="mb-4 flex items-center justify-between gap-3">
            <h3 className="text-lg font-semibold text-white">Review Build</h3>
            <button
              type="button"
              onClick={onClose}
              className="rounded-lg p-2 text-slate-400 hover:bg-slate-700 hover:text-white"
              aria-label="Close build moderation modal"
            >
              <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          {error && (
            <div className="mb-4 rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-300">
              {error}
            </div>
          )}

          {validationErrors.length > 0 && (
            <div className="mb-4 rounded-lg border border-amber-500/30 bg-amber-500/10 p-3 text-sm text-amber-200">
              <p className="font-medium">Build cannot be published yet:</p>
              <ul className="mt-1 list-inside list-disc text-xs text-amber-100">
                {validationErrors.map((validation) => (
                  <li key={`${validation.category}-${validation.code}-${validation.message}`}>{validation.message}</li>
                ))}
              </ul>
            </div>
          )}

          <div className="grid gap-4 md:grid-cols-[minmax(0,1fr),260px]">
            <div className="space-y-3">
              <label className="block">
                <span className="mb-1 block text-xs font-medium uppercase tracking-wide text-slate-400">Title</span>
                <input
                  value={title}
                  onChange={(event) => setTitle(event.target.value)}
                  className="h-11 w-full rounded-lg border border-slate-600 bg-slate-900 px-3 text-white focus:border-primary-500 focus:outline-none"
                />
              </label>
              <label className="block">
                <span className="mb-1 block text-xs font-medium uppercase tracking-wide text-slate-400">Description</span>
                <textarea
                  value={description}
                  onChange={(event) => setDescription(event.target.value)}
                  rows={6}
                  className="w-full rounded-lg border border-slate-600 bg-slate-900 px-3 py-2 text-white focus:border-primary-500 focus:outline-none"
                />
              </label>
            </div>

            <div className="space-y-3 rounded-lg border border-slate-700 bg-slate-900/40 p-3">
              <p className="text-xs font-medium uppercase tracking-wide text-slate-400">Build Image</p>
              <div className="aspect-square overflow-hidden rounded-lg border border-slate-600 bg-slate-800">
                {currentPreview ? (
                  <img src={currentPreview} alt={title || 'Build'} className="h-full w-full object-cover" />
                ) : (
                  <div className="flex h-full items-center justify-center text-slate-500">No image</div>
                )}
              </div>
              <div className="grid gap-2">
                <button
                  type="button"
                  onClick={handleOpenImageModal}
                  className="rounded-lg bg-primary-600 px-3 py-2 text-sm font-medium text-white hover:bg-primary-500"
                >
                  {currentPreview ? 'Change Image' : 'Upload Image'}
                </button>
                {hasExistingImage && (
                  <a
                    href={existingImageUrl || undefined}
                    download={`${(title || 'build').replace(/\s+/g, '-').toLowerCase()}-image`}
                    className="rounded-lg border border-slate-600 px-3 py-2 text-center text-sm text-slate-200 hover:border-slate-500 hover:text-white"
                  >
                    Download Image
                  </a>
                )}
                {hasExistingImage && (
                  <button
                    type="button"
                    onClick={() => void handleDeleteImage()}
                    disabled={isDeletingImage}
                    className="rounded-lg border border-red-500/40 px-3 py-2 text-sm text-red-300 hover:bg-red-500/10 disabled:opacity-50"
                  >
                    {isDeletingImage ? 'Removing...' : 'Remove Image'}
                  </button>
                )}
              </div>
            </div>
          </div>

          <div className="mt-5 flex flex-wrap justify-end gap-2">
            <button
              type="button"
              onClick={onClose}
              className="rounded-lg border border-slate-600 px-4 py-2 text-sm font-medium text-slate-200 hover:border-slate-500 hover:text-white"
            >
              Cancel
            </button>
            <button
              type="button"
              disabled={isSaving || isPublishing}
              onClick={() => void saveChanges(false)}
              className="rounded-lg border border-slate-600 px-4 py-2 text-sm font-medium text-slate-200 hover:border-slate-500 hover:text-white disabled:opacity-60"
            >
              {isSaving ? 'Saving...' : 'Save Changes'}
            </button>
            <button
              type="button"
              disabled={isSaving || isPublishing}
              onClick={() => void saveChanges(true)}
              className="rounded-lg bg-emerald-600 px-4 py-2 text-sm font-medium text-white hover:bg-emerald-500 disabled:opacity-60"
            >
              {isPublishing ? 'Publishing...' : 'Publish Build'}
            </button>
          </div>
        </div>
      </div>

      <ImageUploadModal
        isOpen={showImageModal}
        title={currentPreview ? 'Update Build Image' : 'Upload Build Image'}
        previewUrl={modalImagePreview || currentPreview}
        previewAlt={title || 'Build image preview'}
        placeholder="🚁"
        accept="image/jpeg,image/jpg,image/png"
        helperText="JPEG or PNG. Max 2MB."
        selectButtonLabel={modalImagePreview ? 'Choose Different' : 'Select Image'}
        onSelectFile={(file) => handleSelectImage(file)}
        onClose={handleCloseImageModal}
        onSave={handleSaveImageSelection}
        disableSave={!modalImageFile || !modalImagePreview}
        saveLabel="Use Image"
        errorMessage={imageModalError}
        zIndexClassName="z-[75]"
      />
    </>
  );
}

// Edit Modal Component
interface AdminGearEditModalProps {
  itemId: string;
  onClose: () => void;
  onSave: () => void;
  onDelete: () => void;
}

function AdminGearEditModal({ itemId, onClose, onSave, onDelete }: AdminGearEditModalProps) {
  const [item, setItem] = useState<GearCatalogItem | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [brand, setBrand] = useState('');
  const [model, setModel] = useState('');
  const [variant, setVariant] = useState('');
  const [description, setDescription] = useState('');
  const [msrp, setMsrp] = useState('');
  const [bestFor, setBestFor] = useState<DroneType[]>([]);
  const [status, setStatus] = useState<CatalogItemStatus>('pending');
  const [selectedImageStatus, setSelectedImageStatus] = useState<GearCatalogItem['imageStatus']>('missing');
  const [imageFile, setImageFile] = useState<File | null>(null);
  const [imagePreview, setImagePreview] = useState<string | null>(null);
  const [imageUploadId, setImageUploadId] = useState<string | null>(null);
  const [showImageModal, setShowImageModal] = useState(false);
  const [modalImageFile, setModalImageFile] = useState<File | null>(null);
  const [modalImagePreview, setModalImagePreview] = useState<string | null>(null);
  const [modalImageUploadId, setModalImageUploadId] = useState<string | null>(null);
  const [imageModalStatusText, setImageModalStatusText] = useState<string | null>(null);
  const [imageModalStatusTone, setImageModalStatusTone] = useState<'neutral' | 'success' | 'error'>('neutral');
  const [imageModalStatusReason, setImageModalStatusReason] = useState<string | null>(null);
  const [isModeratingImage, setIsModeratingImage] = useState(false);
  const [imageModalError, setImageModalError] = useState<string | null>(null);
  const [deleteImage, setDeleteImage] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deleteConfirmText, setDeleteConfirmText] = useState('');
  const [error, setError] = useState<string | null>(null);
  const deleteDialogRef = useRef<HTMLDivElement>(null);
  const previouslyFocusedElementRef = useRef<HTMLElement | null>(null);
  const imagePreviewRef = useRef<string | null>(null);
  const modalImagePreviewRef = useRef<string | null>(null);

  useEffect(() => {
    imagePreviewRef.current = imagePreview;
  }, [imagePreview]);

  useEffect(() => {
    modalImagePreviewRef.current = modalImagePreview;
  }, [modalImagePreview]);

  useEffect(() => {
    return () => {
      const urls = new Set<string>();
      if (imagePreviewRef.current?.startsWith('blob:')) urls.add(imagePreviewRef.current);
      if (modalImagePreviewRef.current?.startsWith('blob:')) urls.add(modalImagePreviewRef.current);
      urls.forEach((url) => URL.revokeObjectURL(url));
    };
  }, []);
  
  // Fetch fresh item data when modal opens
  useEffect(() => {
    let cancelled = false;
    
    async function fetchItem() {
      try {
        const freshItem = await adminGetGear(itemId);
        if (cancelled) return;
        
        setItem(freshItem);
        setBrand(freshItem.brand);
        setModel(freshItem.model);
        setVariant(freshItem.variant || '');
        setDescription(freshItem.description || '');
        setMsrp(freshItem.msrp?.toString() || '');
        setBestFor((freshItem.bestFor || []) as DroneType[]);
        setStatus(freshItem.status);
        setSelectedImageStatus(freshItem.imageStatus);
      } catch (err) {
        if (cancelled) return;
        setError(err instanceof Error ? err.message : 'Failed to load item');
      } finally {
        if (!cancelled) setIsLoading(false);
      }
    }
    
    fetchItem();
    return () => { cancelled = true; };
  }, [itemId]);
  
  // Cache-buster timestamp to force browser to fetch fresh images
  const [imageCacheBuster] = useState(() => Date.now());
  
  // Determine if item has an existing image (either URL or stored image)
  const hasExistingImage = item ? (item.imageUrl || item.imageStatus === 'approved' || item.imageStatus === 'scanned') : false;
  const existingImageUrl = item ? (item.imageUrl || ((item.imageStatus === 'approved' || item.imageStatus === 'scanned') ? getAdminGearImageUrl(item.id, imageCacheBuster) : null)) : null;
  const willHaveImage = imageFile !== null || (!deleteImage && Boolean(hasExistingImage));

  useEffect(() => {
    setSelectedImageStatus((prevStatus) => {
      if (!willHaveImage && prevStatus !== 'missing') {
        return 'missing';
      }
      if (willHaveImage && prevStatus === 'missing') {
        return 'scanned';
      }
      return prevStatus;
    });
  }, [willHaveImage]);

  const moderationRequestRef = useRef(0);

  const handleFileChange = async (file: File) => {
    // Validate file size (2MB max)
    if (file.size > 2 * 1024 * 1024) {
      setImageModalError('Image file is too large. Maximum size is 2MB.');
      return;
    }
    
    // Validate file type
    const validTypes = ['image/jpeg', 'image/jpg', 'image/png'];
    if (!validTypes.includes(file.type)) {
      setImageModalError('Invalid image type. Please use JPEG or PNG.');
      return;
    }
    
    const requestId = ++moderationRequestRef.current;

    setError(null);
    setImageModalError(null);
    setImageModalStatusTone('neutral');
    setImageModalStatusReason(null);
    setImageModalStatusText('Checking image for safety…');
    setIsModeratingImage(true);
    setModalImageUploadId(null);

    const previewUrl = URL.createObjectURL(file);
    if (modalImagePreview?.startsWith('blob:') && modalImagePreview !== imagePreview) {
      URL.revokeObjectURL(modalImagePreview);
    }

    setModalImageFile(file);
    setModalImagePreview(previewUrl);

    try {
      const moderation = await moderateGearCatalogImageUpload(file);
      if (requestId !== moderationRequestRef.current) {
        return;
      }

      if (moderation.status === 'APPROVED' && moderation.uploadId) {
        setModalImageUploadId(moderation.uploadId);
        setImageModalStatusTone('success');
        setImageModalStatusText('Approved');
        setImageModalStatusReason(null);
        return;
      }

      if (moderation.status === 'REJECTED') {
        setImageModalStatusTone('error');
        setImageModalStatusText('Not allowed');
        setImageModalStatusReason(moderation.reason ?? 'Image failed safety checks');
        return;
      }

      setImageModalStatusTone('error');
      setImageModalStatusText('Unable to verify right now');
      setImageModalStatusReason(moderation.reason ?? 'Unable to verify image right now');
    } catch (err) {
      if (requestId !== moderationRequestRef.current) {
        return;
      }
      const message = err instanceof Error ? err.message : 'Unable to verify image right now';
      setImageModalStatusTone('error');
      setImageModalStatusText('Unable to verify right now');
      setImageModalStatusReason(message);
      setImageModalError(message);
      setError(message);
    } finally {
      if (requestId === moderationRequestRef.current) {
        setIsModeratingImage(false);
      }
    }
  };

  const handleOpenImageModal = () => {
    setShowImageModal(true);
    setImageModalError(null);
    setModalImageFile(imageFile);
    setModalImagePreview(imagePreview);
    setModalImageUploadId(imageUploadId);
    setImageModalStatusText(null);
    setImageModalStatusTone('neutral');
    setImageModalStatusReason(null);
    setIsModeratingImage(false);
  };

  const handleCloseImageModal = () => {
    setShowImageModal(false);
    moderationRequestRef.current += 1;
    if (modalImagePreview?.startsWith('blob:') && modalImagePreview !== imagePreview) {
      URL.revokeObjectURL(modalImagePreview);
    }
    setModalImageFile(null);
    setModalImagePreview(null);
    setModalImageUploadId(null);
    setImageModalStatusText(null);
    setImageModalStatusTone('neutral');
    setImageModalStatusReason(null);
    setIsModeratingImage(false);
    setImageModalError(null);
  };

  const handleSaveImageSelection = () => {
    if (!modalImageFile || !modalImagePreview || !modalImageUploadId) return;

    setDeleteImage(false);
    if (imagePreview?.startsWith('blob:')) {
      URL.revokeObjectURL(imagePreview);
    }
    setImageFile(modalImageFile);
    setImagePreview(modalImagePreview);
    setImageUploadId(modalImageUploadId);
    setSelectedImageStatus('scanned');
    setError(null);
    setImageModalError(null);
    setShowImageModal(false);
    setModalImageFile(null);
    setModalImagePreview(null);
    setModalImageUploadId(null);
    setImageModalStatusText(null);
    setImageModalStatusTone('neutral');
    setImageModalStatusReason(null);
    setIsModeratingImage(false);
  };

  const handleDeleteImage = () => {
    setDeleteImage(true);
    setImageFile(null);
    setImageUploadId(null);
    if (imagePreview?.startsWith('blob:')) {
      URL.revokeObjectURL(imagePreview);
    }
    setImagePreview(null);
    setModalImageFile(null);
    setModalImageUploadId(null);
    if (modalImagePreview?.startsWith('blob:')) {
      URL.revokeObjectURL(modalImagePreview);
    }
    setModalImagePreview(null);
    setImageModalError(null);
    setSelectedImageStatus('missing');
  };

  const closeDeleteConfirm = useCallback(() => {
    if (isDeleting) return;
    setShowDeleteConfirm(false);
    setDeleteConfirmText('');
  }, [isDeleting]);

  useEffect(() => {
    if (!showDeleteConfirm) {
      previouslyFocusedElementRef.current?.focus();
      previouslyFocusedElementRef.current = null;
      return;
    }

    previouslyFocusedElementRef.current =
      document.activeElement instanceof HTMLElement ? document.activeElement : null;

    const dialog = deleteDialogRef.current;
    if (!dialog) return;

    const getFocusableElements = () =>
      Array.from(
        dialog.querySelectorAll<HTMLElement>(
          'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
        )
      );

    const initialFocusTarget =
      dialog.querySelector<HTMLElement>('[data-delete-initial-focus="true"]') ?? getFocusableElements()[0];
    initialFocusTarget?.focus();

    const handleDialogKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        closeDeleteConfirm();
        return;
      }

      if (event.key !== 'Tab') return;

      const focusableElements = getFocusableElements();
      if (focusableElements.length === 0) {
        event.preventDefault();
        dialog.focus();
        return;
      }

      const firstElement = focusableElements[0];
      const lastElement = focusableElements[focusableElements.length - 1];
      const activeElement = document.activeElement;

      if (event.shiftKey) {
        if (activeElement === firstElement || activeElement === dialog) {
          event.preventDefault();
          lastElement.focus();
        }
        return;
      }

      if (activeElement === lastElement) {
        event.preventDefault();
        firstElement.focus();
      }
    };

    dialog.addEventListener('keydown', handleDialogKeyDown);
    return () => {
      dialog.removeEventListener('keydown', handleDialogKeyDown);
    };
  }, [closeDeleteConfirm, showDeleteConfirm]);

  const handleDeleteItem = async () => {
    if (!item) return;

    const requiresTypedDelete = item.usageCount > 0;
    if (requiresTypedDelete && deleteConfirmText.trim().toLowerCase() !== 'delete') {
      return;
    }

    setIsDeleting(true);
    setError(null);
    try {
      await adminDeleteGear(item.id);
      setShowDeleteConfirm(false);
      setDeleteConfirmText('');
      setIsDeleting(false);
      onDelete();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete gear item');
      setIsDeleting(false);
    }
  };

  const applyChanges = async (statusOverride?: CatalogItemStatus) => {
    if (!item) return;

    const params: AdminUpdateGearCatalogParams = {};

    // Only include changed fields
    if (brand !== item.brand) params.brand = brand;
    if (model !== item.model) params.model = model;
    if (variant !== (item.variant || '')) params.variant = variant;
    if (description !== (item.description || '')) params.description = description;

    // Check if bestFor has changed
    const itemBestFor = (item.bestFor || []) as DroneType[];
    const bestForChanged = bestFor.length !== itemBestFor.length ||
      bestFor.some(t => !itemBestFor.includes(t));
    if (bestForChanged) {
      params.bestFor = bestFor;
    }

    if (msrp !== (item.msrp?.toString() || '')) {
      if (msrp) {
        params.msrp = parseFloat(msrp);
      } else if (item.msrp != null) {
        // Explicitly clear MSRP if it was previously set
        params.clearMsrp = true;
      }
    }

    if (statusOverride) {
      if (item.status !== statusOverride) {
        params.status = statusOverride;
      }
    } else if (item.status === 'pending' && status === item.status) {
      // Default submit action from pending is to approve/publish.
      params.status = 'published';
    } else if (status !== item.status) {
      params.status = status;
    }

    // Allow admins to explicitly set/unset image curation status, including "unapprove" to scanned.
    if (selectedImageStatus !== item.imageStatus || imageFile !== null || deleteImage) {
      params.imageStatus = selectedImageStatus;
    }

    // Handle image: upload new, delete existing, or no change
    if (imageUploadId) {
      await adminSaveGearImageUpload(item.id, imageUploadId);
    } else if (deleteImage && hasExistingImage) {
      // Delete existing image
      await adminDeleteGearImage(item.id);
    }

    // Update other fields if changed
    if (Object.keys(params).length > 0) {
      await adminUpdateGear(item.id, params);
    }
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!item) return;

    setIsSaving(true);
    setError(null);

    try {
      await applyChanges();
      onSave(); // Signal that we're done, parent will refresh
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update gear item');
    } finally {
      setIsSaving(false);
    }
  };

  // Show loading state
  if (isLoading) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />
        <div className="relative bg-slate-800 rounded-xl shadow-2xl max-w-2xl w-full p-8">
          <div className="flex items-center justify-center">
            <div className="w-8 h-8 border-2 border-primary-500/30 border-t-primary-500 rounded-full animate-spin" />
            <span className="ml-3 text-slate-400">Loading...</span>
          </div>
        </div>
      </div>
    );
  }

  if (!item) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />
        <div className="relative bg-slate-800 rounded-xl shadow-2xl max-w-2xl w-full p-8">
          <button
            onClick={onClose}
            aria-label="Close edit gear modal"
            className="absolute top-4 right-4 p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
          <div className="text-center text-red-400">{error || 'Item not found'}</div>
        </div>
      </div>
    );
  }

  const saveButtonLabel = (() => {
    if (item.status === 'pending' && status === item.status) {
      return 'Approve';
    }
    if (status !== item.status) {
      if (status === 'published') return 'Publish';
      if (status === 'pending') return 'Move to Pending';
      return 'Remove';
    }
    return 'Save Changes';
  })();

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={onClose}
      />

      {/* Modal */}
      <div
        className="relative bg-slate-800 rounded-xl shadow-2xl max-w-2xl w-full max-h-[90vh] overflow-hidden"
        aria-hidden={showDeleteConfirm}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-700">
          <h2 className="text-lg font-semibold text-white">
            Edit Gear Item
          </h2>
          <button
            onClick={onClose}
            className="p-1 text-slate-400 hover:text-white transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Form */}
        <form id="gear-edit-form" onSubmit={handleSubmit} className="p-6 space-y-4 overflow-y-auto max-h-[calc(90vh-140px)]">
          {error && (
            <div className="p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">
              {error}
            </div>
          )}

          {/* Read-only info */}
          <div className="p-3 bg-slate-700/50 rounded-lg">
            <p className="text-sm text-slate-400">
              <strong>Gear Type:</strong> {item.gearType}
            </p>
            <p className="text-sm text-slate-400 mt-1">
              <strong>Upload Date:</strong> {formatDateTime(item.createdAt)}
            </p>
            <p className="text-sm text-slate-400 mt-1">
              <strong>Last Edit:</strong> {formatDateTime(item.updatedAt)}
            </p>
            <p className="text-sm text-slate-400 mt-1">
              <strong>Status:</strong>{' '}
              <span className={getCatalogStatusTextClass(item.status)}>
                {getCatalogStatusLabel(item.status)}
              </span>
            </p>
            <p className="text-sm text-slate-400 mt-1">
              <strong>Image Status:</strong>{' '}
              <span className={getImageStatusTextClass(item.imageStatus)}>
                {item.imageStatus}
              </span>
            </p>
          </div>

          {/* Catalog status */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Catalog Status
            </label>
            <select
              value={status}
              onChange={(e) => setStatus(e.target.value as CatalogItemStatus)}
              className="w-full h-11 px-3 bg-slate-700 border border-slate-600 rounded-lg text-white focus:outline-none focus:border-primary-500"
            >
              <option value="pending">Pending</option>
              <option value="published">Published</option>
              <option value="removed">Removed</option>
            </select>
            <p className="text-xs text-slate-500 mt-1">
              Only <span className="text-green-400">Published</span> items appear in the public catalog.
            </p>
          </div>

          {/* Image moderation status */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Image Status
            </label>
            <select
              value={selectedImageStatus}
              onChange={(e) => setSelectedImageStatus(e.target.value as GearCatalogItem['imageStatus'])}
              className="w-full h-11 px-3 bg-slate-700 border border-slate-600 rounded-lg text-white focus:outline-none focus:border-primary-500"
            >
              {willHaveImage ? (
                <>
                  <option value="scanned">Scanned (Needs Review)</option>
                  <option value="approved">Approved</option>
                </>
              ) : (
                <option value="missing">Missing</option>
              )}
            </select>
            <p className="text-xs text-slate-500 mt-1">
              Set to <span className="text-blue-400">Scanned</span> to unapprove while keeping the image.
            </p>
          </div>

          {/* Brand & Model */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-1">
                Brand
              </label>
              <input
                type="text"
                value={brand}
                onChange={(e) => setBrand(e.target.value)}
                className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white focus:outline-none focus:border-primary-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-1">
                Model
              </label>
              <input
                type="text"
                value={model}
                onChange={(e) => setModel(e.target.value)}
                className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white focus:outline-none focus:border-primary-500"
              />
            </div>
          </div>

          {/* Variant */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Variant
            </label>
            <input
              type="text"
              value={variant}
              onChange={(e) => setVariant(e.target.value)}
              placeholder="e.g., 1950KV, V2"
              className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500"
            />
          </div>

          {/* Description */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Description
            </label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              rows={3}
              placeholder="Brief description of the gear..."
              className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500 resize-none"
            />
          </div>

          {/* Best For - Drone Types */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Best For (optional)
            </label>
            <div className="flex flex-wrap gap-2">
              {DRONE_TYPES.map(type => (
                <button
                  key={type.value}
                  type="button"
                  onClick={() => {
                    setBestFor(prev => 
                      prev.includes(type.value)
                        ? prev.filter(t => t !== type.value)
                        : [...prev, type.value]
                    );
                  }}
                  className={`px-3 py-1.5 text-sm rounded-lg border transition-colors ${
                    bestFor.includes(type.value)
                      ? 'bg-primary-600 border-primary-500 text-white'
                      : 'bg-slate-700 border-slate-600 text-slate-300 hover:border-slate-500'
                  }`}
                >
                  {type.label}
                </button>
              ))}
            </div>
            <p className="text-xs text-slate-500 mt-1">
              Select what drone types this gear is best suited for
            </p>
          </div>

          {/* MSRP */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              MSRP
            </label>
            <div className="relative">
              <span className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400">$</span>
              <input
                type="number"
                step="0.01"
                min="0"
                value={msrp}
                onChange={(e) => setMsrp(e.target.value)}
                placeholder="0.00"
                className="w-full pl-7 pr-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500"
              />
            </div>
          </div>

          {/* Image Upload (Admin only) */}
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Product Image
              <span className="ml-2 text-xs text-primary-400">(Max 2MB, JPEG/PNG)</span>
            </label>
            
            {/* Show existing image or new preview */}
            {!deleteImage && (imagePreview || existingImageUrl) && (
              <div className="mb-3">
                <div className="relative inline-block">
                <img
                  src={imagePreview || existingImageUrl || ''}
                  alt="Preview"
                  className="w-32 h-32 object-cover rounded-lg bg-slate-700"
                  onError={(e) => {
                    (e.target as HTMLImageElement).style.display = 'none';
                  }}
                />
                {(existingImageUrl || imagePreview) && (
                  <button
                    type="button"
                    onClick={handleDeleteImage}
                    className="absolute -top-2 -right-2 p-1 bg-red-600 hover:bg-red-700 text-white rounded-full transition-colors"
                    title="Remove image"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                )}
                </div>
                {existingImageUrl && !imagePreview && (
                  <a
                    href={existingImageUrl}
                    download={`${(item.brand || 'gear').replace(/\s+/g, '-').toLowerCase()}-${(item.model || 'image').replace(/\s+/g, '-').toLowerCase()}`}
                    className="mt-2 inline-flex items-center gap-2 text-sm text-primary-300 hover:text-primary-200 transition-colors"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 3v12m0 0l-4-4m4 4l4-4m-9 8h10" />
                    </svg>
                    Download current image
                  </a>
                )}
              </div>
            )}
            
            <button
              type="button"
              onClick={handleOpenImageModal}
              className="px-3 py-2 bg-slate-700 hover:bg-slate-600 border border-slate-600 rounded-lg text-sm text-white transition-colors"
            >
              {(imagePreview || (!deleteImage && existingImageUrl)) ? 'Change Image' : 'Add Image'}
            </button>
            
            {deleteImage && hasExistingImage && (
              <p className="mt-2 text-sm text-amber-400">
                Image will be removed when you save.
              </p>
            )}

            <p className="mt-2 text-xs text-slate-500">
              Use the image modal to choose a file. JPEG or PNG. Max 2MB.
            </p>

          </div>

        </form>

        {/* Footer */}
        <div className="flex items-center justify-between gap-3 px-6 py-4 border-t border-slate-700 bg-slate-800/50">
          <button
            type="button"
            onClick={() => setShowDeleteConfirm(true)}
            disabled={isSaving || isDeleting || showDeleteConfirm}
            className="px-4 py-2 bg-red-600 hover:bg-red-500 disabled:opacity-50 disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors"
          >
            Delete Item
          </button>
          <button
            type="submit"
            form="gear-edit-form"
            disabled={isSaving || isDeleting || showDeleteConfirm}
            className="px-4 py-2 bg-primary-600 hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors flex items-center gap-2"
          >
            {isSaving ? (
              <>
                <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                Saving...
              </>
            ) : (
              <>
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                </svg>
                {saveButtonLabel}
              </>
            )}
          </button>
        </div>
      </div>

      <ImageUploadModal
        isOpen={showImageModal}
        title="Edit Gear Image"
        previewUrl={modalImagePreview || (!deleteImage ? (imagePreview || existingImageUrl || null) : null)}
        previewAlt={modalImagePreview ? 'Gear preview' : 'Current gear image'}
        placeholder="📦"
        accept="image/jpeg,image/jpg,image/png"
        helperText="JPEG or PNG. Max 2MB."
        selectButtonLabel={modalImagePreview ? 'Choose Different' : 'Select Image'}
        onSelectFile={handleFileChange}
        onClose={handleCloseImageModal}
        onSave={handleSaveImageSelection}
        disableSelect={isModeratingImage}
        disableSave={!modalImageFile || !modalImagePreview || !modalImageUploadId || isModeratingImage}
        statusText={imageModalStatusText}
        statusTone={imageModalStatusTone}
        statusReason={imageModalStatusReason ?? undefined}
        errorMessage={imageModalError}
      />

      {showDeleteConfirm && (
        <div className="absolute inset-0 z-10 flex items-center justify-center p-4">
          <div className="absolute inset-0 bg-black/70" onClick={closeDeleteConfirm} />
          <div
            ref={deleteDialogRef}
            role="dialog"
            aria-modal="true"
            aria-labelledby="delete-gear-dialog-title"
            aria-describedby="delete-gear-dialog-description"
            tabIndex={-1}
            className="relative bg-slate-800 rounded-xl p-6 max-w-md w-full shadow-2xl border border-red-500/50"
          >
            <div className="flex items-start justify-between gap-3 mb-4">
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 bg-red-500/20 rounded-full flex items-center justify-center">
                  <svg className="w-5 h-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </div>
                <h3 id="delete-gear-dialog-title" className="text-lg font-semibold text-white">Delete Gear Item?</h3>
              </div>
              <button
                onClick={closeDeleteConfirm}
                disabled={isDeleting}
                aria-label="Close delete gear modal"
                className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors disabled:opacity-50"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <div id="delete-gear-dialog-description" className="mb-4">
              <p className="text-slate-300 mb-3">
                <strong className="text-red-400">This action cannot be undone.</strong> Deleting this catalog item will permanently remove:
              </p>
              <ul className="text-sm text-slate-400 space-y-2 mb-4">
                <li className="flex items-start gap-2">
                  <span className="text-red-400 mt-0.5">•</span>
                  <span>
                    Catalog entry for <span className="font-medium text-slate-200">{item.brand} {item.model}{item.variant ? ` ${item.variant}` : ''}</span>
                  </span>
                </li>
                <li className="flex items-start gap-2">
                  <span className="text-red-400 mt-0.5">•</span>
                  <span>Any curated catalog image associated with this item</span>
                </li>
                <li className="flex items-start gap-2">
                  <span className="text-red-400 mt-0.5">•</span>
                  <span>Catalog links from related inventory records (inventory items themselves are kept)</span>
                </li>
              </ul>

              {item.usageCount > 0 && (
                <>
                  <p className="text-sm text-amber-300 mb-2">
                    This item is currently linked to {item.usageCount} inventory record{item.usageCount !== 1 ? 's' : ''}.
                  </p>
                  <p className="text-sm text-slate-400">
                    Type <span className="font-mono text-red-400 bg-red-400/10 px-1.5 py-0.5 rounded">delete</span> to confirm:
                  </p>
                </>
              )}
            </div>

            {item.usageCount > 0 && (
              <input
                type="text"
                value={deleteConfirmText}
                onChange={(e) => setDeleteConfirmText(e.target.value)}
                placeholder="Type 'delete' to confirm"
                className="w-full px-4 py-2 bg-slate-700 border border-red-500/50 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-red-500 focus:border-transparent mb-4"
                data-delete-initial-focus="true"
                disabled={isDeleting}
              />
            )}

            <div className="flex items-center justify-end gap-2">
              <button
                type="button"
                onClick={closeDeleteConfirm}
                disabled={isDeleting}
                data-delete-initial-focus={item.usageCount === 0 ? 'true' : undefined}
                className="px-3 py-2 text-sm text-slate-300 hover:text-white hover:bg-slate-700 rounded-lg transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={() => void handleDeleteItem()}
                disabled={
                  isDeleting ||
                  (item.usageCount > 0 && deleteConfirmText.trim().toLowerCase() !== 'delete')
                }
                className="px-4 py-2 bg-red-600 hover:bg-red-500 text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isDeleting ? 'Deleting...' : 'Delete Item'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default AdminGearModeration;

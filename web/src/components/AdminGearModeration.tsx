import { useState, useEffect, useCallback, useRef, type FormEvent, type ChangeEvent } from 'react';
import type { GearCatalogItem, GearType, ImageStatusFilter, AdminUpdateGearCatalogParams, DroneType } from '../gearCatalogTypes';
import { GEAR_TYPES, DRONE_TYPES } from '../gearCatalogTypes';
import { adminSearchGear, adminUpdateGear, adminUploadGearImage, adminDeleteGearImage, adminDeleteGear, adminGetGear, getAdminGearImageUrl } from '../adminApi';
import { CatalogSearchModal } from './CatalogSearchModal';

interface AdminGearModerationProps {
  isAdmin: boolean;
  authLoading?: boolean;
}

export function AdminGearModeration({ isAdmin, authLoading }: AdminGearModerationProps) {
  const [items, setItems] = useState<GearCatalogItem[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Filters
  const [query, setQuery] = useState('');
  const [appliedQuery, setAppliedQuery] = useState('');
  const [gearType, setGearType] = useState<GearType | ''>('');
  const [imageStatus, setImageStatus] = useState<ImageStatusFilter | ''>(''); // Default to "Needs Work"
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
  const [deleteTargetItem, setDeleteTargetItem] = useState<GearCatalogItem | null>(null);
  const [deleteConfirmText, setDeleteConfirmText] = useState('');
  const [isDeletingItem, setIsDeletingItem] = useState(false);

  const loadItems = useCallback(async (reset = false, forceRefresh = false) => {
    if (!isAdmin) return;
    
    // Prevent concurrent loads by default; allow forced resets to supersede in-flight loads.
    if (isLoadingRef.current && !(reset && forceRefresh)) return;
    isLoadingRef.current = true;
    const requestId = ++latestLoadRequestRef.current;

    if (reset) {
      setIsLoading(true);
      currentOffsetRef.current = 0;
    } else {
      setIsLoadingMore(true);
    }
    setError(null);

    const offset = currentOffsetRef.current;

    try {
      const response = await adminSearchGear({
        query: appliedQuery || undefined,
        gearType: gearType || undefined,
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
  }, [isAdmin, appliedQuery, gearType, imageStatus]);

  // Initial load and auto-search when filters change
  useEffect(() => {
    if (isAdmin) {
      loadItems(true);
    }
  }, [isAdmin, loadItems]);

  // Handle search button click
  const handleSearch = useCallback(() => {
    setAppliedQuery(query);
  }, [query]);

  // Handle enter key in search input
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSearch();
    }
  };

  // Handle clearing search
  const handleClearSearch = () => {
    setQuery('');
    setAppliedQuery('');
  };

  // Infinite scroll observer
  // Note: loadItems prevents concurrent calls by default (except forced reset refreshes),
  // so we don't need to check loading state here - just trigger on intersection.
  useEffect(() => {
    const element = loadMoreRef.current;
    if (!element || !hasMore) return;

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
  }, [hasMore, loadItems]);

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

  const handleDeleteClick = useCallback((item: GearCatalogItem) => {
    setDeleteTargetItem(item);
    setDeleteConfirmText('');
    setError(null);
  }, []);

  const handleCancelDelete = useCallback(() => {
    if (isDeletingItem) return;
    setDeleteTargetItem(null);
    setDeleteConfirmText('');
  }, [isDeletingItem]);

  const handleConfirmDelete = useCallback(async () => {
    if (!deleteTargetItem) return;

    const requiresTypedDelete = deleteTargetItem.usageCount > 0;
    if (requiresTypedDelete && deleteConfirmText.trim().toLowerCase() !== 'delete') {
      return;
    }

    setIsDeletingItem(true);
    setError(null);
    try {
      await adminDeleteGear(deleteTargetItem.id);
      if (editingItemId === deleteTargetItem.id) {
        setEditingItemId(null);
      }
      setDeleteTargetItem(null);
      setDeleteConfirmText('');
      await loadItems(true, true);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete gear item');
    } finally {
      setIsDeletingItem(false);
    }
  }, [deleteConfirmText, deleteTargetItem, editingItemId, loadItems]);

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

  // Show loading while auth state is being determined
  if (authLoading) {
    return (
      <div className="p-8 text-center">
        <div className="w-8 h-8 border-2 border-primary-500/30 border-t-primary-500 rounded-full animate-spin mx-auto" />
        <p className="text-slate-400 mt-4">Loading...</p>
      </div>
    );
  }

  if (!isAdmin) {
    return (
      <div className="p-8 text-center">
        <h1 className="text-2xl font-bold text-red-400 mb-4">Access Denied</h1>
        <p className="text-slate-400">You must be an admin to access this page.</p>
      </div>
    );
  }

  return (
    <>
      {/* Main flex container - matches news section pattern */}
      <div className="flex-1 flex flex-col min-h-0 relative">
        {/* Header - flex-shrink-0 keeps it fixed in place */}
        <div className="flex-shrink-0 z-10 bg-slate-900">
          <div className="bg-slate-800 border-b border-slate-700 px-4 md:px-6 py-3 md:py-4">
            <h1 className="text-lg md:text-2xl font-bold text-white mb-3">Gear Moderation</h1>

          {/* Filters */}
          <div className="flex flex-col gap-3">
            {/* Search query - full width with button */}
            <div className="flex gap-2">
              <div className="relative flex-1">
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
                  onKeyDown={handleKeyDown}
                  placeholder="Search brand or model..."
                  className="w-full pl-10 pr-4 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent"
                />
              </div>
              <button
                onClick={handleSearch}
                className="px-3 py-2 bg-primary-600 hover:bg-primary-700 text-white text-sm font-medium rounded-lg transition-colors"
              >
                Search
              </button>
              {appliedQuery && (
                <button
                  onClick={handleClearSearch}
                  className="px-3 py-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
                >
                  Clear
                </button>
              )}
            </div>

            {/* Filter row */}
            <div className="flex items-center gap-2">
              {/* Gear type filter */}
              <select
                value={gearType}
                onChange={(e) => setGearType(e.target.value as GearType | '')}
                className="flex-1 px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
              >
                <option value="">All Types</option>
                {GEAR_TYPES.map((type) => (
                  <option key={type.value} value={type.value}>
                    {type.label}
                  </option>
                ))}
              </select>

              {/* Image status filter */}
              <select
                value={imageStatus}
                onChange={(e) => setImageStatus(e.target.value as ImageStatusFilter | '')}
                className="flex-1 px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:ring-2 focus:ring-primary-500"
              >
                <option value="">Needs Work</option>
                <option value="all">All Records</option>
                <option value="missing">Needs Image</option>
                <option value="approved">Has Image</option>
                <option value="recently-curated">Recently Updated (24h)</option>
              </select>
            </div>

            {/* Results count and actions */}
            <div className="flex items-center justify-between gap-3">
              <p className="text-slate-400 text-sm">
                {totalCount} item{totalCount !== 1 ? 's' : ''} found
              </p>
              <button
                onClick={handleAddGearClick}
                className="px-3 py-2 bg-primary-600 hover:bg-primary-700 text-white text-sm font-medium rounded-lg transition-colors flex items-center gap-2"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                Add Gear
              </button>
            </div>
          </div>

          {/* Error message */}
          {error && (
            <div className="mt-3 p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">
              {error}
            </div>
          )}
        </div>
      </div>

      {/* Scrollable list */}
      <div 
        className="flex-1 overflow-y-auto min-h-0 px-4 md:px-6 pt-4 md:pt-6 pb-20"
        onScroll={() => {
          // Dismiss keyboard on scroll for mobile
          if (document.activeElement instanceof HTMLElement) {
            document.activeElement.blur();
          }
        }}
      >
        {/* Items table - desktop */}
        <div className="hidden md:block bg-slate-800 rounded-lg overflow-hidden">
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
            <table className="w-full">
            <thead className="bg-slate-700/50">
              <tr>
                <th className="px-4 py-3 text-left text-sm font-medium text-slate-300">
                  Created
                </th>
                <th className="px-4 py-3 text-left text-sm font-medium text-slate-300">
                  Type
                </th>
                <th className="px-4 py-3 text-left text-sm font-medium text-slate-300">
                  Brand
                </th>
                <th className="px-4 py-3 text-left text-sm font-medium text-slate-300">
                  Model
                </th>
                <th className="px-4 py-3 text-left text-sm font-medium text-slate-300">
                  Variant
                </th>
                <th className="px-4 py-3 text-left text-sm font-medium text-slate-300">
                  Image
                </th>
                <th className="px-4 py-3 text-left text-sm font-medium text-slate-300">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-700">
              {items.map((item) => (
                <tr key={item.id} className="hover:bg-slate-700/30">
                  <td className="px-4 py-3 text-sm text-slate-400">
                    {new Date(item.createdAt).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-3 text-sm text-slate-300">
                    <span className="px-2 py-0.5 bg-slate-700 rounded text-xs">
                      {item.gearType}
                    </span>
                  </td>
                  <td className="px-4 py-3 text-sm text-white font-medium">
                    {item.brand}
                  </td>
                  <td className="px-4 py-3 text-sm text-slate-300">
                    {item.model}
                  </td>
                  <td className="px-4 py-3 text-sm text-slate-400">
                    {item.variant || '—'}
                  </td>
                  <td className="px-4 py-3 text-sm">
                    {item.imageStatus === 'approved' ? (
                      <span className="px-2 py-0.5 bg-green-500/20 text-green-400 rounded text-xs">
                        Approved
                      </span>
                    ) : (
                      <span className="px-2 py-0.5 bg-yellow-500/20 text-yellow-400 rounded text-xs">
                        Missing
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-sm">
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => handleEditClick(item)}
                        className="px-3 py-1 bg-primary-600 hover:bg-primary-700 text-white text-xs font-medium rounded transition-colors"
                      >
                        Edit
                      </button>
                      <button
                        onClick={() => handleDeleteClick(item)}
                        className="px-3 py-1 bg-red-600 hover:bg-red-700 text-white text-xs font-medium rounded transition-colors"
                      >
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Items cards - mobile */}
      <div className="md:hidden space-y-3">
        {isLoading ? (
          <div className="p-8 text-center bg-slate-800 rounded-lg">
            <div className="w-8 h-8 border-2 border-primary-500/30 border-t-primary-500 rounded-full animate-spin mx-auto" />
            <p className="text-slate-400 mt-4">Loading...</p>
          </div>
        ) : items.length === 0 ? (
          <div className="p-8 text-center bg-slate-800 rounded-lg">
            <p className="text-slate-400">No items found</p>
          </div>
        ) : (
          items.map((item) => (
            <div
              key={item.id}
              className="bg-slate-800 rounded-lg p-4"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="px-2 py-0.5 bg-slate-700 rounded text-xs text-slate-300">
                      {item.gearType}
                    </span>
                    {item.imageStatus === 'approved' ? (
                      <span className="px-2 py-0.5 bg-green-500/20 text-green-400 rounded text-xs">
                        Has Image
                      </span>
                    ) : (
                      <span className="px-2 py-0.5 bg-yellow-500/20 text-yellow-400 rounded text-xs">
                        No Image
                      </span>
                    )}
                  </div>
                  <h3 className="text-white font-medium truncate">
                    {item.brand} {item.model}
                  </h3>
                  {item.variant && (
                    <p className="text-sm text-slate-400 truncate">{item.variant}</p>
                  )}
                  <p className="text-xs text-slate-500 mt-1">
                    Added {new Date(item.createdAt).toLocaleDateString()}
                  </p>
                </div>
                <div className="flex flex-col gap-2 flex-shrink-0">
                  <button
                    onClick={() => handleEditClick(item)}
                    className="px-3 py-1.5 bg-primary-600 hover:bg-primary-700 text-white text-sm font-medium rounded transition-colors"
                  >
                    Edit
                  </button>
                  <button
                    onClick={() => handleDeleteClick(item)}
                    className="px-3 py-1.5 bg-red-600 hover:bg-red-700 text-white text-sm font-medium rounded transition-colors"
                  >
                    Delete
                  </button>
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
      </div>
      </div>

      {/* Edit Modal */}
      {editingItemId && (
        <AdminGearEditModal
          key={modalKey}
          itemId={editingItemId}
          onClose={handleEditClose}
          onSave={handleEditSave}
        />
      )}

      <CatalogSearchModal
        isOpen={showAddGearModal}
        onClose={handleAddGearClose}
        onSelectItem={handleAddGearSelect}
        startInCreateMode
        onUploadCatalogImage={adminUploadGearImage}
      />

      {/* Delete Gear Confirmation Modal */}
      {deleteTargetItem && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
          <div className="bg-slate-800 rounded-xl p-6 max-w-md w-full shadow-2xl border border-red-500/50">
            <div className="flex items-center gap-3 mb-4">
              <div className="w-10 h-10 bg-red-500/20 rounded-full flex items-center justify-center">
                <svg className="w-5 h-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-white">Delete Gear Item?</h3>
            </div>

            <div className="mb-4">
              <p className="text-slate-300 mb-3">
                <strong className="text-red-400">This action cannot be undone.</strong> Deleting this catalog item will permanently remove:
              </p>
              <ul className="text-sm text-slate-400 space-y-2 mb-4">
                <li className="flex items-start gap-2">
                  <span className="text-red-400 mt-0.5">•</span>
                  <span>
                    Catalog entry for <span className="font-medium text-slate-200">{deleteTargetItem.brand} {deleteTargetItem.model}{deleteTargetItem.variant ? ` ${deleteTargetItem.variant}` : ''}</span>
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

              {deleteTargetItem.usageCount > 0 && (
                <>
                  <p className="text-sm text-amber-300 mb-2">
                    This item is currently linked to {deleteTargetItem.usageCount} inventory record{deleteTargetItem.usageCount !== 1 ? 's' : ''}.
                  </p>
                  <p className="text-sm text-slate-400">
                    Type <span className="font-mono text-red-400 bg-red-400/10 px-1.5 py-0.5 rounded">delete</span> to confirm:
                  </p>
                </>
              )}
            </div>

            {deleteTargetItem.usageCount > 0 && (
              <input
                type="text"
                value={deleteConfirmText}
                onChange={(e) => setDeleteConfirmText(e.target.value)}
                placeholder="Type 'delete' to confirm"
                className="w-full px-4 py-2 bg-slate-700 border border-red-500/50 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-red-500 focus:border-transparent mb-4"
                autoFocus
                disabled={isDeletingItem}
              />
            )}

            <div className="flex gap-3">
              <button
                onClick={handleCancelDelete}
                disabled={isDeletingItem}
                className="flex-1 px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-lg font-medium transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                onClick={() => void handleConfirmDelete()}
                disabled={
                  isDeletingItem ||
                  (deleteTargetItem.usageCount > 0 && deleteConfirmText.trim().toLowerCase() !== 'delete')
                }
                className="flex-1 px-4 py-2 bg-red-600 hover:bg-red-500 text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isDeletingItem ? 'Deleting...' : 'Delete Item'}
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

// Edit Modal Component
interface AdminGearEditModalProps {
  itemId: string;
  onClose: () => void;
  onSave: () => void;
}

function AdminGearEditModal({ itemId, onClose, onSave }: AdminGearEditModalProps) {
  const [item, setItem] = useState<GearCatalogItem | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [brand, setBrand] = useState('');
  const [model, setModel] = useState('');
  const [variant, setVariant] = useState('');
  const [description, setDescription] = useState('');
  const [msrp, setMsrp] = useState('');
  const [bestFor, setBestFor] = useState<DroneType[]>([]);
  const [imageFile, setImageFile] = useState<File | null>(null);
  const [imagePreview, setImagePreview] = useState<string | null>(null);
  const [deleteImage, setDeleteImage] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
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
  const hasExistingImage = item ? (item.imageUrl || item.imageStatus === 'approved') : false;
  const existingImageUrl = item ? (item.imageUrl || (item.imageStatus === 'approved' ? getAdminGearImageUrl(item.id, imageCacheBuster) : null)) : null;

  const handleFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) {
      setImageFile(null);
      setImagePreview(null);
      return;
    }
    
    // Validate file size (1MB max)
    if (file.size > 1024 * 1024) {
      setError('Image file is too large. Maximum size is 1MB.');
      e.target.value = '';
      return;
    }
    
    // Validate file type
    const validTypes = ['image/jpeg', 'image/png', 'image/webp'];
    if (!validTypes.includes(file.type)) {
      setError('Invalid image type. Please use JPEG, PNG, or WebP.');
      e.target.value = '';
      return;
    }
    
    setError(null);
    setDeleteImage(false);
    setImageFile(file);
    
    // Create preview
    const reader = new FileReader();
    reader.onload = (event) => {
      setImagePreview(event.target?.result as string);
    };
    reader.readAsDataURL(file);
  };

  const handleDeleteImage = () => {
    setDeleteImage(true);
    setImageFile(null);
    setImagePreview(null);
  };

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setIsSaving(true);
    setError(null);

    if (!item) return;

    try {
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
      
      // Handle image: upload new, delete existing, or no change
      if (imageFile) {
        // Upload new image
        await adminUploadGearImage(item.id, imageFile);
      } else if (deleteImage && hasExistingImage) {
        // Delete existing image
        await adminDeleteGearImage(item.id);
      }
      
      // Update other fields if changed
      if (Object.keys(params).length > 0) {
        await adminUpdateGear(item.id, params);
      }

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
          <div className="text-center text-red-400">{error || 'Item not found'}</div>
          <button onClick={onClose} className="mt-4 px-4 py-2 bg-slate-700 rounded text-white">Close</button>
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="relative bg-slate-800 rounded-xl shadow-2xl max-w-2xl w-full max-h-[90vh] overflow-hidden">
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
              <strong>Created:</strong> {new Date(item.createdAt).toLocaleString()}
            </p>
            <p className="text-sm text-slate-400 mt-1">
              <strong>Image Status:</strong>{' '}
              <span className={item.imageStatus === 'approved' ? 'text-green-400' : 'text-yellow-400'}>
                {item.imageStatus}
              </span>
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
              <span className="ml-2 text-xs text-primary-400">(Max 1MB, JPEG/PNG/WebP)</span>
            </label>
            
            {/* Show existing image or new preview */}
            {!deleteImage && (imagePreview || existingImageUrl) && (
              <div className="mb-3 relative inline-block">
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
            )}
            
            {/* File input */}
            <input
              type="file"
              accept="image/jpeg,image/png,image/webp"
              onChange={handleFileChange}
              className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white file:mr-4 file:py-1 file:px-3 file:rounded file:border-0 file:text-sm file:font-medium file:bg-primary-600 file:text-white hover:file:bg-primary-700 file:cursor-pointer cursor-pointer"
            />
            
            {deleteImage && hasExistingImage && (
              <p className="mt-2 text-sm text-amber-400">
                Image will be removed when you save.
              </p>
            )}
          </div>
        </form>

        {/* Footer */}
        <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-slate-700 bg-slate-800/50">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 text-slate-300 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
          >
            Cancel
          </button>
          <button
            type="submit"
            form="gear-edit-form"
            disabled={isSaving}
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
                Save Changes
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}

export default AdminGearModeration;

import { useState, useEffect, useCallback, useRef } from 'react';
import type { GearCatalogItem, GearType, ImageStatus, AdminUpdateGearCatalogParams } from '../gearCatalogTypes';
import { GEAR_TYPES } from '../gearCatalogTypes';
import { adminSearchGear, adminUpdateGear, adminUploadGearImage, adminDeleteGearImage, getGearImageUrl } from '../adminApi';

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
  const [gearType, setGearType] = useState<GearType | ''>('');
  const [imageStatus, setImageStatus] = useState<ImageStatus | ''>(''); // Default to all items
  const pageSize = 30;
  const [hasMore, setHasMore] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const loadMoreRef = useRef<HTMLDivElement>(null);
  
  // Use refs to track current offset and prevent race conditions
  const currentOffsetRef = useRef(0);
  const isLoadingRef = useRef(false);

  // Edit modal state
  const [editingItem, setEditingItem] = useState<GearCatalogItem | null>(null);

  const loadItems = useCallback(async (reset = false) => {
    if (!isAdmin) return;
    
    // Prevent concurrent loads
    if (isLoadingRef.current) return;
    isLoadingRef.current = true;

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
        query: query || undefined,
        gearType: gearType || undefined,
        imageStatus: imageStatus || undefined,
        limit: pageSize,
        offset: offset,
      });
      
      if (reset) {
        setItems(response.items);
      } else {
        setItems(prev => [...prev, ...response.items]);
      }
      currentOffsetRef.current = offset + response.items.length;
      setTotalCount(response.totalCount);
      setHasMore(response.items.length === pageSize && currentOffsetRef.current < response.totalCount);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load gear items');
    } finally {
      setIsLoading(false);
      setIsLoadingMore(false);
      isLoadingRef.current = false;
    }
  }, [isAdmin, query, gearType, imageStatus]);

  // Initial load
  useEffect(() => {
    loadItems(true);
  }, [loadItems]);

  // Infinite scroll observer
  useEffect(() => {
    if (!loadMoreRef.current || !hasMore) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !isLoadingRef.current) {
          loadItems(false);
        }
      },
      { threshold: 0.1 }
    );

    observer.observe(loadMoreRef.current);
    return () => observer.disconnect();
  }, [hasMore, loadItems]);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    loadItems(true);
  };

  const handleEditClick = (item: GearCatalogItem) => {
    setEditingItem(item);
  };

  const handleEditClose = () => {
    setEditingItem(null);
  };

  const handleEditSave = async () => {
    // Refresh the list after saving
    setEditingItem(null);
    loadItems(true);
  };

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
      {/* Fixed header on mobile - title, filters, count */}
      <div className="fixed md:relative top-14 md:top-0 left-0 right-0 md:left-auto md:right-auto z-20 md:z-10 bg-slate-900 p-4 md:p-6 pb-2 md:pb-4">
        <h1 className="text-xl md:text-2xl font-bold text-white mb-3 md:mb-4">Gear Moderation</h1>

        {/* Filters */}
        <form onSubmit={handleSearch} className="bg-slate-800 rounded-lg p-3 md:p-4 mb-3">
          <div className="grid grid-cols-2 md:grid-cols-4 gap-2 md:gap-4">
            {/* Search query */}
            <div className="col-span-2 md:col-span-1">
              <label className="block text-xs md:text-sm font-medium text-slate-300 mb-1">
                Search
              </label>
              <input
                type="text"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Brand or model..."
                className="w-full px-3 py-1.5 md:py-2 bg-slate-700 border border-slate-600 rounded-lg text-white text-sm placeholder-slate-400 focus:outline-none focus:border-primary-500"
              />
            </div>

            {/* Gear type filter */}
            <div>
              <label className="block text-xs md:text-sm font-medium text-slate-300 mb-1">
                Gear Type
              </label>
              <select
                value={gearType}
                onChange={(e) => setGearType(e.target.value as GearType | '')}
                className="w-full px-2 md:px-3 py-1.5 md:py-2 bg-slate-700 border border-slate-600 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500"
              >
                <option value="">All Types</option>
                {GEAR_TYPES.map((type) => (
                  <option key={type.value} value={type.value}>
                    {type.label}
                  </option>
                ))}
              </select>
            </div>

            {/* Image status filter */}
            <div>
              <label className="block text-xs md:text-sm font-medium text-slate-300 mb-1">
                Image Status
              </label>
              <select
                value={imageStatus}
                onChange={(e) => setImageStatus(e.target.value as ImageStatus | '')}
                className="w-full px-2 md:px-3 py-1.5 md:py-2 bg-slate-700 border border-slate-600 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500"
              >
                <option value="">All</option>
                <option value="missing">Needs Image</option>
                <option value="approved">Has Image</option>
              </select>
            </div>

            {/* Search button */}
            <div className="col-span-2 md:col-span-1 flex items-end">
              <button
                type="submit"
                className="w-full px-4 py-1.5 md:py-2 bg-primary-600 hover:bg-primary-700 text-white text-sm font-medium rounded-lg transition-colors"
              >
                Search
              </button>
            </div>
          </div>
        </form>

        {/* Error message */}
        {error && (
          <div className="p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm mb-2">
            {error}
          </div>
        )}

        {/* Results count */}
        <p className="text-slate-400 text-sm">
          {totalCount} item{totalCount !== 1 ? 's' : ''} found
        </p>
      </div>

      {/* Scrollable list - with padding for fixed header on mobile */}
      <div className="flex-1 overflow-y-auto pt-[280px] md:pt-0 px-4 md:px-6 pb-20">
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
                    <button
                      onClick={() => handleEditClick(item)}
                      className="px-3 py-1 bg-primary-600 hover:bg-primary-700 text-white text-xs font-medium rounded transition-colors"
                    >
                      Edit
                    </button>
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
                <button
                  onClick={() => handleEditClick(item)}
                  className="px-3 py-1.5 bg-primary-600 hover:bg-primary-700 text-white text-sm font-medium rounded transition-colors flex-shrink-0"
                >
                  Edit
                </button>
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

      {/* Edit Modal */}
      {editingItem && (
        <AdminGearEditModal
          item={editingItem}
          onClose={handleEditClose}
          onSave={handleEditSave}
        />
      )}
    </>
  );
}

// Edit Modal Component
interface AdminGearEditModalProps {
  item: GearCatalogItem;
  onClose: () => void;
  onSave: (item: GearCatalogItem) => void;
}

function AdminGearEditModal({ item, onClose, onSave }: AdminGearEditModalProps) {
  const [brand, setBrand] = useState(item.brand);
  const [model, setModel] = useState(item.model);
  const [variant, setVariant] = useState(item.variant || '');
  const [description, setDescription] = useState(item.description || '');
  const [msrp, setMsrp] = useState(item.msrp?.toString() || '');
  const [imageFile, setImageFile] = useState<File | null>(null);
  const [imagePreview, setImagePreview] = useState<string | null>(null);
  const [deleteImage, setDeleteImage] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  // Determine if item has an existing image (either URL or stored image)
  const hasExistingImage = item.imageUrl || item.imageStatus === 'approved';
  const existingImageUrl = item.imageUrl || (item.imageStatus === 'approved' ? getGearImageUrl(item.id) : null);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
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

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSaving(true);
    setError(null);

    try {
      const params: AdminUpdateGearCatalogParams = {};

      // Only include changed fields
      if (brand !== item.brand) params.brand = brand;
      if (model !== item.model) params.model = model;
      if (variant !== (item.variant || '')) params.variant = variant;
      if (description !== (item.description || '')) params.description = description;
      if (msrp !== (item.msrp?.toString() || '')) {
        params.msrp = msrp ? parseFloat(msrp) : undefined;
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

      onSave(item); // Signal that we're done, parent will refresh
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update gear item');
    } finally {
      setIsSaving(false);
    }
  };

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
        <form onSubmit={handleSubmit} className="p-6 space-y-4 overflow-y-auto max-h-[calc(90vh-140px)]">
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
            onClick={handleSubmit}
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

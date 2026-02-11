import { useState, useEffect, useCallback, useRef } from 'react';
import type { GearCatalogItem, GearType, CreateGearCatalogParams, DroneType } from '../gearCatalogTypes';
import { GEAR_TYPES, DRONE_TYPES, getCatalogItemDisplayName } from '../gearCatalogTypes';
import { searchGearCatalog, createGearCatalogItem, findNearMatches, getPopularGear } from '../gearCatalogApi';

type ModerationStatus = 'APPROVED' | 'REJECTED' | 'PENDING_REVIEW';

interface ModerationResult {
  status: ModerationStatus;
  reason?: string;
  uploadId?: string;
}

interface CatalogSearchModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSelectItem: (item: GearCatalogItem) => void;
  initialGearType?: GearType;
  startInCreateMode?: boolean;
  onUploadCatalogImage?: (itemId: string, imageFile: File) => Promise<void>;
  onModerateCatalogImage?: (imageFile: File) => Promise<ModerationResult>;
  onSaveCatalogImageUpload?: (itemId: string, uploadId: string) => Promise<void>;
}

export function CatalogSearchModal({
  isOpen,
  onClose,
  onSelectItem,
  initialGearType,
  startInCreateMode = false,
  onUploadCatalogImage,
  onModerateCatalogImage,
  onSaveCatalogImageUpload,
}: CatalogSearchModalProps) {
  const [query, setQuery] = useState('');
  const [gearType, setGearType] = useState<GearType | ''>(initialGearType || '');
  const [results, setResults] = useState<GearCatalogItem[]>([]);
  const [popularItems, setPopularItems] = useState<GearCatalogItem[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [showCreateForm, setShowCreateForm] = useState(startInCreateMode);
  const [error, setError] = useState<string | null>(null);
  
  const searchInputRef = useRef<HTMLInputElement>(null);
  const searchTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Focus search input when modal opens
  useEffect(() => {
    if (isOpen && searchInputRef.current) {
      setTimeout(() => searchInputRef.current?.focus(), 100);
    }
  }, [isOpen]);

  // Reset the starting mode each time the modal opens.
  useEffect(() => {
    if (isOpen) {
      setShowCreateForm(startInCreateMode);
    }
  }, [isOpen, startInCreateMode]);

  // Load popular items
  const loadPopularItems = useCallback(async () => {
    try {
      const { items } = await getPopularGear(gearType || undefined, 10);
      setPopularItems(items);
    } catch (err) {
      console.error('Failed to load popular items:', err);
    }
  }, [gearType]);

  // Load popular items on mount
  useEffect(() => {
    if (isOpen && !query) {
      loadPopularItems();
    }
  }, [isOpen, query, loadPopularItems]);

  // Debounced search
  const handleSearch = useCallback((searchQuery: string) => {
    if (searchTimeoutRef.current) {
      clearTimeout(searchTimeoutRef.current);
    }

    if (!searchQuery || searchQuery.length < 2) {
      setResults([]);
      return;
    }

    setIsLoading(true);
    searchTimeoutRef.current = setTimeout(async () => {
      try {
        const response = await searchGearCatalog({
          query: searchQuery,
          gearType: gearType || undefined,
          limit: 20,
        });
        setResults(response.items);
        setError(null);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Search failed');
        setResults([]);
      } finally {
        setIsLoading(false);
      }
    }, 300);
  }, [gearType]);

  useEffect(() => {
    handleSearch(query);
    return () => {
      if (searchTimeoutRef.current) {
        clearTimeout(searchTimeoutRef.current);
      }
    };
  }, [query, handleSearch]);

  const handleSelectItem = (item: GearCatalogItem) => {
    // Only call onSelectItem - let the parent component decide whether to close
    // When used from AddGearModal, this transitions to the details form step
    onSelectItem(item);
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="relative bg-slate-800 border border-slate-700 rounded-2xl w-full max-w-2xl max-h-[calc(100vh-2rem)] sm:max-h-[85vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-700">
          <div>
            <h2 className="text-lg font-semibold text-white">
              {showCreateForm ? 'Add New Gear to Catalog' : 'Search Gear Catalog'}
            </h2>
            <p className="text-sm text-slate-400 mt-0.5">
              {showCreateForm 
                ? 'Create a new entry in the shared gear catalog'
                : 'Search our community gear database'}
            </p>
          </div>
          <button
            onClick={onClose}
            className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {showCreateForm ? (
          <CreateCatalogItemForm
            initialGearType={gearType || undefined}
            initialQuery={query}
            onSuccess={handleSelectItem}
            onCancel={() => setShowCreateForm(false)}
            onUploadCatalogImage={onUploadCatalogImage}
            onModerateCatalogImage={onModerateCatalogImage}
            onSaveCatalogImageUpload={onSaveCatalogImageUpload}
          />
        ) : (
          <>
            {/* Search Controls */}
            <div className="px-6 py-4 border-b border-slate-700 space-y-3">
              {/* Search input */}
              <div className="relative">
                <svg 
                  className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-400" 
                  fill="none" 
                  stroke="currentColor" 
                  viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
                <input
                  ref={searchInputRef}
                  type="text"
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                  placeholder="Search by brand, model, or part name..."
                  className="w-full pl-10 pr-4 py-2.5 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500"
                />
                {isLoading && (
                  <div className="absolute right-3 top-1/2 -translate-y-1/2">
                    <div className="w-5 h-5 border-2 border-slate-600 border-t-primary-500 rounded-full animate-spin" />
                  </div>
                )}
              </div>

              {/* Gear type filter */}
              <div className="flex items-center gap-2">
                <span className="text-sm text-slate-400">Filter by type:</span>
                <select
                  value={gearType}
                  onChange={(e) => setGearType(e.target.value as GearType | '')}
                  className="px-3 py-1.5 bg-slate-700 border border-slate-600 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500"
                >
                  <option value="">All Types</option>
                  {GEAR_TYPES.map(type => (
                    <option key={type.value} value={type.value}>{type.label}</option>
                  ))}
                </select>
              </div>
            </div>

            {/* Results */}
            <div className="flex-1 overflow-y-auto">
              {error && (
                <div className="m-4 p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">
                  {error}
                </div>
              )}

              {query.length >= 2 && results.length > 0 && (
                <div className="p-4">
                  <h3 className="text-sm font-medium text-slate-400 mb-3">
                    Search Results ({results.length})
                  </h3>
                  <div className="space-y-2">
                    {results.map(item => (
                      <CatalogItemRow 
                        key={item.id} 
                        item={item} 
                        onSelect={handleSelectItem} 
                      />
                    ))}
                  </div>
                </div>
              )}

              {query.length >= 2 && results.length === 0 && !isLoading && (
                <div className="p-6 text-center">
                  <div className="w-12 h-12 mx-auto mb-3 rounded-full bg-slate-700 flex items-center justify-center">
                    <svg className="w-6 h-6 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                  </div>
                  <p className="text-slate-400 mb-4">No gear found matching "{query}"</p>
                  <button
                    onClick={() => setShowCreateForm(true)}
                    className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white font-medium rounded-lg transition-colors"
                  >
                    Add "{query}" to Catalog
                  </button>
                </div>
              )}

              {query.length < 2 && popularItems.length > 0 && (
                <div className="p-4">
                  <h3 className="text-sm font-medium text-slate-400 mb-3">
                    Popular Gear
                  </h3>
                  <div className="space-y-2">
                    {popularItems.map(item => (
                      <CatalogItemRow 
                        key={item.id} 
                        item={item} 
                        onSelect={handleSelectItem} 
                      />
                    ))}
                  </div>
                </div>
              )}

              {query.length < 2 && popularItems.length === 0 && !isLoading && (
                <div className="p-6 text-center text-slate-400">
                  <p>Start typing to search the gear catalog...</p>
                </div>
              )}
            </div>

            {/* Footer */}
            <div className="flex items-center justify-start px-6 py-4 border-t border-slate-700 bg-slate-800/50">
              <button
                onClick={() => setShowCreateForm(true)}
                className="text-primary-400 hover:text-primary-300 text-sm font-medium transition-colors flex items-center gap-1"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                Can't find it? Add new gear
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

// Individual catalog item row
interface CatalogItemRowProps {
  item: GearCatalogItem;
  onSelect: (item: GearCatalogItem) => void;
}

function CatalogItemRow({ item, onSelect }: CatalogItemRowProps) {
  const gearTypeLabel = GEAR_TYPES.find(t => t.value === item.gearType)?.label || item.gearType;

  return (
    <button
      onClick={() => onSelect(item)}
      className="w-full text-left p-3 bg-slate-700/50 hover:bg-slate-700 border border-slate-600 hover:border-primary-500/50 rounded-lg transition-all group"
    >
      <div className="flex items-start gap-3">
        {/* Image or placeholder */}
        <div className="w-12 h-12 rounded-lg bg-slate-600 flex-shrink-0 flex items-center justify-center overflow-hidden">
          {item.imageUrl ? (
            <img src={item.imageUrl} alt="" className="w-full h-full object-cover" />
          ) : (
            <svg className="w-6 h-6 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
            </svg>
          )}
        </div>

        {/* Details */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="font-medium text-white truncate">
              {getCatalogItemDisplayName(item)}
            </span>
            <span className="px-1.5 py-0.5 bg-slate-600 text-slate-300 text-xs rounded">
              {gearTypeLabel}
            </span>
          </div>
          <div className="text-sm text-slate-400 mt-0.5">
            {item.brand}
            {item.usageCount > 0 && (
              <span className="ml-2 text-slate-500">
                • {item.usageCount} {item.usageCount === 1 ? 'user' : 'users'}
              </span>
            )}
          </div>
        </div>

        {/* Arrow */}
        <svg 
          className="w-5 h-5 text-slate-500 group-hover:text-primary-400 transition-colors flex-shrink-0" 
          fill="none" 
          stroke="currentColor" 
          viewBox="0 0 24 24"
        >
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
        </svg>
      </div>
    </button>
  );
}

// Create catalog item form
interface CreateCatalogItemFormProps {
  initialGearType?: GearType;
  initialQuery?: string;
  onSuccess: (item: GearCatalogItem) => void;
  onCancel: () => void;
  onUploadCatalogImage?: (itemId: string, imageFile: File) => Promise<void>;
  onModerateCatalogImage?: (imageFile: File) => Promise<ModerationResult>;
  onSaveCatalogImageUpload?: (itemId: string, uploadId: string) => Promise<void>;
}

function CreateCatalogItemForm({
  initialGearType,
  initialQuery,
  onSuccess,
  onCancel,
  onUploadCatalogImage,
  onModerateCatalogImage,
  onSaveCatalogImageUpload,
}: CreateCatalogItemFormProps) {
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [nearMatches, setNearMatches] = useState<GearCatalogItem[]>([]);
  const [checkingDuplicates, setCheckingDuplicates] = useState(false);
  const [showImageModal, setShowImageModal] = useState(false);
  const [isImageUploading, setIsImageUploading] = useState(false);
  const [imageStatusText, setImageStatusText] = useState<string | null>(null);
  const [imageStatusTone, setImageStatusTone] = useState<'neutral' | 'success' | 'error'>('neutral');
  const imageInputRef = useRef<HTMLInputElement>(null);

  type SelectedCatalogImage = {
    file?: File;
    previewUrl: string;
    uploadId?: string;
    moderationStatus?: ModerationStatus;
    moderationReason?: string;
  };

  const [selectedImage, setSelectedImage] = useState<SelectedCatalogImage | null>(null);
  const [modalImage, setModalImage] = useState<SelectedCatalogImage | null>(null);
  const usesModerationFlow = !!onModerateCatalogImage && !!onSaveCatalogImageUpload;

  const revokePreviewUrl = (url?: string) => {
    if (typeof url === 'string' && url.startsWith('blob:')) {
      URL.revokeObjectURL(url);
    }
  };

  // Parse initial query into brand/model if possible
  const parseInitialQuery = () => {
    if (!initialQuery) return { brand: '', model: '' };
    const parts = initialQuery.trim().split(/\s+/);
    if (parts.length >= 2) {
      return { brand: parts[0], model: parts.slice(1).join(' ') };
    }
    return { brand: '', model: initialQuery };
  };

  const { brand: initialBrand, model: initialModel } = parseInitialQuery();

  const [gearType, setGearType] = useState<GearType>(initialGearType || 'other');
  const [brand, setBrand] = useState(initialBrand);
  const [model, setModel] = useState(initialModel);
  const [variant, setVariant] = useState('');
  const [bestFor, setBestFor] = useState<DroneType[]>([]);
  const [msrp, setMsrp] = useState('');
  const [description, setDescription] = useState('');

  const selectedImageRef = useRef<SelectedCatalogImage | null>(null);
  const modalImageRef = useRef<SelectedCatalogImage | null>(null);

  useEffect(() => {
    selectedImageRef.current = selectedImage;
  }, [selectedImage]);

  useEffect(() => {
    modalImageRef.current = modalImage;
  }, [modalImage]);

  useEffect(() => {
    return () => {
      const previewUrls = new Set<string>();
      if (selectedImageRef.current?.previewUrl) previewUrls.add(selectedImageRef.current.previewUrl);
      if (modalImageRef.current?.previewUrl) previewUrls.add(modalImageRef.current.previewUrl);
      previewUrls.forEach((url) => revokePreviewUrl(url));
    };
  }, []);

  const handleImageFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) {
      return;
    }
    if (imageInputRef.current) {
      imageInputRef.current.value = '';
    }

    // Keep validation aligned with gear catalog image endpoint.
    if (file.size > 1024 * 1024) {
      setError('Image file is too large. Maximum size is 1MB.');
      return;
    }

    const validTypes = ['image/jpeg', 'image/png', 'image/webp'];
    if (!validTypes.includes(file.type)) {
      setError('Invalid image type. Please use JPEG, PNG, or WebP.');
      return;
    }

    setError(null);
    const previewUrl = URL.createObjectURL(file);
    if (modalImage?.previewUrl && modalImage.previewUrl !== selectedImage?.previewUrl) {
      revokePreviewUrl(modalImage.previewUrl);
    }

    if (!usesModerationFlow || !onModerateCatalogImage) {
      setModalImage({ file, previewUrl });
      setImageStatusText(null);
      setImageStatusTone('neutral');
      return;
    }

    setModalImage({ previewUrl, moderationStatus: 'PENDING_REVIEW' });
    setIsImageUploading(true);
    setImageStatusTone('neutral');
    setImageStatusText('Uploading image…');

    try {
      await new Promise((resolve) => setTimeout(resolve, 120));
      setImageStatusText('Checking image for safety…');

      const moderation = await onModerateCatalogImage(file);
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
      setError(err instanceof Error ? err.message : 'Unable to verify image right now');
    } finally {
      setIsImageUploading(false);
    }
  };

  const handleOpenImageModal = () => {
    setShowImageModal(true);
    setImageStatusText(null);
    setImageStatusTone('neutral');
    if (selectedImage) {
      setModalImage({ ...selectedImage });
    } else {
      setModalImage(null);
    }
  };

  const handleCloseImageModal = () => {
    setShowImageModal(false);
    if (modalImage?.previewUrl && modalImage.previewUrl !== selectedImage?.previewUrl) {
      revokePreviewUrl(modalImage.previewUrl);
    }
    setModalImage(null);
    setImageStatusText(null);
    setImageStatusTone('neutral');
    setIsImageUploading(false);
    if (imageInputRef.current) {
      imageInputRef.current.value = '';
    }
  };

  const handleSaveImageSelection = () => {
    if (!modalImage) return;
    if (usesModerationFlow && (!modalImage.uploadId || modalImage.moderationStatus !== 'APPROVED')) {
      return;
    }
    if (selectedImage?.previewUrl && selectedImage.previewUrl !== modalImage.previewUrl) {
      revokePreviewUrl(selectedImage.previewUrl);
    }
    setSelectedImage(modalImage);
    setShowImageModal(false);
    setModalImage(null);
    setImageStatusText(null);
    setImageStatusTone('neutral');
  };

  const handleRemoveImage = () => {
    if (selectedImage?.previewUrl) {
      revokePreviewUrl(selectedImage.previewUrl);
    }
    if (modalImage?.previewUrl && modalImage.previewUrl !== selectedImage?.previewUrl) {
      revokePreviewUrl(modalImage.previewUrl);
    }
    setSelectedImage(null);
    setModalImage(null);
    setImageStatusText(null);
    setImageStatusTone('neutral');
    setError(null);
    if (imageInputRef.current) {
      imageInputRef.current.value = '';
    }
  };

  // Check for duplicates
  const checkForDuplicates = useCallback(async () => {
    if (!brand || !model) return;
    
    setCheckingDuplicates(true);
    try {
      const response = await findNearMatches({
        gearType,
        brand,
        model,
        threshold: 0.3,
      });
      setNearMatches(response.matches.map(m => m.item));
    } catch (err) {
      console.error('Failed to check duplicates:', err);
    } finally {
      setCheckingDuplicates(false);
    }
  }, [brand, model, gearType]);

  // Check for duplicates when brand/model change
  useEffect(() => {
    if (brand && model && gearType) {
      checkForDuplicates();
    } else {
      setNearMatches([]);
    }
  }, [brand, model, gearType, checkForDuplicates]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setError(null);

    try {
      const params: CreateGearCatalogParams = {
        gearType,
        brand: brand.trim(),
        model: model.trim(),
        variant: variant.trim() || undefined,
        bestFor: bestFor.length > 0 ? bestFor : undefined,
        msrp: msrp ? parseFloat(msrp) : undefined,
        description: description.trim() || undefined,
      };

      const response = await createGearCatalogItem(params);

      if (selectedImage) {
        if (usesModerationFlow) {
          if (selectedImage.uploadId && onSaveCatalogImageUpload) {
            await onSaveCatalogImageUpload(response.item.id, selectedImage.uploadId);
          }
        } else if (onUploadCatalogImage && selectedImage.file) {
          await onUploadCatalogImage(response.item.id, selectedImage.file);
        }
      }
      
      if (response.existing) {
        // Found existing item
        setError(`Found existing entry: ${getCatalogItemDisplayName(response.item)}. Using that instead.`);
      }
      
      onSuccess(response.item);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create catalog item');
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="flex flex-col min-h-0 flex-1">
      <div className="flex-1 overflow-y-auto p-6 space-y-4 min-h-0">
        {error && (
          <div className="p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">
            {error}
          </div>
        )}

        {/* Near matches warning */}
        {nearMatches.length > 0 && (
          <div className="p-3 bg-yellow-500/10 border border-yellow-500/20 rounded-lg">
            <p className="text-yellow-400 text-sm font-medium mb-2">
              Similar items found - did you mean one of these?
            </p>
            <div className="space-y-1">
              {nearMatches.slice(0, 3).map(item => (
                <button
                  key={item.id}
                  type="button"
                  onClick={() => onSuccess(item)}
                  className="w-full text-left px-2 py-1.5 bg-slate-700/50 hover:bg-slate-700 rounded text-sm text-slate-300 hover:text-white transition-colors"
                >
                  {getCatalogItemDisplayName(item)}
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Gear Type */}
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1">
            Gear Type <span className="text-red-400">*</span>
          </label>
          <select
            value={gearType}
            onChange={(e) => setGearType(e.target.value as GearType)}
            required
            className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white focus:outline-none focus:border-primary-500"
          >
            {GEAR_TYPES.map(type => (
              <option key={type.value} value={type.value}>{type.label}</option>
            ))}
          </select>
        </div>

        {/* Brand & Model */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Brand <span className="text-red-400">*</span>
            </label>
            <input
              type="text"
              value={brand}
              onChange={(e) => setBrand(e.target.value)}
              required
              placeholder="e.g., T-Motor"
              className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Model <span className="text-red-400">*</span>
            </label>
            <input
              type="text"
              value={model}
              onChange={(e) => setModel(e.target.value)}
              required
              placeholder="e.g., F60 Pro IV"
              className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500"
            />
          </div>
        </div>

        {/* Variant */}
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1">
            Variant (optional)
          </label>
          <input
            type="text"
            value={variant}
            onChange={(e) => setVariant(e.target.value)}
            placeholder="e.g., 1950KV, V2, Pro, LR"
            className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500"
          />
          <p className="text-xs text-slate-500 mt-1">
            Specify KV rating, version, or other distinguishing features
          </p>
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
            MSRP (optional)
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
          <p className="text-xs text-slate-500 mt-1">
            Manufacturer's suggested retail price
          </p>
        </div>

        {/* Description */}
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1">
            Description (optional)
          </label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={2}
            placeholder="Brief description of the gear..."
            className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500 resize-none"
          />
        </div>

        {/* Catalog image upload */}
        {(onUploadCatalogImage || usesModerationFlow) && (
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">
              Catalog Image (optional)
              <span className="ml-2 text-xs text-primary-400">(Max 1MB, JPEG/PNG/WebP)</span>
            </label>

            <div className="flex items-start gap-3">
              <button
                type="button"
                onClick={handleOpenImageModal}
                className="px-3 py-2 bg-slate-700 hover:bg-slate-600 border border-slate-600 rounded-lg text-sm text-white transition-colors"
              >
                {selectedImage ? 'Choose Different' : 'Add Image'}
              </button>

              {selectedImage && (
                <div className="relative">
                  <img
                    src={selectedImage.previewUrl}
                    alt="Catalog image preview"
                    className="w-20 h-20 object-cover rounded-lg bg-slate-700"
                  />
                  <button
                    type="button"
                    onClick={handleRemoveImage}
                    className="absolute -top-2 -right-2 p-1 bg-red-600 hover:bg-red-700 text-white rounded-full transition-colors"
                    title="Remove image"
                  >
                    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>
              )}
            </div>

            {usesModerationFlow && selectedImage && (
              <p className="mt-2 text-xs text-slate-400">
                {selectedImage.uploadId ? 'Approved image ready to save.' : 'Image is not approved yet.'}
              </p>
            )}

            <p className="text-xs text-slate-500 mt-2">
              {usesModerationFlow
                ? 'Image moderation runs in the modal before this can be saved.'
                : 'This uploads to the shared catalog item image and stays in admin review.'}
            </p>
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="flex items-center justify-end gap-3 px-6 py-4 border-t border-slate-700 bg-slate-800/50 flex-shrink-0">
        <button
          type="button"
          onClick={onCancel}
          className="px-4 py-2 text-slate-300 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
        >
          Back to Search
        </button>
        <button
          type="submit"
          disabled={isSubmitting || !brand.trim() || !model.trim() || checkingDuplicates}
          className="px-4 py-2 bg-primary-600 hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors flex items-center gap-2"
        >
          {isSubmitting || checkingDuplicates ? (
            <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
          ) : (
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
          )}
          Add to Catalog
        </button>
      </div>

      {showImageModal && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-[70] p-4">
          <div className="bg-slate-800 rounded-xl p-6 max-w-md w-full shadow-2xl border border-slate-700">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-white">Edit Gear Image</h3>
              <button
                type="button"
                onClick={handleCloseImageModal}
                className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors disabled:opacity-50"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>

            <div className="flex flex-col items-center gap-4 mb-4">
              <div className="w-36 h-36 rounded-lg overflow-hidden border-2 border-slate-600 bg-slate-700">
                {modalImage?.previewUrl ? (
                  <img src={modalImage.previewUrl} alt="Gear preview" className="w-full h-full object-cover" />
                ) : selectedImage?.previewUrl ? (
                  <img src={selectedImage.previewUrl} alt="Current gear image" className="w-full h-full object-cover" />
                ) : (
                  <div className="w-full h-full flex items-center justify-center text-slate-500 text-4xl">📦</div>
                )}
              </div>

              <input
                ref={imageInputRef}
                type="file"
                accept="image/jpeg,image/png,image/webp"
                onChange={handleImageFileChange}
                className="hidden"
              />
              <button
                type="button"
                onClick={() => imageInputRef.current?.click()}
                disabled={isImageUploading}
                className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-lg text-sm transition-colors disabled:opacity-50"
              >
                {modalImage?.previewUrl ? 'Choose Different' : 'Select Image'}
              </button>
              <p className="text-xs text-slate-500">JPEG, PNG, or WebP. Max 1MB.</p>
            </div>

            {usesModerationFlow && imageStatusText && (
              <div
                className={`mb-4 p-3 rounded-lg text-sm border ${
                  imageStatusTone === 'success'
                    ? 'bg-green-500/10 border-green-500/30 text-green-400'
                    : imageStatusTone === 'error'
                      ? 'bg-red-500/10 border-red-500/30 text-red-400'
                      : 'bg-slate-700/50 border-slate-600 text-slate-300'
                }`}
              >
                <p>{imageStatusText}</p>
                {modalImage?.moderationReason && imageStatusTone !== 'success' && (
                  <p className="mt-1 text-xs text-slate-300">{modalImage.moderationReason}</p>
                )}
              </div>
            )}

            <div className="flex gap-2">
              <button
                type="button"
                onClick={handleCloseImageModal}
                className="flex-1 px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-lg text-sm transition-colors disabled:opacity-50"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleSaveImageSelection}
                disabled={
                  isImageUploading ||
                  !modalImage ||
                  (usesModerationFlow && (!modalImage.uploadId || modalImage.moderationStatus !== 'APPROVED'))
                }
                className="flex-1 px-4 py-2 bg-primary-600 hover:bg-primary-500 text-white rounded-lg text-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Save
              </button>
            </div>
          </div>
        </div>
      )}
    </form>
  );
}

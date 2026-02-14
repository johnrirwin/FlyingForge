import { useState, useEffect, useCallback, useRef } from 'react';
import type { GearCatalogItem, GearType, CreateGearCatalogParams, DroneType, NearMatch } from '../gearCatalogTypes';
import { GEAR_TYPES, DRONE_TYPES, getCatalogItemDisplayName } from '../gearCatalogTypes';
import { searchGearCatalog, createGearCatalogItem, findNearMatches, getPopularGear } from '../gearCatalogApi';
import { adminFindNearMatches } from '../adminApi';
import { ImageUploadModal } from './ImageUploadModal';

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
  enableJsonImport?: boolean;
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
  enableJsonImport = false,
  onUploadCatalogImage,
  onModerateCatalogImage,
  onSaveCatalogImageUpload,
}: CatalogSearchModalProps) {
  const [query, setQuery] = useState('');
  const [gearType, setGearType] = useState<GearType | ''>(initialGearType || '');
  const [results, setResults] = useState<GearCatalogItem[]>([]);
  const [popularItems, setPopularItems] = useState<GearCatalogItem[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  type CatalogModalMode = 'search' | 'create' | 'import-json';
  const [mode, setMode] = useState<CatalogModalMode>(startInCreateMode ? 'create' : 'search');
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
      setMode(startInCreateMode ? 'create' : 'search');
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
              {mode === 'create'
                ? 'Add New Gear to Catalog'
                : mode === 'import-json'
                  ? 'Import Gear from JSON'
                  : 'Search Gear Catalog'}
            </h2>
            <p className="text-sm text-slate-400 mt-0.5">
              {mode === 'create'
                ? 'Create a new entry in the shared gear catalog'
                : mode === 'import-json'
                  ? 'Upload a JSON file and review items before saving'
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

        {(mode === 'create' || mode === 'import-json') && enableJsonImport && (
          <div className="px-6 py-3 border-b border-slate-700 bg-slate-800/40">
            <div className="inline-flex rounded-lg border border-slate-700 bg-slate-900/60 p-1">
              <button
                type="button"
                onClick={() => setMode('create')}
                className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                  mode === 'create'
                    ? 'bg-primary-600 text-white'
                    : 'text-slate-300 hover:bg-slate-700 hover:text-white'
                }`}
              >
                Single Item
              </button>
              <button
                type="button"
                onClick={() => setMode('import-json')}
                className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                  mode === 'import-json'
                    ? 'bg-primary-600 text-white'
                    : 'text-slate-300 hover:bg-slate-700 hover:text-white'
                }`}
              >
                Import JSON
              </button>
            </div>
          </div>
        )}

        {mode === 'create' ? (
          <CreateCatalogItemForm
            initialGearType={gearType || undefined}
            initialQuery={query}
            onSuccess={handleSelectItem}
            onCancel={() => setMode('search')}
            onUploadCatalogImage={onUploadCatalogImage}
            onModerateCatalogImage={onModerateCatalogImage}
            onSaveCatalogImageUpload={onSaveCatalogImageUpload}
            onImportJson={enableJsonImport ? () => setMode('import-json') : undefined}
          />
        ) : mode === 'import-json' ? (
          <ImportCatalogItemsForm
            onSuccess={handleSelectItem}
            onCancel={() => setMode(startInCreateMode ? 'create' : 'search')}
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
                  className="h-10 px-3 bg-slate-700 border border-slate-600 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500"
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
                    onClick={() => setMode('create')}
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
                onClick={() => setMode('create')}
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
  onImportJson?: () => void;
  onUploadCatalogImage?: (itemId: string, imageFile: File) => Promise<void>;
  onModerateCatalogImage?: (imageFile: File) => Promise<ModerationResult>;
  onSaveCatalogImageUpload?: (itemId: string, uploadId: string) => Promise<void>;
}

function CreateCatalogItemForm({
  initialGearType,
  initialQuery,
  onSuccess,
  onCancel,
  onImportJson,
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
  const [imageModalError, setImageModalError] = useState<string | null>(null);

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

  const handleImageFileChange = async (file: File) => {
    // Keep validation aligned with gear catalog image endpoint.
    if (file.size > 2 * 1024 * 1024) {
      setImageModalError('Image file is too large. Maximum size is 2MB.');
      return;
    }

    const validTypes = ['image/jpeg', 'image/jpg', 'image/png'];
    if (!validTypes.includes(file.type)) {
      setImageModalError('Invalid image type. Please use JPEG or PNG.');
      return;
    }

    setImageModalError(null);
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
      const message = err instanceof Error ? err.message : 'Unable to verify image right now';
      setModalImage({
        previewUrl,
        moderationStatus: 'PENDING_REVIEW',
      });
      setImageStatusTone('error');
      setImageStatusText('Unable to verify right now');
      setImageModalError(message);
      setError(message);
    } finally {
      setIsImageUploading(false);
    }
  };

  const handleOpenImageModal = () => {
    setShowImageModal(true);
    setImageModalError(null);
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
    setImageModalError(null);
    setIsImageUploading(false);
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
    setImageModalError(null);
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
    setImageModalError(null);
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
            className="w-full h-11 px-3 bg-slate-700 border border-slate-600 rounded-lg text-white focus:outline-none focus:border-primary-500"
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
              <span className="ml-2 text-xs text-primary-400">(Max 2MB, JPEG/PNG)</span>
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
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 px-6 py-4 border-t border-slate-700 bg-slate-800/50 flex-shrink-0">
        <div className="flex flex-col sm:flex-row sm:items-center gap-3">
          <button
            type="button"
            onClick={onCancel}
            className="px-4 py-2 text-slate-300 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
          >
            Back to Search
          </button>
          {onImportJson && (
            <button
              type="button"
              onClick={onImportJson}
              className="text-primary-400 hover:text-primary-300 text-sm font-medium transition-colors flex items-center gap-1 justify-center sm:justify-start"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
              </svg>
              Import from JSON
            </button>
          )}
        </div>
        <button
          type="submit"
          disabled={isSubmitting || !brand.trim() || !model.trim() || checkingDuplicates}
          className="px-4 py-2 bg-primary-600 hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors flex items-center gap-2 justify-center"
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

      <ImageUploadModal
        isOpen={showImageModal}
        title="Edit Gear Image"
        previewUrl={modalImage?.previewUrl || selectedImage?.previewUrl || null}
        previewAlt={modalImage?.previewUrl ? 'Gear preview' : 'Current gear image'}
        placeholder="📦"
        accept="image/jpeg,image/jpg,image/png"
        helperText="JPEG or PNG. Max 2MB."
        selectButtonLabel={modalImage?.previewUrl ? 'Choose Different' : 'Select Image'}
        onSelectFile={handleImageFileChange}
        onClose={handleCloseImageModal}
        onSave={handleSaveImageSelection}
        disableSelect={isImageUploading}
        disableSave={
          isImageUploading ||
          !modalImage ||
          (usesModerationFlow && (!modalImage.uploadId || modalImage.moderationStatus !== 'APPROVED'))
        }
        statusText={usesModerationFlow ? imageStatusText : null}
        statusTone={imageStatusTone}
        statusReason={modalImage?.moderationReason}
        errorMessage={imageModalError}
      />
    </form>
  );
}

interface ImportCatalogItemsFormProps {
  onSuccess: (item: GearCatalogItem) => void;
  onCancel: () => void;
}

type ImportCatalogRow = {
  id: string;
  params: CreateGearCatalogParams;
  duplicates?: NearMatch[];
  duplicateError?: string;
  result?: 'created' | 'existing' | 'error';
  error?: string;
};

function ImportCatalogItemsForm({ onSuccess, onCancel }: ImportCatalogItemsFormProps) {
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [fileName, setFileName] = useState<string | null>(null);
  const [rows, setRows] = useState<ImportCatalogRow[]>([]);
  const [parseErrors, setParseErrors] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [progress, setProgress] = useState<{ current: number; total: number } | null>(null);
  const [summary, setSummary] = useState<{ created: number; existing: number; failed: number } | null>(null);
  const [isCheckingDuplicates, setIsCheckingDuplicates] = useState(false);
  const [duplicateProgress, setDuplicateProgress] = useState<{ current: number; total: number } | null>(null);
  const duplicateCheckRequestIdRef = useRef(0);

  const createRowId = () => {
    if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
      return crypto.randomUUID();
    }
    return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
  };

  const validGearTypes = useRef(new Set(GEAR_TYPES.map((t) => t.value)));
  const validDroneTypes = useRef(new Set(DRONE_TYPES.map((t) => t.value)));

  useEffect(() => {
    return () => {
      // Cancel any in-flight duplicate checks.
      duplicateCheckRequestIdRef.current += 1;
    };
  }, []);

  const cancelDuplicateChecks = useCallback(() => {
    duplicateCheckRequestIdRef.current += 1;
    setIsCheckingDuplicates(false);
    setDuplicateProgress(null);
  }, []);

  const checkDuplicates = useCallback(async (targetRows: ImportCatalogRow[]) => {
    if (targetRows.length === 0) return;

    const requestId = ++duplicateCheckRequestIdRef.current;
    setIsCheckingDuplicates(true);
    setDuplicateProgress({ current: 0, total: targetRows.length });
    setRows((prev) => prev.map((row) => ({ ...row, duplicates: undefined, duplicateError: undefined })));

    for (let i = 0; i < targetRows.length; i++) {
      setDuplicateProgress({ current: i + 1, total: targetRows.length });
      const row = targetRows[i];

      try {
        const response = await adminFindNearMatches({
          gearType: row.params.gearType,
          brand: row.params.brand,
          model: row.params.model,
          threshold: 0.3,
        });

        if (duplicateCheckRequestIdRef.current !== requestId) return;
        setRows((prev) =>
          prev.map((candidate) =>
            candidate.id === row.id ? { ...candidate, duplicates: response.matches ?? [] } : candidate
          )
        );
      } catch (err) {
        if (duplicateCheckRequestIdRef.current !== requestId) return;
        const message = err instanceof Error ? err.message : 'Failed to check for duplicates';
        setRows((prev) =>
          prev.map((candidate) =>
            candidate.id === row.id ? { ...candidate, duplicates: [], duplicateError: message } : candidate
          )
        );
      }
    }

    if (duplicateCheckRequestIdRef.current === requestId) {
      setIsCheckingDuplicates(false);
      setDuplicateProgress(null);
    }
  }, []);

  const parseFile = useCallback(
    async (file: File) => {
      setError(null);
      setSummary(null);
      setProgress(null);
      cancelDuplicateChecks();

      const text = await file.text();
      let data: unknown;
      try {
        data = JSON.parse(text);
      } catch (err) {
        throw new Error(err instanceof Error ? err.message : 'Invalid JSON');
      }

      let items: unknown[] | null = null;
      if (Array.isArray(data)) {
        items = data;
      } else if (data && typeof data === 'object') {
        const obj = data as Record<string, unknown>;
        if (Array.isArray(obj.items)) items = obj.items;
        else if (Array.isArray(obj.catalog)) items = obj.catalog;
      }

      if (!items) {
        throw new Error('JSON must be an array of items (or an object with an "items" array)');
      }

      const nextErrors: string[] = [];
      const nextRows: ImportCatalogRow[] = [];

      items.forEach((raw, index) => {
        if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
          nextErrors.push(`Row ${index + 1}: expected an object`);
          return;
        }

        const item = raw as Record<string, unknown>;
        const rawGearType = item.gearType;
        const rawBrand = item.brand;
        const rawModel = item.model;

        if (typeof rawGearType !== 'string' || !validGearTypes.current.has(rawGearType as GearType)) {
          nextErrors.push(`Row ${index + 1}: invalid gearType "${String(rawGearType)}"`);
          return;
        }

        const brand = typeof rawBrand === 'string' ? rawBrand.trim() : '';
        const model = typeof rawModel === 'string' ? rawModel.trim() : '';
        if (!brand) {
          nextErrors.push(`Row ${index + 1}: brand is required`);
          return;
        }
        if (!model) {
          nextErrors.push(`Row ${index + 1}: model is required`);
          return;
        }

        const variant = typeof item.variant === 'string' ? item.variant.trim() : '';
        const description = typeof item.description === 'string' ? item.description.trim() : '';

        let msrp: number | undefined;
        if (typeof item.msrp === 'number' && Number.isFinite(item.msrp)) {
          msrp = item.msrp;
        } else if (typeof item.msrp === 'string' && item.msrp.trim()) {
          const parsed = Number(item.msrp.trim());
          if (Number.isFinite(parsed)) {
            msrp = parsed;
          }
        }

        let specs: Record<string, unknown> | undefined;
        if (item.specs && typeof item.specs === 'object' && !Array.isArray(item.specs)) {
          specs = item.specs as Record<string, unknown>;
        }

        let bestFor: DroneType[] | undefined;
        if (Array.isArray(item.bestFor)) {
          const filtered = item.bestFor.filter(
            (value): value is DroneType =>
              typeof value === 'string' && validDroneTypes.current.has(value as DroneType)
          );
          if (filtered.length > 0) {
            bestFor = filtered;
          }
        }

        const params: CreateGearCatalogParams = {
          gearType: rawGearType as GearType,
          brand,
          model,
          variant: variant || undefined,
          description: description || undefined,
          msrp,
          specs,
          bestFor,
        };

        nextRows.push({ id: createRowId(), params });
      });

      setFileName(file.name);
      setParseErrors(nextErrors);
      setRows(nextRows);
      void checkDuplicates(nextRows);
    },
    [cancelDuplicateChecks, checkDuplicates]
  );

  const handleFileSelected = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    // Allow choosing the same file again.
    event.target.value = '';
    if (!file) return;

    setIsSubmitting(false);
    setError(null);

    try {
      await parseFile(file);
    } catch (err) {
      setFileName(file.name);
      setRows([]);
      setParseErrors([]);
      setError(err instanceof Error ? err.message : 'Failed to parse JSON file');
    }
  };

  const handleDeleteRow = (id: string) => {
    if (isSubmitting) return;
    setRows((prev) => prev.filter((row) => row.id !== id));
  };

  const handleUploadClick = () => {
    if (isSubmitting) return;
    fileInputRef.current?.click();
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (rows.length === 0) {
      setError('Upload a JSON file with at least one valid item');
      return;
    }

    cancelDuplicateChecks();
    setIsSubmitting(true);
    setError(null);
    setSummary(null);

    const nextRows: ImportCatalogRow[] = rows.map((row) => ({ ...row, result: undefined, error: undefined }));
    let created = 0;
    let existing = 0;
    let failed = 0;
    let firstItem: GearCatalogItem | null = null;

    for (let i = 0; i < nextRows.length; i++) {
      setProgress({ current: i + 1, total: nextRows.length });
      try {
        const response = await createGearCatalogItem(nextRows[i].params);
        if (!firstItem) firstItem = response.item;
        if (response.existing) {
          existing += 1;
          nextRows[i].result = 'existing';
        } else {
          created += 1;
          nextRows[i].result = 'created';
        }
      } catch (err) {
        failed += 1;
        nextRows[i].result = 'error';
        nextRows[i].error = err instanceof Error ? err.message : 'Failed to save item';
      }
    }

    setRows(nextRows);
    setProgress(null);
    setSummary({ created, existing, failed });
    setIsSubmitting(false);

    if (failed === 0 && firstItem) {
      onSuccess(firstItem);
    } else if (failed > 0) {
      setError(`Saved ${created + existing} item${created + existing === 1 ? '' : 's'} with ${failed} failure${failed === 1 ? '' : 's'}.`);
    }
  };

  const possibleDuplicateCount = rows.filter((row) => (row.duplicates?.length ?? 0) > 0).length;

  return (
    <form onSubmit={handleSubmit} className="flex flex-col min-h-0 flex-1">
      <div className="flex-1 overflow-y-auto p-6 space-y-4 min-h-0">
        {error && (
          <div className="p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">
            {error}
          </div>
        )}

        <div className="rounded-xl border border-slate-700 bg-slate-900/40 p-4">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
            <div className="min-w-0">
              <p className="text-sm font-medium text-white">Upload JSON file</p>
              <p className="text-xs text-slate-400 mt-1">
                Expected fields: <span className="text-slate-300">gearType, brand, model</span>. Optional: variant, specs, bestFor, msrp, description.
              </p>
              {fileName && (
                <p className="text-xs text-slate-500 mt-2 truncate">
                  File: <span className="text-slate-300">{fileName}</span>
                </p>
              )}
            </div>
            <div className="flex items-center gap-2">
              <input
                ref={fileInputRef}
                type="file"
                accept="application/json,.json"
                onChange={handleFileSelected}
                className="hidden"
              />
              <button
                type="button"
                onClick={handleUploadClick}
                disabled={isSubmitting}
                className="px-3 py-2 bg-slate-700 hover:bg-slate-600 border border-slate-600 rounded-lg text-sm text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Choose File
              </button>
              {rows.length > 0 && (
                <button
                  type="button"
                  onClick={() => void checkDuplicates(rows)}
                  disabled={isSubmitting || isCheckingDuplicates}
                  className="px-3 py-2 bg-slate-700 hover:bg-slate-600 border border-slate-600 rounded-lg text-sm text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {isCheckingDuplicates ? 'Checking…' : 'Re-check duplicates'}
                </button>
              )}
            </div>
          </div>
        </div>

        {parseErrors.length > 0 && (
          <div className="p-3 bg-yellow-500/10 border border-yellow-500/20 rounded-lg text-yellow-300 text-sm">
            <p className="font-medium mb-1">Some rows were skipped:</p>
            <ul className="list-disc list-inside space-y-1">
              {parseErrors.slice(0, 8).map((msg, idx) => (
                <li key={idx}>{msg}</li>
              ))}
            </ul>
            {parseErrors.length > 8 && (
              <p className="mt-2 text-xs text-yellow-200/80">
                And {parseErrors.length - 8} more…
              </p>
            )}
          </div>
        )}

        {duplicateProgress && (
          <div className="flex items-center gap-3 text-sm text-slate-300">
            <div className="w-4 h-4 border-2 border-slate-500/40 border-t-yellow-400 rounded-full animate-spin" />
            Checking duplicates {duplicateProgress.current} / {duplicateProgress.total}…
          </div>
        )}

        {possibleDuplicateCount > 0 && (
          <div className="p-3 bg-yellow-500/10 border border-yellow-500/20 rounded-lg text-yellow-300 text-sm">
            Possible duplicates found for {possibleDuplicateCount} item{possibleDuplicateCount === 1 ? '' : 's'}. Review before saving.
          </div>
        )}

        {summary && (
          <div className="p-3 bg-slate-700/30 border border-slate-600 rounded-lg text-slate-200 text-sm">
            Imported results: <span className="text-green-300">{summary.created} created</span>,{' '}
            <span className="text-blue-300">{summary.existing} existing</span>,{' '}
            <span className={summary.failed ? 'text-red-300' : 'text-slate-300'}>
              {summary.failed} failed
            </span>
            .
          </div>
        )}

        {progress && (
          <div className="flex items-center gap-3 text-sm text-slate-300">
            <div className="w-4 h-4 border-2 border-slate-500/40 border-t-primary-500 rounded-full animate-spin" />
            Saving {progress.current} / {progress.total}…
          </div>
        )}

        <div>
          <div className="flex items-center justify-between gap-3 mb-3">
            <h3 className="text-sm font-medium text-slate-300">
              Review Items ({rows.length})
            </h3>
            {rows.length > 0 && (
              <p className="text-xs text-slate-500">Delete any rows you don’t want to import.</p>
            )}
          </div>

          {rows.length === 0 ? (
            <div className="p-6 text-center text-slate-400 border border-slate-700 bg-slate-900/40 rounded-xl">
              Upload a JSON file to preview items here.
            </div>
          ) : (
            <div className="space-y-2">
              {rows.map((row) => {
                const duplicates = row.duplicates ?? [];
                const hasDuplicates = duplicates.length > 0;
                const borderClass =
                  row.result === 'error'
                    ? 'border-red-500/40'
                    : row.result === 'created'
                      ? 'border-green-500/30'
                      : row.result === 'existing'
                        ? 'border-blue-500/30'
                        : hasDuplicates
                          ? 'border-yellow-500/30'
                          : 'border-slate-600';

                return (
                  <div
                    key={row.id}
                    className={`p-3 rounded-lg border bg-slate-700/30 flex items-start justify-between gap-3 ${borderClass}`}
                  >
                    <div className="min-w-0">
                      <p className="text-white font-medium truncate">
                        {row.params.brand} {row.params.model}
                        {row.params.variant ? ` ${row.params.variant}` : ''}
                      </p>
                      <p className="text-xs text-slate-400 mt-0.5">
                        <span className="px-2 py-0.5 bg-slate-600/60 text-slate-200 rounded text-[11px]">
                          {row.params.gearType}
                        </span>
                        {row.params.bestFor && row.params.bestFor.length > 0 && (
                          <span className="ml-2 text-slate-500">
                            • bestFor: {row.params.bestFor.join(', ')}
                          </span>
                        )}
                        {typeof row.params.msrp === 'number' && (
                          <span className="ml-2 text-slate-500">• MSRP: ${row.params.msrp}</span>
                        )}
                      </p>

                      {row.result === 'created' && (
                        <p className="mt-1 text-xs text-green-300">Created</p>
                      )}
                      {row.result === 'existing' && (
                        <p className="mt-1 text-xs text-blue-300">Already exists</p>
                      )}
                      {row.result === 'error' && row.error && (
                        <p className="mt-1 text-xs text-red-300">Error: {row.error}</p>
                      )}

                      {row.duplicateError && (
                        <p className="mt-2 text-xs text-yellow-300">
                          Unable to check duplicates: {row.duplicateError}
                        </p>
                      )}

                      {hasDuplicates && (
                        <div className="mt-2 p-2 bg-yellow-500/10 border border-yellow-500/20 rounded text-xs text-yellow-300">
                          <p className="font-medium">
                            Possible duplicate{duplicates.length === 1 ? '' : 's'}:
                          </p>
                          <ul className="list-disc list-inside mt-1 space-y-0.5">
                            {duplicates.slice(0, 3).map((match) => {
                              const label = getCatalogItemDisplayName(match.item);
                              const status = match.item.status ? ` • ${match.item.status}` : '';
                              const similarity =
                                typeof match.similarity === 'number' && match.similarity > 0
                                  ? ` • ${Math.round(match.similarity * 100)}%`
                                  : '';

                              return (
                                <li key={match.item.id} className="text-yellow-200/90">
                                  {label}
                                  {status}
                                  {similarity}
                                </li>
                              );
                            })}
                          </ul>
                        </div>
                      )}
                    </div>
                    <button
                      type="button"
                      onClick={() => handleDeleteRow(row.id)}
                      disabled={isSubmitting}
                      className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                      aria-label={`Remove ${row.params.brand} ${row.params.model}`}
                    >
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                      </svg>
                    </button>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>

      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 px-6 py-4 border-t border-slate-700 bg-slate-800/50 flex-shrink-0">
        <button
          type="button"
          onClick={onCancel}
          disabled={isSubmitting}
          className="px-4 py-2 text-slate-300 hover:text-white hover:bg-slate-700 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        >
          Back
        </button>
        <button
          type="submit"
          disabled={isSubmitting || isCheckingDuplicates || rows.length === 0}
          className="px-4 py-2 bg-primary-600 hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors flex items-center gap-2 justify-center"
        >
          {isSubmitting ? (
            <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
          ) : (
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
          )}
          Save {rows.length} Item{rows.length === 1 ? '' : 's'}
        </button>
      </div>
    </form>
  );
}

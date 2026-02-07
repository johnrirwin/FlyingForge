import { useState, useEffect, useCallback, useRef } from 'react';
import type { EquipmentItem, InventoryItem, EquipmentCategory, ItemCondition, AddInventoryParams } from '../equipmentTypes';
import { EQUIPMENT_CATEGORIES, ITEM_CONDITIONS } from '../equipmentTypes';
import type { GearCatalogItem } from '../gearCatalogTypes';
import { getCatalogItemDisplayName, gearTypeToEquipmentCategory } from '../gearCatalogTypes';
import { CatalogSearchModal } from './CatalogSearchModal';

interface AddGearModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (params: AddInventoryParams) => Promise<void>;
  equipmentItem?: EquipmentItem | null;
  catalogItem?: GearCatalogItem | null; // Pre-selected from gear catalog page
  editItem?: InventoryItem | null;
}

export function AddGearModal({ isOpen, onClose, onSubmit, equipmentItem, catalogItem, editItem }: AddGearModalProps) {
  // Only show details form for editing existing items or when coming from shop
  const isEditing = !!editItem;
  const hasEquipmentItem = !!equipmentItem;
  const hasPreselectedCatalogItem = !!catalogItem;
  
  // If we have a pre-selected catalog item, auto-add it
  const showDetailsForm = isEditing || hasEquipmentItem;
  const showCatalogSearch = !showDetailsForm && !hasPreselectedCatalogItem;
  
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  // Track whether auto-add has been triggered to prevent duplicate submissions
  const autoAddTriggeredRef = useRef<string | null>(null);

  // Form state (for editing/equipment items)
  const [name, setName] = useState('');
  const [category, setCategory] = useState<EquipmentCategory>('accessories');
  const [manufacturer, setManufacturer] = useState('');
  const [quantity, setQuantity] = useState(1);
  const [condition, setCondition] = useState<ItemCondition>('new');
  const [purchasePrice, setPurchasePrice] = useState('');
  const [purchaseSeller, setPurchaseSeller] = useState('');
  const [purchaseDate, setPurchaseDate] = useState('');
  const [notes, setNotes] = useState('');
  const [buildId, setBuildId] = useState('');
  const [imageUrl, setImageUrl] = useState('');

  // Auto-add pre-selected catalog item to inventory
  const autoAddCatalogItem = useCallback(async (item: GearCatalogItem) => {
    setIsSubmitting(true);
    setError(null);
    
    try {
      const params: AddInventoryParams = {
        name: getCatalogItemDisplayName(item),
        category: gearTypeToEquipmentCategory(item.gearType),
        manufacturer: item.brand,
        quantity: 1,
        condition: 'new',
        purchasePrice: item.msrp,
        imageUrl: item.imageUrl,
        catalogId: item.id,
      };

      console.log('[AddGearModal] Auto-adding catalog item:', params);
      await onSubmit(params);
      console.log('[AddGearModal] Auto-add successful');
      onClose();
    } catch (err) {
      console.error('[AddGearModal] Auto-add failed:', err);
      setError(err instanceof Error ? err.message : 'Failed to add item');
    } finally {
      setIsSubmitting(false);
    }
  }, [onSubmit, onClose]);

  // Auto-add pre-selected catalog item when modal opens
  useEffect(() => {
    // Only auto-add if we haven't already triggered for this catalog item
    // This prevents duplicate submissions when callback references change
    if (isOpen && hasPreselectedCatalogItem && catalogItem && !isEditing) {
      if (autoAddTriggeredRef.current !== catalogItem.id) {
        autoAddTriggeredRef.current = catalogItem.id;
        autoAddCatalogItem(catalogItem);
      }
    }
    // Reset the ref when modal closes
    if (!isOpen) {
      autoAddTriggeredRef.current = null;
    }
  }, [isOpen, hasPreselectedCatalogItem, catalogItem, isEditing, autoAddCatalogItem]);

  // Handler when catalog item is selected from search
  const handleCatalogSelect = useCallback((item: GearCatalogItem) => {
    // Auto-add to inventory immediately
    autoAddCatalogItem(item);
  }, [autoAddCatalogItem]);

  // Pre-fill form from equipment item or edit item
  useEffect(() => {
    if (equipmentItem) {
      setName(equipmentItem.name);
      setCategory(equipmentItem.category);
      setManufacturer(equipmentItem.manufacturer || '');
      setPurchasePrice(equipmentItem.price.toFixed(2));
      setPurchaseSeller(equipmentItem.seller);
      setImageUrl(equipmentItem.imageUrl || '');
      setQuantity(1);
      setCondition('new');
      setNotes('');
      setBuildId('');
      setPurchaseDate('');
    } else if (editItem) {
      setName(editItem.name);
      setCategory(editItem.category);
      setManufacturer(editItem.manufacturer || '');
      setQuantity(editItem.quantity);
      setCondition(editItem.condition);
      setPurchasePrice(editItem.purchasePrice?.toFixed(2) || '');
      setPurchaseSeller(editItem.purchaseSeller || '');
      setPurchaseDate(editItem.purchaseDate ? editItem.purchaseDate.split('T')[0] : '');
      setNotes(editItem.notes || '');
      setBuildId(editItem.buildId || '');
      setImageUrl(editItem.imageUrl || '');
    } else {
      // Reset form
      setName('');
      setCategory('accessories');
      setManufacturer('');
      setQuantity(1);
      setCondition('new');
      setPurchasePrice('');
      setPurchaseSeller('');
      setPurchaseDate('');
      setNotes('');
      setBuildId('');
      setImageUrl('');
    }
    setError(null);
  }, [equipmentItem, editItem, isOpen]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);
    setError(null);

    try {
      const params: AddInventoryParams = {
        name: name.trim(),
        category,
        manufacturer: manufacturer.trim() || undefined,
        quantity,
        condition,
        purchasePrice: purchasePrice ? parseFloat(purchasePrice) : undefined,
        purchaseSeller: purchaseSeller.trim() || undefined,
        purchaseDate: purchaseDate || undefined,
        notes: notes.trim() || undefined,
        buildId: buildId.trim() || undefined,
        imageUrl: imageUrl.trim() || undefined,
        sourceEquipmentId: equipmentItem?.id,
      };

      console.log('[AddGearModal] Submitting inventory params:', params);
      await onSubmit(params);
      console.log('[AddGearModal] Submit successful');
      onClose();
    } catch (err) {
      console.error('[AddGearModal] Submit failed:', err);
      setError(err instanceof Error ? err.message : 'Failed to save item');
    } finally {
      setIsSubmitting(false);
    }
  };

  if (!isOpen) return null;

  // Show loading state when auto-adding
  if (isSubmitting && hasPreselectedCatalogItem) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" />
        <div className="relative bg-slate-800 border border-slate-700 rounded-2xl p-8 flex flex-col items-center gap-4">
          <div className="w-8 h-8 border-2 border-primary-400 border-t-transparent rounded-full animate-spin" />
          <p className="text-white">Adding to inventory...</p>
        </div>
      </div>
    );
  }

  // Show error if auto-add failed
  if (error && hasPreselectedCatalogItem) {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />
        <div className="relative bg-slate-800 border border-slate-700 rounded-2xl p-6 max-w-sm">
          <p className="text-red-400 mb-4">{error}</p>
          <button
            onClick={onClose}
            className="w-full px-4 py-2 bg-slate-700 hover:bg-slate-600 text-white rounded-lg"
          >
            Close
          </button>
        </div>
      </div>
    );
  }

  // Show catalog search
  if (showCatalogSearch) {
    return (
      <CatalogSearchModal
        isOpen={true}
        onClose={onClose}
        onSelectItem={handleCatalogSelect}
      />
    );
  }

  // Show details form (only for editing or adding from equipment shop)
  const title = editItem 
    ? 'Edit Inventory Item' 
    : 'Add to My Inventory';

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="relative bg-slate-800 border border-slate-700 rounded-2xl w-full max-w-lg max-h-[90vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-700">
          <h2 className="text-lg font-semibold text-white">{title}</h2>
          <button
            onClick={onClose}
            className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="flex-1 overflow-y-auto">
          <div className="p-6 space-y-4">
            {/* Error */}
            {error && (
              <div className="p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">
                {error}
              </div>
            )}

            {/* Name */}
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-1">
                Name <span className="text-red-400">*</span>
              </label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
                className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500"
                placeholder="Item name"
              />
            </div>

            {/* Category & Condition */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Category <span className="text-red-400">*</span>
                </label>
                <select
                  value={category}
                  onChange={(e) => setCategory(e.target.value as EquipmentCategory)}
                  required
                  className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white focus:outline-none focus:border-primary-500"
                >
                  {EQUIPMENT_CATEGORIES.map(cat => (
                    <option key={cat.value} value={cat.value}>{cat.label}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Condition <span className="text-red-400">*</span>
                </label>
                <select
                  value={condition}
                  onChange={(e) => setCondition(e.target.value as ItemCondition)}
                  required
                  className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white focus:outline-none focus:border-primary-500"
                >
                  {ITEM_CONDITIONS.map(cond => (
                    <option key={cond.value} value={cond.value}>{cond.label}</option>
                  ))}
                </select>
              </div>
            </div>

            {/* Manufacturer & Quantity */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Manufacturer
                </label>
                <input
                  type="text"
                  value={manufacturer}
                  onChange={(e) => setManufacturer(e.target.value)}
                  className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500"
                  placeholder="Brand"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Quantity
                </label>
                <input
                  type="number"
                  value={quantity}
                  onChange={(e) => setQuantity(parseInt(e.target.value) || 1)}
                  min={1}
                  className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white focus:outline-none focus:border-primary-500"
                />
              </div>
            </div>

            {/* Purchase Price & Seller */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Purchase Price
                </label>
                <div className="relative">
                  <span className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400">$</span>
                  <input
                    type="number"
                    step="0.01"
                    value={purchasePrice}
                    onChange={(e) => setPurchasePrice(e.target.value)}
                    className="w-full pl-7 pr-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500"
                    placeholder="0.00"
                  />
                </div>
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Purchased From
                </label>
                <input
                  type="text"
                  value={purchaseSeller}
                  onChange={(e) => setPurchaseSeller(e.target.value)}
                  className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500"
                  placeholder="Seller name"
                />
              </div>
            </div>

            {/* Purchase Date & Build Name */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Purchase Date
                </label>
                <input
                  type="date"
                  value={purchaseDate}
                  onChange={(e) => setPurchaseDate(e.target.value)}
                  className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white focus:outline-none focus:border-primary-500"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-300 mb-1">
                  Build/Quad Name
                </label>
                <input
                  type="text"
                  value={buildId}
                  onChange={(e) => setBuildId(e.target.value)}
                  className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500"
                  placeholder='e.g., 5" Freestyle'
                />
              </div>
            </div>

            {/* Image URL */}
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-1">
                Image URL
              </label>
              <input
                type="url"
                value={imageUrl}
                onChange={(e) => setImageUrl(e.target.value)}
                className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500"
                placeholder="https://..."
              />
            </div>

            {/* Notes */}
            <div>
              <label className="block text-sm font-medium text-slate-300 mb-1">
                Notes
              </label>
              <textarea
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                rows={3}
                className="w-full px-3 py-2 bg-slate-700 border border-slate-600 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500 resize-none"
                placeholder="Any additional notes... (serial number, personal reminders, etc.)"
              />
            </div>
          </div>

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
              disabled={isSubmitting || !name.trim()}
              className="px-4 py-2 bg-primary-600 hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed text-white font-medium rounded-lg transition-colors flex items-center gap-2"
            >
              {isSubmitting && (
                <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
              )}
              {editItem ? 'Save Changes' : 'Add to My Inventory'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

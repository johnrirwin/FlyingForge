import { useState, useEffect } from 'react';
import type { EquipmentItem, InventoryItem, EquipmentCategory, ItemCondition, AddInventoryParams } from '../equipmentTypes';
import { EQUIPMENT_CATEGORIES, ITEM_CONDITIONS } from '../equipmentTypes';
import type { GearCatalogItem } from '../gearCatalogTypes';
import { getCatalogItemDisplayName, gearTypeToEquipmentCategory } from '../gearCatalogTypes';
import { CatalogSearchModal } from './CatalogSearchModal';

type AddGearStep = 'search' | 'details';

interface AddGearModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (params: AddInventoryParams) => Promise<void>;
  equipmentItem?: EquipmentItem | null;
  catalogItem?: GearCatalogItem | null; // Pre-selected from gear catalog page
  editItem?: InventoryItem | null;
}

export function AddGearModal({ isOpen, onClose, onSubmit, equipmentItem, catalogItem, editItem }: AddGearModalProps) {
  // Determine if we should show catalog search or go straight to details
  const isEditing = !!editItem;
  const hasEquipmentItem = !!equipmentItem;
  const hasPreselectedCatalogItem = !!catalogItem;
  
  const [step, setStep] = useState<AddGearStep>(isEditing || hasEquipmentItem || hasPreselectedCatalogItem ? 'details' : 'search');
  const [selectedCatalogItem, setSelectedCatalogItem] = useState<GearCatalogItem | null>(catalogItem || null);
  
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Form state
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

  // Reset step when modal opens/closes
  useEffect(() => {
    if (isOpen) {
      setStep(isEditing || hasEquipmentItem || hasPreselectedCatalogItem ? 'details' : 'search');
      setSelectedCatalogItem(catalogItem || null);
    }
  }, [isOpen, isEditing, hasEquipmentItem, hasPreselectedCatalogItem, catalogItem]);

  // Pre-fill from various sources
  useEffect(() => {
    if (selectedCatalogItem) {
      // Fill from selected catalog item
      setName(getCatalogItemDisplayName(selectedCatalogItem));
      setCategory(gearTypeToEquipmentCategory(selectedCatalogItem.gearType));
      setManufacturer(selectedCatalogItem.brand);
      setImageUrl(selectedCatalogItem.imageUrl || '');
      setQuantity(1);
      setCondition('new');
      setNotes('');
      setBuildId('');
      setPurchaseDate('');
      setPurchasePrice('');
      setPurchaseSeller('');
    } else if (equipmentItem) {
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
    } else if (!selectedCatalogItem && !equipmentItem && !editItem) {
      // Reset form for completely new entry
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
  }, [selectedCatalogItem, equipmentItem, editItem, isOpen]);

  const handleCatalogSelect = (item: GearCatalogItem) => {
    setSelectedCatalogItem(item);
    setStep('details');
  };

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
        catalogId: selectedCatalogItem?.id,
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

  // Show catalog search step
  if (step === 'search') {
    return (
      <CatalogSearchModal
        isOpen={true}
        onClose={onClose}
        onSelectItem={handleCatalogSelect}
      />
    );
  }

  // Show details form
  const title = editItem 
    ? 'Edit Inventory Item' 
    : selectedCatalogItem 
      ? 'Add to My Inventory' 
      : equipmentItem 
        ? 'Add to My Inventory' 
        : 'Add New Item';

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
          <div className="flex items-center gap-3">
            {!editItem && !hasEquipmentItem && (
              <button
                onClick={() => {
                  setSelectedCatalogItem(null);
                  setStep('search');
                }}
                className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
                </svg>
              </button>
            )}
            <h2 className="text-lg font-semibold text-white">{title}</h2>
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

        {/* Selected catalog item banner */}
        {selectedCatalogItem && (
          <div className="px-6 py-3 bg-primary-600/10 border-b border-slate-700">
            <div className="flex items-center gap-3">
              {selectedCatalogItem.imageUrl ? (
                <img 
                  src={selectedCatalogItem.imageUrl} 
                  alt="" 
                  className="w-10 h-10 rounded-lg object-cover"
                />
              ) : (
                <div className="w-10 h-10 rounded-lg bg-slate-700 flex items-center justify-center">
                  <svg className="w-5 h-5 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
                  </svg>
                </div>
              )}
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-white truncate">
                  {getCatalogItemDisplayName(selectedCatalogItem)}
                </p>
                <p className="text-xs text-slate-400">
                  From community catalog • {selectedCatalogItem.usageCount} users
                </p>
              </div>
              <span className="px-2 py-0.5 bg-primary-600/20 text-primary-400 text-xs rounded-full">
                Linked
              </span>
            </div>
          </div>
        )}

        {/* Form */}
        <form onSubmit={handleSubmit} className="flex-1 overflow-y-auto">
          <div className="p-6 space-y-4">
            {/* Error */}
            {error && (
              <div className="p-3 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm">
                {error}
              </div>
            )}

            {/* Name - hidden if catalog item selected */}
            {!selectedCatalogItem && (
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
            )}

            {/* Category & Condition */}
            <div className="grid grid-cols-2 gap-4">
              {!selectedCatalogItem && (
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
              )}
              <div className={selectedCatalogItem ? 'col-span-2' : ''}>
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
              {!selectedCatalogItem && (
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
              )}
              <div className={selectedCatalogItem ? 'col-span-2' : ''}>
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

            {/* Image URL - only if no catalog item */}
            {!selectedCatalogItem && (
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
            )}

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
              disabled={isSubmitting || (!selectedCatalogItem && !name.trim())}
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

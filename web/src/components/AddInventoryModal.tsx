import { useState, useEffect } from 'react';
import type { EquipmentItem, InventoryItem, EquipmentCategory, ItemCondition, AddInventoryParams } from '../equipmentTypes';
import { EQUIPMENT_CATEGORIES, ITEM_CONDITIONS } from '../equipmentTypes';

interface AddInventoryModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (params: AddInventoryParams) => Promise<void>;
  equipmentItem?: EquipmentItem | null;
  editItem?: InventoryItem | null;
}

export function AddInventoryModal({ isOpen, onClose, onSubmit, equipmentItem, editItem }: AddInventoryModalProps) {
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
  const [notes, setNotes] = useState('');
  const [buildId, setBuildId] = useState('');
  const [imageUrl, setImageUrl] = useState('');

  // Pre-fill from equipment item or edit item
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
    } else if (editItem) {
      setName(editItem.name);
      setCategory(editItem.category);
      setManufacturer(editItem.manufacturer || '');
      setQuantity(editItem.quantity);
      setCondition(editItem.condition);
      setPurchasePrice(editItem.purchasePrice?.toFixed(2) || '');
      setPurchaseSeller(editItem.purchaseSeller || '');
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
        notes: notes.trim() || undefined,
        buildId: buildId.trim() || undefined,
        imageUrl: imageUrl.trim() || undefined,
        sourceEquipmentId: equipmentItem?.id,
      };

      await onSubmit(params);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save item');
    } finally {
      setIsSubmitting(false);
    }
  };

  if (!isOpen) return null;

  const title = editItem ? 'Edit Inventory Item' : equipmentItem ? 'Add to My Inventory' : 'Add New Item';

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
        <form onSubmit={handleSubmit} className="flex-1 overflow-y-auto overflow-x-hidden">
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
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
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
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
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
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
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

            {/* Build Name */}
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
                placeholder="Any additional notes..."
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
              {editItem ? 'Save Changes' : 'Add to Inventory'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

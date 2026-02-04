import type { InventoryItem } from '../equipmentTypes';
import { EQUIPMENT_CATEGORIES, ITEM_CONDITIONS } from '../equipmentTypes';

interface InventoryCardProps {
  item: InventoryItem;
  onEdit: (item: InventoryItem) => void;
  onDelete: (item: InventoryItem) => void;
  onAdjustQuantity: (item: InventoryItem, delta: number) => void;
}

export function InventoryCard({ item, onEdit, onDelete, onAdjustQuantity }: InventoryCardProps) {
  const category = EQUIPMENT_CATEGORIES.find(c => c.value === item.category);
  const condition = ITEM_CONDITIONS.find(c => c.value === item.condition);

  const formatPrice = (price?: number) => {
    if (price === undefined || price === null) return null;
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
    }).format(price);
  };

  const conditionColors: Record<string, string> = {
    new: 'bg-green-500/20 text-green-400',
    used: 'bg-yellow-500/20 text-yellow-400',
    broken: 'bg-red-500/20 text-red-400',
    spare: 'bg-blue-500/20 text-blue-400',
  };

  return (
    <div className="bg-slate-800 border border-slate-700 rounded-xl p-4 hover:border-slate-600 transition-all">
      <div className="flex gap-4">
        {/* Image */}
        <div className="flex-shrink-0 w-20 h-20 rounded-lg overflow-hidden bg-slate-700">
          {item.imageUrl ? (
            <img
              src={item.imageUrl}
              alt={item.name}
              className="w-full h-full object-cover"
              loading="lazy"
              onError={(e) => {
                (e.target as HTMLImageElement).style.display = 'none';
              }}
            />
          ) : (
            <div className="w-full h-full flex items-center justify-center text-slate-500">
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
              </svg>
            </div>
          )}
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          {/* Header */}
          <div className="flex items-start justify-between gap-2 mb-1">
            <div className="flex items-center gap-2 flex-wrap">
              <span className="px-2 py-0.5 bg-slate-700 text-slate-300 rounded text-xs">
                {category?.label || item.category}
              </span>
              <span className={`px-2 py-0.5 rounded text-xs ${conditionColors[item.condition] || 'bg-slate-700 text-slate-300'}`}>
                {condition?.label || item.condition}
              </span>
            </div>
          </div>

          {/* Title */}
          <h3 className="text-white font-medium mb-1 line-clamp-1">
            {item.name}
          </h3>

          {/* Manufacturer & price */}
          <div className="flex items-center gap-2 text-sm text-slate-400 mb-2">
            {item.manufacturer && <span>{item.manufacturer}</span>}
            {item.purchasePrice && (
              <>
                {item.manufacturer && <span>•</span>}
                <span className="text-primary-400">{formatPrice(item.purchasePrice)}</span>
              </>
            )}
            {item.purchaseSeller && (
              <>
                <span>•</span>
                <span>from {item.purchaseSeller}</span>
              </>
            )}
          </div>

          {/* Notes */}
          {item.notes && (
            <p className="text-slate-500 text-sm line-clamp-1 mb-2">
              {item.notes}
            </p>
          )}

          {/* Actions */}
          <div className="flex items-center justify-between">
            {/* Quantity controls */}
            <div className="flex items-center gap-2">
              <button
                onClick={() => onAdjustQuantity(item, -1)}
                disabled={item.quantity <= 0}
                className="w-7 h-7 rounded-lg bg-slate-700 hover:bg-slate-600 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center text-slate-300"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 12H4" />
                </svg>
              </button>
              <span className="text-white font-medium min-w-[2rem] text-center">
                {item.quantity}
              </span>
              <button
                onClick={() => onAdjustQuantity(item, 1)}
                className="w-7 h-7 rounded-lg bg-slate-700 hover:bg-slate-600 flex items-center justify-center text-slate-300"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
              </button>
            </div>

            {/* Edit/Delete */}
            <div className="flex items-center gap-1">
              <button
                onClick={() => onEdit(item)}
                className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
                title="Edit"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                </svg>
              </button>
              <button
                onClick={() => onDelete(item)}
                className="p-2 text-slate-400 hover:text-red-400 hover:bg-slate-700 rounded-lg transition-colors"
                title="Delete"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

interface InventoryListProps {
  items: InventoryItem[];
  isLoading: boolean;
  hasLoaded: boolean;
  error: string | null;
  onEdit: (item: InventoryItem) => void;
  onDelete: (item: InventoryItem) => void;
  onAdjustQuantity: (item: InventoryItem, delta: number) => void;
}

export function InventoryList({ items, isLoading, hasLoaded, error, onEdit, onDelete, onAdjustQuantity }: InventoryListProps) {
  if (error) {
    return (
      <div className="flex-1 flex items-center justify-center p-8">
        <div className="text-center max-w-md">
          <div className="w-16 h-16 bg-red-500/20 rounded-full flex items-center justify-center mx-auto mb-4">
            <svg className="w-8 h-8 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <h3 className="text-lg font-medium text-white mb-2">Failed to Load Inventory</h3>
          <p className="text-slate-400 text-sm">{error}</p>
        </div>
      </div>
    );
  }

  // Only show skeleton on initial load (never loaded yet), not when filtering
  if (isLoading && !hasLoaded) {
    return (
      <div className="flex-1 p-6 space-y-4">
        {[...Array(4)].map((_, i) => (
          <div key={i} className="bg-slate-800 border border-slate-700 rounded-xl p-4 animate-pulse">
            <div className="flex gap-4">
              <div className="w-20 h-20 bg-slate-700 rounded-lg" />
              <div className="flex-1 space-y-3">
                <div className="flex gap-2">
                  <div className="w-16 h-5 bg-slate-700 rounded" />
                  <div className="w-12 h-5 bg-slate-700 rounded" />
                </div>
                <div className="h-5 bg-slate-700 rounded w-2/3" />
                <div className="h-4 bg-slate-700 rounded w-1/3" />
              </div>
            </div>
          </div>
        ))}
      </div>
    );
  }

  if (!isLoading && items.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center p-8">
        <div className="text-center max-w-md">
          <div className="w-16 h-16 bg-slate-700 rounded-full flex items-center justify-center mx-auto mb-4">
            <svg className="w-8 h-8 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
            </svg>
          </div>
          <h3 className="text-lg font-medium text-white mb-2">No Gear Yet</h3>
          <p className="text-slate-400 text-sm">
            Start building your inventory by adding equipment from the Equipment section or manually.
          </p>
        </div>
      </div>
    );
  }

  // Group items by category
  const itemsByCategory = items.reduce((acc, item) => {
    const category = item.category;
    if (!acc[category]) {
      acc[category] = [];
    }
    acc[category].push(item);
    return acc;
  }, {} as Record<string, InventoryItem[]>);

  // Sort categories by the order in EQUIPMENT_CATEGORIES
  const sortedCategories = EQUIPMENT_CATEGORIES
    .filter(cat => itemsByCategory[cat.value])
    .map(cat => ({
      value: cat.value,
      label: cat.label,
      items: itemsByCategory[cat.value],
    }));

  return (
    <div className="flex-1 overflow-y-auto p-4 md:p-6 relative">
      {/* Show subtle loading overlay when filtering existing items */}
      {isLoading && items.length > 0 && (
        <div className="absolute inset-0 bg-slate-900/50 flex items-center justify-center z-10">
          <div className="w-8 h-8 border-2 border-primary-500 border-t-transparent rounded-full animate-spin" />
        </div>
      )}
      <div className={`space-y-6 md:space-y-8 ${isLoading ? 'opacity-50' : ''}`}>
        {sortedCategories.map(category => (
          <section key={category.value}>
            <div className="flex items-center gap-3 mb-3 md:mb-4">
              <h2 className="text-base md:text-lg font-semibold text-white">{category.label}</h2>
              <span className="px-2 py-0.5 bg-slate-700 rounded-full text-xs text-slate-400">
                {category.items.length}
              </span>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3 md:gap-4">
              {category.items.map(item => (
                <InventoryCard
                  key={item.id}
                  item={item}
                  onEdit={onEdit}
                  onDelete={onDelete}
                  onAdjustQuantity={onAdjustQuantity}
                />
              ))}
            </div>
          </section>
        ))}
      </div>
    </div>
  );
}

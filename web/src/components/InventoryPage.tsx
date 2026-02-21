import { useState } from 'react';
import type { EquipmentCategory, InventoryItem, InventorySummary } from '../equipmentTypes';
import { EQUIPMENT_CATEGORIES } from '../equipmentTypes';
import { InventoryList } from './InventoryCard';
import { MobileFloatingControls } from './MobileFloatingControls';

interface InventoryPageProps {
  inventoryCategory: EquipmentCategory | null;
  inventorySummary: InventorySummary | null;
  inventoryItems: InventoryItem[];
  isInventoryLoading: boolean;
  inventoryHasLoaded: boolean;
  inventoryError: string | null;
  onInventoryCategoryFilterChange: (category: EquipmentCategory | null) => void;
  onAddItem: () => void;
  onOpenItem: (item: InventoryItem) => void;
}

export function InventoryPage({
  inventoryCategory,
  inventorySummary,
  inventoryItems,
  isInventoryLoading,
  inventoryHasLoaded,
  inventoryError,
  onInventoryCategoryFilterChange,
  onAddItem,
  onOpenItem,
}: InventoryPageProps) {
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const formattedTotalValue = new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(inventorySummary?.totalValue || 0);

  const controls = (
    <div className="px-4 md:px-6 py-4 border-b border-slate-800 bg-slate-900">
      <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div>
          <h1 className="text-xl font-semibold text-white">My Inventory</h1>
          <p className="text-sm text-slate-400">
            Track your drone equipment inventory
          </p>
        </div>
        <button
          onClick={() => {
            onAddItem();
            setIsMobileMenuOpen(false);
          }}
          className="w-full sm:w-auto px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white font-medium rounded-lg transition-colors flex items-center justify-center gap-2"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Add Item
        </button>
      </div>

      <div className="mt-4 flex flex-col gap-3 lg:flex-row lg:items-end">
        {inventoryCategory && (
          <button
            onClick={() => onInventoryCategoryFilterChange(null)}
            className="w-full sm:w-auto px-4 py-2 text-sm text-slate-300 hover:text-white hover:bg-slate-800 rounded-lg transition-colors border border-slate-700"
          >
            Clear Category
          </button>
        )}

        {inventorySummary && (
          <div className="grid grid-cols-2 gap-2 sm:flex sm:items-center sm:gap-3 lg:ml-auto">
            <div className="px-3 py-2 rounded-lg bg-slate-800 border border-slate-700">
              <div className="text-[11px] uppercase tracking-wide text-slate-500">Total Items</div>
              <div className="text-sm font-semibold text-white">{inventorySummary.totalItems}</div>
            </div>
            <div className="px-3 py-2 rounded-lg bg-slate-800 border border-slate-700">
              <div className="text-[11px] uppercase tracking-wide text-slate-500">Total Value</div>
              <div className="text-sm font-semibold text-primary-400">{formattedTotalValue}</div>
            </div>
          </div>
        )}
      </div>

      <div className="mt-4 flex gap-2 overflow-x-auto pb-2 scrollbar-thin scrollbar-thumb-slate-700">
        <button
          onClick={() => onInventoryCategoryFilterChange(null)}
          className={`px-3 py-1.5 text-sm font-medium rounded-lg whitespace-nowrap transition-colors ${
            !inventoryCategory
              ? 'bg-primary-600 text-white'
              : 'bg-slate-800 text-slate-300 hover:bg-slate-700 hover:text-white'
          }`}
        >
          All Categories
        </button>
        {EQUIPMENT_CATEGORIES.map(category => (
          <button
            key={category.value}
            onClick={() => onInventoryCategoryFilterChange(category.value)}
            className={`px-3 py-1.5 text-sm font-medium rounded-lg whitespace-nowrap transition-colors ${
              inventoryCategory === category.value
                ? 'bg-primary-600 text-white'
                : 'bg-slate-800 text-slate-300 hover:bg-slate-700 hover:text-white'
            }`}
          >
            {category.label}
          </button>
        ))}
      </div>
    </div>
  );

  return (
    <div className="relative flex-1 flex flex-col min-h-0 overflow-hidden">
      <div className="hidden md:block flex-shrink-0">{controls}</div>

      <InventoryList
        items={inventoryItems}
        isLoading={isInventoryLoading}
        hasLoaded={inventoryHasLoaded}
        error={inventoryError}
        onOpenItem={onOpenItem}
        mobileTopInset
        onListScroll={() => setIsMobileMenuOpen((prev) => (prev ? false : prev))}
      />

      <MobileFloatingControls
        label="Inventory Controls"
        isOpen={isMobileMenuOpen}
        onToggle={() => setIsMobileMenuOpen((prev) => !prev)}
      >
        {controls}
      </MobileFloatingControls>
    </div>
  );
}

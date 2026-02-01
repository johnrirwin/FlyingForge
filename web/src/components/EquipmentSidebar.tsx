import type { EquipmentCategory, EquipmentSearchParams, SellerInfo, InventorySummary } from '../equipmentTypes';
import { EQUIPMENT_CATEGORIES, ITEM_CONDITIONS } from '../equipmentTypes';
import type { AppSection } from '../equipmentTypes';
import type { User } from '../authTypes';

interface EquipmentSidebarProps {
  activeSection: AppSection;
  onSectionChange: (section: AppSection) => void;
  searchParams: EquipmentSearchParams;
  onSearchChange: (params: Partial<EquipmentSearchParams>) => void;
  sellers: SellerInfo[];
  inventorySummary: InventorySummary | null;
  inventoryCategory: EquipmentCategory | null;
  inventoryCondition: string | null;
  onInventoryFilterChange: (category: EquipmentCategory | null, condition: string | null) => void;
  // Auth props
  isAuthenticated: boolean;
  user: User | null;
  authLoading: boolean;
  onSignIn: () => void;
  onSignOut: () => void;
}

export function EquipmentSidebar({
  activeSection,
  onSectionChange,
  searchParams,
  onSearchChange,
  inventorySummary,
  inventoryCategory,
  inventoryCondition,
  onInventoryFilterChange,
  isAuthenticated,
  user,
  authLoading,
  onSignIn,
  onSignOut,
}: EquipmentSidebarProps) {
  const handleCategorySelect = (category: EquipmentCategory | undefined) => {
    if (activeSection === 'equipment') {
      onSearchChange({ category });
    } else {
      onInventoryFilterChange(category || null, inventoryCondition);
    }
  };

  const selectedCategory = activeSection === 'equipment' 
    ? searchParams.category 
    : inventoryCategory;

  // Navigation item component with lock state for unauthenticated sections
  const NavItem = ({
    section,
    icon,
    label,
    requiresAuth = false,
    badge,
  }: {
    section: AppSection;
    icon: React.ReactNode;
    label: string;
    requiresAuth?: boolean;
    badge?: number;
  }) => {
    const isLocked = requiresAuth && !isAuthenticated;
    const isActive = activeSection === section;

    return (
      <button
        onClick={() => onSectionChange(section)}
        disabled={false} // Always allow click - handleSectionChange in App.tsx will handle auth prompt
        className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-left transition-colors ${
          isActive
            ? 'bg-primary-600/20 text-primary-400'
            : isLocked
            ? 'text-slate-500 hover:text-slate-400 hover:bg-slate-800/50'
            : 'text-slate-400 hover:text-white hover:bg-slate-800'
        }`}
        title={isLocked ? 'Sign in to access' : undefined}
      >
        {icon}
        <span className="font-medium flex-1">{label}</span>
        {isLocked && (
          <svg className="w-4 h-4 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
          </svg>
        )}
        {badge !== undefined && badge > 0 && !isLocked && (
          <span className="px-2 py-0.5 bg-slate-700 rounded-full text-xs text-slate-300">
            {badge}
          </span>
        )}
      </button>
    );
  };

  return (
    <aside className="w-64 flex-shrink-0 bg-slate-900 border-r border-slate-800 flex flex-col overflow-hidden">
      {/* Section Switcher */}
      <div className="p-4 border-b border-slate-800">
        <nav className="flex flex-col gap-1">
          {/* Dashboard - only shown when authenticated */}
          {isAuthenticated && (
            <NavItem
              section="dashboard"
              label="Dashboard"
              icon={
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
                </svg>
              }
            />
          )}

          {/* News Feed - always accessible */}
          <NavItem
            section="news"
            label="News Feed"
            icon={
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z" />
              </svg>
            }
          />

          {/* Shop - always accessible */}
          <NavItem
            section="equipment"
            label="Shop"
            icon={
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 11V7a4 4 0 00-8 0v4M5 9h14l1 12H4L5 9z" />
              </svg>
            }
          />

          {/* My Gear - requires auth */}
          <NavItem
            section="inventory"
            label="My Gear"
            requiresAuth
            badge={inventorySummary?.totalItems}
            icon={
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
              </svg>
            }
          />

          {/* My Aircraft - requires auth */}
          <NavItem
            section="aircraft"
            label="My Aircraft"
            requiresAuth
            icon={
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
              </svg>
            }
          />

          {/* My Radio - requires auth */}
          <NavItem
            section="radio"
            label="My Radio"
            requiresAuth
            icon={
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
              </svg>
            }
          />
        </nav>
      </div>

      {/* Inventory filters (only for inventory section) */}
      {activeSection === 'inventory' && (
        <div className="p-4 border-b border-slate-800">
          <div className="space-y-3">
            {/* Condition filter */}
            <div>
              <label className="block text-xs font-medium text-slate-500 uppercase mb-1.5">
                Condition
              </label>
              <select
                value={inventoryCondition || ''}
                onChange={(e) => onInventoryFilterChange(inventoryCategory, e.target.value || null)}
                className="w-full px-3 py-1.5 bg-slate-800 border border-slate-700 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500"
              >
                <option value="">All Conditions</option>
                {ITEM_CONDITIONS.map(cond => (
                  <option key={cond.value} value={cond.value}>{cond.label}</option>
                ))}
              </select>
            </div>

            {/* Inventory summary */}
            {inventorySummary && (
              <div className="pt-2 border-t border-slate-800">
                <div className="grid grid-cols-2 gap-2 text-sm">
                  <div className="bg-slate-800 rounded-lg p-2">
                    <div className="text-slate-400 text-xs">Total Items</div>
                    <div className="text-white font-semibold">{inventorySummary.totalItems}</div>
                  </div>
                  <div className="bg-slate-800 rounded-lg p-2">
                    <div className="text-slate-400 text-xs">Total Value</div>
                    <div className="text-primary-400 font-semibold">
                      ${inventorySummary.totalValue.toFixed(0)}
                    </div>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Categories (only for inventory section) */}
      {activeSection === 'inventory' && (
        <div className="flex-1 overflow-y-auto">
          <div className="p-4">
            <h3 className="text-xs font-medium text-slate-500 uppercase mb-2">
              Categories
            </h3>
            <nav className="space-y-0.5">
              <button
                onClick={() => handleCategorySelect(undefined)}
                className={`w-full flex items-center gap-2 px-3 py-2 rounded-lg text-left text-sm transition-colors ${
                  !selectedCategory
                    ? 'bg-slate-800 text-white'
                    : 'text-slate-400 hover:text-white hover:bg-slate-800/50'
                }`}
              >
                <span className="w-1.5 h-1.5 rounded-full bg-slate-500" />
                All Categories
              </button>
              {EQUIPMENT_CATEGORIES.map(cat => (
                <button
                  key={cat.value}
                  onClick={() => handleCategorySelect(cat.value)}
                  className={`w-full flex items-center gap-2 px-3 py-2 rounded-lg text-left text-sm transition-colors ${
                    selectedCategory === cat.value
                      ? 'bg-slate-800 text-white'
                      : 'text-slate-400 hover:text-white hover:bg-slate-800/50'
                  }`}
                >
                  <span className="w-1.5 h-1.5 rounded-full bg-primary-500" />
                  {cat.label}
                </button>
              ))}
            </nav>
          </div>
        </div>
      )}

      {/* Spacer to push user section to bottom - only when no inventory filters showing */}
      {activeSection !== 'inventory' && <div className="flex-1" />}

      {/* User section at bottom */}
      <div className="p-4 border-t border-slate-800">
        {authLoading ? (
          <div className="flex items-center justify-center py-2">
            <div className="w-5 h-5 border-2 border-slate-600 border-t-primary-500 rounded-full animate-spin" />
          </div>
        ) : isAuthenticated && user ? (
          <div className="space-y-3">
            <div className="flex items-center gap-3">
              {user.avatarUrl ? (
                <img
                  src={user.avatarUrl}
                  alt={user.displayName || 'User'}
                  className="w-9 h-9 rounded-full flex-shrink-0"
                />
              ) : (
                <div className="w-9 h-9 rounded-full bg-primary-600 flex items-center justify-center flex-shrink-0">
                  <span className="text-white text-sm font-medium">
                    {(user.displayName || user.email || '?')[0].toUpperCase()}
                  </span>
                </div>
              )}
              <div className="flex-1 min-w-0">
                <div className="text-sm font-medium text-white truncate">
                  {user.displayName || user.email}
                </div>
                {user.displayName && user.email && (
                  <div className="text-xs text-slate-500 truncate">
                    {user.email}
                  </div>
                )}
              </div>
            </div>
            <button
              onClick={onSignOut}
              className="w-full flex items-center justify-center gap-2 px-3 py-2 text-sm text-slate-400 hover:text-white hover:bg-slate-800 rounded-lg transition-colors"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
              </svg>
              Sign Out
            </button>
          </div>
        ) : (
          <button
            onClick={onSignIn}
            className="w-full flex items-center justify-center gap-2 px-4 py-2.5 bg-primary-600 hover:bg-primary-700 text-white text-sm font-medium rounded-lg transition-colors"
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 16l-4-4m0 0l4-4m-4 4h14m-5 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h7a3 3 0 013 3v1" />
            </svg>
            Sign In
          </button>
        )}
      </div>
    </aside>
  );
}

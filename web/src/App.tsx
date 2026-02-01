import { useState, useEffect, useCallback } from 'react';
import { TopBar, FeedList, ItemDetail, InventoryList, AddInventoryModal, EquipmentSidebar, ShopSection, AircraftList, AircraftForm, AircraftDetail, AuthCallback } from './components';
import { LoginPage } from './components/LoginPage';
import { SignupPage } from './components/SignupPage';
import { getItems, getSources, refreshFeeds } from './api';
import { getSellers, getInventory, addInventoryItem, updateInventoryItem, deleteInventoryItem, getInventorySummary, addEquipmentToInventory } from './equipmentApi';
import { listAircraft, createAircraft, updateAircraft, deleteAircraft, getAircraftDetails, setAircraftComponent, setELRSSettings } from './aircraftApi';
import { useFilters, useDebounce } from './hooks';
import { useAuth } from './contexts/AuthContext';
import type { FeedItem, SourceInfo, FilterParams } from './types';
import type { EquipmentItem, SellerInfo, InventoryItem, EquipmentSearchParams, EquipmentCategory, ItemCondition, AddInventoryParams, InventorySummary, AppSection } from './equipmentTypes';
import type { Aircraft, AircraftDetailsResponse, CreateAircraftParams, UpdateAircraftParams, SetComponentParams, ELRSConfig } from './aircraftTypes';

type AuthModal = 'none' | 'login' | 'signup';

function App() {
  // Check if this is the OAuth callback
  const isAuthCallback = window.location.pathname === '/auth/callback';
  
  // Auth state - ALL HOOKS MUST BE CALLED BEFORE ANY CONDITIONAL RETURNS
  const { isAuthenticated, user, logout, isLoading: authLoading } = useAuth();
  const [authModal, setAuthModal] = useState<AuthModal>('none');

  // Section state
  const [activeSection, setActiveSection] = useState<AppSection>('news');

  // News feed state
  const [items, setItems] = useState<FeedItem[]>([]);
  const [sources, setSources] = useState<SourceInfo[]>([]);
  const [selectedItem, setSelectedItem] = useState<FeedItem | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [totalCount, setTotalCount] = useState(0);

  // Equipment state (for search params only)
  const [sellers, setSellers] = useState<SellerInfo[]>([]);
  const [equipmentSearchParams, setEquipmentSearchParams] = useState<EquipmentSearchParams>({});

  // Inventory state
  const [inventoryItems, setInventoryItems] = useState<InventoryItem[]>([]);
  const [inventorySummary, setInventorySummary] = useState<InventorySummary | null>(null);
  const [inventoryCategory, setInventoryCategory] = useState<EquipmentCategory | null>(null);
  const [inventoryCondition, setInventoryCondition] = useState<string | null>(null);
  const [isInventoryLoading, setIsInventoryLoading] = useState(false);
  const [inventoryError, setInventoryError] = useState<string | null>(null);

  // Modal state
  const [showAddInventoryModal, setShowAddInventoryModal] = useState(false);
  const [selectedEquipmentForInventory, setSelectedEquipmentForInventory] = useState<EquipmentItem | null>(null);
  const [editingInventoryItem, setEditingInventoryItem] = useState<InventoryItem | null>(null);

  // Aircraft state
  const [aircraftItems, setAircraftItems] = useState<Aircraft[]>([]);
  const [isAircraftLoading, setIsAircraftLoading] = useState(false);
  const [aircraftError, setAircraftError] = useState<string | null>(null);
  const [showAircraftForm, setShowAircraftForm] = useState(false);
  const [editingAircraft, setEditingAircraft] = useState<Aircraft | null>(null);
  const [selectedAircraftDetails, setSelectedAircraftDetails] = useState<AircraftDetailsResponse | null>(null);

  // Filters
  const { filters, updateFilter } = useFilters();
  const debouncedQuery = useDebounce(filters.query, 300);

  // Handle OAuth callback - AFTER all hooks are called
  if (isAuthCallback) {
    return <AuthCallback />;
  }

  // Load sources and sellers on mount
  useEffect(() => {
    getSources()
      .then(response => {
        setSources(response.sources);
      })
      .catch(err => {
        console.error('Failed to load sources:', err);
      });

    getSellers()
      .then(response => {
        setSellers(response.sellers);
      })
      .catch(err => {
        console.error('Failed to load sellers:', err);
      });
  }, []);

  // Load items when filters change
  useEffect(() => {
    const loadItems = async () => {
      setIsLoading(true);
      setError(null);

      try {
        const params: FilterParams = {
          limit: 50,
          sort: filters.sort,
        };

        if (filters.sources.length > 0) {
          params.sources = filters.sources;
        }

        if (filters.sourceType !== 'all') {
          params.sourceType = filters.sourceType;
        }

        if (debouncedQuery) {
          params.query = debouncedQuery;
        }

        if (filters.fromDate) {
          params.fromDate = filters.fromDate;
        }

        if (filters.toDate) {
          params.toDate = filters.toDate;
        }

        const response = await getItems(params);
        setItems(response.items || []);
        setTotalCount(response.totalCount || 0);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load items');
        setItems([]);
      } finally {
        setIsLoading(false);
      }
    };

    loadItems();
  }, [
    filters.sources,
    filters.sourceType,
    filters.sort,
    filters.fromDate,
    filters.toDate,
    debouncedQuery,
  ]);

  // Load inventory when section becomes active or filters change
  useEffect(() => {
    if (activeSection !== 'inventory') return;

    const loadInventory = async () => {
      setIsInventoryLoading(true);
      setInventoryError(null);

      try {
        const [inventoryResponse, summaryResponse] = await Promise.all([
          getInventory({
            category: inventoryCategory || undefined,
            condition: inventoryCondition as ItemCondition || undefined,
          }),
          getInventorySummary(),
        ]);
        
        setInventoryItems(inventoryResponse.items || []);
        setInventorySummary(summaryResponse);
      } catch (err) {
        setInventoryError(err instanceof Error ? err.message : 'Failed to load inventory');
        setInventoryItems([]);
      } finally {
        setIsInventoryLoading(false);
      }
    };

    loadInventory();
  }, [activeSection, inventoryCategory, inventoryCondition]);

  // Load aircraft when section becomes active
  useEffect(() => {
    if (activeSection !== 'aircraft') return;

    const loadAircraft = async () => {
      setIsAircraftLoading(true);
      setAircraftError(null);

      try {
        const response = await listAircraft();
        setAircraftItems(response.aircraft || []);
      } catch (err) {
        setAircraftError(err instanceof Error ? err.message : 'Failed to load aircraft');
        setAircraftItems([]);
      } finally {
        setIsAircraftLoading(false);
      }
    };

    loadAircraft();
  }, [activeSection]);

  // Refresh handler
  const handleRefresh = useCallback(async () => {
    setIsRefreshing(true);
    setError(null);

    try {
      // First, trigger the backend to refresh feeds from sources
      await refreshFeeds(
        filters.sources.length > 0 ? filters.sources : undefined
      );
      
      // Then re-fetch items with current filters
      const params: FilterParams = {
        limit: 50,
        sort: filters.sort,
      };

      if (filters.sources.length > 0) {
        params.sources = filters.sources;
      }

      if (filters.sourceType !== 'all') {
        params.sourceType = filters.sourceType;
      }

      if (debouncedQuery) {
        params.query = debouncedQuery;
      }

      const response = await getItems(params);
      setItems(response.items || []);
      setTotalCount(response.totalCount || 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to refresh feeds');
    } finally {
      setIsRefreshing(false);
    }
  }, [filters.sources, filters.sourceType, filters.sort, debouncedQuery]);

  // Equipment search handler
  const handleEquipmentSearchChange = useCallback((params: Partial<EquipmentSearchParams>) => {
    setEquipmentSearchParams(prev => ({ ...prev, ...params }));
  }, []);

  // Inventory filter handler
  const handleInventoryFilterChange = useCallback((category: EquipmentCategory | null, condition: string | null) => {
    setInventoryCategory(category);
    setInventoryCondition(condition);
  }, []);

  // Edit inventory item handler
  const handleEditInventoryItem = useCallback((item: InventoryItem) => {
    setEditingInventoryItem(item);
    setSelectedEquipmentForInventory(null);
    setShowAddInventoryModal(true);
  }, []);

  // Delete inventory item handler
  const handleDeleteInventoryItem = useCallback(async (item: InventoryItem) => {
    if (!confirm(`Delete "${item.name}" from your inventory?`)) return;

    try {
      await deleteInventoryItem(item.id);
      setInventoryItems(prev => prev.filter(i => i.id !== item.id));
      const summaryResponse = await getInventorySummary();
      setInventorySummary(summaryResponse);
    } catch (err) {
      console.error('Failed to delete inventory item:', err);
    }
  }, []);

  // Adjust inventory quantity handler
  const handleAdjustQuantity = useCallback(async (item: InventoryItem, delta: number) => {
    const newQuantity = Math.max(0, item.quantity + delta);
    
    try {
      const updated = await updateInventoryItem(item.id, { quantity: newQuantity });
      setInventoryItems(prev => prev.map(i => i.id === item.id ? updated : i));
      const summaryResponse = await getInventorySummary();
      setInventorySummary(summaryResponse);
    } catch (err) {
      console.error('Failed to update quantity:', err);
    }
  }, []);

  // Submit inventory modal handler
  const handleInventorySubmit = useCallback(async (params: AddInventoryParams) => {
    if (editingInventoryItem) {
      // Update existing item
      const updated = await updateInventoryItem(editingInventoryItem.id, params);
      setInventoryItems(prev => prev.map(i => i.id === editingInventoryItem.id ? updated : i));
    } else if (selectedEquipmentForInventory) {
      // Add from equipment
      const newItem = await addEquipmentToInventory(
        selectedEquipmentForInventory.id,
        selectedEquipmentForInventory.name,
        selectedEquipmentForInventory.category,
        selectedEquipmentForInventory.manufacturer,
        selectedEquipmentForInventory.price,
        selectedEquipmentForInventory.seller,
        selectedEquipmentForInventory.productUrl,
        selectedEquipmentForInventory.imageUrl,
        selectedEquipmentForInventory.keySpecs,
        params.quantity,
        params.condition,
        params.notes
      );
      setInventoryItems(prev => [...prev, newItem]);
    } else {
      // Add new manual item
      const newItem = await addInventoryItem(params);
      setInventoryItems(prev => [...prev, newItem]);
    }
    
    const summaryResponse = await getInventorySummary();
    setInventorySummary(summaryResponse);
  }, [editingInventoryItem, selectedEquipmentForInventory]);

  // Aircraft handlers
  const handleCreateAircraft = useCallback(async (params: CreateAircraftParams): Promise<Aircraft> => {
    const newAircraft = await createAircraft(params);
    setAircraftItems(prev => [...prev, newAircraft]);
    return newAircraft;
  }, []);

  const handleUpdateAircraft = useCallback(async (params: UpdateAircraftParams): Promise<Aircraft> => {
    if (!editingAircraft) throw new Error('No aircraft selected for editing');
    const updated = await updateAircraft(editingAircraft.id, params);
    setAircraftItems(prev => prev.map(a => a.id === editingAircraft.id ? updated : a));
    return updated;
  }, [editingAircraft]);

  const handleDeleteAircraft = useCallback(async (aircraft: Aircraft) => {
    if (!confirm(`Delete "${aircraft.name}"? This will not delete any gear from your inventory.`)) return;
    await deleteAircraft(aircraft.id);
    setAircraftItems(prev => prev.filter(a => a.id !== aircraft.id));
  }, []);

  const handleSelectAircraft = useCallback(async (aircraft: Aircraft) => {
    try {
      const details = await getAircraftDetails(aircraft.id);
      setSelectedAircraftDetails(details);
    } catch (err) {
      console.error('Failed to load aircraft details:', err);
    }
  }, []);

  const handleSetAircraftComponent = useCallback(async (params: SetComponentParams) => {
    if (!selectedAircraftDetails) return;
    await setAircraftComponent(selectedAircraftDetails.aircraft.id, params);
  }, [selectedAircraftDetails]);

  const handleSetELRSSettings = useCallback(async (settings: ELRSConfig) => {
    if (!selectedAircraftDetails) return;
    await setELRSSettings(selectedAircraftDetails.aircraft.id, { settings });
  }, [selectedAircraftDetails]);

  const refreshAircraftDetails = useCallback(async () => {
    if (!selectedAircraftDetails) return;
    try {
      const details = await getAircraftDetails(selectedAircraftDetails.aircraft.id);
      setSelectedAircraftDetails(details);
    } catch (err) {
      console.error('Failed to refresh aircraft details:', err);
    }
  }, [selectedAircraftDetails]);

  // Handle keyboard shortcuts
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (showAircraftForm) {
          setShowAircraftForm(false);
          setEditingAircraft(null);
        } else if (selectedAircraftDetails) {
          setSelectedAircraftDetails(null);
        } else if (showAddInventoryModal) {
          setShowAddInventoryModal(false);
          setSelectedEquipmentForInventory(null);
          setEditingInventoryItem(null);
        } else if (selectedItem) {
          setSelectedItem(null);
        }
      }
      if (e.key === '/' && !selectedItem && !showAddInventoryModal && !showAircraftForm && !selectedAircraftDetails) {
        e.preventDefault();
        const searchInput = document.querySelector('input[type="text"]') as HTMLInputElement;
        searchInput?.focus();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [selectedItem, showAddInventoryModal, showAircraftForm, selectedAircraftDetails]);

  const sourceMap = new Map(sources.map(s => [s.id, s]));

  // Handle section change with auth check for inventory and aircraft
  const handleSectionChange = useCallback((section: AppSection) => {
    if ((section === 'inventory' || section === 'aircraft') && !isAuthenticated) {
      setAuthModal('login');
      return;
    }
    setActiveSection(section);
  }, [isAuthenticated]);

  return (
    <div className="flex h-screen bg-slate-900 text-white overflow-hidden">
      {/* Sidebar with section navigation */}
      <EquipmentSidebar
        activeSection={activeSection}
        onSectionChange={handleSectionChange}
        searchParams={equipmentSearchParams}
        onSearchChange={handleEquipmentSearchChange}
        sellers={sellers}
        inventorySummary={inventorySummary}
        inventoryCategory={inventoryCategory}
        inventoryCondition={inventoryCondition}
        onInventoryFilterChange={handleInventoryFilterChange}
        isAuthenticated={isAuthenticated}
        user={user}
        authLoading={authLoading}
        onSignIn={() => setAuthModal('login')}
        onSignOut={logout}
      />

      {/* Main Content */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* News Section */}
        {activeSection === 'news' && (
          <>
            <TopBar
              query={filters.query}
              onQueryChange={q => updateFilter('query', q)}
              fromDate={filters.fromDate}
              toDate={filters.toDate}
              onFromDateChange={d => updateFilter('fromDate', d)}
              onToDateChange={d => updateFilter('toDate', d)}
              sort={filters.sort}
              onSortChange={s => updateFilter('sort', s)}
              onRefresh={handleRefresh}
              isRefreshing={isRefreshing}
              totalCount={totalCount}
            />
            <FeedList
              items={items}
              sources={sources}
              isLoading={isLoading}
              error={error}
              onItemClick={setSelectedItem}
            />
          </>
        )}

        {/* Shop Section */}
        {activeSection === 'equipment' && (
          <ShopSection />
        )}

        {/* Inventory Section */}
        {activeSection === 'inventory' && (
          <>
            <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800">
              <div>
                <h1 className="text-xl font-semibold text-white">My Gear</h1>
                <p className="text-sm text-slate-400">
                  Track your drone equipment inventory
                </p>
              </div>
              <button
                onClick={() => {
                  setSelectedEquipmentForInventory(null);
                  setEditingInventoryItem(null);
                  setShowAddInventoryModal(true);
                }}
                className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white font-medium rounded-lg transition-colors flex items-center gap-2"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                Add Item
              </button>
            </div>
            <InventoryList
              items={inventoryItems}
              isLoading={isInventoryLoading}
              error={inventoryError}
              onEdit={handleEditInventoryItem}
              onDelete={handleDeleteInventoryItem}
              onAdjustQuantity={handleAdjustQuantity}
            />
          </>
        )}

        {/* Aircraft Section */}
        {activeSection === 'aircraft' && (
          <>
            <div className="flex items-center justify-between px-6 py-4 border-b border-slate-800">
              <div>
                <h1 className="text-xl font-semibold text-white">My Aircraft</h1>
                <p className="text-sm text-slate-400">
                  Manage your drones, components, and ELRS settings
                </p>
              </div>
              <button
                onClick={() => {
                  setEditingAircraft(null);
                  setShowAircraftForm(true);
                }}
                className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white font-medium rounded-lg transition-colors flex items-center gap-2"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                Add Aircraft
              </button>
            </div>
            <AircraftList
              aircraft={aircraftItems}
              isLoading={isAircraftLoading}
              error={aircraftError}
              onSelect={handleSelectAircraft}
              onEdit={(aircraft) => {
                setEditingAircraft(aircraft);
                setShowAircraftForm(true);
              }}
              onDelete={handleDeleteAircraft}
            />
          </>
        )}
      </div>

      {/* Item Detail Modal (News) */}
      {selectedItem && (
        <ItemDetail
          item={selectedItem}
          source={sourceMap.get(selectedItem.source)}
          onClose={() => setSelectedItem(null)}
        />
      )}

      {/* Add/Edit Inventory Modal */}
      <AddInventoryModal
        isOpen={showAddInventoryModal}
        onClose={() => {
          setShowAddInventoryModal(false);
          setSelectedEquipmentForInventory(null);
          setEditingInventoryItem(null);
        }}
        onSubmit={handleInventorySubmit}
        equipmentItem={selectedEquipmentForInventory}
        editItem={editingInventoryItem}
      />

      {/* Aircraft Form Modal */}
      <AircraftForm
        isOpen={showAircraftForm}
        aircraft={editingAircraft}
        onClose={async () => {
          setShowAircraftForm(false);
          setEditingAircraft(null);
          // Refresh aircraft list to get updated hasImage status
          try {
            const response = await listAircraft();
            setAircraftItems(response.aircraft || []);
          } catch (err) {
            console.error('Failed to refresh aircraft list:', err);
          }
        }}
        onSubmit={editingAircraft ? handleUpdateAircraft : handleCreateAircraft}
      />

      {/* Aircraft Detail Modal */}
      {selectedAircraftDetails && (
        <AircraftDetail
          details={selectedAircraftDetails}
          onClose={() => setSelectedAircraftDetails(null)}
          onSetComponent={handleSetAircraftComponent}
          onSetELRSSettings={handleSetELRSSettings}
          onRefresh={refreshAircraftDetails}
        />
      )}

      {/* Auth Modals */}
      {authModal === 'login' && (
        <LoginPage
          onSwitchToSignup={() => setAuthModal('signup')}
          onClose={() => setAuthModal('none')}
        />
      )}
      {authModal === 'signup' && (
        <SignupPage
          onSwitchToLogin={() => setAuthModal('login')}
          onClose={() => setAuthModal('none')}
        />
      )}
    </div>
  );
}

export default App;

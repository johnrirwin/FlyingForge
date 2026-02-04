import { useState, useEffect, useCallback, useRef } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { TopBar, FeedList, ItemDetail, InventoryList, AddInventoryModal, Sidebar, ShopSection, AircraftList, AircraftForm, AircraftDetail, AuthCallback, Dashboard, Homepage, GettingStarted, RadioSection, BatterySection, MyProfile, SocialPage, PilotProfile, OrdersPage } from './components';
import { LoginPage } from './components/LoginPage';
import { getItems, getSources, refreshFeeds, RateLimitError } from './api';
import { getInventory, addInventoryItem, updateInventoryItem, deleteInventoryItem, getInventorySummary, addEquipmentToInventory } from './equipmentApi';
import { listAircraft, createAircraft, updateAircraft, deleteAircraft, getAircraftDetails, setAircraftComponent, setReceiverSettings } from './aircraftApi';
import { useFilters, useDebounce } from './hooks';
import { useAuth } from './hooks/useAuth';
import { useGoogleAnalytics } from './hooks/useGoogleAnalytics';
import type { FeedItem, SourceInfo, FilterParams } from './types';
import type { EquipmentItem, InventoryItem, EquipmentSearchParams, EquipmentCategory, ItemCondition, AddInventoryParams, InventorySummary, AppSection } from './equipmentTypes';
import type { Aircraft, AircraftDetailsResponse, CreateAircraftParams, UpdateAircraftParams, SetComponentParams, ReceiverConfig } from './aircraftTypes';

type AuthModal = 'none' | 'login';


// Map URL paths to AppSection values
const pathToSection: Record<string, AppSection> = {
  '/': 'home',
  '/getting-started': 'getting-started',
  '/dashboard': 'dashboard',
  '/news': 'news',
  '/shop': 'equipment',
  '/inventory': 'inventory',
  '/aircraft': 'aircraft',
  '/orders': 'orders',
  '/radio': 'radio',
  '/batteries': 'batteries',
  '/social': 'social',
  '/profile': 'profile',
};

const sectionToPath: Record<AppSection, string> = {
  'home': '/',
  'getting-started': '/getting-started',
  'dashboard': '/dashboard',
  'news': '/news',
  'equipment': '/shop',
  'inventory': '/inventory',
  'aircraft': '/aircraft',
  'orders': '/orders',
  'radio': '/radio',
  'batteries': '/batteries',
  'social': '/social',
  'profile': '/profile',
  'pilot-profile': '/social/pilots', // Dynamic - handled separately
};

// Pagination constant for news feed infinite scroll
const ITEMS_PER_PAGE = 30;

function App() {
  // Router hooks
  const location = useLocation();
  const navigate = useNavigate();
  
  // Initialize Google Analytics and track page views
  useGoogleAnalytics();
  
  // Check if this is the OAuth callback
  const isAuthCallback = location.pathname === '/auth/callback';
  
  // Auth state - ALL HOOKS MUST BE CALLED BEFORE ANY CONDITIONAL RETURNS
  const { isAuthenticated, user, logout, isLoading: authLoading } = useAuth();
  const [authModal, setAuthModal] = useState<AuthModal>('none');
  
  // Mobile menu state
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);

  // Track previous auth state to detect logout
  const [wasAuthenticated, setWasAuthenticated] = useState<boolean | null>(null);

  // Derive activeSection from URL path
  const activeSection: AppSection = pathToSection[location.pathname] || 'home';

  // News feed state
  const [items, setItems] = useState<FeedItem[]>([]);
  const [sources, setSources] = useState<SourceInfo[]>([]);
  const [selectedItem, setSelectedItem] = useState<FeedItem | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [refreshCooldown, setRefreshCooldown] = useState(0);
  const cooldownIntervalRef = useRef<number | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [totalCount, setTotalCount] = useState(0);
  const [currentOffset, setCurrentOffset] = useState(0);

  // Equipment state (for search params only)
  const [equipmentSearchParams, setEquipmentSearchParams] = useState<EquipmentSearchParams>({});

  // Inventory state
  const [inventoryItems, setInventoryItems] = useState<InventoryItem[]>([]);
  const [inventorySummary, setInventorySummary] = useState<InventorySummary | null>(null);
  const [inventoryCategory, setInventoryCategory] = useState<EquipmentCategory | null>(null);
  const [inventoryCondition, setInventoryCondition] = useState<string | null>(null);
  const [isInventoryLoading, setIsInventoryLoading] = useState(false);
  const [inventoryHasLoaded, setInventoryHasLoaded] = useState(false);
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

  // Social/Pilot state
  const [selectedPilotId, setSelectedPilotId] = useState<string | null>(null);

  // Filters
  const { filters, updateFilter } = useFilters();
  const debouncedQuery = useDebounce(filters.query, 300);

  // Handle auth state changes for routing
  useEffect(() => {
    if (authLoading) return;
    
    // Protected paths that require authentication
    const protectedPaths = ['/dashboard', '/inventory', '/aircraft', '/radio', '/batteries', '/orders', '/profile', '/social'];
    const isProtectedPath = protectedPaths.some(p => location.pathname.startsWith(p));
    
    // On initial load after auth check completes
    if (wasAuthenticated === null) {
      setWasAuthenticated(isAuthenticated);
      // Only redirect if we're on the root path and authenticated
      // This preserves the current section on refresh
      if (isAuthenticated && location.pathname === '/') {
        navigate('/dashboard', { replace: true });
      }
      // If trying to access protected path while not authenticated, show login modal
      if (!isAuthenticated && isProtectedPath) {
        setAuthModal('login');
      }
      return;
    }
    
    // Detect logout: was authenticated, now not
    if (wasAuthenticated && !isAuthenticated) {
      navigate('/', { replace: true });
    }
    
    // Detect login: was not authenticated, now is
    if (!wasAuthenticated && isAuthenticated) {
      navigate('/dashboard', { replace: true });
    }
    
    setWasAuthenticated(isAuthenticated);
  }, [isAuthenticated, authLoading, wasAuthenticated, navigate, location.pathname]);

  // Load sources on mount
  useEffect(() => {
    getSources()
      .then(response => {
        setSources(response.sources);
      })
      .catch(err => {
        console.error('Failed to load sources:', err);
      });
  }, []);

  // Load items when filters change (reset to first page)
  useEffect(() => {
    const loadItems = async () => {
      setIsLoading(true);
      setError(null);
      setCurrentOffset(0);

      try {
        const params: FilterParams = {
          limit: ITEMS_PER_PAGE,
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
        setCurrentOffset(ITEMS_PER_PAGE);
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

  // Load more items (infinite scroll)
  const loadMoreItems = useCallback(async () => {
    if (isLoadingMore || items.length >= totalCount) return;
    
    setIsLoadingMore(true);

    try {
      const params: FilterParams = {
        limit: ITEMS_PER_PAGE,
        offset: currentOffset,
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
      if (response.items?.length) {
        setItems(prev => [...prev, ...response.items]);
        setCurrentOffset(prev => prev + response.items.length);
      }
    } catch (err) {
      console.error('Failed to load more items:', err);
    } finally {
      setIsLoadingMore(false);
    }
  }, [currentOffset, isLoadingMore, items.length, totalCount, filters, debouncedQuery]);

  // Load inventory when section becomes active or filters change (also for dashboard)
  useEffect(() => {
    if (activeSection !== 'inventory' && activeSection !== 'dashboard') return;
    if (!isAuthenticated) return;

    const loadInventory = async () => {
      setIsInventoryLoading(true);
      setInventoryError(null);

      try {
        const [inventoryResponse, summaryResponse] = await Promise.all([
          getInventory({
            category: activeSection === 'inventory' ? (inventoryCategory || undefined) : undefined,
            condition: activeSection === 'inventory' ? (inventoryCondition as ItemCondition || undefined) : undefined,
          }),
          getInventorySummary(),
        ]);
        
        setInventoryItems(inventoryResponse.items || []);
        setInventorySummary(summaryResponse);
        setInventoryHasLoaded(true);
      } catch (err) {
        setInventoryError(err instanceof Error ? err.message : 'Failed to load inventory');
        setInventoryItems([]);
      } finally {
        setIsInventoryLoading(false);
      }
    };

    loadInventory();
  }, [activeSection, inventoryCategory, inventoryCondition, isAuthenticated]);

  // Load aircraft when section becomes active (also for dashboard)
  useEffect(() => {
    if (activeSection !== 'aircraft' && activeSection !== 'dashboard') return;
    if (!isAuthenticated) return;

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
  }, [activeSection, isAuthenticated]);

  // Start cooldown timer
  const startCooldown = useCallback((seconds: number) => {
    setRefreshCooldown(seconds);
    
    // Clear any existing interval
    if (cooldownIntervalRef.current) {
      clearInterval(cooldownIntervalRef.current);
    }
    
    // Start countdown
    cooldownIntervalRef.current = window.setInterval(() => {
      setRefreshCooldown(prev => {
        if (prev <= 1) {
          if (cooldownIntervalRef.current) {
            clearInterval(cooldownIntervalRef.current);
            cooldownIntervalRef.current = null;
          }
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  }, []);

  // Cleanup cooldown interval on unmount
  useEffect(() => {
    return () => {
      if (cooldownIntervalRef.current) {
        clearInterval(cooldownIntervalRef.current);
      }
    };
  }, []);

  // Refresh handler
  const handleRefresh = useCallback(async () => {
    setIsRefreshing(true);
    setError(null);

    try {
      // First, trigger the backend to refresh feeds from sources
      await refreshFeeds(
        filters.sources.length > 0 ? filters.sources : undefined
      );
      
      // Reset infinite scroll state
      setCurrentOffset(0);
      
      // Then re-fetch items with current filters
      const params: FilterParams = {
        limit: ITEMS_PER_PAGE,
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
      setCurrentOffset(ITEMS_PER_PAGE);
    } catch (err) {
      if (err instanceof RateLimitError) {
        // Start 2 minute cooldown
        startCooldown(120);
      } else {
        setError(err instanceof Error ? err.message : 'Failed to refresh feeds');
      }
    } finally {
      setIsRefreshing(false);
    }
  }, [filters.sources, filters.sourceType, filters.sort, filters.fromDate, filters.toDate, debouncedQuery, startCooldown]);

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

  const handleSetReceiverSettings = useCallback(async (settings: ReceiverConfig) => {
    if (!selectedAircraftDetails) return;
    await setReceiverSettings(selectedAircraftDetails.aircraft.id, { settings });
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

  // Handle section change with auth check for protected sections
  const handleSectionChange = useCallback((section: AppSection) => {
    // When authenticated user clicks home, redirect to dashboard
    if (section === 'home' && isAuthenticated) {
      navigate('/dashboard');
      return;
    }
    // Protected sections that require authentication
    const protectedSections = ['dashboard', 'inventory', 'aircraft', 'radio', 'batteries', 'orders', 'profile', 'social'];
    if (protectedSections.includes(section) && !isAuthenticated) {
      setAuthModal('login');
      return;
    }
    navigate(sectionToPath[section]);
  }, [isAuthenticated, navigate]);

  // Handle logout with redirect to news feed
  const handleLogout = useCallback(async () => {
    await logout();
    // Navigation to news is handled by the auth state change effect
  }, [logout]);

  // Memoized callbacks for Sidebar to prevent re-renders
  const handleOpenLogin = useCallback(() => setAuthModal('login'), []);
  const handleCloseMobileMenu = useCallback(() => setIsMobileMenuOpen(false), []);

  // Handle OAuth callback - must be after all hooks are called
  if (isAuthCallback) {
    return <AuthCallback />;
  }

  return (
    <div className="flex h-screen bg-slate-900 text-white overflow-hidden">
      {/* Mobile Header */}
      <div className="md:hidden fixed top-0 left-0 right-0 z-30 bg-slate-900 border-b border-slate-800 px-4 py-3 flex items-center justify-between">
        <button
          onClick={() => setIsMobileMenuOpen(true)}
          className="p-2 text-slate-400 hover:text-white rounded-lg hover:bg-slate-800"
        >
          <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
          </svg>
        </button>
        <span className="text-lg font-semibold text-primary-400">FlyingForge</span>
        <div className="w-10" /> {/* Spacer for balance */}
      </div>

      {/* Sidebar with section navigation */}
      <Sidebar
        activeSection={activeSection}
        onSectionChange={handleSectionChange}
        searchParams={equipmentSearchParams}
        onSearchChange={handleEquipmentSearchChange}
        inventorySummary={inventorySummary}
        inventoryCategory={inventoryCategory}
        inventoryCondition={inventoryCondition}
        onInventoryFilterChange={handleInventoryFilterChange}
        isAuthenticated={isAuthenticated}
        user={user}
        authLoading={authLoading}
        onSignIn={handleOpenLogin}
        onSignOut={handleLogout}
        isMobileMenuOpen={isMobileMenuOpen}
        onMobileMenuClose={handleCloseMobileMenu}
      />

      {/* Main Content */}
      <div className="flex-1 flex flex-col min-w-0 pt-14 md:pt-0">
        {/* Homepage Section - for unauthenticated users */}
        {activeSection === 'home' && !isAuthenticated && (
          <Homepage
            onSignIn={() => setAuthModal('login')}
            onExploreNews={() => navigate('/news')}
          />
        )}

        {/* Getting Started Section - public education page */}
        {activeSection === 'getting-started' && (
          <GettingStarted
            onSignIn={() => setAuthModal('login')}
          />
        )}

        {/* When on 'home' but authenticated, show the dashboard to avoid a blank state */}
        {activeSection === 'home' && isAuthenticated && (
          <Dashboard
            recentAircraft={aircraftItems}
            recentNews={items}
            sources={sources}
            isAircraftLoading={isAircraftLoading}
            isNewsLoading={isLoading}
            onViewAllNews={() => navigate('/news')}
            onViewAllAircraft={() => navigate('/aircraft')}
            onViewAllOrders={() => navigate('/orders')}
            onSelectAircraft={handleSelectAircraft}
            onSelectNewsItem={setSelectedItem}
            onSelectPilot={(pilotId) => {
              setSelectedPilotId(pilotId);
              navigate('/social');
            }}
            onGoToSocial={() => navigate('/social')}
          />
        )}

        {/* Dashboard Section - only for authenticated users */}
        {activeSection === 'dashboard' && isAuthenticated && (
          <Dashboard
            recentAircraft={aircraftItems}
            recentNews={items}
            sources={sources}
            isAircraftLoading={isAircraftLoading}
            isNewsLoading={isLoading}
            onViewAllNews={() => navigate('/news')}
            onViewAllAircraft={() => navigate('/aircraft')}
            onViewAllOrders={() => navigate('/orders')}
            onSelectAircraft={handleSelectAircraft}
            onSelectNewsItem={setSelectedItem}
            onSelectPilot={(pilotId) => {
              setSelectedPilotId(pilotId);
              navigate('/social');
            }}
            onGoToSocial={() => navigate('/social')}
          />
        )}

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
              sourceType={filters.sourceType}
              onSourceTypeChange={t => updateFilter('sourceType', t)}
              onRefresh={handleRefresh}
              isRefreshing={isRefreshing}
              refreshCooldown={refreshCooldown}
              totalCount={totalCount}
            />
            <FeedList
              items={items}
              sources={sources}
              isLoading={isLoading || isLoadingMore}
              error={error}
              onItemClick={setSelectedItem}
              hasMore={items.length < totalCount}
              onLoadMore={loadMoreItems}
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
              hasLoaded={inventoryHasLoaded}
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
                  Manage your drones, components, and receiver settings
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

        {/* Orders Section */}
        {activeSection === 'orders' && (
          <OrdersPage />
        )}

        {/* Radio Section */}
        {activeSection === 'radio' && (
          <RadioSection
            onError={(message) => setError(message)}
          />
        )}

        {/* Battery Section */}
        {activeSection === 'batteries' && (
          <BatterySection
            onError={(message) => setError(message)}
          />
        )}

        {/* Profile Section */}
        {activeSection === 'profile' && (
          <MyProfile />
        )}

        {/* Social/Pilot Directory Section */}
        {activeSection === 'social' && !selectedPilotId && (
          <SocialPage
            onSelectPilot={(pilotId) => setSelectedPilotId(pilotId)}
          />
        )}

        {/* Pilot Profile View */}
        {activeSection === 'social' && selectedPilotId && (
          <PilotProfile
            pilotId={selectedPilotId}
            onBack={() => setSelectedPilotId(null)}
            onSelectPilot={(pilotId) => setSelectedPilotId(pilotId)}
          />
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
          onSetReceiverSettings={handleSetReceiverSettings}
          onRefresh={refreshAircraftDetails}
        />
      )}

      {/* Auth Modal */}
      {authModal === 'login' && (
        <LoginPage
          onClose={() => setAuthModal('none')}
        />
      )}
    </div>
  );
}

export default App;

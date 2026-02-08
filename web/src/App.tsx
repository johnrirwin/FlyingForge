import { useState, useEffect, useCallback, useRef } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { TopBar, FeedList, ItemDetail, InventoryList, AddGearModal, Sidebar, ShopSection, AircraftList, AircraftForm, AircraftDetail, AuthCallback, Dashboard, Homepage, GettingStarted, RadioSection, BatterySection, MyProfile, SocialPage, PilotProfile, GearCatalogPage, AdminGearModeration, AdminUserManagement } from './components';
import { LoginPage } from './components/LoginPage';
import { getItems, getSources, refreshFeeds, RateLimitError } from './api';
import { getInventory, addInventoryItem, updateInventoryItem, deleteInventoryItem, getInventorySummary, addEquipmentToInventory } from './equipmentApi';
import { listAircraft, createAircraft, updateAircraft, deleteAircraft, getAircraftDetails, setAircraftComponent, setReceiverSettings } from './aircraftApi';
import { useFilters } from './hooks';
import { useAuth } from './hooks/useAuth';
import { useGoogleAnalytics, trackEvent } from './hooks/useGoogleAnalytics';
import type { FeedItem, SourceInfo, FilterParams } from './types';
import { EQUIPMENT_CATEGORIES, type EquipmentItem, type InventoryItem, type EquipmentCategory, type AddInventoryParams, type InventorySummary, type AppSection } from './equipmentTypes';
import type { Aircraft, AircraftDetailsResponse, CreateAircraftParams, UpdateAircraftParams, SetComponentParams, ReceiverConfig } from './aircraftTypes';
import type { GearCatalogItem } from './gearCatalogTypes';

type AuthModal = 'none' | 'login';

const FOCUSABLE_SELECTOR = 'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])';


// Map URL paths to AppSection values
const pathToSection: Record<string, AppSection> = {
  '/': 'home',
  '/getting-started': 'getting-started',
  '/dashboard': 'dashboard',
  '/news': 'news',
  '/shop': 'equipment',
  '/gear-catalog': 'gear-catalog',
  '/inventory': 'inventory',
  '/aircraft': 'aircraft',
  '/radio': 'radio',
  '/batteries': 'batteries',
  '/social': 'social',
  '/profile': 'profile',
  '/admin/gear': 'admin-gear',
  '/admin/users': 'admin-users',
};

const sectionToPath: Record<AppSection, string> = {
  'home': '/',
  'getting-started': '/getting-started',
  'dashboard': '/dashboard',
  'news': '/news',
  'equipment': '/shop',
  'gear-catalog': '/gear-catalog',
  'inventory': '/inventory',
  'aircraft': '/aircraft',
  'radio': '/radio',
  'batteries': '/batteries',
  'social': '/social',
  'profile': '/profile',
  'pilot-profile': '/social/pilots', // Dynamic - handled separately
  'admin-gear': '/admin/gear',
  'admin-users': '/admin/users',
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

  // Inventory state
  const [inventoryItems, setInventoryItems] = useState<InventoryItem[]>([]);
  const [inventorySummary, setInventorySummary] = useState<InventorySummary | null>(null);
  const [inventoryCategory, setInventoryCategory] = useState<EquipmentCategory | null>(null);
  const [isInventoryLoading, setIsInventoryLoading] = useState(false);
  const [inventoryHasLoaded, setInventoryHasLoaded] = useState(false);
  const [inventoryError, setInventoryError] = useState<string | null>(null);

  // Modal state
  const [showAddInventoryModal, setShowAddInventoryModal] = useState(false);
  const [selectedEquipmentForInventory, setSelectedEquipmentForInventory] = useState<EquipmentItem | null>(null);
  const [selectedCatalogItemForInventory, setSelectedCatalogItemForInventory] = useState<GearCatalogItem | null>(null);
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
  const appShellRef = useRef<HTMLDivElement | null>(null);
  const pilotProfileDialogRef = useRef<HTMLDivElement | null>(null);
  const pilotProfileCloseButtonRef = useRef<HTMLButtonElement | null>(null);
  const lastFocusedElementBeforePilotModalRef = useRef<HTMLElement | null>(null);
  const isPilotProfileModalOpen = activeSection === 'social' && Boolean(selectedPilotId);

  // Filters
  const { filters, updateFilter } = useFilters();
  const [appliedQuery, setAppliedQuery] = useState('');

  // Handle explicit search trigger for news feed
  const handleNewsSearch = useCallback(() => {
    setAppliedQuery(filters.query);
  }, [filters.query]);

  // Handle auth state changes for routing
  useEffect(() => {
    if (authLoading) return;
    
    // Protected paths that require authentication
    const protectedPaths = ['/dashboard', '/inventory', '/aircraft', '/radio', '/batteries', '/profile', '/social', '/admin'];
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

        if (appliedQuery) {
          params.query = appliedQuery;
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
    appliedQuery,
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

      if (appliedQuery) {
        params.query = appliedQuery;
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
  }, [currentOffset, isLoadingMore, items.length, totalCount, filters, appliedQuery]);

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
  }, [activeSection, inventoryCategory, isAuthenticated]);

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

      if (appliedQuery) {
        params.query = appliedQuery;
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
  }, [filters.sources, filters.sourceType, filters.sort, filters.fromDate, filters.toDate, appliedQuery, startCooldown]);

  // Inventory category filter handler
  const handleInventoryCategoryFilterChange = useCallback((category: EquipmentCategory | null) => {
    setInventoryCategory(category);
  }, []);

  // Edit inventory item handler
  const handleEditInventoryItem = useCallback((item: InventoryItem) => {
    setEditingInventoryItem(item);
    setSelectedEquipmentForInventory(null);
    setShowAddInventoryModal(true);
  }, []);

  // Delete inventory item handler
  const handleDeleteInventoryItem = useCallback(async (item: InventoryItem) => {
    try {
      await deleteInventoryItem(item.id);
      setInventoryItems(prev => prev.filter(i => i.id !== item.id));
      const summaryResponse = await getInventorySummary();
      setInventorySummary(summaryResponse);
    } catch (err) {
      console.error('Failed to delete inventory item:', err);
      throw err instanceof Error ? err : new Error('Failed to delete inventory item');
    }
  }, []);

  // Submit inventory modal handler
  const handleInventorySubmit = useCallback(async (params: AddInventoryParams) => {
    console.log('[App] handleInventorySubmit called with params:', params);
    console.log('[App] editingInventoryItem:', editingInventoryItem);
    console.log('[App] selectedEquipmentForInventory:', selectedEquipmentForInventory);
    
    if (editingInventoryItem) {
      // Update existing item
      console.log('[App] Updating existing inventory item');
      const updated = await updateInventoryItem(editingInventoryItem.id, params);
      setInventoryItems(prev => prev.map(i => i.id === editingInventoryItem.id ? updated : i));
    } else if (selectedEquipmentForInventory) {
      // Add from equipment
      console.log('[App] Adding from equipment shop');
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
        params.notes
      );
      setInventoryItems(prev => [...prev, newItem]);
      // Track gear addition for GA4 conversions
      trackEvent('gear_added', { category: selectedEquipmentForInventory.category, method: 'from_shop' });
    } else {
      // Add new manual item
      console.log('[App] Adding new manual/catalog item');
      const newItem = await addInventoryItem(params);
      console.log('[App] New item created:', newItem);
      setInventoryItems(prev => [...prev, newItem]);
      // Track gear addition for GA4 conversions
      trackEvent('gear_added', { category: params.category, method: 'manual' });
    }
    
    console.log('[App] Refreshing inventory summary');
    const summaryResponse = await getInventorySummary();
    setInventorySummary(summaryResponse);
  }, [editingInventoryItem, selectedEquipmentForInventory]);

  // Aircraft handlers
  const handleCreateAircraft = useCallback(async (params: CreateAircraftParams): Promise<Aircraft> => {
    const newAircraft = await createAircraft(params);
    setAircraftItems(prev => [...prev, newAircraft]);
    // Track aircraft creation for GA4 conversions
    trackEvent('aircraft_created', { aircraft_type: params.type });
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
        if (selectedPilotId) {
          setSelectedPilotId(null);
        } else if (showAircraftForm) {
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
      if (e.key === '/' && !selectedPilotId && !selectedItem && !showAddInventoryModal && !showAircraftForm && !selectedAircraftDetails) {
        e.preventDefault();
        const searchInput = document.querySelector('input[type="text"]') as HTMLInputElement;
        searchInput?.focus();
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [selectedItem, selectedPilotId, showAddInventoryModal, showAircraftForm, selectedAircraftDetails]);

  useEffect(() => {
    const appShell = appShellRef.current;
    if (!appShell) return;

    if (isPilotProfileModalOpen) {
      appShell.setAttribute('aria-hidden', 'true');
      appShell.setAttribute('inert', '');
    } else {
      appShell.removeAttribute('aria-hidden');
      appShell.removeAttribute('inert');
    }
  }, [isPilotProfileModalOpen]);

  useEffect(() => {
    if (!isPilotProfileModalOpen) return;

    const dialog = pilotProfileDialogRef.current;
    if (!dialog) return;

    lastFocusedElementBeforePilotModalRef.current = document.activeElement instanceof HTMLElement
      ? document.activeElement
      : null;

    const focusableElements = dialog.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR);
    const firstElement = focusableElements[0] ?? pilotProfileCloseButtonRef.current ?? dialog;
    firstElement.focus();

    const handleTabKey = (event: KeyboardEvent) => {
      if (event.key !== 'Tab') return;

      const elements = dialog.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR);
      if (elements.length === 0) {
        event.preventDefault();
        return;
      }

      const first = elements[0];
      const last = elements[elements.length - 1];
      const activeElement = document.activeElement;

      if (event.shiftKey) {
        if (activeElement === first || !dialog.contains(activeElement)) {
          event.preventDefault();
          last.focus();
        }
        return;
      }

      if (activeElement === last || !dialog.contains(activeElement)) {
        event.preventDefault();
        first.focus();
      }
    };

    document.addEventListener('keydown', handleTabKey);
    return () => {
      document.removeEventListener('keydown', handleTabKey);
      lastFocusedElementBeforePilotModalRef.current?.focus();
    };
  }, [isPilotProfileModalOpen]);

  const sourceMap = new Map(sources.map(s => [s.id, s]));

  // Handle section change with auth check for protected sections
  const handleSectionChange = useCallback((section: AppSection) => {
    // When authenticated user clicks home, redirect to dashboard
    if (section === 'home' && isAuthenticated) {
      navigate('/dashboard');
      return;
    }
    // Protected sections that require authentication
    const protectedSections = ['dashboard', 'inventory', 'aircraft', 'radio', 'batteries', 'profile', 'social', 'admin-gear', 'admin-users'];
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
    <div className="flex h-screen supports-[height:100dvh]:h-[100dvh] bg-slate-900 text-white overflow-hidden">
      <div ref={appShellRef} className="flex flex-1 min-h-0 min-w-0">
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
        isAuthenticated={isAuthenticated}
        user={user}
        authLoading={authLoading}
        onSignIn={handleOpenLogin}
        onSignOut={handleLogout}
        isMobileMenuOpen={isMobileMenuOpen}
        onMobileMenuClose={handleCloseMobileMenu}
      />

      {/* Main Content */}
      <div className="flex-1 flex flex-col min-h-0 min-w-0 pt-14 md:pt-0">
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
            onViewAllGear={() => navigate('/inventory')}
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
            onViewAllGear={() => navigate('/inventory')}
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
            {/* TopBar - fixed on mobile, normal flow on desktop */}
            <div className="fixed md:relative top-14 md:top-0 left-0 right-0 md:left-auto md:right-auto z-20 md:z-10 bg-slate-900">
              <TopBar
                query={filters.query}
                onQueryChange={q => updateFilter('query', q)}
                onSearch={handleNewsSearch}
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
                isCollapsed={false}
              />
            </div>
            {/* Scrollable feed list - extra top padding on mobile to account for fixed TopBar (~180px) */}
            <div 
              className="flex-1 overflow-y-auto pt-[180px] md:pt-0"
              onScroll={() => {
                // Dismiss keyboard on scroll for mobile
                if (document.activeElement instanceof HTMLElement) {
                  document.activeElement.blur();
                }
              }}
            >
              <FeedList
                items={items}
                sources={sources}
                isLoading={isLoading || isLoadingMore}
                error={error}
                onItemClick={setSelectedItem}
                hasMore={items.length < totalCount}
                onLoadMore={loadMoreItems}
              />
            </div>
          </>
        )}

        {/* Admin: Gear Moderation Section */}
        {activeSection === 'admin-gear' && (
          <AdminGearModeration hasGearAdminAccess={Boolean(user?.isAdmin || user?.isGearAdmin)} authLoading={authLoading} />
        )}

        {/* Admin: User Admin Section */}
        {activeSection === 'admin-users' && (
          <AdminUserManagement isAdmin={Boolean(user?.isAdmin)} currentUserId={user?.id} authLoading={authLoading} />
        )}

        {/* Shop Section */}
        {activeSection === 'equipment' && (
          <ShopSection />
        )}

        {/* Gear Catalog Section - Public browsable catalog like PCPartPicker */}
        {activeSection === 'gear-catalog' && (
          <GearCatalogPage 
            onAddToInventory={(catalogItem) => {
              // When user clicks add on a catalog item, open the add gear modal with it selected
              setSelectedEquipmentForInventory(null);
              setSelectedCatalogItemForInventory(catalogItem);
              setEditingInventoryItem(null);
              setShowAddInventoryModal(true);
            }}
          />
        )}

        {/* Inventory Section */}
        {activeSection === 'inventory' && (
          <div className="flex-1 flex flex-col min-h-0">
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
                    setSelectedEquipmentForInventory(null);
                    setEditingInventoryItem(null);
                    setShowAddInventoryModal(true);
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
                    onClick={() => handleInventoryCategoryFilterChange(null)}
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
                      <div className="text-sm font-semibold text-primary-400">${inventorySummary.totalValue.toFixed(0)}</div>
                    </div>
                  </div>
                )}
              </div>

              <div className="mt-4 flex gap-2 overflow-x-auto pb-2 scrollbar-thin scrollbar-thumb-slate-700">
                <button
                  onClick={() => handleInventoryCategoryFilterChange(null)}
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
                    onClick={() => handleInventoryCategoryFilterChange(category.value)}
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
            <InventoryList
              items={inventoryItems}
              isLoading={isInventoryLoading}
              hasLoaded={inventoryHasLoaded}
              error={inventoryError}
              onOpenItem={handleEditInventoryItem}
            />
          </div>
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
        {activeSection === 'social' && (
          <SocialPage
            onSelectPilot={(pilotId) => setSelectedPilotId(pilotId)}
          />
        )}
      </div>
      </div>

      {/* Pilot Profile Modal (Social) */}
      {isPilotProfileModalOpen && selectedPilotId && (
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby="pilot-profile-modal-title"
          className="fixed inset-0 z-[70] flex items-start md:items-center justify-center p-4 md:p-6"
        >
          <div
            className="absolute inset-0 bg-black/60 backdrop-blur-sm"
            onClick={() => setSelectedPilotId(null)}
          />
          <div
            ref={pilotProfileDialogRef}
            tabIndex={-1}
            className="relative w-full max-w-4xl h-[92vh] max-h-[92vh] overflow-hidden bg-slate-900 border border-slate-700 rounded-2xl flex flex-col"
          >
            <div className="flex items-center justify-between px-6 py-4 border-b border-slate-700">
              <h2 id="pilot-profile-modal-title" className="text-lg font-semibold text-white">Pilot Profile</h2>
              <button
                ref={pilotProfileCloseButtonRef}
                onClick={() => setSelectedPilotId(null)}
                aria-label="Close pilot profile modal"
                className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            <div className="flex-1 min-h-0">
              <PilotProfile
                pilotId={selectedPilotId}
                onBack={() => setSelectedPilotId(null)}
                onSelectPilot={(pilotId) => setSelectedPilotId(pilotId)}
                isModal
              />
            </div>
          </div>
        </div>
      )}

      {/* Item Detail Modal (News) */}
      {selectedItem && (
        <ItemDetail
          item={selectedItem}
          source={sourceMap.get(selectedItem.source)}
          onClose={() => setSelectedItem(null)}
        />
      )}

      {/* Add/Edit Inventory Modal */}
      <AddGearModal
        isOpen={showAddInventoryModal}
        onClose={() => {
          setShowAddInventoryModal(false);
          setSelectedEquipmentForInventory(null);
          setSelectedCatalogItemForInventory(null);
          setEditingInventoryItem(null);
        }}
        onSubmit={handleInventorySubmit}
        onDelete={handleDeleteInventoryItem}
        equipmentItem={selectedEquipmentForInventory}
        catalogItem={selectedCatalogItemForInventory}
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

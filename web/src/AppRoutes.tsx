import type { ComponentProps, ReactNode } from 'react';
import { Navigate, Outlet, Route, Routes, useLocation, useNavigate } from 'react-router-dom';
import {
  Homepage,
  GettingStarted,
  NewsPage,
  ShopSection,
  GearCatalogPage,
  PublicBuildsPage,
  PublicBuildDetailsPage,
  TempBuildPage,
  MyBuildsPage,
  InventoryPage,
  AircraftPage,
  RadioSection,
  BatterySection,
  MyProfile,
  SocialPage,
  AdminGearModeration,
  AdminUserManagement,
  TopBar,
} from './components';
import type { User } from './authTypes';
import type { FeedItem, SourceInfo } from './types';
import type {
  EquipmentCategory,
  InventoryItem,
  InventorySummary,
} from './equipmentTypes';
import type { Aircraft } from './aircraftTypes';
import type { GearCatalogItem } from './gearCatalogTypes';
import { buildLoginPath, getCurrentPathWithSearchAndHash } from './authRouting';

interface AppRoutesProps {
  isAuthenticated: boolean;
  user: User | null;
  authLoading: boolean;
  dashboardElement: ReactNode;
  onOpenLogin: () => void;

  newsTopBarProps: ComponentProps<typeof TopBar>;
  newsItems: FeedItem[];
  newsSources: SourceInfo[];
  isNewsLoading: boolean;
  isNewsLoadingMore: boolean;
  newsError: string | null;
  newsTotalCount: number;
  onSelectNewsItem: (item: FeedItem) => void;
  onLoadMoreNews: () => void;

  onAddToInventoryFromCatalog: (catalogItem: GearCatalogItem) => void;

  inventoryCategory: EquipmentCategory | null;
  inventorySummary: InventorySummary | null;
  inventoryItems: InventoryItem[];
  isInventoryLoading: boolean;
  inventoryHasLoaded: boolean;
  inventoryError: string | null;
  onInventoryCategoryFilterChange: (category: EquipmentCategory | null) => void;
  onAddInventoryItem: () => void;
  onOpenInventoryItem: (item: InventoryItem) => void;

  aircraftItems: Aircraft[];
  isAircraftLoading: boolean;
  aircraftError: string | null;
  onSelectAircraft: (aircraft: Aircraft) => void;
  onEditAircraft: (aircraft: Aircraft) => void;
  onDeleteAircraft: (aircraft: Aircraft) => void;
  onAddAircraft: () => void;

  onRadioError: (message: string) => void;
  onBatteryError: (message: string) => void;
  onSelectPilot: (pilotId: string) => void;
}

interface RequireAuthRouteProps {
  isAuthenticated: boolean;
  authLoading: boolean;
  loadingFallback: ReactNode;
}

function RequireAuthRoute({
  isAuthenticated,
  authLoading,
  loadingFallback,
}: RequireAuthRouteProps) {
  const location = useLocation();

  if (authLoading) {
    return <>{loadingFallback}</>;
  }

  if (!isAuthenticated) {
    return (
      <Navigate
        to={buildLoginPath(getCurrentPathWithSearchAndHash({
          pathname: location.pathname,
          search: location.search,
          hash: location.hash,
        }))}
        replace
      />
    );
  }

  return <Outlet />;
}

export function AppRoutes({
  isAuthenticated,
  user,
  authLoading,
  dashboardElement,
  onOpenLogin,
  newsTopBarProps,
  newsItems,
  newsSources,
  isNewsLoading,
  isNewsLoadingMore,
  newsError,
  newsTotalCount,
  onSelectNewsItem,
  onLoadMoreNews,
  onAddToInventoryFromCatalog,
  inventoryCategory,
  inventorySummary,
  inventoryItems,
  isInventoryLoading,
  inventoryHasLoaded,
  inventoryError,
  onInventoryCategoryFilterChange,
  onAddInventoryItem,
  onOpenInventoryItem,
  aircraftItems,
  isAircraftLoading,
  aircraftError,
  onSelectAircraft,
  onEditAircraft,
  onDeleteAircraft,
  onAddAircraft,
  onRadioError,
  onBatteryError,
  onSelectPilot,
}: AppRoutesProps) {
  const navigate = useNavigate();
  const protectedFallback = (
    <div className="flex-1 overflow-y-auto p-6">
      <div className="mx-auto w-full max-w-4xl rounded-xl border border-slate-700 bg-slate-800/60 p-8 text-center text-slate-400">
        Loading...
      </div>
    </div>
  );

  return (
    <Routes>
      <Route
        path="/"
        element={
          isAuthenticated ? (
            dashboardElement
          ) : (
            <Homepage
              onSignIn={onOpenLogin}
              onExploreNews={() => navigate('/news')}
            />
          )
        }
      />
      <Route
        path="/getting-started"
        element={
          <GettingStarted
            onSignIn={onOpenLogin}
          />
        }
      />
      <Route
        element={(
          <RequireAuthRoute
            isAuthenticated={isAuthenticated}
            authLoading={authLoading}
            loadingFallback={protectedFallback}
          />
        )}
      >
        <Route
          path="/dashboard"
          element={dashboardElement}
        />
        <Route
          path="/inventory"
          element={
            <InventoryPage
              inventoryCategory={inventoryCategory}
              inventorySummary={inventorySummary}
              inventoryItems={inventoryItems}
              isInventoryLoading={isInventoryLoading}
              inventoryHasLoaded={inventoryHasLoaded}
              inventoryError={inventoryError}
              onInventoryCategoryFilterChange={onInventoryCategoryFilterChange}
              onAddItem={onAddInventoryItem}
              onOpenItem={onOpenInventoryItem}
            />
          }
        />
        <Route path="/me/builds" element={<MyBuildsPage />} />
        <Route
          path="/aircraft"
          element={
            <AircraftPage
              aircraftItems={aircraftItems}
              isAircraftLoading={isAircraftLoading}
              aircraftError={aircraftError}
              onSelectAircraft={onSelectAircraft}
              onEditAircraft={onEditAircraft}
              onDeleteAircraft={onDeleteAircraft}
              onAddAircraft={onAddAircraft}
            />
          }
        />
        <Route
          path="/radio"
          element={(
            <RadioSection
              onError={onRadioError}
            />
          )}
        />
        <Route
          path="/batteries"
          element={(
            <div className="flex-1 min-h-0 flex flex-col overflow-hidden">
              <BatterySection
                onError={onBatteryError}
              />
            </div>
          )}
        />
        <Route path="/profile" element={<MyProfile />} />
        <Route
          path="/social"
          element={
            <SocialPage
              onSelectPilot={onSelectPilot}
            />
          }
        />
        <Route
          path="/admin/content"
          element={
            <AdminGearModeration
              hasContentAdminAccess={Boolean(user?.isAdmin || user?.isContentAdmin || user?.isGearAdmin)}
              authLoading={authLoading}
            />
          }
        />
        <Route path="/admin" element={<Navigate to="/admin/content" replace />} />
        <Route path="/admin/gear" element={<Navigate to="/admin/content" replace />} />
        <Route
          path="/admin/users"
          element={
            <AdminUserManagement
              isAdmin={Boolean(user?.isAdmin)}
              currentUserId={user?.id}
              authLoading={authLoading}
            />
          }
        />
      </Route>
      <Route
        path="/news"
        element={
          <NewsPage
            topBarProps={newsTopBarProps}
            items={newsItems}
            sources={newsSources}
            isLoading={isNewsLoading}
            isLoadingMore={isNewsLoadingMore}
            error={newsError}
            totalCount={newsTotalCount}
            onItemClick={onSelectNewsItem}
            onLoadMore={onLoadMoreNews}
          />
        }
      />
      <Route path="/shop" element={<ShopSection />} />
      <Route path="/builds" element={<PublicBuildsPage />} />
      <Route path="/builds/:id" element={<PublicBuildDetailsPage />} />
      <Route path="/builds/temp/:token" element={<TempBuildPage />} />
      <Route
        path="/gear-catalog"
        element={
          <GearCatalogPage
            onAddToInventory={onAddToInventoryFromCatalog}
          />
        }
      />
      <Route path="*" element={<Navigate to={isAuthenticated ? '/dashboard' : '/'} replace />} />
    </Routes>
  );
}

import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom';
import { AppRoutes } from './AppRoutes';
import type { User } from './authTypes';

type AppRoutesProps = Parameters<typeof AppRoutes>[0];

const mockUser: User = {
  id: 'user-1',
  email: 'pilot@example.com',
  displayName: 'Test Pilot',
  avatarUrl: undefined,
  status: 'active',
  emailVerified: true,
  isAdmin: false,
  isContentAdmin: false,
  isGearAdmin: false,
  createdAt: '2025-01-01T00:00:00Z',
};

function LoginLocationProbe() {
  const location = useLocation();
  return <div data-testid="login-location">{`${location.pathname}${location.search}`}</div>;
}

function createDefaultProps(): AppRoutesProps {
  return {
    isAuthenticated: false,
    user: null,
    authLoading: false,
    dashboardElement: <div>Dashboard Content</div>,
    onOpenLogin: vi.fn(),
    newsTopBarProps: {
      query: '',
      onQueryChange: vi.fn(),
      onSearch: vi.fn(),
      fromDate: '',
      toDate: '',
      onFromDateChange: vi.fn(),
      onToDateChange: vi.fn(),
      sort: 'newest' as const,
      onSortChange: vi.fn(),
      sourceType: 'all' as const,
      onSourceTypeChange: vi.fn(),
      onRefresh: vi.fn(),
      isRefreshing: false,
      refreshCooldown: 0,
      totalCount: 0,
    },
    newsItems: [],
    newsSources: [],
    isNewsLoading: false,
    isNewsLoadingMore: false,
    newsError: null,
    newsTotalCount: 0,
    onSelectNewsItem: vi.fn(),
    onLoadMoreNews: vi.fn(),
    onAddToInventoryFromCatalog: vi.fn(),
    inventoryCategory: null,
    inventorySummary: null,
    inventoryItems: [],
    isInventoryLoading: false,
    inventoryHasLoaded: false,
    inventoryError: null,
    onInventoryCategoryFilterChange: vi.fn(),
    onAddInventoryItem: vi.fn(),
    onOpenInventoryItem: vi.fn(),
    aircraftItems: [],
    isAircraftLoading: false,
    aircraftError: null,
    onSelectAircraft: vi.fn(),
    onEditAircraft: vi.fn(),
    onDeleteAircraft: vi.fn(),
    onAddAircraft: vi.fn(),
    onRadioError: vi.fn(),
    onBatteryError: vi.fn(),
    onSelectPilot: vi.fn(),
  };
}

function renderAppRoutes(initialPath: string, overrides: Partial<AppRoutesProps> = {}) {
  const props: AppRoutesProps = {
    ...createDefaultProps(),
    ...overrides,
  };

  render(
    <MemoryRouter initialEntries={[initialPath]}>
      <Routes>
        <Route path="/login" element={<LoginLocationProbe />} />
        <Route path="/*" element={<AppRoutes {...props} />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('AppRoutes protected route handling', () => {
  it('redirects unauthenticated users to /login with next path', () => {
    renderAppRoutes('/inventory?tab=all');

    expect(screen.getByTestId('login-location')).toHaveTextContent('/login?next=%2Finventory%3Ftab%3Dall');
  });

  it('guards /admin prefix routes for unauthenticated users', () => {
    renderAppRoutes('/admin');

    expect(screen.getByTestId('login-location')).toHaveTextContent('/login?next=%2Fadmin');
  });

  it('renders protected route when authenticated', () => {
    renderAppRoutes('/dashboard', { isAuthenticated: true, user: mockUser });

    expect(screen.getByText('Dashboard Content')).toBeInTheDocument();
    expect(screen.queryByTestId('login-location')).not.toBeInTheDocument();
  });

  it('shows loading fallback while auth status is pending', () => {
    renderAppRoutes('/dashboard', { authLoading: true });

    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });
});

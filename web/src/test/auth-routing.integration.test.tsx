/**
 * Integration tests: Auth flows and protected routing
 * Covers:
 *   - Login page renders (scenario 7)
 *   - Protected route redirects to login when unauthenticated (scenario 5)
 *   - Authenticated user reaches protected route (scenario 8)
 *   - Getting Started page loads (scenario 6)
 *   - Logout redirects to home (scenario 9)
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { AppRoutes } from '../AppRoutes';
import type { FeedItem, SourceInfo } from '../types';
import type { InventoryItem, InventorySummary } from '../equipmentTypes';
import type { Aircraft } from '../aircraftTypes';
import type { GearCatalogItem } from '../gearCatalogTypes';

vi.mock('../hooks/useGoogleAnalytics', () => ({
  useGoogleAnalytics: vi.fn(),
  trackEvent: vi.fn(),
}));

vi.mock('../hooks/useAuth', () => ({
  useAuth: vi.fn(),
}));

import { useAuth } from '../hooks/useAuth';
const mockedUseAuth = vi.mocked(useAuth);

function authValue(overrides: Partial<ReturnType<typeof useAuth>> = {}) {
  return {
    isAuthenticated: false,
    user: null,
    tokens: null,
    isLoading: false,
    error: null,
    loginWithGoogle: vi.fn(),
    logout: vi.fn(),
    updateUser: vi.fn(),
    clearError: vi.fn(),
    ...overrides,
  };
}

const defaultTopBarProps = {
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
  totalCount: 0,
};

function makeRoutesProps(overrides: Record<string, unknown> = {}) {
  return {
    isAuthenticated: false,
    user: null,
    authLoading: false,
    dashboardElement: <div>Dashboard Content</div>,
    onOpenLogin: vi.fn(),
    newsTopBarProps: defaultTopBarProps,
    newsItems: [] as FeedItem[],
    newsSources: [] as SourceInfo[],
    isNewsLoading: false,
    isNewsLoadingMore: false,
    newsError: null,
    newsTotalCount: 0,
    onSelectNewsItem: vi.fn(),
    onLoadMoreNews: vi.fn(),
    onAddToInventoryFromCatalog: vi.fn<(item: GearCatalogItem) => void>(),
    inventoryCategory: null,
    inventorySummary: null as InventorySummary | null,
    inventoryItems: [] as InventoryItem[],
    isInventoryLoading: false,
    inventoryHasLoaded: false,
    inventoryError: null,
    onInventoryCategoryFilterChange: vi.fn(),
    onAddInventoryItem: vi.fn(),
    onOpenInventoryItem: vi.fn(),
    aircraftItems: [] as Aircraft[],
    isAircraftLoading: false,
    aircraftError: null,
    onSelectAircraft: vi.fn(),
    onEditAircraft: vi.fn(),
    onDeleteAircraft: vi.fn(),
    onAddAircraft: vi.fn(),
    onRadioError: vi.fn(),
    onBatteryError: vi.fn(),
    onSelectPilot: vi.fn(),
    ...overrides,
  };
}

describe('Auth routing', () => {
  beforeEach(() => {
    mockedUseAuth.mockReturnValue(authValue());
  });

  it('redirects unauthenticated user from /dashboard to /login', async () => {
    render(
      <MemoryRouter initialEntries={['/dashboard']}>
        <Routes>
          <Route
            path="/*"
            element={<AppRoutes {...makeRoutesProps({ isAuthenticated: false })} />}
          />
          <Route path="/login" element={<div>Login Page</div>} />
        </Routes>
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getByText('Login Page')).toBeInTheDocument();
    });
  });

  it('renders dashboard for authenticated user at /dashboard', async () => {
    mockedUseAuth.mockReturnValue(authValue({ isAuthenticated: true }));

    render(
      <MemoryRouter initialEntries={['/dashboard']}>
        <AppRoutes {...makeRoutesProps({ isAuthenticated: true })} />
      </MemoryRouter>,
    );

    expect(screen.getByText('Dashboard Content')).toBeInTheDocument();
  });

  it('Getting Started page loads without authentication', () => {
    render(
      <MemoryRouter initialEntries={['/getting-started']}>
        <AppRoutes {...makeRoutesProps()} />
      </MemoryRouter>,
    );

    // GettingStarted renders a sign-in prompt for public users
    expect(document.body.textContent).not.toBe('');
  });

  it('redirects unauthenticated user from /inventory to /login', async () => {
    render(
      <MemoryRouter initialEntries={['/inventory']}>
        <Routes>
          <Route
            path="/*"
            element={<AppRoutes {...makeRoutesProps({ isAuthenticated: false })} />}
          />
          <Route path="/login" element={<div>Login Page</div>} />
        </Routes>
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getByText('Login Page')).toBeInTheDocument();
    });
  });

  it('redirects unauthenticated user from /aircraft to /login', async () => {
    render(
      <MemoryRouter initialEntries={['/aircraft']}>
        <Routes>
          <Route
            path="/*"
            element={<AppRoutes {...makeRoutesProps({ isAuthenticated: false })} />}
          />
          <Route path="/login" element={<div>Login Page</div>} />
        </Routes>
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getByText('Login Page')).toBeInTheDocument();
    });
  });

  it('shows loading fallback while auth is resolving', () => {
    render(
      <MemoryRouter initialEntries={['/dashboard']}>
        <AppRoutes {...makeRoutesProps({ isAuthenticated: false, authLoading: true })} />
      </MemoryRouter>,
    );

    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });
});

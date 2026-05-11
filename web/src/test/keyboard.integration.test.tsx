/**
 * Integration tests: Keyboard shortcuts & modal close order
 * Covers:
 *   - Escape key closes modals in correct order (scenario 21)
 *   - "/" hotkey focuses search input on news page (scenario 22)
 *
 * These tests render App directly (with mocked auth and analytics) so that
 * the global keydown listener in App.tsx is exercised.
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthProvider } from '../contexts/AuthContext';
import App from '../App';
import type { ReactNode } from 'react';

// Mock everything App depends on that isn't covered by MSW
vi.mock('../hooks/useAuth', () => ({
  useAuth: vi.fn(),
}));

vi.mock('../hooks/useGoogleAnalytics', () => ({
  useGoogleAnalytics: vi.fn(),
  trackEvent: vi.fn(),
}));

vi.mock('../api', () => ({
  getItems: vi.fn(),
  getSources: vi.fn(),
}));

vi.mock('../equipmentApi', () => ({
  getInventory: vi.fn(),
  getInventorySummary: vi.fn(),
  addInventoryItem: vi.fn(),
  updateInventoryItem: vi.fn(),
  deleteInventoryItem: vi.fn(),
  addEquipmentToInventory: vi.fn(),
}));

vi.mock('../aircraftApi', () => ({
  listAircraft: vi.fn(),
  createAircraft: vi.fn(),
  updateAircraft: vi.fn(),
  deleteAircraft: vi.fn(),
  getAircraftDetails: vi.fn(),
  setAircraftComponent: vi.fn(),
  setReceiverSettings: vi.fn(),
  getAircraftImageUrl: vi.fn(),
}));

vi.mock('../gearCatalogApi', () => ({
  getPopularGear: vi.fn().mockResolvedValue({ items: [] }),
  searchGearCatalog: vi.fn().mockResolvedValue({ items: [], totalCount: 0 }),
  moderateGearCatalogImageUpload: vi.fn(),
  saveGearCatalogImageUpload: vi.fn(),
  typeaheadSearch: vi.fn().mockResolvedValue({ items: [] }),
  findNearMatches: vi.fn().mockResolvedValue({ matches: [] }),
  getOrCreateCatalogItem: vi.fn(),
  createGearCatalogItem: vi.fn(),
  getGearCatalogItem: vi.fn(),
}));

import { useAuth } from '../hooks/useAuth';
import { getItems, getSources } from '../api';
import { getInventory, getInventorySummary } from '../equipmentApi';
import { listAircraft } from '../aircraftApi';

const mockedUseAuth = vi.mocked(useAuth);
const mockedGetItems = vi.mocked(getItems);
const mockedGetSources = vi.mocked(getSources);
const mockedGetInventory = vi.mocked(getInventory);
const mockedGetInventorySummary = vi.mocked(getInventorySummary);
const mockedListAircraft = vi.mocked(listAircraft);

import type { AggregatedResponse, FeedItem } from '../types';
import type { InventoryResponse, InventorySummary } from '../equipmentTypes';

const mockFeedItem: FeedItem = {
  id: 'item-1',
  title: 'Best FPV Motors of 2026',
  url: 'https://example.com/motors',
  source: 'src-1',
  sourceType: 'rss',
  publishedAt: '2026-03-01T12:00:00Z',
  summary: 'A roundup of the best motors.',
  tags: ['motors'],
};

const emptyNews: AggregatedResponse = {
  items: [],
  fetchedSources: [],
  failedSources: [],
  cacheHitRate: 1,
  generatedAt: '2026-03-26T00:00:00Z',
  totalCount: 0,
};

const emptyInventory: InventoryResponse = { items: [], totalCount: 0 };
const emptySummary: InventorySummary = {
  totalItems: 0,
  totalValue: 0,
  byCategory: {
    motors: 0, cameras: 0, frames: 0, vtx: 0, flight_controllers: 0,
    esc: 0, aio: 0, stacks: 0, propellers: 0, receivers: 0,
    batteries: 0, antennas: 0, gps: 0, accessories: 0,
  },
};

function makeAuthValue(overrides = {}) {
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

function NewsWrapper({ children }: { children: ReactNode }) {
  return (
    <MemoryRouter initialEntries={['/news']}>
      <AuthProvider>{children}</AuthProvider>
    </MemoryRouter>
  );
}


describe('App keyboard shortcuts', () => {
  beforeEach(() => {
    mockedUseAuth.mockReturnValue(makeAuthValue());
    mockedGetItems.mockResolvedValue({ ...emptyNews, items: [mockFeedItem], totalCount: 1 });
    mockedGetSources.mockResolvedValue({ sources: [], count: 0 });
    mockedGetInventory.mockResolvedValue(emptyInventory);
    mockedGetInventorySummary.mockResolvedValue(emptySummary);
    mockedListAircraft.mockResolvedValue({ aircraft: [], totalCount: 0 });
  });

  it('"/" hotkey focuses the first text input when no modal is open', async () => {
    const user = userEvent.setup();

    render(<App />, { wrapper: NewsWrapper });

    // Wait for the news feed to render
    await waitFor(() => {
      expect(screen.getByText('Best FPV Motors of 2026')).toBeInTheDocument();
    });

    // Move focus away from any input
    document.body.focus();

    await user.keyboard('/');

    // The search input should now be focused
    const searchInput = document.querySelector('input[type="text"]') as HTMLInputElement;
    expect(searchInput).not.toBeNull();
    expect(document.activeElement).toBe(searchInput);
  });

  it('Escape key closes the news item detail modal', async () => {
    const user = userEvent.setup();

    render(<App />, { wrapper: NewsWrapper });

    // Wait for news items to load
    await waitFor(() => {
      expect(screen.getByText('Best FPV Motors of 2026')).toBeInTheDocument();
    });

    // Click the news item to open the detail modal
    await user.click(screen.getByText('Best FPV Motors of 2026'));

    // The ItemDetail modal should appear (it renders the title a second time)
    await waitFor(() => {
      const links = screen.getAllByText('Best FPV Motors of 2026');
      expect(links.length).toBeGreaterThan(1);
    });

    // Press Escape to close the modal
    await user.keyboard('{Escape}');

    await waitFor(() => {
      // After Escape, only one instance of the title (the feed card) should remain
      expect(screen.getAllByText('Best FPV Motors of 2026')).toHaveLength(1);
    });
  });

  it('Escape key closes the aircraft form modal', async () => {
    const user = userEvent.setup();
    mockedUseAuth.mockReturnValue(
      makeAuthValue({
        isAuthenticated: true,
        user: {
          id: 'u1',
          email: 'a@b.com',
          displayName: 'Pilot',
          status: 'active',
          emailVerified: true,
          isAdmin: false,
          isContentAdmin: false,
          isGearAdmin: false,
          createdAt: '2026-01-01T00:00:00Z',
        },
      }),
    );

    render(
      <MemoryRouter initialEntries={['/aircraft']}>
        <AuthProvider>
          <App />
        </AuthProvider>
      </MemoryRouter>,
    );

    // Wait for aircraft page with the Add Aircraft button
    await waitFor(() => {
      expect(screen.queryAllByRole('button', { name: /add aircraft/i }).length).toBeGreaterThan(0);
    });

    // Click the Add Aircraft button (desktop controls)
    const addButtons = screen.getAllByRole('button', { name: /add aircraft/i });
    await user.click(addButtons[0]);

    // AircraftForm renders heading "Add New Aircraft" when not editing
    await waitFor(() => {
      expect(screen.getByText('Add New Aircraft')).toBeInTheDocument();
    });

    // Press Escape — App's keydown handler sets showAircraftForm=false
    await user.keyboard('{Escape}');

    await waitFor(() => {
      expect(screen.queryByText('Add New Aircraft')).not.toBeInTheDocument();
    });
  });
});

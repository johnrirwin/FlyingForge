import { describe, expect, it, vi, beforeEach } from 'vitest';
import { fireEvent, render, screen, waitFor } from '../test/test-utils';

import { GearCatalogPage } from './GearCatalogPage';
import type { GearCatalogItem } from '../gearCatalogTypes';

vi.mock('../gearCatalogApi', () => ({
  searchGearCatalog: vi.fn(),
  getPopularGear: vi.fn(),
  getGearCatalogItem: vi.fn(),
}));

vi.mock('../hooks/useAuth', () => ({
  useAuth: vi.fn(),
}));

import { getGearCatalogItem, getPopularGear, searchGearCatalog } from '../gearCatalogApi';
import { useAuth } from '../hooks/useAuth';

const mockedSearchGearCatalog = vi.mocked(searchGearCatalog);
const mockedGetPopularGear = vi.mocked(getPopularGear);
const mockedGetGearCatalogItem = vi.mocked(getGearCatalogItem);
const mockedUseAuth = vi.mocked(useAuth);

function buildCatalogItem(overrides: Partial<GearCatalogItem> = {}): GearCatalogItem {
  return {
    id: 'gear-1',
    gearType: 'prop',
    brand: 'Gemfan',
    model: '2520 Hurricane 3-Blade',
    source: 'admin',
    status: 'published',
    canonicalKey: 'prop|gemfan|2520-hurricane-3-blade',
    usageCount: 1,
    createdAt: '2026-02-19T00:00:00Z',
    updatedAt: '2026-02-19T00:00:00Z',
    imageStatus: 'approved',
    descriptionStatus: 'approved',
    description: 'Popular propeller',
    ...overrides,
  };
}

describe('GearCatalogPage', () => {
  beforeEach(() => {
    mockedUseAuth.mockReturnValue({
      isAuthenticated: true,
      user: null,
      isLoading: false,
      tokens: null,
      error: null,
      loginWithGoogle: vi.fn(),
      logout: vi.fn(),
      updateUser: vi.fn(),
      clearError: vi.fn(),
    });
    mockedSearchGearCatalog.mockResolvedValue({
      items: [],
      totalCount: 0,
    });
  });

  it('hydrates popular item detail before rendering shopping links in modal', async () => {
    const popularItem = buildCatalogItem({
      shoppingLinks: undefined,
    });
    const fullDetail = buildCatalogItem({
      shoppingLinks: ['https://pyrodrone.com/products/gemfan-2520'],
    });

    mockedGetPopularGear.mockResolvedValue({ items: [popularItem] });
    mockedGetGearCatalogItem.mockResolvedValue(fullDetail);

    render(<GearCatalogPage />);

    const card = await screen.findByLabelText('View details for Gemfan 2520 Hurricane 3-Blade');
    fireEvent.click(card);

    await waitFor(() => {
      expect(mockedGetGearCatalogItem).toHaveBeenCalledWith(popularItem.id);
    });

    expect(await screen.findByText('Shopping Links')).toBeInTheDocument();
    expect(screen.getByText('pyrodrone.com')).toBeInTheDocument();
  });
});

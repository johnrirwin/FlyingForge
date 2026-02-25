import { beforeEach, describe, expect, it, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '../test/test-utils';
import { CatalogSearchModal } from './CatalogSearchModal';
import type { GearCatalogItem } from '../gearCatalogTypes';
import type { InventoryItem } from '../equipmentTypes';

vi.mock('../gearCatalogApi', () => ({
  searchGearCatalog: vi.fn(),
  createGearCatalogItem: vi.fn(),
  findNearMatches: vi.fn(),
  getPopularGear: vi.fn(),
  getGearCatalogItem: vi.fn(),
}));

vi.mock('../equipmentApi', () => ({
  getInventory: vi.fn(),
}));

vi.mock('../adminApi', () => ({
  adminBulkDeleteGear: vi.fn(),
  adminFindNearMatches: vi.fn(),
}));

vi.mock('../hooks/useAuth', () => ({
  useAuth: vi.fn(),
}));

import { getPopularGear } from '../gearCatalogApi';
import { getGearCatalogItem } from '../gearCatalogApi';
import { getInventory } from '../equipmentApi';
import { useAuth } from '../hooks/useAuth';

const mockedGetPopularGear = vi.mocked(getPopularGear);
const mockedGetGearCatalogItem = vi.mocked(getGearCatalogItem);
const mockedGetInventory = vi.mocked(getInventory);
const mockedUseAuth = vi.mocked(useAuth);

function mockAuth(isAuthenticated: boolean) {
  mockedUseAuth.mockReturnValue({
    isAuthenticated,
    isLoading: false,
    user: null,
    tokens: null,
    error: null,
    loginWithGoogle: vi.fn(),
    logout: vi.fn(),
    updateUser: vi.fn(),
    clearError: vi.fn(),
  });
}

function renderModal(options?: { showInventoryMatches?: boolean; onSelectItem?: (item: GearCatalogItem) => void }) {
  render(
    <CatalogSearchModal
      isOpen
      onClose={vi.fn()}
      showInventoryMatches={options?.showInventoryMatches}
      onSelectItem={options?.onSelectItem ?? vi.fn()}
    />,
  );
}

describe('CatalogSearchModal', () => {
  beforeEach(() => {
    mockedGetPopularGear.mockReset();
    mockedGetPopularGear.mockResolvedValue({ items: [] });
    mockedGetGearCatalogItem.mockReset();
    mockedGetInventory.mockReset();
    mockedGetInventory.mockResolvedValue({ items: [], totalCount: 0 });
  });

  it('hides add-new-gear CTA for unauthenticated users', async () => {
    mockAuth(false);
    renderModal();
    await waitFor(() => {
      expect(mockedGetPopularGear).toHaveBeenCalled();
    });

    expect(screen.queryByText("Can't find it? Add new gear")).not.toBeInTheDocument();
  });

  it('shows add-new-gear CTA for authenticated users', async () => {
    mockAuth(true);
    renderModal();
    await waitFor(() => {
      expect(mockedGetPopularGear).toHaveBeenCalled();
    });

    expect(screen.getByText("Can't find it? Add new gear")).toBeInTheDocument();
  });

  it('shows inventory matches and lets user select one for build use', async () => {
    mockAuth(true);
    const onSelectItem = vi.fn();
    const inventoryItem: InventoryItem = {
      id: 'inv-1',
      name: 'GEP-MK5 O4 Pro DC Frame 5 inch',
      category: 'frames',
      manufacturer: 'GEPRC',
      quantity: 2,
      catalogId: 'catalog-frame-1',
      createdAt: '2026-02-01T00:00:00Z',
      updatedAt: '2026-02-01T00:00:00Z',
    };
    mockedGetInventory.mockResolvedValue({ items: [inventoryItem], totalCount: 1 });

    const catalogItem: GearCatalogItem = {
      id: 'catalog-frame-1',
      gearType: 'frame',
      brand: 'GEPRC',
      model: 'GEP-MK5 O4 Pro DC Frame 5 inch',
      status: 'published',
      source: 'admin',
      canonicalKey: 'frame:geprc:gep-mk5-o4-pro-dc-frame-5-inch',
      usageCount: 10,
      imageStatus: 'approved',
      descriptionStatus: 'approved',
      createdAt: '2026-02-01T00:00:00Z',
      updatedAt: '2026-02-01T00:00:00Z',
    };
    mockedGetGearCatalogItem.mockResolvedValue(catalogItem);

    renderModal({ showInventoryMatches: true, onSelectItem });

    expect(await screen.findByText('My Inventory')).toBeInTheDocument();
    const inventoryMatchButton = await screen.findByRole('button', { name: /GEP-MK5 O4 Pro DC Frame 5 inch/i });
    await userEvent.click(inventoryMatchButton);

    await waitFor(() => {
      expect(mockedGetGearCatalogItem).toHaveBeenCalledWith('catalog-frame-1');
      expect(onSelectItem).toHaveBeenCalledWith(catalogItem);
    });
  });
});

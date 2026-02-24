import { beforeEach, describe, expect, it, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '../test/test-utils';
import { PublicAircraftModal } from './PublicAircraftModal';
import type { AircraftPublic } from '../socialTypes';
import type { GearCatalogItem } from '../gearCatalogTypes';

vi.mock('../gearCatalogApi', () => ({
  getGearCatalogItem: vi.fn(),
}));

vi.mock('../hooks/useAuth', () => ({
  useAuth: vi.fn(),
}));

vi.mock('./GearDetailModal', () => ({
  GearDetailModal: ({ item, isOpen, onAddToInventory }: {
    item: GearCatalogItem;
    isOpen: boolean;
    onAddToInventory?: (item: GearCatalogItem) => void;
  }) => (isOpen ? (
    <div>
      <p>{item.model}</p>
      <button type="button" onClick={() => onAddToInventory?.(item)}>Add to My Inventory</button>
    </div>
  ) : null),
}));

import { getGearCatalogItem } from '../gearCatalogApi';
import { useAuth } from '../hooks/useAuth';

const mockedGetGearCatalogItem = vi.mocked(getGearCatalogItem);
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

function catalogItemFixture(overrides: Partial<GearCatalogItem> = {}): GearCatalogItem {
  return {
    id: 'catalog-frame-1',
    gearType: 'frame',
    brand: 'GEPRC',
    model: 'GEP-MK5 O4 Pro',
    status: 'published',
    source: 'admin',
    canonicalKey: 'frame:geprc:gep-mk5-o4-pro',
    usageCount: 10,
    imageStatus: 'approved',
    descriptionStatus: 'approved',
    createdAt: '2026-02-01T00:00:00Z',
    updatedAt: '2026-02-01T00:00:00Z',
    ...overrides,
  };
}

function aircraftFixture(overrides: Partial<AircraftPublic> = {}): AircraftPublic {
  return {
    id: 'aircraft-1',
    name: 'Kayou',
    type: 'freestyle',
    hasImage: false,
    createdAt: '2026-02-01T00:00:00Z',
    components: [
      {
        category: 'frame',
        manufacturer: 'GEPRC',
        name: 'GEP-MK5 O4 Pro DC Frame 5 inch',
        catalogId: 'catalog-frame-1',
      },
    ],
    ...overrides,
  };
}

describe('PublicAircraftModal', () => {
  beforeEach(() => {
    mockAuth(true);
    mockedGetGearCatalogItem.mockReset();
    mockedGetGearCatalogItem.mockResolvedValue(catalogItemFixture());
  });

  it('opens component details and can add to inventory', async () => {
    const onAddToInventory = vi.fn();
    render(
      <PublicAircraftModal
        aircraft={aircraftFixture()}
        onClose={vi.fn()}
        onAddToInventory={onAddToInventory}
      />,
    );

    await userEvent.click(screen.getByRole('button', { name: /View details for GEP-MK5 O4 Pro DC Frame 5 inch/i }));

    await waitFor(() => {
      expect(mockedGetGearCatalogItem).toHaveBeenCalledWith('catalog-frame-1');
    });

    expect(await screen.findByText('GEP-MK5 O4 Pro')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: 'Add to My Inventory' }));
    expect(onAddToInventory).toHaveBeenCalledWith(expect.objectContaining({ id: 'catalog-frame-1' }));
  });

  it('shows non-interactive rows when catalog details are unavailable', () => {
    render(
      <PublicAircraftModal
        aircraft={aircraftFixture({
          components: [
            {
              category: 'receiver',
              manufacturer: 'Acme',
              name: 'Receiver 123',
            },
          ],
        })}
        onClose={vi.fn()}
      />,
    );

    expect(screen.queryByRole('button', { name: /View details for Receiver 123/i })).not.toBeInTheDocument();
    expect(screen.getByText('Details unavailable')).toBeInTheDocument();
  });

  it('keeps a fixed modal height across tabs', async () => {
    render(
      <PublicAircraftModal
        aircraft={aircraftFixture()}
        onClose={vi.fn()}
      />,
    );

    const footer = screen.getByText("Viewing Kayou's build details");
    const modal = footer.parentElement;
    if (!(modal instanceof HTMLElement)) {
      throw new Error('Modal container not found');
    }

    expect(modal).toHaveClass('h-[90vh]');

    await userEvent.click(screen.getByRole('button', { name: /^Tuning/i }));
    expect(modal).toHaveClass('h-[90vh]');

    await userEvent.click(screen.getByRole('button', { name: /^Receiver/i }));
    expect(modal).toHaveClass('h-[90vh]');
  });
});

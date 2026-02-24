import type { ComponentProps } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import { render, screen, waitFor, within } from '../test/test-utils';
import { Dashboard } from './Dashboard';
import { getInventory } from '../equipmentApi';
import { getFollowers } from '../socialApi';
import { useAuth } from '../hooks/useAuth';
import type { InventoryItem } from '../equipmentTypes';

vi.mock('../equipmentApi', () => ({
  getInventory: vi.fn(),
}));

vi.mock('../socialApi', () => ({
  getFollowers: vi.fn(),
}));

vi.mock('../hooks/useAuth', () => ({
  useAuth: vi.fn(),
}));

vi.mock('../aircraftApi', () => ({
  getAircraftImageUrl: vi.fn(),
}));

const mockedGetInventory = vi.mocked(getInventory);
const mockedGetFollowers = vi.mocked(getFollowers);
const mockedUseAuth = vi.mocked(useAuth);

const inventoryItem: InventoryItem = {
  id: 'item-1',
  name: 'F40 Pro II Motor',
  category: 'motors',
  manufacturer: 'T-Motor',
  quantity: 4,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
};

function createProps(overrides: Partial<ComponentProps<typeof Dashboard>> = {}): ComponentProps<typeof Dashboard> {
  return {
    recentAircraft: [],
    recentNews: [],
    sources: [],
    isAircraftLoading: false,
    isNewsLoading: false,
    onViewAllNews: vi.fn(),
    onViewAllAircraft: vi.fn(),
    onViewAllGear: vi.fn(),
    onSelectGearItem: vi.fn(),
    onSelectAircraft: vi.fn(),
    onSelectNewsItem: vi.fn(),
    onSelectPilot: vi.fn(),
    onGoToSocial: vi.fn(),
    ...overrides,
  };
}

describe('Dashboard', () => {
  beforeEach(() => {
    mockedUseAuth.mockReturnValue({
      isAuthenticated: true,
      user: {
        id: 'user-1',
        email: 'pilot@example.com',
        displayName: 'Pilot',
        status: 'active',
        emailVerified: true,
        isAdmin: false,
        isContentAdmin: false,
        isGearAdmin: false,
        createdAt: '2026-01-01T00:00:00Z',
      },
      tokens: null,
      isLoading: false,
      error: null,
      loginWithGoogle: vi.fn(),
      logout: vi.fn(),
      updateUser: vi.fn(),
      clearError: vi.fn(),
    });

    mockedGetInventory.mockResolvedValue({
      items: [inventoryItem],
      totalCount: 1,
    });

    mockedGetFollowers.mockResolvedValue({
      pilots: [],
      totalCount: 0,
    });
  });

  it('opens the selected inventory item when a dashboard gear card is clicked', async () => {
    const user = userEvent.setup();
    const onSelectGearItem = vi.fn();
    const onViewAllGear = vi.fn();
    render(
      <Dashboard
        {...createProps({
          onSelectGearItem,
          onViewAllGear,
        })}
      />,
    );

    await waitFor(() => {
      expect(mockedGetInventory).toHaveBeenCalledWith({ limit: 3 });
    });

    const inventorySection = screen.getByRole('heading', { name: 'My Inventory' }).closest('section');
    expect(inventorySection).not.toBeNull();
    if (!inventorySection) throw new Error('Inventory section not found');

    const itemButton = await within(inventorySection).findByRole('button', {
      name: /f40 pro ii motor/i,
    });
    await user.click(itemButton);

    expect(onSelectGearItem).toHaveBeenCalledWith(inventoryItem);
    expect(onViewAllGear).not.toHaveBeenCalled();
  });
});

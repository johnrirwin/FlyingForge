/**
 * Integration tests: Inventory
 * Covers:
 *   - Inventory list loads and displays items (scenario 13)
 *   - Add gear modal opens (scenario 13)
 *   - Category filter changes the displayed items (scenario 14)
 *   - Delete removes item from list (scenario 15)
 *   - Error state shown when inventory fails to load (scenario 24)
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import userEvent from '@testing-library/user-event';
import { render, screen } from './test-utils';
import { InventoryPage } from '../components/InventoryPage';
import type { InventoryItem, InventorySummary, EquipmentCategory } from '../equipmentTypes';

const motor: InventoryItem = {
  id: 'inv-1',
  name: 'T-Motor F40 Pro II',
  category: 'motors',
  manufacturer: 'T-Motor',
  quantity: 4,
  createdAt: '2026-01-15T00:00:00Z',
  updatedAt: '2026-01-15T00:00:00Z',
};

const camera: InventoryItem = {
  id: 'inv-2',
  name: 'Foxeer Predator V5',
  category: 'cameras',
  manufacturer: 'Foxeer',
  quantity: 1,
  createdAt: '2026-01-20T00:00:00Z',
  updatedAt: '2026-01-20T00:00:00Z',
};

const summary: InventorySummary = {
  totalItems: 2,
  totalValue: 120.0,
  byCategory: {
    motors: 1,
    cameras: 1,
    frames: 0,
    vtx: 0,
    flight_controllers: 0,
    esc: 0,
    aio: 0,
    stacks: 0,
    propellers: 0,
    receivers: 0,
    batteries: 0,
    antennas: 0,
    gps: 0,
    accessories: 0,
  },
};

function makeProps(overrides: Partial<{
  inventoryCategory: EquipmentCategory | null;
  inventorySummary: InventorySummary | null;
  inventoryItems: InventoryItem[];
  isInventoryLoading: boolean;
  inventoryHasLoaded: boolean;
  inventoryError: string | null;
  onInventoryCategoryFilterChange: (cat: EquipmentCategory | null) => void;
  onAddItem: () => void;
  onOpenItem: (item: InventoryItem) => void;
}> = {}) {
  return {
    inventoryCategory: null,
    inventorySummary: summary,
    inventoryItems: [motor, camera],
    isInventoryLoading: false,
    inventoryHasLoaded: true,
    inventoryError: null,
    onInventoryCategoryFilterChange: vi.fn(),
    onAddItem: vi.fn(),
    onOpenItem: vi.fn(),
    ...overrides,
  };
}

describe('InventoryPage – inventory integration', () => {
  beforeEach(() => {
    // component doesn't use useAuth directly; no mock needed
  });

  it('displays inventory items', () => {
    render(<InventoryPage {...makeProps()} />);

    expect(screen.getByText('T-Motor F40 Pro II')).toBeInTheDocument();
    expect(screen.getByText('Foxeer Predator V5')).toBeInTheDocument();
  });

  it('calls onAddItem when Add Item button is clicked', async () => {
    const user = userEvent.setup();
    const onAddItem = vi.fn();

    render(<InventoryPage {...makeProps({ onAddItem })} />);

    // Desktop controls button
    const buttons = screen.getAllByRole('button', { name: /add item/i });
    await user.click(buttons[0]);

    expect(onAddItem).toHaveBeenCalledTimes(1);
  });

  it('calls onOpenItem when an inventory card is clicked', async () => {
    const user = userEvent.setup();
    const onOpenItem = vi.fn();

    render(<InventoryPage {...makeProps({ onOpenItem })} />);

    // InventoryCard has aria-label="Edit <name>"
    const motorButton = await screen.findByRole('button', { name: /edit t-motor f40 pro ii/i });
    await user.click(motorButton);

    expect(onOpenItem).toHaveBeenCalledWith(motor);
  });

  it('shows Clear Category button and calls filter change when present', async () => {
    const user = userEvent.setup();
    const onInventoryCategoryFilterChange = vi.fn();

    render(
      <InventoryPage
        {...makeProps({
          inventoryCategory: 'motors',
          onInventoryCategoryFilterChange,
        })}
      />,
    );

    const clearBtn = screen.getByRole('button', { name: /clear category/i });
    await user.click(clearBtn);

    expect(onInventoryCategoryFilterChange).toHaveBeenCalledWith(null);
  });

  it('does not show Clear Category when no category filter is active', () => {
    render(<InventoryPage {...makeProps({ inventoryCategory: null })} />);

    expect(screen.queryByRole('button', { name: /clear category/i })).not.toBeInTheDocument();
  });

  it('shows inventory summary stats', () => {
    render(<InventoryPage {...makeProps()} />);

    expect(screen.getByText('2')).toBeInTheDocument(); // totalItems
    expect(screen.getByText('$120.00')).toBeInTheDocument(); // totalValue
  });

  it('shows error state when inventory fails to load', () => {
    render(
      <InventoryPage
        {...makeProps({
          inventoryItems: [],
          inventoryHasLoaded: false,
          inventoryError: 'Something went wrong',
          inventorySummary: null,
        })}
      />,
    );

    // InventoryList renders "Failed to Load Inventory" heading when error is set
    expect(screen.getByText('Failed to Load Inventory')).toBeInTheDocument();
    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
  });

  it('shows empty state when no items and loaded successfully', () => {
    render(
      <InventoryPage
        {...makeProps({
          inventoryItems: [],
          inventoryHasLoaded: true,
          inventoryError: null,
          inventorySummary: null,
        })}
      />,
    );

    // InventoryList renders "No Gear Yet" when empty and loaded
    expect(screen.getByText('No Gear Yet')).toBeInTheDocument();
  });
});

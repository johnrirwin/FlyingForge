import { beforeEach, describe, it, expect, vi } from 'vitest';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { render } from '../test/test-utils';

const { catalogSearchModalMock } = vi.hoisted(() => ({
  catalogSearchModalMock: vi.fn(),
}));

vi.mock('./CatalogSearchModal', () => ({
  CatalogSearchModal: (props: { initialGearType?: string }) => {
    catalogSearchModalMock(props);
    return <div data-testid="catalog-search-modal" />;
  },
}));

import { AddGearModal } from './AddGearModal';
import type { InventoryItem } from '../equipmentTypes';
import { INVENTORY_ITEM_DETAILS_SPEC_KEY } from '../inventoryItemDetails';

const editItem: InventoryItem = {
  id: 'inv-1',
  name: 'Test Motor',
  category: 'motors',
  manufacturer: 'Acme',
  quantity: 1,
  notes: '',
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
};

describe('AddGearModal quantity editing', () => {
  beforeEach(() => {
    catalogSearchModalMock.mockClear();
  });

  it('passes initial category to catalog search as initial gear type', () => {
    render(
      <AddGearModal
        isOpen
        onClose={vi.fn()}
        onSubmit={vi.fn().mockResolvedValue(undefined)}
        initialCategory="cameras"
      />,
    );

    expect(screen.getByTestId('catalog-search-modal')).toBeInTheDocument();
    expect(catalogSearchModalMock).toHaveBeenCalledWith(
      expect.objectContaining({ initialGearType: 'camera' }),
    );
  });

  it('allows clearing and replacing an existing quantity', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);

    render(
      <AddGearModal
        isOpen
        onClose={vi.fn()}
        onSubmit={onSubmit}
        editItem={editItem}
      />,
    );

    const quantityInput = screen.getByRole('spinbutton') as HTMLInputElement;
    const submitButton = screen.getByRole('button', { name: 'Save Changes' });

    fireEvent.change(quantityInput, { target: { value: '' } });
    expect(quantityInput.value).toBe('');
    expect(submitButton).toBeDisabled();

    fireEvent.change(quantityInput, { target: { value: '5' } });
    expect(quantityInput.value).toBe('5');

    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith(
        expect.objectContaining({ quantity: 5 }),
      );
    });
  });

  it('submits quantity 0 when requested', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);

    render(
      <AddGearModal
        isOpen
        onClose={vi.fn()}
        onSubmit={onSubmit}
        editItem={editItem}
      />,
    );

    const quantityInput = screen.getByRole('spinbutton') as HTMLInputElement;
    fireEvent.change(quantityInput, { target: { value: '0' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }));

    await waitFor(() => {
      expect(onSubmit).toHaveBeenCalledWith(
        expect.objectContaining({ quantity: 0 }),
      );
    });
  });

  it('does not render item tabs when editing a single-quantity item', () => {
    render(
      <AddGearModal
        isOpen
        onClose={vi.fn()}
        onSubmit={vi.fn().mockResolvedValue(undefined)}
        editItem={editItem}
      />,
    );

    expect(screen.queryByRole('button', { name: 'Item 1' })).not.toBeInTheDocument();
    expect(screen.getByText('Set purchase and build details for this item.')).toBeInTheDocument();
  });

  it('shows the item image while editing when available', () => {
    render(
      <AddGearModal
        isOpen
        onClose={vi.fn()}
        onSubmit={vi.fn().mockResolvedValue(undefined)}
        editItem={{
          ...editItem,
          imageUrl: 'https://example.com/motor.jpg',
        }}
      />,
    );

    const image = screen.getByRole('img', { name: 'Test Motor' });
    expect(image).toHaveAttribute('src', 'https://example.com/motor.jpg');
  });

  it('prefills newly added quantity tabs with the existing item purchase price', () => {
    const singleItem: InventoryItem = {
      ...editItem,
      quantity: 1,
      purchasePrice: 124.95,
      purchaseSeller: 'RDQ',
      buildId: 'Quad One',
    };

    render(
      <AddGearModal
        isOpen
        onClose={vi.fn()}
        onSubmit={vi.fn().mockResolvedValue(undefined)}
        editItem={singleItem}
      />,
    );

    fireEvent.change(screen.getByRole('spinbutton'), { target: { value: '2' } });
    fireEvent.click(screen.getByRole('button', { name: 'Item 2' }));

    expect((screen.getByPlaceholderText('0.00') as HTMLInputElement).value).toBe('124.95');
  });

  it('shows per-item tabs and persists per-item purchase details while editing', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    const multiItem: InventoryItem = {
      ...editItem,
      quantity: 2,
      purchasePrice: 79.99,
      purchaseSeller: 'RDQ',
      buildId: 'Quad One',
    };

    render(
      <AddGearModal
        isOpen
        onClose={vi.fn()}
        onSubmit={onSubmit}
        editItem={multiItem}
      />,
    );

    expect(screen.getByRole('button', { name: 'Item 1' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Item 2' })).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Item 2' }));
    fireEvent.change(screen.getByPlaceholderText('0.00'), { target: { value: '91.5' } });
    fireEvent.change(screen.getByPlaceholderText('Seller name'), { target: { value: 'GetFPV' } });
    fireEvent.change(screen.getByPlaceholderText('e.g., 5" Freestyle'), { target: { value: 'Quad Two' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));

    const submitted = onSubmit.mock.calls[0][0];
    expect(submitted).toEqual(
      expect.objectContaining({
        quantity: 2,
        purchasePrice: 79.99,
        purchaseSeller: 'RDQ',
        buildId: 'Quad One',
      }),
    );
    expect(submitted.specs).toEqual(
      expect.objectContaining({
        [INVENTORY_ITEM_DETAILS_SPEC_KEY]: [
          { purchasePrice: 79.99, purchaseSeller: 'RDQ', buildId: 'Quad One' },
          { purchasePrice: 91.5, purchaseSeller: 'GetFPV', buildId: 'Quad Two' },
        ],
      }),
    );
  });

  it('rejects quantity 0 while adding new gear', async () => {
    const onSubmit = vi.fn().mockResolvedValue(undefined);

    render(
      <AddGearModal
        isOpen
        onClose={vi.fn()}
        onSubmit={onSubmit}
        equipmentItem={{
          id: 'equip-1',
          name: 'Shop Motor',
          category: 'motors',
          manufacturer: 'Acme',
          price: 21.5,
          currency: 'USD',
          seller: 'Test Seller',
          sellerId: 'seller-1',
          productUrl: 'https://example.com/motor',
          inStock: true,
          stockQty: 10,
          lastChecked: '2026-02-11T00:00:00Z',
        }}
      />,
    );

    const quantityInput = screen.getByRole('spinbutton') as HTMLInputElement;
    fireEvent.change(quantityInput, { target: { value: '0' } });
    fireEvent.click(screen.getByRole('button', { name: 'Add to My Inventory' }));

    expect(quantityInput).toHaveAttribute('min', '1');
    expect(onSubmit).not.toHaveBeenCalled();
  });
});

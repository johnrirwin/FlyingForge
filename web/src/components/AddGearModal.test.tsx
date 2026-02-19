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

import { describe, it, expect, vi } from 'vitest';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { render } from '../test/test-utils';
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
});

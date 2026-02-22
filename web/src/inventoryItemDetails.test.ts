import { describe, expect, it } from 'vitest';
import type { InventoryItem } from './equipmentTypes';
import {
  buildInventoryItemDetails,
  getInventoryItemDetailsFromSpecs,
  INVENTORY_ITEM_DETAILS_SPEC_KEY,
  setInventoryItemDetailsOnSpecs,
} from './inventoryItemDetails';

const baseItem: InventoryItem = {
  id: 'inv-1',
  name: 'Frame',
  category: 'frames',
  quantity: 2,
  purchasePrice: 89.99,
  purchaseSeller: 'RDQ',
  buildId: 'Freestyle 5',
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
};

describe('inventoryItemDetails', () => {
  it('falls back to top-level fields when specs do not include item details', () => {
    expect(buildInventoryItemDetails(baseItem)).toEqual([
      { purchasePrice: 89.99, purchaseSeller: 'RDQ', buildId: 'Freestyle 5' },
      { purchasePrice: 89.99, purchaseSeller: 'RDQ', buildId: 'Freestyle 5' },
    ]);
  });

  it('reads and normalizes item details from specs', () => {
    const details = getInventoryItemDetailsFromSpecs({
      [INVENTORY_ITEM_DETAILS_SPEC_KEY]: [
        { purchasePrice: '90.50', purchaseSeller: '  RDQ  ', buildId: ' Quad A ' },
        { purchasePrice: 95.0, purchaseSeller: 'GetFPV', buildId: '' },
      ],
    });

    expect(details).toEqual([
      { purchasePrice: 90.5, purchaseSeller: 'RDQ', buildId: 'Quad A' },
      { purchasePrice: 95, purchaseSeller: 'GetFPV', buildId: undefined },
    ]);
  });

  it('preserves array position when a stored detail entry is malformed', () => {
    const details = getInventoryItemDetailsFromSpecs({
      [INVENTORY_ITEM_DETAILS_SPEC_KEY]: [
        { purchasePrice: 90, purchaseSeller: 'RDQ', buildId: 'Quad A' },
        'unexpected-value',
        { purchasePrice: 95, purchaseSeller: 'GetFPV', buildId: 'Quad C' },
      ],
    });

    expect(details).toEqual([
      { purchasePrice: 90, purchaseSeller: 'RDQ', buildId: 'Quad A' },
      {},
      { purchasePrice: 95, purchaseSeller: 'GetFPV', buildId: 'Quad C' },
    ]);
  });

  it('preserves existing specs while storing per-item details', () => {
    const updated = setInventoryItemDetailsOnSpecs(
      { kv: '1950', [INVENTORY_ITEM_DETAILS_SPEC_KEY]: [{ purchaseSeller: 'Old Seller' }] },
      [
        { purchasePrice: 80, purchaseSeller: 'Seller A', buildId: 'A' },
        { purchasePrice: 85, purchaseSeller: 'Seller B', buildId: 'B' },
      ],
    );

    expect(updated).toEqual({
      kv: '1950',
      [INVENTORY_ITEM_DETAILS_SPEC_KEY]: [
        { purchasePrice: 80, purchaseSeller: 'Seller A', buildId: 'A' },
        { purchasePrice: 85, purchaseSeller: 'Seller B', buildId: 'B' },
      ],
    });
  });

  it('fills missing stored detail entries using top-level fallback fields', () => {
    const item: InventoryItem = {
      ...baseItem,
      quantity: 2,
      specs: {
        [INVENTORY_ITEM_DETAILS_SPEC_KEY]: [
          { purchasePrice: 89.99, purchaseSeller: 'RDQ', buildId: 'Freestyle 5' },
        ],
      },
    };

    expect(buildInventoryItemDetails(item)).toEqual([
      { purchasePrice: 89.99, purchaseSeller: 'RDQ', buildId: 'Freestyle 5' },
      { purchasePrice: 89.99, purchaseSeller: 'RDQ', buildId: 'Freestyle 5' },
    ]);
  });
});

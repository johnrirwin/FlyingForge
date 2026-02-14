import { describe, it, expect } from 'vitest';
import { upsertInventoryItem } from './inventoryUtils';
import type { InventoryItem } from './equipmentTypes';

describe('upsertInventoryItem', () => {
  it('appends when the item is new', () => {
    const next: InventoryItem = {
      id: 'inv-1',
      name: 'Test Motor',
      category: 'motors',
      quantity: 1,
      createdAt: '2026-02-14T00:00:00Z',
      updatedAt: '2026-02-14T00:00:00Z',
    };

    expect(upsertInventoryItem([], next)).toEqual([next]);
  });

  it('replaces an existing item (no duplicates)', () => {
    const existing: InventoryItem = {
      id: 'inv-1',
      name: 'Test Motor',
      category: 'motors',
      quantity: 3,
      createdAt: '2026-02-14T00:00:00Z',
      updatedAt: '2026-02-14T00:00:00Z',
    };

    const updated: InventoryItem = {
      ...existing,
      quantity: 4,
      updatedAt: '2026-02-14T01:00:00Z',
    };

    const result = upsertInventoryItem([existing], updated);
    expect(result).toHaveLength(1);
    expect(result[0]).toEqual(updated);
  });
});


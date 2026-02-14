import type { InventoryItem } from './equipmentTypes';

export function upsertInventoryItem(items: InventoryItem[], next: InventoryItem): InventoryItem[] {
  const existingIndex = items.findIndex((item) => item.id === next.id);
  if (existingIndex === -1) {
    return [...items, next];
  }

  const updated = items.slice();
  updated[existingIndex] = next;
  return updated;
}


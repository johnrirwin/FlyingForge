import type { InventoryItem } from './equipmentTypes';

export const INVENTORY_ITEM_DETAILS_SPEC_KEY = '__ff_inventory_item_details';

export interface InventoryItemDetail {
  purchasePrice?: number;
  purchaseSeller?: string;
  buildId?: string;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function trimString(value: unknown): string | undefined {
  if (typeof value !== 'string') return undefined;
  const trimmed = value.trim();
  return trimmed.length > 0 ? trimmed : undefined;
}

function parsePrice(value: unknown): number | undefined {
  if (typeof value === 'number') {
    return Number.isFinite(value) && value >= 0 ? value : undefined;
  }
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value);
    return Number.isFinite(parsed) && parsed >= 0 ? parsed : undefined;
  }
  return undefined;
}

function sanitizeDetail(detail: {
  purchasePrice?: unknown;
  purchaseSeller?: unknown;
  buildId?: unknown;
}): InventoryItemDetail {
  return {
    purchasePrice: parsePrice(detail.purchasePrice),
    purchaseSeller: trimString(detail.purchaseSeller),
    buildId: trimString(detail.buildId),
  };
}

function cloneDetail(detail: InventoryItemDetail): InventoryItemDetail {
  return {
    purchasePrice: detail.purchasePrice,
    purchaseSeller: detail.purchaseSeller,
    buildId: detail.buildId,
  };
}

function baseSpecsWithoutDetails(specs?: Record<string, unknown>): Record<string, unknown> {
  if (!isRecord(specs)) return {};
  const next: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(specs)) {
    if (key === INVENTORY_ITEM_DETAILS_SPEC_KEY) continue;
    next[key] = value;
  }
  return next;
}

export function getInventoryItemDetailsFromSpecs(specs?: Record<string, unknown>): InventoryItemDetail[] {
  if (!isRecord(specs)) return [];

  const rawDetails = specs[INVENTORY_ITEM_DETAILS_SPEC_KEY];
  if (!Array.isArray(rawDetails)) return [];

  const parsed: InventoryItemDetail[] = [];
  for (const detail of rawDetails) {
    const record = isRecord(detail) ? detail : {};
    parsed.push(
      sanitizeDetail({
        purchasePrice: record.purchasePrice,
        purchaseSeller: record.purchaseSeller,
        buildId: record.buildId,
      }),
    );
  }
  return parsed;
}

export function buildInventoryItemDetails(item: InventoryItem): InventoryItemDetail[] {
  const quantity = Number.isInteger(item.quantity) && item.quantity > 0 ? item.quantity : 0;
  const detailsFromSpecs = getInventoryItemDetailsFromSpecs(item.specs);
  const fallback = sanitizeDetail({
    purchasePrice: item.purchasePrice,
    purchaseSeller: item.purchaseSeller,
    buildId: item.buildId,
  });

  if (quantity === 0) {
    return [];
  }

  const details = detailsFromSpecs.map(cloneDetail);
  if (details.length === 0) {
    return Array.from({ length: quantity }, () => cloneDetail(fallback));
  }

  if (details.length < quantity) {
    const missing = quantity - details.length;
    for (let i = 0; i < missing; i += 1) {
      details.push(cloneDetail(fallback));
    }
  }

  if (details.length > quantity) {
    return details.slice(0, quantity);
  }

  return details;
}

export function setInventoryItemDetailsOnSpecs(
  specs: Record<string, unknown> | undefined,
  details: InventoryItemDetail[],
): Record<string, unknown> | undefined {
  const nextSpecs = baseSpecsWithoutDetails(specs);
  if (details.length > 0) {
    nextSpecs[INVENTORY_ITEM_DETAILS_SPEC_KEY] = details.map(sanitizeDetail);
  }

  return Object.keys(nextSpecs).length > 0 ? nextSpecs : undefined;
}

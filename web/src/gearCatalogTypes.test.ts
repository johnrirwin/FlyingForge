import { describe, expect, it } from 'vitest';

import {
  GEAR_TYPES,
  equipmentCategoryToGearType,
  extractDomainFromUrl,
  gearTypeToEquipmentCategory,
  getShoppingLinkDisplayName,
} from './gearCatalogTypes';

describe('getShoppingLinkDisplayName', () => {
  it('maps Amazon short links to amazon label', () => {
    expect(getShoppingLinkDisplayName('https://a.co/d/0hg3M3zy')).toBe('amazon');
  });

  it('maps amazon domains to amazon label', () => {
    expect(getShoppingLinkDisplayName('https://www.amazon.com/dp/B0TEST123')).toBe('amazon');
  });

  it('falls back to extracted domain for non-amazon links', () => {
    expect(getShoppingLinkDisplayName('https://pyrodrone.com/products/foo')).toBe('pyrodrone.com');
  });
});

describe('extractDomainFromUrl', () => {
  it('returns empty string for invalid urls', () => {
    expect(extractDomainFromUrl('not-a-url')).toBe('');
  });
});

describe('stack gear type support', () => {
  it('maps stack gear type to stacks inventory category', () => {
    expect(gearTypeToEquipmentCategory('stack')).toBe('stacks');
  });

  it('maps stacks inventory category back to stack gear type', () => {
    expect(equipmentCategoryToGearType('stacks')).toBe('stack');
  });

  it('exposes stack in gear type selector options', () => {
    expect(GEAR_TYPES.some((entry) => entry.value === 'stack')).toBe(true);
  });

  it('maps battery gear type to batteries inventory category', () => {
    expect(gearTypeToEquipmentCategory('battery')).toBe('batteries');
  });

  it('maps batteries inventory category back to battery gear type', () => {
    expect(equipmentCategoryToGearType('batteries')).toBe('battery');
  });
});

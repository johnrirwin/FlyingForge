import { describe, expect, it } from 'vitest';

import { extractDomainFromUrl, getShoppingLinkDisplayName } from './gearCatalogTypes';

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

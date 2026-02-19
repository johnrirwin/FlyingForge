import { describe, expect, it, vi } from 'vitest';
import { render, screen } from '../test/test-utils';

import { GearDetailModal } from './GearDetailModal';
import type { GearCatalogItem } from '../gearCatalogTypes';

function buildItem(overrides: Partial<GearCatalogItem> = {}): GearCatalogItem {
  return {
    id: 'gear-1',
    gearType: 'prop',
    brand: 'Gemfan',
    model: '2520 Hurricane 3-Blade',
    source: 'admin',
    status: 'published',
    canonicalKey: 'prop|gemfan|2520-hurricane-3-blade',
    usageCount: 1,
    shoppingLinks: ['https://pyrodrone.com/products/gemfan-2520'],
    createdAt: '2026-02-19T00:00:00Z',
    updatedAt: '2026-02-19T00:00:00Z',
    imageStatus: 'approved',
    descriptionStatus: 'approved',
    ...overrides,
  };
}

describe('GearDetailModal', () => {
  it('renders shopping links as pill labels', () => {
    render(
      <GearDetailModal
        item={buildItem()}
        isOpen
        onClose={vi.fn()}
        isAuthenticated
      />,
    );

    const link = screen.getByRole('link', { name: /Visit pyrodrone\.com/i });
    expect(link).toBeInTheDocument();
    expect(link.className).toContain('rounded-full');
    expect(link.className).toContain('bg-primary-600/20');
  });
});

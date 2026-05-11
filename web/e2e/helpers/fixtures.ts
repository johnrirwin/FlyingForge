/**
 * Shared fixture data for Playwright E2E tests.
 */

export const MOCK_FEED_ITEMS = [
  {
    id: 'item-1',
    title: 'Best FPV Motors of 2026',
    url: 'https://fpvblog.example.com/motors',
    source: 'src-1',
    sourceType: 'rss',
    publishedAt: '2026-03-01T12:00:00Z',
    summary: 'A roundup of the best motors for freestyle and racing.',
    tags: ['motors'],
  },
  {
    id: 'item-2',
    title: 'New DJI Goggles Review',
    url: 'https://fpvblog.example.com/goggles',
    source: 'src-1',
    sourceType: 'rss',
    publishedAt: '2026-03-02T12:00:00Z',
    summary: 'We test the latest DJI goggles.',
    tags: ['goggles'],
  },
];

export const MOCK_SOURCES = [
  {
    id: 'src-1',
    name: 'FPV Blog',
    url: 'https://fpvblog.example.com',
    sourceType: 'rss',
    description: 'FPV news',
    feedType: 'rss',
    enabled: true,
  },
];

export const MOCK_AGGREGATED_RESPONSE = {
  items: MOCK_FEED_ITEMS,
  fetchedSources: ['src-1'],
  failedSources: [],
  cacheHitRate: 1,
  generatedAt: '2026-03-26T00:00:00Z',
  totalCount: 2,
};

export const MOCK_SOURCES_RESPONSE = {
  sources: MOCK_SOURCES,
  count: 1,
};

export const MOCK_AIRCRAFT = [
  {
    id: 'aircraft-1',
    userId: 'user-1',
    name: 'Shredder 5"',
    type: 'freestyle',
    hasImage: false,
    createdAt: '2026-02-01T00:00:00Z',
    updatedAt: '2026-02-01T00:00:00Z',
  },
];

export const MOCK_INVENTORY = [
  {
    id: 'inv-1',
    name: 'T-Motor F40 Pro II',
    category: 'motors',
    manufacturer: 'T-Motor',
    quantity: 4,
    createdAt: '2026-01-15T00:00:00Z',
    updatedAt: '2026-01-15T00:00:00Z',
  },
];

export const MOCK_INVENTORY_SUMMARY = {
  totalItems: 1,
  totalValue: 60.0,
  byCategory: {
    motors: 1,
    cameras: 0,
    frames: 0,
    vtx: 0,
    flight_controllers: 0,
    esc: 0,
    aio: 0,
    stacks: 0,
    propellers: 0,
    receivers: 0,
    batteries: 0,
    antennas: 0,
    gps: 0,
    accessories: 0,
  },
};

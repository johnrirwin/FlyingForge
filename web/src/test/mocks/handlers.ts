import { http, HttpResponse } from 'msw';
import type { FeedItem, SourceInfo, AggregatedResponse } from '../../types';
import type { InventoryItem, InventoryResponse, InventorySummary } from '../../equipmentTypes';
import type { Aircraft, AircraftListResponse, AircraftDetailsResponse } from '../../aircraftTypes';
import type { User } from '../../authTypes';

// ── Fixtures ──────────────────────────────────────────────────────────────────

export const mockUser: User = {
  id: 'user-1',
  email: 'pilot@example.com',
  displayName: 'Test Pilot',
  status: 'active',
  emailVerified: true,
  isAdmin: false,
  isContentAdmin: false,
  isGearAdmin: false,
  createdAt: '2026-01-01T00:00:00Z',
};

export const mockSource: SourceInfo = {
  id: 'src-1',
  name: 'FPV Blog',
  url: 'https://fpvblog.example.com',
  sourceType: 'rss',
  description: 'FPV news',
  feedType: 'rss',
  enabled: true,
};

export const mockFeedItem: FeedItem = {
  id: 'item-1',
  title: 'Best FPV Motors of 2026',
  url: 'https://fpvblog.example.com/motors',
  source: 'src-1',
  sourceType: 'rss',
  publishedAt: '2026-03-01T12:00:00Z',
  summary: 'A roundup of the best motors for freestyle and racing.',
  tags: ['motors', 'freestyle'],
};

export const mockFeedItem2: FeedItem = {
  id: 'item-2',
  title: 'New DJI Goggles Review',
  url: 'https://fpvblog.example.com/goggles',
  source: 'src-1',
  sourceType: 'rss',
  publishedAt: '2026-03-02T12:00:00Z',
  summary: 'We test the latest DJI goggles.',
  tags: ['goggles', 'fpv'],
};

export const mockInventoryItem: InventoryItem = {
  id: 'inv-1',
  name: 'T-Motor F40 Pro II',
  category: 'motors',
  manufacturer: 'T-Motor',
  quantity: 4,
  createdAt: '2026-01-15T00:00:00Z',
  updatedAt: '2026-01-15T00:00:00Z',
};

export const mockInventoryItem2: InventoryItem = {
  id: 'inv-2',
  name: 'Foxeer Predator V5',
  category: 'cameras',
  manufacturer: 'Foxeer',
  quantity: 1,
  createdAt: '2026-01-20T00:00:00Z',
  updatedAt: '2026-01-20T00:00:00Z',
};

export const mockInventorySummary: InventorySummary = {
  totalItems: 2,
  totalValue: 120.0,
  byCategory: {
    motors: 1,
    cameras: 1,
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

export const mockAircraft: Aircraft = {
  id: 'aircraft-1',
  userId: 'user-1',
  name: 'Shredder 5"',
  type: 'freestyle',
  hasImage: false,
  createdAt: '2026-02-01T00:00:00Z',
  updatedAt: '2026-02-01T00:00:00Z',
};

export const mockAircraftDetails: AircraftDetailsResponse = {
  aircraft: mockAircraft,
  components: [],
};

// ── Handler factories (allow tests to override per-suite) ─────────────────────

export function makeAggregatedResponse(overrides?: Partial<AggregatedResponse>): AggregatedResponse {
  return {
    items: [mockFeedItem, mockFeedItem2],
    fetchedSources: ['src-1'],
    failedSources: [],
    cacheHitRate: 1,
    generatedAt: '2026-03-26T00:00:00Z',
    totalCount: 2,
    ...overrides,
  };
}

export function makeInventoryResponse(overrides?: Partial<InventoryResponse>): InventoryResponse {
  return {
    items: [mockInventoryItem, mockInventoryItem2],
    totalCount: 2,
    ...overrides,
  };
}

export function makeAircraftListResponse(overrides?: Partial<AircraftListResponse>): AircraftListResponse {
  return {
    aircraft: [mockAircraft],
    totalCount: 1,
    ...overrides,
  };
}

// ── Default MSW handlers ──────────────────────────────────────────────────────

export const handlers = [
  // News
  http.get('/api/items', () => HttpResponse.json(makeAggregatedResponse())),
  http.get('/api/sources', () =>
    HttpResponse.json({ sources: [mockSource], count: 1 }),
  ),
  http.get('/api/items/:id', ({ params }) => {
    const item = params.id === 'item-1' ? mockFeedItem : mockFeedItem2;
    return HttpResponse.json(item);
  }),

  // Auth
  http.get('/api/auth/me', () => HttpResponse.json(mockUser)),
  http.post('/api/auth/google', () =>
    HttpResponse.json({
      user: mockUser,
      tokens: {
        accessToken: 'access-token',
        refreshToken: 'refresh-token',
        tokenType: 'Bearer',
        expiresIn: 3600,
      },
    }),
  ),
  http.post('/api/auth/logout', () => HttpResponse.json({ success: true })),
  http.post('/api/auth/refresh', () =>
    HttpResponse.json({
      accessToken: 'new-access-token',
      refreshToken: 'new-refresh-token',
      tokenType: 'Bearer',
      expiresIn: 3600,
    }),
  ),

  // Inventory
  http.get('/api/inventory', () => HttpResponse.json(makeInventoryResponse())),
  http.get('/api/inventory/summary', () => HttpResponse.json(mockInventorySummary)),
  http.post('/api/inventory', () => HttpResponse.json(mockInventoryItem, { status: 201 })),
  http.put('/api/inventory/:id', ({ params }) => {
    const updated: InventoryItem = { ...mockInventoryItem, id: params.id as string };
    return HttpResponse.json(updated);
  }),
  http.delete('/api/inventory/:id', () => new HttpResponse(null, { status: 204 })),

  // Aircraft
  http.get('/api/aircraft', () => HttpResponse.json(makeAircraftListResponse())),
  http.post('/api/aircraft', () => HttpResponse.json(mockAircraft, { status: 201 })),
  http.get('/api/aircraft/:id', () => HttpResponse.json(mockAircraftDetails)),
  http.put('/api/aircraft/:id', () => HttpResponse.json(mockAircraft)),
  http.delete('/api/aircraft/:id', () => new HttpResponse(null, { status: 204 })),
  http.put('/api/aircraft/:id/components/:category', () =>
    HttpResponse.json({ success: true }),
  ),

  // Social / Pilots
  http.get('/api/pilots', () =>
    HttpResponse.json({ pilots: [], totalCount: 0 }),
  ),
  http.get('/api/pilots/discover', () =>
    HttpResponse.json({ pilots: [], totalCount: 0 }),
  ),
  http.get('/api/pilots/:id', () =>
    HttpResponse.json({ pilot: null }),
  ),
  http.get('/api/social/following', () =>
    HttpResponse.json({ pilots: [], totalCount: 0 }),
  ),
  http.get('/api/social/followers', () =>
    HttpResponse.json({ pilots: [], totalCount: 0 }),
  ),

  // Profile
  http.get('/api/me/profile', () =>
    HttpResponse.json({
      ...mockUser,
      effectiveAvatarUrl: '',
      updatedAt: '2026-01-01T00:00:00Z',
    }),
  ),

  // Radio
  http.get('/api/radio', () =>
    HttpResponse.json({ profiles: [], totalCount: 0 }),
  ),

  // Batteries
  http.get('/api/batteries', () =>
    HttpResponse.json({ batteries: [], totalCount: 0 }),
  ),

  // Builds
  http.get('/api/builds', () =>
    HttpResponse.json({ builds: [], totalCount: 0 }),
  ),

  // Gear catalog
  http.get('/api/gear-catalog', () =>
    HttpResponse.json({ items: [], totalCount: 0 }),
  ),
  http.get('/api/gear-catalog/popular', () =>
    HttpResponse.json({ items: [], totalCount: 0 }),
  ),
  http.get('/api/gear-catalog/:id', () =>
    HttpResponse.json({ item: null }),
  ),

  // Health
  http.get('/health', () =>
    HttpResponse.json({ status: 'ok', timestamp: '2026-03-26T00:00:00Z' }),
  ),
];

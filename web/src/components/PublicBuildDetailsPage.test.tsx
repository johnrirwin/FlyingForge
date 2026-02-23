import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from '../test/test-utils';
import { PublicBuildDetailsPage } from './PublicBuildDetailsPage';
import type { Build } from '../buildTypes';
import type { GearCatalogItem } from '../gearCatalogTypes';

vi.mock('../buildApi', () => ({
  createDraftBuild: vi.fn(),
  getPublicBuild: vi.fn(),
  createTempBuild: vi.fn(),
}));

vi.mock('../gearCatalogApi', () => ({
  getGearCatalogItem: vi.fn(),
}));

vi.mock('../hooks/useAuth', () => ({
  useAuth: vi.fn(),
}));

vi.mock('../buildShare', async () => {
  const actual = await vi.importActual<typeof import('../buildShare')>('../buildShare');
  return {
    ...actual,
    copyURLToClipboard: vi.fn(),
  };
});

import { createDraftBuild, createTempBuild, getPublicBuild } from '../buildApi';
import { copyURLToClipboard } from '../buildShare';
import { getGearCatalogItem } from '../gearCatalogApi';
import { useAuth } from '../hooks/useAuth';

const mockedCreateDraftBuild = vi.mocked(createDraftBuild);
const mockedCreateTempBuild = vi.mocked(createTempBuild);
const mockedCopyURLToClipboard = vi.mocked(copyURLToClipboard);
const mockedGetPublicBuild = vi.mocked(getPublicBuild);
const mockedGetGearCatalogItem = vi.mocked(getGearCatalogItem);
const mockedUseAuth = vi.mocked(useAuth);

function createDeferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

function mockAuth(isAuthenticated: boolean) {
  mockedUseAuth.mockReturnValue({
    isAuthenticated,
    user: null,
    isLoading: false,
    tokens: null,
    error: null,
    loginWithGoogle: vi.fn(),
    logout: vi.fn(),
    updateUser: vi.fn(),
    clearError: vi.fn(),
  });
}

function buildCatalogItem(overrides: Partial<GearCatalogItem> = {}): GearCatalogItem {
  return {
    id: 'catalog-item-1',
    gearType: 'frame',
    brand: 'Kayou',
    model: 'Kayoumini',
    variant: '2.5 inch',
    status: 'published',
    source: 'admin',
    canonicalKey: 'frame:kayou:kayoumini:2.5-inch',
    usageCount: 11,
    imageStatus: 'approved',
    descriptionStatus: 'approved',
    createdAt: '2026-02-01T00:00:00Z',
    updatedAt: '2026-02-01T00:00:00Z',
    ...overrides,
  };
}

function buildFixture(): Build {
  return {
    id: 'build-1',
    status: 'PUBLISHED',
    title: 'Micro Racer',
    description: 'Fast and lightweight.',
    createdAt: '2026-02-01T00:00:00Z',
    updatedAt: '2026-02-01T00:00:00Z',
    verified: true,
    parts: [
      {
        id: 'part-1',
        buildId: 'build-1',
        gearType: 'frame',
        catalogItemId: 'frame-1',
        catalogItem: {
          id: 'frame-1',
          gearType: 'frame',
          brand: 'Kayou',
          model: 'Kayoumini',
          variant: '2.5 inch',
          status: 'published',
        },
      },
      {
        id: 'part-2',
        buildId: 'build-1',
        gearType: 'motor',
        catalogItemId: 'motor-1',
        catalogItem: {
          id: 'motor-1',
          gearType: 'motor',
          brand: 'BetaFPV',
          model: '1103',
          variant: '11000KV',
          status: 'published',
        },
      },
    ],
  };
}

describe('PublicBuildDetailsPage', () => {
  beforeEach(() => {
    mockAuth(false);
    mockedCreateDraftBuild.mockReset();
    mockedCreateDraftBuild.mockResolvedValue({
      id: 'draft-copy-1',
      status: 'DRAFT',
      title: 'Micro Racer Copy',
      description: 'Fast and lightweight.',
      createdAt: '2026-02-01T00:00:00Z',
      updatedAt: '2026-02-01T00:00:00Z',
      verified: false,
      parts: [],
    });
    mockedCreateTempBuild.mockReset();
    mockedCreateTempBuild.mockResolvedValue({
      token: 'temp-token-1',
      url: '/builds/temp/temp-token-1',
      build: {
        id: 'temp-build-1',
        status: 'TEMP',
        title: 'Micro Racer Copy',
        description: 'Fast and lightweight.',
        createdAt: '2026-02-01T00:00:00Z',
        updatedAt: '2026-02-01T00:00:00Z',
        verified: false,
        parts: [],
      },
    });
    mockedCopyURLToClipboard.mockReset();
    mockedCopyURLToClipboard.mockResolvedValue(undefined);
    mockedGetPublicBuild.mockResolvedValue(buildFixture());
    mockedGetGearCatalogItem.mockImplementation(async (id: string) => {
      if (id === 'frame-1') {
        return buildCatalogItem({
          id: 'frame-1',
          gearType: 'frame',
          msrp: 89.99,
          shoppingLinks: ['https://www.racedayquads.com/products/kayou-frame'],
        });
      }

      if (id === 'motor-1') {
        return buildCatalogItem({
          id: 'motor-1',
          gearType: 'motor',
          brand: 'BetaFPV',
          model: '1103',
          variant: '11000KV',
          canonicalKey: 'motor:betafpv:1103:11000kv',
          msrp: 25,
        });
      }

      throw new Error(`Unexpected catalog id: ${id}`);
    });
  });

  function renderPage(onAddToInventory?: (item: GearCatalogItem) => void) {
    render(
      <MemoryRouter initialEntries={['/builds/build-1']}>
        <Routes>
          <Route
            path="/builds/:id"
            element={<PublicBuildDetailsPage onAddToInventory={onAddToInventory} />}
          />
          <Route path="/me/builds" element={<div>My Builds</div>} />
        </Routes>
      </MemoryRouter>,
    );
  }

  it('shows an estimated MSRP total from build components', async () => {
    renderPage();

    await waitFor(() => {
      expect(mockedGetPublicBuild).toHaveBeenCalledWith('build-1');
    });

    await waitFor(() => {
      expect(mockedGetGearCatalogItem).toHaveBeenCalledWith('frame-1');
      expect(mockedGetGearCatalogItem).toHaveBeenCalledWith('motor-1');
    });

    expect(await screen.findByText('Estimated MSRP')).toBeInTheDocument();
    expect(screen.getByText('$114.99')).toBeInTheDocument();
  });

  it('opens gear details from a part row and allows adding to inventory', async () => {
    mockAuth(true);
    const onAddToInventory = vi.fn();
    const user = userEvent.setup();

    renderPage(onAddToInventory);

    const frameButton = await screen.findByRole('button', {
      name: /view details for kayou kayoumini 2\.5 inch/i,
    });
    await user.click(frameButton);

    expect(await screen.findByRole('dialog')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /racedayquads\.com/i })).toHaveAttribute(
      'href',
      'https://www.racedayquads.com/products/kayou-frame',
    );

    await user.click(screen.getByRole('button', { name: /add to my inventory/i }));
    expect(onAddToInventory).toHaveBeenCalledTimes(1);
    expect(onAddToInventory).toHaveBeenCalledWith(expect.objectContaining({ id: 'frame-1' }));
  });

  it('shows loading state for whitespace-padded catalog ids', async () => {
    const user = userEvent.setup();
    const deferred = createDeferred<GearCatalogItem>();

    mockedGetPublicBuild.mockResolvedValue({
      ...buildFixture(),
      parts: [
        {
          id: 'part-1',
          buildId: 'build-1',
          gearType: 'frame',
          catalogItemId: ' frame-1 ',
          catalogItem: {
            id: 'frame-1',
            gearType: 'frame',
            brand: 'Kayou',
            model: 'Kayoumini',
            variant: '2.5 inch',
            status: 'published',
          },
        },
      ],
    });

    mockedGetGearCatalogItem.mockImplementation((id: string) => {
      if (id === 'frame-1') {
        return deferred.promise;
      }
      throw new Error(`Unexpected catalog id: ${id}`);
    });

    renderPage();

    const frameButton = await screen.findByRole('button', {
      name: /view details for kayou kayoumini 2\.5 inch/i,
    });

    await user.click(frameButton);

    expect(mockedGetGearCatalogItem).toHaveBeenCalledWith('frame-1');
    expect(screen.getByText('Loading details...')).toBeInTheDocument();

    deferred.resolve(buildCatalogItem({
      id: 'frame-1',
      gearType: 'frame',
      msrp: 89.99,
    }));

    expect(await screen.findByRole('dialog')).toBeInTheDocument();
  });

  it('creates a prefilled draft when authenticated users click Build Your Own', async () => {
    mockAuth(true);
    const user = userEvent.setup();
    renderPage();

    await waitFor(() => {
      expect(mockedGetPublicBuild).toHaveBeenCalledWith('build-1');
    });

    await user.click(await screen.findByRole('button', { name: /build your own/i }));

    expect(mockedCreateDraftBuild).toHaveBeenCalledWith({
      title: 'Micro Racer Copy',
      description: 'Fast and lightweight.',
      parts: [
        {
          gearType: 'frame',
          catalogItemId: 'frame-1',
          notes: undefined,
          position: undefined,
        },
        {
          gearType: 'motor',
          catalogItemId: 'motor-1',
          notes: undefined,
          position: undefined,
        },
      ],
    });
    expect(mockedCreateTempBuild).not.toHaveBeenCalled();
  });

  it('copies published build URL', async () => {
    const user = userEvent.setup();

    renderPage();

    await waitFor(() => {
      expect(mockedGetPublicBuild).toHaveBeenCalledWith('build-1');
    });

    await user.click(await screen.findByRole('button', { name: /copy build url/i }));

    expect(mockedCopyURLToClipboard).toHaveBeenCalledTimes(1);
    expect(mockedCopyURLToClipboard.mock.calls[0][0]).toContain('/builds/build-1');
    expect(await screen.findByText('Build URL copied')).toBeInTheDocument();
  });
});

import { describe, expect, it, vi, beforeEach } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { fireEvent, render, screen, waitFor } from '../test/test-utils';
import { PublicBuildsPage } from './PublicBuildsPage';
import type { Build } from '../buildTypes';

vi.mock('../buildApi', () => ({
  listPublicBuilds: vi.fn(),
  createTempBuild: vi.fn(),
  setBuildReaction: vi.fn(),
  clearBuildReaction: vi.fn(),
}));

vi.mock('../hooks/useAuth', () => ({
  useAuth: vi.fn(),
}));

import { listPublicBuilds, setBuildReaction } from '../buildApi';
import { clearBuildReaction } from '../buildApi';
import { useAuth } from '../hooks/useAuth';

const mockedListPublicBuilds = vi.mocked(listPublicBuilds);
const mockedSetBuildReaction = vi.mocked(setBuildReaction);
const mockedClearBuildReaction = vi.mocked(clearBuildReaction);
const mockedUseAuth = vi.mocked(useAuth);

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

function makeBuild(overrides: Partial<Build> = {}): Build {
  return {
    id: 'build-1',
    status: 'PUBLISHED',
    title: 'Race Rig',
    description: '',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    parts: [],
    verified: false,
    likeCount: 2,
    dislikeCount: 1,
    ...overrides,
  };
}

describe('PublicBuildsPage', () => {
  beforeEach(() => {
    mockAuth(false);
    mockedListPublicBuilds.mockResolvedValue({
      builds: [],
      totalCount: 0,
      sort: 'newest',
    });
    mockedSetBuildReaction.mockReset();
    mockedClearBuildReaction.mockReset();
  });

  it('renders without crashing and loads public builds', async () => {
    render(
      <MemoryRouter>
        <PublicBuildsPage />
      </MemoryRouter>,
    );

    expect(screen.getByText('Public Builds')).toBeInTheDocument();

    await waitFor(() => {
      expect(mockedListPublicBuilds).toHaveBeenCalled();
    });

    expect(screen.getByText(/No public builds/i)).toBeInTheDocument();
  });

  it('shows estimated MSRP on build cards', async () => {
    mockedListPublicBuilds.mockResolvedValue({
      builds: [
        makeBuild({
          parts: [
            {
              gearType: 'frame',
              catalogItemId: 'frame-1',
              catalogItem: {
                id: 'frame-1',
                gearType: 'frame',
                brand: 'Kayou',
                model: 'Kayoumini',
                status: 'published',
                msrp: 89.99,
              },
            },
            {
              gearType: 'motor',
              catalogItemId: 'motor-1',
              catalogItem: {
                id: 'motor-1',
                gearType: 'motor',
                brand: 'BetaFPV',
                model: '1103',
                status: 'published',
                msrp: 25,
              },
            },
          ],
        }),
      ],
      totalCount: 1,
      sort: 'newest',
    });

    render(
      <MemoryRouter>
        <PublicBuildsPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText('Est. MSRP: $114.99')).toBeInTheDocument();
  });

  it('shows a plus suffix when some listed part prices are missing', async () => {
    mockedListPublicBuilds.mockResolvedValue({
      builds: [
        makeBuild({
          parts: [
            {
              gearType: 'frame',
              catalogItemId: 'frame-1',
              catalogItem: {
                id: 'frame-1',
                gearType: 'frame',
                brand: 'Kayou',
                model: 'Kayoumini',
                status: 'published',
                msrp: 89.99,
              },
            },
            {
              gearType: 'vtx',
              catalogItemId: 'vtx-1',
              catalogItem: {
                id: 'vtx-1',
                gearType: 'vtx',
                brand: 'DJI',
                model: 'O4',
                status: 'published',
              },
            },
          ],
        }),
      ],
      totalCount: 1,
      sort: 'newest',
    });

    render(
      <MemoryRouter>
        <PublicBuildsPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText('Est. MSRP: $89.99+')).toBeInTheDocument();
  });

  it('allows authenticated users to like a build card', async () => {
    mockAuth(true);

    mockedListPublicBuilds.mockResolvedValue({
      builds: [makeBuild()],
      totalCount: 1,
      sort: 'newest',
    });

    mockedSetBuildReaction.mockResolvedValue(makeBuild({
      likeCount: 3,
      dislikeCount: 1,
      viewerReaction: 'LIKE',
    }));

    render(
      <MemoryRouter>
        <PublicBuildsPage />
      </MemoryRouter>,
    );

    const likeButton = await screen.findByRole('button', { name: /^Like Race Rig$/i });
    fireEvent.click(likeButton);

    await waitFor(() => {
      expect(mockedSetBuildReaction).toHaveBeenCalledWith('build-1', 'LIKE');
    });
  });

  it('allows authenticated users to dislike a build card', async () => {
    mockAuth(true);
    mockedListPublicBuilds.mockResolvedValue({
      builds: [makeBuild()],
      totalCount: 1,
      sort: 'newest',
    });
    mockedSetBuildReaction.mockResolvedValue(makeBuild({
      likeCount: 2,
      dislikeCount: 2,
      viewerReaction: 'DISLIKE',
    }));

    render(
      <MemoryRouter>
        <PublicBuildsPage />
      </MemoryRouter>,
    );

    const dislikeButton = await screen.findByRole('button', { name: /^Dislike Race Rig$/i });
    fireEvent.click(dislikeButton);

    await waitFor(() => {
      expect(mockedSetBuildReaction).toHaveBeenCalledWith('build-1', 'DISLIKE');
    });
  });

  it('toggles reaction off when clicking the same reaction again', async () => {
    mockAuth(true);
    mockedListPublicBuilds.mockResolvedValue({
      builds: [makeBuild({ likeCount: 3, viewerReaction: 'LIKE' })],
      totalCount: 1,
      sort: 'newest',
    });
    mockedClearBuildReaction.mockResolvedValue(makeBuild({
      likeCount: 2,
      dislikeCount: 1,
      viewerReaction: undefined,
    }));

    render(
      <MemoryRouter>
        <PublicBuildsPage />
      </MemoryRouter>,
    );

    const likeButton = await screen.findByRole('button', { name: /^Like Race Rig$/i });
    fireEvent.click(likeButton);

    await waitFor(() => {
      expect(mockedClearBuildReaction).toHaveBeenCalledWith('build-1');
    });
  });

  it('shows sign-in message when unauthenticated users try to react', async () => {
    mockedListPublicBuilds.mockResolvedValue({
      builds: [makeBuild()],
      totalCount: 1,
      sort: 'newest',
    });

    render(
      <MemoryRouter>
        <PublicBuildsPage />
      </MemoryRouter>,
    );

    const likeButton = await screen.findByRole('button', { name: /^Like Race Rig$/i });
    fireEvent.click(likeButton);

    expect(screen.getByText('Sign in to like or dislike builds.')).toBeInTheDocument();
    expect(mockedSetBuildReaction).not.toHaveBeenCalled();
    expect(mockedClearBuildReaction).not.toHaveBeenCalled();
  });

  it('shows API error when saving a reaction fails', async () => {
    mockAuth(true);
    mockedListPublicBuilds.mockResolvedValue({
      builds: [makeBuild()],
      totalCount: 1,
      sort: 'newest',
    });
    mockedSetBuildReaction.mockRejectedValue(new Error('Could not save reaction'));

    render(
      <MemoryRouter>
        <PublicBuildsPage />
      </MemoryRouter>,
    );

    const likeButton = await screen.findByRole('button', { name: /^Like Race Rig$/i });
    fireEvent.click(likeButton);

    await waitFor(() => {
      expect(screen.getByText('Could not save reaction')).toBeInTheDocument();
    });
  });

  it('renders multiple builds with their own reaction states', async () => {
    mockAuth(true);
    mockedListPublicBuilds.mockResolvedValue({
      builds: [
        makeBuild({
          id: 'build-1',
          title: 'Race Rig',
          likeCount: 5,
          dislikeCount: 1,
          viewerReaction: 'LIKE',
        }),
        makeBuild({
          id: 'build-2',
          title: 'Freestyle Rig',
          likeCount: 2,
          dislikeCount: 4,
          viewerReaction: 'DISLIKE',
        }),
      ],
      totalCount: 2,
      sort: 'newest',
    });

    render(
      <MemoryRouter>
        <PublicBuildsPage />
      </MemoryRouter>,
    );

    const raceLike = await screen.findByRole('button', { name: /^Like Race Rig$/i });
    const freestyleDislike = await screen.findByRole('button', { name: /^Dislike Freestyle Rig$/i });

    expect(raceLike.className).toContain('border-emerald-400/60');
    expect(freestyleDislike.className).toContain('border-rose-400/60');
  });
});

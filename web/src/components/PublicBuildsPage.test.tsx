import { describe, expect, it, vi, beforeEach } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { fireEvent, render, screen, waitFor } from '../test/test-utils';
import { PublicBuildsPage } from './PublicBuildsPage';

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
import { useAuth } from '../hooks/useAuth';

const mockedListPublicBuilds = vi.mocked(listPublicBuilds);
const mockedSetBuildReaction = vi.mocked(setBuildReaction);
const mockedUseAuth = vi.mocked(useAuth);

describe('PublicBuildsPage', () => {
  beforeEach(() => {
    mockedUseAuth.mockReturnValue({
      isAuthenticated: false,
      user: null,
      isLoading: false,
      tokens: null,
      error: null,
      loginWithGoogle: vi.fn(),
      logout: vi.fn(),
      updateUser: vi.fn(),
      clearError: vi.fn(),
    });
    mockedListPublicBuilds.mockResolvedValue({
      builds: [],
      totalCount: 0,
      sort: 'newest',
    });
    mockedSetBuildReaction.mockReset();
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

  it('allows authenticated users to like a build card', async () => {
    mockedUseAuth.mockReturnValue({
      isAuthenticated: true,
      user: null,
      isLoading: false,
      tokens: null,
      error: null,
      loginWithGoogle: vi.fn(),
      logout: vi.fn(),
      updateUser: vi.fn(),
      clearError: vi.fn(),
    });

    mockedListPublicBuilds.mockResolvedValue({
      builds: [{
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
      }],
      totalCount: 1,
      sort: 'newest',
    });

    mockedSetBuildReaction.mockResolvedValue({
      id: 'build-1',
      status: 'PUBLISHED',
      title: 'Race Rig',
      description: '',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      parts: [],
      verified: false,
      likeCount: 3,
      dislikeCount: 1,
      viewerReaction: 'LIKE',
    });

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
});

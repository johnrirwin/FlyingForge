import { describe, expect, it, vi, beforeEach } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { render, screen, waitFor } from '../test/test-utils';
import { PublicBuildsPage } from './PublicBuildsPage';

vi.mock('../buildApi', () => ({
  listPublicBuilds: vi.fn(),
  createTempBuild: vi.fn(),
}));

vi.mock('../hooks/useAuth', () => ({
  useAuth: vi.fn(),
}));

import { listPublicBuilds } from '../buildApi';
import { useAuth } from '../hooks/useAuth';

const mockedListPublicBuilds = vi.mocked(listPublicBuilds);
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
});

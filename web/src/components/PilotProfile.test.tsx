import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import { render, screen, waitFor } from '../test/test-utils';
import { PilotProfile } from './PilotProfile';
import type { PilotProfile as PilotProfileType } from '../socialTypes';

vi.mock('../pilotApi', () => ({
  getPilotProfile: vi.fn(),
}));

vi.mock('../socialApi', () => {
  class MockApiError extends Error {
    code?: string;
  }

  return {
    followPilot: vi.fn(),
    unfollowPilot: vi.fn(),
    ApiError: MockApiError,
  };
});

vi.mock('../profileApi', () => ({
  updateProfile: vi.fn(),
}));

vi.mock('../hooks/useAuth', () => ({
  useAuth: vi.fn(),
}));

vi.mock('./PublicAircraftModal', () => ({
  PublicAircraftModal: () => null,
}));

vi.mock('./FollowListModal', () => ({
  FollowListModal: () => null,
}));

vi.mock('./SocialPage', () => ({
  CallSignPromptModal: () => null,
}));

import { getPilotProfile } from '../pilotApi';
import { useAuth } from '../hooks/useAuth';

const mockedGetPilotProfile = vi.mocked(getPilotProfile);
const mockedUseAuth = vi.mocked(useAuth);

function mockAuth() {
  mockedUseAuth.mockReturnValue({
    isAuthenticated: true,
    isLoading: false,
    user: {
      id: 'viewer-1',
      email: 'viewer@example.com',
      callSign: 'Viewer',
      displayName: 'Viewer',
      googleName: 'Viewer',
      status: 'active',
      emailVerified: true,
      isAdmin: false,
      isContentAdmin: false,
      avatarType: 'google',
      createdAt: '2026-02-01T00:00:00Z',
    },
    tokens: null,
    error: null,
    loginWithGoogle: vi.fn(),
    logout: vi.fn(),
    updateUser: vi.fn(),
    clearError: vi.fn(),
  });
}

function profileFixture(overrides: Partial<PilotProfileType> = {}): PilotProfileType {
  return {
    id: 'pilot-1',
    callSign: 'UmbraVenti',
    displayName: 'Umbra Venti',
    googleName: 'Umbra Venti',
    effectiveAvatarUrl: '',
    createdAt: '2026-01-01T00:00:00Z',
    aircraft: [],
    publishedBuilds: [
      {
        id: 'build-1',
        status: 'PUBLISHED',
        title: 'Kayou Mini Build',
        description: 'Compact freestyle setup.',
        createdAt: '2026-02-01T00:00:00Z',
        updatedAt: '2026-02-01T00:00:00Z',
        publishedAt: '2026-02-02T00:00:00Z',
        verified: true,
        parts: [],
      },
    ],
    isFollowing: false,
    followerCount: 12,
    followingCount: 8,
    ...overrides,
  };
}

function renderProfile() {
  render(
    <MemoryRouter>
      <PilotProfile pilotId="pilot-1" onBack={vi.fn()} />
    </MemoryRouter>,
  );
}

describe('PilotProfile', () => {
  beforeEach(() => {
    mockAuth();
    mockedGetPilotProfile.mockReset();
  });

  it('shows published builds with links to build pages', async () => {
    mockedGetPilotProfile.mockResolvedValue(profileFixture());

    renderProfile();

    await waitFor(() => {
      expect(mockedGetPilotProfile).toHaveBeenCalledWith('pilot-1');
    });

    expect(await screen.findByText('Published Builds')).toBeInTheDocument();
    expect(screen.getByText('Kayou Mini Build')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /Kayou Mini Build/i })).toHaveAttribute('href', '/builds/build-1');
  });

  it('shows an empty published builds state when none exist', async () => {
    mockedGetPilotProfile.mockResolvedValue(profileFixture({ publishedBuilds: [] }));

    renderProfile();

    expect(await screen.findByText('No published builds yet')).toBeInTheDocument();
    expect(screen.getByText('0 builds')).toBeInTheDocument();
  });
});

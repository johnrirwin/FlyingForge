import { beforeEach, describe, expect, it, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { render, screen, waitFor } from '../test/test-utils';
import { PilotProfile } from './PilotProfile';
import type { PilotProfile as PilotProfileType } from '../socialTypes';

vi.mock('../pilotApi', () => ({
  getPilotProfile: vi.fn(),
}));

vi.mock('../buildApi', () => ({
  getPublicBuild: vi.fn(),
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
import { getPublicBuild } from '../buildApi';
import { useAuth } from '../hooks/useAuth';

const mockedGetPilotProfile = vi.mocked(getPilotProfile);
const mockedGetPublicBuild = vi.mocked(getPublicBuild);
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
    mockedGetPublicBuild.mockReset();
    mockedGetPublicBuild.mockResolvedValue({
      id: 'build-1',
      status: 'PUBLISHED',
      title: 'Kayou Mini Build',
      description: 'Compact freestyle setup.',
      createdAt: '2026-02-01T00:00:00Z',
      updatedAt: '2026-02-01T00:00:00Z',
      publishedAt: '2026-02-02T00:00:00Z',
      verified: true,
      parts: [],
    });
  });

  it('opens published build details in a modal and links to build page', async () => {
    mockedGetPilotProfile.mockResolvedValue(profileFixture());

    renderProfile();

    await waitFor(() => {
      expect(mockedGetPilotProfile).toHaveBeenCalledWith('pilot-1');
    });

    expect(await screen.findByText('Published Builds')).toBeInTheDocument();
    expect(screen.getByText('Kayou Mini Build')).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: /Kayou Mini Build/i }));

    await waitFor(() => {
      expect(mockedGetPublicBuild).toHaveBeenCalledWith('build-1');
    });

    expect(await screen.findByRole('heading', { name: 'Build Details' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Open Build Page' })).toHaveAttribute('href', '/builds/build-1');
  });

  it('embeds build and flight YouTube players in build details when video URLs exist', async () => {
    mockedGetPilotProfile.mockResolvedValue(profileFixture());
    mockedGetPublicBuild.mockResolvedValue({
      id: 'build-1',
      status: 'PUBLISHED',
      title: 'Kayou Mini Build',
      description: 'Compact freestyle setup.',
      youtubeUrl: 'https://youtu.be/demo123',
      flightYoutubeUrl: 'https://www.youtube.com/watch?v=flightxyz',
      createdAt: '2026-02-01T00:00:00Z',
      updatedAt: '2026-02-01T00:00:00Z',
      publishedAt: '2026-02-02T00:00:00Z',
      verified: true,
      parts: [],
    });

    renderProfile();

    await waitFor(() => {
      expect(mockedGetPilotProfile).toHaveBeenCalledWith('pilot-1');
    });

    await userEvent.click(await screen.findByRole('button', { name: /Kayou Mini Build/i }));

    await waitFor(() => {
      expect(mockedGetPublicBuild).toHaveBeenCalledWith('build-1');
    });

    const buildIframe = await screen.findByTitle(/kayou mini build.*build video/i);
    expect(buildIframe).toHaveAttribute('src', 'https://www.youtube.com/embed/demo123?rel=0');

    const flightIframe = await screen.findByTitle(/kayou mini build.*flight video/i);
    expect(flightIframe).toHaveAttribute('src', 'https://www.youtube.com/embed/flightxyz?rel=0');
  });

  it('shows an empty published builds state when none exist', async () => {
    mockedGetPilotProfile.mockResolvedValue(profileFixture({ publishedBuilds: [] }));

    renderProfile();

    expect(await screen.findByText('No published builds yet')).toBeInTheDocument();
    expect(screen.getByText('0 builds')).toBeInTheDocument();
  });

  it('deduplicates reused catalog items when calculating MSRP in build preview', async () => {
    mockedGetPilotProfile.mockResolvedValue(profileFixture());
    mockedGetPublicBuild.mockResolvedValue({
      id: 'build-1',
      status: 'PUBLISHED',
      title: 'Kayou Mini Build',
      description: 'Compact freestyle setup.',
      createdAt: '2026-02-01T00:00:00Z',
      updatedAt: '2026-02-01T00:00:00Z',
      publishedAt: '2026-02-02T00:00:00Z',
      verified: true,
      parts: [
        {
          gearType: 'aio',
          catalogItemId: 'aio-1',
          catalogItem: {
            id: 'aio-1',
            gearType: 'aio',
            brand: 'SpeedyBee',
            model: 'F405 AIO',
            status: 'published',
            msrp: 79.99,
          },
        },
        {
          gearType: 'receiver',
          catalogItemId: 'aio-1',
          catalogItem: {
            id: 'aio-1',
            gearType: 'aio',
            brand: 'SpeedyBee',
            model: 'F405 AIO',
            status: 'published',
            msrp: 79.99,
          },
        },
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
      ],
    });

    renderProfile();

    await waitFor(() => {
      expect(mockedGetPilotProfile).toHaveBeenCalledWith('pilot-1');
    });

    await userEvent.click(await screen.findByRole('button', { name: /Kayou Mini Build/i }));

    expect(await screen.findByText('Estimated MSRP')).toBeInTheDocument();
    expect(screen.getByText('$169.98')).toBeInTheDocument();
  });
});

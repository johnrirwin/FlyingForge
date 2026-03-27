/**
 * Integration tests: Social / Pilots page
 * Covers:
 *   - Social page loads without crashing (scenario 18)
 *   - Clicking a pilot card fires onSelectPilot callback (scenario 18)
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import userEvent from '@testing-library/user-event';
import { render, screen, waitFor } from './test-utils';
import { SocialPage } from '../components/SocialPage';
import type { PilotSummaryWithFollowers, PilotProfile, FeaturedPilotsResponse, PilotSearchResponse, FollowListResponse } from '../socialTypes';

vi.mock('../hooks/useAuth', () => ({
  useAuth: vi.fn(),
}));

vi.mock('../pilotApi', () => ({
  searchPilots: vi.fn(),
  getPilotProfile: vi.fn(),
  discoverPilots: vi.fn(),
  getPublicPilotProfile: vi.fn(),
}));

vi.mock('../socialApi', () => ({
  getFollowing: vi.fn(),
  getFollowers: vi.fn(),
  followPilot: vi.fn(),
  unfollowPilot: vi.fn(),
}));

vi.mock('../profileApi', () => ({
  updateProfile: vi.fn(),
  validateCallSign: vi.fn().mockReturnValue(null),
}));

vi.mock('../hooks/useGoogleAnalytics', () => ({
  useGoogleAnalytics: vi.fn(),
  trackEvent: vi.fn(),
}));

import { useAuth } from '../hooks/useAuth';
import { discoverPilots, searchPilots, getPilotProfile } from '../pilotApi';
import { getFollowing, getFollowers } from '../socialApi';

const mockedUseAuth = vi.mocked(useAuth);
const mockedDiscoverPilots = vi.mocked(discoverPilots);
const mockedSearchPilots = vi.mocked(searchPilots);
const mockedGetPilotProfile = vi.mocked(getPilotProfile);
const mockedGetFollowing = vi.mocked(getFollowing);
const mockedGetFollowers = vi.mocked(getFollowers);

const pilotSummary: PilotSummaryWithFollowers = {
  id: 'pilot-1',
  callSign: 'SkyFox',
  displayName: 'Alex Fox',
  effectiveAvatarUrl: '',
  followerCount: 10,
};

const featuredResponse: FeaturedPilotsResponse = {
  popular: [pilotSummary],
  recent: [],
};

const emptySearchResponse: PilotSearchResponse = {
  pilots: [],
  total: 0,
};

const emptyFollowList: FollowListResponse = {
  pilots: [],
  totalCount: 0,
};

const mockProfile: PilotProfile = {
  id: 'user-1',
  callSign: 'TestPilot',
  displayName: 'Test Pilot',
  effectiveAvatarUrl: '',
  createdAt: '2026-01-01T00:00:00Z',
  aircraft: [],
  publishedBuilds: [],
  isFollowing: false,
  followerCount: 0,
  followingCount: 0,
};

function makeAuthValue() {
  return {
    isAuthenticated: true,
    user: {
      id: 'user-1',
      email: 'pilot@example.com',
      displayName: 'Test Pilot',
      callSign: 'TestPilot', // callSign required to skip CallSignPromptModal
      status: 'active' as const,
      emailVerified: true,
      isAdmin: false,
      isContentAdmin: false,
      isGearAdmin: false,
      createdAt: '2026-01-01T00:00:00Z',
    },
    tokens: null,
    isLoading: false,
    error: null,
    loginWithGoogle: vi.fn(),
    logout: vi.fn(),
    updateUser: vi.fn(),
    clearError: vi.fn(),
  };
}

describe('SocialPage – social/pilots integration', () => {
  beforeEach(() => {
    mockedUseAuth.mockReturnValue(makeAuthValue());
    mockedDiscoverPilots.mockResolvedValue(featuredResponse);
    mockedSearchPilots.mockResolvedValue(emptySearchResponse);
    mockedGetPilotProfile.mockResolvedValue(mockProfile);
    mockedGetFollowing.mockResolvedValue(emptyFollowList);
    mockedGetFollowers.mockResolvedValue(emptyFollowList);
  });

  it('renders the Social page with search input', async () => {
    render(<SocialPage onSelectPilot={vi.fn()} />);

    // SocialPage renders a search input with this placeholder
    await waitFor(() => {
      expect(
        screen.getByPlaceholderText(/search by callsign or name/i),
      ).toBeInTheDocument();
    });
  });

  it('displays discovered popular pilots after load', async () => {
    render(<SocialPage onSelectPilot={vi.fn()} />);

    // DiscoveryPilotCard renders the pilot callSign
    await waitFor(() => {
      expect(screen.getByText('SkyFox')).toBeInTheDocument();
    });
  });

  it('calls onSelectPilot when a popular pilot card is clicked', async () => {
    const user = userEvent.setup();
    const onSelectPilot = vi.fn();

    render(<SocialPage onSelectPilot={onSelectPilot} />);

    // Wait for the pilot card to appear (DiscoveryPilotCard, a div)
    const pilotText = await screen.findByText('SkyFox');
    await user.click(pilotText);

    expect(onSelectPilot).toHaveBeenCalledWith('pilot-1');
  });

  it('shows Following and Followers tab buttons for authenticated user', async () => {
    render(<SocialPage onSelectPilot={vi.fn()} />);

    // Tab buttons contain span text "Following" and "Followers"
    await waitFor(() => {
      // Use getAllByRole to avoid ambiguity with profile section buttons
      const allButtons = screen.getAllByRole('button');
      const buttonTexts = allButtons.map((b) => b.textContent?.trim() ?? '');
      expect(buttonTexts.some((t) => /following/i.test(t))).toBe(true);
      expect(buttonTexts.some((t) => /followers/i.test(t))).toBe(true);
    });
  });
});

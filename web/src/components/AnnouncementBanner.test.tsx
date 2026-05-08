import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import type { Announcement } from '../announcementTypes';
import { getActiveAnnouncements } from '../announcementApi';
import { AnnouncementBanner, AnnouncementPlacementBanner } from './AnnouncementBanner';

vi.mock('../announcementApi', () => ({
  getActiveAnnouncements: vi.fn(),
}));

const mockGetActiveAnnouncements = vi.mocked(getActiveAnnouncements);

function createAnnouncement(overrides: Partial<Announcement> = {}): Announcement {
  return {
    id: 'announcement-1',
    title: 'MCP integrations are now available',
    body: 'Connect FlyingForge to ChatGPT.',
    status: 'published',
    priority: 100,
    placements: ['home'],
    audience: 'all',
    ctaLabel: 'Learn more',
    ctaUrl: '/news',
    dismissible: true,
    createdAt: '2026-05-07T12:00:00Z',
    updatedAt: '2026-05-07T12:00:00Z',
    ...overrides,
  };
}

describe('AnnouncementBanner', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.clearAllMocks();
  });

  it('renders announcement content and internal CTA link', () => {
    render(
      <MemoryRouter>
        <AnnouncementBanner announcement={createAnnouncement()} />
      </MemoryRouter>,
    );

    expect(screen.getByText('MCP integrations are now available')).toBeInTheDocument();
    expect(screen.getByText('Connect FlyingForge to ChatGPT.')).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /learn more/i })).toHaveAttribute('href', '/news');
  });

  it('renders external CTA links with noopener noreferrer', () => {
    render(
      <MemoryRouter>
        <AnnouncementBanner announcement={createAnnouncement({ ctaUrl: 'https://example.com/news' })} />
      </MemoryRouter>,
    );

    expect(screen.getByRole('link', { name: /learn more/i })).toHaveAttribute('rel', 'noopener noreferrer');
  });

  it('renders no button when the announcement has no call to action', () => {
    render(
      <MemoryRouter>
        <AnnouncementBanner announcement={createAnnouncement({ ctaLabel: undefined, ctaUrl: undefined })} />
      </MemoryRouter>,
    );

    expect(screen.queryByRole('link')).not.toBeInTheDocument();
  });

  it('loads active announcements for a placement and dismisses the current banner', async () => {
    const first = createAnnouncement();
    const second = createAnnouncement({
      id: 'announcement-2',
      title: 'New dashboard widgets',
      body: 'Your dashboard now shows recent announcements.',
      dismissible: false,
      updatedAt: '2026-05-08T12:00:00Z',
    });
    mockGetActiveAnnouncements.mockResolvedValue({
      announcements: [first, second],
      totalCount: 2,
    });

    render(
      <MemoryRouter>
        <AnnouncementPlacementBanner placement="home" />
      </MemoryRouter>,
    );

    expect(await screen.findByText(first.title)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Dismiss announcement' }));

    await waitFor(() => {
      expect(screen.getByText(second.title)).toBeInTheDocument();
    });
    expect(localStorage.getItem(`flyingforge:announcement:dismissed:${first.id}:${first.updatedAt}`)).toBe('true');
  });
});

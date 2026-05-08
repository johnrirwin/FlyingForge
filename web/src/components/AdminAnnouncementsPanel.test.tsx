import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { Announcement } from '../announcementTypes';
import {
  adminCreateAnnouncement,
  adminDeleteAnnouncement,
  adminListAnnouncements,
  adminUpdateAnnouncement,
} from '../adminApi';
import { AdminAnnouncementsPanel } from './AdminAnnouncementsPanel';

vi.mock('../adminApi', () => ({
  adminListAnnouncements: vi.fn(),
  adminCreateAnnouncement: vi.fn(),
  adminUpdateAnnouncement: vi.fn(),
  adminDeleteAnnouncement: vi.fn(),
}));

const mockAdminListAnnouncements = vi.mocked(adminListAnnouncements);
const mockAdminCreateAnnouncement = vi.mocked(adminCreateAnnouncement);
const mockAdminUpdateAnnouncement = vi.mocked(adminUpdateAnnouncement);
const mockAdminDeleteAnnouncement = vi.mocked(adminDeleteAnnouncement);

function createAnnouncement(overrides: Partial<Announcement> = {}): Announcement {
  return {
    id: 'announcement-1',
    title: 'ChatGPT MCP integrations are now available',
    body: 'Connect FlyingForge to ChatGPT.',
    status: 'published',
    priority: 100,
    placements: ['home', 'news'],
    audience: 'all',
    ctaLabel: 'Learn more',
    ctaUrl: '/news',
    dismissible: true,
    createdAt: '2026-05-07T12:00:00Z',
    updatedAt: '2026-05-07T12:00:00Z',
    ...overrides,
  };
}

describe('AdminAnnouncementsPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAdminListAnnouncements.mockResolvedValue({
      announcements: [createAnnouncement()],
      totalCount: 1,
    });
    mockAdminCreateAnnouncement.mockResolvedValue(createAnnouncement({ id: 'announcement-2', title: 'Created' }));
    mockAdminUpdateAnnouncement.mockResolvedValue(createAnnouncement({ title: 'Updated title' }));
    mockAdminDeleteAnnouncement.mockResolvedValue();
  });

  it('loads and renders announcement rows', async () => {
    render(<AdminAnnouncementsPanel />);

    expect(await screen.findAllByText('ChatGPT MCP integrations are now available')).toHaveLength(2);
    expect(mockAdminListAnnouncements).toHaveBeenCalledWith({
      query: undefined,
      status: undefined,
      limit: 100,
      offset: 0,
    });
  });

  it('creates a new announcement from the editor modal', async () => {
    render(<AdminAnnouncementsPanel />);
    await screen.findAllByText('ChatGPT MCP integrations are now available');

    fireEvent.click(screen.getByRole('button', { name: /new announcement/i }));

    expect(screen.getByLabelText('Enable Call to Action')).not.toBeChecked();
    expect(screen.queryByLabelText('Call to Action Label')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Call to Action Link')).not.toBeInTheDocument();

    fireEvent.change(screen.getByLabelText('Title'), { target: { value: 'New release' } });
    fireEvent.change(screen.getByLabelText('Body'), { target: { value: 'Body copy' } });
    fireEvent.click(screen.getByRole('button', { name: 'Create Announcement' }));

    await waitFor(() => {
      expect(mockAdminCreateAnnouncement).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'New release',
          body: 'Body copy',
          status: 'draft',
          placements: ['global'],
          audience: 'all',
          dismissible: true,
        }),
      );
    });
  });

  it('reveals call to action fields when enabled and saves the CTA values', async () => {
    render(<AdminAnnouncementsPanel />);
    await screen.findAllByText('ChatGPT MCP integrations are now available');

    fireEvent.click(screen.getByRole('button', { name: /new announcement/i }));
    fireEvent.click(screen.getByLabelText('Enable Call to Action'));

    fireEvent.change(screen.getByLabelText('Title'), { target: { value: 'New release' } });
    fireEvent.change(screen.getByLabelText('Body'), { target: { value: 'Body copy' } });
    fireEvent.change(screen.getByLabelText('Call to Action Label'), { target: { value: 'Learn more' } });
    fireEvent.change(screen.getByLabelText('Call to Action Link'), { target: { value: '/news' } });
    fireEvent.click(screen.getByRole('button', { name: 'Create Announcement' }));

    await waitFor(() => {
      expect(mockAdminCreateAnnouncement).toHaveBeenCalledWith(
        expect.objectContaining({
          ctaLabel: 'Learn more',
          ctaUrl: '/news',
        }),
      );
    });
  });

  it('updates an existing announcement from the editor modal', async () => {
    render(<AdminAnnouncementsPanel />);

    const row = await screen.findByRole('button', { name: 'Open editor for ChatGPT MCP integrations are now available' });
    fireEvent.click(row);

    expect(screen.getByLabelText('Enable Call to Action')).toBeChecked();
    expect(screen.getByLabelText('Call to Action Label')).toHaveValue('Learn more');
    expect(screen.getByLabelText('Call to Action Link')).toHaveValue('/news');

    fireEvent.change(screen.getByLabelText('Title'), { target: { value: 'Updated title' } });
    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }));

    await waitFor(() => {
      expect(mockAdminUpdateAnnouncement).toHaveBeenCalledWith(
        'announcement-1',
        expect.objectContaining({
          title: 'Updated title',
        }),
      );
    });
  });

  it('clears call to action values when disabled before saving', async () => {
    render(<AdminAnnouncementsPanel />);

    const row = await screen.findByRole('button', { name: 'Open editor for ChatGPT MCP integrations are now available' });
    fireEvent.click(row);

    fireEvent.click(screen.getByLabelText('Enable Call to Action'));

    expect(screen.queryByLabelText('Call to Action Label')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Call to Action Link')).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }));

    await waitFor(() => {
      expect(mockAdminUpdateAnnouncement).toHaveBeenCalled();
    });

    const [, params] = mockAdminUpdateAnnouncement.mock.calls[0];
    expect(params.ctaLabel).toBeUndefined();
    expect(params.ctaUrl).toBeUndefined();
  });

  it('uses an in-app delete confirmation modal before deleting', async () => {
    render(<AdminAnnouncementsPanel />);

    const row = await screen.findByRole('button', { name: 'Open editor for ChatGPT MCP integrations are now available' });
    fireEvent.click(row);

    fireEvent.click(screen.getByRole('button', { name: 'Delete' }));

    expect(screen.getByRole('dialog', { name: 'Delete Announcement?' })).toBeInTheDocument();
    expect(screen.getByText(/This will permanently delete/i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Delete Announcement' }));

    await waitFor(() => {
      expect(mockAdminDeleteAnnouncement).toHaveBeenCalledWith('announcement-1');
    });
  });
});

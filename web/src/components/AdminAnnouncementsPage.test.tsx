import { render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { AdminAnnouncementsPage } from './AdminAnnouncementsPage';

vi.mock('./AdminAnnouncementsPanel', () => ({
  AdminAnnouncementsPanel: () => <div>Announcements Panel</div>,
}));

describe('AdminAnnouncementsPage', () => {
  it('shows a loading state while auth is pending', () => {
    render(<AdminAnnouncementsPage hasContentAdminAccess={false} authLoading />);

    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('shows access denied when the user lacks content admin access', () => {
    render(<AdminAnnouncementsPage hasContentAdminAccess={false} authLoading={false} />);

    expect(screen.getByText('Access Denied')).toBeInTheDocument();
    expect(screen.queryByText('Announcements Panel')).not.toBeInTheDocument();
  });

  it('renders the dedicated announcements admin page for authorized users', () => {
    render(<AdminAnnouncementsPage hasContentAdminAccess authLoading={false} />);

    expect(screen.getByText('Announcements Panel')).toBeInTheDocument();
  });
});

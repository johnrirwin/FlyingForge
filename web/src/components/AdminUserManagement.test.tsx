import { describe, it, expect, vi, beforeEach } from 'vitest';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { render } from '../test/test-utils';
import { AdminUserManagement } from './AdminUserManagement';
import type { AdminUser } from '../adminUserTypes';
import {
  adminDeleteUser,
  adminDeleteUserAvatar,
  adminGetUser,
  adminSearchUsers,
  adminUpdateUser,
} from '../adminApi';

vi.mock('../adminApi', () => ({
  adminSearchUsers: vi.fn(),
  adminGetUser: vi.fn(),
  adminUpdateUser: vi.fn(),
  adminDeleteUserAvatar: vi.fn(),
  adminDeleteUser: vi.fn(),
}));

const mockAdminSearchUsers = vi.mocked(adminSearchUsers);
const mockAdminGetUser = vi.mocked(adminGetUser);
const mockAdminUpdateUser = vi.mocked(adminUpdateUser);
const mockAdminDeleteUserAvatar = vi.mocked(adminDeleteUserAvatar);
const mockAdminDeleteUser = vi.mocked(adminDeleteUser);

const mockUser: AdminUser = {
  id: 'user-1',
  callSign: 'Umbra',
  email: 'pilot@example.com',
  displayName: 'Pilot One',
  status: 'active',
  isAdmin: false,
  isGearAdmin: false,
  avatarUrl: 'https://example.com/avatar.jpg',
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
  lastLoginAt: '2026-01-02T00:00:00Z',
};

describe('AdminUserManagement', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAdminSearchUsers.mockResolvedValue({
      users: [mockUser],
      totalCount: 1,
    });
    mockAdminGetUser.mockResolvedValue(mockUser);
  });

  it('shows access denied for non-admin users', () => {
    render(<AdminUserManagement isAdmin={false} authLoading={false} />);
    expect(screen.getByText('Access Denied')).toBeInTheDocument();
    expect(mockAdminSearchUsers).not.toHaveBeenCalled();
  });

  it('renders row identity as callsign, email, then display name', async () => {
    render(<AdminUserManagement isAdmin currentUserId="admin-1" authLoading={false} />);
    const callSignNodes = await screen.findAllByText('Umbra');
    expect(callSignNodes.length).toBeGreaterThan(0);

    expect(screen.getAllByText('Umbra')[0]).toBeInTheDocument();
    expect(screen.getAllByText('pilot@example.com')[0]).toBeInTheDocument();
    expect(screen.getAllByText('Pilot One')[0]).toBeInTheDocument();
  });

  it('opens profile modal on row click and saves role changes', async () => {
    const updatedUser: AdminUser = {
      ...mockUser,
      status: 'disabled',
      isGearAdmin: true,
    };
    mockAdminUpdateUser.mockResolvedValue(updatedUser);

    render(<AdminUserManagement isAdmin currentUserId="admin-1" authLoading={false} />);
    const emailCell = (await screen.findAllByText('pilot@example.com'))[0];
    fireEvent.click(emailCell);

    expect(await screen.findByText('User Profile')).toBeInTheDocument();
    await waitFor(() => {
      expect(mockAdminGetUser).toHaveBeenCalledWith('user-1');
    });

    const statusSelect = screen.getByLabelText('Account Status');
    fireEvent.change(statusSelect, { target: { value: 'disabled' } });

    const gearAdminCheckbox = screen.getByLabelText('Gear Admin');
    fireEvent.click(gearAdminCheckbox);

    fireEvent.click(screen.getByText('Save Changes'));

    await waitFor(() => {
      expect(mockAdminUpdateUser).toHaveBeenCalledWith('user-1', {
        status: 'disabled',
        isAdmin: false,
        isGearAdmin: true,
      });
    });

    expect(await screen.findByText('User was updated')).toBeInTheDocument();
  });

  it('opens profile modal from keyboard interaction on desktop row', async () => {
    render(<AdminUserManagement isAdmin currentUserId="admin-1" authLoading={false} />);

    const openProfileRow = await screen.findByRole('button', { name: 'Open profile for Umbra' });
    fireEvent.keyDown(openProfileRow, { key: 'Enter' });

    expect(await screen.findByText('User Profile')).toBeInTheDocument();
    await waitFor(() => {
      expect(mockAdminGetUser).toHaveBeenCalledWith('user-1');
    });
  });

  it('removes a user avatar from the profile modal', async () => {
    const updatedWithoutAvatar: AdminUser = {
      ...mockUser,
      avatarUrl: undefined,
      googleAvatarUrl: undefined,
      customAvatarUrl: undefined,
    };
    mockAdminDeleteUserAvatar.mockResolvedValue(updatedWithoutAvatar);

    render(<AdminUserManagement isAdmin currentUserId="admin-1" authLoading={false} />);
    const emailCell = (await screen.findAllByText('pilot@example.com'))[0];
    fireEvent.click(emailCell);
    expect(await screen.findByText('User Profile')).toBeInTheDocument();

    fireEvent.click(screen.getByText('Remove Profile Picture'));
    expect(await screen.findByText('Remove Profile Picture?')).toBeInTheDocument();
    const confirmInput = screen.getByPlaceholderText("Type 'delete' to confirm");
    fireEvent.change(confirmInput, { target: { value: 'delete' } });
    fireEvent.click(screen.getByText('Remove Picture'));

    await waitFor(() => {
      expect(mockAdminDeleteUserAvatar).toHaveBeenCalledWith('user-1');
    });
  });

  it('uses profile modal delete action and confirmation modal before deleting user', async () => {
    mockAdminDeleteUser.mockResolvedValue();

    render(<AdminUserManagement isAdmin currentUserId="admin-1" authLoading={false} />);
    const emailCell = (await screen.findAllByText('pilot@example.com'))[0];
    fireEvent.click(emailCell);
    expect(await screen.findByText('User Profile')).toBeInTheDocument();

    fireEvent.click(screen.getByText('Delete User'));
    expect(screen.getByText('Delete User Account?')).toBeInTheDocument();

    const confirmInput = screen.getByPlaceholderText("Type 'delete' to confirm");
    fireEvent.change(confirmInput, { target: { value: 'delete' } });
    const deleteButtons = screen.getAllByRole('button', { name: 'Delete User' });
    fireEvent.click(deleteButtons[deleteButtons.length - 1]);

    await waitFor(() => {
      expect(mockAdminDeleteUser).toHaveBeenCalledWith('user-1');
    });
    expect(mockAdminSearchUsers).toHaveBeenCalledTimes(2); // initial load + refresh after delete
  });
});

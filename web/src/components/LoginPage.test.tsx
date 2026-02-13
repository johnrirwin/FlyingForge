import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { LoginPage } from './LoginPage';

const mockUseAuth = vi.fn();

vi.mock('../hooks/useAuth', () => ({
  useAuth: () => mockUseAuth(),
}));

describe('LoginPage', () => {
  beforeEach(() => {
    mockUseAuth.mockReset();
    sessionStorage.clear();
  });

  it('redirects authenticated users to the next path', async () => {
    mockUseAuth.mockReturnValue({
      isLoading: false,
      isAuthenticated: true,
      error: null,
    });

    render(
      <MemoryRouter initialEntries={['/login?next=%2Finventory']}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/inventory" element={<div>Inventory Target</div>} />
          <Route path="/dashboard" element={<div>Dashboard Target</div>} />
        </Routes>
      </MemoryRouter>,
    );

    expect(await screen.findByText('Inventory Target')).toBeInTheDocument();
  });

  it('redirects authenticated users to /dashboard when next is missing', async () => {
    mockUseAuth.mockReturnValue({
      isLoading: false,
      isAuthenticated: true,
      error: null,
    });

    render(
      <MemoryRouter initialEntries={['/login']}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/dashboard" element={<div>Dashboard Target</div>} />
        </Routes>
      </MemoryRouter>,
    );

    expect(await screen.findByText('Dashboard Target')).toBeInTheDocument();
  });

  it('shows expired-session messaging when reason=expired', () => {
    mockUseAuth.mockReturnValue({
      isLoading: false,
      isAuthenticated: false,
      error: null,
    });

    render(
      <MemoryRouter initialEntries={['/login?next=%2Finventory&reason=expired']}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
        </Routes>
      </MemoryRouter>,
    );

    expect(screen.getByText('Your session expired. Please sign in again to continue.')).toBeInTheDocument();
  });
});

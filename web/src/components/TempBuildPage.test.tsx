import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { render, screen, waitFor } from '../test/test-utils';
import { TempBuildPage } from './TempBuildPage';
import userEvent from '@testing-library/user-event';

vi.mock('../buildApi', () => ({
  getTempBuild: vi.fn(),
  updateTempBuild: vi.fn(),
  shareTempBuild: vi.fn(),
}));

import { getTempBuild, shareTempBuild } from '../buildApi';

const mockedGetTempBuild = vi.mocked(getTempBuild);
const mockedShareTempBuild = vi.mocked(shareTempBuild);

describe('TempBuildPage', () => {
  beforeEach(() => {
    mockedGetTempBuild.mockResolvedValue({
      id: 'temp-build-1',
      status: 'TEMP',
      title: 'Temporary Build',
      description: '',
      createdAt: '2026-02-11T00:00:00Z',
      updatedAt: '2026-02-11T00:00:00Z',
      expiresAt: '2026-02-12T00:00:00Z',
      parts: [],
      verified: false,
    });
    mockedShareTempBuild.mockResolvedValue({
      token: 'shared-token-1',
      url: '/builds/temp/shared-token-1',
      build: {
        id: 'shared-build-1',
        status: 'SHARED',
        title: 'Temporary Build',
        description: '',
        createdAt: '2026-02-11T00:00:00Z',
        updatedAt: '2026-02-11T00:00:00Z',
        parts: [],
        verified: false,
      },
    });
    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    });
  });

  it('renders temp build route without crashing', async () => {
    render(
      <MemoryRouter initialEntries={['/builds/temp/abc123']}>
        <Routes>
          <Route path="/builds/temp/:token" element={<TempBuildPage />} />
        </Routes>
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(mockedGetTempBuild).toHaveBeenCalledWith('abc123');
    });

    expect(screen.getByText('Temporary Build')).toBeInTheDocument();
    expect(screen.getByText(/expires on/i)).toBeInTheDocument();
  });

  it('copies a permanent shared URL with one button', async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter initialEntries={['/builds/temp/abc123']}>
        <Routes>
          <Route path="/builds/temp/:token" element={<TempBuildPage />} />
        </Routes>
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(mockedGetTempBuild).toHaveBeenCalledWith('abc123');
    });

    expect(screen.queryByRole('button', { name: /^share$/i })).not.toBeInTheDocument();

    const copyButton = await screen.findByRole('button', { name: /copy share url/i });
    await user.click(copyButton);

    await waitFor(() => {
      expect(mockedShareTempBuild).toHaveBeenCalledWith('abc123');
    });
  });
});

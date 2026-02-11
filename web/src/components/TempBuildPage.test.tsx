import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { render, screen, waitFor } from '../test/test-utils';
import { TempBuildPage } from './TempBuildPage';

vi.mock('../buildApi', () => ({
  getTempBuild: vi.fn(),
  updateTempBuild: vi.fn(),
}));

import { getTempBuild } from '../buildApi';

const mockedGetTempBuild = vi.mocked(getTempBuild);

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
});

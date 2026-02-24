import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { act, fireEvent, render, screen, waitFor } from '../test/test-utils';
import { MyBuildsPage } from './MyBuildsPage';
import type { Build } from '../buildTypes';

vi.mock('../buildApi', () => ({
  createTempBuild: vi.fn(),
  createBuildFromAircraft: vi.fn(),
  createDraftBuild: vi.fn(),
  deleteMyBuild: vi.fn(),
  getMyBuildImageUrl: vi.fn(),
  getMyBuild: vi.fn(),
  listMyBuilds: vi.fn(),
  moderateBuildImageUpload: vi.fn(),
  publishMyBuild: vi.fn(),
  saveBuildImageUpload: vi.fn(),
  unpublishMyBuild: vi.fn(),
  updateTempBuild: vi.fn(),
  updateMyBuild: vi.fn(),
}));

vi.mock('../aircraftApi', () => ({
  listAircraft: vi.fn(),
}));

vi.mock('../buildShare', async () => {
  const actual = await vi.importActual<typeof import('../buildShare')>('../buildShare');
  return {
    ...actual,
    copyURLToClipboard: vi.fn(),
  };
});

vi.mock('./BuildBuilder', () => ({
  BuildBuilder: ({ title, onTitleChange, onPartsChange }: {
    title: string;
    onTitleChange?: (value: string) => void;
    onPartsChange?: (parts: Build['parts']) => void;
  }) => (
    <div>
      <p data-testid="builder-title">{title}</p>
      <button type="button" onClick={() => onTitleChange?.('Updated Build')}>Change Title</button>
      <button
        type="button"
        onClick={() => onPartsChange?.([
          {
            gearType: 'frame',
            catalogItemId: 'frame-1',
          },
        ])}
      >
        Change Parts
      </button>
    </div>
  ),
}));

vi.mock('./ImageUploadModal', () => ({
  ImageUploadModal: () => null,
}));

import {
  createTempBuild,
  getMyBuild,
  getMyBuildImageUrl,
  listMyBuilds,
  updateTempBuild,
} from '../buildApi';
import { listAircraft } from '../aircraftApi';
import { copyURLToClipboard } from '../buildShare';

const mockedCreateTempBuild = vi.mocked(createTempBuild);
const mockedGetMyBuild = vi.mocked(getMyBuild);
const mockedGetMyBuildImageUrl = vi.mocked(getMyBuildImageUrl);
const mockedListMyBuilds = vi.mocked(listMyBuilds);
const mockedListAircraft = vi.mocked(listAircraft);
const mockedUpdateTempBuild = vi.mocked(updateTempBuild);
const mockedCopyURLToClipboard = vi.mocked(copyURLToClipboard);

function draftBuildFixture(overrides: Partial<Build> = {}): Build {
  return {
    id: 'build-1',
    status: 'DRAFT',
    title: 'Untitled Build',
    description: '',
    createdAt: '2026-02-24T00:00:00Z',
    updatedAt: '2026-02-24T00:00:00Z',
    verified: false,
    parts: [],
    ...overrides,
  };
}

function tempResponse(token: string, path?: string) {
  return {
    token,
    url: path || `/builds/temp/${token}`,
    build: {
      ...draftBuildFixture(),
      id: `temp-${token}`,
      status: 'TEMP' as const,
    },
  };
}

async function waitForAutoShareSync() {
  await act(async () => {
    await new Promise((resolve) => {
      setTimeout(resolve, 420);
    });
  });
}

function renderPage() {
  render(
    <MemoryRouter initialEntries={['/me/builds']}>
      <Routes>
        <Route path="/me/builds" element={<MyBuildsPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('MyBuildsPage share URL behavior', () => {
  beforeEach(() => {
    mockedListMyBuilds.mockReset();
    mockedGetMyBuild.mockReset();
    mockedListAircraft.mockReset();
    mockedCreateTempBuild.mockReset();
    mockedUpdateTempBuild.mockReset();
    mockedCopyURLToClipboard.mockReset();
    mockedGetMyBuildImageUrl.mockReset();

    mockedListMyBuilds.mockResolvedValue({
      builds: [draftBuildFixture()],
      totalCount: 1,
      sort: 'newest',
    });
    mockedGetMyBuild.mockResolvedValue(draftBuildFixture());
    mockedListAircraft.mockResolvedValue({ aircraft: [], totalCount: 0 });
    mockedCreateTempBuild.mockResolvedValue(tempResponse('temp-1'));
    mockedUpdateTempBuild.mockResolvedValue(tempResponse('temp-2'));
    mockedCopyURLToClipboard.mockResolvedValue(undefined);
    mockedGetMyBuildImageUrl.mockReturnValue('/api/builds/build-1/image');
  });

  it('auto-generates a share URL and updates it when build content changes', async () => {
    renderPage();

    await waitFor(() => {
      expect(mockedListMyBuilds).toHaveBeenCalled();
      expect(mockedGetMyBuild).toHaveBeenCalledWith('build-1');
    });

    await waitForAutoShareSync();

    await waitFor(() => {
      expect(mockedCreateTempBuild).toHaveBeenCalledWith({
        title: 'Untitled Build',
        description: '',
        sourceAircraftId: undefined,
        parts: [],
      });
    });

    expect(await screen.findByText(/builds\/temp\/temp-1/i)).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: /change title/i }));

    await waitForAutoShareSync();

    await waitFor(() => {
      expect(mockedUpdateTempBuild).toHaveBeenCalledWith('temp-1', {
        title: 'Updated Build',
        description: '',
        sourceAircraftId: undefined,
        parts: [],
      });
    });

    expect(await screen.findByText(/builds\/temp\/temp-2/i)).toBeInTheDocument();
  });

  it('falls back to createTempBuild and logs when updateTempBuild fails', async () => {
    const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
    mockedCreateTempBuild
      .mockResolvedValueOnce(tempResponse('temp-10'))
      .mockResolvedValueOnce(tempResponse('temp-11'));
    mockedUpdateTempBuild.mockRejectedValueOnce(new Error('update failed'));

    renderPage();

    await waitFor(() => {
      expect(mockedGetMyBuild).toHaveBeenCalledWith('build-1');
    });

    await waitForAutoShareSync();
    await waitFor(() => {
      expect(mockedCreateTempBuild).toHaveBeenCalledTimes(1);
    });

    fireEvent.click(screen.getByRole('button', { name: /change parts/i }));

    await waitForAutoShareSync();

    await waitFor(() => {
      expect(mockedUpdateTempBuild).toHaveBeenCalledWith('temp-10', {
        title: 'Untitled Build',
        description: '',
        sourceAircraftId: undefined,
        parts: [
          {
            gearType: 'frame',
            catalogItemId: 'frame-1',
            position: undefined,
            notes: undefined,
          },
        ],
      });
      expect(mockedCreateTempBuild).toHaveBeenCalledTimes(2);
    });

    expect(warnSpy).toHaveBeenCalledWith(
      'Failed to update temp build, falling back to createTempBuild',
      expect.any(Error),
    );
    expect(await screen.findByText(/builds\/temp\/temp-11/i)).toBeInTheDocument();

    warnSpy.mockRestore();
  });

  it('copies the currently generated share URL without creating a new one', async () => {
    renderPage();

    await waitFor(() => {
      expect(mockedGetMyBuild).toHaveBeenCalledWith('build-1');
    });

    await waitForAutoShareSync();
    await waitFor(() => {
      expect(mockedCreateTempBuild).toHaveBeenCalledTimes(1);
    });

    fireEvent.click(screen.getByRole('button', { name: /copy share url/i }));

    await waitFor(() => {
      expect(mockedCopyURLToClipboard).toHaveBeenCalledWith(expect.stringContaining('/builds/temp/temp-1'));
    });
    expect(mockedCreateTempBuild).toHaveBeenCalledTimes(1);
    expect(await screen.findByText('Share URL copied')).toBeInTheDocument();
  });
});

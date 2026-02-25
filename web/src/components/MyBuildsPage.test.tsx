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
  publishMyBuild,
  updateMyBuild,
  updateTempBuild,
} from '../buildApi';
import { listAircraft } from '../aircraftApi';
import { copyURLToClipboard } from '../buildShare';

const mockedCreateTempBuild = vi.mocked(createTempBuild);
const mockedGetMyBuild = vi.mocked(getMyBuild);
const mockedGetMyBuildImageUrl = vi.mocked(getMyBuildImageUrl);
const mockedListMyBuilds = vi.mocked(listMyBuilds);
const mockedListAircraft = vi.mocked(listAircraft);
const mockedPublishMyBuild = vi.mocked(publishMyBuild);
const mockedUpdateMyBuild = vi.mocked(updateMyBuild);
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
    mockedPublishMyBuild.mockReset();
    mockedUpdateMyBuild.mockReset();
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
    mockedUpdateMyBuild.mockImplementation(async (_id, params) => draftBuildFixture({
      title: params?.title ?? 'Untitled Build',
      description: params?.description ?? '',
      parts: params?.parts ?? [],
      youtubeUrl: params?.youtubeUrl,
      flightYoutubeUrl: params?.flightYoutubeUrl,
    }));
    mockedPublishMyBuild.mockResolvedValue({
      build: draftBuildFixture({ status: 'PENDING_REVIEW' }),
      validation: { valid: true, errors: [] },
    });
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

  it('switches to the new draft id after saving changes from a published build', async () => {
    const publishedBuild = draftBuildFixture({
      id: 'pub-1',
      status: 'PUBLISHED',
      title: 'Published Build',
      parts: [
        { gearType: 'frame', catalogItemId: 'frame-1' },
        { gearType: 'aio', catalogItemId: 'aio-1' },
      ],
    });
    const revisionBuild = draftBuildFixture({
      id: 'rev-1',
      status: 'DRAFT',
      title: 'Published Build',
      parts: [
        { gearType: 'frame', catalogItemId: 'frame-1' },
        { gearType: 'aio', catalogItemId: 'aio-1' },
      ],
    });

    mockedListMyBuilds.mockResolvedValue({
      builds: [publishedBuild],
      totalCount: 1,
      sort: 'newest',
    });
    mockedGetMyBuild.mockImplementation(async (id: string) => {
      if (id === 'rev-1') return revisionBuild;
      return publishedBuild;
    });
    mockedUpdateMyBuild.mockResolvedValue(revisionBuild);

    renderPage();

    await waitFor(() => {
      expect(mockedGetMyBuild).toHaveBeenCalledWith('pub-1');
    });

    fireEvent.click(screen.getByRole('button', { name: /save draft/i }));

    await waitFor(() => {
      expect(mockedUpdateMyBuild).toHaveBeenCalledWith('pub-1', expect.any(Object));
    });

    await waitFor(() => {
      expect(mockedGetMyBuild).toHaveBeenCalledWith('rev-1');
    });

    expect(await screen.findByRole('button', { name: /submit for review/i })).toBeInTheDocument();
  });

  it('publishes the id returned by the save step', async () => {
    const startingBuild = draftBuildFixture({
      id: 'build-1',
      status: 'DRAFT',
      parts: [
        { gearType: 'frame', catalogItemId: 'frame-1' },
        { gearType: 'aio', catalogItemId: 'aio-1' },
      ],
    });
    const savedBuild = draftBuildFixture({
      id: 'build-2',
      status: 'DRAFT',
      parts: [
        { gearType: 'frame', catalogItemId: 'frame-1' },
        { gearType: 'aio', catalogItemId: 'aio-1' },
      ],
    });

    mockedListMyBuilds.mockResolvedValue({
      builds: [startingBuild],
      totalCount: 1,
      sort: 'newest',
    });
    mockedGetMyBuild.mockImplementation(async (id: string) => {
      if (id === 'build-2') return savedBuild;
      return startingBuild;
    });
    mockedUpdateMyBuild.mockResolvedValue(savedBuild);

    renderPage();

    await waitFor(() => {
      expect(mockedGetMyBuild).toHaveBeenCalledWith('build-1');
    });

    fireEvent.click(screen.getByRole('button', { name: /submit for review/i }));

    await waitFor(() => {
      expect(mockedPublishMyBuild).toHaveBeenCalledWith('build-2');
    });
  });

  it('shows submit changes action for published builds with staged revisions', async () => {
    const publishedWithStagedChanges = draftBuildFixture({
      id: 'build-published',
      status: 'PUBLISHED',
      stagedRevisionStatus: 'DRAFT',
      parts: [
        { gearType: 'frame', catalogItemId: 'frame-1' },
        { gearType: 'aio', catalogItemId: 'aio-1' },
      ],
    });

    mockedListMyBuilds.mockResolvedValue({
      builds: [publishedWithStagedChanges],
      totalCount: 1,
      sort: 'newest',
    });
    mockedGetMyBuild.mockResolvedValue(publishedWithStagedChanges);

    renderPage();

    await waitFor(() => {
      expect(mockedGetMyBuild).toHaveBeenCalledWith('build-published');
    });

    expect(await screen.findByRole('button', { name: /submit changes for review/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /^unpublish$/i })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /^delete$/i })).not.toBeInTheDocument();
  });

  it('shows a disabled published button when published build has no staged or unsaved changes', async () => {
    const publishedBuild = draftBuildFixture({
      id: 'build-published',
      status: 'PUBLISHED',
      title: 'Published Build',
      parts: [
        { gearType: 'frame', catalogItemId: 'frame-1' },
      ],
    });

    mockedListMyBuilds.mockResolvedValue({
      builds: [publishedBuild],
      totalCount: 1,
      sort: 'newest',
    });
    mockedGetMyBuild.mockResolvedValue(publishedBuild);

    renderPage();

    await waitFor(() => {
      expect(mockedGetMyBuild).toHaveBeenCalledWith('build-published');
    });

    expect(await screen.findByRole('button', { name: /^published$/i })).toBeDisabled();
    expect(screen.getByRole('button', { name: /^unpublish$/i })).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /^delete$/i })).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: /submit changes for review/i })).not.toBeInTheDocument();
  });

  it('enables submit changes action after editing a published build', async () => {
    const publishedBuild = draftBuildFixture({
      id: 'build-published',
      status: 'PUBLISHED',
      title: 'Published Build',
      parts: [
        { gearType: 'frame', catalogItemId: 'frame-1' },
      ],
    });

    mockedListMyBuilds.mockResolvedValue({
      builds: [publishedBuild],
      totalCount: 1,
      sort: 'newest',
    });
    mockedGetMyBuild.mockResolvedValue(publishedBuild);

    renderPage();

    await waitFor(() => {
      expect(mockedGetMyBuild).toHaveBeenCalledWith('build-published');
    });

    expect(await screen.findByRole('button', { name: /^published$/i })).toBeDisabled();

    fireEvent.click(screen.getByRole('button', { name: /change title/i }));

    const submitChangesButton = await screen.findByRole('button', { name: /submit changes for review/i });
    expect(submitChangesButton).toBeEnabled();
  });

  it('shows moderator decline feedback in a modal for unpublished builds', async () => {
    const declinedBuild = draftBuildFixture({
      id: 'build-declined',
      status: 'UNPUBLISHED',
      title: 'Needs Fixes',
      moderationReason: 'Please provide a complete parts list and build description.',
      parts: [
        { gearType: 'frame', catalogItemId: 'frame-1' },
      ],
    });

    mockedListMyBuilds.mockResolvedValue({
      builds: [declinedBuild],
      totalCount: 1,
      sort: 'newest',
    });
    mockedGetMyBuild.mockResolvedValue(declinedBuild);

    renderPage();

    await waitFor(() => {
      expect(mockedGetMyBuild).toHaveBeenCalledWith('build-declined');
    });

    const feedbackModal = await screen.findByRole('dialog', { name: 'Build moderation feedback' });
    expect(feedbackModal).toHaveTextContent('Please provide a complete parts list and build description.');
    const gotItButton = screen.getByRole('button', { name: 'Got it' });
    await waitFor(() => {
      expect(gotItButton).toHaveFocus();
    });

    fireEvent.keyDown(window, { key: 'Escape' });
    await waitFor(() => {
      expect(screen.queryByRole('dialog', { name: 'Build moderation feedback' })).not.toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole('button', { name: /change title/i }));
    expect(screen.queryByRole('dialog', { name: 'Build moderation feedback' })).not.toBeInTheDocument();
  });
});

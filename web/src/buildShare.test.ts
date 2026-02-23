import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('./buildApi', () => ({
  createTempBuild: vi.fn(),
  shareTempBuild: vi.fn(),
}));

import { createTempBuild, shareTempBuild } from './buildApi';
import {
  copyURLToClipboard,
  createShareUrlForBuild,
  getBuildURLContext,
  getPublishedBuildUrl,
  toAbsoluteBuildUrl,
} from './buildShare';

const mockedCreateTempBuild = vi.mocked(createTempBuild);
const mockedShareTempBuild = vi.mocked(shareTempBuild);

describe('buildShare', () => {
  beforeEach(() => {
    mockedCreateTempBuild.mockReset();
    mockedShareTempBuild.mockReset();
  });

  it('returns published build url without creating temp snapshots', async () => {
    const url = await createShareUrlForBuild({
      id: 'published-1',
      status: 'PUBLISHED',
      title: 'Published Build',
      description: '',
      createdAt: '2026-02-20T00:00:00Z',
      updatedAt: '2026-02-20T00:00:00Z',
      verified: true,
      parts: [],
    });

    expect(url).toContain('/builds/published-1');
    expect(mockedCreateTempBuild).not.toHaveBeenCalled();
    expect(mockedShareTempBuild).not.toHaveBeenCalled();
  });

  it('creates and shares a temp snapshot for non-published builds', async () => {
    mockedCreateTempBuild.mockResolvedValue({
      token: 'temp-token-1',
      url: '/builds/temp/temp-token-1',
      build: {
        id: 'temp-build-1',
        status: 'TEMP',
        title: 'Draft Build',
        description: 'Draft desc',
        createdAt: '2026-02-20T00:00:00Z',
        updatedAt: '2026-02-20T00:00:00Z',
        verified: false,
        parts: [],
      },
    });
    mockedShareTempBuild.mockResolvedValue({
      token: 'shared-token-1',
      url: '/builds/temp/shared-token-1',
      build: {
        id: 'shared-build-1',
        status: 'SHARED',
        title: 'Draft Build',
        description: 'Draft desc',
        createdAt: '2026-02-20T00:00:00Z',
        updatedAt: '2026-02-20T00:00:00Z',
        verified: false,
        parts: [],
      },
    });

    const url = await createShareUrlForBuild({
      id: 'draft-1',
      status: 'DRAFT',
      title: 'Draft Build',
      description: 'Draft desc',
      createdAt: '2026-02-20T00:00:00Z',
      updatedAt: '2026-02-20T00:00:00Z',
      verified: false,
      parts: [
        {
          gearType: 'frame',
          catalogItemId: 'frame-1',
          notes: 'strong',
        },
      ],
    });

    expect(mockedCreateTempBuild).toHaveBeenCalledWith({
      title: 'Draft Build',
      description: 'Draft desc',
      sourceAircraftId: undefined,
      parts: [
        {
          gearType: 'frame',
          catalogItemId: 'frame-1',
          position: undefined,
          notes: 'strong',
        },
      ],
    });
    expect(mockedShareTempBuild).toHaveBeenCalledWith('temp-token-1');
    expect(url).toContain('/builds/temp/shared-token-1');
  });

  it('falls back to token path when shared response omits url', async () => {
    mockedCreateTempBuild.mockResolvedValue({
      token: 'temp-token-2',
      url: '/builds/temp/temp-token-2',
      build: {
        id: 'temp-build-2',
        status: 'TEMP',
        title: 'Draft Build',
        description: '',
        createdAt: '2026-02-20T00:00:00Z',
        updatedAt: '2026-02-20T00:00:00Z',
        verified: false,
        parts: [],
      },
    });
    mockedShareTempBuild.mockResolvedValue({
      token: 'shared-token-2',
      url: '',
      build: {
        id: 'shared-build-2',
        status: 'SHARED',
        title: 'Draft Build',
        description: '',
        createdAt: '2026-02-20T00:00:00Z',
        updatedAt: '2026-02-20T00:00:00Z',
        verified: false,
        parts: [],
      },
    });

    const url = await createShareUrlForBuild({
      id: 'draft-2',
      status: 'UNPUBLISHED',
      title: 'Draft Build',
      description: '',
      createdAt: '2026-02-20T00:00:00Z',
      updatedAt: '2026-02-20T00:00:00Z',
      verified: false,
      parts: [],
    });

    expect(url).toContain('/builds/temp/shared-token-2');
  });

  it('normalizes relative urls into absolute urls', () => {
    expect(toAbsoluteBuildUrl('/builds/test-1')).toContain('/builds/test-1');
    expect(getPublishedBuildUrl('pub-1')).toContain('/builds/pub-1');
  });

  it('returns public context for published builds', () => {
    const context = getBuildURLContext({
      id: 'published-2',
      status: 'PUBLISHED',
      title: 'Published Build',
      description: '',
      createdAt: '2026-02-20T00:00:00Z',
      updatedAt: '2026-02-20T00:00:00Z',
      verified: true,
      parts: [],
    });

    expect(context.label).toBe('Public build URL');
    expect(context.url).toContain('/builds/published-2');
    expect(context.emptyMessage).toBe('');
  });

  it('returns share context for non-published builds', () => {
    const emptyContext = getBuildURLContext({
      id: 'draft-3',
      status: 'DRAFT',
      title: 'Draft Build',
      description: '',
      createdAt: '2026-02-20T00:00:00Z',
      updatedAt: '2026-02-20T00:00:00Z',
      verified: false,
      parts: [],
    });
    expect(emptyContext.label).toBe('Share URL');
    expect(emptyContext.url).toBe('');
    expect(emptyContext.emptyMessage).toContain('Copy Share URL');

    const generatedContext = getBuildURLContext({
      id: 'draft-3',
      status: 'DRAFT',
      title: 'Draft Build',
      description: '',
      createdAt: '2026-02-20T00:00:00Z',
      updatedAt: '2026-02-20T00:00:00Z',
      verified: false,
      parts: [],
    }, 'https://example.com/builds/temp/abc');
    expect(generatedContext.url).toBe('https://example.com/builds/temp/abc');
  });

  it('writes url to clipboard', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText },
    });

    await copyURLToClipboard('https://example.com/builds/1');

    expect(writeText).toHaveBeenCalledWith('https://example.com/builds/1');
  });

  it('falls back to execCommand copy when clipboard write is denied', async () => {
    const writeText = vi.fn().mockRejectedValue(new Error('NotAllowedError'));
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText },
    });

    const execCommandMock = vi.fn().mockReturnValue(true);
    Object.defineProperty(document, 'execCommand', {
      configurable: true,
      value: execCommandMock,
    });

    await copyURLToClipboard('https://example.com/builds/2');

    expect(writeText).toHaveBeenCalledWith('https://example.com/builds/2');
    expect(execCommandMock).toHaveBeenCalledWith('copy');
  });

  it('returns a clear error when clipboard copy is unavailable', async () => {
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: undefined,
    });
    Object.defineProperty(document, 'execCommand', {
      configurable: true,
      value: undefined,
    });

    await expect(copyURLToClipboard('https://example.com/builds/3')).rejects.toThrow(
      'Unable to copy automatically. Please copy the URL manually.',
    );
  });
});

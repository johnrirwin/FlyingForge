import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { render } from '../test/test-utils';
import { AdminGearModeration } from './AdminGearModeration';
import type { GearCatalogItem } from '../gearCatalogTypes';
import type { Build } from '../buildTypes';
import {
  adminSearchBuilds,
  adminGetBuild,
  adminUpdateBuild,
  adminPublishBuild,
  adminUnpublishBuild,
  adminUploadBuildImage,
  adminDeleteBuildImage,
  adminDeleteGear,
  adminBulkDeleteGear,
  adminDeleteGearImage,
  adminGetGear,
  adminSaveGearImageUpload,
  adminSearchGear,
  adminUpdateGear,
  getAdminBuildImageUrl,
  getAdminGearImageUrl,
} from '../adminApi';
import { moderateGearCatalogImageUpload } from '../gearCatalogApi';

vi.mock('../adminApi', () => ({
  adminSearchBuilds: vi.fn(),
  adminGetBuild: vi.fn(),
  adminUpdateBuild: vi.fn(),
  adminPublishBuild: vi.fn(),
  adminUnpublishBuild: vi.fn(),
  adminUploadBuildImage: vi.fn(),
  adminDeleteBuildImage: vi.fn(),
  adminSearchGear: vi.fn(),
  adminUpdateGear: vi.fn(),
  adminSaveGearImageUpload: vi.fn(),
  adminDeleteGearImage: vi.fn(),
  adminDeleteGear: vi.fn(),
  adminBulkDeleteGear: vi.fn(),
  adminGetGear: vi.fn(),
  getAdminBuildImageUrl: vi.fn(() => '/mock-build-image.png'),
  getAdminGearImageUrl: vi.fn(() => '/mock-image.png'),
}));

vi.mock('../gearCatalogApi', () => ({
  searchGearCatalog: vi.fn().mockResolvedValue({ items: [], totalCount: 0 }),
  createGearCatalogItem: vi.fn().mockResolvedValue({
    item: {
      id: 'mock-gear',
      gearType: 'other',
      brand: 'Mock',
      model: 'Item',
      source: 'admin',
      status: 'pending',
      canonicalKey: 'other|mock|item',
      usageCount: 0,
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-01T00:00:00Z',
      imageStatus: 'missing',
      descriptionStatus: 'missing',
    },
    existing: true,
  }),
  findNearMatches: vi.fn().mockResolvedValue({ matches: [] }),
  getPopularGear: vi.fn().mockResolvedValue({ items: [] }),
  moderateGearCatalogImageUpload: vi.fn(),
}));

const mockAdminSearchBuilds = vi.mocked(adminSearchBuilds);
const mockAdminGetBuild = vi.mocked(adminGetBuild);
const mockAdminUpdateBuild = vi.mocked(adminUpdateBuild);
const mockAdminPublishBuild = vi.mocked(adminPublishBuild);
const mockAdminUnpublishBuild = vi.mocked(adminUnpublishBuild);
const mockAdminUploadBuildImage = vi.mocked(adminUploadBuildImage);
const mockAdminDeleteBuildImage = vi.mocked(adminDeleteBuildImage);
const mockAdminSearchGear = vi.mocked(adminSearchGear);
const mockAdminUpdateGear = vi.mocked(adminUpdateGear);
const mockAdminSaveGearImageUpload = vi.mocked(adminSaveGearImageUpload);
const mockAdminDeleteGearImage = vi.mocked(adminDeleteGearImage);
const mockAdminDeleteGear = vi.mocked(adminDeleteGear);
const mockAdminBulkDeleteGear = vi.mocked(adminBulkDeleteGear);
const mockAdminGetGear = vi.mocked(adminGetGear);
const mockGetAdminBuildImageUrl = vi.mocked(getAdminBuildImageUrl);
const mockGetAdminGearImageUrl = vi.mocked(getAdminGearImageUrl);
const mockModerateGearCatalogImageUpload = vi.mocked(moderateGearCatalogImageUpload);

type ObjectUrlStatics = {
  createObjectURL?: (obj: Blob) => string;
  revokeObjectURL?: (url: string) => void;
};

const mockItem: GearCatalogItem = {
  id: 'gear-1',
  gearType: 'motor',
  brand: 'EMAX',
  model: 'ECO II',
  variant: '2207',
  specs: {},
  bestFor: ['freestyle'],
  msrp: 19.99,
  source: 'admin',
  createdByUserId: 'admin-1',
  status: 'published',
  canonicalKey: 'motor-emax-eco-ii-2207',
  imageUrl: undefined,
  description: 'Great all-around motor',
  usageCount: 0,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-05T00:00:00Z',
  imageStatus: 'missing',
  imageCuratedByUserId: undefined,
  imageCuratedAt: undefined,
  descriptionStatus: 'missing',
  descriptionCuratedByUserId: undefined,
  descriptionCuratedAt: undefined,
};

const mockBuild: Build = {
  id: 'build-1',
  ownerUserId: 'pilot-1',
  status: 'PUBLISHED',
  title: 'Blue Beast Build',
  description: 'Fast freestyle build',
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-05T00:00:00Z',
  publishedAt: '2026-01-04T00:00:00Z',
  verified: true,
  parts: [],
  pilot: {
    userId: 'pilot-1',
    callSign: 'BlueBeast',
    displayName: 'BlueBeast',
    isProfilePublic: true,
  },
};

describe('AdminGearModeration', () => {
  let hadCreateObjectURL = false;
  let hadRevokeObjectURL = false;
  let originalCreateObjectURL: ObjectUrlStatics['createObjectURL'];
  let originalRevokeObjectURL: ObjectUrlStatics['revokeObjectURL'];

  beforeEach(() => {
    vi.clearAllMocks();
    // JSDOM doesn't implement these; gear/build moderation UIs rely on them.
    // Restore in afterEach to avoid leaking state into other test files.
    const urlStatics = URL as unknown as ObjectUrlStatics;
    hadCreateObjectURL = typeof urlStatics.createObjectURL === 'function';
    hadRevokeObjectURL = typeof urlStatics.revokeObjectURL === 'function';
    originalCreateObjectURL = urlStatics.createObjectURL;
    originalRevokeObjectURL = urlStatics.revokeObjectURL;
    urlStatics.createObjectURL = vi.fn(() => 'blob:mock-url');
    urlStatics.revokeObjectURL = vi.fn();

    mockAdminSearchGear.mockResolvedValue({
      items: [mockItem],
      totalCount: 1,
    });
    mockAdminGetGear.mockResolvedValue(mockItem);
    mockAdminUpdateGear.mockResolvedValue(mockItem);
    mockAdminSaveGearImageUpload.mockResolvedValue();
    mockAdminDeleteGearImage.mockResolvedValue();
    mockAdminDeleteGear.mockResolvedValue();
    mockAdminBulkDeleteGear.mockResolvedValue({
      deletedIds: ['gear-1'],
      deletedCount: 1,
      notFoundIds: [],
      notFoundCount: 0,
    });
    mockAdminSearchBuilds.mockResolvedValue({ builds: [], totalCount: 0, sort: 'newest' });
    mockAdminGetBuild.mockRejectedValue(new Error('Build not mocked'));
    mockAdminUpdateBuild.mockRejectedValue(new Error('Build not mocked'));
    mockAdminPublishBuild.mockRejectedValue(new Error('Build not mocked'));
    mockAdminUnpublishBuild.mockRejectedValue(new Error('Build not mocked'));
    mockAdminUploadBuildImage.mockResolvedValue();
    mockAdminDeleteBuildImage.mockResolvedValue();
    mockGetAdminBuildImageUrl.mockReturnValue('/mock-build-image.png');
    mockGetAdminGearImageUrl.mockReturnValue('/mock-image.png');
    mockModerateGearCatalogImageUpload.mockResolvedValue({ status: 'APPROVED', uploadId: 'upload-1' });
  });

  afterEach(() => {
    const urlStatics = URL as unknown as ObjectUrlStatics;
    if (hadCreateObjectURL) {
      urlStatics.createObjectURL = originalCreateObjectURL;
    } else {
      delete urlStatics.createObjectURL;
    }

    if (hadRevokeObjectURL) {
      urlStatics.revokeObjectURL = originalRevokeObjectURL;
    } else {
      delete urlStatics.revokeObjectURL;
    }
  });

  it('shows upload and last edit columns and opens modal by clicking a row', async () => {
    render(<AdminGearModeration hasContentAdminAccess authLoading={false} />);

    expect(await screen.findByText('EMAX')).toBeInTheDocument();
    const table = screen.getByRole('table');
    expect(within(table).getByRole('columnheader', { name: 'Upload Date' })).toBeInTheDocument();
    expect(within(table).getByRole('columnheader', { name: 'Last Edit' })).toBeInTheDocument();

    const row = screen.getByRole('button', { name: 'Open editor for EMAX ECO II 2207' });
    fireEvent.click(row);

    expect(await screen.findByText('Edit Gear Item')).toBeInTheDocument();
    expect(screen.getByText(/Upload Date:/)).toBeInTheDocument();
    expect(screen.getByText(/Last Edit:/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Delete Item' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Save Changes' })).toBeInTheDocument();
  });

  it('defaults gear moderation filters to all types, pending, and all records', async () => {
    render(<AdminGearModeration hasContentAdminAccess authLoading={false} />);

    expect(await screen.findByText('EMAX')).toBeInTheDocument();

    expect(mockAdminSearchGear).toHaveBeenCalledWith({
      query: undefined,
      gearType: undefined,
      status: 'pending',
      imageStatus: 'all',
      limit: 30,
      offset: 0,
    });
  });

  it('runs server-side search even while a prior page load is still in flight', async () => {
    let resolveInitialRequest!: (value: { items: GearCatalogItem[]; totalCount: number }) => void;
    const initialRequest = new Promise<{ items: GearCatalogItem[]; totalCount: number }>((resolve) => {
      resolveInitialRequest = resolve;
    });

    mockAdminSearchGear.mockImplementationOnce(() => initialRequest);

    render(<AdminGearModeration hasContentAdminAccess authLoading={false} />);

    const searchInput = screen.getByPlaceholderText('Search brand or model...');
    fireEvent.change(searchInput, { target: { value: 'DJI' } });
    fireEvent.click(screen.getByRole('button', { name: 'Search' }));

    await waitFor(() => {
      expect(mockAdminSearchGear).toHaveBeenCalledTimes(2);
    });

    expect(mockAdminSearchGear).toHaveBeenNthCalledWith(2, {
      query: 'DJI',
      gearType: undefined,
      status: 'pending',
      imageStatus: 'all',
      limit: 30,
      offset: 0,
    });

    resolveInitialRequest({
      items: [mockItem],
      totalCount: 1,
    });
  });

  it('opens the edit modal with keyboard interaction on a table row', async () => {
    render(<AdminGearModeration hasContentAdminAccess authLoading={false} />);

    const row = await screen.findByRole('button', { name: 'Open editor for EMAX ECO II 2207' });
    fireEvent.keyDown(row, { key: 'Enter' });

    expect(await screen.findByText('Edit Gear Item')).toBeInTheDocument();
    await waitFor(() => {
      expect(mockAdminGetGear).toHaveBeenCalledWith('gear-1');
    });
  });

  it('adds and saves specs from within the edit modal', async () => {
    render(<AdminGearModeration hasContentAdminAccess authLoading={false} />);

    const row = await screen.findByRole('button', { name: 'Open editor for EMAX ECO II 2207' });
    fireEvent.click(row);
    expect(await screen.findByText('Edit Gear Item')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Add Spec' }));

    fireEvent.change(screen.getByPlaceholderText('Key'), { target: { value: 'kv' } });
    fireEvent.change(screen.getByPlaceholderText('Value'), { target: { value: '1950' } });

    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }));

    await waitFor(() => {
      expect(mockAdminUpdateGear).toHaveBeenCalledWith(
        'gear-1',
        expect.objectContaining({
          specs: { kv: '1950' },
        })
      );
    });
  });

  it('allows moderators to unpublish a published build', async () => {
    const unpublishedBuild: Build = {
      ...mockBuild,
      status: 'UNPUBLISHED',
    };

    mockAdminSearchBuilds.mockResolvedValue({
      builds: [mockBuild],
      totalCount: 1,
      sort: 'newest',
    });
    mockAdminGetBuild.mockResolvedValue(mockBuild);
    mockAdminUpdateBuild.mockResolvedValue(mockBuild);
    mockAdminUnpublishBuild.mockResolvedValue(unpublishedBuild);

    render(<AdminGearModeration hasContentAdminAccess authLoading={false} />);

    fireEvent.click(screen.getByRole('button', { name: /^Builds$/i }));

    const row = await screen.findByRole('button', { name: /open editor for blue beast build/i });
    fireEvent.click(row);

    expect(await screen.findByText('Review Build')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Unpublish Build' }));

    await waitFor(() => {
      expect(mockAdminUnpublishBuild).toHaveBeenCalledWith('build-1');
    });
  });

  it('updates gear type from within the edit modal', async () => {
    render(<AdminGearModeration hasContentAdminAccess authLoading={false} />);

    const row = await screen.findByRole('button', { name: 'Open editor for EMAX ECO II 2207' });
    fireEvent.click(row);
    expect(await screen.findByText('Edit Gear Item')).toBeInTheDocument();

    fireEvent.change(screen.getByRole('combobox', { name: 'Gear Type' }), { target: { value: 'esc' } });

    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }));

    await waitFor(() => {
      expect(mockAdminUpdateGear).toHaveBeenCalledWith(
        'gear-1',
        expect.objectContaining({
          gearType: 'esc',
        })
      );
    });
  });

  it('loads existing specs into the edit modal and allows editing', async () => {
    mockAdminGetGear.mockResolvedValueOnce({
      ...mockItem,
      specs: { kv: '1950', stator: '27mm' },
    });

    render(<AdminGearModeration hasContentAdminAccess authLoading={false} />);

    const row = await screen.findByRole('button', { name: 'Open editor for EMAX ECO II 2207' });
    fireEvent.click(row);

    expect(await screen.findByText('Edit Gear Item')).toBeInTheDocument();

    // Existing specs should be loaded into the form
    expect(screen.getByDisplayValue('kv')).toBeInTheDocument();
    expect(screen.getByDisplayValue('1950')).toBeInTheDocument();
    expect(screen.getByDisplayValue('stator')).toBeInTheDocument();
    expect(screen.getByDisplayValue('27mm')).toBeInTheDocument();

    // Edit one of the existing specs
    fireEvent.change(screen.getByDisplayValue('1950'), { target: { value: '2200' } });

    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }));

    await waitFor(() => {
      expect(mockAdminUpdateGear).toHaveBeenCalledWith(
        'gear-1',
        expect.objectContaining({
          specs: { kv: '2200', stator: '27mm' },
        })
      );
    });
  });

  it('allows removing a spec before saving', async () => {
    mockAdminGetGear.mockResolvedValueOnce({
      ...mockItem,
      specs: { kv: '1950', stator: '27mm' },
    });

    render(<AdminGearModeration hasContentAdminAccess authLoading={false} />);

    const row = await screen.findByRole('button', { name: 'Open editor for EMAX ECO II 2207' });
    fireEvent.click(row);
    expect(await screen.findByText('Edit Gear Item')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Remove spec stator' }));

    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }));

    await waitFor(() => {
      expect(mockAdminUpdateGear).toHaveBeenCalledWith(
        'gear-1',
        expect.objectContaining({
          specs: { kv: '1950' },
        })
      );
    });
  });

  it('prevents saving when duplicate spec keys are entered', async () => {
    render(<AdminGearModeration hasContentAdminAccess authLoading={false} />);

    const row = await screen.findByRole('button', { name: 'Open editor for EMAX ECO II 2207' });
    fireEvent.click(row);
    expect(await screen.findByText('Edit Gear Item')).toBeInTheDocument();

    // First spec
    fireEvent.click(screen.getByRole('button', { name: 'Add Spec' }));
    fireEvent.change(screen.getByPlaceholderText('Key'), { target: { value: 'kv' } });
    fireEvent.change(screen.getByPlaceholderText('Value'), { target: { value: '1950' } });

    // Second spec with duplicate key
    fireEvent.click(screen.getByRole('button', { name: 'Add Spec' }));
    const keyInputs = screen.getAllByPlaceholderText('Key');
    const valueInputs = screen.getAllByPlaceholderText('Value');

    fireEvent.change(keyInputs[1], { target: { value: 'kv' } });
    fireEvent.change(valueInputs[1], { target: { value: '2200' } });

    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }));

    expect(await screen.findByText('Duplicate spec key: kv')).toBeInTheDocument();
    expect(mockAdminUpdateGear).not.toHaveBeenCalled();
  });

  it('prevents saving when duplicate spec keys match existing specs', async () => {
    mockAdminGetGear.mockResolvedValueOnce({
      ...mockItem,
      specs: { kv: '1950' },
    });

    render(<AdminGearModeration hasContentAdminAccess authLoading={false} />);

    const row = await screen.findByRole('button', { name: 'Open editor for EMAX ECO II 2207' });
    fireEvent.click(row);
    expect(await screen.findByText('Edit Gear Item')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Add Spec' }));

    const keyInputs = screen.getAllByPlaceholderText('Key');
    const valueInputs = screen.getAllByPlaceholderText('Value');

    fireEvent.change(keyInputs[1], { target: { value: 'kv' } });
    fireEvent.change(valueInputs[1], { target: { value: '1950' } });

    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }));

    expect(await screen.findByText('Duplicate spec key: kv')).toBeInTheDocument();
    expect(mockAdminUpdateGear).not.toHaveBeenCalled();
  });

  it('filters out specs with empty keys before saving', async () => {
    mockAdminGetGear.mockResolvedValueOnce({
      ...mockItem,
      specs: { kv: '1950' },
    });

    render(<AdminGearModeration hasContentAdminAccess authLoading={false} />);

    const row = await screen.findByRole('button', { name: 'Open editor for EMAX ECO II 2207' });
    fireEvent.click(row);
    expect(await screen.findByText('Edit Gear Item')).toBeInTheDocument();

    fireEvent.change(screen.getByDisplayValue('kv'), { target: { value: '' } });

    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }));

    await waitFor(() => {
      expect(mockAdminUpdateGear).toHaveBeenCalledWith(
        'gear-1',
        expect.objectContaining({
          specs: {},
        })
      );
    });
  });

  it('deletes from within the edit modal and refreshes list', async () => {
    mockAdminSearchGear
      .mockResolvedValueOnce({ items: [mockItem], totalCount: 1 })
      .mockResolvedValueOnce({ items: [], totalCount: 0 });

    render(<AdminGearModeration hasContentAdminAccess authLoading={false} />);

    const row = await screen.findByRole('button', { name: 'Open editor for EMAX ECO II 2207' });
    fireEvent.click(row);
    expect(await screen.findByText('Edit Gear Item')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Delete Item' }));
    const deleteDialog = await screen.findByRole('dialog', { name: 'Delete Gear Item?' });
    expect(deleteDialog).toHaveAttribute('aria-modal', 'true');
    fireEvent.click(within(deleteDialog).getByRole('button', { name: 'Delete Item' }));

    await waitFor(() => {
      expect(mockAdminDeleteGear).toHaveBeenCalledWith('gear-1');
    });

    await waitFor(() => {
      expect(mockAdminSearchGear).toHaveBeenCalledTimes(2);
    });
  });

  it('shows an in-modal validation error when uploaded image exceeds 2MB', async () => {
    render(<AdminGearModeration hasContentAdminAccess authLoading={false} />);

    const row = await screen.findByRole('button', { name: 'Open editor for EMAX ECO II 2207' });
    fireEvent.click(row);
    expect(await screen.findByText('Edit Gear Item')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Add Image' }));
    expect(await screen.findByText('Edit Gear Image')).toBeInTheDocument();

    const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement;
    expect(fileInput).toBeTruthy();

    const oversizedFile = new File([new Uint8Array((2 * 1024 * 1024) + 1)], 'too-large.jpg', {
      type: 'image/jpeg',
    });

    fireEvent.change(fileInput, { target: { files: [oversizedFile] } });

    expect(await screen.findByText('Image file is too large. Maximum size is 2MB.')).toBeInTheDocument();
  });

  it('moderates and saves a scanned image when updating a gear record', async () => {
    render(<AdminGearModeration hasContentAdminAccess authLoading={false} />);

    const row = await screen.findByRole('button', { name: 'Open editor for EMAX ECO II 2207' });
    fireEvent.click(row);
    expect(await screen.findByText('Edit Gear Item')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Add Image' }));
    expect(await screen.findByText('Edit Gear Image')).toBeInTheDocument();

    const fileInput = document.querySelector('input[type="file"]') as HTMLInputElement;
    const validFile = new File([new Uint8Array(128)], 'ok.jpg', { type: 'image/jpeg' });
    fireEvent.change(fileInput, { target: { files: [validFile] } });

    await waitFor(() => {
      expect(mockModerateGearCatalogImageUpload).toHaveBeenCalled();
    });
    expect(await screen.findByText('Approved')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }));

    await waitFor(() => {
      expect(mockAdminSaveGearImageUpload).toHaveBeenCalledWith('gear-1', 'upload-1');
    });
    await waitFor(() => {
      expect(mockAdminUpdateGear).toHaveBeenCalledWith('gear-1', expect.objectContaining({ imageStatus: 'scanned' }));
    });
  });

  it('saves external imageUrl overrides and marks imageStatus approved', async () => {
    render(<AdminGearModeration hasContentAdminAccess authLoading={false} />);

    const row = await screen.findByRole('button', { name: 'Open editor for EMAX ECO II 2207' });
    fireEvent.click(row);
    expect(await screen.findByText('Edit Gear Item')).toBeInTheDocument();

    const input = screen.getByPlaceholderText('https://example.com/image.jpg');
    fireEvent.change(input, { target: { value: 'https://example.com/new.jpg' } });

    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }));

    await waitFor(() => {
      expect(mockAdminUpdateGear).toHaveBeenCalled();
    });

    const [, params] = mockAdminUpdateGear.mock.calls[0];
    expect(params).toMatchObject({ imageUrl: 'https://example.com/new.jpg', imageStatus: 'approved' });
  });

  it('clears external imageUrl overrides when the field is emptied', async () => {
    const itemWithExternal: GearCatalogItem = {
      ...mockItem,
      imageUrl: 'https://example.com/old.jpg',
      imageStatus: 'approved',
    };
    mockAdminGetGear.mockResolvedValueOnce(itemWithExternal);

    render(<AdminGearModeration hasContentAdminAccess authLoading={false} />);

    const row = await screen.findByRole('button', { name: 'Open editor for EMAX ECO II 2207' });
    fireEvent.click(row);
    expect(await screen.findByText('Edit Gear Item')).toBeInTheDocument();

    const input = screen.getByDisplayValue('https://example.com/old.jpg');
    fireEvent.change(input, { target: { value: '' } });

    fireEvent.click(screen.getByRole('button', { name: 'Save Changes' }));

    await waitFor(() => {
      expect(mockAdminUpdateGear).toHaveBeenCalled();
    });

    const [, params] = mockAdminUpdateGear.mock.calls[0];
    expect(params).toMatchObject({ imageUrl: '', imageStatus: 'missing' });
  });

  it('bulk deletes selected gear items from the list view', async () => {
    mockAdminSearchGear
      .mockResolvedValueOnce({ items: [mockItem], totalCount: 1 })
      .mockResolvedValueOnce({ items: [], totalCount: 0 });

    render(<AdminGearModeration hasContentAdminAccess authLoading={false} />);

    expect(await screen.findByText('EMAX')).toBeInTheDocument();

    fireEvent.click(screen.getAllByRole('button', { name: 'Bulk Edit' })[0]);

    const table = screen.getByRole('table');
    fireEvent.click(within(table).getByRole('button', { name: 'Select EMAX ECO II 2207' }));

    const deleteSelectedButton = screen.getAllByRole('button', { name: 'Delete Selected (1)' })[0];
    fireEvent.click(deleteSelectedButton);

    const dialog = await screen.findByRole('dialog', { name: 'Delete Selected Gear Items?' });
    expect(dialog).toHaveAttribute('aria-modal', 'true');

    fireEvent.change(within(dialog).getByPlaceholderText('DELETE'), { target: { value: 'DELETE' } });
    fireEvent.click(within(dialog).getByRole('button', { name: 'Delete 1' }));

    await waitFor(() => {
      expect(mockAdminBulkDeleteGear).toHaveBeenCalledWith(['gear-1']);
    });

    await waitFor(() => {
      expect(mockAdminSearchGear).toHaveBeenCalledTimes(2);
    });
  });
});

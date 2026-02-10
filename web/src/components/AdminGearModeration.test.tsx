import { describe, it, expect, vi, beforeEach } from 'vitest';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { render } from '../test/test-utils';
import { AdminGearModeration } from './AdminGearModeration';
import type { GearCatalogItem } from '../gearCatalogTypes';
import {
  adminDeleteGear,
  adminDeleteGearImage,
  adminGetGear,
  adminSearchGear,
  adminUpdateGear,
  adminUploadGearImage,
  getAdminGearImageUrl,
} from '../adminApi';

vi.mock('../adminApi', () => ({
  adminSearchGear: vi.fn(),
  adminUpdateGear: vi.fn(),
  adminUploadGearImage: vi.fn(),
  adminDeleteGearImage: vi.fn(),
  adminDeleteGear: vi.fn(),
  adminGetGear: vi.fn(),
  getAdminGearImageUrl: vi.fn(() => '/mock-image.png'),
}));

const mockAdminSearchGear = vi.mocked(adminSearchGear);
const mockAdminUpdateGear = vi.mocked(adminUpdateGear);
const mockAdminUploadGearImage = vi.mocked(adminUploadGearImage);
const mockAdminDeleteGearImage = vi.mocked(adminDeleteGearImage);
const mockAdminDeleteGear = vi.mocked(adminDeleteGear);
const mockAdminGetGear = vi.mocked(adminGetGear);
const mockGetAdminGearImageUrl = vi.mocked(getAdminGearImageUrl);

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
  status: 'active',
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

describe('AdminGearModeration', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockAdminSearchGear.mockResolvedValue({
      items: [mockItem],
      totalCount: 1,
    });
    mockAdminGetGear.mockResolvedValue(mockItem);
    mockAdminUpdateGear.mockResolvedValue(mockItem);
    mockAdminUploadGearImage.mockResolvedValue();
    mockAdminDeleteGearImage.mockResolvedValue();
    mockAdminDeleteGear.mockResolvedValue();
    mockGetAdminGearImageUrl.mockReturnValue('/mock-image.png');
  });

  it('shows upload and last edit columns and opens modal by clicking a row', async () => {
    render(<AdminGearModeration hasGearAdminAccess authLoading={false} />);

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

  it('opens the edit modal with keyboard interaction on a table row', async () => {
    render(<AdminGearModeration hasGearAdminAccess authLoading={false} />);

    const row = await screen.findByRole('button', { name: 'Open editor for EMAX ECO II 2207' });
    fireEvent.keyDown(row, { key: 'Enter' });

    expect(await screen.findByText('Edit Gear Item')).toBeInTheDocument();
    await waitFor(() => {
      expect(mockAdminGetGear).toHaveBeenCalledWith('gear-1');
    });
  });

  it('deletes from within the edit modal and refreshes list', async () => {
    mockAdminSearchGear
      .mockResolvedValueOnce({ items: [mockItem], totalCount: 1 })
      .mockResolvedValueOnce({ items: [], totalCount: 0 });

    render(<AdminGearModeration hasGearAdminAccess authLoading={false} />);

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
});

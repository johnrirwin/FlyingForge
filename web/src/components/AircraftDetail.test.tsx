import { beforeEach, describe, expect, it, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import { fireEvent, render, screen, waitFor, within } from '../test/test-utils';
import { AircraftDetail } from './AircraftDetail';
import type { AircraftDetailsResponse } from '../aircraftTypes';
import type { AddInventoryParams } from '../equipmentTypes';

vi.mock('../equipmentApi', () => ({
  getInventory: vi.fn(),
}));

vi.mock('../fcConfigApi', () => ({
  getAircraftTuning: vi.fn(),
  createTuningSnapshot: vi.fn(),
}));

vi.mock('./AddGearModal', () => ({
  AddGearModal: (props: {
    isOpen: boolean;
    initialCategory?: string;
    onSubmit: (params: AddInventoryParams) => Promise<void>;
  }) => {
    if (!props.isOpen) {
      return null;
    }

    return (
      <div data-testid="mock-add-gear-modal">
        <span data-testid="mock-add-gear-category">{props.initialCategory ?? ''}</span>
        <button
          type="button"
          onClick={() => {
            void props
              .onSubmit({
                name: 'HD Camera',
                category: 'cameras',
                quantity: 1,
              })
              .catch(() => {});
          }}
        >
          submit-matching
        </button>
        <button
          type="button"
          onClick={() => {
            void props
              .onSubmit({
                name: '2307 Motor',
                category: 'motors',
                quantity: 1,
              })
              .catch(() => {});
          }}
        >
          submit-mismatch
        </button>
      </div>
    );
  },
}));

import { getInventory } from '../equipmentApi';
import { getAircraftTuning } from '../fcConfigApi';

const mockedGetInventory = vi.mocked(getInventory);
const mockedGetAircraftTuning = vi.mocked(getAircraftTuning);

const details: AircraftDetailsResponse = {
  aircraft: {
    id: 'ac-1',
    userId: 'user-1',
    name: 'Blue Beast',
    type: 'freestyle',
    hasImage: false,
    createdAt: '2026-02-19T00:00:00Z',
    updatedAt: '2026-02-19T00:00:00Z',
  },
  components: [],
  receiverSettings: {
    aircraftId: 'ac-1',
    settings: {},
    updatedAt: '2026-02-19T00:00:00Z',
  },
};

function openCameraAddFlow() {
  const cameraHeading = screen.getByRole('heading', { name: 'Camera' });
  const cameraCard = cameraHeading.closest('div[class*="bg-slate-700"]');
  if (!(cameraCard instanceof HTMLElement)) {
    throw new Error('Camera card not found');
  }

  fireEvent.click(within(cameraCard).getByRole('button', { name: 'Assign' }));
  fireEvent.click(screen.getByRole('button', { name: '+ Add Camera to Inventory' }));
}

describe('AircraftDetail modal gear assignment flow', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockedGetInventory.mockResolvedValue({ items: [], totalCount: 0 });
    mockedGetAircraftTuning.mockResolvedValue({
      aircraftId: details.aircraft.id,
      hasTuning: false,
    });
  });

  it('opens add gear modal with component category context and assigns matching items', async () => {
    const onSetComponent = vi.fn().mockResolvedValue(undefined);
    const onRefresh = vi.fn();

    render(
      <AircraftDetail
        details={details}
        onClose={vi.fn()}
        onSetComponent={onSetComponent}
        onSetReceiverSettings={vi.fn().mockResolvedValue(undefined)}
        onRefresh={onRefresh}
      />,
    );

    await waitFor(() => expect(mockedGetInventory).toHaveBeenCalledTimes(1));

    openCameraAddFlow();

    expect(screen.getByTestId('mock-add-gear-modal')).toBeInTheDocument();
    expect(screen.getByTestId('mock-add-gear-category')).toHaveTextContent('cameras');

    fireEvent.click(screen.getByRole('button', { name: 'submit-matching' }));

    await waitFor(() => {
      expect(onSetComponent).toHaveBeenCalledWith(
        expect.objectContaining({
          category: 'camera',
          newGear: expect.objectContaining({
            name: 'HD Camera',
            category: 'cameras',
          }),
        }),
      );
    });
    expect(onRefresh).toHaveBeenCalledTimes(1);
  });

  it('rejects mismatched category submissions from add gear modal', async () => {
    const onSetComponent = vi.fn().mockResolvedValue(undefined);

    render(
      <AircraftDetail
        details={details}
        onClose={vi.fn()}
        onSetComponent={onSetComponent}
        onSetReceiverSettings={vi.fn().mockResolvedValue(undefined)}
        onRefresh={vi.fn()}
      />,
    );

    await waitFor(() => expect(mockedGetInventory).toHaveBeenCalledTimes(1));
    openCameraAddFlow();

    fireEvent.click(screen.getByRole('button', { name: 'submit-mismatch' }));

    await waitFor(() => {
      expect(onSetComponent).not.toHaveBeenCalled();
    });
  });

  it('closes when clicking outside the modal content', async () => {
    const onClose = vi.fn();

    render(
      <AircraftDetail
        details={details}
        onClose={onClose}
        onSetComponent={vi.fn().mockResolvedValue(undefined)}
        onSetReceiverSettings={vi.fn().mockResolvedValue(undefined)}
        onRefresh={vi.fn()}
      />,
    );

    await waitFor(() => expect(mockedGetInventory).toHaveBeenCalledTimes(1));

    fireEvent.click(screen.getByTestId('aircraft-detail-overlay'));

    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('keeps the same modal height across tabs', async () => {
    const user = userEvent.setup();

    render(
      <AircraftDetail
        details={details}
        onClose={vi.fn()}
        onSetComponent={vi.fn().mockResolvedValue(undefined)}
        onSetReceiverSettings={vi.fn().mockResolvedValue(undefined)}
        onRefresh={vi.fn()}
      />,
    );

    await waitFor(() => expect(mockedGetInventory).toHaveBeenCalledTimes(1));

    const overlay = screen.getByTestId('aircraft-detail-overlay');
    const modal = overlay.firstElementChild;
    if (!(modal instanceof HTMLElement)) {
      throw new Error('Modal container not found');
    }

    expect(modal).toHaveClass('h-[90vh]');

    await user.click(screen.getByRole('button', { name: 'Receiver Settings' }));
    expect(modal).toHaveClass('h-[90vh]');

    await user.click(screen.getByRole('button', { name: 'Tuning' }));
    await waitFor(() => {
      expect(mockedGetAircraftTuning).toHaveBeenCalledWith(details.aircraft.id);
    });
    expect(modal).toHaveClass('h-[90vh]');
  });
});

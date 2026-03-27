/**
 * Integration tests: Aircraft
 * Covers:
 *   - Aircraft list loads and displays aircraft (scenario 11)
 *   - Create aircraft modal opens when Add Aircraft clicked (scenario 11)
 *   - Aircraft detail opens when aircraft card clicked (scenario 12)
 *   - Loading state displayed correctly
 *   - Error state displayed correctly
 */

import { describe, it, expect, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import { render, screen } from './test-utils';
import { AircraftPage } from '../components/AircraftPage';
import type { Aircraft } from '../aircraftTypes';

const shredder: Aircraft = {
  id: 'aircraft-1',
  userId: 'user-1',
  name: 'Shredder 5"',
  type: 'freestyle',
  hasImage: false,
  createdAt: '2026-02-01T00:00:00Z',
  updatedAt: '2026-02-01T00:00:00Z',
};

const racer: Aircraft = {
  id: 'aircraft-2',
  userId: 'user-1',
  name: 'Speedy Racer',
  type: 'racing',
  hasImage: false,
  createdAt: '2026-02-10T00:00:00Z',
  updatedAt: '2026-02-10T00:00:00Z',
};

function makeProps(overrides: Partial<{
  aircraftItems: Aircraft[];
  isAircraftLoading: boolean;
  aircraftError: string | null;
  onSelectAircraft: (a: Aircraft) => void;
  onEditAircraft: (a: Aircraft) => void;
  onDeleteAircraft: (a: Aircraft) => void;
  onAddAircraft: () => void;
}> = {}) {
  return {
    aircraftItems: [shredder, racer],
    isAircraftLoading: false,
    aircraftError: null,
    onSelectAircraft: vi.fn(),
    onEditAircraft: vi.fn(),
    onDeleteAircraft: vi.fn(),
    onAddAircraft: vi.fn(),
    ...overrides,
  };
}

describe('AircraftPage – aircraft integration', () => {
  it('displays aircraft names', () => {
    render(<AircraftPage {...makeProps()} />);

    expect(screen.getByText('Shredder 5"')).toBeInTheDocument();
    expect(screen.getByText('Speedy Racer')).toBeInTheDocument();
  });

  it('calls onAddAircraft when Add Aircraft button is clicked', async () => {
    const user = userEvent.setup();
    const onAddAircraft = vi.fn();

    render(<AircraftPage {...makeProps({ onAddAircraft })} />);

    // Desktop controls button
    const buttons = screen.getAllByRole('button', { name: /add aircraft/i });
    await user.click(buttons[0]);

    expect(onAddAircraft).toHaveBeenCalledTimes(1);
  });

  it('calls onSelectAircraft when aircraft card is clicked', async () => {
    const user = userEvent.setup();
    const onSelectAircraft = vi.fn();

    render(<AircraftPage {...makeProps({ onSelectAircraft })} />);

    // AircraftCard renders the aircraft name in an h3 inside a clickable div
    const nameHeading = screen.getByText('Shredder 5"');
    await user.click(nameHeading);

    expect(onSelectAircraft).toHaveBeenCalledWith(shredder);
  });

  it('shows loading spinner while aircraft are loading', () => {
    render(
      <AircraftPage
        {...makeProps({ aircraftItems: [], isAircraftLoading: true })}
      />,
    );

    expect(screen.getByText(/loading aircraft/i)).toBeInTheDocument();
  });

  it('shows error state when aircraft fails to load', () => {
    render(
      <AircraftPage
        {...makeProps({
          aircraftItems: [],
          aircraftError: 'Failed to load aircraft',
        })}
      />,
    );

    expect(screen.getByText(/failed to load aircraft/i)).toBeInTheDocument();
  });

  it('shows empty state when no aircraft and no error', () => {
    render(
      <AircraftPage
        {...makeProps({ aircraftItems: [], isAircraftLoading: false, aircraftError: null })}
      />,
    );

    expect(screen.getByText(/no aircraft yet/i)).toBeInTheDocument();
  });

  it('calls onEditAircraft when edit button is clicked', async () => {
    const user = userEvent.setup();
    const onEditAircraft = vi.fn();

    // Render with only one aircraft so we can target it unambiguously
    render(<AircraftPage {...makeProps({ onEditAircraft, aircraftItems: [shredder] })} />);

    // AircraftCard edit button has title="Edit"
    const editButton = screen.getByTitle('Edit');
    await user.click(editButton);

    expect(onEditAircraft).toHaveBeenCalledWith(shredder);
  });
});

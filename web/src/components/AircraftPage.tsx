import { useState } from 'react';
import type { Aircraft } from '../aircraftTypes';
import { AircraftList } from './AircraftList';
import { MobileFloatingControls } from './MobileFloatingControls';

interface AircraftPageProps {
  aircraftItems: Aircraft[];
  isAircraftLoading: boolean;
  aircraftError: string | null;
  onSelectAircraft: (aircraft: Aircraft) => void;
  onEditAircraft: (aircraft: Aircraft) => void;
  onDeleteAircraft: (aircraft: Aircraft) => void;
  onAddAircraft: () => void;
}

export function AircraftPage({
  aircraftItems,
  isAircraftLoading,
  aircraftError,
  onSelectAircraft,
  onEditAircraft,
  onDeleteAircraft,
  onAddAircraft,
}: AircraftPageProps) {
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);

  const controls = (
    <div className="px-4 md:px-6 py-4 border-b border-slate-800 bg-slate-900">
      <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
        <div>
          <h1 className="text-xl font-semibold text-white">My Aircraft</h1>
          <p className="text-sm text-slate-400">
            Manage your drones, components, and receiver settings
          </p>
        </div>
        <button
          onClick={() => {
            onAddAircraft();
            setIsMobileMenuOpen(false);
          }}
          className="w-full sm:w-auto px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white font-medium rounded-lg transition-colors flex items-center justify-center gap-2"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
          </svg>
          Add Aircraft
        </button>
      </div>
    </div>
  );

  return (
    <div className="relative flex-1 min-h-0 flex flex-col overflow-hidden">
      <div className="hidden md:block flex-shrink-0">{controls}</div>

      <AircraftList
        aircraft={aircraftItems}
        isLoading={isAircraftLoading}
        error={aircraftError}
        onSelect={onSelectAircraft}
        onEdit={onEditAircraft}
        onDelete={onDeleteAircraft}
        mobileTopInset
        onListScroll={() => setIsMobileMenuOpen((prev) => (prev ? false : prev))}
      />

      <MobileFloatingControls
        label="Aircraft Controls"
        isOpen={isMobileMenuOpen}
        onToggle={() => setIsMobileMenuOpen((prev) => !prev)}
      >
        {controls}
      </MobileFloatingControls>
    </div>
  );
}

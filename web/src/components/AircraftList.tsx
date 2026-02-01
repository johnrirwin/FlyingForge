import type { Aircraft } from '../aircraftTypes';
import { AircraftCard } from './AircraftCard';

interface AircraftListProps {
  aircraft: Aircraft[];
  isLoading: boolean;
  error: string | null;
  onSelect: (aircraft: Aircraft) => void;
  onEdit: (aircraft: Aircraft) => void;
  onDelete: (aircraft: Aircraft) => void;
}

export function AircraftList({
  aircraft,
  isLoading,
  error,
  onSelect,
  onEdit,
  onDelete,
}: AircraftListProps) {
  if (isLoading) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="flex flex-col items-center gap-3">
          <div className="animate-spin rounded-full h-8 w-8 border-2 border-slate-600 border-t-primary-500" />
          <p className="text-slate-400">Loading aircraft...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="text-center">
          <div className="text-red-400 mb-2">⚠️ {error}</div>
          <p className="text-slate-500">Please try again later</p>
        </div>
      </div>
    );
  }

  if (aircraft.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="text-center">
          <div className="text-6xl mb-4">🚁</div>
          <h3 className="text-xl font-medium text-white mb-2">No aircraft yet</h3>
          <p className="text-slate-400">
            Add your first drone to track its components and settings
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
        {aircraft.map((item) => (
          <AircraftCard
            key={item.id}
            aircraft={item}
            onSelect={onSelect}
            onEdit={onEdit}
            onDelete={onDelete}
          />
        ))}
      </div>
    </div>
  );
}

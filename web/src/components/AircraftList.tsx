import type { Aircraft } from '../aircraftTypes';
import { AIRCRAFT_TYPES } from '../aircraftTypes';
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

  // Group aircraft by type
  const aircraftByType = aircraft.reduce((acc, item) => {
    const type = item.type;
    if (!acc[type]) {
      acc[type] = [];
    }
    acc[type].push(item);
    return acc;
  }, {} as Record<string, Aircraft[]>);

  // Sort types by the order in AIRCRAFT_TYPES
  const sortedTypes = AIRCRAFT_TYPES
    .filter(t => aircraftByType[t.value])
    .map(t => ({
      value: t.value,
      label: t.label,
      icon: t.icon,
      items: aircraftByType[t.value],
    }));

  return (
    <div className="flex-1 overflow-y-auto p-4 md:p-6">
      <div className="space-y-6 md:space-y-8">
        {sortedTypes.map(type => (
          <section key={type.value}>
            <div className="flex items-center gap-3 mb-3 md:mb-4">
              <span className="text-xl">{type.icon}</span>
              <h2 className="text-base md:text-lg font-semibold text-white">{type.label}</h2>
              <span className="px-2 py-0.5 bg-slate-700 rounded-full text-xs text-slate-400">
                {type.items.length}
              </span>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3 md:gap-4">
              {type.items.map((item) => (
                <AircraftCard
                  key={item.id}
                  aircraft={item}
                  onSelect={onSelect}
                  onEdit={onEdit}
                  onDelete={onDelete}
                />
              ))}
            </div>
          </section>
        ))}
      </div>
    </div>
  );
}

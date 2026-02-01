import type { Aircraft } from '../aircraftTypes';
import { AIRCRAFT_TYPES } from '../aircraftTypes';
import { getAircraftImageUrl } from '../aircraftApi';

interface AircraftCardProps {
  aircraft: Aircraft;
  onSelect: (aircraft: Aircraft) => void;
  onEdit: (aircraft: Aircraft) => void;
  onDelete: (aircraft: Aircraft) => void;
}

export function AircraftCard({ aircraft, onSelect, onEdit, onDelete }: AircraftCardProps) {
  const aircraftType = AIRCRAFT_TYPES.find(t => t.value === aircraft.type);

  return (
    <div 
      className="bg-slate-800 border border-slate-700 rounded-xl p-4 hover:border-slate-600 transition-all cursor-pointer"
      onClick={() => onSelect(aircraft)}
    >
      <div className="flex gap-4">
        {/* Image */}
        <div className="flex-shrink-0 w-24 h-24 rounded-lg overflow-hidden bg-slate-700">
          {aircraft.hasImage ? (
            <img
              src={getAircraftImageUrl(aircraft.id)}
              alt={aircraft.name}
              className="w-full h-full object-cover"
              loading="lazy"
              onError={(e) => {
                (e.target as HTMLImageElement).style.display = 'none';
              }}
            />
          ) : (
            <div className="w-full h-full flex items-center justify-center text-4xl">
              {aircraftType?.icon || '🚁'}
            </div>
          )}
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          {/* Header */}
          <div className="flex items-start justify-between gap-2 mb-1">
            <div className="flex items-center gap-2 flex-wrap">
              <span className="px-2 py-0.5 bg-slate-700 text-slate-300 rounded text-xs">
                {aircraftType?.label || aircraft.type}
              </span>
            </div>
            {/* Actions - stop propagation so clicking doesn't select */}
            <div 
              className="flex items-center gap-1"
              onClick={(e) => e.stopPropagation()}
            >
              <button
                onClick={() => onEdit(aircraft)}
                className="p-1.5 text-slate-400 hover:text-white hover:bg-slate-700 rounded transition-colors"
                title="Edit"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                </svg>
              </button>
              <button
                onClick={() => onDelete(aircraft)}
                className="p-1.5 text-slate-400 hover:text-red-400 hover:bg-slate-700 rounded transition-colors"
                title="Delete"
              >
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </button>
            </div>
          </div>

          {/* Title */}
          <h3 className="text-white font-medium mb-1">
            {aircraft.name}
          </h3>

          {/* Nickname */}
          {aircraft.nickname && (
            <p className="text-primary-400 text-sm mb-1">
              "{aircraft.nickname}"
            </p>
          )}

          {/* Description */}
          {aircraft.description && (
            <p className="text-slate-500 text-sm line-clamp-2">
              {aircraft.description}
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

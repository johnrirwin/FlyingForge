// Read-only aircraft detail modal for public pilot profiles
// Shows components and sanitized ELRS settings (no sensitive data)

import { useState } from 'react';
import type { AircraftPublic, ComponentCategory } from '../socialTypes';
import { getPublicAircraftImageUrl } from '../socialApi';

interface PublicAircraftModalProps {
  aircraft: AircraftPublic;
  onClose: () => void;
}

type ViewMode = 'components' | 'elrs';

// Component category display info
const COMPONENT_INFO: Record<ComponentCategory, { label: string; icon: string }> = {
  fc: { label: 'Flight Controller', icon: '🧠' },
  esc: { label: 'ESC', icon: '⚡' },
  elrs_module: { label: 'ELRS Receiver', icon: '📡' },
  vtx: { label: 'Video Transmitter', icon: '📺' },
  motors: { label: 'Motors', icon: '🔄' },
  camera: { label: 'Camera', icon: '📷' },
  frame: { label: 'Frame', icon: '🏗️' },
  propellers: { label: 'Propellers', icon: '🍃' },
  antenna: { label: 'Antenna', icon: '📶' },
};

// All component categories in display order
const CATEGORY_ORDER: ComponentCategory[] = [
  'fc', 'esc', 'elrs_module', 'vtx', 'motors', 'camera', 'frame', 'propellers', 'antenna'
];

export function PublicAircraftModal({ aircraft, onClose }: PublicAircraftModalProps) {
  const [viewMode, setViewMode] = useState<ViewMode>('components');
  const [imageError, setImageError] = useState(false);

  const hasComponents = aircraft.components && aircraft.components.length > 0;
  const hasElrsSettings = aircraft.elrsSettings && 
    Object.values(aircraft.elrsSettings).some(v => v);

  // Get component by category
  const getComponentByCategory = (category: ComponentCategory) => {
    return aircraft.components?.find(c => c.category === category);
  };

  // Format aircraft type for display
  const formatType = (type?: string) => {
    if (!type) return 'Aircraft';
    return type.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
  };

  return (
    <div 
      className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div className="bg-slate-800 rounded-xl w-full max-w-2xl max-h-[90vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-slate-700">
          <div className="flex items-center gap-4">
            {/* Aircraft image/icon */}
            <div className="w-16 h-16 rounded-lg overflow-hidden bg-slate-700 flex-shrink-0">
              {aircraft.hasImage && !imageError ? (
                <img
                  src={getPublicAircraftImageUrl(aircraft.id)}
                  alt={aircraft.name}
                  className="w-full h-full object-cover"
                  onError={() => setImageError(true)}
                />
              ) : (
                <div className="w-full h-full flex items-center justify-center">
                  <svg className="w-8 h-8 text-slate-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
                  </svg>
                </div>
              )}
            </div>
            <div>
              <h2 className="text-lg font-semibold text-white">{aircraft.name}</h2>
              {aircraft.nickname && (
                <p className="text-primary-400 text-sm">"{aircraft.nickname}"</p>
              )}
              <span className="inline-block mt-1 px-2 py-0.5 bg-slate-700 text-slate-300 rounded text-xs">
                {formatType(aircraft.type)}
              </span>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-2 text-slate-400 hover:text-white transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Tabs */}
        <div className="flex border-b border-slate-700">
          <button
            onClick={() => setViewMode('components')}
            className={`flex-1 px-4 py-3 text-sm font-medium transition-colors ${
              viewMode === 'components'
                ? 'text-primary-400 border-b-2 border-primary-400'
                : 'text-slate-400 hover:text-white'
            }`}
          >
            Components
          </button>
          <button
            onClick={() => setViewMode('elrs')}
            className={`flex-1 px-4 py-3 text-sm font-medium transition-colors ${
              viewMode === 'elrs'
                ? 'text-primary-400 border-b-2 border-primary-400'
                : 'text-slate-400 hover:text-white'
            }`}
          >
            ELRS Settings
            {hasElrsSettings && (
              <span className="ml-2 px-1.5 py-0.5 bg-green-500/20 text-green-400 text-[10px] rounded">
                Safe View
              </span>
            )}
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4">
          {viewMode === 'components' && (
            <div className="space-y-3">
              {CATEGORY_ORDER.map((category) => {
                const component = getComponentByCategory(category);
                const info = COMPONENT_INFO[category];

                return (
                  <div
                    key={category}
                    className="bg-slate-700/50 border border-slate-700 rounded-lg p-3"
                  >
                    <div className="flex items-center gap-3">
                      <div className="w-10 h-10 bg-slate-600 rounded-lg flex items-center justify-center text-xl">
                        {info.icon}
                      </div>
                      <div className="flex-1 min-w-0">
                        <h4 className="text-white font-medium">{info.label}</h4>
                        {component ? (
                          <p className="text-slate-300 text-sm truncate">
                            {component.manufacturer && (
                              <span className="text-slate-500">{component.manufacturer} </span>
                            )}
                            {component.name || 'Assigned'}
                          </p>
                        ) : (
                          <p className="text-slate-500 text-sm italic">Not assigned</p>
                        )}
                      </div>
                      {component && (
                        <div className="flex-shrink-0">
                          <span className="px-2 py-1 bg-green-500/20 text-green-400 text-xs rounded">
                            Installed
                          </span>
                        </div>
                      )}
                    </div>
                  </div>
                );
              })}
              
              {!hasComponents && (
                <div className="text-center py-8 text-slate-500">
                  <p>No components have been added to this aircraft yet.</p>
                </div>
              )}
            </div>
          )}

          {viewMode === 'elrs' && (
            <div className="space-y-4">
              {hasElrsSettings ? (
                <>
                  {/* Safe view notice */}
                  <div className="bg-green-500/10 border border-green-500/30 rounded-lg p-3">
                    <div className="flex items-center gap-2 text-green-400 text-sm">
                      <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                      </svg>
                      <span>Safe View - Sensitive data (bind phrase, model match) is hidden</span>
                    </div>
                  </div>

                  {/* ELRS Settings Display */}
                  <div className="bg-slate-700/50 border border-slate-700 rounded-lg p-4 space-y-3">
                    <h4 className="text-white font-medium mb-4">ELRS Configuration</h4>
                    
                    <div className="grid grid-cols-2 gap-4">
                      <ElrsField label="Receiver Model" value={aircraft.elrsSettings?.receiverModel} />
                      <ElrsField label="Packet Rate" value={aircraft.elrsSettings?.packetRate} />
                      <ElrsField label="Telemetry Ratio" value={aircraft.elrsSettings?.telemetryRatio} />
                      <ElrsField label="TX Power" value={aircraft.elrsSettings?.outputPower} />
                      <ElrsField label="Switch Mode" value={aircraft.elrsSettings?.switchMode} />
                      <ElrsField label="Regulatory Domain" value={aircraft.elrsSettings?.regulatoryDomain} />
                      <ElrsField label="Firmware Version" value={aircraft.elrsSettings?.firmwareVersion} />
                      <ElrsField label="RX Protocol" value={aircraft.elrsSettings?.rxProtocol} />
                    </div>
                  </div>

                  {/* Hidden fields notice */}
                  <div className="text-center text-slate-500 text-sm py-2">
                    <p>Bind phrase, model match, and UID are not shown for security.</p>
                  </div>
                </>
              ) : (
                <div className="text-center py-8 text-slate-500">
                  <svg className="w-12 h-12 mx-auto mb-3 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
                  </svg>
                  <p>No ELRS settings configured for this aircraft.</p>
                </div>
              )}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="p-4 border-t border-slate-700 text-center text-slate-500 text-sm">
          Viewing {aircraft.name}'s build details
        </div>
      </div>
    </div>
  );
}

// Helper component for displaying ELRS fields
function ElrsField({ label, value }: { label: string; value?: string }) {
  return (
    <div>
      <label className="block text-xs font-medium text-slate-400 mb-1">
        {label}
      </label>
      <div className="px-3 py-2 bg-slate-600 border border-slate-500 rounded-lg text-white text-sm">
        {value || <span className="text-slate-500 italic">Not set</span>}
      </div>
    </div>
  );
}

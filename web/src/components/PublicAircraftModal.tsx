// Read-only aircraft detail modal for public pilot profiles
// Shows components, sanitized receiver settings (no sensitive data), and tuning data

import { useCallback, useEffect, useRef, useState } from 'react';
import { getGearCatalogItem } from '../gearCatalogApi';
import type { GearCatalogItem } from '../gearCatalogTypes';
import type { AircraftPublic, ComponentCategory } from '../socialTypes';
import { useAuth } from '../hooks/useAuth';
import { AircraftImage } from './AircraftImage';
import { GearDetailModal } from './GearDetailModal';

interface PublicAircraftModalProps {
  aircraft: AircraftPublic;
  onClose: () => void;
  onAddToInventory?: (item: GearCatalogItem) => void;
}

type ViewMode = 'components' | 'receiver' | 'tuning';

// Component category display info
const COMPONENT_INFO: Record<ComponentCategory, { label: string; icon: string }> = {
  fc: { label: 'Flight Controller', icon: '🧠' },
  esc: { label: 'ESC', icon: '⚡' },
  aio: { label: 'AIO (FC/ESC)', icon: '🔌' },
  stack: { label: 'FC/ESC Stack', icon: '🧩' },
  receiver: { label: 'Receiver', icon: '📡' },
  vtx: { label: 'Video Transmitter', icon: '📺' },
  motors: { label: 'Motors', icon: '🔄' },
  camera: { label: 'Camera', icon: '📷' },
  frame: { label: 'Frame', icon: '🏗️' },
  propellers: { label: 'Propellers', icon: '🍃' },
  antenna: { label: 'Antenna', icon: '📶' },
  gps: { label: 'GPS', icon: '🛰️' },
};

// All component categories in display order
const CATEGORY_ORDER: ComponentCategory[] = [
  'fc', 'esc', 'aio', 'stack', 'receiver', 'vtx', 'motors', 'camera', 'frame', 'propellers', 'antenna', 'gps'
];

export function PublicAircraftModal({ aircraft, onClose, onAddToInventory }: PublicAircraftModalProps) {
  const { isAuthenticated } = useAuth();
  const [viewMode, setViewMode] = useState<ViewMode>('components');
  const [selectedCatalogItem, setSelectedCatalogItem] = useState<GearCatalogItem | null>(null);
  const [isDetailModalOpen, setIsDetailModalOpen] = useState(false);
  const [loadingCatalogItemId, setLoadingCatalogItemId] = useState<string | null>(null);
  const [componentDetailError, setComponentDetailError] = useState<string | null>(null);
  const catalogItemsRef = useRef<Record<string, GearCatalogItem>>({});
  const pendingCatalogRequestsRef = useRef<Map<string, Promise<GearCatalogItem>>>(new Map());

  const hasComponents = aircraft.components && aircraft.components.length > 0;
  const hasReceiverSettings = aircraft.receiverSettings && 
    Object.values(aircraft.receiverSettings).some(v => v);
  const hasTuning = aircraft.tuning && aircraft.tuning.parsedTuning;

  // Get component by category
  const getComponentByCategory = (category: ComponentCategory) => {
    return aircraft.components?.find(c => c.category === category);
  };

  // Format aircraft type for display
  const formatType = (type?: string) => {
    if (!type) return 'Aircraft';
    return type.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
  };

  useEffect(() => {
    catalogItemsRef.current = {};
    pendingCatalogRequestsRef.current.clear();
    setSelectedCatalogItem(null);
    setIsDetailModalOpen(false);
    setLoadingCatalogItemId(null);
    setComponentDetailError(null);
  }, [aircraft.id]);

  const loadCatalogItemDetails = useCallback(async (catalogItemId: string): Promise<GearCatalogItem> => {
    const catalogID = catalogItemId.trim();
    if (!catalogID) {
      throw new Error('Missing catalog item ID');
    }

    const cached = catalogItemsRef.current[catalogID];
    if (cached) {
      return cached;
    }

    const pending = pendingCatalogRequestsRef.current.get(catalogID);
    if (pending) {
      return pending;
    }

    const request = getGearCatalogItem(catalogID)
      .then((item) => {
        if (catalogItemsRef.current[catalogID]) {
          return item;
        }
        catalogItemsRef.current = { ...catalogItemsRef.current, [catalogID]: item };
        return item;
      })
      .finally(() => {
        pendingCatalogRequestsRef.current.delete(catalogID);
      });

    pendingCatalogRequestsRef.current.set(catalogID, request);
    return request;
  }, []);

  const handleOpenComponentDetails = useCallback(async (catalogItemID?: string) => {
    const catalogID = catalogItemID?.trim();
    if (!catalogID) {
      return;
    }

    setComponentDetailError(null);
    setLoadingCatalogItemId(catalogID);

    try {
      const detail = await loadCatalogItemDetails(catalogID);
      setSelectedCatalogItem(detail);
      setIsDetailModalOpen(true);
    } catch (err) {
      setComponentDetailError(err instanceof Error ? err.message : 'Failed to load component details');
    } finally {
      setLoadingCatalogItemId((current) => (current === catalogID ? null : current));
    }
  }, [loadCatalogItemDetails]);

  const handleClosePartDetails = useCallback(() => {
    setIsDetailModalOpen(false);
    setSelectedCatalogItem(null);
  }, []);

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
            <AircraftImage
              aircraftId={aircraft.id}
              aircraftName={aircraft.name}
              hasImage={aircraft.hasImage}
              className="w-16 h-16 rounded-lg object-cover"
            />
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
            onClick={() => setViewMode('tuning')}
            className={`flex-1 px-4 py-3 text-sm font-medium transition-colors ${
              viewMode === 'tuning'
                ? 'text-primary-400 border-b-2 border-primary-400'
                : 'text-slate-400 hover:text-white'
            }`}
          >
            Tuning
            {hasTuning && (
              <span className="ml-2 px-1.5 py-0.5 bg-blue-500/20 text-blue-400 text-[10px] rounded">
                PIDs
              </span>
            )}
          </button>
          <button
            onClick={() => setViewMode('receiver')}
            className={`flex-1 px-4 py-3 text-sm font-medium transition-colors ${
              viewMode === 'receiver'
                ? 'text-primary-400 border-b-2 border-primary-400'
                : 'text-slate-400 hover:text-white'
            }`}
          >
            Receiver
            {hasReceiverSettings && (
              <span className="ml-2 px-1.5 py-0.5 bg-green-500/20 text-green-400 text-[10px] rounded">
                Safe
              </span>
            )}
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4">
          {viewMode === 'components' && (
            <div className="space-y-3">
              {componentDetailError && (
                <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-300">
                  {componentDetailError}
                </div>
              )}
              {CATEGORY_ORDER.map((category) => {
                const component = getComponentByCategory(category);
                const info = COMPONENT_INFO[category];
                const catalogID = component?.catalogId?.trim() || '';
                const isInteractive = Boolean(catalogID);
                const isLoading = Boolean(loadingCatalogItemId && loadingCatalogItemId === catalogID);
                const content = (
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 bg-slate-600 rounded-lg flex items-center justify-center text-xl">
                      {info.icon}
                    </div>
                    <div className="flex-1 min-w-0">
                      <h4 className="text-white font-medium">{info.label}</h4>
                      {component ? (
                        <>
                          <p className="text-slate-300 text-sm truncate">
                            {component.manufacturer && (
                              <span className="text-slate-500">{component.manufacturer} </span>
                            )}
                            {component.name || 'Assigned'}
                          </p>
                          {isInteractive ? (
                            <p className="mt-1 text-xs text-primary-300">
                              {isLoading ? 'Loading details...' : 'View details'}
                            </p>
                          ) : (
                            <p className="mt-1 text-xs text-slate-500">
                              Details unavailable
                            </p>
                          )}
                        </>
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
                );

                if (isInteractive) {
                  return (
                    <button
                      key={category}
                      type="button"
                      onClick={() => handleOpenComponentDetails(catalogID)}
                      disabled={isLoading}
                      className="w-full bg-slate-700/50 border border-slate-700 rounded-lg p-3 text-left transition hover:border-primary-500/50 hover:bg-slate-700/70 disabled:cursor-wait disabled:opacity-70"
                      aria-label={`View details for ${component?.name || info.label}`}
                    >
                      {content}
                    </button>
                  );
                }

                return (
                  <div
                    key={category}
                    className="bg-slate-700/50 border border-slate-700 rounded-lg p-3"
                  >
                    {content}
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

          {viewMode === 'tuning' && (
            <div className="space-y-4">
              {hasTuning ? (
                <>
                  {/* Firmware Info */}
                  <div className="bg-slate-700/50 border border-slate-700 rounded-lg p-4">
                    <div className="flex items-center justify-between mb-3">
                      <h4 className="text-white font-medium">Firmware</h4>
                      {aircraft.tuning?.snapshotDate && (
                        <span className="text-xs text-slate-500">
                          Updated {new Date(aircraft.tuning.snapshotDate).toLocaleDateString()}
                        </span>
                      )}
                    </div>
                    <div className="grid grid-cols-2 gap-3 text-sm">
                      <div>
                        <span className="text-slate-400">Version:</span>
                        <span className="ml-2 text-white">
                          {aircraft.tuning?.firmwareName} {aircraft.tuning?.firmwareVersion}
                        </span>
                      </div>
                      {aircraft.tuning?.boardName && (
                        <div>
                          <span className="text-slate-400">Board:</span>
                          <span className="ml-2 text-white">{aircraft.tuning.boardName}</span>
                        </div>
                      )}
                    </div>
                  </div>

                  {/* PID Display */}
                  {aircraft.tuning?.parsedTuning?.pids && (
                    <div className="bg-slate-700/50 border border-slate-700 rounded-lg p-4">
                      <h4 className="text-white font-medium mb-3">PIDs</h4>
                      <div className="overflow-x-auto">
                        <table className="w-full text-sm">
                          <thead>
                            <tr className="text-slate-400 text-left">
                              <th className="pb-2">Axis</th>
                              <th className="pb-2 text-center">P</th>
                              <th className="pb-2 text-center">I</th>
                              <th className="pb-2 text-center">D</th>
                              <th className="pb-2 text-center">F</th>
                            </tr>
                          </thead>
                          <tbody className="text-white">
                            <tr>
                              <td className="py-1 text-slate-300">Roll</td>
                              <td className="py-1 text-center font-mono">{aircraft.tuning.parsedTuning.pids.roll?.p ?? '-'}</td>
                              <td className="py-1 text-center font-mono">{aircraft.tuning.parsedTuning.pids.roll?.i ?? '-'}</td>
                              <td className="py-1 text-center font-mono">{aircraft.tuning.parsedTuning.pids.roll?.d ?? '-'}</td>
                              <td className="py-1 text-center font-mono">{aircraft.tuning.parsedTuning.pids.roll?.ff ?? '-'}</td>
                            </tr>
                            <tr>
                              <td className="py-1 text-slate-300">Pitch</td>
                              <td className="py-1 text-center font-mono">{aircraft.tuning.parsedTuning.pids.pitch?.p ?? '-'}</td>
                              <td className="py-1 text-center font-mono">{aircraft.tuning.parsedTuning.pids.pitch?.i ?? '-'}</td>
                              <td className="py-1 text-center font-mono">{aircraft.tuning.parsedTuning.pids.pitch?.d ?? '-'}</td>
                              <td className="py-1 text-center font-mono">{aircraft.tuning.parsedTuning.pids.pitch?.ff ?? '-'}</td>
                            </tr>
                            <tr>
                              <td className="py-1 text-slate-300">Yaw</td>
                              <td className="py-1 text-center font-mono">{aircraft.tuning.parsedTuning.pids.yaw?.p ?? '-'}</td>
                              <td className="py-1 text-center font-mono">{aircraft.tuning.parsedTuning.pids.yaw?.i ?? '-'}</td>
                              <td className="py-1 text-center font-mono">{aircraft.tuning.parsedTuning.pids.yaw?.d ?? '-'}</td>
                              <td className="py-1 text-center font-mono">{aircraft.tuning.parsedTuning.pids.yaw?.ff ?? '-'}</td>
                            </tr>
                          </tbody>
                        </table>
                      </div>
                    </div>
                  )}

                  {/* Rates Display - matches AircraftDetail layout */}
                  {aircraft.tuning?.parsedTuning?.rates && (
                    <div className="bg-slate-700/50 border border-slate-700 rounded-lg p-4">
                      <div className="flex items-center justify-between mb-3">
                        <h4 className="text-white font-medium">Rates</h4>
                        {aircraft.tuning.parsedTuning.rates.rateType && (
                          <span className="text-xs bg-slate-600 text-slate-300 px-2 py-0.5 rounded">
                            {aircraft.tuning.parsedTuning.rates.rateType}
                          </span>
                        )}
                      </div>
                      <div className="overflow-x-auto">
                        <table className="w-full text-sm">
                          <thead>
                            <tr className="text-slate-400 text-left">
                              <th className="pb-2"></th>
                              <th className="pb-2 text-center">Roll</th>
                              <th className="pb-2 text-center">Pitch</th>
                              <th className="pb-2 text-center">Yaw</th>
                            </tr>
                          </thead>
                          <tbody className="text-white">
                            <tr>
                              <td className="py-1 text-slate-300">RC Rate</td>
                              <td className="py-1 text-center font-mono">{aircraft.tuning.parsedTuning.rates.rcRates?.roll ?? '-'}</td>
                              <td className="py-1 text-center font-mono">{aircraft.tuning.parsedTuning.rates.rcRates?.pitch ?? '-'}</td>
                              <td className="py-1 text-center font-mono">{aircraft.tuning.parsedTuning.rates.rcRates?.yaw ?? '-'}</td>
                            </tr>
                            <tr>
                              <td className="py-1 text-slate-300">Super Rate</td>
                              <td className="py-1 text-center font-mono">{aircraft.tuning.parsedTuning.rates.superRates?.roll ?? '-'}</td>
                              <td className="py-1 text-center font-mono">{aircraft.tuning.parsedTuning.rates.superRates?.pitch ?? '-'}</td>
                              <td className="py-1 text-center font-mono">{aircraft.tuning.parsedTuning.rates.superRates?.yaw ?? '-'}</td>
                            </tr>
                            <tr>
                              <td className="py-1 text-slate-300">RC Expo</td>
                              <td className="py-1 text-center font-mono">{aircraft.tuning.parsedTuning.rates.rcExpo?.roll ?? '-'}</td>
                              <td className="py-1 text-center font-mono">{aircraft.tuning.parsedTuning.rates.rcExpo?.pitch ?? '-'}</td>
                              <td className="py-1 text-center font-mono">{aircraft.tuning.parsedTuning.rates.rcExpo?.yaw ?? '-'}</td>
                            </tr>
                          </tbody>
                        </table>
                      </div>
                    </div>
                  )}

                  {/* Filters Display */}
                  {aircraft.tuning?.parsedTuning?.filters && (
                    <div className="bg-slate-700/50 border border-slate-700 rounded-lg p-4">
                      <h4 className="text-white font-medium mb-3">Filters</h4>
                      <div className="grid grid-cols-2 gap-3 text-sm">
                        {aircraft.tuning.parsedTuning.filters.gyroLowpassHz !== undefined && (
                          <div>
                            <span className="text-slate-400">Gyro Lowpass:</span>
                            <span className="ml-2 text-white">{aircraft.tuning.parsedTuning.filters.gyroLowpassHz} Hz</span>
                          </div>
                        )}
                        {aircraft.tuning.parsedTuning.filters.dtermLowpassHz !== undefined && (
                          <div>
                            <span className="text-slate-400">D-Term Lowpass:</span>
                            <span className="ml-2 text-white">{aircraft.tuning.parsedTuning.filters.dtermLowpassHz} Hz</span>
                          </div>
                        )}
                        {aircraft.tuning.parsedTuning.filters.dynNotchEnabled !== undefined && (
                          <div>
                            <span className="text-slate-400">Dynamic Notch:</span>
                            <span className={`ml-2 ${aircraft.tuning.parsedTuning.filters.dynNotchEnabled ? 'text-green-400' : 'text-slate-500'}`}>
                              {aircraft.tuning.parsedTuning.filters.dynNotchEnabled ? 'Enabled' : 'Disabled'}
                            </span>
                          </div>
                        )}
                        {aircraft.tuning.parsedTuning.filters.rpmFilterEnabled !== undefined && (
                          <div>
                            <span className="text-slate-400">RPM Filter:</span>
                            <span className={`ml-2 ${aircraft.tuning.parsedTuning.filters.rpmFilterEnabled ? 'text-green-400' : 'text-slate-500'}`}>
                              {aircraft.tuning.parsedTuning.filters.rpmFilterEnabled ? 'Enabled' : 'Disabled'}
                            </span>
                          </div>
                        )}
                      </div>
                    </div>
                  )}
                </>
              ) : (
                <div className="text-center py-8 text-slate-500">
                  <svg className="w-12 h-12 mx-auto mb-3 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                  </svg>
                  <p>No tuning data uploaded for this aircraft yet.</p>
                </div>
              )}
            </div>
          )}

          {viewMode === 'receiver' && (
            <div className="space-y-4">
              {hasReceiverSettings ? (
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

                  {/* Receiver Settings Display */}
                  <div className="bg-slate-700/50 border border-slate-700 rounded-lg p-4 space-y-3">
                    <h4 className="text-white font-medium mb-4">Receiver Configuration</h4>
                    
                    <div className="grid grid-cols-2 gap-4">
                      <ReceiverField 
                        label="Packet Rate" 
                        value={aircraft.receiverSettings?.rate ? `${aircraft.receiverSettings.rate} Hz` : undefined} 
                      />
                      <ReceiverField 
                        label="Telemetry Ratio" 
                        value={aircraft.receiverSettings?.tlm !== undefined 
                          ? aircraft.receiverSettings.tlm === 0 ? 'Off' : `1:${aircraft.receiverSettings.tlm}`
                          : undefined
                        } 
                      />
                      <ReceiverField 
                        label="TX Power" 
                        value={aircraft.receiverSettings?.power ? `${aircraft.receiverSettings.power} mW` : undefined} 
                      />
                      <ReceiverField label="Device Name" value={aircraft.receiverSettings?.deviceName} />
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
                  <p>No receiver settings configured for this aircraft.</p>
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
      {selectedCatalogItem && (
        <GearDetailModal
          item={selectedCatalogItem}
          isOpen={isDetailModalOpen}
          onClose={handleClosePartDetails}
          onAddToInventory={onAddToInventory}
          isAuthenticated={isAuthenticated}
        />
      )}
    </div>
  );
}

// Helper component for displaying receiver fields
function ReceiverField({ label, value }: { label: string; value?: string }) {
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

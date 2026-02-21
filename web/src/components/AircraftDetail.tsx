import { useState, useEffect } from 'react';
import type { 
  AircraftDetailsResponse, 
  AircraftComponent, 
  ComponentCategory,
  ReceiverConfig,
  SetComponentParams 
} from '../aircraftTypes';
import { AIRCRAFT_TYPES, COMPONENT_CATEGORIES } from '../aircraftTypes';
import type { InventoryItem, AddInventoryParams } from '../equipmentTypes';
import { getInventory } from '../equipmentApi';
import { getAircraftImageUrl } from '../aircraftApi';
import { getAircraftTuning, createTuningSnapshot } from '../fcConfigApi';
import type { AircraftTuningResponse, PIDProfile, RateProfile, FilterSettings } from '../fcConfigTypes';
import { trackEvent } from '../hooks/useGoogleAnalytics';
import { AddGearModal } from './AddGearModal';

interface AircraftDetailProps {
  details: AircraftDetailsResponse;
  onClose: () => void;
  onSetComponent: (params: SetComponentParams) => Promise<void>;
  onSetReceiverSettings: (settings: ReceiverConfig) => Promise<void>;
  onRefresh: () => void;
}

type ViewMode = 'components' | 'receiver' | 'tuning';

export function AircraftDetail({
  details,
  onClose,
  onSetComponent,
  onSetReceiverSettings,
  onRefresh,
}: AircraftDetailProps) {
  const [viewMode, setViewMode] = useState<ViewMode>('components');
  const [inventoryItems, setInventoryItems] = useState<InventoryItem[]>([]);
  const [isLoadingInventory, setIsLoadingInventory] = useState(true);
  const [selectedCategory, setSelectedCategory] = useState<ComponentCategory | null>(null);
  const [addModalCategory, setAddModalCategory] = useState<ComponentCategory | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  
  // Receiver settings state
  const [receiverSettings, setReceiverSettings] = useState<ReceiverConfig>(
    details.receiverSettings?.settings || {}
  );
  const [isSavingReceiver, setIsSavingReceiver] = useState(false);
  const [receiverSaved, setReceiverSaved] = useState(false);

  // Tuning state
  const [tuningData, setTuningData] = useState<AircraftTuningResponse | null>(null);
  const [isLoadingTuning, setIsLoadingTuning] = useState(false);
  const [showCliUpload, setShowCliUpload] = useState(false);
  const [cliDump, setCliDump] = useState('');
  const [diffBackup, setDiffBackup] = useState('');
  const [isUploadingTuning, setIsUploadingTuning] = useState(false);
  const [uploadMode, setUploadMode] = useState<'dump' | 'backup'>('dump');

  const { aircraft, components } = details;
  const aircraftType = AIRCRAFT_TYPES.find(t => t.value === aircraft.type);

  // Load inventory items
  useEffect(() => {
    const loadInventory = async () => {
      setIsLoadingInventory(true);
      try {
        const response = await getInventory({ limit: 500 });
        setInventoryItems(response.items || []);
      } catch (err) {
        console.error('Failed to load inventory:', err);
      } finally {
        setIsLoadingInventory(false);
      }
    };
    loadInventory();
  }, []);

  // Load tuning data when switching to tuning tab
  useEffect(() => {
    if (viewMode === 'tuning' && !tuningData) {
      loadTuningData();
    }
  }, [viewMode]);

  const loadTuningData = async () => {
    setIsLoadingTuning(true);
    try {
      const data = await getAircraftTuning(aircraft.id);
      setTuningData(data);
    } catch (err) {
      console.error('Failed to load tuning data:', err);
    } finally {
      setIsLoadingTuning(false);
    }
  };

  const handleUploadTuning = async () => {
    // Validate based on mode
    if (uploadMode === 'dump' && !cliDump.trim()) return;
    if (uploadMode === 'backup' && !diffBackup.trim()) return;
    
    setIsUploadingTuning(true);
    try {
      if (uploadMode === 'backup') {
        // Diff backup only mode - update existing or create new with just backup
        await createTuningSnapshot(aircraft.id, { 
          diffBackup: diffBackup.trim(),
          diffBackupOnly: tuningData?.hasTuning ? true : false,
        });
      } else {
        // Dump mode
        await createTuningSnapshot(aircraft.id, { 
          rawCliDump: cliDump,
        });
      }
      setCliDump('');
      setDiffBackup('');
      setShowCliUpload(false);
      setUploadMode('dump');
      // Track FC config upload for GA4 conversions
      trackEvent('fc_config_uploaded', { mode: uploadMode });
      // Reload tuning data
      const data = await getAircraftTuning(aircraft.id);
      setTuningData(data);
    } catch (err) {
      console.error('Failed to upload tuning:', err);
      alert(uploadMode === 'backup' 
        ? 'Failed to update backup.' 
        : 'Failed to upload tuning data. Please check the CLI dump format.');
    } finally {
      setIsUploadingTuning(false);
    }
  };

  // Get component by category
  const getComponentByCategory = (category: ComponentCategory): AircraftComponent | undefined => {
    return components.find(c => c.category === category);
  };

  // Get inventory item by ID
  const getInventoryItemById = (id: string): InventoryItem | undefined => {
    return inventoryItems.find(i => i.id === id);
  };

  // Get available inventory items for a component category
  const getAvailableItems = (category: ComponentCategory): InventoryItem[] => {
    const componentCat = COMPONENT_CATEGORIES.find(c => c.value === category);
    if (!componentCat) return [];
    return inventoryItems.filter(i => i.category === componentCat.equipmentCategory);
  };

  // Handle component assignment
  const handleAssignComponent = async (category: ComponentCategory, inventoryItemId: string) => {
    setIsSubmitting(true);
    try {
      await onSetComponent({ category, inventoryItemId });
      onRefresh();
    } catch (err) {
      console.error('Failed to assign component:', err);
    } finally {
      setIsSubmitting(false);
      setSelectedCategory(null);
    }
  };

  // Handle auto-add gear
  const handleAutoAddGear = async (category: ComponentCategory, newGear: AddInventoryParams) => {
    setIsSubmitting(true);
    try {
      await onSetComponent({ category, newGear });
      onRefresh();
      // Reload inventory to get the new item
      const response = await getInventory({ limit: 500 });
      setInventoryItems(response.items || []);
    } catch (err) {
      console.error('Failed to auto-add gear:', err);
      throw err;
    } finally {
      setIsSubmitting(false);
      setSelectedCategory(null);
    }
  };

  const openAddGearModal = (category: ComponentCategory) => {
    setAddModalCategory(category);
  };

  const handleModalAddSubmit = async (newGear: AddInventoryParams) => {
    if (!addModalCategory) {
      throw new Error('No component category selected');
    }

    const componentCategory = COMPONENT_CATEGORIES.find((c) => c.value === addModalCategory);
    if (!componentCategory) {
      throw new Error('Invalid component category');
    }

    const expectedCategory = componentCategory.equipmentCategory;
    const incomingCategory = newGear.category;

    if (incomingCategory !== expectedCategory) {
      throw new Error(
        `Selected item category (${incomingCategory}) does not match ${componentCategory.label}. Please choose matching gear.`,
      );
    }

    await handleAutoAddGear(addModalCategory, {
      ...newGear,
      category: expectedCategory,
    });
  };

  // Handle component removal
  const handleRemoveComponent = async (category: ComponentCategory) => {
    if (!confirm('Remove this component from the aircraft?')) return;
    setIsSubmitting(true);
    try {
      await onSetComponent({ category, inventoryItemId: '' });
      onRefresh();
    } catch (err) {
      console.error('Failed to remove component:', err);
    } finally {
      setIsSubmitting(false);
    }
  };

  // Handle receiver settings save
  const handleSaveReceiver = async () => {
    setIsSavingReceiver(true);
    setReceiverSaved(false);
    try {
      await onSetReceiverSettings(receiverSettings);
      await onRefresh();
      setReceiverSaved(true);
      // Reset saved state after 2 seconds
      setTimeout(() => setReceiverSaved(false), 2000);
    } catch (err) {
      console.error('Failed to save receiver settings:', err);
    } finally {
      setIsSavingReceiver(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm z-50 flex items-center justify-center p-4">
      <div className="bg-slate-800 rounded-xl w-full max-w-4xl max-h-[90vh] overflow-hidden flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-slate-700">
          <div className="flex items-center gap-4">
            {/* Aircraft image/icon */}
            <div className="w-16 h-16 rounded-lg overflow-hidden bg-slate-700 flex-shrink-0">
              {aircraft.hasImage ? (
                <img
                  src={getAircraftImageUrl(aircraft.id)}
                  alt={aircraft.name}
                  className="w-full h-full object-cover"
                />
              ) : (
                <div className="w-full h-full flex items-center justify-center text-3xl">
                  {aircraftType?.icon || '🚁'}
                </div>
              )}
            </div>
            <div>
              <h2 className="text-lg font-semibold text-white">{aircraft.name}</h2>
              {aircraft.nickname && (
                <p className="text-primary-400 text-sm">"{aircraft.nickname}"</p>
              )}
              <span className="inline-block mt-1 px-2 py-0.5 bg-slate-700 text-slate-300 rounded text-xs">
                {aircraftType?.label || aircraft.type}
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
          </button>
          <button
            onClick={() => setViewMode('receiver')}
            className={`flex-1 px-4 py-3 text-sm font-medium transition-colors ${
              viewMode === 'receiver'
                ? 'text-primary-400 border-b-2 border-primary-400'
                : 'text-slate-400 hover:text-white'
            }`}
          >
            Receiver Settings
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4">
          {viewMode === 'components' && (
            <div className="space-y-3">
              {COMPONENT_CATEGORIES.map((cat) => {
                const component = getComponentByCategory(cat.value);
                // Check both server-provided inventoryItem and local lookup
                const inventoryItem = component?.inventoryItem || (component ? getInventoryItemById(component.inventoryItemId) : null);
                const hasAssignedItem = !!inventoryItem;
                const availableItems = getAvailableItems(cat.value);
                const isSelecting = selectedCategory === cat.value;

                return (
                  <div
                    key={cat.value}
                    className="bg-slate-700/50 border border-slate-700 rounded-lg p-3"
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <div className="w-10 h-10 bg-slate-600 rounded-lg flex items-center justify-center text-xl">
                          {cat.value === 'fc' && '🧠'}
                          {cat.value === 'esc' && '⚡'}
                          {cat.value === 'aio' && '🔌'}
                          {cat.value === 'stack' && '🧩'}
                          {cat.value === 'receiver' && '📡'}
                          {cat.value === 'vtx' && '📺'}
                          {cat.value === 'motors' && '🔄'}
                          {cat.value === 'camera' && '📷'}
                          {cat.value === 'frame' && '🏗️'}
                          {cat.value === 'props' && '🍃'}
                          {cat.value === 'antenna' && '📶'}
                        </div>
                        <div>
                          <h4 className="text-white font-medium">{cat.label}</h4>
                          {inventoryItem ? (
                            <p className="text-slate-400 text-sm">{inventoryItem.name}</p>
                          ) : (
                            <p className="text-slate-500 text-sm italic">Not assigned</p>
                          )}
                        </div>
                      </div>

                      {/* Actions */}
                      <div className="flex items-center gap-2">
                        {hasAssignedItem && (
                          <button
                            onClick={() => handleRemoveComponent(cat.value)}
                            disabled={isSubmitting}
                            className="px-3 py-1.5 text-sm text-slate-400 hover:text-red-400 transition-colors"
                          >
                            Remove
                          </button>
                        )}
                        <button
                          onClick={() => setSelectedCategory(isSelecting ? null : cat.value)}
                          disabled={isSubmitting}
                          className="px-3 py-1.5 text-sm bg-primary-600 hover:bg-primary-700 text-white rounded-lg transition-colors"
                        >
                          {hasAssignedItem ? 'Change' : 'Assign'}
                        </button>
                      </div>
                    </div>

                    {/* Assignment dropdown */}
                    {isSelecting && (
                      <div className="mt-3 pt-3 border-t border-slate-600">
                        {isLoadingInventory ? (
                          <div className="text-slate-400 text-sm">Loading inventory...</div>
                        ) : availableItems.length === 0 ? (
                          <div className="space-y-2">
                            <p className="text-slate-400 text-sm">
                              No matching items in your inventory.
                            </p>
                            <button
                              type="button"
                              onClick={() => openAddGearModal(cat.value)}
                              disabled={isSubmitting}
                              className="inline-flex items-center gap-2 px-3 py-1.5 bg-primary-600 hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed text-white text-sm rounded-lg transition-colors"
                            >
                              + Add {cat.label} to Inventory
                            </button>
                          </div>
                        ) : (
                          <div className="space-y-2">
                            <select
                              className="w-full h-11 px-3 bg-slate-600 border border-slate-500 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500"
                              onChange={(e) => {
                                if (e.target.value) {
                                  handleAssignComponent(cat.value, e.target.value);
                                }
                              }}
                              defaultValue=""
                            >
                              <option value="">Select from inventory...</option>
                              {availableItems.map((item) => (
                                <option key={item.id} value={item.id}>
                                  {item.name} {item.manufacturer ? `(${item.manufacturer})` : ''}
                                </option>
                              ))}
                            </select>
                            <div className="text-xs text-slate-500">
                              Need another option?
                            </div>
                            <button
                              type="button"
                              onClick={() => openAddGearModal(cat.value)}
                              disabled={isSubmitting}
                              className="inline-flex items-center gap-2 px-3 py-1.5 bg-primary-600 hover:bg-primary-700 disabled:opacity-50 disabled:cursor-not-allowed text-white text-sm rounded-lg transition-colors"
                            >
                              + Add {cat.label} to Inventory
                            </button>
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}

          {viewMode === 'receiver' && (
            <div className="space-y-4">
              <div className="bg-slate-700/50 border border-slate-700 rounded-lg p-4">
                <h4 className="text-white font-medium mb-4">Receiver Configuration</h4>
                
                <div className="space-y-4">
                  {/* Binding Phrase */}
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-1">
                      Binding Phrase
                    </label>
                    <input
                      type="text"
                      value={receiverSettings.bindingPhrase || ''}
                      onChange={(e) => setReceiverSettings({ ...receiverSettings, bindingPhrase: e.target.value })}
                      placeholder="Your binding phrase"
                      className="w-full px-3 py-2 bg-slate-600 border border-slate-500 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500"
                    />
                  </div>

                  {/* Model Match Number */}
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-1">
                      Model Match Number
                    </label>
                    <input
                      type="number"
                      min="0"
                      max="63"
                      value={receiverSettings.modelMatch ?? ''}
                      onChange={(e) => setReceiverSettings({ ...receiverSettings, modelMatch: e.target.value ? parseInt(e.target.value) : undefined })}
                      placeholder="0-63 (optional)"
                      className="w-full px-3 py-2 bg-slate-600 border border-slate-500 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500"
                    />
                    <p className="mt-1 text-xs text-slate-500">Set to match your transmitter model ID for model matching</p>
                  </div>

                  {/* Rate */}
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-1">
                      Packet Rate (Hz)
                    </label>
                    <select
                      value={receiverSettings.rate || ''}
                      onChange={(e) => setReceiverSettings({ ...receiverSettings, rate: parseInt(e.target.value) || undefined })}
                      className="w-full h-11 px-3 bg-slate-600 border border-slate-500 rounded-lg text-white focus:outline-none focus:border-primary-500"
                    >
                      <option value="">Select rate...</option>
                      <option value="50">50 Hz</option>
                      <option value="100">100 Hz</option>
                      <option value="150">150 Hz</option>
                      <option value="200">200 Hz</option>
                      <option value="250">250 Hz</option>
                      <option value="333">333 Hz</option>
                      <option value="500">500 Hz</option>
                      <option value="1000">1000 Hz</option>
                    </select>
                  </div>

                  {/* Telemetry Ratio */}
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-1">
                      Telemetry Ratio
                    </label>
                    <select
                      value={receiverSettings.tlm || ''}
                      onChange={(e) => setReceiverSettings({ ...receiverSettings, tlm: parseInt(e.target.value) || undefined })}
                      className="w-full h-11 px-3 bg-slate-600 border border-slate-500 rounded-lg text-white focus:outline-none focus:border-primary-500"
                    >
                      <option value="">Select ratio...</option>
                      <option value="0">Off</option>
                      <option value="2">1:2</option>
                      <option value="4">1:4</option>
                      <option value="8">1:8</option>
                      <option value="16">1:16</option>
                      <option value="32">1:32</option>
                      <option value="64">1:64</option>
                      <option value="128">1:128</option>
                    </select>
                  </div>

                  {/* Power */}
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-1">
                      TX Power (mW)
                    </label>
                    <select
                      value={receiverSettings.power || ''}
                      onChange={(e) => setReceiverSettings({ ...receiverSettings, power: parseInt(e.target.value) || undefined })}
                      className="w-full h-11 px-3 bg-slate-600 border border-slate-500 rounded-lg text-white focus:outline-none focus:border-primary-500"
                    >
                      <option value="">Select power...</option>
                      <option value="10">10 mW</option>
                      <option value="25">25 mW</option>
                      <option value="50">50 mW</option>
                      <option value="100">100 mW</option>
                      <option value="250">250 mW</option>
                      <option value="500">500 mW</option>
                      <option value="1000">1000 mW</option>
                    </select>
                  </div>

                  {/* Device Name */}
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-1">
                      Device Name
                    </label>
                    <input
                      type="text"
                      value={receiverSettings.deviceName || ''}
                      onChange={(e) => setReceiverSettings({ ...receiverSettings, deviceName: e.target.value })}
                      placeholder="e.g., MyQuad RX"
                      className="w-full px-3 py-2 bg-slate-600 border border-slate-500 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500"
                    />
                  </div>
                </div>

                <div className="mt-6 flex justify-end">
                  <button
                    onClick={handleSaveReceiver}
                    disabled={isSavingReceiver}
                    className={`px-4 py-2 ${receiverSaved ? 'bg-green-600' : 'bg-primary-600 hover:bg-primary-700'} disabled:bg-primary-600/50 text-white font-medium rounded-lg transition-colors flex items-center gap-2`}
                  >
                    {isSavingReceiver ? (
                      <>
                        <div className="animate-spin rounded-full h-4 w-4 border-2 border-white/30 border-t-white" />
                        Saving...
                      </>
                    ) : receiverSaved ? (
                      <>
                        <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                        </svg>
                        Saved!
                      </>
                    ) : (
                      'Save Receiver Settings'
                    )}
                  </button>
                </div>
              </div>
            </div>
          )}

          {viewMode === 'tuning' && (
            <TuningTabContent
              aircraftName={aircraft.name}
              tuningData={tuningData}
              isLoading={isLoadingTuning}
              showCliUpload={showCliUpload}
              setShowCliUpload={setShowCliUpload}
              cliDump={cliDump}
              setCliDump={setCliDump}
              diffBackup={diffBackup}
              setDiffBackup={setDiffBackup}
              isUploading={isUploadingTuning}
              onUpload={handleUploadTuning}
              uploadMode={uploadMode}
              setUploadMode={setUploadMode}
            />
          )}
        </div>
      </div>

      <AddGearModal
        isOpen={addModalCategory !== null}
        onClose={() => setAddModalCategory(null)}
        onSubmit={handleModalAddSubmit}
        initialCategory={addModalCategory ? COMPONENT_CATEGORIES.find((cat) => cat.value === addModalCategory)?.equipmentCategory : undefined}
      />
    </div>
  );
}

// Tuning Tab Content
interface TuningTabContentProps {
  aircraftName: string;
  tuningData: AircraftTuningResponse | null;
  isLoading: boolean;
  showCliUpload: boolean;
  setShowCliUpload: (show: boolean) => void;
  cliDump: string;
  setCliDump: (dump: string) => void;
  diffBackup: string;
  setDiffBackup: (backup: string) => void;
  isUploading: boolean;
  onUpload: () => void;
  uploadMode: 'dump' | 'backup';
  setUploadMode: (mode: 'dump' | 'backup') => void;
}

function TuningTabContent({
  aircraftName,
  tuningData,
  isLoading,
  showCliUpload,
  setShowCliUpload,
  cliDump,
  setCliDump,
  diffBackup,
  setDiffBackup,
  isUploading,
  onUpload,
  uploadMode,
  setUploadMode,
}: TuningTabContentProps) {
  const handleDownloadBackup = () => {
    if (!tuningData?.diffBackup) return;
    const blob = new Blob([tuningData.diffBackup], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    // Build filename: aircraft-name_board-name_YYYY-MM-DD.txt
    const safeName = aircraftName.replace(/[^a-zA-Z0-9-_]/g, '-').toLowerCase();
    const board = tuningData.boardName?.replace(/[^a-zA-Z0-9-_]/g, '-').toLowerCase() || 'unknown';
    const date = new Date().toISOString().split('T')[0];
    a.download = `${safeName}_${board}_${date}.txt`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-2 border-primary-400 border-t-transparent"></div>
      </div>
    );
  }

  if (showCliUpload) {
    const isInitialUpload = !tuningData?.hasTuning;
    const title = isInitialUpload 
      ? 'Upload Tuning Data' 
      : (uploadMode === 'dump' ? 'Update Tuning' : 'Update Backup');

    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h4 className="text-white font-medium">{title}</h4>
          <button
            onClick={() => setShowCliUpload(false)}
            className="text-slate-400 hover:text-white transition-colors"
          >
            Cancel
          </button>
        </div>
        <div className="bg-slate-700/50 border border-slate-700 rounded-lg p-4 space-y-4">
          {/* Mode selector - show on initial upload OR when updating */}
          {isInitialUpload && (
            <div className="flex gap-2 p-1 bg-slate-600/50 rounded-lg w-fit">
              <button
                onClick={() => setUploadMode('dump')}
                className={`px-3 py-1.5 text-sm rounded-md transition-colors ${
                  uploadMode === 'dump' 
                    ? 'bg-primary-600 text-white' 
                    : 'text-slate-400 hover:text-white'
                }`}
              >
                CLI Dump
              </button>
              <button
                onClick={() => setUploadMode('backup')}
                className={`px-3 py-1.5 text-sm rounded-md transition-colors ${
                  uploadMode === 'backup' 
                    ? 'bg-primary-600 text-white' 
                    : 'text-slate-400 hover:text-white'
                }`}
              >
                Diff Backup
              </button>
            </div>
          )}

          {uploadMode === 'dump' ? (
            <div>
              <p className="text-sm text-slate-300 mb-2">
                Paste your full CLI dump below. Run <code className="bg-slate-600 px-1 rounded">dump</code> in the Betaflight CLI tab to get this.
              </p>
              <textarea
                value={cliDump}
                onChange={(e) => setCliDump(e.target.value)}
                placeholder="Paste your CLI dump here..."
                rows={10}
                className="w-full px-3 py-2 bg-slate-600 border border-slate-500 rounded-lg text-white text-sm font-mono placeholder-slate-400 focus:outline-none focus:border-primary-500"
              />
            </div>
          ) : (
            <div>
              <p className="text-sm text-slate-300 mb-2">
                Paste your diff backup below. Run <code className="bg-slate-600 px-1 rounded">diff all</code> in the Betaflight CLI tab.
                {!isInitialUpload && ' This will update only the backup without changing your tuning data.'}
              </p>
              <textarea
                value={diffBackup}
                onChange={(e) => setDiffBackup(e.target.value)}
                placeholder="Paste your diff all output here..."
                rows={10}
                className="w-full px-3 py-2 bg-slate-600 border border-slate-500 rounded-lg text-white text-sm font-mono placeholder-slate-400 focus:outline-none focus:border-primary-500"
              />
            </div>
          )}

          <div className="flex justify-end gap-3">
            <button
              onClick={() => setShowCliUpload(false)}
              className="px-4 py-2 text-slate-400 hover:text-white transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={onUpload}
              disabled={isUploading || (uploadMode === 'dump' ? !cliDump.trim() : !diffBackup.trim())}
              className="px-4 py-2 bg-primary-600 hover:bg-primary-700 disabled:bg-primary-600/50 text-white font-medium rounded-lg transition-colors flex items-center gap-2"
            >
              {isUploading ? (
                <>
                  <div className="animate-spin rounded-full h-4 w-4 border-2 border-white/30 border-t-white" />
                  {uploadMode === 'dump' ? 'Parsing...' : 'Saving...'}
                </>
              ) : (
                uploadMode === 'dump' ? 'Upload & Parse' : 'Save Backup'
              )}
            </button>
          </div>
        </div>
      </div>
    );
  }

  if (!tuningData?.hasTuning) {
    return (
      <div className="text-center py-12">
        <div className="text-4xl mb-4">🎛️</div>
        <h4 className="text-white font-medium mb-2">No Tuning Data</h4>
        <p className="text-slate-400 text-sm mb-6">
          Upload a CLI dump to view tuning settings, or a diff backup for easy restore.
        </p>
        <button
          onClick={() => { setUploadMode('dump'); setShowCliUpload(true); }}
          className="px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white font-medium rounded-lg transition-colors"
        >
          Upload Tuning Data
        </button>
      </div>
    );
  }

  const { tuning, firmwareName, firmwareVersion, boardName, parseStatus, snapshotDate } = tuningData;

  return (
    <div className="space-y-4">
      {/* Header info */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2 text-sm">
            <span className="text-slate-400">Firmware:</span>
            <span className="text-white font-medium">
              {firmwareName || 'Unknown'} {firmwareVersion && `v${firmwareVersion}`}
            </span>
          </div>
          {boardName && (
            <div className="flex items-center gap-2 text-sm">
              <span className="text-slate-400">Board:</span>
              <span className="text-white">{boardName}</span>
            </div>
          )}
        </div>
        <div className="flex items-center gap-3">
          {tuningData.hasDiffBackup && (
            <button
              onClick={handleDownloadBackup}
              className="text-sm text-green-400 hover:text-green-300 transition-colors flex items-center gap-1"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
              </svg>
              Download Backup
            </button>
          )}
          <button
            onClick={() => { setUploadMode('backup'); setShowCliUpload(true); }}
            className="text-sm text-slate-400 hover:text-slate-300 transition-colors"
          >
            Update Backup
          </button>
          <button
            onClick={() => { setUploadMode('dump'); setShowCliUpload(true); }}
            className="text-sm text-primary-400 hover:text-primary-300 transition-colors"
          >
            Update Tuning
          </button>
        </div>
      </div>

      {parseStatus === 'partial' && (
        <div className="bg-yellow-900/30 border border-yellow-700/50 rounded-lg px-3 py-2 text-sm text-yellow-300">
          ⚠️ Some settings could not be parsed from the CLI dump.
        </div>
      )}

      {snapshotDate && (
        <div className="text-xs text-slate-500">
          Last updated: {new Date(snapshotDate).toLocaleDateString()}
        </div>
      )}

      {/* PIDs */}
      {tuning?.pids && <PIDDisplay pids={tuning.pids} />}

      {/* Rates */}
      {tuning?.rates && <RatesDisplay rates={tuning.rates} />}

      {/* Filters */}
      {tuning?.filters && <FiltersDisplay filters={tuning.filters} />}

      {/* Motor/Loop Settings */}
      {tuning?.motorMixer && (
        <div className="bg-slate-700/50 border border-slate-700 rounded-lg p-4">
          <h4 className="text-white font-medium mb-3">Motor & Loop</h4>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {tuning.motorMixer.motorProtocol && (
              <ValueCard label="Protocol" value={tuning.motorMixer.motorProtocol} />
            )}
            {tuning.motorMixer.gyroHz && (
              <ValueCard label="Gyro" value={`${tuning.motorMixer.gyroHz} Hz`} />
            )}
            {tuning.motorMixer.pidHz && (
              <ValueCard label="PID Loop" value={`${tuning.motorMixer.pidHz} Hz`} />
            )}
            {tuning.motorMixer.digitalIdlePercent !== undefined && (
              <ValueCard label="Motor Idle" value={`${(tuning.motorMixer.digitalIdlePercent / 10).toFixed(1)}%`} />
            )}
            {tuning.motorMixer.dshotBidir !== undefined && (
              <ValueCard label="Bidirectional" value={tuning.motorMixer.dshotBidir ? 'Yes' : 'No'} />
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// PID Display Component
function PIDDisplay({ pids }: { pids: PIDProfile }) {
  return (
    <div className="bg-slate-700/50 border border-slate-700 rounded-lg p-4">
      <h4 className="text-white font-medium mb-3">PID Values</h4>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="text-slate-400 text-left">
              <th className="pb-2"></th>
              <th className="pb-2 text-center">P</th>
              <th className="pb-2 text-center">I</th>
              <th className="pb-2 text-center">D</th>
              <th className="pb-2 text-center">FF</th>
            </tr>
          </thead>
          <tbody className="text-white">
            <tr>
              <td className="py-1 text-slate-300">Roll</td>
              <td className="py-1 text-center font-mono">{pids.roll.p}</td>
              <td className="py-1 text-center font-mono">{pids.roll.i}</td>
              <td className="py-1 text-center font-mono">{pids.roll.d}</td>
              <td className="py-1 text-center font-mono">{pids.roll.ff ?? '-'}</td>
            </tr>
            <tr>
              <td className="py-1 text-slate-300">Pitch</td>
              <td className="py-1 text-center font-mono">{pids.pitch.p}</td>
              <td className="py-1 text-center font-mono">{pids.pitch.i}</td>
              <td className="py-1 text-center font-mono">{pids.pitch.d}</td>
              <td className="py-1 text-center font-mono">{pids.pitch.ff ?? '-'}</td>
            </tr>
            <tr>
              <td className="py-1 text-slate-300">Yaw</td>
              <td className="py-1 text-center font-mono">{pids.yaw.p}</td>
              <td className="py-1 text-center font-mono">{pids.yaw.i}</td>
              <td className="py-1 text-center font-mono">{pids.yaw.d}</td>
              <td className="py-1 text-center font-mono">{pids.yaw.ff ?? '-'}</td>
            </tr>
          </tbody>
        </table>
      </div>

      {/* Additional PID settings */}
      <div className="mt-4 grid grid-cols-2 md:grid-cols-4 gap-3">
        {pids.iTermRelax && (
          <ValueCard label="I-Term Relax" value={pids.iTermRelax} small />
        )}
        {pids.antiGravityGain && (
          <ValueCard label="Anti-Gravity" value={pids.antiGravityGain.toString()} small />
        )}
        {pids.tpaRate !== undefined && pids.tpaRate > 0 && (
          <ValueCard label="TPA Rate" value={`${pids.tpaRate}%`} small />
        )}
        {pids.tpaBreakpoint !== undefined && pids.tpaBreakpoint > 0 && (
          <ValueCard label="TPA Breakpoint" value={pids.tpaBreakpoint.toString()} small />
        )}
      </div>
    </div>
  );
}

// Rates Display Component
function RatesDisplay({ rates }: { rates: RateProfile }) {
  return (
    <div className="bg-slate-700/50 border border-slate-700 rounded-lg p-4">
      <div className="flex items-center justify-between mb-3">
        <h4 className="text-white font-medium">Rates</h4>
        {rates.rateType && (
          <span className="text-xs bg-slate-600 text-slate-300 px-2 py-0.5 rounded">
            {rates.rateType}
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
              <td className="py-1 text-center font-mono">{rates.rcRates?.roll ?? '-'}</td>
              <td className="py-1 text-center font-mono">{rates.rcRates?.pitch ?? '-'}</td>
              <td className="py-1 text-center font-mono">{rates.rcRates?.yaw ?? '-'}</td>
            </tr>
            <tr>
              <td className="py-1 text-slate-300">Super Rate</td>
              <td className="py-1 text-center font-mono">{rates.superRates?.roll ?? '-'}</td>
              <td className="py-1 text-center font-mono">{rates.superRates?.pitch ?? '-'}</td>
              <td className="py-1 text-center font-mono">{rates.superRates?.yaw ?? '-'}</td>
            </tr>
            <tr>
              <td className="py-1 text-slate-300">RC Expo</td>
              <td className="py-1 text-center font-mono">{rates.rcExpo?.roll ?? '-'}</td>
              <td className="py-1 text-center font-mono">{rates.rcExpo?.pitch ?? '-'}</td>
              <td className="py-1 text-center font-mono">{rates.rcExpo?.yaw ?? '-'}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  );
}

// Filters Display Component
function FiltersDisplay({ filters }: { filters: FilterSettings }) {
  return (
    <div className="bg-slate-700/50 border border-slate-700 rounded-lg p-4">
      <h4 className="text-white font-medium mb-3">Filters</h4>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Gyro Filters */}
        <div>
          <h5 className="text-sm text-slate-400 mb-2">Gyro Lowpass</h5>
          <div className="space-y-1 text-sm">
            {filters.gyroLowpassEnabled && filters.gyroLowpassHz && (
              <div className="flex justify-between">
                <span className="text-slate-300">LPF 1</span>
                <span className="text-white font-mono">{filters.gyroLowpassHz} Hz ({filters.gyroLowpassType || 'PT1'})</span>
              </div>
            )}
            {filters.gyroLowpass2Enabled && filters.gyroLowpass2Hz && (
              <div className="flex justify-between">
                <span className="text-slate-300">LPF 2</span>
                <span className="text-white font-mono">{filters.gyroLowpass2Hz} Hz ({filters.gyroLowpass2Type || 'PT1'})</span>
              </div>
            )}
            {filters.gyroDynLowpassEnabled && (
              <div className="flex justify-between">
                <span className="text-slate-300">Dynamic</span>
                <span className="text-white font-mono">{filters.gyroDynLowpassMinHz}-{filters.gyroDynLowpassMaxHz} Hz</span>
              </div>
            )}
          </div>
        </div>

        {/* D-term Filters */}
        <div>
          <h5 className="text-sm text-slate-400 mb-2">D-Term Lowpass</h5>
          <div className="space-y-1 text-sm">
            {filters.dtermLowpassEnabled && filters.dtermLowpassHz && (
              <div className="flex justify-between">
                <span className="text-slate-300">LPF 1</span>
                <span className="text-white font-mono">{filters.dtermLowpassHz} Hz ({filters.dtermLowpassType || 'PT1'})</span>
              </div>
            )}
            {filters.dtermLowpass2Enabled && filters.dtermLowpass2Hz && (
              <div className="flex justify-between">
                <span className="text-slate-300">LPF 2</span>
                <span className="text-white font-mono">{filters.dtermLowpass2Hz} Hz ({filters.dtermLowpass2Type || 'PT1'})</span>
              </div>
            )}
            {filters.dtermDynLowpassEnabled && (
              <div className="flex justify-between">
                <span className="text-slate-300">Dynamic</span>
                <span className="text-white font-mono">{filters.dtermDynLowpassMinHz}-{filters.dtermDynLowpassMaxHz} Hz</span>
              </div>
            )}
          </div>
        </div>

        {/* Notch Filters */}
        <div>
          <h5 className="text-sm text-slate-400 mb-2">Notch Filters</h5>
          <div className="space-y-1 text-sm">
            {filters.dynNotchEnabled && (
              <div className="flex justify-between">
                <span className="text-slate-300">Dynamic Notch</span>
                <span className="text-white font-mono">{filters.dynNotchCount}x @ {filters.dynNotchMinHz}-{filters.dynNotchMaxHz} Hz</span>
              </div>
            )}
            {filters.gyroNotch1Enabled && filters.gyroNotch1Hz && (
              <div className="flex justify-between">
                <span className="text-slate-300">Gyro Notch 1</span>
                <span className="text-white font-mono">{filters.gyroNotch1Hz} Hz</span>
              </div>
            )}
            {filters.gyroNotch2Enabled && filters.gyroNotch2Hz && (
              <div className="flex justify-between">
                <span className="text-slate-300">Gyro Notch 2</span>
                <span className="text-white font-mono">{filters.gyroNotch2Hz} Hz</span>
              </div>
            )}
          </div>
        </div>

        {/* RPM Filter */}
        <div>
          <h5 className="text-sm text-slate-400 mb-2">RPM Filter</h5>
          <div className="space-y-1 text-sm">
            {filters.rpmFilterEnabled ? (
              <>
                <div className="flex justify-between">
                  <span className="text-slate-300">Harmonics</span>
                  <span className="text-white font-mono">{filters.rpmFilterHarmonics}</span>
                </div>
                {filters.rpmFilterMinHz && (
                  <div className="flex justify-between">
                    <span className="text-slate-300">Min Hz</span>
                    <span className="text-white font-mono">{filters.rpmFilterMinHz}</span>
                  </div>
                )}
              </>
            ) : (
              <span className="text-slate-500">Disabled</span>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

// Small value card component
function ValueCard({ label, value, small }: { label: string; value: string; small?: boolean }) {
  return (
    <div className={small ? '' : 'bg-slate-600/50 rounded-lg p-2'}>
      <div className="text-xs text-slate-400">{label}</div>
      <div className={`text-white ${small ? 'text-sm' : 'font-medium'}`}>{value}</div>
    </div>
  );
}

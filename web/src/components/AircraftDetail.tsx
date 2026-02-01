import { useState, useEffect } from 'react';
import type { 
  AircraftDetailsResponse, 
  AircraftComponent, 
  ComponentCategory,
  ELRSConfig,
  SetComponentParams 
} from '../aircraftTypes';
import { AIRCRAFT_TYPES, COMPONENT_CATEGORIES } from '../aircraftTypes';
import type { InventoryItem, AddInventoryParams } from '../equipmentTypes';
import { getInventory } from '../equipmentApi';
import { getAircraftImageUrl } from '../aircraftApi';

interface AircraftDetailProps {
  details: AircraftDetailsResponse;
  onClose: () => void;
  onSetComponent: (params: SetComponentParams) => Promise<void>;
  onSetELRSSettings: (settings: ELRSConfig) => Promise<void>;
  onRefresh: () => void;
}

type ViewMode = 'components' | 'elrs';

export function AircraftDetail({
  details,
  onClose,
  onSetComponent,
  onSetELRSSettings,
  onRefresh,
}: AircraftDetailProps) {
  const [viewMode, setViewMode] = useState<ViewMode>('components');
  const [inventoryItems, setInventoryItems] = useState<InventoryItem[]>([]);
  const [isLoadingInventory, setIsLoadingInventory] = useState(true);
  const [selectedCategory, setSelectedCategory] = useState<ComponentCategory | null>(null);
  const [isSubmitting, setIsSubmitting] = useState(false);
  
  // ELRS settings state
  const [elrsSettings, setElrsSettings] = useState<ELRSConfig>(
    details.elrsSettings?.settings || {}
  );
  const [isSavingElrs, setIsSavingElrs] = useState(false);

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
    } finally {
      setIsSubmitting(false);
      setSelectedCategory(null);
    }
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

  // Handle ELRS settings save
  const handleSaveElrs = async () => {
    setIsSavingElrs(true);
    try {
      await onSetELRSSettings(elrsSettings);
      onRefresh();
    } catch (err) {
      console.error('Failed to save ELRS settings:', err);
    } finally {
      setIsSavingElrs(false);
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
            onClick={() => setViewMode('elrs')}
            className={`flex-1 px-4 py-3 text-sm font-medium transition-colors ${
              viewMode === 'elrs'
                ? 'text-primary-400 border-b-2 border-primary-400'
                : 'text-slate-400 hover:text-white'
            }`}
          >
            ELRS Settings
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4">
          {viewMode === 'components' && (
            <div className="space-y-3">
              {COMPONENT_CATEGORIES.map((cat) => {
                const component = getComponentByCategory(cat.value);
                const inventoryItem = component ? getInventoryItemById(component.inventoryItemId) : null;
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
                          {cat.value === 'elrs_module' && '📡'}
                          {cat.value === 'vtx' && '📺'}
                          {cat.value === 'motors' && '🔄'}
                          {cat.value === 'camera' && '📷'}
                          {cat.value === 'frame' && '🏗️'}
                          {cat.value === 'props' && '🍃'}
                          {cat.value === 'battery' && '🔋'}
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
                        {component && (
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
                          {component ? 'Change' : 'Assign'}
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
                            <QuickAddForm
                              category={cat}
                              onSubmit={(newGear) => handleAutoAddGear(cat.value, newGear)}
                              isSubmitting={isSubmitting}
                            />
                          </div>
                        ) : (
                          <div className="space-y-2">
                            <select
                              className="w-full px-3 py-2 bg-slate-600 border border-slate-500 rounded-lg text-white text-sm focus:outline-none focus:border-primary-500"
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
                              Or add a new item:
                            </div>
                            <QuickAddForm
                              category={cat}
                              onSubmit={(newGear) => handleAutoAddGear(cat.value, newGear)}
                              isSubmitting={isSubmitting}
                            />
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}

          {viewMode === 'elrs' && (
            <div className="space-y-4">
              <div className="bg-slate-700/50 border border-slate-700 rounded-lg p-4">
                <h4 className="text-white font-medium mb-4">ELRS Configuration</h4>
                
                <div className="space-y-4">
                  {/* Binding Phrase */}
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-1">
                      Binding Phrase
                    </label>
                    <input
                      type="text"
                      value={elrsSettings.bindingPhrase || ''}
                      onChange={(e) => setElrsSettings({ ...elrsSettings, bindingPhrase: e.target.value })}
                      placeholder="Your binding phrase"
                      className="w-full px-3 py-2 bg-slate-600 border border-slate-500 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500"
                    />
                  </div>

                  {/* Rate */}
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-1">
                      Packet Rate (Hz)
                    </label>
                    <select
                      value={elrsSettings.rate || ''}
                      onChange={(e) => setElrsSettings({ ...elrsSettings, rate: parseInt(e.target.value) || undefined })}
                      className="w-full px-3 py-2 bg-slate-600 border border-slate-500 rounded-lg text-white focus:outline-none focus:border-primary-500"
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
                      value={elrsSettings.tlm || ''}
                      onChange={(e) => setElrsSettings({ ...elrsSettings, tlm: parseInt(e.target.value) || undefined })}
                      className="w-full px-3 py-2 bg-slate-600 border border-slate-500 rounded-lg text-white focus:outline-none focus:border-primary-500"
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
                      value={elrsSettings.power || ''}
                      onChange={(e) => setElrsSettings({ ...elrsSettings, power: parseInt(e.target.value) || undefined })}
                      className="w-full px-3 py-2 bg-slate-600 border border-slate-500 rounded-lg text-white focus:outline-none focus:border-primary-500"
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
                      value={elrsSettings.deviceName || ''}
                      onChange={(e) => setElrsSettings({ ...elrsSettings, deviceName: e.target.value })}
                      placeholder="e.g., MyQuad RX"
                      className="w-full px-3 py-2 bg-slate-600 border border-slate-500 rounded-lg text-white placeholder-slate-400 focus:outline-none focus:border-primary-500"
                    />
                  </div>
                </div>

                <div className="mt-6 flex justify-end">
                  <button
                    onClick={handleSaveElrs}
                    disabled={isSavingElrs}
                    className="px-4 py-2 bg-primary-600 hover:bg-primary-700 disabled:bg-primary-600/50 text-white font-medium rounded-lg transition-colors flex items-center gap-2"
                  >
                    {isSavingElrs ? (
                      <>
                        <div className="animate-spin rounded-full h-4 w-4 border-2 border-white/30 border-t-white" />
                        Saving...
                      </>
                    ) : (
                      'Save ELRS Settings'
                    )}
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// Quick add form for auto-add gear
interface QuickAddFormProps {
  category: { value: ComponentCategory; label: string; equipmentCategory: string };
  onSubmit: (params: AddInventoryParams) => void;
  isSubmitting: boolean;
}

function QuickAddForm({ category, onSubmit, isSubmitting }: QuickAddFormProps) {
  const [name, setName] = useState('');
  const [manufacturer, setManufacturer] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;

    onSubmit({
      name: name.trim(),
      category: category.equipmentCategory as any,
      manufacturer: manufacturer.trim() || undefined,
      quantity: 1,
      condition: 'new',
    });
  };

  return (
    <form onSubmit={handleSubmit} className="flex gap-2">
      <input
        type="text"
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder={`New ${category.label} name...`}
        className="flex-1 px-3 py-1.5 bg-slate-600 border border-slate-500 rounded-lg text-white text-sm placeholder-slate-400 focus:outline-none focus:border-primary-500"
      />
      <input
        type="text"
        value={manufacturer}
        onChange={(e) => setManufacturer(e.target.value)}
        placeholder="Brand"
        className="w-24 px-3 py-1.5 bg-slate-600 border border-slate-500 rounded-lg text-white text-sm placeholder-slate-400 focus:outline-none focus:border-primary-500"
      />
      <button
        type="submit"
        disabled={isSubmitting || !name.trim()}
        className="px-3 py-1.5 bg-green-600 hover:bg-green-700 disabled:bg-green-600/50 text-white text-sm rounded-lg transition-colors"
      >
        Add
      </button>
    </form>
  );
}

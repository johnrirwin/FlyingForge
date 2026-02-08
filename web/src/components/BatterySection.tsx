import { useState, useEffect, useCallback, useRef } from 'react';
import { createPortal } from 'react-dom';
import type {
  Battery,
  BatteryLog,
  BatteryChemistry,
  BatteryFormState,
  BatteryLogFormState,
  LabelSize,
} from '../batteryTypes';
import {
  BATTERY_CHEMISTRY_OPTIONS,
  CELL_COUNT_OPTIONS,
  INITIAL_BATTERY_FORM_STATE,
  createInitialLogFormState,
  formatChemistry,
  formatCellCount,
  formatCapacity,
} from '../batteryTypes';
import {
  getBatteries,
  getBattery,
  createBattery,
  updateBattery,
  deleteBattery,
  getBatteryLogs,
  createBatteryLog,
  deleteBatteryLog,
  printBatteryLabel,
} from '../batteryApi';

interface BatterySectionProps {
  onError?: (message: string) => void;
}

type ViewMode = 'list' | 'create' | 'detail' | 'edit';

const FOCUSABLE_SELECTOR = 'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])';

export function BatterySection({ onError }: BatterySectionProps) {
  // State
  const [batteries, setBatteries] = useState<Battery[]>([]);
  const [selectedBattery, setSelectedBattery] = useState<Battery | null>(null);
  const [logs, setLogs] = useState<BatteryLog[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [isLogsLoading, setIsLogsLoading] = useState(false);
  const [viewMode, setViewMode] = useState<ViewMode>('list');

  // Form state
  const [formState, setFormState] = useState<BatteryFormState>(INITIAL_BATTERY_FORM_STATE);
  const [formErrors, setFormErrors] = useState<Record<string, string>>({});

  // Log form state
  const [showLogModal, setShowLogModal] = useState(false);
  const [logFormState, setLogFormState] = useState<BatteryLogFormState>(createInitialLogFormState(4));
  const [batteryPendingDelete, setBatteryPendingDelete] = useState<Battery | null>(null);
  const [isDeletingBattery, setIsDeletingBattery] = useState(false);
  const listContainerRef = useRef<HTMLDivElement | null>(null);
  const formDialogRef = useRef<HTMLDivElement | null>(null);
  const detailDialogRef = useRef<HTMLDivElement | null>(null);
  const lastFocusedElementBeforeModalRef = useRef<HTMLElement | null>(null);

  // Filter state
  const [searchQuery, setSearchQuery] = useState('');
  const [filterChemistry, setFilterChemistry] = useState<BatteryChemistry | ''>('');
  const [filterCells, setFilterCells] = useState<number | ''>('');
  const [sortBy, setSortBy] = useState<'name' | 'created_at' | 'capacity_mah' | 'cells'>('created_at');

  // Load batteries
  const loadBatteries = useCallback(async () => {
    setIsLoading(true);
    try {
      const resolvedSortOrder: 'asc' | 'desc' = sortBy === 'created_at' ? 'desc' : 'asc';
      const response = await getBatteries({
        query: searchQuery.trim() || undefined,
        chemistry: filterChemistry || undefined,
        cells: filterCells || undefined,
        sort_by: sortBy,
        sort_order: resolvedSortOrder,
      });
      setBatteries(response.batteries);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load batteries';
      onError?.(message);
    } finally {
      setIsLoading(false);
    }
  }, [searchQuery, filterChemistry, filterCells, sortBy, onError]);

  useEffect(() => {
    loadBatteries();
  }, [loadBatteries]);

  // Load logs when battery is selected
  const loadLogs = useCallback(async () => {
    if (!selectedBattery) {
      setLogs([]);
      return;
    }
    setIsLogsLoading(true);
    try {
      const batteryLogs = await getBatteryLogs(selectedBattery.id);
      setLogs(batteryLogs);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load logs';
      onError?.(message);
    } finally {
      setIsLogsLoading(false);
    }
  }, [selectedBattery, onError]);

  useEffect(() => {
    if (viewMode === 'detail' && selectedBattery) {
      loadLogs();
    }
  }, [viewMode, selectedBattery, loadLogs]);

  useEffect(() => {
    const listContainer = listContainerRef.current;
    if (!listContainer) return;

    if (viewMode !== 'list') {
      listContainer.setAttribute('aria-hidden', 'true');
      listContainer.setAttribute('inert', '');
    } else {
      listContainer.removeAttribute('aria-hidden');
      listContainer.removeAttribute('inert');
    }
  }, [viewMode]);

  useEffect(() => {
    const activeDialog =
      viewMode === 'create' || viewMode === 'edit'
        ? formDialogRef.current
        : viewMode === 'detail'
          ? detailDialogRef.current
          : null;

    if (!activeDialog) return;

    lastFocusedElementBeforeModalRef.current = document.activeElement instanceof HTMLElement
      ? document.activeElement
      : null;

    const focusableElements = activeDialog.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR);
    const firstElement = focusableElements[0] ?? activeDialog;
    firstElement.focus();

    const handleTabKey = (event: KeyboardEvent) => {
      if (event.key !== 'Tab') return;

      const elements = activeDialog.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR);
      if (elements.length === 0) {
        event.preventDefault();
        return;
      }

      const first = elements[0];
      const last = elements[elements.length - 1];
      const activeElement = document.activeElement;

      if (event.shiftKey) {
        if (activeElement === first || !activeDialog.contains(activeElement)) {
          event.preventDefault();
          last.focus();
        }
        return;
      }

      if (activeElement === last || !activeDialog.contains(activeElement)) {
        event.preventDefault();
        first.focus();
      }
    };

    document.addEventListener('keydown', handleTabKey);
    return () => {
      document.removeEventListener('keydown', handleTabKey);
      lastFocusedElementBeforeModalRef.current?.focus();
    };
  }, [viewMode]);

  // Form validation
  const validateForm = (): boolean => {
    const errors: Record<string, string> = {};

    if (!formState.name.trim()) {
      errors.name = 'Name is required';
    }
    if (formState.capacity_mah < 1 || formState.capacity_mah > 50000) {
      errors.capacity_mah = 'Capacity must be between 1 and 50000 mAh';
    }
    if (formState.cells < 1 || formState.cells > 8) {
      errors.cells = 'Cell count must be between 1 and 8';
    }

    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const closeDetailModal = useCallback(() => {
    setViewMode('list');
    setShowLogModal(false);
    setBatteryPendingDelete(null);
  }, []);

  const closeFormModal = useCallback(() => {
    setViewMode(prev => (prev === 'edit' ? 'detail' : 'list'));
    setFormErrors({});
  }, []);

  // Handlers
  const handleViewBattery = async (battery: Battery) => {
    try {
      const full = await getBattery(battery.id);
      setSelectedBattery(full);
      setViewMode('detail');
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load battery';
      onError?.(message);
    }
  };

  const handleEditBattery = (battery: Battery) => {
    setFormState({
      name: battery.name,
      chemistry: battery.chemistry,
      cells: battery.cells,
      capacity_mah: battery.capacity_mah,
      c_rating: battery.c_rating?.toString() || '',
      weight_grams: battery.weight_grams?.toString() || '',
      brand: battery.brand || '',
      model: battery.model || '',
      purchase_date: battery.purchase_date?.split('T')[0] || '',
      notes: battery.notes || '',
    });
    setSelectedBattery(battery);
    setViewMode('edit');
  };

  const handleCreateBattery = async () => {
    if (!validateForm()) return;

    try {
      const newBattery = await createBattery({
        name: formState.name,
        chemistry: formState.chemistry,
        cells: formState.cells,
        capacity_mah: formState.capacity_mah,
        c_rating: formState.c_rating ? parseInt(formState.c_rating, 10) : undefined,
        weight_grams: formState.weight_grams ? parseInt(formState.weight_grams, 10) : undefined,
        brand: formState.brand || undefined,
        model: formState.model || undefined,
        purchase_date: formState.purchase_date || undefined,
        notes: formState.notes || undefined,
      });
      setBatteries(prev => [newBattery, ...prev]);
      setFormState(INITIAL_BATTERY_FORM_STATE);
      setSelectedBattery(null);
      setFormErrors({});
      setViewMode('list');
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to create battery';
      onError?.(message);
    }
  };

  const handleUpdateBattery = async () => {
    if (!selectedBattery || !validateForm()) return;

    try {
      const updated = await updateBattery(selectedBattery.id, {
        name: formState.name,
        chemistry: formState.chemistry,
        cells: formState.cells,
        capacity_mah: formState.capacity_mah,
        c_rating: formState.c_rating ? parseInt(formState.c_rating, 10) : undefined,
        weight_grams: formState.weight_grams ? parseInt(formState.weight_grams, 10) : undefined,
        brand: formState.brand || undefined,
        model: formState.model || undefined,
        purchase_date: formState.purchase_date || undefined,
        notes: formState.notes || undefined,
      });
      setBatteries(prev => prev.map(b => b.id === updated.id ? updated : b));
      setSelectedBattery(updated);
      setViewMode('detail');
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to update battery';
      onError?.(message);
    }
  };

  const handleDeleteBattery = async () => {
    if (!batteryPendingDelete || isDeletingBattery) return;
    setIsDeletingBattery(true);
    try {
      await deleteBattery(batteryPendingDelete.id);
      setBatteries(prev => prev.filter(b => b.id !== batteryPendingDelete.id));
      if (selectedBattery?.id === batteryPendingDelete.id) {
        setSelectedBattery(null);
        setViewMode('list');
      }
      setBatteryPendingDelete(null);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to delete battery';
      onError?.(message);
    } finally {
      setIsDeletingBattery(false);
    }
  };

  const handlePrintLabel = async (battery: Battery, size: LabelSize) => {
    try {
      await printBatteryLabel(battery.id, size);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      alert('Failed to print label: ' + message);
    }
  };

  // Log handlers
  const handleOpenLogModal = () => {
    if (!selectedBattery) return;
    setLogFormState(createInitialLogFormState(selectedBattery.cells));
    setShowLogModal(true);
  };

  const handleCreateLog = async () => {
    if (!selectedBattery) return;

    try {
      const irValues = logFormState.ir_milliohms
        .map(v => v ? parseFloat(v) : null)
        .filter((v): v is number => v !== null && !isNaN(v));

      const voltage = logFormState.min_cell_v ? parseFloat(logFormState.min_cell_v) : undefined;

      const newLog = await createBatteryLog(selectedBattery.id, {
        log_date: logFormState.log_date,
        cycle_count: logFormState.cycle_count ? parseInt(logFormState.cycle_count, 10) : undefined,
        ir_milliohms: irValues.length > 0 ? irValues : undefined,
        min_cell_v: voltage,
        max_cell_v: logFormState.max_cell_v ? parseFloat(logFormState.max_cell_v) : voltage,
        storage_voltage_ok: logFormState.storage_voltage_ok,
        notes: logFormState.notes || undefined,
      });
      setLogs(prev => [newLog, ...prev]);
      setShowLogModal(false);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to create log';
      onError?.(message);
    }
  };

  const handleDeleteLog = async (log: BatteryLog) => {
    if (!selectedBattery) return;
    if (!confirm('Delete this log entry?')) return;

    try {
      await deleteBatteryLog(selectedBattery.id, log.id);
      setLogs(prev => prev.filter(l => l.id !== log.id));
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to delete log';
      onError?.(message);
    }
  };

  // Render list view
  const renderList = () => (
    <div ref={listContainerRef} className="flex-1 min-h-0 flex flex-col overflow-hidden">
      <div className="px-4 md:px-6 py-4 border-b border-slate-800 bg-slate-900/95 backdrop-blur supports-[backdrop-filter]:bg-slate-900/85 flex-shrink-0">
        {/* Header */}
        <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
          <div>
            <h2 className="text-xl font-semibold text-white">Batteries</h2>
            <p className="text-sm text-slate-400">Track battery health, logs, and printable labels</p>
          </div>
          <button
            onClick={() => {
              setFormState(INITIAL_BATTERY_FORM_STATE);
              setFormErrors({});
              setViewMode('create');
            }}
            className="w-full sm:w-auto px-4 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors flex items-center justify-center gap-2"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            Add Battery
          </button>
        </div>

        {/* Filters */}
        <div className="mt-4 flex flex-wrap gap-4 p-4 bg-slate-800 rounded-lg border border-slate-700">
          <div className="w-full sm:max-w-xs">
            <label className="block text-xs font-medium text-slate-400 uppercase mb-1.5">Search</label>
            <input
              type="text"
              value={searchQuery}
              onChange={e => setSearchQuery(e.target.value)}
              placeholder="Name or battery code..."
              className="w-full px-3 py-1.5 bg-slate-900 border border-slate-700 rounded-lg text-sm text-white placeholder-slate-500 focus:outline-none focus:border-primary-500"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-slate-400 uppercase mb-1.5">Chemistry</label>
            <select
              value={filterChemistry}
              onChange={e => setFilterChemistry(e.target.value as BatteryChemistry | '')}
              className="px-3 py-1.5 bg-slate-900 border border-slate-700 rounded-lg text-sm text-white focus:outline-none focus:border-primary-500"
            >
              <option value="">All</option>
              {BATTERY_CHEMISTRY_OPTIONS.map(opt => (
                <option key={opt.value} value={opt.value}>{opt.label}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-xs font-medium text-slate-400 uppercase mb-1.5">Cells</label>
            <select
              value={filterCells}
              onChange={e => setFilterCells(e.target.value ? parseInt(e.target.value, 10) : '')}
              className="px-3 py-1.5 bg-slate-900 border border-slate-700 rounded-lg text-sm text-white focus:outline-none focus:border-primary-500"
            >
              <option value="">All</option>
              {CELL_COUNT_OPTIONS.map(count => (
                <option key={count} value={count}>{count}S</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-xs font-medium text-slate-400 uppercase mb-1.5">Sort By</label>
            <select
              value={sortBy}
              onChange={e => setSortBy(e.target.value as typeof sortBy)}
              className="px-3 py-1.5 bg-slate-900 border border-slate-700 rounded-lg text-sm text-white focus:outline-none focus:border-primary-500"
            >
              <option value="created_at">Date Added</option>
              <option value="name">Name</option>
              <option value="capacity_mah">Capacity</option>
              <option value="cells">Cells</option>
            </select>
          </div>
        </div>
      </div>

      <div
        className="flex-1 min-h-0 overflow-y-auto overscroll-y-contain p-4 md:p-6 pb-24"
        onScroll={() => {
          if (document.activeElement instanceof HTMLElement) {
            document.activeElement.blur();
          }
        }}
      >
        {/* Battery Grid */}
        {isLoading ? (
          <div className="text-center py-8 text-slate-400">Loading batteries...</div>
        ) : batteries.length === 0 ? (
          <div className="text-center py-8 text-slate-400">
            {searchQuery || filterChemistry || filterCells
              ? 'No batteries match the current filters.'
              : 'No batteries found. Add your first battery to get started!'}
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {batteries.map(battery => (
              <div
                key={battery.id}
                role="button"
                tabIndex={0}
                className="p-4 bg-slate-800 rounded-lg border border-slate-700 hover:border-slate-600 transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary-500"
                onClick={() => handleViewBattery(battery)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    handleViewBattery(battery);
                  }
                }}
              >
                <div className="flex justify-between items-start mb-2">
                  <h3 className="font-medium text-white">{battery.name}</h3>
                  <span className="text-xs px-2 py-1 bg-slate-700 text-slate-300 rounded font-mono">
                    {battery.battery_code}
                  </span>
                </div>
                <div className="text-sm text-slate-400 space-y-1">
                  <div className="flex gap-2">
                    <span className="px-2 py-0.5 bg-blue-500/20 text-blue-400 rounded text-xs">
                      {formatChemistry(battery.chemistry)}
                    </span>
                    <span className="px-2 py-0.5 bg-green-500/20 text-green-400 rounded text-xs">
                      {formatCellCount(battery.cells)}
                    </span>
                    <span className="px-2 py-0.5 bg-purple-500/20 text-purple-400 rounded text-xs">
                      {formatCapacity(battery.capacity_mah)}
                    </span>
                  </div>
                  {battery.brand && (
                    <p className="text-slate-500">{battery.brand} {battery.model}</p>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );

  // Render form (create/edit)
  const renderForm = () => (
    <div className="bg-slate-800 border border-slate-700 rounded-2xl w-full max-h-[92vh] overflow-hidden flex flex-col">
      <div className="flex items-center justify-between px-6 py-4 border-b border-slate-700">
        <h2 id="battery-form-modal-title" className="text-lg font-semibold text-white">
          {viewMode === 'create' ? 'Add New Battery' : 'Edit Battery'}
        </h2>
        <button
          onClick={closeFormModal}
          aria-label="Close battery form"
          className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <div className="p-6 space-y-4 overflow-y-auto">
        {/* Name */}
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1">Name *</label>
          <input
            type="text"
            value={formState.name}
            onChange={e => setFormState(prev => ({ ...prev, name: e.target.value }))}
            className={`w-full px-3 py-2 bg-slate-900 border rounded-lg text-white placeholder-slate-500 focus:outline-none focus:border-primary-500 ${formErrors.name ? 'border-red-500' : 'border-slate-700'}`}
            placeholder="e.g., Race Pack 1"
          />
          {formErrors.name && <p className="text-red-400 text-sm mt-1">{formErrors.name}</p>}
        </div>

        {/* Chemistry and Cells */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">Chemistry *</label>
            <select
              value={formState.chemistry}
              onChange={e => setFormState(prev => ({ ...prev, chemistry: e.target.value as BatteryChemistry }))}
              className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white focus:outline-none focus:border-primary-500"
            >
              {BATTERY_CHEMISTRY_OPTIONS.map(opt => (
                <option key={opt.value} value={opt.value}>{opt.label}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">Cell Count *</label>
            <select
              value={formState.cells}
              onChange={e => setFormState(prev => ({ ...prev, cells: parseInt(e.target.value, 10) }))}
              className={`w-full px-3 py-2 bg-slate-900 border rounded-lg text-white focus:outline-none focus:border-primary-500 ${formErrors.cells ? 'border-red-500' : 'border-slate-700'}`}
            >
              {CELL_COUNT_OPTIONS.map(count => (
                <option key={count} value={count}>{count}S</option>
              ))}
            </select>
            {formErrors.cells && <p className="text-red-400 text-sm mt-1">{formErrors.cells}</p>}
          </div>
        </div>

        {/* Capacity and Discharge */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">Capacity (mAh) *</label>
            <input
              type="number"
              value={formState.capacity_mah}
              onChange={e => setFormState(prev => ({ ...prev, capacity_mah: parseInt(e.target.value, 10) || 0 }))}
              className={`w-full px-3 py-2 bg-slate-900 border rounded-lg text-white focus:outline-none focus:border-primary-500 ${formErrors.capacity_mah ? 'border-red-500' : 'border-slate-700'}`}
              min="1"
              max="50000"
            />
            {formErrors.capacity_mah && <p className="text-red-400 text-sm mt-1">{formErrors.capacity_mah}</p>}
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">Discharge Rating</label>
            <input
              type="text"
              value={formState.c_rating}
              onChange={e => setFormState(prev => ({ ...prev, c_rating: e.target.value }))}
              className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:border-primary-500"
              placeholder="e.g., 75C"
            />
          </div>
        </div>

        {/* Brand and Model */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">Brand</label>
            <input
              type="text"
              value={formState.brand}
              onChange={e => setFormState(prev => ({ ...prev, brand: e.target.value }))}
              className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:border-primary-500"
              placeholder="e.g., CNHL"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">Model</label>
            <input
              type="text"
              value={formState.model}
              onChange={e => setFormState(prev => ({ ...prev, model: e.target.value }))}
              className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:border-primary-500"
              placeholder="e.g., Black Series"
            />
          </div>
        </div>

        {/* Weight and Purchase Date */}
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">Weight (grams)</label>
            <input
              type="number"
              value={formState.weight_grams}
              onChange={e => setFormState(prev => ({ ...prev, weight_grams: e.target.value }))}
              className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:border-primary-500"
              placeholder="e.g., 185"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-300 mb-1">Purchase Date</label>
            <input
              type="date"
              value={formState.purchase_date}
              onChange={e => setFormState(prev => ({ ...prev, purchase_date: e.target.value }))}
              className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white focus:outline-none focus:border-primary-500"
            />
          </div>
        </div>

        {/* Notes */}
        <div>
          <label className="block text-sm font-medium text-slate-300 mb-1">Notes</label>
          <textarea
            value={formState.notes}
            onChange={e => setFormState(prev => ({ ...prev, notes: e.target.value }))}
            className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:border-primary-500"
            rows={3}
            placeholder="Any additional notes..."
          />
        </div>

        {/* Submit Button */}
        <div className="flex gap-4 pt-4">
          <button
            onClick={viewMode === 'create' ? handleCreateBattery : handleUpdateBattery}
            className="px-6 py-2 bg-primary-600 text-white rounded-lg hover:bg-primary-700 transition-colors"
          >
            {viewMode === 'create' ? 'Create Battery' : 'Save Changes'}
          </button>
        </div>
      </div>
    </div>
  );

  // Render detail view
  const renderDetail = () => {
    if (!selectedBattery) return null;

    return (
      <div className="h-full flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between gap-4 px-4 md:px-6 py-4 border-b border-slate-700 flex-shrink-0">
          <div className="flex items-center gap-3 md:gap-4 min-w-0">
            <h2 id="battery-detail-modal-title" className="text-xl font-semibold text-white truncate">{selectedBattery.name}</h2>
            <span className="text-xs sm:text-sm px-2 py-1 bg-slate-700 text-slate-300 rounded font-mono">
              {selectedBattery.battery_code}
            </span>
          </div>
          <button
            onClick={closeDetailModal}
            aria-label="Close battery details"
            className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
          >
            <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div className="flex-1 min-h-0 overflow-y-auto px-4 md:px-6 py-4 md:py-6">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 md:gap-6">
            {/* Battery Info */}
            <div className="bg-slate-800 rounded-lg border border-slate-700 p-6 md:h-[320px] overflow-y-auto">
              <h3 className="font-medium text-white mb-4">Battery Details</h3>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
                <div>
                  <span className="text-slate-400">Chemistry:</span>
                  <span className="ml-2 font-medium text-white">{formatChemistry(selectedBattery.chemistry)}</span>
                </div>
                <div>
                  <span className="text-slate-400">Cells:</span>
                  <span className="ml-2 font-medium text-white">{formatCellCount(selectedBattery.cells)}</span>
                </div>
                <div>
                  <span className="text-slate-400">Capacity:</span>
                  <span className="ml-2 font-medium text-white">{formatCapacity(selectedBattery.capacity_mah)}</span>
                </div>
                {selectedBattery.c_rating && (
                  <div>
                    <span className="text-slate-400">Discharge:</span>
                    <span className="ml-2 font-medium text-white">{selectedBattery.c_rating}</span>
                  </div>
                )}
                {selectedBattery.brand && (
                  <div>
                    <span className="text-slate-400">Brand:</span>
                    <span className="ml-2 font-medium text-white">{selectedBattery.brand}</span>
                  </div>
                )}
                {selectedBattery.model && (
                  <div>
                    <span className="text-slate-400">Model:</span>
                    <span className="ml-2 font-medium text-white">{selectedBattery.model}</span>
                  </div>
                )}
                {selectedBattery.weight_grams && (
                  <div>
                    <span className="text-slate-400">Weight:</span>
                    <span className="ml-2 font-medium text-white">{selectedBattery.weight_grams}g</span>
                  </div>
                )}
                {selectedBattery.purchase_date && (
                  <div>
                    <span className="text-slate-400">Purchased:</span>
                    <span className="ml-2 font-medium text-white">
                      {new Date(selectedBattery.purchase_date).toLocaleDateString()}
                    </span>
                  </div>
                )}
              </div>

              {selectedBattery.notes && (
                <div className="mt-4 pt-4 border-t border-slate-700">
                  <span className="text-slate-400 text-sm">Notes:</span>
                  <p className="mt-1 text-slate-300">{selectedBattery.notes}</p>
                </div>
              )}
            </div>

            {/* Print Label */}
            <div className="bg-slate-800 rounded-lg border border-slate-700 p-6 md:h-[320px] flex flex-col">
              <h3 className="font-medium text-white mb-4">Print Label</h3>
              <p className="text-sm text-slate-400 mb-4">
                Generate a printable label with QR code for this battery.
              </p>
              <div className="space-y-2 mt-auto">
                <button
                  onClick={() => handlePrintLabel(selectedBattery, 'small')}
                  className="w-full px-4 py-2 border border-slate-600 text-slate-300 rounded-lg hover:bg-slate-700 text-sm transition-colors"
                >
                  Small Label (1" × 0.5")
                </button>
                <button
                  onClick={() => handlePrintLabel(selectedBattery, 'standard')}
                  className="w-full px-4 py-2 border border-slate-600 text-slate-300 rounded-lg hover:bg-slate-700 text-sm transition-colors"
                >
                  Standard Label (2" × 1")
                </button>
              </div>
            </div>

            {/* Health Logs */}
            <div className="bg-slate-800 rounded-lg border border-slate-700 p-6 md:h-[320px] flex flex-col">
              <div className="flex justify-between items-center mb-4 flex-shrink-0">
                <h3 className="font-medium text-white">Health Logs</h3>
                <button
                  onClick={handleOpenLogModal}
                  className="text-sm px-3 py-1 bg-green-600 text-white rounded hover:bg-green-700 transition-colors"
                >
                  + Add Log
                </button>
              </div>

              {isLogsLoading ? (
                <div className="flex-1 flex items-center">
                  <p className="text-slate-400 text-sm">Loading logs...</p>
                </div>
              ) : logs.length === 0 ? (
                <div className="flex-1 flex items-center">
                  <p className="text-slate-400 text-sm">No health logs yet. Add your first log to track battery health.</p>
                </div>
              ) : (
                <div className="flex-1 min-h-0 overflow-y-auto pr-1 space-y-3">
                  {logs.map(log => (
                    <div key={log.id} className="p-3 bg-slate-900 rounded-lg border border-slate-700">
                      <div className="flex justify-between items-start">
                        <div className="text-sm">
                          <span className="font-medium text-white">
                            {new Date(log.log_date).toLocaleDateString()}
                          </span>
                          {log.cycle_count !== undefined && (
                            <span className="ml-3 text-slate-400">Cycles: {log.cycle_count}</span>
                          )}
                          {(log.min_cell_v !== undefined || log.max_cell_v !== undefined) && (
                            <span className="ml-3 text-slate-400">
                              Voltage: {
                                log.min_cell_v !== undefined && log.max_cell_v !== undefined
                                  ? (log.min_cell_v === log.max_cell_v
                                    ? `${log.min_cell_v.toFixed(2)}V/cell`
                                    : `${log.min_cell_v.toFixed(2)}-${log.max_cell_v.toFixed(2)}V/cell`)
                                  : log.min_cell_v !== undefined
                                    ? `${log.min_cell_v.toFixed(2)}V/cell`
                                    : `${log.max_cell_v!.toFixed(2)}V/cell`
                              }
                            </span>
                          )}
                          {log.storage_voltage_ok !== undefined && (
                            <span className={`ml-3 ${log.storage_voltage_ok ? 'text-green-400' : 'text-red-400'}`}>
                              Storage: {log.storage_voltage_ok ? '✓' : '✗'}
                            </span>
                          )}
                        </div>
                        <button
                          onClick={() => handleDeleteLog(log)}
                          className="text-red-400 hover:text-red-300 text-sm"
                        >
                          ×
                        </button>
                      </div>
                      {log.ir_milliohms && log.ir_milliohms.length > 0 && (
                        <div className="mt-2 text-xs text-slate-400">
                          IR (mΩ): {log.ir_milliohms.map((ir, i) => (
                            <span key={i} className="ml-1 px-1 bg-slate-700 rounded">
                              C{i + 1}: {ir}
                            </span>
                          ))}
                        </div>
                      )}
                      {log.notes && (
                        <p className="mt-2 text-sm text-slate-400">{log.notes}</p>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Quick Stats */}
            <div className="bg-slate-800 rounded-lg border border-slate-700 p-6 md:h-[320px]">
              <h3 className="font-medium text-white mb-4">Quick Stats</h3>
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-slate-400">Total Logs:</span>
                  <span className="font-medium text-white">{logs.length}</span>
                </div>
                {logs.length > 0 && logs[0].cycle_count !== undefined && (
                  <div className="flex justify-between">
                    <span className="text-slate-400">Latest Cycles:</span>
                    <span className="font-medium text-white">{logs[0].cycle_count}</span>
                  </div>
                )}
                <div className="flex justify-between">
                  <span className="text-slate-400">Added:</span>
                  <span className="font-medium text-white">
                    {new Date(selectedBattery.created_at).toLocaleDateString()}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="border-t border-slate-700 bg-slate-900/95 px-4 md:px-6 py-4 flex-shrink-0">
          <div className="flex flex-col sm:flex-row sm:justify-end gap-3">
            <button
              onClick={() => handleEditBattery(selectedBattery)}
              className="w-full sm:w-auto px-6 py-3 text-base font-medium text-white bg-slate-700 hover:bg-slate-600 rounded-lg transition-colors"
            >
              Edit
            </button>
            <button
              onClick={() => setBatteryPendingDelete(selectedBattery)}
              className="w-full sm:w-auto px-6 py-3 text-base font-medium bg-red-600 hover:bg-red-500 text-white rounded-lg transition-colors"
            >
              Delete
            </button>
          </div>
        </div>

        {/* Log Modal */}
        {showLogModal && (
          <div 
            className="fixed inset-0 bg-black bg-opacity-70 flex items-center justify-center z-50"
            onClick={() => setShowLogModal(false)}
            role="dialog"
            aria-modal="true"
            aria-labelledby="log-modal-title"
          >
            <div 
              className="bg-slate-800 rounded-lg p-6 max-w-md w-full mx-4 border border-slate-700"
              onClick={(e) => e.stopPropagation()}
              onKeyDown={(e) => {
                if (e.key === 'Escape') {
                  setShowLogModal(false);
                }
              }}
            >
              <div className="flex items-center justify-between mb-4">
                <h3 id="log-modal-title" className="text-lg font-medium text-white">Add Health Log</h3>
                <button
                  onClick={() => setShowLogModal(false)}
                  aria-label="Close add health log modal"
                  className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
                >
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>

              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-1">Date</label>
                  <input
                    type="date"
                    value={logFormState.log_date}
                    onChange={e => setLogFormState(prev => ({ ...prev, log_date: e.target.value }))}
                    className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white focus:outline-none focus:border-primary-500"
                  />
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-1">Cycle Count</label>
                    <input
                      type="number"
                      value={logFormState.cycle_count}
                      onChange={e => setLogFormState(prev => ({ ...prev, cycle_count: e.target.value }))}
                      className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:border-primary-500"
                      placeholder="e.g., 50"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-1">Min Voltage/Cell</label>
                    <input
                      type="number"
                      step="0.01"
                      value={logFormState.min_cell_v}
                      onChange={e => setLogFormState(prev => ({ ...prev, min_cell_v: e.target.value }))}
                      className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:border-primary-500"
                      placeholder="e.g., 3.80"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-slate-300 mb-1">Max Voltage/Cell</label>
                    <input
                      type="number"
                      step="0.01"
                      value={logFormState.max_cell_v}
                      onChange={e => setLogFormState(prev => ({ ...prev, max_cell_v: e.target.value }))}
                      className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:border-primary-500"
                      placeholder="e.g., 3.85"
                    />
                  </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-1">
                    Internal Resistance (mΩ per cell)
                  </label>
                  <div className="grid grid-cols-4 gap-2">
                    {logFormState.ir_milliohms.map((ir, i) => (
                      <input
                        key={i}
                        type="number"
                        step="0.1"
                        value={ir}
                        onChange={e => {
                          const newIR = [...logFormState.ir_milliohms];
                          newIR[i] = e.target.value;
                          setLogFormState(prev => ({ ...prev, ir_milliohms: newIR }));
                        }}
                        className="px-2 py-1 bg-slate-900 border border-slate-700 rounded text-sm text-white placeholder-slate-500 focus:outline-none focus:border-primary-500"
                        placeholder={`C${i + 1}`}
                      />
                    ))}
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    id="storage_voltage_ok"
                    checked={logFormState.storage_voltage_ok}
                    onChange={e => setLogFormState(prev => ({ ...prev, storage_voltage_ok: e.target.checked }))}
                    className="rounded bg-slate-900 border-slate-700 text-primary-600 focus:ring-primary-500"
                  />
                  <label htmlFor="storage_voltage_ok" className="text-sm text-slate-300">
                    Storage voltage OK
                  </label>
                </div>

                <div>
                  <label className="block text-sm font-medium text-slate-300 mb-1">Notes</label>
                  <textarea
                    value={logFormState.notes}
                    onChange={e => setLogFormState(prev => ({ ...prev, notes: e.target.value }))}
                    className="w-full px-3 py-2 bg-slate-900 border border-slate-700 rounded-lg text-white placeholder-slate-500 focus:outline-none focus:border-primary-500"
                    rows={2}
                    placeholder="Any observations..."
                  />
                </div>
              </div>

              <div className="flex gap-4 mt-6">
                <button
                  onClick={handleCreateLog}
                  className="w-full px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors"
                >
                  Add Log
                </button>
              </div>
            </div>
          </div>
        )}

        {/* Delete Battery Modal */}
        {batteryPendingDelete && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <div
              className="absolute inset-0 bg-black/70"
              onClick={() => {
                if (!isDeletingBattery) {
                  setBatteryPendingDelete(null);
                }
              }}
            />
            <div
              role="dialog"
              aria-modal="true"
              aria-labelledby="battery-delete-dialog-title"
              aria-describedby="battery-delete-dialog-description"
              className="relative bg-slate-800 rounded-xl p-6 max-w-md w-full shadow-2xl border border-red-500/50"
            >
              <div className="flex items-start justify-between gap-3 mb-4">
                <div className="flex items-center gap-3">
                  <div className="w-10 h-10 bg-red-500/20 rounded-full flex items-center justify-center">
                    <svg className="w-5 h-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                    </svg>
                  </div>
                  <h3 id="battery-delete-dialog-title" className="text-lg font-semibold text-white">Delete Battery?</h3>
                </div>
                <button
                  type="button"
                  onClick={() => setBatteryPendingDelete(null)}
                  disabled={isDeletingBattery}
                  aria-label="Close delete battery modal"
                  className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors disabled:opacity-50"
                >
                  <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
              <p id="battery-delete-dialog-description" className="text-slate-300 mb-5">
                Are you sure you want to delete <span className="text-white font-medium">{batteryPendingDelete.name}</span>?
              </p>
              <div className="flex">
                <button
                  type="button"
                  onClick={() => void handleDeleteBattery()}
                  disabled={isDeletingBattery}
                  className="w-full px-4 py-2 bg-red-600 hover:bg-red-500 text-white rounded-lg font-medium transition-colors disabled:opacity-50"
                >
                  {isDeletingBattery ? 'Deleting...' : 'Delete Battery'}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    );
  };

  // Main render
  return (
    <div className="flex flex-col flex-1 min-h-0 overflow-hidden">
      {renderList()}
      {typeof document !== 'undefined' && (viewMode === 'create' || viewMode === 'edit') && createPortal(
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby="battery-form-modal-title"
          className="fixed inset-0 z-[70] flex items-start md:items-center justify-center p-4 md:p-6"
        >
          <div
            className="absolute inset-0 bg-black/60 backdrop-blur-sm"
            onClick={closeFormModal}
          />
          <div ref={formDialogRef} tabIndex={-1} className="relative w-full max-w-3xl">
            {renderForm()}
          </div>
        </div>,
        document.body
      )}
      {typeof document !== 'undefined' && viewMode === 'detail' && selectedBattery && createPortal(
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby="battery-detail-modal-title"
          className="fixed inset-0 z-[70] flex items-start md:items-center justify-center p-4 md:p-6"
        >
          <div
            className="absolute inset-0 bg-black/60 backdrop-blur-sm"
            onClick={closeDetailModal}
          />
          <div
            ref={detailDialogRef}
            tabIndex={-1}
            className="relative w-full max-w-6xl h-[92vh] max-h-[92vh] overflow-hidden bg-slate-900 border border-slate-700 rounded-2xl flex flex-col"
          >
            {renderDetail()}
          </div>
        </div>,
        document.body
      )}
    </div>
  );
}

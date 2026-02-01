// Battery chemistry types
export type BatteryChemistry = 'LIPO' | 'LIPO_HV' | 'LIION';

export const BATTERY_CHEMISTRY_LABELS: Record<BatteryChemistry, string> = {
  LIPO: 'LiPo',
  LIPO_HV: 'LiPo HV',
  LIION: 'Li-Ion',
};

export const BATTERY_CHEMISTRY_OPTIONS: { value: BatteryChemistry; label: string }[] = [
  { value: 'LIPO', label: 'LiPo' },
  { value: 'LIPO_HV', label: 'LiPo HV' },
  { value: 'LIION', label: 'Li-Ion' },
];

// Cell count options (1S-8S)
export const CELL_COUNT_OPTIONS = [1, 2, 3, 4, 5, 6, 7, 8] as const;

// Label sizes
export type LabelSize = 'small' | 'standard';

// Battery model
export interface Battery {
  id: string;
  user_id: string;
  battery_code: string;
  name: string;
  chemistry: BatteryChemistry;
  cells: number;
  capacity_mah: number;
  c_rating?: string;
  weight_grams?: number;
  brand?: string;
  model?: string;
  purchase_date?: string;
  notes?: string;
  created_at: string;
  updated_at: string;
}

// Battery log for health tracking
export interface BatteryLog {
  id: string;
  battery_id: string;
  log_date: string;
  cycle_count?: number;
  ir_milliohms?: number[];
  min_cell_v?: number;
  max_cell_v?: number;
  storage_voltage_ok?: boolean;
  notes?: string;
  created_at: string;
}

// Create battery params
export interface CreateBatteryParams {
  name: string;
  chemistry: BatteryChemistry;
  cells: number;
  capacity_mah: number;
  c_rating?: string;
  weight_grams?: number;
  brand?: string;
  model?: string;
  purchase_date?: string;
  notes?: string;
}

// Update battery params
export interface UpdateBatteryParams {
  name?: string;
  chemistry?: BatteryChemistry;
  cells?: number;
  capacity_mah?: number;
  c_rating?: string;
  weight_grams?: number;
  brand?: string;
  model?: string;
  purchase_date?: string;
  notes?: string;
}

// Battery list filter/sort params
export interface BatteryListParams {
  chemistry?: BatteryChemistry;
  cells?: number;
  min_capacity?: number;
  max_capacity?: number;
  sort_by?: 'name' | 'created_at' | 'capacity_mah' | 'cells';
  sort_order?: 'asc' | 'desc';
  limit?: number;
  offset?: number;
}

// Create battery log params
export interface CreateBatteryLogParams {
  log_date: string;
  cycle_count?: number;
  ir_milliohms?: number[];
  min_cell_v?: number;
  max_cell_v?: number;
  storage_voltage_ok?: boolean;
  notes?: string;
}

// API response for battery list
export interface BatteryListResponse {
  batteries: Battery[];
  total: number;
}

// Form state for battery creation/editing
export interface BatteryFormState {
  name: string;
  chemistry: BatteryChemistry;
  cells: number;
  capacity_mah: number;
  c_rating: string;
  weight_grams: string;
  brand: string;
  model: string;
  purchase_date: string;
  notes: string;
}

// Initial form state
export const INITIAL_BATTERY_FORM_STATE: BatteryFormState = {
  name: '',
  chemistry: 'LIPO',
  cells: 4,
  capacity_mah: 1500,
  c_rating: '',
  weight_grams: '',
  brand: '',
  model: '',
  purchase_date: '',
  notes: '',
};

// Form state for battery log
export interface BatteryLogFormState {
  log_date: string;
  cycle_count: string;
  ir_milliohms: string[];
  min_cell_v: string;
  max_cell_v: string;
  storage_voltage_ok: boolean;
  notes: string;
}

// Create initial log form state based on cell count
export const createInitialLogFormState = (cells: number): BatteryLogFormState => ({
  log_date: new Date().toISOString().split('T')[0],
  cycle_count: '',
  ir_milliohms: Array(cells).fill(''),
  min_cell_v: '',
  max_cell_v: '',
  storage_voltage_ok: true,
  notes: '',
});

// Helper to format battery display name
export const formatBatteryName = (battery: Battery): string => {
  return `${battery.name} (${battery.cells}S ${battery.capacity_mah}mAh)`;
};

// Helper to format chemistry for display
export const formatChemistry = (chemistry: BatteryChemistry): string => {
  return BATTERY_CHEMISTRY_LABELS[chemistry] || chemistry;
};

// Helper to format cell count
export const formatCellCount = (cells: number): string => {
  return `${cells}S`;
};

// Helper to format capacity
export const formatCapacity = (capacityMah: number): string => {
  if (capacityMah >= 1000) {
    return `${(capacityMah / 1000).toFixed(capacityMah % 1000 === 0 ? 0 : 1)}Ah`;
  }
  return `${capacityMah}mAh`;
};

// Validation helpers
export const isValidCellCount = (cells: number): boolean => {
  return cells >= 1 && cells <= 8;
};

export const isValidCapacity = (capacityMah: number): boolean => {
  return capacityMah > 0 && capacityMah <= 50000;
};

export const isValidIRArray = (irArray: number[], cellCount: number): boolean => {
  return irArray.length === cellCount && irArray.every(ir => ir >= 0);
};

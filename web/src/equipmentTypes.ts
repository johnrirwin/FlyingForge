// Equipment and inventory types matching the Go server schema

// Equipment categories
export type EquipmentCategory =
  | 'frames'
  | 'vtx'
  | 'flight_controllers'
  | 'esc'
  | 'aio'
  | 'motors'
  | 'propellers'
  | 'receivers'
  | 'cameras'
  | 'antennas'
  | 'accessories';

export const EQUIPMENT_CATEGORIES: { value: EquipmentCategory; label: string }[] = [
  { value: 'frames', label: 'Frames' },
  { value: 'vtx', label: 'Video Transmitters' },
  { value: 'flight_controllers', label: 'Flight Controllers' },
  { value: 'esc', label: 'ESCs' },
  { value: 'aio', label: 'AIO (FC/ESC)' },
  { value: 'motors', label: 'Motors' },
  { value: 'propellers', label: 'Propellers' },
  { value: 'receivers', label: 'Receivers' },
  { value: 'cameras', label: 'Cameras' },
  { value: 'antennas', label: 'Antennas' },
  { value: 'accessories', label: 'Accessories' },
];

// Item condition
export type ItemCondition = 'new' | 'used' | 'broken' | 'spare';

export const ITEM_CONDITIONS: { value: ItemCondition; label: string; color: string }[] = [
  { value: 'new', label: 'New', color: 'green' },
  { value: 'used', label: 'Used', color: 'yellow' },
  { value: 'broken', label: 'Broken', color: 'red' },
  { value: 'spare', label: 'Spare', color: 'blue' },
];

// Equipment item from seller search
export interface EquipmentItem {
  id: string;
  name: string;
  category: EquipmentCategory;
  manufacturer: string;
  price: number;
  currency: string;
  seller: string;
  sellerId: string;
  productUrl: string;
  imageUrl?: string;
  keySpecs?: Record<string, unknown>;
  inStock: boolean;
  stockQty?: number;
  lastChecked: string;
  description?: string;
  sku?: string;
  rating?: number;
  reviewCount?: number;
}

// Seller/retailer info
export interface SellerInfo {
  id: string;
  name: string;
  url: string;
  description: string;
  logoUrl?: string;
  categories: string[];
  enabled: boolean;
  region?: string;
}

// Equipment search params
export interface EquipmentSearchParams {
  query?: string;
  category?: EquipmentCategory;
  seller?: string;
  minPrice?: number;
  maxPrice?: number;
  inStockOnly?: boolean;
  limit?: number;
  offset?: number;
  sort?: 'price_asc' | 'price_desc' | 'name' | 'newest';
}

// Equipment search response
export interface EquipmentSearchResponse {
  items: EquipmentItem[];
  totalCount: number;
  page: number;
  pageSize: number;
  query?: string;
  filters: {
    category?: string;
    seller?: string;
    priceRange?: [number, number];
    inStockOnly: boolean;
  };
}

// Sellers response
export interface SellersResponse {
  sellers: SellerInfo[];
  count: number;
}

// Inventory item (personal equipment)
export interface InventoryItem {
  id: string;
  userId?: string;
  name: string;
  category: EquipmentCategory;
  manufacturer?: string;
  quantity: number;
  condition: ItemCondition;
  notes?: string;
  catalogId?: string; // Link to gear catalog item
  buildId?: string;
  purchasePrice?: number;
  purchaseDate?: string;
  purchaseSeller?: string;
  productUrl?: string;
  imageUrl?: string;
  specs?: Record<string, unknown>;
  sourceEquipmentId?: string;
  createdAt: string;
  updatedAt: string;
}

// Add inventory item params
export interface AddInventoryParams {
  name: string;
  category: EquipmentCategory;
  manufacturer?: string;
  quantity?: number;
  condition?: ItemCondition;
  notes?: string;
  catalogId?: string; // Link to gear catalog item
  buildId?: string;
  purchasePrice?: number;
  purchaseDate?: string;
  purchaseSeller?: string;
  productUrl?: string;
  imageUrl?: string;
  specs?: Record<string, unknown>;
  sourceEquipmentId?: string;
}

// Update inventory item params
export interface UpdateInventoryParams {
  name?: string;
  category?: EquipmentCategory;
  manufacturer?: string;
  quantity?: number;
  condition?: ItemCondition;
  notes?: string;
  buildId?: string;
  purchasePrice?: number;
  purchaseDate?: string;
  purchaseSeller?: string;
  productUrl?: string;
  imageUrl?: string;
  specs?: Record<string, unknown>;
}

// Inventory filter params
export interface InventoryFilterParams {
  category?: EquipmentCategory;
  condition?: ItemCondition;
  buildId?: string;
  query?: string;
  limit?: number;
  offset?: number;
}

// Inventory response
export interface InventoryResponse {
  items: InventoryItem[];
  totalCount: number;
  categories?: Record<EquipmentCategory, number>;
}

// Inventory summary
export interface InventorySummary {
  totalItems: number;
  totalValue: number;
  byCategory: Record<EquipmentCategory, number>;
  byCondition: Record<ItemCondition, number>;
}

// App section navigation
export type AppSection = 'home' | 'getting-started' | 'dashboard' | 'news' | 'equipment' | 'gear-catalog' | 'inventory' | 'aircraft' | 'radio' | 'batteries' | 'social' | 'profile' | 'pilot-profile';

export const APP_SECTIONS: { value: AppSection; label: string; icon: string; requiresAuth?: boolean }[] = [
  { value: 'home', label: 'Home', icon: '🏠' },
  { value: 'getting-started', label: 'Taking Off', icon: '→' },
  { value: 'dashboard', label: 'Dashboard', icon: '📊', requiresAuth: true },
  { value: 'news', label: 'News', icon: '📰' },
  { value: 'equipment', label: 'Equipment', icon: '🛒' },
  { value: 'gear-catalog', label: 'Gear Catalog', icon: '📦' },
  { value: 'inventory', label: 'My Inventory', icon: '🎒', requiresAuth: true },
  { value: 'aircraft', label: 'My Aircraft', icon: '🚁', requiresAuth: true },
  { value: 'radio', label: 'My Radio', icon: '📻', requiresAuth: true },
  { value: 'batteries', label: 'Batteries', icon: '🔋', requiresAuth: true },
];

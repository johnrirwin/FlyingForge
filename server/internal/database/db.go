package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Config holds database configuration
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		Host:            "localhost",
		Port:            5432,
		User:            "postgres",
		Password:        "postgres",
		Database:        "mcp_drone",
		SSLMode:         "disable",
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}
}

// DB wraps the sql.DB connection
type DB struct {
	*sql.DB
	config Config
}

// New creates a new database connection
func New(config Config) (*DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.Database, config.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{DB: db, config: config}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// Migrate runs database migrations
func (db *DB) Migrate(ctx context.Context) error {
	migrations := []string{
		migrationUsers,
		migrationUserIdentities,
		migrationRefreshTokens,
		migrationSellers,
		migrationEquipmentItems,
		migrationInventoryItems,
		migrationIndexes,
		migrationAircraft,
		migrationAircraftComponents,
		migrationAircraftReceiverSettings,
		migrationAircraftIndexes,
		migrationAircraftImageStorage,
		migrationRadios,
		migrationRadioBackups,
		migrationRadioIndexes,
		migrationBatteries,
		migrationBatteryLogs,
		migrationBatteryIndexes,
		migrationUserProfiles,
		migrationSocialSettings,
		migrationFollows,
		migrationOrders,
		migrationCustomAvatarText,
		migrationFCConfigs,
		migrationAircraftTuningSnapshots,
		migrationTuningSnapshotDiffBackup,
		migrationDropPasswordHash,
		migrationGearCatalog,                               // Creates gear_catalog table
		migrationPgTrgm,                                    // Adds trigram search for gear_catalog
		migrationInventoryCatalogLink,                      // Adds FK to gear_catalog (depends on migrationGearCatalog)
		migrationGearCatalogBestFor,                        // Adds best_for column for drone type
		migrationGearCatalogMSRP,                           // Adds msrp column for price
		migrationGearCatalogCuration,                       // Adds image curation fields
		migrationUserIsAdmin,                               // Adds is_admin flag to users
		migrationUserIsGearAdmin,                           // Adds is_gear_admin flag to users
		migrationGearCatalogImageData,                      // Adds image_data binary storage for gear images
		migrationInventoryCatalogUnique,                    // Adds unique constraint on (user_id, catalog_id)
		migrationDropInventoryPurchaseDate,                 // Drops unused purchase_date column
		migrationDropInventoryCondition,                    // Drops unused condition column
		migrationImageAssets,                               // Adds centralized image asset storage + references
		migrationGearCatalogImageScanned,                   // Marks moderated user uploads as scanned for admin review
		migrationGearCatalogPublishedStatus,                // Normalizes catalog status to published/pending/removed
		migrationGearCatalogPublishAutoApproveScannedImage, // Aligns published items to approved image status
		migrationBuilds,                                    // Adds user/public/temp builds with part mappings
	}

	for i, migration := range migrations {
		if _, err := db.ExecContext(ctx, migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	return nil
}

// Migration SQL statements
const migrationUsers = `
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    avatar_url VARCHAR(1024),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_login_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(LOWER(email));
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
`

const migrationUserIdentities = `
CREATE TABLE IF NOT EXISTS user_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL,
    provider_subject VARCHAR(255) NOT NULL,
    provider_email VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(provider, provider_subject)
);

CREATE INDEX IF NOT EXISTS idx_user_identities_user ON user_identities(user_id);
CREATE INDEX IF NOT EXISTS idx_user_identities_provider ON user_identities(provider, provider_subject);
`

const migrationRefreshTokens = `
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_hash ON refresh_tokens(token_hash);
`

const migrationSellers = `
CREATE TABLE IF NOT EXISTS sellers (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    url VARCHAR(512) NOT NULL,
    description TEXT,
    logo_url VARCHAR(512),
    categories JSONB DEFAULT '[]',
    enabled BOOLEAN DEFAULT true,
    region VARCHAR(10),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
`

const migrationEquipmentItems = `
CREATE TABLE IF NOT EXISTS equipment_items (
    id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(512) NOT NULL,
    category VARCHAR(50) NOT NULL,
    manufacturer VARCHAR(255),
    price DECIMAL(10,2),
    currency VARCHAR(10) DEFAULT 'USD',
    seller_id VARCHAR(50) REFERENCES sellers(id),
    seller_name VARCHAR(255) NOT NULL,
    product_url VARCHAR(1024) NOT NULL,
    image_url VARCHAR(1024),
    key_specs JSONB DEFAULT '{}',
    in_stock BOOLEAN DEFAULT false,
    stock_qty INTEGER,
    sku VARCHAR(100),
    description TEXT,
    rating DECIMAL(3,2),
    review_count INTEGER,
    last_checked TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
`

const migrationInventoryItems = `
CREATE TABLE IF NOT EXISTS inventory_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(512) NOT NULL,
    category VARCHAR(50) NOT NULL,
    manufacturer VARCHAR(255),
    quantity INTEGER NOT NULL DEFAULT 1,
    notes TEXT,
    build_id VARCHAR(100),
    purchase_price DECIMAL(10,2),
    purchase_seller VARCHAR(255),
    product_url VARCHAR(1024),
    image_url VARCHAR(1024),
    specs JSONB DEFAULT '{}',
    source_equipment_id VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
`

const migrationIndexes = `
CREATE INDEX IF NOT EXISTS idx_equipment_category ON equipment_items(category);
CREATE INDEX IF NOT EXISTS idx_equipment_seller ON equipment_items(seller_id);
CREATE INDEX IF NOT EXISTS idx_equipment_manufacturer ON equipment_items(manufacturer);
CREATE INDEX IF NOT EXISTS idx_equipment_price ON equipment_items(price);
CREATE INDEX IF NOT EXISTS idx_equipment_in_stock ON equipment_items(in_stock);
CREATE INDEX IF NOT EXISTS idx_equipment_name_search ON equipment_items USING gin(to_tsvector('english', name));

CREATE INDEX IF NOT EXISTS idx_inventory_user ON inventory_items(user_id);
CREATE INDEX IF NOT EXISTS idx_inventory_category ON inventory_items(category);
CREATE INDEX IF NOT EXISTS idx_inventory_build ON inventory_items(build_id);
CREATE INDEX IF NOT EXISTS idx_inventory_name_search ON inventory_items USING gin(to_tsvector('english', name));
`

const migrationAircraft = `
CREATE TABLE IF NOT EXISTS aircraft (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    nickname VARCHAR(255),
    type VARCHAR(50),
    image_url VARCHAR(1024),
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
`

const migrationAircraftComponents = `
CREATE TABLE IF NOT EXISTS aircraft_components (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aircraft_id UUID NOT NULL REFERENCES aircraft(id) ON DELETE CASCADE,
    category VARCHAR(50) NOT NULL,
    inventory_item_id UUID REFERENCES inventory_items(id) ON DELETE SET NULL,
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(aircraft_id, category)
);
`

const migrationAircraftReceiverSettings = `
CREATE TABLE IF NOT EXISTS aircraft_receiver_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aircraft_id UUID NOT NULL REFERENCES aircraft(id) ON DELETE CASCADE UNIQUE,
    settings_json JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
`

const migrationAircraftIndexes = `
CREATE INDEX IF NOT EXISTS idx_aircraft_user ON aircraft(user_id);
CREATE INDEX IF NOT EXISTS idx_aircraft_type ON aircraft(type);
CREATE INDEX IF NOT EXISTS idx_aircraft_name_search ON aircraft USING gin(to_tsvector('english', name));
CREATE INDEX IF NOT EXISTS idx_aircraft_components_aircraft ON aircraft_components(aircraft_id);
CREATE INDEX IF NOT EXISTS idx_aircraft_components_inventory ON aircraft_components(inventory_item_id);
CREATE INDEX IF NOT EXISTS idx_aircraft_receiver_aircraft ON aircraft_receiver_settings(aircraft_id);
`

const migrationAircraftImageStorage = `
ALTER TABLE aircraft ADD COLUMN IF NOT EXISTS image_data BYTEA;
ALTER TABLE aircraft ADD COLUMN IF NOT EXISTS image_type VARCHAR(20);
ALTER TABLE aircraft DROP COLUMN IF EXISTS image_url;
`

const migrationRadios = `
CREATE TABLE IF NOT EXISTS radios (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    manufacturer VARCHAR(100) NOT NULL,
    model VARCHAR(255) NOT NULL,
    firmware_family VARCHAR(50),
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
`

const migrationRadioBackups = `
CREATE TABLE IF NOT EXISTS radio_backups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    radio_id UUID NOT NULL REFERENCES radios(id) ON DELETE CASCADE,
    backup_name VARCHAR(255) NOT NULL,
    backup_type VARCHAR(50) NOT NULL,
    file_name VARCHAR(512) NOT NULL,
    file_size BIGINT NOT NULL,
    checksum VARCHAR(128),
    storage_path VARCHAR(1024) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
`

const migrationRadioIndexes = `
CREATE INDEX IF NOT EXISTS idx_radios_user ON radios(user_id);
CREATE INDEX IF NOT EXISTS idx_radios_manufacturer ON radios(manufacturer);
CREATE INDEX IF NOT EXISTS idx_radio_backups_radio ON radio_backups(radio_id);
CREATE INDEX IF NOT EXISTS idx_radio_backups_created ON radio_backups(created_at DESC);
`

const migrationBatteries = `
CREATE TABLE IF NOT EXISTS batteries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    battery_code VARCHAR(20) NOT NULL,
    name VARCHAR(255),
    chemistry VARCHAR(20) NOT NULL,
    cells INTEGER NOT NULL CHECK (cells >= 1 AND cells <= 8),
    capacity_mah INTEGER NOT NULL CHECK (capacity_mah > 0 AND capacity_mah <= 50000),
    c_rating INTEGER,
    connector VARCHAR(50),
    weight_grams INTEGER,
    brand VARCHAR(100),
    model VARCHAR(100),
    purchase_date DATE,
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, battery_code)
);
`

const migrationBatteryLogs = `
CREATE TABLE IF NOT EXISTS battery_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    battery_id UUID NOT NULL REFERENCES batteries(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    logged_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cycle_delta INTEGER DEFAULT 0,
    ir_mohm_per_cell JSONB,
    min_cell_v DECIMAL(4,2),
    max_cell_v DECIMAL(4,2),
    storage_ok BOOLEAN,
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
`

const migrationBatteryIndexes = `
CREATE INDEX IF NOT EXISTS idx_batteries_user ON batteries(user_id);
CREATE INDEX IF NOT EXISTS idx_batteries_chemistry ON batteries(chemistry);
CREATE INDEX IF NOT EXISTS idx_batteries_cells ON batteries(cells);
CREATE INDEX IF NOT EXISTS idx_batteries_code ON batteries(battery_code);
CREATE INDEX IF NOT EXISTS idx_battery_logs_battery ON battery_logs(battery_id);
CREATE INDEX IF NOT EXISTS idx_battery_logs_user ON battery_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_battery_logs_logged_at ON battery_logs(logged_at DESC);
`

const migrationUserProfiles = `
-- Add profile fields to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS call_sign VARCHAR(20) UNIQUE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS google_name VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS google_avatar_url VARCHAR(1024);
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_type VARCHAR(20) DEFAULT 'google';
ALTER TABLE users ADD COLUMN IF NOT EXISTS custom_avatar_url VARCHAR(1024);

-- Create index for callsign search (case-insensitive)
CREATE INDEX IF NOT EXISTS idx_users_call_sign ON users(LOWER(call_sign));

-- Create index for pilot search (name search)
CREATE INDEX IF NOT EXISTS idx_users_display_name ON users(LOWER(display_name));
CREATE INDEX IF NOT EXISTS idx_users_google_name ON users(LOWER(google_name));
`

const migrationSocialSettings = `
-- Add social settings columns to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS profile_visibility VARCHAR(20) DEFAULT 'public';
ALTER TABLE users ADD COLUMN IF NOT EXISTS show_aircraft BOOLEAN DEFAULT true;
ALTER TABLE users ADD COLUMN IF NOT EXISTS allow_search BOOLEAN DEFAULT true;

-- Create index for searchable users (respecting allow_search setting)
CREATE INDEX IF NOT EXISTS idx_users_allow_search ON users(allow_search) WHERE allow_search = true;
`

const migrationFollows = `
-- Create follows table for user relationships
CREATE TABLE IF NOT EXISTS follows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    follower_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    followed_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Prevent duplicate follow relationships
    UNIQUE(follower_user_id, followed_user_id),
    
    -- Prevent self-following
    CHECK (follower_user_id != followed_user_id)
);

-- Indexes for efficient lookups
CREATE INDEX IF NOT EXISTS idx_follows_follower ON follows(follower_user_id);
CREATE INDEX IF NOT EXISTS idx_follows_followed ON follows(followed_user_id);
CREATE INDEX IF NOT EXISTS idx_follows_created ON follows(created_at DESC);
`

// Migration for orders table
const migrationOrders = `
-- Create orders table for shipment tracking
CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    carrier VARCHAR(50) NOT NULL DEFAULT 'other',
    tracking_number VARCHAR(100) NOT NULL,
    label VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'unknown',
    status_details TEXT,
    estimated_date TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    last_checked_at TIMESTAMPTZ,
    archived BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for orders
CREATE INDEX IF NOT EXISTS idx_orders_user ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created ON orders(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_orders_archived ON orders(archived);
`

// Migration to change avatar_url columns from VARCHAR to TEXT for base64 images
const migrationCustomAvatarText = `
-- Change avatar_url and custom_avatar_url from VARCHAR(1024) to TEXT to allow base64-encoded images
ALTER TABLE users ALTER COLUMN avatar_url TYPE TEXT;
ALTER TABLE users ALTER COLUMN custom_avatar_url TYPE TEXT;
`

// Migration for flight controller configs
const migrationFCConfigs = `
-- Create fc_configs table for storing Betaflight CLI dumps
CREATE TABLE IF NOT EXISTS fc_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    inventory_item_id UUID NOT NULL REFERENCES inventory_items(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    notes TEXT,
    raw_cli_dump TEXT NOT NULL,
    firmware_name VARCHAR(50) NOT NULL DEFAULT 'betaflight',
    firmware_version VARCHAR(50),
    board_target VARCHAR(50),
    board_name VARCHAR(50),
    mcu_type VARCHAR(50),
    parse_status VARCHAR(20) NOT NULL DEFAULT 'success',
    parse_warnings JSONB DEFAULT '[]',
    parsed_tuning JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for fc_configs
CREATE INDEX IF NOT EXISTS idx_fc_configs_user ON fc_configs(user_id);
CREATE INDEX IF NOT EXISTS idx_fc_configs_inventory_item ON fc_configs(inventory_item_id);
CREATE INDEX IF NOT EXISTS idx_fc_configs_created ON fc_configs(created_at DESC);
`

// Migration for aircraft tuning snapshots
const migrationAircraftTuningSnapshots = `
-- Create aircraft_tuning_snapshots table for storing extracted tuning data
CREATE TABLE IF NOT EXISTS aircraft_tuning_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aircraft_id UUID NOT NULL REFERENCES aircraft(id) ON DELETE CASCADE,
    flight_controller_id UUID REFERENCES inventory_items(id) ON DELETE SET NULL,
    flight_controller_config_id UUID REFERENCES fc_configs(id) ON DELETE SET NULL,
    firmware_name VARCHAR(50) NOT NULL DEFAULT 'betaflight',
    firmware_version VARCHAR(50),
    board_target VARCHAR(50),
    board_name VARCHAR(50),
    tuning_data JSONB NOT NULL,
    parse_status VARCHAR(20) NOT NULL DEFAULT 'success',
    parse_warnings JSONB DEFAULT '[]',
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for aircraft_tuning_snapshots
CREATE INDEX IF NOT EXISTS idx_tuning_snapshots_aircraft ON aircraft_tuning_snapshots(aircraft_id);
CREATE INDEX IF NOT EXISTS idx_tuning_snapshots_fc ON aircraft_tuning_snapshots(flight_controller_id);
CREATE INDEX IF NOT EXISTS idx_tuning_snapshots_config ON aircraft_tuning_snapshots(flight_controller_config_id);
CREATE INDEX IF NOT EXISTS idx_tuning_snapshots_created ON aircraft_tuning_snapshots(created_at DESC);

-- Add unique constraint to ensure one active snapshot per aircraft (most recent)
-- Note: This is soft enforcement - application logic handles "active" concept
`

const migrationTuningSnapshotDiffBackup = `
-- Add diff_backup column to store 'diff all' output for restore purposes
ALTER TABLE aircraft_tuning_snapshots ADD COLUMN IF NOT EXISTS diff_backup TEXT;
`

const migrationDropPasswordHash = `
-- Remove password_hash column as we now use Google-only authentication
ALTER TABLE users DROP COLUMN IF EXISTS password_hash;
`

// Migration for gear catalog (crowd-sourced)
const migrationGearCatalog = `
-- Create gear_catalog table for crowd-sourced gear definitions
CREATE TABLE IF NOT EXISTS gear_catalog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gear_type VARCHAR(50) NOT NULL,
    brand VARCHAR(255) NOT NULL,
    model VARCHAR(512) NOT NULL,
    variant VARCHAR(255),
    specs JSONB DEFAULT '{}',
    source VARCHAR(50) NOT NULL DEFAULT 'user-submitted',
    created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    canonical_key VARCHAR(1024) NOT NULL,
    image_url TEXT,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Unique index on canonical_key for deduplication
CREATE UNIQUE INDEX IF NOT EXISTS idx_gear_catalog_canonical_key ON gear_catalog(canonical_key);

-- Indexes for searching
CREATE INDEX IF NOT EXISTS idx_gear_catalog_gear_type ON gear_catalog(gear_type);
CREATE INDEX IF NOT EXISTS idx_gear_catalog_brand ON gear_catalog(LOWER(brand));
CREATE INDEX IF NOT EXISTS idx_gear_catalog_status ON gear_catalog(status);
CREATE INDEX IF NOT EXISTS idx_gear_catalog_created_by ON gear_catalog(created_by_user_id);

-- Full-text search on brand, model, variant
CREATE INDEX IF NOT EXISTS idx_gear_catalog_search ON gear_catalog USING gin(
    to_tsvector('english', brand || ' ' || model || ' ' || COALESCE(variant, ''))
);

-- GIN index on specs JSONB for filtering
CREATE INDEX IF NOT EXISTS idx_gear_catalog_specs ON gear_catalog USING gin(specs);
`

// Migration to enable pg_trgm extension for fuzzy search
const migrationPgTrgm = `
-- Enable pg_trgm extension for similarity search (may require superuser)
-- This will fail silently on hosted databases where extensions are restricted
DO $$
BEGIN
    CREATE EXTENSION IF NOT EXISTS pg_trgm;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'pg_trgm extension not available, fuzzy search will use fallback';
END $$;

-- Create trigram indexes if extension is available
DO $$
BEGIN
    CREATE INDEX IF NOT EXISTS idx_gear_catalog_brand_trgm ON gear_catalog USING gin(brand gin_trgm_ops);
    CREATE INDEX IF NOT EXISTS idx_gear_catalog_model_trgm ON gear_catalog USING gin(model gin_trgm_ops);
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Could not create trigram indexes, using text search only';
END $$;
`

// Migration to add catalog_id to inventory_items
// DEPENDENCY: Requires migrationGearCatalog to run first (gear_catalog table must exist)
// This is guaranteed by the ordering in the migrations slice in RunMigrations()
const migrationInventoryCatalogLink = `
-- Add catalog_id column to link inventory items to gear catalog
-- NOTE: gear_catalog table must exist before this migration runs
ALTER TABLE inventory_items ADD COLUMN IF NOT EXISTS catalog_id UUID REFERENCES gear_catalog(id) ON DELETE SET NULL;

-- Index for looking up inventory items by catalog item
CREATE INDEX IF NOT EXISTS idx_inventory_catalog ON inventory_items(catalog_id);
`

// Migration to add best_for column to gear_catalog
// Stores array of drone types this gear is best suited for (freestyle, long-range, cinematic, etc.)
const migrationGearCatalogBestFor = `
-- Add best_for column as a text array for drone types
ALTER TABLE gear_catalog ADD COLUMN IF NOT EXISTS best_for TEXT[] DEFAULT '{}';

-- Index for filtering by drone type
CREATE INDEX IF NOT EXISTS idx_gear_catalog_best_for ON gear_catalog USING gin(best_for);
`

// Migration to add msrp column to gear_catalog
const migrationGearCatalogMSRP = `
-- Add msrp column for manufacturer suggested retail price
ALTER TABLE gear_catalog ADD COLUMN IF NOT EXISTS msrp DECIMAL(10,2);
`

// Migration to add image curation fields to gear_catalog
const migrationGearCatalogCuration = `
-- Add image curation status (missing = needs admin review, approved = has curated image)
ALTER TABLE gear_catalog ADD COLUMN IF NOT EXISTS image_status VARCHAR(20) DEFAULT 'missing';
ALTER TABLE gear_catalog ADD COLUMN IF NOT EXISTS image_curated_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE gear_catalog ADD COLUMN IF NOT EXISTS image_curated_at TIMESTAMPTZ;

-- Add description curation status
ALTER TABLE gear_catalog ADD COLUMN IF NOT EXISTS description_status VARCHAR(20) DEFAULT 'missing';
ALTER TABLE gear_catalog ADD COLUMN IF NOT EXISTS description_curated_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE gear_catalog ADD COLUMN IF NOT EXISTS description_curated_at TIMESTAMPTZ;

-- Index for filtering by image status (for admin moderation queue)
CREATE INDEX IF NOT EXISTS idx_gear_catalog_image_status ON gear_catalog(image_status);

-- Set existing items with images to 'approved', items without to 'missing'
-- Only update if image_status is NULL (i.e., never been explicitly set)
UPDATE gear_catalog SET image_status = 'approved' WHERE image_status IS NULL AND ((image_url IS NOT NULL AND image_url != '') OR image_data IS NOT NULL);
UPDATE gear_catalog SET image_status = 'missing' WHERE image_status IS NULL AND (image_url IS NULL OR image_url = '') AND image_data IS NULL;
UPDATE gear_catalog SET description_status = 'approved' WHERE description_status IS NULL AND description IS NOT NULL AND description != '';
UPDATE gear_catalog SET description_status = 'missing' WHERE description_status IS NULL AND (description IS NULL OR description = '');

-- Fix any rows that have image_data but were incorrectly marked as 'missing'
UPDATE gear_catalog SET image_status = 'approved', image_curated_at = COALESCE(image_curated_at, NOW()) 
WHERE image_status = 'missing' AND image_data IS NOT NULL;
`

// Migration to add is_admin flag to users
const migrationUserIsAdmin = `
-- Add is_admin flag (defaults to false)
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_admin BOOLEAN DEFAULT FALSE;

-- Index for finding admin users
CREATE INDEX IF NOT EXISTS idx_users_is_admin ON users(is_admin) WHERE is_admin = TRUE;
`

// Migration to add is_gear_admin flag to users
const migrationUserIsGearAdmin = `
-- Add is_gear_admin flag (defaults to false)
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_gear_admin BOOLEAN DEFAULT FALSE;

-- Add is_content_admin flag (defaults to false) and migrate existing gear moderators.
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_content_admin BOOLEAN DEFAULT FALSE;
UPDATE users
SET is_content_admin = TRUE
WHERE COALESCE(is_content_admin, FALSE) = FALSE
  AND COALESCE(is_gear_admin, FALSE) = TRUE;

-- Keep legacy is_gear_admin in sync as an alias during the transition.
UPDATE users
SET is_gear_admin = COALESCE(is_content_admin, FALSE)
WHERE COALESCE(is_gear_admin, FALSE) != COALESCE(is_content_admin, FALSE);

-- Index for finding legacy gear admin users
CREATE INDEX IF NOT EXISTS idx_users_is_gear_admin ON users(is_gear_admin) WHERE is_gear_admin = TRUE;
-- Index for finding content admin users
CREATE INDEX IF NOT EXISTS idx_users_is_content_admin ON users(is_content_admin) WHERE is_content_admin = TRUE;
`

// Migration to add binary image storage to gear_catalog
const migrationGearCatalogImageData = `
-- Add image_data column for storing actual image binary (max 2MB enforced by app)
ALTER TABLE gear_catalog ADD COLUMN IF NOT EXISTS image_data BYTEA;
ALTER TABLE gear_catalog ADD COLUMN IF NOT EXISTS image_type VARCHAR(50);

-- When image_data is set, clear the old image_url field
-- (we're moving away from URL-based images to uploaded images)
`

// Migration to add unique partial index on (user_id, catalog_id) for inventory items
// This prevents duplicate entries for the same catalog item per user and enables UPSERT
const migrationInventoryCatalogUnique = `
-- Step 1: Update the oldest duplicate (by created_at) to have the sum of all quantities
-- Only process rows where both user_id and catalog_id are NOT NULL to avoid NULL comparison issues
UPDATE inventory_items i
SET quantity = sub.total_quantity
FROM (
    SELECT DISTINCT ON (user_id, catalog_id) 
           id as keep_id, 
           user_id, 
           catalog_id,
           SUM(quantity) OVER (PARTITION BY user_id, catalog_id) as total_quantity
    FROM inventory_items
    WHERE user_id IS NOT NULL
      AND catalog_id IS NOT NULL
      AND (user_id, catalog_id) IN (
          SELECT user_id, catalog_id 
          FROM inventory_items 
          WHERE user_id IS NOT NULL
            AND catalog_id IS NOT NULL 
          GROUP BY user_id, catalog_id 
          HAVING COUNT(*) > 1
      )
    ORDER BY user_id, catalog_id, created_at ASC
) sub
WHERE i.id = sub.keep_id;

-- Step 2: Delete all but the oldest duplicate (by created_at)
-- Must use same predicates as Step 1 to avoid deleting rows whose quantities weren't summed
DELETE FROM inventory_items
WHERE id IN (
    SELECT id FROM (
        SELECT id,
               ROW_NUMBER() OVER (PARTITION BY user_id, catalog_id ORDER BY created_at ASC) as rn
        FROM inventory_items
        WHERE user_id IS NOT NULL
          AND catalog_id IS NOT NULL
    ) ranked
    WHERE rn > 1
);

-- Step 3: Create unique partial index (only for non-null user_id and catalog_id)
CREATE UNIQUE INDEX IF NOT EXISTS idx_inventory_user_catalog_unique 
    ON inventory_items(user_id, catalog_id) WHERE user_id IS NOT NULL AND catalog_id IS NOT NULL;
`

// Migration to drop unused purchase_date column from inventory_items
const migrationDropInventoryPurchaseDate = `
ALTER TABLE inventory_items DROP COLUMN IF EXISTS purchase_date;
`

// Migration to drop unused condition column from inventory_items
const migrationDropInventoryCondition = `
DROP INDEX IF EXISTS idx_inventory_condition;
ALTER TABLE inventory_items DROP COLUMN IF EXISTS condition;
`

// Migration to add centralized image asset storage for moderated user uploads.
const migrationImageAssets = `
CREATE TABLE IF NOT EXISTS image_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entity_type VARCHAR(20) NOT NULL,
    entity_id UUID,
    image_bytes BYTEA NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('APPROVED', 'REJECTED')),
    moderation_labels JSONB NOT NULL DEFAULT '[]',
    moderation_max_confidence DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_image_assets_owner ON image_assets(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_image_assets_entity ON image_assets(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_image_assets_status ON image_assets(status);

ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_image_asset_id UUID;
ALTER TABLE aircraft ADD COLUMN IF NOT EXISTS image_asset_id UUID;
ALTER TABLE gear_catalog ADD COLUMN IF NOT EXISTS image_asset_id UUID;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_users_avatar_image_asset'
    ) THEN
        ALTER TABLE users
        ADD CONSTRAINT fk_users_avatar_image_asset
        FOREIGN KEY (avatar_image_asset_id) REFERENCES image_assets(id) ON DELETE SET NULL;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_aircraft_image_asset'
    ) THEN
        ALTER TABLE aircraft
        ADD CONSTRAINT fk_aircraft_image_asset
        FOREIGN KEY (image_asset_id) REFERENCES image_assets(id) ON DELETE SET NULL;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_gear_catalog_image_asset'
    ) THEN
        ALTER TABLE gear_catalog
        ADD CONSTRAINT fk_gear_catalog_image_asset
        FOREIGN KEY (image_asset_id) REFERENCES image_assets(id) ON DELETE SET NULL;
    END IF;
END $$;
`

// Migration to introduce "scanned" image_status for moderated user-submitted catalog images.
const migrationGearCatalogImageScanned = `
-- Mark existing moderated user-submitted images as scanned (pending admin curation)
UPDATE gear_catalog
SET image_status = 'scanned',
    image_curated_by_user_id = NULL,
    image_curated_at = NULL
WHERE COALESCE(image_status, 'missing') = 'missing'
  AND image_asset_id IS NOT NULL
  AND image_curated_by_user_id IS NULL
  AND image_curated_at IS NULL;
`

// Migration to normalize gear catalog status values to published/pending/removed.
const migrationGearCatalogPublishedStatus = `
-- Map legacy status values to new canonical values.
UPDATE gear_catalog SET status = 'published' WHERE status = 'active';
UPDATE gear_catalog SET status = 'removed' WHERE status IN ('flagged', 'rejected');
UPDATE gear_catalog SET status = 'pending' WHERE status IS NULL OR status NOT IN ('published', 'pending', 'removed');

-- Default new rows to pending moderation unless explicitly published by admin action.
ALTER TABLE gear_catalog ALTER COLUMN status SET DEFAULT 'pending';

-- Enforce canonical status values.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'chk_gear_catalog_status'
    ) THEN
        ALTER TABLE gear_catalog DROP CONSTRAINT chk_gear_catalog_status;
    END IF;
END $$;

ALTER TABLE gear_catalog
ADD CONSTRAINT chk_gear_catalog_status
CHECK (status IN ('published', 'pending', 'removed'));
`

// Migration to keep published records aligned with finalized image curation.
const migrationGearCatalogPublishAutoApproveScannedImage = `
UPDATE gear_catalog
SET image_status = 'approved',
    image_curated_by_user_id = COALESCE(image_curated_by_user_id, created_by_user_id),
    image_curated_at = COALESCE(image_curated_at, NOW()),
    updated_at = NOW()
WHERE status = 'published'
  AND COALESCE(image_status, 'missing') = 'scanned'
  AND (
    image_asset_id IS NOT NULL
    OR image_data IS NOT NULL
    OR (image_url IS NOT NULL AND image_url != '')
  );
`

// Migration to add reusable build definitions (public, draft, and temporary).
const migrationBuilds = `
CREATE TABLE IF NOT EXISTS builds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    image_asset_id UUID,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    token VARCHAR(128),
    expires_at TIMESTAMPTZ,
    title VARCHAR(255) NOT NULL DEFAULT 'Untitled Build',
    description TEXT,
    source_aircraft_id UUID REFERENCES aircraft(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    published_at TIMESTAMPTZ
);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'chk_builds_status'
    ) THEN
        ALTER TABLE builds DROP CONSTRAINT chk_builds_status;
    END IF;
END $$;

ALTER TABLE builds
ADD CONSTRAINT chk_builds_status
CHECK (status IN ('TEMP', 'SHARED', 'DRAFT', 'PENDING_REVIEW', 'PUBLISHED', 'UNPUBLISHED'));

CREATE TABLE IF NOT EXISTS build_parts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    build_id UUID NOT NULL REFERENCES builds(id) ON DELETE CASCADE,
    gear_type VARCHAR(20) NOT NULL,
    catalog_item_id UUID REFERENCES gear_catalog(id) ON DELETE SET NULL,
    position INTEGER NOT NULL DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(build_id, gear_type, position)
);

CREATE INDEX IF NOT EXISTS idx_builds_owner_updated ON builds(owner_user_id, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_builds_status_published ON builds(status, published_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_builds_token_unique ON builds(token) WHERE token IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_builds_expires_at ON builds(expires_at);
CREATE INDEX IF NOT EXISTS idx_build_parts_build ON build_parts(build_id);
CREATE INDEX IF NOT EXISTS idx_build_parts_catalog ON build_parts(catalog_item_id);

ALTER TABLE builds ADD COLUMN IF NOT EXISTS image_asset_id UUID;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_builds_image_asset'
    ) THEN
        ALTER TABLE builds
        ADD CONSTRAINT fk_builds_image_asset
        FOREIGN KEY (image_asset_id) REFERENCES image_assets(id) ON DELETE SET NULL;
    END IF;
END $$;
`

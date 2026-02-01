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
		migrationAircraftELRSSettings,
		migrationAircraftIndexes,
		migrationAircraftImageStorage,
		migrationRadios,
		migrationRadioBackups,
		migrationRadioIndexes,
		migrationBatteries,
		migrationBatteryLogs,
		migrationBatteryIndexes,
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
    password_hash VARCHAR(255),
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
    condition VARCHAR(20) NOT NULL DEFAULT 'new',
    notes TEXT,
    build_id VARCHAR(100),
    purchase_price DECIMAL(10,2),
    purchase_date DATE,
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
CREATE INDEX IF NOT EXISTS idx_inventory_condition ON inventory_items(condition);
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

const migrationAircraftELRSSettings = `
CREATE TABLE IF NOT EXISTS aircraft_elrs_settings (
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
CREATE INDEX IF NOT EXISTS idx_aircraft_elrs_aircraft ON aircraft_elrs_settings(aircraft_id);
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

// Package testutil provides utilities for testing
package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// TestDB wraps a test database connection
type TestDB struct {
	*sql.DB
	t *testing.T
}

// getTestDSN builds the DSN from environment variables or defaults
func getTestDSN() string {
	host := getEnvOrDefault("DB_HOST", "localhost")
	port := getEnvOrDefault("DB_PORT", "5432")
	user := getEnvOrDefault("DB_USER", "test")
	password := getEnvOrDefault("DB_PASSWORD", "test")
	dbname := getEnvOrDefault("DB_NAME", "flyingforge_test")
	sslmode := getEnvOrDefault("DB_SSLMODE", "disable")

	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode,
	)
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

// NewTestDB creates a new test database connection
// It skips the test if the database is not available or schema is not set up
func NewTestDB(t *testing.T) *TestDB {
	t.Helper()

	dsn := getTestDSN()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Skipf("Skipping test: unable to open database: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Skipf("Skipping test: unable to connect to database: %v", err)
	}

	// Check if schema exists (users table is a good indicator)
	var exists bool
	err = db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			AND table_name = 'users'
		)
	`).Scan(&exists)
	if err != nil || !exists {
		db.Close()
		t.Skipf("Skipping test: database schema not set up (users table not found)")
	}

	return &TestDB{DB: db, t: t}
}

// Close closes the test database connection
func (tdb *TestDB) Close() {
	if err := tdb.DB.Close(); err != nil {
		tdb.t.Errorf("Failed to close test database: %v", err)
	}
}

// Cleanup removes all test data from tables
func (tdb *TestDB) Cleanup(ctx context.Context) {
	tdb.t.Helper()

	// Order matters due to foreign key constraints
	tables := []string{
		"battery_logs",
		"batteries",
		"radio_backups",
		"radios",
		"aircraft_receiver_settings",
		"aircraft_components",
		"aircraft",
		"inventory_items",
		"equipment_items",
		"refresh_tokens",
		"user_identities",
		"users",
	}

	for _, table := range tables {
		_, err := tdb.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			// Table might not exist, that's ok
			tdb.t.Logf("Warning: failed to cleanup table %s: %v", table, err)
		}
	}
}

// MustExec executes a query and fails the test on error
func (tdb *TestDB) MustExec(ctx context.Context, query string, args ...interface{}) {
	tdb.t.Helper()
	_, err := tdb.ExecContext(ctx, query, args...)
	if err != nil {
		tdb.t.Fatalf("Failed to execute query: %v\nQuery: %s", err, query)
	}
}

package config

import (
	"flag"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server     ServerConfig
	Cache      CacheConfig
	Database   DatabaseConfig
	Logging    LoggingConfig
	Auth       AuthConfig
	Crypto     CryptoConfig
	Moderation ModerationConfig
}

// ServerConfig holds HTTP/MCP server configuration
type ServerConfig struct {
	HTTPAddr            string
	MCPMode             bool
	RefreshOnceMode     bool
	EnableManualRefresh bool
	RateLimitDur        time.Duration
	FeedRetentionDays   int
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	Backend   string // "memory" or "redis"
	TTL       time.Duration
	RedisAddr string
}

// DatabaseConfig holds PostgreSQL configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level string
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret          string
	JWTIssuer          string
	JWTAudience        string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURI  string
	EnableAdminTools   bool
}

// CryptoConfig holds encryption configuration for sensitive data at rest
type CryptoConfig struct {
	// EncryptionKey must be exactly 32 bytes for AES-256 encryption.
	// Used to encrypt sensitive user data like receiver bind phrases.
	// CRITICAL: This key must be kept secret and backed up securely.
	// Losing this key means losing access to all encrypted data.
	EncryptionKey []byte
}

// ModerationConfig holds image moderation settings.
type ModerationConfig struct {
	Enabled          bool
	AWSRegion        string
	RejectConfidence float64
	Timeout          time.Duration
	PendingUploadTTL time.Duration
}

// Load parses flags and environment variables to build configuration
func Load() *Config {
	cfg := &Config{}

	// Define flags with defaults
	httpAddr := flag.String("http", ":8080", "HTTP server address")
	mcpMode := flag.Bool("mcp", false, "Run in MCP stdio mode")
	refreshOnceMode := flag.Bool("refresh-once", false, "Run a single feed refresh and exit")
	cacheTTL := flag.Duration("cache-ttl", 5*time.Minute, "Cache TTL for feed items")
	cacheBackend := flag.String("cache-backend", "memory", "Cache backend: memory or redis")
	redisAddr := flag.String("redis-addr", "localhost:6379", "Redis server address")
	rateLimitDur := flag.Duration("rate-limit", time.Second, "Minimum delay between requests to same host")
	feedRetentionDays := flag.Int("feed-retention-days", 90, "Number of days to retain feed items in the database (0 to disable)")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	dbHost := flag.String("db-host", "localhost", "PostgreSQL host")
	dbPort := flag.Int("db-port", 5432, "PostgreSQL port")
	dbUser := flag.String("db-user", "postgres", "PostgreSQL user")
	dbPassword := flag.String("db-password", "postgres", "PostgreSQL password")
	dbName := flag.String("db-name", "drone_inventory", "PostgreSQL database name")
	dbSSLMode := flag.String("db-sslmode", "disable", "PostgreSQL SSL mode")

	flag.Parse()

	// Apply environment variable overrides
	enableManualRefresh := false
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("ENABLE_MANUAL_REFRESH"))); v == "true" || v == "1" {
		enableManualRefresh = true
	}

	applyEnvOverrides(httpAddr, mcpMode, refreshOnceMode, cacheTTL, cacheBackend, redisAddr, rateLimitDur, feedRetentionDays, logLevel, dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	// Build config struct
	cfg.Server = ServerConfig{
		HTTPAddr:            *httpAddr,
		MCPMode:             *mcpMode,
		RefreshOnceMode:     *refreshOnceMode,
		EnableManualRefresh: enableManualRefresh,
		RateLimitDur:        *rateLimitDur,
		FeedRetentionDays:   *feedRetentionDays,
	}

	cfg.Cache = CacheConfig{
		Backend:   *cacheBackend,
		TTL:       *cacheTTL,
		RedisAddr: *redisAddr,
	}

	cfg.Database = DatabaseConfig{
		Host:     *dbHost,
		Port:     *dbPort,
		User:     *dbUser,
		Password: *dbPassword,
		Database: *dbName,
		SSLMode:  *dbSSLMode,
	}

	cfg.Logging = LoggingConfig{
		Level: *logLevel,
	}

	// Load auth config from environment
	cfg.Auth = loadAuthConfig()

	// Load crypto config from environment
	cfg.Crypto = loadCryptoConfig()

	// Load moderation config from environment
	cfg.Moderation = loadModerationConfig()

	return cfg
}

func loadAuthConfig() AuthConfig {
	accessTTL := 15 * time.Minute
	if v := os.Getenv("AUTH_ACCESS_TOKEN_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			accessTTL = d
		}
	}

	refreshTTL := 7 * 24 * time.Hour // 7 days
	if v := os.Getenv("AUTH_REFRESH_TOKEN_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			refreshTTL = d
		}
	}

	return AuthConfig{
		JWTSecret:          getEnvOrDefault("AUTH_JWT_SECRET", "change-me-in-production"),
		JWTIssuer:          getEnvOrDefault("AUTH_JWT_ISSUER", "flyingforge"),
		JWTAudience:        getEnvOrDefault("AUTH_JWT_AUDIENCE", "flyingforge-users"),
		AccessTokenTTL:     accessTTL,
		RefreshTokenTTL:    refreshTTL,
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURI:  getEnvOrDefault("GOOGLE_REDIRECT_URI", "http://localhost:8080/api/auth/google/callback"),
		EnableAdminTools:   os.Getenv("ENABLE_ADMIN_TOOLS") == "true",
	}
}

// loadCryptoConfig loads encryption configuration from environment variables.
// BIND_PHRASE_ENCRYPTION_KEY must be exactly 32 bytes (characters) for AES-256.
func loadCryptoConfig() CryptoConfig {
	key := os.Getenv("BIND_PHRASE_ENCRYPTION_KEY")
	if key == "" {
		// Use a default key for development only - MUST be overridden in production
		key = "CHANGE-THIS-32-BYTE-KEY-IN-PROD"
	}

	return CryptoConfig{
		EncryptionKey: []byte(key),
	}
}

func loadModerationConfig() ModerationConfig {
	rejectConfidence := 70.0
	if v := os.Getenv("MODERATION_REJECT_CONFIDENCE"); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed > 0 {
			rejectConfidence = parsed
		}
	}

	timeout := 5 * time.Second
	if v := os.Getenv("MODERATION_TIMEOUT"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil && parsed > 0 {
			timeout = parsed
		}
	}

	pendingTTL := 10 * time.Minute
	if v := os.Getenv("MODERATION_PENDING_TTL"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil && parsed > 0 {
			pendingTTL = parsed
		}
	}

	enabled := true
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("IMAGE_MODERATION_ENABLED"))); v == "false" || v == "0" {
		enabled = false
	}

	return ModerationConfig{
		Enabled:          enabled,
		AWSRegion:        os.Getenv("AWS_REGION"),
		RejectConfidence: rejectConfidence,
		Timeout:          timeout,
		PendingUploadTTL: pendingTTL,
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func applyEnvOverrides(
	httpAddr *string,
	mcpMode *bool,
	refreshOnceMode *bool,
	cacheTTL *time.Duration,
	cacheBackend *string,
	redisAddr *string,
	rateLimitDur *time.Duration,
	feedRetentionDays *int,
	logLevel *string,
	dbHost *string,
	dbPort *int,
	dbUser *string,
	dbPassword *string,
	dbName *string,
	dbSSLMode *string,
) {
	if v := os.Getenv("HTTP_ADDR"); v != "" {
		*httpAddr = v
	}
	if v := os.Getenv("MCP_MODE"); v == "true" || v == "1" {
		*mcpMode = true
	}
	if v := os.Getenv("REFRESH_ONCE_MODE"); v == "true" || v == "1" {
		*refreshOnceMode = true
	}
	if v := os.Getenv("CACHE_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			*cacheTTL = d
		}
	}
	if v := os.Getenv("CACHE_BACKEND"); v != "" {
		*cacheBackend = v
	}
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		*redisAddr = v
	}
	if v := os.Getenv("RATE_LIMIT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			*rateLimitDur = d
		}
	}
	if v := os.Getenv("FEED_RETENTION_DAYS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			*feedRetentionDays = parsed
		}
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		*logLevel = v
	}
	if v := os.Getenv("DB_HOST"); v != "" {
		*dbHost = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			*dbPort = p
		}
	}
	if v := os.Getenv("DB_USER"); v != "" {
		*dbUser = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		*dbPassword = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		*dbName = v
	}
	if v := os.Getenv("DB_SSLMODE"); v != "" {
		*dbSSLMode = v
	}
}

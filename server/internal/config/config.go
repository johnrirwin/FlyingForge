package config

import (
	"flag"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Cache    CacheConfig
	Database DatabaseConfig
	Logging  LoggingConfig
	Auth     AuthConfig
}

// ServerConfig holds HTTP/MCP server configuration
type ServerConfig struct {
	HTTPAddr     string
	MCPMode      bool
	RateLimitDur time.Duration
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

// Load parses flags and environment variables to build configuration
func Load() *Config {
	cfg := &Config{}

	// Define flags with defaults
	httpAddr := flag.String("http", ":8080", "HTTP server address")
	mcpMode := flag.Bool("mcp", false, "Run in MCP stdio mode")
	cacheTTL := flag.Duration("cache-ttl", 5*time.Minute, "Cache TTL for feed items")
	cacheBackend := flag.String("cache-backend", "memory", "Cache backend: memory or redis")
	redisAddr := flag.String("redis-addr", "localhost:6379", "Redis server address")
	rateLimitDur := flag.Duration("rate-limit", time.Second, "Minimum delay between requests to same host")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	dbHost := flag.String("db-host", "localhost", "PostgreSQL host")
	dbPort := flag.Int("db-port", 5432, "PostgreSQL port")
	dbUser := flag.String("db-user", "postgres", "PostgreSQL user")
	dbPassword := flag.String("db-password", "postgres", "PostgreSQL password")
	dbName := flag.String("db-name", "drone_inventory", "PostgreSQL database name")
	dbSSLMode := flag.String("db-sslmode", "disable", "PostgreSQL SSL mode")

	flag.Parse()

	// Apply environment variable overrides
	applyEnvOverrides(httpAddr, mcpMode, cacheTTL, cacheBackend, redisAddr, rateLimitDur, logLevel, dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	// Build config struct
	cfg.Server = ServerConfig{
		HTTPAddr:     *httpAddr,
		MCPMode:      *mcpMode,
		RateLimitDur: *rateLimitDur,
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

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func applyEnvOverrides(
	httpAddr *string,
	mcpMode *bool,
	cacheTTL *time.Duration,
	cacheBackend *string,
	redisAddr *string,
	rateLimitDur *time.Duration,
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

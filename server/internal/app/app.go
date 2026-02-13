package app

import (
	"context"
	"fmt"
	"time"

	"github.com/johnrirwin/flyingforge/internal/aggregator"
	"github.com/johnrirwin/flyingforge/internal/aircraft"
	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/battery"
	"github.com/johnrirwin/flyingforge/internal/builds"
	"github.com/johnrirwin/flyingforge/internal/cache"
	"github.com/johnrirwin/flyingforge/internal/config"
	"github.com/johnrirwin/flyingforge/internal/crypto"
	"github.com/johnrirwin/flyingforge/internal/database"
	"github.com/johnrirwin/flyingforge/internal/equipment"
	"github.com/johnrirwin/flyingforge/internal/httpapi"
	"github.com/johnrirwin/flyingforge/internal/images"
	"github.com/johnrirwin/flyingforge/internal/inventory"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/mcp"
	"github.com/johnrirwin/flyingforge/internal/models"
	"github.com/johnrirwin/flyingforge/internal/moderation"
	"github.com/johnrirwin/flyingforge/internal/radio"
	"github.com/johnrirwin/flyingforge/internal/ratelimit"
	"github.com/johnrirwin/flyingforge/internal/sellers"
	"github.com/johnrirwin/flyingforge/internal/sources"
	"github.com/johnrirwin/flyingforge/internal/tagging"
)

// App holds all application dependencies
type App struct {
	Config           *config.Config
	Logger           *logging.Logger
	Cache            cache.Cache
	Aggregator       *aggregator.Aggregator
	EquipmentSvc     *equipment.Service
	InventorySvc     inventory.InventoryManager
	AircraftSvc      *aircraft.Service
	BuildSvc         *builds.Service
	RadioSvc         *radio.Service
	BatterySvc       *battery.Service
	AuthService      *auth.Service
	AuthMiddleware   *auth.Middleware
	HTTPServer       *httpapi.Server
	MCPServer        *mcp.Server
	db               *database.DB
	userStore        *database.UserStore
	aircraftStore    *database.AircraftStore
	fcConfigStore    *database.FCConfigStore
	inventoryStore   *database.InventoryStore
	buildStore       *database.BuildStore
	gearCatalogStore *database.GearCatalogStore
	imageAssetStore  *database.ImageAssetStore
	imageSvc         *images.Service
	refreshLimiter   ratelimit.RateLimiter
}

// New creates and initializes a new App instance
func New(cfg *config.Config) (*App, error) {
	app := &App{Config: cfg}

	// Initialize logger
	app.Logger = app.initLogger()

	// Initialize cache
	app.Cache = app.initCache()

	// Initialize rate limiter and tagger
	limiter := ratelimit.New(cfg.Server.RateLimitDur)
	tagger := tagging.New()

	// Initialize feed fetchers
	fetchers := app.initFetchers(limiter)

	// Initialize aggregator
	app.Aggregator = aggregator.New(fetchers, app.Cache, tagger, app.Logger)
	app.Aggregator.SetRetentionDays(cfg.Server.FeedRetentionDays)

	// Initialize seller registry
	sellerRegistry := app.initSellers(limiter)

	// Initialize equipment service
	app.EquipmentSvc = equipment.NewService(sellerRegistry, app.Cache, app.Logger)

	// Initialize database, inventory, and auth services
	app.initDatabaseServices()

	// Initialize servers
	app.initServers()

	return app, nil
}

// Run starts the application in the appropriate mode
func (a *App) Run(ctx context.Context) error {
	if a.Config.Server.RefreshOnceMode {
		return a.runRefreshOnceMode(ctx)
	}
	if a.Config.Server.MCPMode {
		return a.runMCPMode(ctx)
	}
	return a.runHTTPMode(ctx)
}

// Shutdown gracefully shuts down the application
func (a *App) Shutdown(ctx context.Context) error {
	if a.HTTPServer != nil {
		if err := a.HTTPServer.Shutdown(ctx); err != nil {
			a.Logger.Error("HTTP server shutdown error", logging.WithField("error", err.Error()))
		}
	}

	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.Logger.Error("Database close error", logging.WithField("error", err.Error()))
		}
	}

	return nil
}

func (a *App) initLogger() *logging.Logger {
	level := logging.LevelInfo
	switch a.Config.Logging.Level {
	case "debug":
		level = logging.LevelDebug
	case "warn":
		level = logging.LevelWarn
	case "error":
		level = logging.LevelError
	}
	return logging.New(level)
}

func (a *App) initCache() cache.Cache {
	switch a.Config.Cache.Backend {
	case "redis":
		a.Logger.Info("Using Redis cache backend", logging.WithField("addr", a.Config.Cache.RedisAddr))
		redisCache, err := cache.NewRedis(cache.RedisConfig{
			Addr:   a.Config.Cache.RedisAddr,
			Prefix: "mcp-news:",
		}, a.Config.Cache.TTL)
		if err != nil {
			a.Logger.Error("Failed to connect to Redis, falling back to memory cache", logging.WithField("error", err.Error()))
			a.refreshLimiter = ratelimit.New(2 * time.Minute)
			return cache.NewMemory(a.Config.Cache.TTL)
		}
		// Use Redis for distributed rate limiting when available
		a.refreshLimiter = ratelimit.NewRedis(redisCache.Client(), "ratelimit:refresh:", 2*time.Minute)
		a.Logger.Info("Using Redis for distributed rate limiting")
		return redisCache
	default:
		a.Logger.Info("Using in-memory cache backend")
		a.refreshLimiter = ratelimit.New(2 * time.Minute)
		return cache.NewMemory(a.Config.Cache.TTL)
	}
}

func (a *App) initFetchers(limiter *ratelimit.Limiter) []sources.Fetcher {
	fetcherConfig := sources.DefaultConfig()

	// Try to load feeds from config file
	configPath := sources.FindFeedsConfig()
	if configPath != "" {
		feedsConfig, err := sources.LoadFeedsConfig(configPath)
		if err != nil {
			a.Logger.Warn("Failed to load feeds config, using defaults", logging.WithFields(map[string]interface{}{
				"path":  configPath,
				"error": err.Error(),
			}))
		} else {
			a.Logger.Info("Loaded feeds configuration", logging.WithFields(map[string]interface{}{
				"path":    configPath,
				"sources": len(feedsConfig.Sources),
			}))
			return sources.CreateFetchersFromConfig(feedsConfig, limiter, fetcherConfig)
		}
	} else {
		a.Logger.Info("No feeds.json found, using default sources")
	}

	// Fallback to default config
	defaultConfig := sources.GetDefaultFeedsConfig()
	return sources.CreateFetchersFromConfig(defaultConfig, limiter, fetcherConfig)
}

func (a *App) initSellers(limiter *ratelimit.Limiter) *sellers.Registry {
	registry := sellers.NewRegistry()
	registry.Register(sellers.NewRaceDayQuads(limiter, a.Cache))
	registry.Register(sellers.NewGetFPV(limiter, a.Cache))
	a.Logger.Info("Registered seller adapters", logging.WithField("count", len(registry.List())))
	return registry
}

func (a *App) initDatabaseServices() {
	dbConfig := database.Config{
		Host:     a.Config.Database.Host,
		Port:     a.Config.Database.Port,
		User:     a.Config.Database.User,
		Password: a.Config.Database.Password,
		Database: a.Config.Database.Database,
		SSLMode:  a.Config.Database.SSLMode,
	}

	db, err := database.New(dbConfig)
	if err != nil {
		a.Logger.Warn("Failed to connect to PostgreSQL, using in-memory inventory (auth disabled)", logging.WithField("error", err.Error()))
		a.InventorySvc = inventory.NewInMemoryService(a.Logger)
		// Auth service requires database, so we create a no-op middleware
		a.AuthMiddleware = auth.NewMiddleware(nil)
		return
	}

	a.Logger.Info("Connected to PostgreSQL")
	if err := db.Migrate(context.Background()); err != nil {
		a.Logger.Warn("Failed to run migrations, using in-memory inventory (auth disabled)", logging.WithField("error", err.Error()))
		a.InventorySvc = inventory.NewInMemoryService(a.Logger)
		a.AuthMiddleware = auth.NewMiddleware(nil)
		return
	}

	a.db = db
	// Persist aggregated feed items in Postgres when available so we can keep history
	// across refresh runs.
	if a.Aggregator != nil {
		a.Aggregator.SetStore(database.NewFeedItemStore(db))
	}

	// Initialize encryptor for sensitive data
	encryptor, err := crypto.NewEncryptor(a.Config.Crypto.EncryptionKey)
	if err != nil {
		a.Logger.Warn("Failed to initialize encryptor - bind phrases will NOT be encrypted",
			logging.WithField("error", err.Error()),
			logging.WithField("hint", "Set BIND_PHRASE_ENCRYPTION_KEY env var to exactly 32 characters"))
		encryptor = nil
	}

	// Initialize inventory
	a.inventoryStore = database.NewInventoryStore(db)
	a.InventorySvc = inventory.NewService(a.inventoryStore, a.Logger)

	// Initialize centralized image storage + moderation pipeline
	a.imageAssetStore = database.NewImageAssetStore(db)
	moderatorSvc, err := a.newModerationService()
	if err != nil {
		a.Logger.Warn("Image moderation setup failed, uploads will return PENDING_REVIEW",
			logging.WithField("error", err.Error()))
		moderatorSvc = &moderation.MockModerator{Err: fmt.Errorf("moderation unavailable")}
	}
	var pendingStore images.PendingStore = images.NewInMemoryPendingStore(a.Config.Moderation.PendingUploadTTL)
	if redisCache, ok := a.Cache.(*cache.RedisCache); ok {
		a.Logger.Info("Using Redis pending upload store")
		pendingStore = images.NewRedisPendingStore(redisCache.Client(), a.Config.Moderation.PendingUploadTTL)
	} else {
		a.Logger.Info("Using in-memory pending upload store")
	}
	a.imageSvc = images.NewService(moderatorSvc, a.imageAssetStore, pendingStore, a.Config.Moderation.Timeout)

	// Initialize gear catalog store (before aircraft, since aircraft contributes to catalog)
	a.gearCatalogStore = database.NewGearCatalogStore(db)

	// Initialize aircraft (with encryption support and gear catalog contribution)
	a.aircraftStore = database.NewAircraftStore(db, encryptor)
	a.AircraftSvc = aircraft.NewService(a.aircraftStore, a.InventorySvc, a.gearCatalogStore, a.imageSvc, a.Logger)

	// Initialize builds service (public builds + draft/temp builder)
	a.buildStore = database.NewBuildStore(db)
	a.BuildSvc = builds.NewService(a.buildStore, a.aircraftStore, a.gearCatalogStore, a.imageSvc, a.Logger)

	// Initialize radio
	radioStore := database.NewRadioStore(db)
	a.RadioSvc = radio.NewService(radioStore, "", a.Logger) // Empty string uses default storage dir

	// Initialize battery
	batteryStore := database.NewBatteryStore(db)
	a.BatterySvc = battery.NewService(batteryStore, a.Logger)

	// Initialize auth
	a.userStore = database.NewUserStore(db)
	a.AuthService = auth.NewService(a.userStore, a.Config.Auth, a.Logger)
	a.AuthMiddleware = auth.NewMiddleware(a.AuthService)

	// Initialize FC config store
	a.fcConfigStore = database.NewFCConfigStore(db)

	a.Logger.Info("Authentication service initialized")
}

func (a *App) initServers() {
	// Initialize HTTP server with auth, aircraft, radio, battery, fc-config, gear-catalog, and profile/pilot support
	a.HTTPServer = httpapi.New(a.Aggregator, a.EquipmentSvc, a.InventorySvc, a.AircraftSvc, a.BuildSvc, a.RadioSvc, a.BatterySvc, a.AuthService, a.AuthMiddleware, a.userStore, a.aircraftStore, a.fcConfigStore, a.inventoryStore, a.gearCatalogStore, a.imageSvc, a.refreshLimiter, a.Config.Server.EnableManualRefresh, a.Logger)

	// Initialize MCP server
	mcpHandler := mcp.NewHandler(a.Aggregator, a.EquipmentSvc, a.InventorySvc, a.Logger)
	a.MCPServer = mcp.NewServer(mcpHandler, a.Logger)
}

func (a *App) newModerationService() (images.Moderator, error) {
	if !a.Config.Moderation.Enabled {
		a.Logger.Warn("Image moderation explicitly disabled; uploads will auto-approve")
		return &moderation.MockModerator{
			Decision: &models.ModerationDecision{
				Status: models.ImageModerationApproved,
				Reason: "Approved",
			},
		}, nil
	}

	detector, err := moderation.NewAWSDetector(context.Background(), a.Config.Moderation.AWSRegion)
	if err != nil {
		return nil, err
	}
	return moderation.NewService(detector, a.Config.Moderation.RejectConfidence), nil
}

func (a *App) runMCPMode(ctx context.Context) error {
	a.Logger.Info("Starting MCP server in stdio mode")

	a.Logger.Info("Pre-fetching feeds...")
	if err := a.Aggregator.Refresh(ctx); err != nil {
		a.Logger.Warn("Initial fetch had errors", logging.WithField("error", err.Error()))
	}

	return a.MCPServer.Run(ctx)
}

func (a *App) runHTTPMode(ctx context.Context) error {
	a.Logger.Info("Starting HTTP server", logging.WithField("addr", a.Config.Server.HTTPAddr))

	if a.BuildSvc != nil {
		go a.runTempBuildCleanup(ctx)
	}

	return a.HTTPServer.Start(a.Config.Server.HTTPAddr)
}

func (a *App) runRefreshOnceMode(ctx context.Context) error {
	a.Logger.Info("Running one-shot feed refresh")
	if err := a.Aggregator.Refresh(ctx); err != nil {
		a.Logger.Warn("One-shot feed refresh had errors", logging.WithField("error", err.Error()))
		return err
	}
	a.Logger.Info("One-shot feed refresh complete")
	return nil
}

func (a *App) runTempBuildCleanup(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	cleanup := func() {
		deleted, err := a.BuildSvc.CleanupExpiredTemp(ctx)
		if err != nil {
			a.Logger.Warn("Temp build cleanup failed", logging.WithField("error", err.Error()))
			return
		}
		if deleted > 0 {
			a.Logger.Info("Removed expired temp builds", logging.WithField("count", deleted))
		}
	}

	// Run once at startup, then periodically.
	cleanup()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanup()
		}
	}
}

package app

import (
	"context"

	"github.com/johnrirwin/flyingforge/internal/aggregator"
	"github.com/johnrirwin/flyingforge/internal/aircraft"
	"github.com/johnrirwin/flyingforge/internal/auth"
	"github.com/johnrirwin/flyingforge/internal/battery"
	"github.com/johnrirwin/flyingforge/internal/cache"
	"github.com/johnrirwin/flyingforge/internal/config"
	"github.com/johnrirwin/flyingforge/internal/database"
	"github.com/johnrirwin/flyingforge/internal/equipment"
	"github.com/johnrirwin/flyingforge/internal/httpapi"
	"github.com/johnrirwin/flyingforge/internal/inventory"
	"github.com/johnrirwin/flyingforge/internal/logging"
	"github.com/johnrirwin/flyingforge/internal/mcp"
	"github.com/johnrirwin/flyingforge/internal/radio"
	"github.com/johnrirwin/flyingforge/internal/ratelimit"
	"github.com/johnrirwin/flyingforge/internal/sellers"
	"github.com/johnrirwin/flyingforge/internal/sources"
	"github.com/johnrirwin/flyingforge/internal/tagging"
)

// App holds all application dependencies
type App struct {
	Config         *config.Config
	Logger         *logging.Logger
	Cache          cache.Cache
	Aggregator     *aggregator.Aggregator
	EquipmentSvc   *equipment.Service
	InventorySvc   inventory.InventoryManager
	AircraftSvc    *aircraft.Service
	RadioSvc       *radio.Service
	BatterySvc     *battery.Service
	AuthService    *auth.Service
	AuthMiddleware *auth.Middleware
	HTTPServer     *httpapi.Server
	MCPServer      *mcp.Server
	db             *database.DB
	userStore      *database.UserStore
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
			return cache.NewMemory(a.Config.Cache.TTL)
		}
		return redisCache
	default:
		a.Logger.Info("Using in-memory cache backend")
		return cache.NewMemory(a.Config.Cache.TTL)
	}
}

func (a *App) initFetchers(limiter *ratelimit.Limiter) []sources.Fetcher {
	sourcesConfig := sources.DefaultConfig()
	var fetchers []sources.Fetcher
	fetchers = append(fetchers, sources.CreateDroneRSSFetchers(limiter, sourcesConfig)...)
	fetchers = append(fetchers, sources.CreateDroneRedditFetchers(limiter, sourcesConfig)...)
	return fetchers
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

	// Initialize inventory
	inventoryStore := database.NewInventoryStore(db)
	a.InventorySvc = inventory.NewService(inventoryStore, a.Logger)

	// Initialize aircraft
	aircraftStore := database.NewAircraftStore(db)
	a.AircraftSvc = aircraft.NewService(aircraftStore, a.InventorySvc, a.Logger)

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

	a.Logger.Info("Authentication service initialized")
}

func (a *App) initServers() {
	// Initialize HTTP server with auth, aircraft, radio, and battery
	a.HTTPServer = httpapi.New(a.Aggregator, a.EquipmentSvc, a.InventorySvc, a.AircraftSvc, a.RadioSvc, a.BatterySvc, a.AuthService, a.AuthMiddleware, a.Logger)

	// Initialize MCP server
	mcpHandler := mcp.NewHandler(a.Aggregator, a.EquipmentSvc, a.InventorySvc, a.Logger)
	a.MCPServer = mcp.NewServer(mcpHandler, a.Logger)
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

	// Pre-fetch feeds in background
	go func() {
		a.Logger.Info("Pre-fetching feeds in background...")
		if err := a.Aggregator.Refresh(ctx); err != nil {
			a.Logger.Warn("Initial fetch had errors", logging.WithField("error", err.Error()))
		}
		a.Logger.Info("Initial fetch complete")
	}()

	return a.HTTPServer.Start(a.Config.Server.HTTPAddr)
}

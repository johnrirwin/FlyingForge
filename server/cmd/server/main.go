package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/johnrirwin/mcp-news-feed/internal/aggregator"
	"github.com/johnrirwin/mcp-news-feed/internal/cache"
	"github.com/johnrirwin/mcp-news-feed/internal/httpapi"
	"github.com/johnrirwin/mcp-news-feed/internal/logging"
	"github.com/johnrirwin/mcp-news-feed/internal/mcp"
	"github.com/johnrirwin/mcp-news-feed/internal/ratelimit"
	"github.com/johnrirwin/mcp-news-feed/internal/sources"
	"github.com/johnrirwin/mcp-news-feed/internal/tagging"
)

func main() {
	httpAddr := flag.String("http", ":8080", "HTTP server address")
	mcpMode := flag.Bool("mcp", false, "Run in MCP stdio mode")
	cacheTTL := flag.Duration("cache-ttl", 5*time.Minute, "Cache TTL for feed items")
	rateLimitDur := flag.Duration("rate-limit", time.Second, "Minimum delay between requests to same host")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

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
	if v := os.Getenv("RATE_LIMIT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			*rateLimitDur = d
		}
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		*logLevel = v
	}

	level := logging.LevelInfo
	switch *logLevel {
	case "debug":
		level = logging.LevelDebug
	case "warn":
		level = logging.LevelWarn
	case "error":
		level = logging.LevelError
	}
	logger := logging.New(level)

	feedCache := cache.New(*cacheTTL)
	limiter := ratelimit.New(*rateLimitDur)
	tagger := tagging.New()

	config := sources.DefaultConfig()
	var fetchers []sources.Fetcher
	fetchers = append(fetchers, sources.CreateDroneRSSFetchers(limiter, config)...)
	fetchers = append(fetchers, sources.CreateDroneRedditFetchers(limiter, config)...)

	agg := aggregator.New(fetchers, feedCache, tagger, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("Shutting down...")
		cancel()
	}()

	if *mcpMode {
		logger.Info("Starting MCP server in stdio mode")
		mcpHandler := mcp.NewHandler(agg, logger)
		mcpServer := mcp.NewServer(mcpHandler, logger)

		logger.Info("Pre-fetching feeds...")
		if err := agg.Refresh(ctx); err != nil {
			logger.Warn("Initial fetch had errors", logging.WithField("error", err.Error()))
		}

		if err := mcpServer.Run(ctx); err != nil && err != context.Canceled {
			logger.Error("MCP server error", logging.WithField("error", err.Error()))
			os.Exit(1)
		}
	} else {
		logger.Info("Starting HTTP server", logging.WithField("addr", *httpAddr))

		httpServer := httpapi.New(agg, logger)

		go func() {
			logger.Info("Pre-fetching feeds in background...")
			if err := agg.Refresh(ctx); err != nil {
				logger.Warn("Initial fetch had errors", logging.WithField("error", err.Error()))
			}
			logger.Info("Initial fetch complete")
		}()

		if err := httpServer.Start(*httpAddr); err != nil {
			logger.Error("HTTP server error", logging.WithField("error", err.Error()))
			os.Exit(1)
		}
	}
}

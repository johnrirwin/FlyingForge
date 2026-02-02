# FlyingForge Server Architecture

This document provides a comprehensive overview of the FlyingForge MCP server architecture, including all components, endpoints, data storage, and deployment considerations.

## Table of Contents

1. [Overview](#overview)
2. [Architecture Diagram](#architecture-diagram)
3. [Infrastructure](#infrastructure)
4. [Core Components](#core-components)
5. [Operating Modes](#operating-modes)
6. [HTTP API Endpoints](#http-api-endpoints)
7. [MCP Protocol](#mcp-protocol)
8. [Data Models](#data-models)
9. [Source Fetchers](#source-fetchers)
10. [Configuration](#configuration)
11. [Production Deployment](#production-deployment)

---

## Overview

FlyingForge is a Go application that provides drone equipment management, news aggregation, and inventory tracking. It operates in two modes:

1. **HTTP Mode** (default): Serves a REST API for the React frontend
2. **MCP Mode**: Provides tools to AI assistants via the Model Context Protocol

The server manages user authentication, equipment catalogs from multiple sellers, personal inventory, aircraft configurations, radio setups, battery tracking, and aggregated news from RSS feeds and Reddit.

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                                  FlyingForge Server                                      │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                          │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────────────────────────────────┐    │
│  │  HTTP API   │     │  MCP Server │     │              Services                   │    │
│  │  (port 8080)│     │   (stdio)   │     │                                         │    │
│  │             │     │             │     │  ┌─────────┐  ┌───────────┐  ┌───────┐  │    │
│  │ /api/auth/* │────▶│ tools/call  │────▶│  │  Auth   │  │ Equipment │  │ Radio │  │    │
│  │ /api/equip/*│     │ tools/list  │     │  └────┬────┘  └─────┬─────┘  └───┬───┘  │    │
│  │ /api/inv/*  │     │ initialize  │     │       │             │            │       │    │
│  │ /api/items  │     │             │     │  ┌────┴────┐  ┌─────┴─────┐  ┌───┴────┐ │    │
│  │ /api/aircraft│    └─────────────┘     │  │Aircraft │  │ Inventory │  │Battery │ │    │
│  │ /api/radio  │                         │  └─────────┘  └───────────┘  └────────┘ │    │
│  │ /api/battery│                         │                                         │    │
│  │ /health     │                         │       ┌─────────────────────┐           │    │
│  └──────┬──────┘                         │       │     Aggregator      │           │    │
│         │                                │       │  (News/RSS/Reddit)  │           │    │
│         │                                │       └──────────┬──────────┘           │    │
│         │                                └──────────────────┼──────────────────────┘    │
│         │                                                   │                           │
│  ┌──────┴───────────────────────────────────────────────────┴──────────────────────┐   │
│  │                              Data Access Layer                                   │   │
│  │                                                                                  │   │
│  │  ┌───────────────────┐       ┌─────────────────────┐       ┌────────────────┐   │   │
│  │  │  Database Stores  │       │   Cache Interface   │       │  Rate Limiter  │   │   │
│  │  │                   │       │                     │       │                │   │   │
│  │  │ • UserStore       │       │  • Memory Cache     │       │  Per-host      │   │   │
│  │  │ • EquipmentStore  │       │  • Redis Cache      │       │  throttling    │   │   │
│  │  │ • InventoryStore  │       │    (production)     │       │  (1s default)  │   │   │
│  │  │ • AircraftStore   │       │                     │       │                │   │   │
│  │  │ • RadioStore      │       │                     │       │                │   │   │
│  │  │ • BatteryStore    │       │                     │       │                │   │   │
│  │  └─────────┬─────────┘       └──────────┬──────────┘       └────────────────┘   │   │
│  │            │                            │                                        │   │
│  └────────────┼────────────────────────────┼────────────────────────────────────────┘   │
│               │                            │                                            │
└───────────────┼────────────────────────────┼────────────────────────────────────────────┘
                │                            │
                ▼                            ▼
┌───────────────────────────┐    ┌───────────────────────────┐
│        PostgreSQL         │    │          Redis            │
│         (port 5432)       │    │        (port 6379)        │
│                           │    │                           │
│  • users                  │    │  • Session cache          │
│  • user_identities        │    │  • Feed item cache        │
│  • refresh_tokens         │    │  • Rate limit tracking    │
│  • equipment_items        │    │  • API response cache     │
│  • inventory_items        │    │                           │
│  • sellers                │    │  TTL: 5 minutes (default) │
│  • aircraft               │    │  Prefix: mcp-news:        │
│  • aircraft_components    │    │                           │
│  • radios                 │    │  Persistence: AOF         │
│  • radio_backups          │    │                           │
│  • batteries              │    │                           │
│  • battery_logs           │    │                           │
└───────────────────────────┘    └───────────────────────────┘
```

---

## Infrastructure

### PostgreSQL Database

FlyingForge uses PostgreSQL 16 as its primary data store for all persistent data.

**Connection Configuration:**

| Setting | Default | Environment Variable |
|---------|---------|---------------------|
| Host | `localhost` | `DB_HOST` |
| Port | `5432` | `DB_PORT` |
| User | `postgres` | `DB_USER` |
| Password | `postgres` | `DB_PASSWORD` |
| Database | `mcp_drone` | `DB_NAME` |
| SSL Mode | `disable` | `DB_SSLMODE` |
| Max Open Connections | `25` | - |
| Max Idle Connections | `5` | - |
| Connection Max Lifetime | `5m` | - |

**Database Schema:**

The server automatically runs migrations on startup. Key tables include:

| Table | Description |
|-------|-------------|
| `users` | User accounts with email, password hash, display name, avatar |
| `user_identities` | OAuth provider links (Google, etc.) |
| `refresh_tokens` | JWT refresh token storage with expiration |
| `sellers` | Equipment retailer information |
| `equipment_items` | Catalog of drone equipment from sellers |
| `inventory_items` | User's personal equipment inventory |
| `aircraft` | User's drone configurations |
| `aircraft_components` | Components assigned to aircraft |
| `aircraft_elrs_settings` | ELRS radio configuration per aircraft |
| `radios` | User's radio transmitter configurations |
| `radio_backups` | Radio configuration backup storage |
| `batteries` | User's battery inventory with specs |
| `battery_logs` | Battery charge/discharge cycle history |

**Production Recommendations:**
- Enable SSL mode (`DB_SSLMODE=require`) 
- Use connection pooling (PgBouncer) for high-traffic deployments
- Set up read replicas for scalability
- Configure automated backups
- Monitor with pg_stat_statements

### Redis Cache

Redis 7 provides high-performance caching for API responses and session data.

**Connection Configuration:**

| Setting | Default | Environment Variable |
|---------|---------|---------------------|
| Address | `localhost:6379` | `REDIS_ADDR` |
| Password | (none) | `REDIS_PASSWORD` |
| Database | `0` | `REDIS_DB` |
| Key Prefix | `mcp-news:` | - |

**Cache Usage:**

| Key Pattern | Purpose | TTL |
|-------------|---------|-----|
| `mcp-news:items:*` | Aggregated feed items | 5 minutes |
| `mcp-news:sources` | Source list | 5 minutes |
| `mcp-news:equipment:*` | Equipment search results | 5 minutes |

**Features:**
- Append-only file (AOF) persistence enabled
- Automatic key expiration
- Thread-safe operations via go-redis client

**Production Recommendations:**
- Enable Redis AUTH with a strong password
- Configure maxmemory and eviction policy
- Enable TLS for encrypted connections
- Set up Redis Sentinel or Cluster for HA
- Monitor memory usage and hit rates

---

## Core Components

### 1. Application (`internal/app/app.go`)

The main application coordinator that initializes all services and dependencies.

**Responsibilities:**
- Database connection management
- Cache initialization (memory or Redis)
- Service dependency injection
- HTTP server lifecycle
- Graceful shutdown handling

### 2. Authentication Service (`internal/auth/service.go`)

Handles user authentication and authorization.

**Features:**
- JWT access/refresh token management
- Google OAuth integration
- Password hashing (bcrypt)
- Session management via refresh tokens

**Key Methods:**

| Method | Description |
|--------|-------------|
| `SignUp(email, password, name)` | Create new user account |
| `SignIn(email, password)` | Authenticate with credentials |
| `GoogleAuth(code)` | OAuth authentication via Google |
| `RefreshTokens(refreshToken)` | Issue new token pair |
| `ValidateAccessToken(token)` | Verify JWT and extract claims |

### 3. Database Stores (`internal/database/`)

Data access layer for PostgreSQL operations.

| Store | Responsibilities |
|-------|-----------------|
| `UserStore` | User CRUD, identity linking, token management |
| `EquipmentStore` | Equipment catalog queries, seller management |
| `InventoryStore` | User inventory CRUD, search, filtering |
| `AircraftStore` | Aircraft configs, components, ELRS settings |
| `RadioStore` | Radio profiles, configuration backups |
| `BatteryStore` | Battery inventory, charge logs, health tracking |

### 4. Aggregator (`internal/aggregator/aggregator.go`)

The central component that coordinates all data fetching and processing.

**Responsibilities:**
- Manages all source fetchers
- Coordinates parallel fetching from all sources
- Applies automatic tagging to items
- Deduplicates items by URL and title
- Sorts items by date or score
- Applies filters (sources, tags, date range, search)
- Caches aggregated results

**Key Methods:**

| Method | Description |
|--------|-------------|
| `Refresh(ctx)` | Fetches fresh data from all sources concurrently |
| `GetItems(params)` | Returns filtered and paginated items |
| `GetSources()` | Returns list of all configured sources |

**Filtering Logic:**

```go
FilterParams {
    Limit      int       // Max items to return (default: 50)
    Offset     int       // Pagination offset
    Sources    []string  // Filter by source IDs
    SourceType string    // "news" or "community"
    Query      string    // Full-text search in title/summary
    Sort       string    // "newest" or "score"
    FromDate   string    // Start date filter (YYYY-MM-DD)
    ToDate     string    // End date filter (YYYY-MM-DD)
    Tag        string    // Filter by tag
}
```

### 5. Cache (`internal/cache/`)

Caching layer supporting both in-memory and Redis backends.

**Interface:**
```go
type Cache interface {
    Get(key string) (interface{}, bool)
    Set(key string, value interface{})
    SetWithTTL(key string, value interface{}, ttl time.Duration)
    Delete(key string)
    Clear()
}
```

**Implementations:**

| Backend | Use Case | Configuration |
|---------|----------|---------------|
| `MemoryCache` | Development, single-instance | `CACHE_BACKEND=memory` |
| `RedisCache` | Production, multi-instance | `CACHE_BACKEND=redis` |

**Features:**
- Thread-safe operations
- Configurable TTL (default: 5 minutes)
- Automatic cleanup of expired entries
- JSON serialization for Redis

### 6. Tagger (`internal/tagging/tagger.go`)

Automatic tag inference based on keyword matching.

**Predefined Tag Categories:**

| Tag | Keywords |
|-----|----------|
| FAA | faa, federal aviation, part 107, remote id, airspace |
| DJI | dji, mavic, phantom, mini, air 2, avata, inspire |
| FPV | fpv, first person view, goggles, betaflight, freestyle |
| Racing | racing, race, multiGP, drone racing league, drl |
| Photography | photography, photo, camera, aerial photo |
| Commercial | commercial, enterprise, industrial, professional |
| Military | military, defense, army, navy, warfare |
| Delivery | delivery, package, logistics, amazon, wing, zipline |
| Agriculture | agriculture, farming, crop, spray, precision ag |
| Mapping | mapping, survey, lidar, photogrammetry, gis |
| News | news, announcement, release, update, launch |
| Review | review, test, hands-on, comparison |
| Tutorial | tutorial, how to, guide, tips, learn |
| Regulation | regulation, law, rule, policy, compliance |
| Safety | safety, crash, accident, incident, hazard |
| Technology | technology, tech, innovation, sensor, battery |
| Autonomous | autonomous, ai, machine learning, obstacle avoidance |

### 7. Rate Limiter (`internal/ratelimit/limiter.go`)

Prevents overwhelming external sources with requests.

**Behavior:**
- Tracks last request time per host
- Enforces minimum interval between requests to same host
- Default interval: 1 second
- Thread-safe implementation

---

## Operating Modes

### HTTP Mode (Default)

Started when `MCP_MODE` is not set or is `false`.

```bash
# Start in HTTP mode
./server -http :8080

# Or via environment
HTTP_ADDR=:8080 ./server
```

The server:
1. Starts HTTP server on specified address
2. Registers API endpoints
3. Begins background pre-fetch of all sources
4. Serves requests via REST API

### MCP Mode

Started when `MCP_MODE=true` or `-mcp` flag is set.

```bash
# Start in MCP mode
./server -mcp

# Or via environment
MCP_MODE=true ./server
```

The server:
1. Pre-fetches all sources synchronously
2. Reads JSON-RPC requests from stdin
3. Writes JSON-RPC responses to stdout
4. Continues until EOF or SIGTERM

---

## HTTP API Endpoints

### GET `/api/items`

Retrieves aggregated feed items with optional filtering.

**Query Parameters:**

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `limit` | int | 50 | Maximum number of items to return |
| `offset` | int | 0 | Pagination offset |
| `sources` | string | - | Comma-separated source IDs (e.g., `dronedj,r-fpv`) |
| `sourceType` | string | - | Filter by type: `news` or `community` |
| `q` | string | - | Search query (searches title and summary) |
| `sort` | string | `newest` | Sort order: `newest` or `score` |
| `fromDate` | string | - | Start date (YYYY-MM-DD or MM/DD/YYYY) |
| `toDate` | string | - | End date (YYYY-MM-DD or MM/DD/YYYY) |
| `tag` | string | - | Filter by tag |

**Response:**

```json
{
  "items": [
    {
      "id": "8cb57c68cad588a3",
      "title": "DJI Announces New Mini 5 Pro",
      "url": "https://dronedj.com/...",
      "source": "DroneDJ",
      "sourceType": "rss",
      "author": "Josh Smith",
      "summary": "DJI has unveiled...",
      "content": "Full article content...",
      "publishedAt": "2026-01-30T10:00:00Z",
      "fetchedAt": "2026-01-30T12:00:00Z",
      "thumbnail": "https://...",
      "tags": ["DJI", "News", "Photography"],
      "engagement": {
        "upvotes": 150,
        "comments": 42
      }
    }
  ],
  "totalCount": 228,
  "fetchedAt": "2026-01-30T12:00:00Z",
  "sourceCount": 10
}
```

**Example Requests:**

```bash
# Get latest 20 items
curl "http://localhost:8080/api/items?limit=20"

# Get FPV community posts from today
curl "http://localhost:8080/api/items?sources=r-fpv&fromDate=2026-01-30"

# Search for DJI-related news
curl "http://localhost:8080/api/items?q=dji&sourceType=news"

# Get top-scoring posts
curl "http://localhost:8080/api/items?sort=score&limit=10"
```

---

### GET `/api/sources`

Returns all configured news sources.

**Response:**

```json
{
  "sources": [
    {
      "id": "dronedj",
      "name": "DroneDJ",
      "url": "https://dronedj.com/feed/",
      "sourceType": "news",
      "description": "RSS feed from DroneDJ",
      "feedType": "rss",
      "enabled": true
    },
    {
      "id": "r-fpv",
      "name": "r/fpv",
      "url": "https://www.reddit.com/r/fpv",
      "sourceType": "community",
      "description": "Reddit community r/fpv",
      "feedType": "reddit",
      "enabled": true
    }
  ],
  "count": 10
}
```

**Source Types:**

| Type | Description |
|------|-------------|
| `news` | Professional news sites (RSS feeds) |
| `community` | Community forums (Reddit, forums) |

---

### POST `/api/refresh`

Triggers a manual refresh of all feed sources.

**Request:** No body required

**Response (Success):**

```json
{
  "status": "success",
  "message": "Feed refreshed successfully"
}
```

**Response (Error):**

```json
{
  "status": "error",
  "message": "Timeout fetching from DroneDJ"
}
```

**Notes:**
- Has a 2-minute timeout
- Fetches from all sources concurrently
- Returns success even if some sources fail (partial refresh)

---

### GET `/health`

Health check endpoint for monitoring and load balancers.

**Response:**

```json
{
  "status": "healthy"
}
```

---

## MCP Protocol

The server implements the [Model Context Protocol](https://modelcontextprotocol.io/) for AI assistant integration.

### Protocol Version

`2024-11-05`

### Supported Methods

| Method | Description |
|--------|-------------|
| `initialize` | Handshake and capability negotiation |
| `initialized` | Acknowledgment (no response) |
| `tools/list` | List available tools |
| `tools/call` | Execute a tool |
| `ping` | Health check |

### Available Tools

#### `get_drone_news`

Retrieves drone news and community posts.

**Input Schema:**

```json
{
  "type": "object",
  "properties": {
    "limit": {
      "type": "integer",
      "description": "Maximum number of items to return (default: 20)"
    },
    "sources": {
      "type": "array",
      "items": { "type": "string" },
      "description": "Filter by source IDs"
    },
    "tag": {
      "type": "string",
      "description": "Filter by tag (e.g., DJI, FPV, FAA)"
    },
    "query": {
      "type": "string",
      "description": "Search query to filter items"
    }
  }
}
```

**Example Usage (JSON-RPC):**

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "get_drone_news",
    "arguments": {
      "limit": 5,
      "tag": "DJI"
    }
  }
}
```

#### `get_drone_news_sources`

Lists all available news sources.

**Input Schema:**

```json
{
  "type": "object",
  "properties": {}
}
```

#### `refresh_drone_news`

Manually refreshes the feed from all sources.

**Input Schema:**

```json
{
  "type": "object",
  "properties": {}
}
```

### MCP Response Format

All tool responses are wrapped in a content array:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "content": [
      {
        "type": "text",
        "text": "{ ... JSON result ... }"
      }
    ],
    "isError": false
  }
}
```

---

## Data Models

### FeedItem

Represents a single news item or community post.

```go
type FeedItem struct {
    ID          string      // SHA256 hash of source + URL (first 8 bytes)
    Title       string      // Article/post title
    URL         string      // Link to original content
    Source      string      // Source name (e.g., "DroneDJ", "r/fpv")
    SourceType  string      // "rss" or "reddit"
    Author      string      // Author name (if available)
    Summary     string      // Short description or excerpt
    Content     string      // Full content (if available)
    PublishedAt time.Time   // Original publication time
    FetchedAt   time.Time   // When we fetched it
    Thumbnail   string      // Image URL (if available)
    Tags        []string    // Inferred and original tags
    Engagement  *Engagement // Upvotes/comments (Reddit only)
}
```

### SourceInfo

Describes a configured news source.

```go
type SourceInfo struct {
    ID          string  // Unique identifier (e.g., "dronedj", "r-fpv")
    Name        string  // Display name
    URL         string  // Feed URL
    SourceType  string  // "news" or "community"
    Description string  // Human-readable description
    FeedType    string  // "rss", "reddit", or "forum"
    Enabled     bool    // Whether source is active
}
```

---

## Source Fetchers

### RSS Fetcher

Fetches content from standard RSS/Atom feeds using `gofeed` library.

**Configured Sources:**

| Source | URL |
|--------|-----|
| DroneDJ | https://dronedj.com/feed/ |
| DroneLife | https://dronelife.com/feed/ |
| sUAS News | https://www.suasnews.com/feed/ |
| DroneBlog | https://www.droneblog.com/feed/ |
| Haye's UAV | https://www.yourdroneadvisor.com/feed/ |
| Commercial UAV News | https://www.commercialuavnews.com/feed |

**Processing:**
1. Rate-limited request to feed URL
2. Parse RSS/Atom XML
3. Extract title, URL, author, description, content, published date
4. Generate unique ID from source + URL hash
5. Extract thumbnail from enclosure/media

### Reddit Fetcher

Fetches hot posts from drone-related subreddits using Reddit's JSON API.

**Configured Subreddits:**

| Subreddit | Description |
|-----------|-------------|
| r/drones | General drone discussion |
| r/djimavic | DJI Mavic series |
| r/fpv | FPV flying and racing |
| r/Multicopter | Multirotor building/flying |

**Processing:**
1. Rate-limited request to `/r/{subreddit}/hot.json`
2. Parse Reddit JSON response
3. Extract title, selftext, author, permalink, score, comments
4. Include post flair as initial tag
5. Capture engagement metrics (upvotes, comments)

### Forum Fetcher (Extensible)

HTML scraping fetcher for web forums. Currently configured but no active sources.

**Features:**
- Configurable CSS selectors for post extraction
- Supports custom link/title/date selectors
- Rate-limited requests

---

## Configuration

### Command Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-http` | `:8080` | HTTP server address |
| `-mcp` | `false` | Run in MCP stdio mode |
| `-cache-ttl` | `5m` | Cache TTL for feed items |
| `-rate-limit` | `1s` | Minimum delay between requests |
| `-log-level` | `info` | Log level (debug/info/warn/error) |

### Environment Variables

#### Server Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_ADDR` | `:8080` | HTTP server address |
| `MCP_MODE` | `false` | Set to `true` or `1` for MCP mode |
| `LOG_LEVEL` | `info` | Log level (debug/info/warn/error) |
| `RATE_LIMIT` | `1s` | Rate limit interval between requests |
| `CORS_ORIGIN` | `*` | Allowed CORS origins |

#### Database Configuration (PostgreSQL)

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `postgres` | Database username |
| `DB_PASSWORD` | `postgres` | Database password |
| `DB_NAME` | `mcp_drone` | Database name |
| `DB_SSLMODE` | `disable` | SSL mode (disable/require/verify-full) |

#### Cache Configuration (Redis)

| Variable | Default | Description |
|----------|---------|-------------|
| `CACHE_BACKEND` | `memory` | Cache backend (`memory` or `redis`) |
| `CACHE_TTL` | `5m` | Cache TTL (e.g., `10m`, `1h`) |
| `REDIS_ADDR` | `localhost:6379` | Redis server address |
| `REDIS_PASSWORD` | (empty) | Redis password |
| `REDIS_DB` | `0` | Redis database number |

#### Authentication Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `AUTH_JWT_SECRET` | (required) | Secret key for JWT signing |
| `AUTH_JWT_ISSUER` | `flyingforge` | JWT issuer claim |
| `AUTH_JWT_AUDIENCE` | `flyingforge-users` | JWT audience claim |
| `ACCESS_TOKEN_TTL` | `15m` | Access token expiration |
| `REFRESH_TOKEN_TTL` | `7d` | Refresh token expiration |
| `GOOGLE_CLIENT_ID` | (required for OAuth) | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | (required for OAuth) | Google OAuth client secret |
| `GOOGLE_REDIRECT_URI` | (required for OAuth) | OAuth callback URL |

### Adding New Sources

To add a new RSS source, edit `internal/sources/rss.go`:

```go
func CreateDroneRSSFetchers(limiter *ratelimit.Limiter, config FetcherConfig) []Fetcher {
    sources := []struct {
        name string
        url  string
    }{
        {"DroneDJ", "https://dronedj.com/feed/"},
        // Add new source here:
        {"New Source", "https://newsource.com/feed/"},
    }
    // ...
}
```

To add a new subreddit, edit `internal/sources/reddit.go`:

```go
func CreateDroneRedditFetchers(limiter *ratelimit.Limiter, config FetcherConfig) []Fetcher {
    subreddits := []string{
        "drones",
        "djimavic",
        "fpv",
        "Multicopter",
        // Add new subreddit here:
        "newsubreddit",
    }
    // ...
}
```

---

## Error Handling

The server is designed to be resilient to individual source failures:

1. **Partial Refresh**: If some sources fail during refresh, successful sources still update
2. **Timeout Handling**: Each source fetch has a configurable timeout (default: 30s)
3. **Rate Limiting**: Prevents 429 errors from sources
4. **Graceful Degradation**: HTTP API returns cached data if refresh fails

### Common Errors

| Error | Cause | Resolution |
|-------|-------|------------|
| `TLS certificate error` | Self-signed or expired cert | Source may be temporarily unavailable |
| `403 Forbidden` | Rate limited or blocked | Wait and retry, check User-Agent |
| `404 Not Found` | Feed URL changed | Update source configuration |
| `Timeout` | Slow source or network | Increase timeout or retry |

---

## Logging

The server uses structured JSON logging.

**Log Levels:**

| Level | Description |
|-------|-------------|
| `DEBUG` | Detailed request/response info |
| `INFO` | Normal operations (fetches, startup) |
| `WARN` | Non-fatal issues (source failures) |
| `ERROR` | Fatal errors requiring attention |

**Example Log Output:**

```json
{"timestamp":"2026-01-30T12:00:00Z","level":"INFO","message":"Fetched items from source","fields":{"source":"DroneDJ","count":25}}
{"timestamp":"2026-01-30T12:00:01Z","level":"WARN","message":"Failed to fetch from source","fields":{"source":"DroneLife","error":"403 Forbidden"}}
{"timestamp":"2026-01-30T12:00:02Z","level":"INFO","message":"Aggregation complete","fields":{"total_items":228,"sources_used":10}}
```

---

## Production Deployment

### Docker Compose Stack

FlyingForge is deployed as a multi-container application using Docker Compose:

```yaml
services:
  postgres:     # PostgreSQL 16 - Primary data store
  redis:        # Redis 7 - Caching layer
  server:       # Go API server
  web:          # React frontend (Nginx)
```

### Service Dependencies

```
┌─────────────┐
│     web     │
│   (nginx)   │
└──────┬──────┘
       │ HTTP
       ▼
┌─────────────┐
│   server    │
│   (Go API)  │
└──────┬──────┘
       │
   ┌───┴───┐
   │       │
   ▼       ▼
┌──────┐ ┌──────┐
│ psql │ │redis │
└──────┘ └──────┘
```

### Health Checks

All services include health checks for orchestration:

| Service | Check | Interval |
|---------|-------|----------|
| PostgreSQL | `pg_isready -U postgres` | 10s |
| Redis | `redis-cli ping` | 10s |
| Server | `GET /health` | 30s |

### Production Environment Variables

Create a `.env` file for production secrets:

```bash
# Database
DB_HOST=postgres
DB_PORT=5432
DB_USER=flyingforge
DB_PASSWORD=<strong-password>
DB_NAME=flyingforge
DB_SSLMODE=require

# Redis
CACHE_BACKEND=redis
REDIS_ADDR=redis:6379
REDIS_PASSWORD=<strong-password>

# Authentication
AUTH_JWT_SECRET=<256-bit-random-secret>
GOOGLE_CLIENT_ID=<google-oauth-client-id>
GOOGLE_CLIENT_SECRET=<google-oauth-client-secret>
GOOGLE_REDIRECT_URI=https://flyingforge.app/api/auth/google/callback

# Server
CORS_ORIGIN=https://flyingforge.app
LOG_LEVEL=info
```

### Security Considerations

1. **Database Security**
   - Use strong, unique passwords
   - Enable SSL connections (`DB_SSLMODE=require`)
   - Restrict network access to database ports
   - Regular automated backups

2. **Redis Security**
   - Enable AUTH with password
   - Consider Redis TLS for encrypted connections
   - Don't expose Redis to public network

3. **JWT Security**
   - Use cryptographically secure random secret (256+ bits)
   - Rotate secrets periodically
   - Short access token TTL (15 minutes)
   - Store refresh tokens securely

4. **Network Security**
   - Use HTTPS/TLS for all external traffic
   - Configure proper CORS origins
   - Use reverse proxy (nginx) for SSL termination

### Scaling Considerations

| Component | Horizontal Scaling | Notes |
|-----------|-------------------|-------|
| Web (Nginx) | ✅ Stateless | Load balancer required |
| Server (Go) | ✅ Stateless | Redis required for shared cache |
| PostgreSQL | ⚠️ Primary + Replicas | Use read replicas for scaling reads |
| Redis | ⚠️ Cluster/Sentinel | For HA and scaling |

### Monitoring Recommendations

- **Metrics**: Prometheus + Grafana for server/database metrics
- **Logging**: Aggregate logs with Loki or ELK stack
- **Alerts**: Set up alerts for:
  - Database connection failures
  - Redis connection failures
  - High error rates
  - Slow API response times
  - Disk space on volumes

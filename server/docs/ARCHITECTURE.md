# MCP News Feed Server Architecture

This document provides a comprehensive overview of the drone news aggregator MCP server architecture, including all components, endpoints, and data flows.

## Table of Contents

1. [Overview](#overview)
2. [Architecture Diagram](#architecture-diagram)
3. [Core Components](#core-components)
4. [Operating Modes](#operating-modes)
5. [HTTP API Endpoints](#http-api-endpoints)
6. [MCP Protocol](#mcp-protocol)
7. [Data Models](#data-models)
8. [Source Fetchers](#source-fetchers)
9. [Configuration](#configuration)

---

## Overview

The MCP News Feed Server is a Go application that aggregates drone-related news and community content from multiple sources. It operates in two modes:

1. **HTTP Mode** (default): Serves a REST API for the React frontend
2. **MCP Mode**: Provides tools to AI assistants via the Model Context Protocol

The server fetches content from RSS feeds and Reddit, applies automatic tagging, deduplicates entries, and serves the aggregated data through either interface.

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              MCP News Feed Server                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────────────────────┐   │
│  │  HTTP API   │     │  MCP Server │     │         Aggregator          │   │
│  │  (port 8080)│     │   (stdio)   │     │                             │   │
│  │             │     │             │     │  ┌───────┐  ┌───────────┐   │   │
│  │ /api/items  │────▶│ tools/call  │────▶│  │ Cache │  │  Tagger   │   │   │
│  │ /api/sources│     │ tools/list  │     │  └───────┘  └───────────┘   │   │
│  │ /api/refresh│     │ initialize  │     │                             │   │
│  │ /health     │     │             │     │       ┌─────────────┐       │   │
│  └─────────────┘     └─────────────┘     │       │ Rate Limiter│       │   │
│         │                   │            │       └─────────────┘       │   │
│         └───────────────────┴────────────┴──────────────┬──────────────┘   │
│                                                         │                   │
│  ┌──────────────────────────────────────────────────────┴──────────────┐   │
│  │                          Source Fetchers                             │   │
│  │                                                                      │   │
│  │  ┌─────────────┐  ┌─────────────────┐  ┌─────────────────────────┐  │   │
│  │  │ RSS Fetcher │  │ Reddit Fetcher  │  │    Forum Fetcher        │  │   │
│  │  │             │  │                 │  │    (extensible)         │  │   │
│  │  │ - DroneDJ   │  │ - r/drones      │  │                         │  │   │
│  │  │ - DroneLife │  │ - r/djimavic    │  │                         │  │   │
│  │  │ - sUAS News │  │ - r/fpv         │  │                         │  │   │
│  │  │ - DroneBlog │  │ - r/Multicopter │  │                         │  │   │
│  │  │ - etc.      │  │                 │  │                         │  │   │
│  │  └─────────────┘  └─────────────────┘  └─────────────────────────┘  │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
                    ┌─────────────────────────────────────┐
                    │          External Sources           │
                    │                                     │
                    │  • RSS Feeds (dronedj.com, etc.)   │
                    │  • Reddit JSON API                  │
                    │  • Web Forums (HTML scraping)       │
                    └─────────────────────────────────────┘
```

---

## Core Components

### 1. Aggregator (`internal/aggregator/aggregator.go`)

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

### 2. Cache (`internal/cache/cache.go`)

In-memory TTL-based cache for storing aggregated results.

**Features:**
- Thread-safe with RWMutex
- Configurable TTL (default: 5 minutes)
- Automatic cleanup of expired entries
- Simple key-value interface

**Methods:**

| Method | Description |
|--------|-------------|
| `Get(key)` | Retrieve cached value if not expired |
| `Set(key, value)` | Store value with default TTL |
| `SetWithTTL(key, value, ttl)` | Store value with custom TTL |

### 3. Tagger (`internal/tagging/tagger.go`)

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

### 4. Rate Limiter (`internal/ratelimit/limiter.go`)

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

| Variable | Description |
|----------|-------------|
| `HTTP_ADDR` | HTTP server address (overrides `-http`) |
| `MCP_MODE` | Set to `true` or `1` for MCP mode |
| `CACHE_TTL` | Cache TTL (e.g., `10m`, `1h`) |
| `RATE_LIMIT` | Rate limit interval (e.g., `2s`) |
| `LOG_LEVEL` | Log level (debug/info/warn/error) |

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

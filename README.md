# MCP Drone News Feed

A complete drone news aggregator featuring an MCP (Model Context Protocol) server written in Go and a React web application. Aggregates news and community content from multiple drone-related sources with intelligent caching, deduplication, and tag inference.

## Features

### Server (Go)
- **MCP Protocol Support**: Full MCP tools implementation for AI assistant integration
- **HTTP REST API**: Simple REST endpoints for the React frontend
- **Multiple Sources**: RSS feeds, Reddit JSON API, and forum scraping
- **Smart Caching**: Configurable TTL cache with automatic cleanup
- **Rate Limiting**: Per-host rate limiting to be respectful to sources
- **Deduplication**: URL canonicalization and duplicate detection
- **Tag Inference**: Automatic tagging based on content keywords (FAA, DJI, FPV, etc.)
- **Graceful Failure**: Partial results when individual sources fail
- **Structured Logging**: JSON-formatted logs for easy parsing
- **Synchronous Image Moderation**: Rekognition byte-based moderation gates avatar/aircraft/gear uploads before persistence

### Web App (React + TypeScript)
- **Modern UI**: Clean, dark-themed interface with Tailwind CSS
- **Source Filtering**: Filter by individual sources or source type (news/community)
- **Search**: Full-text search across titles, summaries, and content
- **Date Range**: Filter items by publication date
- **Sort Options**: Sort by newest or top score
- **Detail View**: Modal with full item details and external links
- **Persistent Filters**: Filters saved to localStorage
- **Responsive Design**: Works on desktop and tablet

## Folder Structure

```
flyingforge/
├── server/                    # Go MCP server
│   ├── cmd/
│   │   └── server/
│   │       └── main.go        # Entry point
│   ├── internal/
│   │   ├── aggregator/        # Feed aggregation logic
│   │   ├── cache/             # TTL cache implementation
│   │   ├── httpapi/           # REST API handlers
│   │   ├── logging/           # Structured logging
│   │   ├── mcp/               # MCP protocol handlers
│   │   ├── models/            # Data models
│   │   ├── ratelimit/         # Per-host rate limiting
│   │   ├── sources/           # Source fetchers (RSS, Reddit, forums)
│   │   └── tagging/           # Tag inference engine
│   ├── go.mod
│   └── .env.example
├── web/                       # React web application
│   ├── src/
│   │   ├── components/        # React components
│   │   ├── api.ts             # API client
│   │   ├── hooks.ts           # Custom hooks
│   │   ├── types.ts           # TypeScript types
│   │   └── App.tsx            # Main app component
│   ├── package.json
│   ├── vite.config.ts
│   ├── tailwind.config.js
│   └── .env.example
├── mcp.json                   # MCP server configuration
├── docker-compose.yml         # Docker deployment
├── Dockerfile.server
├── Dockerfile.web
└── README.md
```

## Quick Start

### Prerequisites
- Go 1.22+
- Node.js 20+
- npm or yarn

### 1. Start the Go Server

```bash
cd server

# Install dependencies
go mod tidy

# Copy and configure environment
cp .env.example .env

# Run the server (HTTP mode)
go run ./cmd/server

# Or run in MCP mode for AI assistant integration
go run ./cmd/server -mcp
```

The server will start on `http://localhost:8080` by default.

### 2. Start the React App

```bash
cd web

# Install dependencies
npm install

# Copy environment file
cp .env.example .env

# Start development server
npm run dev
```

The web app will be available at `http://localhost:5173`.

## Configuration

### Server Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_ADDR` | `:8080` | HTTP server address |
| `MCP_MODE` | `false` | Run in MCP stdio mode |
| `CACHE_TTL` | `5m` | Cache TTL for feed items |
| `RATE_LIMIT` | `1s` | Min delay between requests to same host |
| `LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `CORS_ORIGIN` | `*` | CORS allowed origin |
| `IMAGE_MODERATION_ENABLED` | `true` | Enable synchronous Rekognition moderation pipeline |
| `AWS_REGION` | (required) | AWS region for Rekognition |
| `MODERATION_REJECT_CONFIDENCE` | `70` | Reject threshold for moderation labels |
| `MODERATION_TIMEOUT` | `5s` | Per-image moderation timeout |
| `MODERATION_PENDING_TTL` | `10m` | TTL for approved-but-not-yet-saved upload tokens |

### Web Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VITE_API_BASE_URL` | (empty) | API base URL (uses Vite proxy in dev) |

## API Reference

### REST Endpoints

#### GET /api/items
Get feed items with optional filtering.

Query parameters:
- `limit` (int): Max items to return (default: 50, max: 100)
- `sources` (string): Comma-separated source IDs
- `sourceType` (string): `news` or `community`
- `q` (string): Search query
- `sort` (string): `newest` or `score`
- `fromDate` (string): ISO date (YYYY-MM-DD)
- `toDate` (string): ISO date (YYYY-MM-DD)

#### GET /api/items/:id
Get a single item by ID.

#### GET /api/sources
List all available sources.

#### POST /api/refresh
Force refresh feeds. Request body:
```json
{
  "sources": ["dronelife", "reddit-drones"]
}
```

#### POST /api/images/upload
Moderates an uploaded image (multipart/form-data `image`) synchronously and returns:
```json
{
  "status": "APPROVED | REJECTED | PENDING_REVIEW",
  "reason": "optional user-safe message",
  "uploadId": "present only when APPROVED"
}
```

#### POST /api/users/avatar
Persists a custom avatar only after moderation approval:
```json
{
  "uploadId": "approved token returned by /api/images/upload"
}
```

#### GET /health
Health check endpoint.

### MCP Tools

The server exposes these MCP tools:

#### get_latest
Get the latest drone news and community posts.
```json
{
  "limit": 20,
  "sources": ["dronelife"],
  "sourceType": "news",
  "since": "2024-01-01T00:00:00Z"
}
```

#### search
Search for items by keyword.
```json
{
  "query": "DJI Mini",
  "limit": 20,
  "fromDate": "2024-01-01T00:00:00Z"
}
```

#### get_item
Get a single item by ID.
```json
{
  "id": "abc123..."
}
```

#### list_sources
List all available sources.

#### refresh
Force refresh of feeds.
```json
{
  "sources": ["dronelife", "dronedj"]
}
```

## Image Moderation Notes

- User-uploaded avatar, aircraft, and gear images are moderated synchronously with Rekognition `DetectModerationLabels` using raw bytes (no S3 required).
- If moderation fails or times out, the API returns `PENDING_REVIEW`; frontend must treat this as not approved.
- Unapproved bytes are never persisted.
- Approved bytes are stored through a storage abstraction backed by `image_assets` (DB), so storage can be swapped to S3 later without changing moderation/UI flow.

### Local Rekognition smoke test

```bash
AWS_PROFILE=dev AWS_REGION=us-east-1 make rekognition-test IMAGE=./testdata/avatar_safe.jpg
```

### IAM (least privilege)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "rekognition:DetectModerationLabels",
      "Resource": "*"
    }
  ]
}
```

## Sources

### News (RSS)
| ID | Name | URL |
|----|------|-----|
| `dronelife` | DroneLife | dronelife.com |
| `dronedj` | DroneDJ | dronedj.com |
| `suasnews` | sUAS News | suasnews.com |
| `droneu` | Drone U | thedroneu.com |
| `droneblog` | Droneblog | droneblog.com |

### Community (Reddit/Forums)
| ID | Name | URL |
|----|------|-----|
| `reddit-drones` | r/drones | reddit.com/r/drones |
| `mavicpilots` | MavicPilots Forum | mavicpilots.com |
| `parrotpilots` | ParrotPilots Forum | parrotpilots.com |
| `commercialdronepilots` | Commercial Drone Pilots | commercialdronepilots.com |
| `fpvdronepilots` | FPV Drone Pilots | fpvdronepilots.com |

## Adding a New Source

### RSS Source

1. Create or modify a file in `server/internal/sources/`:

```go
// In rss.go
func NewMySource() *RSSFetcher {
    return NewRSSFetcher(models.SourceInfo{
        ID:          "mysource",
        Name:        "My Source",
        URL:         "https://mysource.com",
        SourceType:  models.SourceTypeNews,
        Description: "Description here",
        FeedType:    "rss",
        Enabled:     true,
    }, "https://mysource.com/feed/")
}
```

2. Register in `server/internal/sources/fetcher.go`:

```go
func AllSources() map[string]Fetcher {
    return map[string]Fetcher{
        // ... existing sources
        "mysource": NewMySource(),
    }
}
```

### Forum/Scrape Source

1. Add a new fetcher using `ForumFetcher`:

```go
func NewMyForum() *ForumFetcher {
    return NewForumFetcher(models.SourceInfo{
        ID:          "myforum",
        Name:        "My Forum",
        URL:         "https://myforum.com",
        SourceType:  models.SourceTypeCommunity,
        Description: "Forum description",
        FeedType:    "scrape",
        Enabled:     true,
    }, ForumConfig{
        BaseURL:       "https://myforum.com",
        ListPath:      "/forums/news.1/",
        ItemSelector:  ".thread-item",
        TitleSelector: ".thread-title a",
        LinkSelector:  ".thread-title a",
        AuthorSel:     ".author",
        DateSel:       "time",
    })
}
```

## Deployment

### Docker Compose

```bash
# Build and start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

The web app will be available at `http://localhost:3000`.

### Manual Deployment

1. Build the server:
```bash
cd server
go build -o flyingforge ./cmd/server
./flyingforge
```

2. Build the web app:
```bash
cd web
npm run build
# Serve the dist/ folder with any static server
```

### MCP Integration

Add to your MCP client configuration (e.g., Claude Desktop):

```json
{
  "mcpServers": {
    "drone-news-feed": {
      "command": "/path/to/flyingforge",
      "args": ["-mcp"],
      "env": {
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

Or using `go run`:

```json
{
  "mcpServers": {
    "drone-news-feed": {
      "command": "go",
      "args": ["run", "./cmd/server", "-mcp"],
      "cwd": "/path/to/flyingforge/server"
    }
  }
}
```

## Normalized Item Schema

All items from all sources are normalized to this schema:

```typescript
interface FeedItem {
  id: string;              // Stable hash of source + URL
  title: string;
  url: string;
  source: string;          // Source ID
  sourceType: "news" | "community";
  publishedAt?: string;    // ISO 8601
  author?: string;
  summary?: string;
  contentText?: string;
  tags: string[];          // Inferred + from source
  score?: number;          // Reddit upvotes, etc.
  commentsUrl?: string;
  media?: {
    imageUrl?: string;
  };
}
```

## Tag Inference

The server automatically infers tags based on content keywords:

- **Regulatory**: FAA, Part 107, Remote ID, BVLOS, UTM, Airspace
- **Brands**: DJI, Autel, Skydio, Parrot, Yuneec
- **Use Cases**: FPV, Photography, Mapping, Inspection, Agriculture, Delivery
- **Technology**: AI, Autonomous, Battery, Sensors, SDK
- **Content Types**: Review, Tutorial, News, Event

## License

MIT

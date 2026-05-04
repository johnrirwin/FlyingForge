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

The server will start on `http://localhost:8080` by default. In HTTP mode it also exposes the MCP endpoint at `http://localhost:8080/mcp`.

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
| `MCP_PUBLIC_BASE_URL` | (empty) | Public HTTPS base URL used for MCP protected-resource metadata |
| `MCP_ALLOWED_ORIGINS` | `https://chatgpt.com,https://chat.openai.com` | Allowed browser origins for the HTTP MCP endpoint |
| `MCP_AUTH_ISSUER` | (empty) | OIDC issuer for linked-user MCP OAuth |
| `MCP_AUTH_AUDIENCE` | (empty) | Expected audience for MCP access tokens |
| `MCP_AUTH_RESOURCE` | `MCP_PUBLIC_BASE_URL + /mcp` | Protected resource identifier for MCP OAuth |
| `MCP_AUTH_SCOPES` | `flyingforge.read` | Comma-separated scopes required for private MCP tools |
| `MCP_AUTH_DISCOVERY_URL` | (empty) | Optional OIDC discovery override |
| `MCP_AUTH_JWKS_URL` | (empty) | Optional JWKS override |
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
| `VITE_API_URL` | `http://localhost:8080` | Backend origin used by the Google OAuth redirect flow |
| `VITE_GOOGLE_CLIENT_ID` | (required for Google sign-in) | Google OAuth client ID used by the web frontend |

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

#### Public Builds
- `GET /api/public/builds?sort=newest&frameFilter=`
- `GET /api/public/builds/{id}`

#### Temporary Build Builder
- `POST /api/builds/temp` → creates a 24-hour temporary build URL (`/builds/temp/{token}`)
- `GET /api/builds/temp/{token}`
- `PUT /api/builds/temp/{token}`
- `POST /api/builds/temp/{token}/share` → promotes a temporary build link to non-expiring shared status

#### Authenticated Build Management
- `GET /api/builds`
- `POST /api/builds`
- `POST /api/builds/from-aircraft/{aircraftId}`
- `GET /api/builds/{id}`
- `PUT /api/builds/{id}`
- `DELETE /api/builds/{id}`
- `POST /api/builds/{id}/publish` → submits to moderation queue (`PENDING_REVIEW`)
- `POST /api/builds/{id}/unpublish`

#### Content Moderation (Admin / Content Admin)
- `GET /api/admin/gear`
- `GET /api/admin/gear/{id}`
- `PUT /api/admin/gear/{id}`
- `POST /api/admin/gear/{id}/image`
- `GET /api/admin/gear/{id}/image`
- `DELETE /api/admin/gear/{id}/image`
- `GET /api/admin/builds?status=PENDING_REVIEW`
- `GET /api/admin/builds/{id}`
- `PUT /api/admin/builds/{id}`
- `POST /api/admin/builds/{id}/image`
- `GET /api/admin/builds/{id}/image`
- `DELETE /api/admin/builds/{id}/image`
- `POST /api/admin/builds/{id}/publish`

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

The server exposes MCP over:

- **stdio** when running `go run ./cmd/server -mcp`
- **HTTP** at `/mcp` when running the normal HTTP server

The HTTP endpoint is designed for hosted browser-based MCP connectors and supports OAuth-protected private tools.

#### Public read-only tools

- `get_drone_news`
- `get_drone_news_sources`
- `search_equipment`
- `get_equipment_by_category`
- `get_sellers`

These tools work without authentication.

#### Private read-only tools

- `list_my_aircraft`
- `get_aircraft_details`
- `get_aircraft_receiver_summary`
- `get_aircraft_tuning`
- `list_my_radios`
- `get_radio_details`
- `list_radio_backups`

These tools require a linked OAuth identity with the `flyingforge.read` scope.

#### Private tool data boundaries

- `get_aircraft_details` returns aircraft metadata plus component assignments, but not raw receiver JSON.
- `get_aircraft_receiver_summary` returns only sanitized receiver settings.
- `get_aircraft_tuning` returns parsed/latest tuning metadata, but not raw CLI dumps or `diffBackup`.
- `list_radio_backups` returns backup metadata only, never file bytes.

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

#### Local stdio MCP (any stdio-compatible client)

Add to your MCP client configuration:

```json
{
  "mcpServers": {
    "flyingforge": {
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
    "flyingforge": {
      "command": "go",
      "args": ["run", "./cmd/server", "-mcp"],
      "cwd": "/path/to/flyingforge/server"
    }
  }
}
```

#### Hosted HTTPS MCP connectors

1. Run the normal HTTP server:
   ```bash
   cd server
   go run ./cmd/server
   ```
2. Expose it over **public HTTPS**. For local dogfooding, either of these works:
   ```bash
   ngrok http 8080
   ```
   or
   ```bash
   cloudflared tunnel --url http://localhost:8080
   ```
3. Set MCP environment variables so the public base URL matches the tunnel:
   ```bash
   export MCP_PUBLIC_BASE_URL="https://your-public-host.example.com"
   export MCP_AUTH_ISSUER="https://your-oidc-issuer.example.com"
   export MCP_AUTH_AUDIENCE="your-audience"
   export MCP_AUTH_RESOURCE="https://your-public-host.example.com/mcp"
   export MCP_AUTH_SCOPES="flyingforge.read"
   ```
   Recommended path: use an external OIDC provider that federates Google sign-in.
   Users can still authenticate with the same Google account they use on the
   FlyingForge frontend, but the MCP access token should come from the OIDC
   provider rather than directly from the frontend Google OAuth app.
4. In your hosted MCP client, create a connector pointing at:
   - MCP URL: `https://your-public-host.example.com/mcp`
5. Confirm the client can complete a linked-user prompt such as:
   - “Show my aircraft and latest tuning settings.”

The MCP host also serves protected-resource discovery at:

- `https://your-public-host.example.com/.well-known/oauth-protected-resource`

For production deployments behind the bundled Terraform/CloudFront stack:

- Route both `/mcp` and `/.well-known/oauth-protected-resource` to the backend service rather than the SPA origin.
- Set `MCP_AUTH_ISSUER` to enable hosted MCP OAuth at all.
- Set `MCP_PUBLIC_BASE_URL` so the OAuth challenge can advertise the correct protected-resource metadata URL.
- If you use the included GitHub Actions deploy workflow, configure these repository variables:
  - `MCP_AUTH_ISSUER`
  - `MCP_AUTH_AUDIENCE`
  - optional `MCP_AUTH_DISCOVERY_URL`
  - optional `MCP_AUTH_JWKS_URL`

### Recommended Google-backed MCP OAuth setup

For the cleanest production setup:

1. Keep the existing frontend Google sign-in for the web app.
2. Add an external OIDC provider for MCP OAuth (for example Auth0, WorkOS, or Clerk).
3. Configure that provider to use Google as the upstream identity provider.
4. Point FlyingForge MCP at the provider via:
   - `MCP_AUTH_ISSUER`
   - `MCP_AUTH_AUDIENCE`
   - optional discovery/JWKS overrides

In practice, this usually means creating a **separate Google OAuth client in GCP**
for the OIDC provider to use. Users still sign in with the same Google account,
and FlyingForge maps MCP access to the same local user by:

1. existing linked MCP identity (`issuer|sub`)
2. otherwise verified email match
3. otherwise new user creation

The same verified-email fallback is also enforced for frontend Google sign-in, so
web sign-in and MCP sign-in stay consistent.

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

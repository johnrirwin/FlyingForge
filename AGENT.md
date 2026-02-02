# Agent Instructions

## Project Overview

This is a **Drone News Aggregator** - an MCP (Model Context Protocol) server written in Go that aggregates drone news from multiple sources, paired with a React web application for viewing the feed.

## Architecture

```
flyingforge/
├── server/           # Go MCP server
│   ├── cmd/server/   # Entry point
│   └── internal/     # Internal packages
│       ├── models/       # Data structures
│       ├── logging/      # Structured JSON logging
│       ├── cache/        # TTL-based caching
│       ├── ratelimit/    # Per-host rate limiting
│       ├── tagging/      # Keyword-based tag inference
│       ├── sources/      # Feed fetchers (RSS, Reddit, Forum)
│       ├── aggregator/   # Feed aggregation & deduplication
│       ├── mcp/          # MCP JSON-RPC protocol handlers
│       └── httpapi/      # REST API for web frontend
├── web/              # React TypeScript frontend
│   └── src/
│       ├── components/   # UI components
│       ├── api.ts        # API client
│       ├── hooks.ts      # Custom React hooks
│       └── types.ts      # TypeScript interfaces
└── docker-compose.yml
```

## Tech Stack

### Backend (Go 1.22+)
- **MCP Protocol**: JSON-RPC 2.0 over stdio for AI assistant integration
- **HTTP API**: REST endpoints for React frontend
- **Dependencies**: goquery (HTML parsing), gofeed (RSS parsing)

### Frontend (React 18)
- **Build**: Vite + TypeScript
- **Styling**: Tailwind CSS (dark theme)
- **State**: React hooks with localStorage persistence

## Key Files

| File | Purpose |
|------|---------|
| `server/cmd/server/main.go` | Server entry point, flag parsing, mode switching |
| `server/internal/aggregator/aggregator.go` | Core feed aggregation logic |
| `server/internal/httpapi/server.go` | REST API endpoints |
| `server/internal/mcp/server.go` | MCP protocol implementation |
| `web/src/App.tsx` | Main React application |
| `web/src/api.ts` | Frontend API client |

## Running the Project

### Development

```bash
# Backend (HTTP mode)
cd server && go run ./cmd/server

# Frontend (dev server with hot reload)
cd web && npm run dev
```

### Production (Docker)

```bash
docker-compose up --build
```

### MCP Mode (for AI assistants)

```bash
cd server && go run ./cmd/server -mcp
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/items` | Get feed items (supports filtering) |
| GET | `/api/sources` | List all news sources |
| POST | `/api/refresh` | Trigger feed refresh |
| GET | `/health` | Health check |

### Query Parameters for `/api/items`
- `limit` - Max items to return (default: 50)
- `offset` - Pagination offset
- `source` - Filter by source name
- `tag` - Filter by tag
- `search` - Full-text search

## MCP Tools

When running in MCP mode, the server exposes:
- `get_drone_news` - Fetch latest drone news with optional filters
- `get_drone_news_sources` - List available sources
- `refresh_drone_news` - Manually refresh feeds

## Data Flow

1. **Fetchers** pull from RSS feeds, Reddit API, and forum scrapers
2. **Aggregator** deduplicates by ID and normalized title
3. **Tagger** infers tags from content keywords (FAA, DJI, FPV, etc.)
4. **Cache** stores items with configurable TTL
5. **API** serves items to frontend or MCP clients

## Common Tasks

### Adding a New News Source
1. Add fetcher config in `server/internal/sources/rss.go` (for RSS) or create new fetcher
2. Register in `CreateDrone*Fetchers()` function

### Adding a New Tag Rule
Edit `server/internal/tagging/tagger.go` - add keyword patterns to `rules` map

### Modifying the UI
Components are in `web/src/components/` - follows standard React patterns

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_ADDR` | `:8080` | HTTP server address |
| `MCP_MODE` | `false` | Run in MCP stdio mode |
| `CACHE_TTL` | `5m` | Cache time-to-live |
| `RATE_LIMIT` | `1s` | Min delay between requests to same host |
| `LOG_LEVEL` | `info` | Logging level (debug/info/warn/error) |

## Testing

### Running Tests

```bash
# Run all tests
make test

# Go tests only
make test-go
cd server && go test -v ./...

# Frontend tests only
make test-web
cd web && npm run test
```

### Test Requirements

**All new code must include tests.** This is a strict requirement for maintaining code quality.

#### Go Backend Tests
- Place test files alongside source files (e.g., `service.go` → `service_test.go`)
- Use table-driven tests for multiple scenarios
- Test exported functions and error conditions
- Required for:
  - New services or packages
  - New API endpoints
  - Business logic changes
  - Model validation functions

Example test structure:
```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"valid input", "foo", "bar"},
        {"empty input", "", ""},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test logic
        })
    }
}
```

#### React Frontend Tests
- Place test files alongside components (e.g., `Component.tsx` → `Component.test.tsx`)
- Use Vitest + React Testing Library
- Test user interactions and rendered output
- Required for:
  - New components
  - Custom hooks
  - Complex UI logic
  - Form validation

Example test structure:
```tsx
import { render, screen } from '@testing-library/react';
import { ComponentName } from './ComponentName';

describe('ComponentName', () => {
  it('renders correctly', () => {
    render(<ComponentName />);
    expect(screen.getByText('Expected Text')).toBeInTheDocument();
  });
});
```

### Current Test Coverage

| Package | Tests |
|---------|-------|
| `aggregator` | Sorting, deduplication, filtering, pagination |
| `aircraft` | Service validation, CRUD operations |
| `auth` | Error types, password validation |
| `battery` | Chemistry validation, cell count validation |
| `cache` | Set/Get, TTL expiration, concurrent access |
| `equipment` | Service errors, search params |
| `httpapi` | JSON responses, CORS, pagination parsing |
| `models` | Type constants, validation functions |
| `ratelimit` | Allow/Wait, host isolation, concurrency |
| `sellers` | Registry operations, seller info |
| `sources` | RSS/Forum fetchers, config defaults |
| `tagging` | Tag inference, rule management |
| **Frontend** | FeedCard, Sidebar, TopBar, useAuth |

## Conventions

- **Commits**: Conventional Commits format (`feat:`, `fix:`, `chore:`, etc.)
- **Go**: Standard Go project layout, internal packages
- **React**: Functional components with hooks, TypeScript strict mode
- **API**: JSON responses, proper HTTP status codes

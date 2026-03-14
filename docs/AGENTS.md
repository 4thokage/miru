# Miru - AI Agent Development Guide

This document provides comprehensive guidance for AI agents working on the Miru project. Miru is a self-hosted manga and anime streaming platform.

## Table of Contents

1. [Project Architecture](#project-architecture)
2. [Adding New Providers](#adding-new-providers)
3. [API Integration Patterns](#api-integration-patterns)
4. [Testing Guidelines](#testing-guidelines)
5. [Code Patterns](#code-patterns)
6. [Configuration](#configuration)
7. [Common Tasks](#common-tasks)

---

## Project Architecture

### Technology Stack

| Component | Technology |
|-----------|------------|
| Frontend | Next.js 16, React 19, Tailwind CSS 4 |
| Backend | Go 1.26+ (chi router) |
| Database | PostgreSQL |
| Cache | Valkey (Redis-compatible) |
| Events | NATS |

### Directory Structure

```
miru/
├── apps/
│   ├── api/                    # Go backend
│   │   ├── main.go            # Entry point & HTTP handlers
│   │   ├── internal/
│   │   │   ├── mangadex/      # MangaDex API client
│   │   │   ├── gogoanime/     # GoGoAnime scraper
│   │   │   ├── scraper/       # Provider interfaces
│   │   │   ├── storage/       # Database operations
│   │   │   ├── middleware/    # Rate limiting
│   │   │   ├── errors/        # Error handling
│   │   │   └── testutils/     # Test utilities
│   │   └── migrations/       # SQL migrations
│   └── web/                   # Next.js frontend
│       ├── app/               # App router pages
│       ├── components/        # React components
│       ├── lib/               # API client & hooks
│       └── __tests__/         # Test files
├── docs/                      # Documentation
├── docker-compose.yml         # Infrastructure
└── .env.sample               # Environment template
```

### Data Flow

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Frontend  │────▶│  Go API     │────▶│  Providers  │
│  (Next.js)  │◀────│  (chi)      │◀────│  (scrapers) │
└─────────────┘     └─────────────┘     └─────────────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
         ┌─────────┐  ┌─────────┐  ┌─────────┐
         │Postgres │  │ Valkey  │  │  NATS  │
         └─────────┘  └─────────┘  └─────────┘
```

---

## Adding New Providers

### Overview

Miru uses a provider interface pattern that makes adding new manga/anime sources straightforward. The interface is defined in `apps/api/internal/scraper/interface.go`.

### Step-by-Step: Adding a New Anime Provider

#### 1. Create the Provider Package

Create `apps/api/internal/newprovider/client.go`:

```go
package newprovider

import (
    "context"
    "encoding/json"
    "errors"
    "net/http"
    "time"

    "github.com/redis/go-redis/v9"
    "miru-api/internal/scraper"
)

type Client struct {
    baseURL   string
    cdnURL    string
    httpClient *http.Client
    cache     *redis.Client
}

func NewClient(cache *redis.Client) *Client {
    return &Client{
        baseURL:   "https://newprovider.example.com",
        cdnURL:    "https://cdn.newprovider.example.com",
        httpClient: &http.Client{Timeout: 30 * time.Second},
        cache:     cache,
    }
}

type SearchResult struct {
    ID        string `json:"id"`
    Title     string `json:"title"`
    Image     string `json:"image"`
    Release   string `json:"release"`
    Status    string `json:"status"`
}

// Implement scraper.AnimeProvider interface
func (c *Client) Search(ctx context.Context, query string, page int) ([]scraper.SearchResult, error) {
    // Implementation here
}

func (c *Client) GetAnimeDetails(ctx context.Context, id string) (*scraper.AnimeDetails, error) {
    // Implementation here
}

func (c *Client) GetEpisodes(ctx context.Context, id string) ([]scraper.Episode, error) {
    // Implementation here
}

func (c *Client) GetStreamingLinks(ctx context.Context, episodeID string) ([]scraper.StreamSource, error) {
    // Implementation here
}

func (c *Client) GetDownloadLinks(ctx context.Context, episodeID string) ([]scraper.DownloadLink, error) {
    // Implementation here
}

func (c *Client) GetRecent(ctx context.Context, page int) ([]scraper.RecentResult, error) {
    // Implementation here
}

func (c *Client) GetPopular(ctx context.Context, page int) ([]scraper.SearchResult, error) {
    // Implementation here
}
```

#### 2. Register in main.go

Add to `apps/api/main.go`:

```go
// Import your provider
"miru-api/internal/newprovider"

// In main():
newProviderClient := newprovider.NewClient(redisClient)

// Add routes
r.Route("/api/v1/anime", func(r chi.Router) {
    // Existing routes using gogoClient
    // Add new routes using newProviderClient
    r.Get("/newprovider/search", handleNewProviderSearch(newProviderClient))
})
```

#### 3. Add Cache Keys

Follow the caching pattern:

```go
const (
    SearchCacheTTL   = 5 * time.Minute
    DetailsCacheTTL = 15 * time.Minute
    EpisodesCacheTTL = 10 * time.Minute
    SourcesCacheTTL  = 2 * time.Minute
)

func cacheKey(endpoint, id string) string {
    return fmt.Sprintf("newprovider:%s:%s", endpoint, id)
}
```

#### 4. Add Tests

Create `apps/api/internal/newprovider/client_test.go`:

```go
package newprovider_test

import (
    "testing"
    "net/http/httptest"
)

func TestSearch(t *testing.T) {
    server := httptest.NewServer(...)
    defer server.Close()
    
    // Test implementation
}
```

---

### Step-by-Step: Adding a New Manga Provider

#### 1. Create the Provider Package

Create `apps/api/internal/newmanga/client.go`:

```go
package newmanga

import (
    "context"
    "encoding/json"
    "net/http"
    "time"

    "github.com/redis/go-redis/v9"
)

type Client struct {
    baseURL string
    httpClient *http.Client
    cache     *redis.Client
}

func NewClient(cache *redis.Client) *Client {
    return &Client{
        baseURL:   "https://api.newmanga.example.com",
        httpClient: &http.Client{Timeout: 30 * time.Second},
        cache:     cache,
    }
}

type Manga struct {
    ID          string   `json:"id"`
    Title       string   `json:"title"`
    Description string   `json:"description"`
    Cover       string   `json:"cover"`
    Author      string   `json:"author"`
    Status      string   `json:"status"`
    Tags        []string `json:"tags"`
}

type Chapter struct {
    ID        string `json:"id"`
    Number    int    `json:"number"`
    Title     string `json:"title"`
    Volume    int    `json:"volume"`
    Pages     int    `json:"pages"`
    Released  string `json:"released"`
}

type ChapterPages struct {
    ChapterID string   `json:"chapter_id"`
    Pages     []string `json:"pages"`
    Server    string   `json:"server"`
}

func (c *Client) Search(ctx context.Context, query string, limit int) ([]Manga, error) {
    // Implementation here
}

func (c *Client) GetMangaDetails(ctx context.Context, id string) (*Manga, []Chapter, error) {
    // Implementation here
}

func (c *Client) GetChapterPages(ctx context.Context, chapterID string) (*ChapterPages, error) {
    // Implementation here
}

func (c *Client) GetTags(ctx context.Context) (map[string]string, error) {
    // Implementation here - return tag name to ID mapping
}
```

---

## API Integration Patterns

### Backend (Go)

#### Handler Pattern

```go
func handleEndpoint(client *ClientType) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")

        // 1. Parse request
        var req RequestType
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            errors.RespondWithError(w, r, errors.BadRequest("invalid request body"))
            return
        }

        // 2. Validate input
        if req.RequiredField == "" {
            errors.RespondWithError(w, r, errors.InvalidInput("required_field", "field is required"))
            return
        }

        // 3. Call service
        result, err := client.ServiceMethod(r.Context(), req)
        if err != nil {
            errors.HandleServiceError(w, r, "ServiceName", err)
            return
        }

        // 4. Respond
        errors.RespondWithSuccess(w, result)
    }
}
```

#### Error Handling

Use the error utilities in `apps/api/internal/errors/errors.go`:

```go
// In handlers:
if err != nil {
    errors.RespondWithError(w, r, errors.BadRequest("invalid input"))
    return
}

if notFound {
    errors.RespondWithError(w, r, errors.NotFound("manga"))
    return
}

if externalErr {
    errors.HandleServiceError(w, r, "MangaDex", err)
    return
}

// Success response
errors.RespondWithSuccess(w, result)
```

#### Caching Pattern

```go
func (c *Client) GetWithCache(ctx context.Context, key string, ttl time.Duration, fetcher func() (interface{}, error)) (interface{}, error) {
    // Check cache
    cached, err := c.cache.Get(ctx, key).Result()
    if err == nil {
        var result interface{}
        json.Unmarshal([]byte(cached), &result)
        return result, nil
    }

    // Fetch fresh
    result, err := fetcher()
    if err != nil {
        return nil, err
    }

    // Cache result
    jsonBytes, _ := json.Marshal(result)
    c.cache.Set(ctx, key, string(jsonBytes), ttl)

    return result, nil
}
```

### Frontend (TypeScript/React)

#### API Client Pattern

```typescript
// apps/web/lib/api.ts
const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

interface APIResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
}

export async function fetchData<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_URL}${endpoint}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  });

  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`);
  }

  const data: APIResponse<T> = await response.json();
  
  if (!data.success || !data.data) {
    throw new Error(data.error || 'Unknown error');
  }

  return data.data;
}

// Usage:
export async function searchManga(params: SearchRequest): Promise<SearchResponse> {
  return fetchData<SearchResponse>('/api/v1/search', {
    method: 'POST',
    body: JSON.stringify(params),
  });
}
```

#### React Query Pattern

```typescript
// Use in components:
import { useQuery } from '@tanstack/react-query';

function MangaDetails({ mangaId }: { mangaId: string }) {
  const { data, isLoading, error } = useQuery({
    queryKey: ['manga', mangaId],
    queryFn: () => getMangaDetails(mangaId),
    staleTime: 5 * 60 * 1000, // 5 minutes
  });

  if (isLoading) return <Skeleton />;
  if (error) return <ErrorMessage error={error} />;
  
  return <MangaCard manga={data} />;
}
```

---

## Testing Guidelines

### Go Backend Tests

#### Test Structure

```go
package package_test

import (
    "testing"
    "net/http/httptest"
)

func TestFeatureName(t *testing.T) {
    // Arrange
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Mock response
    }))
    defer server.Close()

    // Act
    result, err := client.Method()

    // Assert
    if err != nil {
        t.Fatalf("Expected no error, got %v", err)
    }
    if result != expected {
        t.Errorf("Expected %v, got %v", expected, result)
    }
}
```

#### Running Tests

```bash
# Run all tests
cd apps/api && go test ./...

# Run specific package
cd apps/api && go test ./internal/mangadex/... -v

# Run with coverage
cd apps/api && go test -cover ./...

# Run benchmarks
cd apps/api && go test -bench=. ./...
```

### Frontend Tests

#### Setup

Add to `package.json`:

```json
{
  "scripts": {
    "test": "vitest",
    "test:ui": "vitest --ui",
    "test:coverage": "vitest --coverage"
  },
  "devDependencies": {
    "@vitejs/plugin-react": "^4.0.0",
    "jsdom": "^24.0.0",
    "vitest": "^1.0.0"
  }
}
```

#### Test Example

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest';

describe('API', () => {
  beforeEach(() => {
    vi.stubEnv('NEXT_PUBLIC_API_URL', 'http://localhost:8080');
  });

  it('should fetch data', async () => {
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ success: true, data: {} }),
    });

    const result = await searchManga({ title: 'test' });
    expect(result).toBeDefined();
  });
});
```

#### Running Frontend Tests

```bash
cd apps/web
npm test           # Run tests
npm run test:ui    # Run with UI
```

---

## Code Patterns

### Go Patterns

#### Repository Pattern (Database)

```go
type Repository struct {
    db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
    return &Repository{db: db}
}

func (r *Repository) GetByID(ctx context.Context, id string) (*Entity, error) {
    var entity Entity
    err := r.db.QueryRow(ctx, "SELECT * FROM table WHERE id = $1", id).Scan(&entity)
    if err != nil {
        return nil, fmt.Errorf("failed to get entity: %w", err)
    }
    return &entity, nil
}
```

#### Scraper Pattern

```go
type Scraper struct {
    client  *http.Client
    cache   *redis.Client
}

func NewScraper() *Scraper {
    return &Scraper{
        client: &http.Client{
            Timeout: 30 * time.Second,
            CheckRedirect: func(req *Request, via []*Request) error {
                return http.ErrUseLastResponse
            },
        },
    }
}

func (s *Scraper) parseHTML(html string) (*Document, error) {
    return goquery.NewDocumentFromReader(strings.NewReader(html))
}
```

### Frontend Patterns

#### Component Pattern

```typescript
'use client';

import { useState, useEffect } from 'react';

interface Props {
  id: string;
  onComplete?: () => void;
}

export default function Component({ id, onComplete }: Props) {
  const [data, setData] = useState<Data | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchData(id).then(setData).finally(() => setLoading(false));
  }, [id]);

  if (loading) return <Skeleton />;
  
  return <div>{data?.name}</div>;
}
```

#### Hook Pattern

```typescript
export function useDeviceId() {
  const [deviceId, setDeviceId] = useState<string>('');

  useEffect(() => {
    let id = localStorage.getItem('device_id');
    if (!id) {
      id = generateUUID();
      localStorage.setItem('device_id', id);
    }
    setDeviceId(id);
  }, []);

  return deviceId;
}
```

---

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `API_PORT` | `8080` | API server port |
| `VALKEY_ADDR` | `localhost:6379` | Valkey/Redis address |
| `DATABASE_URL` | PostgreSQL connection string | Database connection |
| `NEXT_PUBLIC_API_URL` | `http://localhost:8080` | Frontend API URL |

### Rate Limiting

Rate limiting is configured in `apps/api/internal/middleware/ratelimit.go`:

```go
// Tiered limits:
- /search: 30 requests/minute
- /chapter: 60 requests/minute
- /sources: 120 requests/minute
- Default: 100 requests/minute
```

### Cache TTLs

| Endpoint | TTL |
|----------|-----|
| Manga search | 5 min |
| Manga details | 10 min |
| Chapter pages | 10 min |
| Anime search | 5 min |
| Anime details | 15 min |
| Anime episodes | 10 min |
| Streaming links | 2 min |
| Tags | 24 hours |

---

## Common Tasks

### Adding a New API Endpoint

1. **Backend**: Add handler in `apps/api/main.go`
2. **Frontend**: Add API function in `apps/web/lib/api.ts`
3. **Types**: Add types in `apps/web/lib/types.ts`
4. **Test**: Add test in appropriate test file

### Modifying an Existing Provider

1. Find the provider in `apps/api/internal/{provider}/client.go`
2. Modify the relevant method
3. Update tests to reflect changes
4. Verify cache invalidation if needed

### Adding a New Page

1. Create route in `apps/web/app/{section}/{id}/page.tsx`
2. Add API calls in `apps/web/lib/api.ts`
3. Create components in `apps/web/components/`
4. Add tests in `apps/web/__tests__/`

### Database Migrations

1. Create new file in `apps/api/migrations/`
2. Follow naming: `00X_description.up.sql` and `00X_description.down.sql`
3. Test migration locally

---

## Testing Checklist

- [ ] Unit tests for all new functionality
- [ ] Integration tests for API endpoints
- [ ] Error handling tests
- [ ] Cache behavior tests
- [ ] Rate limiting tests (manual verification)
- [ ] Frontend component tests

---

## Resources

- [Go Chi Router](https://go-chi.io/)
- [React Query](https://tanstack.com/query/)
- [MangaDex API](https://api.mangadex.org/docs/)
- [Next.js Documentation](https://nextjs.org/docs)

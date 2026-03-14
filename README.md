# Miru

A self-hosted manga and anime streaming platform with reading progress tracking.

[![Go Version](https://img.shields.io/badge/Go-1.26-blue?logo=go)](https://go.dev/)
[![Next.js](https://img.shields.io/badge/Next.js-16-black?logo=next.js)
[![React](https://img.shields.io/badge/React-19-black?logo=react)](https://react.dev)
[![Deploy API](https://github.com/4thokage/miru/actions/workflows/deploy-api.yml/badge.svg)](https://github.com/4thokage/miru/actions/workflows/deploy-api.yml)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

[![DeepWiki](https://deepwiki.ryoppippi.com/4thokage/miru)](https://deepwiki.com/4thokage/miru)

## Features

- **Manga Reading**: Browse and read manga from MangaDex
- **Anime Streaming**: Watch anime episodes via GoGoAnime
- **Progress Tracking**: Automatic reading progress saved to PostgreSQL
- **Responsive Design**: Works on desktop and mobile devices
- **Video Player**: HLS streaming support for anime

## Tech Stack

| Component | Technology |
|-----------|------------|
| Frontend | Next.js 16, React 19, Tailwind CSS 4 |
| Backend | Go (chi router) |
| Database | PostgreSQL |
| Cache | Valkey (Redis-compatible) |
| Events | NATS |
| Manga Source | MangaDex API |
| Anime Source | GoGoAnime |

## Getting Started

### Prerequisites

- Go 1.26+
- Node.js 18+
- PostgreSQL
- Valkey
- NATS

### Installation

1. Clone the repository:
```bash
git clone https://github.com/4thokage/miru.git
cd miru
```

2. Set up environment variables:
```bash
cp .env.sample .env
# Edit .env with your configuration
```

3. Start infrastructure services:
```bash
docker-compose up -d
```

4. Start the API server:
```bash
cd apps/api
go run main.go
```

5. Start the web application:
```bash
cd apps/web
npm install
npm run dev
```

The web app will be available at `http://localhost:3000` and the API at `http://localhost:8080`.

## Project Structure

```
miru/
├── apps/
│   ├── api/           # Go backend API
│   └── web/           # Next.js frontend
├── docker-compose.yml # Infrastructure services
└── .env.sample        # Environment variables template
```

## API Endpoints

### Manga
- `POST /api/v1/search` - Search manga
- `GET /api/v1/tags` - Get available tags
- `GET /api/v1/manga/:id` - Get manga details
- `POST /api/v1/chapter/:id` - Get chapter pages

### Anime
- `POST /api/v1/anime/search` - Search anime
- `GET /api/v1/anime/recent` - Get recent episodes
- `GET /api/v1/anime/popular` - Get popular anime
- `GET /api/v1/anime/:id` - Get anime details
- `GET /api/v1/anime/:id/episodes` - Get episode list
- `GET /api/v1/anime/episode/:id/sources` - Get streaming links

### Progress
- `POST /api/v1/progress` - Save reading progress

## License

MIT

# Miru

A self-hosted manga and anime streaming platform with reading progress tracking.

[![Go Version](https://img.shields.io/badge/Go-1.26-blue?logo=go)](https://go.dev/)
[![React](https://img.shields.io/badge/React-19-black?logo=react)](https://react.dev)
[![Deploy API](https://github.com/4thokage/miru/actions/workflows/deploy-api.yml/badge.svg)](https://github.com/4thokage/miru/actions/workflows/deploy-api.yml)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

[![DeepWiki](https://img.shields.io/badge/DeepWiki-4thokage%2Fmiru-blue.svg?logo=data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACwAAAAyCAYAAAAnWDnqAAAAAXNSR0IArs4c6QAAA05JREFUaEPtmUtyEzEQhtWTQyQLHNak2AB7ZnyXZMEjXMGeK/AIi+QuHrMnbChYY7MIh8g01fJoopFb0uhhEqqcbWTp06/uv1saEDv4O3n3dV60RfP947Mm9/SQc0ICFQgzfc4CYZoTPAswgSJCCUJUnAAoRHOAUOcATwbmVLWdGoH//PB8mnKqScAhsD0kYP3j/Yt5LPQe2KvcXmGvRHcDnpxfL2zOYJ1mFwrryWTz0advv1Ut4CJgf5uhDuDj5eUcAUoahrdY/56ebRWeraTjMt/00Sh3UDtjgHtQNHwcRGOC98BJEAEymycmYcWwOprTgcB6VZ5JK5TAJ+fXGLBm3FDAmn6oPPjR4rKCAoJCal2eAiQp2x0vxTPB3ALO2CRkwmDy5WohzBDwSEFKRwPbknEggCPB/imwrycgxX2NzoMCHhPkDwqYMr9tRcP5qNrMZHkVnOjRMWwLCcr8ohBVb1OMjxLwGCvjTikrsBOiA6fNyCrm8V1rP93iVPpwaE+gO0SsWmPiXB+jikdf6SizrT5qKasx5j8ABbHpFTx+vFXp9EnYQmLx02h1QTTrl6eDqxLnGjporxl3NL3agEvXdT0WmEost648sQOYAeJS9Q7bfUVoMGnjo4AZdUMQku50McDcMWcBPvr0SzbTAFDfvJqwLzgxwATnCgnp4wDl6Aa+Ax283gghmj+vj7feE2KBBRMW3FzOpLOADl0Isb5587h/U4gGvkt5v60Z1VLG8BhYjbzRwyQZemwAd6cCR5/XFWLYZRIMpX39AR0tjaGGiGzLVyhse5C9RKC6ai42ppWPKiBagOvaYk8lO7DajerabOZP46Lby5wKjw1HCRx7p9sVMOWGzb/vA1hwiWc6jm3MvQDTogQkiqIhJV0nBQBTU+3okKCFDy9WwferkHjtxib7t3xIUQtHxnIwtx4mpg26/HfwVNVDb4oI9RHmx5WGelRVlrtiw43zboCLaxv46AZeB3IlTkwouebTr1y2NjSpHz68WNFjHvupy3q8TFn3Hos2IAk4Ju5dCo8B3wP7VPr/FGaKiG+T+v+TQqIrOqMTL1VdWV1DdmcbO8KXBz6esmYWYKPwDL5b5FA1a0hwapHiom0r/cKaoqr+27/XcrS5UwSMbQAAAABJRU5ErkJggg==)](https://deepwiki.com/4thokage/miru)


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

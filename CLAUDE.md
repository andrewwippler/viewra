# ViewRA - Media Server

## Quick Context

ViewRA is a self-hosted media server for organizing and streaming personal media collections.

- **Architecture**: Clean Architecture / DDD
- **Database**: SQLite (default) or PostgreSQL
- **Streaming**: HLS with on-demand transcoding via FFmpeg
- **Plugins**: gRPC-based plugin system for metadata enrichment

## Project Structure

```text
internal/
├── domain/          # Business logic (NO external deps)
├── application/     # Use cases and services
├── infrastructure/  # DB, FFmpeg, filesystem
├── api/             # HTTP handlers
├── app/             # Dependency wiring (container, lifecycle)
├── pkg/             # Internal shared packages
└── testutil/        # Test utilities
plugins/             # gRPC enrichment plugins
├── tmdb/            # Movie/TV metadata from TMDb
├── musicbrainz/     # Music metadata from MusicBrainz
├── semantic-search/ # AI-powered semantic search
├── recommendations/ # Personalized recommendations
├── ai-features/     # AI configuration + Ollama provider
├── ai-provider-*/   # AI provider plugins (Anthropic, OpenAI, Voyage)
pkg/plugin/sdk/      # Plugin SDK for building plugins
api/proto/plugin/    # gRPC protocol definitions
tools/
├── subtitle-extractor/  # Rust tool for fast subtitle extraction
└── ffmpeg-viewra/       # Patched FFmpeg with ViewRA fixes
web/                 # React frontend (TanStack Router) — dual builds: web + TV
migrations/          # SQLite migrations
migrations/postgres/ # PostgreSQL migrations
data/                # Runtime data (db, cache, transcodes)
bin/                 # Built binaries
```

## Development Commands

**ALWAYS use make commands when available.** Run `make help` to see all targets.

```bash
# Setup & Running
make setup            # Initial setup (installs Go tools, checks Rust)
make build-tools      # Build subtitle-extractor (requires Rust)
make dev              # Start backend (8080) + frontend (5173, web UI by default)
make dev-tv           # Start backend (8080) + frontend (5173, TV UI)
make dev-debug        # Start with DEBUG logging
make test             # Run all tests

# Testing & Linting
go test -v -run TestName ./path/to/pkg  # Run single test
golangci-lint run                       # Go lint
cd web && npm run lint                  # Frontend lint

# Code Generation
make sqlc-gen         # Generate DB code after SQL changes
make swagger-gen      # Generate Swagger docs (after API changes)
make api-client-gen   # Generate TypeScript API client
make proto-gen        # Generate Go code from protobuf definitions
make taskgen          # Generate scheduler task registration code

# Plugin Development
make build-plugins                       # Build all plugins
make build-plugin NAME=semantic-search   # Build single plugin
make reload-plugin NAME=semantic-search  # Build + reload in dev server
make reload-plugins                      # Build + reload all plugins
make new-plugin NAME=myplugin            # Create plugin scaffold

# Production
make build            # Builds everything: tools, both frontends (web+TV), backend
make build-ffmpeg     # Build patched FFmpeg (optional, ~10min)
```

## Frontend: Dual UI (Web + TV)

The single Go binary embeds **two** frontend builds simultaneously, served at different paths:

| Route | Build | Target | Assets base |
|---|---|---|---|
| `GET /` | `dist/web/` | Modern browsers | `/` |
| `GET /webos/` | `dist/tv/` | WebOS TV (Chrome 79) | `/webos/` |

The TV build applies Chromium-79-specific transforms (optional chaining, nullish coalescing, `@layer` CSS). TV-specific components (TvHome, TvMovieDetail, TvShowDetail, TvSeasonDetail, spatial navigation) are gated by the build-time constant `__TV_MODE__` and tree-shaken from the web build.

**Chrome 79 constraints:** The WebOS TV browser is Chrome 79-based. All code and CSS in the TV build **must** avoid:
- `?.` (optional chaining) — use `&&` chaining
- `??` (nullish coalescing) — use `||`
- `@layer` CSS — use regular cascade
- ES2020+ features — target is `chrome79` in Vite config
- CSS `gap` in flexbox — not supported; use margins
- CSS `inset` shorthand — not supported; use `top`/`right`/`bottom`/`left` longhands
- CSS `:focus-visible` — not supported; use `:focus`
- CSS `backdrop-filter` — not supported (even prefixed); glass effects degrade to solid backgrounds

```bash
# Build both frontends
cd web && npm run build        # runs build:web + build:tv

# Individual builds
npm run build:web              # VITE_UI_MODE=web → dist/web/
npm run build:tv               # VITE_UI_MODE=tv  → dist/tv/

# Development
npm run dev                    # VITE_UI_MODE=web (standard ViewRA UI)
npm run dev:tv                 # VITE_UI_MODE=tv  (TV UI on WebOS)

# Make shortcuts
make build                     # builds both frontends + Go binary
make dev                       # web UI (default)
make dev-tv                    # TV UI
```

The Jellyfin API is always compiled and mounted regardless of UI mode. The `SystemInfo` and `SystemInfoPublic` endpoints return `WebPath: "/webos/"` so Jellyfin-compatible clients can discover the TV frontend.

### Jellyfin App Compatibility (jellyfin-webos)

The official Jellyfin WebOS app (`github.com/jellyfin/jellyfin-webos`) hardcodes two endpoints during connection:

1. **`GET /System/Info/Public`** — Must return `ProductName: "Jellyfin Server"` (see `internal/jellyfin/auth.go`). The app's auto-discovery silently rejects servers with other values.
2. **`GET /web/manifest.json`** — This is hardcoded by the app. Served at `cmd/viewra/bootstrap/bootstrap.go:71`. **DO NOT remove** — without it, the app cannot discover the web client URL. Returns `{"start_url": "/webos/"}` so the iframe loads the TV build at `/webos/`.

After these two calls, the app loads the `start_url` in an iframe. The app does NOT call `POST /Users/AuthenticateByName` or any other API — auth happens inside the iframe via the jellyfin-web UI.

## Database

SQLite database: `./data/viewra.db`

```bash
# Run migrations
make migrate-up
# Or manually:
~/go/bin/migrate -database "sqlite3://./data/viewra.db" -path ./migrations up

# Create new migration
make migrate-create NAME=add_feature
```

## API Testing

Dev credentials: `dev` / `dev`

```bash
# Helper script with auto-auth (caches token for 10 min)
./scripts/api /api/media/147440              # GET request
./scripts/api /api/media/147440/tracks       # GET tracks
./scripts/api POST /api/libraries/1/scan     # POST request
./scripts/api /health                        # Health check
```

## Code Style

### Go

- Max line 120 chars, max function 150 lines, max complexity 25
- Errors: `var ErrNotFound = errors.New("not found")`, wrap with `fmt.Errorf("context: %w", err)`
- Imports: stdlib first, then external, then local (`github.com/viewra/viewra`)
- Use `*Service` for CRUD operations, `*UseCase` for single-purpose operations

### TypeScript

- Arrow functions only, no classes (exception: extending `Error`)
- Exports at end of file
- Prettier: no semicolons, single quotes, trailing commas
- Use functional patterns with React hooks, not class instances
- **No `??` (nullish coalescing) or `?.` (optional chaining)** — TV target is Chrome 79 which lacks support. Use `||` and `&&` instead.

See [docs/development/CONVENTIONS.md](docs/development/CONVENTIONS.md) for full Go conventions.
See [web/docs/CODING_STYLE.md](web/docs/CODING_STYLE.md) for full frontend conventions.
See [docs/core/ARCHITECTURE.md](docs/core/ARCHITECTURE.md) for detailed system architecture.

## Key Rules

1. **Dual DB**: All SQL must work on SQLite AND PostgreSQL
2. **Domain purity**: Domain layer imports only stdlib
3. **Clean Architecture**: domain/ → application/ → infrastructure/ → api/
4. **DO NOT** run `make dev`, `make dev-clean`, or restart the server - the user manages the dev server
5. **DO NOT** create example files, extra docs, READMEs, or tutorial files unless explicitly requested
6. **NO** stub code, TODOs in production code, or adapter/wrapper patterns
7. **NO** backwards compatibility shims - refactor consumers directly
8. **NO** plan/phase references in code comments (no "Phase 1", ADR numbers, etc.)
9. **STOP and THINK** before implementing - fix root causes, not symptoms
10. Use `~/go/bin/air` for auto-reload instead of manual rebuilds

## Security

ViewRA includes several security hardening measures:

### Transport & Headers
- **Security headers middleware** (`internal/api/middleware/security_headers.go`): HSTS, CSP, X-Content-Type-Options, X-Frame-Options, Referrer-Policy, Permissions-Policy
- **TLS**: Not built-in — deploy behind a reverse proxy (nginx, Caddy) for HTTPS
- **CORS**: Configurable via `CORS_ALLOWED_ORIGINS` env var; streaming endpoints use `StreamingCORS` middleware that reflects the requesting origin

### Rate Limiting
- **Auth endpoints**: 5 login/min, 30 refresh/min, 3 setup/min per IP
- **Global API rate limiting**: 100 req/min per IP on protected routes (configurable via `GLOBAL_RATE_LIMIT`)
- **Plugin rate limiting**: External API clients (TMDb, MusicBrainz) have built-in rate limits

### Request Protection
- **Body size limits**: 10 MB default via `MaxBytesReader` middleware
- **Path traversal prevention**: Comprehensive validation in transcoding and path browser
- **Input validation**: Zod schemas (frontend), Gin binding tags (backend), domain-level validation
- **HTTP timeouts**: All outbound HTTP clients enforce explicit timeouts (30-120s); no `http.DefaultClient` usage

### Authentication
- **Password hashing**: Argon2id with OWASP-recommended parameters
- **JWT tokens**: HS256, 15-minute access tokens, cryptographically random refresh tokens
- **Dev mode**: Gated behind `VIEWRA_DEV_MODE=1` env var; pprof server only starts in dev mode

### Logging & Secrets
- **Query string sanitization**: Auth tokens (`api_key`, `token`) are redacted from error logs
- **Dev credentials**: Not included in production log messages or frontend builds

### Dependency Security
- **npm audit**: Run `npm run audit` in `web/` to check for vulnerable packages
- **Go modules**: Use `govulncheck ./...` to scan for known vulnerabilities

## MCP Tools

- Use `context7` to search documentation for Go, React, TypeScript, FFmpeg, HLS, etc.
- Use `gh_grep` to search GitHub for real-world code examples
- Use `playwright` for browser automation and E2E testing
- Use `filesystem` for enhanced file operations

## Build Requirements

- **Go 1.21+**: Main backend
- **Node.js 18+**: Frontend build
- **Rust/Cargo**: subtitle-extractor tool (install from <https://rustup.rs/>)
- **FFmpeg**: Runtime dependency for transcoding

## Deployment

Deploy both binaries together in the same directory:

```text
/opt/viewra/
├── viewra              # Main server
├── subtitle-extractor  # Subtitle extraction helper
├── plugins/            # Plugin binaries
└── data/               # Database and cache
```

The `viewra` binary automatically finds `subtitle-extractor` in the same directory.

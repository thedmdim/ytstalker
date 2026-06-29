# ytstalker.fun

YouTube random video discovery app with a Telegram bot.

## Binaries (Go 1.21)

| Binary | Entrypoint | Description |
|--------|------------|-------------|
| `app` | `cmd/app/main.go` | Web server, API, static files, background YouTube scraping |
| `tele` | `cmd/tele/main.go` | Telegram bot (echotron v3, long polling) |

## Build

```sh
go build -o app  ./cmd/app
go build -o tele ./cmd/tele
```

No tests, no lint, no typecheck config in repo.

## Run locally

```sh
export YT_API_KEY=... TG_TOKEN=... DSN=server.db
# app listens on :80
./app
# tele calls http://<API_URL>/api/videos/...
./tele
```

Set `LOCAL=1` to disable background YouTube search (avoids API quota burn during dev).

## Env vars

| Var | Required by | Default | Note |
|-----|-------------|---------|------|
| `YT_API_KEY` | app | — | Google YouTube Data API v3 |
| `TG_TOKEN` | tele | — | Telegram Bot token |
| `DSN` | app | `server.db` | SQLite file path |
| `API_URL` | tele | — | Internal URL for app API, e.g. `app/api` in Docker |
| `LOCAL` | app | — | Set non-empty to skip background YouTube scraping |

## Architecture

- **SQLite** via `mattn/go-sqlite3` (cgo, `database/sql` driver). Schema/migrations live inline in `cmd/app/tables.go`.
- **Routes** (`cmd/app/main.go:60-66`): API under `/api/videos/`, pages at `/stats` and `/`, static under `/static/`.
- Background goroutine searches YouTube every 30 min and stores results. Disabled when `LOCAL` is set.
- Telegram bot polls the app API internally — does **not** call YouTube directly.

## Deploy

Push to `master` triggers GitHub Actions:
- `cmd/tele/**` changed → build & push `ghcr.io/thedmdim/ytstalker/tele`, SSH+docker-compose deploy
- `cmd/app/**` or `web/**` changed → build & push `ghcr.io/thedmdim/ytstalker/app`, SSH+docker-compose deploy (app + caddy)

CI fetches `docker-compose.yml` and `Caddyfile` from raw.githubusercontent.com during deploy.

## Production infra

Three containers: `app` (web), `tele` (bot), `caddy` (reverse proxy + TLS). HAProxy config in `haproxy/` for advanced SNI routing (unused in compose yet — future move from caddy to haproxy).

## Notes
- Schema migrations in `cmd/app/tables.go` are ordered and run sequentially on startup. New migrations must be appended.
- Static files served from `web/static/`, templates from `web/pages/` and `web/partials/`.
- `server.db`, `server.db-shm`, `server.db-wal` in `.gitignore`.

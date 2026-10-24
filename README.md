# DataBin

Self-hosted pastebin (Hastebin-compatible API) on Go, Goravel and Redis.

## Features

- Web UI with Tailwind and [highlight.js](https://highlightjs.org/) syntax highlighting
- Hastebin-style HTTP API
- Pastes stored in Redis (optional TTL)
- Docker Compose for one-command deployment

## Quick start (Docker)

```bash
cp .env.example .env
# Set REDIS_PASSWORD in .env (used by app and redis service)
# Generate APP_KEY: go run . artisan key:generate
docker compose up --build -d
```

Open http://localhost:3000

## Local development

```bash
cp .env.example .env
go run . artisan key:generate
# Start Redis (e.g. docker compose up redis -d)
go run .
```

## API (Hastebin compatible)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/documents` | Create paste. Body: raw text. Response: paste id (plain text). |
| `PUT` | `/documents/{id}` | Update existing paste. |
| `DELETE` | `/documents/{id}` | Delete paste. |
| `GET` | `/raw/{id}` | Raw text. |
| `GET` | `/download/{id}` | Download as `{id}.txt`. |
| `GET` | `/{id}` | HTML view with highlighting. Optional `?lang=go`. |

### Examples

```bash
# Create
curl -s -X POST -d 'package main; func main() {}' http://localhost:3000/documents

# Raw
curl -s http://localhost:3000/raw/abcdef1234

# Update
curl -s -X PUT -d 'updated' http://localhost:3000/documents/abcdef1234

# Delete
curl -s -X DELETE http://localhost:3000/documents/abcdef1234
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_HOST` | `127.0.0.1` | Redis host |
| `REDIS_PASSWORD` | - | Redis password (`requirepass` in Docker) |
| `PASTE_TTL_HOURS` | `0` | Expiration; `0` = no expiry |
| `PASTE_MAX_BYTES` | `2097152` | Max paste size (2 MiB) |
| `PASTE_KEY_LENGTH` | `10` | Random id length |

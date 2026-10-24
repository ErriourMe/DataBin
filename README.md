# DataBin

![DataBin — self-hosted paste tool for sharing data](public/og.png)

Lightweight Self-hosted pastebin (Hastebin-compatible API) on Go and Redis.

## Features

- Web UI with Tailwind and [highlight.js](https://highlightjs.org/) syntax highlighting
- Hastebin-style HTTP API
- Pastes stored in Redis (optional TTL)
- Docker Compose for one-command deployment

## Quick start (Docker)

```bash
make env && make deploy
```

Open http://localhost:3000

## Start from local version

**Go on the host:**

```bash
make env
make local    # or: make redis && make dev
```

## API endpoints

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
| `APP_URL` | - | Public site URL for Open Graph (e.g. `https://paste.example.com`) |
| `REDIS_HOST` | `127.0.0.1` | Redis host |
| `REDIS_PASSWORD` | - | Redis password (`requirepass` in Docker) |
| `PASTE_TTL_HOURS` | `0` | Expiration; `0` = no expiry |
| `PASTE_MAX_BYTES` | `2097152` | Max paste size (2 MiB) |
| `PASTE_KEY_LENGTH` | `10` | Random id length |

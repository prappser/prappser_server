# Prappser Server

Backend server for the Prappser platform - provides authentication, application state sync, and event processing.

## Requirements

- Go 1.21+
- PostgreSQL 14+

## Quick Start

```bash
# Set required environment variables
export DATABASE_URL="postgres://user:pass@localhost:5432/prappser?sslmode=disable"
export MASTER_PASSWORD="your-secure-password"

# Run the server
go run .
```

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | - | PostgreSQL connection string |
| `MASTER_PASSWORD` | Yes | - | Master password for owner registration |
| `PORT` | No | `4545` | Server port |
| `EXTERNAL_URL` | No | `http://localhost:{PORT}` | Public URL for the server |
| `ALLOWED_ORIGINS` | No | `https://prappser.app,http://localhost:*` | CORS allowed origins (comma-separated) |
| `LOG_LEVEL` | No | `info` | Log level (debug, info, warn, error) |
| `JWT_EXPIRATION_HOURS` | No | `24` | JWT token expiration time |
| `HOSTING_PROVIDER` | No | - | Set to `zeabur` for automatic URL resolution |

## Development

```bash
# Run tests
go test ./...

# Run with live reload (using air)
air
```

## Database

The server uses PostgreSQL and automatically runs migrations on startup. Tables are created in `files/migrations/`.

## Deployment

### Docker

```bash
docker build -t prappser-server .
docker run -e DATABASE_URL="..." -e MASTER_PASSWORD="..." -p 4545:4545 prappser-server
```

### Zeabur

Set `HOSTING_PROVIDER=zeabur` and `EXTERNAL_URL` to your subdomain (e.g., `myserver` becomes `https://myserver.zeabur.app`).

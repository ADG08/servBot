# servBot

Discord bot (Go) for events and participants. Hexagonal architecture, sqlc + pgx, PostgreSQL.

## Requirements

- Go 1.24+
- PostgreSQL
- Discord bot token

## Setup

1. Copy `.env.example` to `.env` and set `TOKEN` (and optionally `DATABASE_URL`).
2. Run migrations / ensure DB schema exists (sqlc schema in `sqlc/schema/`).
3. Generate sqlc code: `sqlc generate` (see `sqlc.yaml`).

## Run

**Docker (dev)**

```bash
make build
make up
```

**Prod**

```bash
make prod-build
make prod-up
```

**Local (no Docker)**

```bash
go run ./cmd/bot
```

Ensure `DATABASE_URL` in `.env` points to your PostgreSQL instance.

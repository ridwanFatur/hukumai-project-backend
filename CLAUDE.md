# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run with hot reload
air

# Build
go build -o ./tmp/main .

# Run directly
go run main.go

# Add a dependency
go get <package> && go mod tidy
```

Server listens on `:8080` (Gin default). No test suite exists yet.

## Architecture

Flat package layout — no subdirectory nesting beyond what's listed:

- `main.go` — wires everything: DB connect → AutoMigrate → CORS → routes
- `db/db.go` — opens a single global `*gorm.DB` (`db.DB`); loads `.env` via `godotenv`
- `models/user.go` — `User` GORM model; do **not** remove or rename existing fields (ClerkID, Email, Name)
- `handlers/auth.go` — `POST /api/auth/login`
- `handlers/user.go` — `GET /api/users/me` (protected)
- `middleware/auth.go` — `AuthMiddleware()`: validates backend JWT, loads user from DB, sets `"user"` in Gin context

## Auth flow

1. Client sends Clerk JWT → `POST /api/auth/login`
2. Backend verifies with `clerk-sdk-go/v2`, fetches user from Clerk API
3. Upserts `User` row keyed on `ClerkID`
4. Returns `{user, session_token}` — HS256, 7-day expiry, signed with `SECRET_KEY`
5. Client sends `Authorization: Bearer <session_token>` on all subsequent requests

Protected routes access the authenticated user via `c.Get("user").(models.User)`.

## Adding new routes

1. Add handler function in `handlers/`
2. Register in `main.go` — public routes under `api`, protected routes under `protected` (already uses `AuthMiddleware`)
3. If adding a new model, add it to the `AutoMigrate` call in `main.go`

## Environment (`.env`)

| Key | Purpose |
|-----|---------|
| `CLERK_SECRET_KEY` | Clerk backend SDK key |
| `SECRET_KEY` | HS256 signing key for backend session JWTs |
| `CORS_ORIGINS` | Comma-separated allowed origins (default: `http://localhost:3000`) |
| `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` / `POSTGRES_HOST` / `POSTGRES_PORT` | DB connection |

## Deployment

Push to `prod` branch → GitHub Actions (`.github/workflows/deploy.yml`) → Docker image → GCP Artifact Registry → SSH into GCE VM to restart container. The `.env` on the VM is bind-mounted at runtime.

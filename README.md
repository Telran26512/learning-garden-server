# Learning Garden Server

Go backend server for **AI Learning Garden**, a multi-user AI learning community that connects math derivations, runnable code, and paper reading workflows.

This repository owns the backend API scaffold, local infrastructure, persistence, authorization, content ownership, and integration boundaries for the Synapse/Learning Garden app. Architecture, roadmap, data model, and cross-repository conventions live in the project control repository:

- [learning-garden](https://github.com/Telran26512/learning-garden)
- Frontend application: [learning-garden-web](https://github.com/Telran26512/learning-garden-web)

## Current Status

P0 is scaffolded:

- `services/api` runs the Go API with `/healthz`.
- `docker-compose.yml` starts Postgres with pgvector, Redis, and MinIO.
- `Makefile` and GitHub Actions provide backend development and CI commands.
- The landing page stays in the separate [`learning-garden-web`](https://github.com/Telran26512/learning-garden-web) repository.

## P0 Acceptance

Backend P0 is considered ready when these checks pass from this repository:

```bash
go test ./...
go run ./services/api/cmd/server
curl http://localhost:8080/healthz
docker compose up -d
docker compose ps
```

Expected `/healthz` response:

```json
{"status":"ok"}
```

The frontend acceptance item is verified from the dedicated frontend repository:

```bash
cd ../learning-garden-web
pnpm dev
```

## Responsibilities

`learning-garden-server` is responsible for:

- REST API under `/api/v1`.
- User identity, sessions, and authentication.
- Content ownership, visibility, and authorization decisions.
- Learning progress and review state.
- Social features such as comments, discussions, follows, and public profiles.
- PostgreSQL schema migrations and SQL queries.
- Object storage integration for uploaded files.
- API contracts consumed by `learning-garden-web`.

It is not responsible for:

- Rendering or developing the frontend UI.
- Serving as a CMS owned by a third-party vendor.
- Running browser-side Python code.
- Introducing microservices before the project needs them.

## Stack

| Area | Choice |
| --- | --- |
| Language | Go 1.22+ |
| Architecture | Modular monolith |
| HTTP routing | P0 uses `net/http`; chi can be introduced when routes grow |
| Database | PostgreSQL 16 with pgvector |
| Cache / queue foundation | Redis 7 |
| Object storage | S3-compatible storage; MinIO for local development |
| Tests | Go `testing` |
| Quality gates | `gofmt`, `go test ./...`, CI |

## Repository Structure

```text
learning-garden-server/
|-- .github/workflows/ci.yml
|-- docker-compose.yml
|-- Makefile
|-- services/api/
|   |-- cmd/server/       # HTTP server entry point
|   `-- internal/httpserver/
|-- internal/
|   |-- identity/         # Users, sessions, auth
|   |-- content/          # Concepts, papers, notes, markdown content
|   |-- social/           # Comments, discussions, follows
|   |-- learning/         # Progress, review cards, spaced repetition
|   |-- relation/         # Content dependency graph
|   |-- media/            # File metadata and object storage adapter
|   |-- moderation/       # Review and admin workflows
|   `-- platform/         # Config, db, middleware, logging, HTTP errors
|-- migrations/           # Versioned database migrations
|-- queries/              # SQL source files for sqlc
|-- api/                  # REST contract documentation
|-- synapse-dev-spec.html # Development spec
`-- README.md
```

## Architecture Rules

- The service stays a modular monolith until there is a concrete need to split deployment units.
- Each domain module owns its handler, service, repository, and domain types.
- Authorization decisions belong in service layer code.
- Repositories should not decide visibility or ownership rules.
- Handlers should translate HTTP requests and responses, not contain business logic.
- Domain modules must not import another module's repository or domain package.
- Cross-module access goes through explicit service interfaces.
- `platform` may be imported by domain modules; `platform` must not import domain modules.

## Environment Variables

Copy `.env.example` to `.env` for local overrides. Default P0 values are:

```bash
HTTP_ADDR=:8080
CORS_ALLOWED_ORIGINS=http://localhost:3000

POSTGRES_HOST_PORT=5432
POSTGRES_DB=learning_garden
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
DATABASE_URL=postgres://postgres:postgres@localhost:5432/learning_garden?sslmode=disable

REDIS_HOST_PORT=6380
REDIS_ADDR=localhost:6380
REDIS_URL=redis://localhost:6380/0

MINIO_API_PORT=9000
MINIO_CONSOLE_PORT=9001
S3_ENDPOINT=http://localhost:9000
S3_REGION=us-east-1
S3_BUCKET=learning-garden-local
S3_ACCESS_KEY_ID=minioadmin
S3_SECRET_ACCESS_KEY=minioadmin
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin
```

Do not commit `.env` files. Commit `.env.example` when the scaffold introduces real environment requirements.

## Local Infrastructure

`docker compose up -d` starts:

| Service | Image | Host port | Purpose |
| --- | --- | --- | --- |
| Postgres | `pgvector/pgvector:pg16` | `5432` | Primary relational database and vector extension baseline |
| Redis | `redis:7-alpine` | `6380` | Session/cache/queue foundation |
| MinIO | `minio/minio` | `9000`, `9001` | S3-compatible local object storage and console |

Redis uses host port `6380` by default to avoid collisions with a locally installed Redis on `6379`. Change `REDIS_HOST_PORT` in `.env` if needed.

## Local Development

Install Go 1.22 or newer, then start the API:

```bash
go run ./services/api/cmd/server
```

Start local dependencies:

```bash
docker compose up
```

Run checks:

```bash
go test ./...
```

Migration and sqlc commands will be finalized when the migration tool and sqlc configuration are committed. Until then, `make db:migrate` is a no-op placeholder.

## Make Targets

| Target | Command |
| --- | --- |
| `make dev` | Start the API server |
| `make api:dev` | Start the API server |
| `make db:migrate` | Migration placeholder for P0 |
| `make fmt` | Fail if Go files need formatting |
| `make test` | Run `go test ./...` |
| `make ci` | Run formatting and tests |
| `make infra:up` | Start local dependencies in detached mode |
| `make infra:down` | Stop local dependencies |

## API Endpoints

P0 exposes one health endpoint:

| Method | Path | Response |
| --- | --- | --- |
| `GET` | `/healthz` | `{"status":"ok"}` |

Application endpoints will be added under `/api/v1`.

## Frontend Development

The landing page and frontend app are not part of this backend repository. To verify the P0 landing page acceptance item, use the dedicated frontend repository:

```bash
git clone https://github.com/Telran26512/learning-garden-web
cd learning-garden-web
pnpm install
pnpm dev
```

## API Contract

The backend exposes REST endpoints under:

```text
/api/v1
```

Contract documentation should live in `api/`, either as OpenAPI or clear endpoint documentation. The frontend must consume the server only through this contract.

Breaking API changes should include:

- Updated contract documentation.
- Migration notes for frontend changes.
- Tests covering the changed behavior.

## Testing Priorities

High-value backend tests:

- Auth and session behavior.
- Owner-only write authorization.
- Private content visibility rules.
- Public content discovery rules.
- Learning review and spaced repetition logic.
- Repository integration tests against PostgreSQL.
- Migration validity checks.

## Git Conventions

Use Conventional Commits:

```text
feat(identity): add session login
fix(content): prevent private content leakage
test(learning): cover review scheduling
chore(repo): initialize go scaffold
```

Use short feature branches for larger work:

```text
feat/identity-session
feat/content-crud
feat/postgres-migrations
```

## CI

GitHub Actions runs on pushes to `main` and pull requests:

- `gofmt` check for `services/api`.
- `go test ./...`.

Frontend typecheck/lint/build checks belong to `learning-garden-web`.

## Roadmap Entry Point

After P0, backend work continues with:

- Add config loading and structured error handling.
- Add PostgreSQL connection wiring.
- Add migration tooling.
- Add identity and authorization foundations.
- Add CI checks for formatting, linting, tests, migrations, and module boundaries.

See [learning-garden docs](https://github.com/Telran26512/learning-garden/tree/main/docs) for the full roadmap and architecture.

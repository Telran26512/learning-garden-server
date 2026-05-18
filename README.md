# Learning Garden Server

Go backend server for **AI Learning Garden**, a multi-user AI learning community that connects math derivations, runnable code, and paper reading workflows.

This repository owns the backend API, persistence, authorization, content ownership, and integration boundaries. Architecture, roadmap, data model, and cross-repository conventions live in the project control repository:

- [learning-garden](https://github.com/Telran26512/learning-garden)
- Frontend application: [learning-garden-web](https://github.com/Telran26512/learning-garden-web)

## Current Status

This repository is initialized for the backend service. The Go service scaffold will be added during M0 after the project docs are reviewed.

Until the scaffold lands, this README defines the intended repository boundary, runtime expectations, and development conventions.

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

- Rendering the frontend UI.
- Serving as a CMS owned by a third-party vendor.
- Running browser-side Python code.
- Introducing microservices before the project needs them.

## Planned Stack

| Area | Choice |
| --- | --- |
| Language | Go |
| Architecture | Modular monolith |
| HTTP routing | chi on top of `net/http` |
| Database | PostgreSQL |
| Database access | sqlc with pgx |
| Migrations | goose or golang-migrate |
| Authentication | Session cookies, argon2id password hashing |
| Object storage | S3-compatible provider; MinIO for local development |
| Tests | Go `testing`, service unit tests, repository integration tests |
| Quality gates | gofmt, golangci-lint, depguard, migration checks |

## Planned Repository Structure

```text
learning-garden-server/
|-- cmd/server/           # HTTP server entry point
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

Expected local variables after the scaffold is added:

```bash
HTTP_ADDR=:8080
DATABASE_URL=postgres://postgres:postgres@localhost:5432/learning_garden?sslmode=disable
SESSION_SECRET=change-me-in-local-dev
CORS_ALLOWED_ORIGINS=http://localhost:3000

S3_ENDPOINT=http://localhost:9000
S3_REGION=us-east-1
S3_BUCKET=learning-garden-local
S3_ACCESS_KEY_ID=minioadmin
S3_SECRET_ACCESS_KEY=minioadmin
```

Do not commit `.env` files. Commit `.env.example` when the scaffold introduces real environment requirements.

## Local Development

Expected commands after the Go scaffold is added:

```bash
go mod download
go run ./cmd/server
go test ./...
golangci-lint run
```

Migration and sqlc commands will be finalized when the migration tool and sqlc configuration are committed.

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

## Roadmap Entry Point

Backend work starts with M0:

- Initialize Go module and HTTP server.
- Add config loading and structured error handling.
- Add PostgreSQL connection wiring.
- Add migration tooling.
- Add identity and authorization foundations.
- Add CI checks for formatting, linting, tests, migrations, and module boundaries.

See [learning-garden docs](https://github.com/Telran26512/learning-garden/tree/main/docs) for the full roadmap and architecture.

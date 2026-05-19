.PHONY: dev api\:dev db\:migrate ci fmt test infra\:up infra\:down

dev:
	go run ./services/api/cmd/server

api\:dev:
	go run ./services/api/cmd/server

db\:migrate:
	@echo "No migrations to run yet"

ci: fmt test

fmt:
	@test -z "$$(gofmt -l services/api)"

test:
	go test ./...

infra\:up:
	docker compose up -d

infra\:down:
	docker compose down

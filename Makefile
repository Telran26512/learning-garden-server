.PHONY: dev api\:dev db\:migrate ci fmt lint test infra\:up infra\:down

GOLANGCI_LINT_VERSION ?= v2.11.3

dev:
	go run ./services/api/cmd/server

api\:dev:
	go run ./services/api/cmd/server

db\:migrate:
	cd services/api && go run ./cmd/migrate

ci: fmt test

fmt:
	@test -z "$$(gofmt -l services/api)"

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run

test:
	go test ./...

infra\:up:
	docker compose up -d

infra\:down:
	docker compose down

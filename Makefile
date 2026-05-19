.PHONY: dev api\:dev db\:migrate ci fmt test infra\:up infra\:down

dev:
	go run ./services/api/cmd/server

api\:dev:
	go run ./services/api/cmd/server

db\:migrate:
	cd services/api && go run ./cmd/migrate

ci: fmt test

fmt:
	@test -z "$$(gofmt -l services/api)"

test:
	go test ./...

infra\:up:
	docker compose up -d

infra\:down:
	docker compose down

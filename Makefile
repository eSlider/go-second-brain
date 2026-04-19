.PHONY: docs kg-up bot ingest test-integration lint fmt ci

docs:
	docker compose up -d docs

kg-up:
	docker compose --profile kg up -d neo4j qdrant

ingest:
	docker compose --profile kg run --rm kg-ingestor

bot:
	docker compose --profile bot up -d matrix-bot

fmt:
	cd services && gofmt -w .

lint:
	cd services && golangci-lint run ./...

test-integration:
	cd services && go test -tags=integration -count=1 -timeout=30m ./integration/...

ci: fmt lint
	cd services && go test ./...

.PHONY: docs kg-up bot elastic-up elastic-setup ingest test-integration test-rag-e2e lint fmt ci

docs:
	docker compose up -d docs

kg-up:
	docker compose --profile kg up -d neo4j qdrant

ingest:
	docker compose --profile kg run --rm kg-ingestor

bot:
	docker compose --profile bot up -d matrix-bot

elastic-up:
	docker compose --profile elastic up -d elasticsearch kibana filebeat

# Provision Kibana data view + Edelweiss dashboard via Kibana API.
# Override defaults: KIBANA_URL, KIBANA_AUTH=user:password.
elastic-setup:
	KIBANA_URL=$${KIBANA_URL:-http://127.0.0.1:5601} ./deploy/elastic/setup-kibana.sh

fmt:
	cd services && gofmt -w .

lint:
	cd services && golangci-lint run ./...

test-integration:
	cd services && go test -tags=integration -count=1 -timeout=30m ./integration/...

# Full RAG + multi-turn dialogue test (Ollama + testcontainers; no Synapse). Long-running.
test-rag-e2e:
	cd services && RUN_RAG_FUNCTIONAL=1 go test -tags=integration -count=1 -timeout=45m -v ./integration/... -run TestRAGFunctional

ci: fmt lint
	cd services && go test ./...

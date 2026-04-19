# Convenience: run `make <target>` from services/ — delegates to repo root Makefile.
.PHONY: test-integration test-rag-e2e ci lint fmt
test-integration:
	@$(MAKE) -C .. test-integration
test-rag-e2e:
	@$(MAKE) -C .. test-rag-e2e
ci:
	@$(MAKE) -C .. ci
lint:
	@$(MAKE) -C .. lint
fmt:
	@$(MAKE) -C .. fmt

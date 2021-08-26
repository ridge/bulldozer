check: check-go-mod check-vet check-test

STABLE_GO_VERSION=1.17.x

check-go-mod:
	@echo CHECK GO.MOD/GO.SUM
	@if [ "x$$TRAVIS_GO_VERSION" != "x$(STABLE_GO_VERSION)" ]; then \
		echo "Skipping, not a current stable version"; \
	else \
		go mod tidy; \
		if [ -n "$$(git status --porcelain)" ]; then \
			git status -v; \
			git diff; \
			exit 1; \
		fi; \
	fi

check-vet:
	@echo GO VET
	@go vet ./...

check-test:
	@echo GO TEST
	@go test -race ./...

check: check-go-mod check-vet check-test

check-go-mod:
	@echo CHECK GO.MOD/GO.SUM
	@if [ "x$$TRAVIS_GO_VERSION" != x1.15.x ]; then \
		echo "Skipping, not a current stable version"; \
	else \
		go mod tidy; \
		if [ -n "$$(git status --porcelain)" ]; then \
			git status -v; \
			exit 1; \
		fi; \
	fi

check-vet:
	@echo GO VET
	@go vet ./...

check-test:
	@echo GO TEST
	@go test -race ./...

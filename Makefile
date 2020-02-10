check: check-go-mod check-vet check-test

check-go-mod:
	@echo CHECK GO.MOD/GO.SUM
	@go mod tidy
	@if [ -n "$$(git status --porcelain)" ]; then \
		git status -v; \
		exit 1; \
	fi

check-vet:
	@echo GO VET
	@go vet ./...

check-test:
	@echo GO TEST
	@go test -race ./...

# lint: Run golangci-lint for project
.PHONY: lint
lint:
	golangci-lint run

## test: Run all tests in project
test:
	go test -v -race -cover -bench=. ./...

## get: Run go get missing dependencies
get:
	go get ./...

.PHONY: help
all: help
help: Makefile
	@echo
	@echo " Choose a command"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo

.DEFAULT_GOAL := help

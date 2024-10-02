GO := go

.PHONY: all
all: test lint 

.PHONY: test
test:
	$(GO) test -v ./... -skip ^Example

lint: lint-golangci

.PHONY: lint-golangci
lint-golangci:
	$(GO) run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.56.2 run ./...

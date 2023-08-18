GO := go

.PHONY: all
all: test lint 

.PHONY: test
test:
	$(GO) test -v ./...

.PHONY: lint
lint:
	$(GO) run github.com/golangci/golangci-lint/cmd/golangci-lint run ./...

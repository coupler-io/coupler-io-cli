BINARY_NAME=coupler-io
VERSION ?= dev

.PHONY: build install test lint fmt clean

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY_NAME) .

# `go install` names the binary after the module (coupler-io-cli), so build it
# under the command name instead.
install:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o "$(or $(shell go env GOBIN),$(shell go env GOPATH)/bin)/$(BINARY_NAME)" .

test:
	go test -race ./...

lint:
	@test -z "$$(gofmt -l .)" || (echo "gofmt needed on:"; gofmt -l .; exit 1)
	go vet ./...

fmt:
	gofmt -w .

clean:
	rm -f $(BINARY_NAME)
	rm -rf dist

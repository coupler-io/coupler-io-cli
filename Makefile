BINARY_NAME=coupler
VERSION ?= dev

.PHONY: build install test lint fmt clean

build:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BINARY_NAME) .

install:
	go install .

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

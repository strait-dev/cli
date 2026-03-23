.PHONY: build test lint vet fmt clean install hooks

BINARY := strait
VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/strait

install:
	go install $(LDFLAGS) ./cmd/strait

test:
	go test -race -count=1 ./...

test-short:
	go test -short -count=1 ./...

lint:
	golangci-lint run ./...

vet:
	go vet ./...

fmt:
	gofmt -w .
	goimports -w .

clean:
	rm -rf bin/ dist/

tidy:
	go mod tidy

check: vet lint test

hooks:
	lefthook install

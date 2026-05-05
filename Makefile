.PHONY: build test test-short lint vet fmt clean install hooks mutation mutation-dry

BINARY := strait
VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"
GREMLINS_VERSION ?= v0.6.0
GREMLINS := go run github.com/go-gremlins/gremlins/cmd/gremlins@$(GREMLINS_VERSION)
MUTATION_REPORT ?= bin/gremlins-report.json
MUTATION_ARGS ?=

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/strait

install:
	go install $(LDFLAGS) ./cmd/strait

test:
	go test -race -count=1 ./...

test-short:
	go test -short -count=1 ./...

mutation:
	mkdir -p $(dir $(MUTATION_REPORT))
	$(GREMLINS) unleash --output $(MUTATION_REPORT) $(MUTATION_ARGS)

mutation-dry:
	mkdir -p $(dir $(MUTATION_REPORT))
	$(GREMLINS) unleash --dry-run --output $(MUTATION_REPORT) $(MUTATION_ARGS)

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

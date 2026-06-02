.PHONY: build test test-short lint vet fmt clean install hooks mutation mutation-dry refresh-openapi e2e

STRAIT_SERVER ?= http://localhost:8080
OPENAPI_SPEC := internal/client/testdata/openapi.json

BINARY := strait
VERSION ?= 0.2.0-dev
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

# refresh-openapi pulls the server's published OpenAPI spec into testdata so the
# contract test (internal/client TestEndpointsMatchOpenAPISpec) validates CLI
# paths against the live API surface. Point at a running server via STRAIT_SERVER.
refresh-openapi:
	curl -fsS "$(STRAIT_SERVER)/reference/openapi.json" -o "$(OPENAPI_SPEC)"
	@echo "updated $(OPENAPI_SPEC) from $(STRAIT_SERVER)"

# e2e runs the live end-to-end suite against a running server. Requires
# STRAIT_SERVER, STRAIT_API_KEY, and STRAIT_PROJECT to be set.
e2e:
	go test -tags=e2e -count=1 ./cmd/strait/... -run TestE2E

hooks:
	lefthook install

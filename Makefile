GO ?= go
GOLANGCI_LINT ?= golangci-lint
GOLANGCI_LINT_VERSION ?= v1.45.0

COMMIT ?= "$(shell git describe --long --dirty --always --match "" || true)"
LDFLAGS ?= -s -w -X github.com/ethersphere/beekeeper.commit=$(COMMIT)

.PHONY: all
all: build lint vet test-race binary

.PHONY: binary
binary: export CGO_ENABLED=0
binary: dist FORCE
	$(GO) version
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o dist/beekeeper ./cmd/beekeeper

dist:
	mkdir $@

.PHONY: lint
lint: linter
	$(GOLANGCI_LINT) run

.PHONY: linter
linter:
	which $(GOLANGCI_LINT) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$($(GO) env GOPATH)/bin $(GOLANGCI_LINT_VERSION)

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: test-race
test-race:
	$(GO) test -race -v ./...

.PHONY: test
test:
	$(GO) test -v ./...

.PHONY: build
build: export CGO_ENABLED=0
build:
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" ./...

.PHONY: clean
clean:
	$(GO) clean
	rm -rf dist/

FORCE:

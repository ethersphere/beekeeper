GO ?= go
GOLANGCI_LINT ?= golangci-lint
GOLANGCI_LINT_VERSION ?= v1.61.0
BEEKEEPER_IMAGE ?= ethersphere/beekeeper:latest
COMMIT ?= "$(shell git describe --long --dirty --always --match "" || true)"
LDFLAGS ?= -s -w -X github.com/ethersphere/beekeeper.commit=$(COMMIT)

.PHONY: all
all: build lint vet test-race binary

.PHONY: binary
binary: export CGO_ENABLED=0
binary: dist FORCE
	$(GO) version
ifeq ($(OS),Windows_NT)
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o dist/beekeeper.exe ./cmd/beekeeper
else
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o dist/beekeeper ./cmd/beekeeper
endif

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
	$(GO) test -v ./pkg/...

.PHONY: build
build: export CGO_ENABLED=0
build:
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" ./...

.PHONY: clean
clean:
	$(GO) clean
	rm -rf dist/

.PHONY: docker-build
docker-build: binary
	@echo "Build flags: $(LDFLAGS)"
	mkdir -p ./tmp
	cp ./dist/beekeeper ./tmp/beekeeper
	docker build -f Dockerfile.dev -t $(BEEKEEPER_IMAGE) .
	rm -rf ./tmp
	@echo "Docker image: $(BEEKEEPER_IMAGE)"

FORCE:

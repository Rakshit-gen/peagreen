BINARY  := peagreen
CMD     := ./cmd/peagreen
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
GO      ?= go

.DEFAULT_GOAL := build

.PHONY: build
build: ## Build peagreen into ./peagreen
	$(GO) build -trimpath -ldflags '$(LDFLAGS)' -o $(BINARY) $(CMD)

.PHONY: install
install: ## Install peagreen into $(go env GOPATH)/bin
	$(GO) install -trimpath -ldflags '$(LDFLAGS)' $(CMD)

.PHONY: roms
roms: ## Download the test ROMs into testdata/roms
	sh scripts/testroms.sh

.PHONY: test
test: roms ## Run the tests, test ROMs included
	$(GO) test ./...

.PHONY: bench
bench: roms ## Time one emulated frame
	$(GO) test -run XXX -bench . -benchmem ./gb

.PHONY: check
check: ## gofmt, vet and a tidy go.mod
	test -z "$$(gofmt -l .)" || (gofmt -l . && exit 1)
	$(GO) vet ./...
	$(GO) mod tidy -diff

.PHONY: clean
clean: ## Remove the binary and dist/
	rm -rf $(BINARY) dist

.PHONY: help
help: ## List the targets
	@grep -E '^[a-z-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-8s %s\n", $$1, $$2}'

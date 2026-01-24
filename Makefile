.PHONY: build lint test test-integration clean install

VERSION ?= dev
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X github.com/kareemaly/cortex/pkg/version.Version=$(VERSION) \
           -X github.com/kareemaly/cortex/pkg/version.Commit=$(COMMIT) \
           -X github.com/kareemaly/cortex/pkg/version.BuildDate=$(DATE)

build:
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/cortex ./cmd/cortex
	go build -ldflags "$(LDFLAGS)" -o bin/cortexd ./cmd/cortexd

lint:
	golangci-lint run

test:
	go test -v ./...

test-integration:
	go test -tags=integration -v ./internal/daemon/mcp/... ./internal/daemon/api/...

clean:
	rm -rf bin

install: clean build
	@echo "Installing to ~/.local/bin/..."
	@mkdir -p ~/.local/bin
	@rm -f ~/.local/bin/cortex ~/.local/bin/cortexd
	@cp bin/cortex bin/cortexd ~/.local/bin/
	@if [ "$$(uname)" = "Darwin" ]; then \
		echo "Code signing (macOS)..."; \
		codesign --force --sign - ~/.local/bin/cortex ~/.local/bin/cortexd; \
	fi
	@echo ""
	@echo "Validating installation..."
	@~/.local/bin/cortex version
	@echo ""
	@~/.local/bin/cortexd version
	@echo ""
	@echo "Installation complete."

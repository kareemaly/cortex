.PHONY: build lint test test-integration clean install release-build setup-hooks

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

# Cross-compile for all platforms (used for releases)
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64
RELEASE_LDFLAGS := -s -w $(LDFLAGS)

release-build:
	@rm -rf dist
	@mkdir -p dist
	@echo "Building release binaries..."
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		echo "  Building cortex-$$os-$$arch..."; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -ldflags "$(RELEASE_LDFLAGS)" -o dist/cortex-$$os-$$arch ./cmd/cortex; \
		echo "  Building cortexd-$$os-$$arch..."; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -ldflags "$(RELEASE_LDFLAGS)" -o dist/cortexd-$$os-$$arch ./cmd/cortexd; \
	done
	@echo "Generating checksums..."
	@cd dist && shasum -a 256 cortex-* cortexd-* > checksums.txt
	@echo ""
	@echo "Release artifacts:"
	@ls -la dist/
	@echo ""
	@cat dist/checksums.txt

setup-hooks:
	@echo "Configuring git hooks..."
	git config core.hooksPath .githooks
	@echo "Done. Pre-push hook is now active."

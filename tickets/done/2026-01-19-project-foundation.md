# Project Foundation

Set up the Go project structure, build system, and code quality tooling for cortex1.

## Context

This is a fresh Go project. Reference `~/projects/cortex` (cortex0) for patterns but adapt for v1 architecture per `DESIGN.md`.

## Requirements

### 1. Go Module

Create `go.mod` with:
- Module path: `github.com/kareemaly/cortex1`
- Go version: 1.24

Initial dependencies (add to go.mod, run `go mod tidy`):
```
github.com/modelcontextprotocol/go-sdk  # MCP server
github.com/go-chi/chi/v5                # HTTP router
github.com/spf13/cobra                  # CLI framework
github.com/charmbracelet/bubbletea      # TUI framework
github.com/charmbracelet/lipgloss       # TUI styling
github.com/google/uuid                  # UUIDs
gopkg.in/yaml.v3                        # Config parsing
gopkg.in/natefinch/lumberjack.v2        # Log rotation
```

### 2. Directory Structure

```
cortex1/
├── cmd/
│   ├── cortex/           # CLI binary
│   │   └── main.go       # Minimal: prints "cortex v0.0.0-dev"
│   └── cortexd/          # Daemon binary
│       └── main.go       # Minimal: prints "cortexd v0.0.0-dev"
├── internal/
│   ├── cli/              # CLI internals (empty, create .gitkeep)
│   │   ├── sdk/          # HTTP client for daemon
│   │   └── tui/          # Bubbletea views
│   └── daemon/           # Daemon internals (empty, create .gitkeep)
│       ├── api/          # HTTP server
│       ├── mcp/          # MCP server
│       └── config/       # Configuration
├── pkg/
│   └── version/
│       └── version.go    # Version info (ldflags)
├── .golangci.yml
├── lefthook.yml
├── .editorconfig
├── CLAUDE.md
├── Makefile
└── go.mod
```

### 3. pkg/version/version.go

```go
package version

import (
	"fmt"
	"runtime"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

type Info struct {
	Version   string
	Commit    string
	BuildDate string
	GoVersion string
	Platform  string
}

func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

func String() string {
	return fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, BuildDate)
}
```

### 4. Makefile

```makefile
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -ldflags "-X github.com/kareemaly/cortex1/pkg/version.Version=$(VERSION) \
                     -X github.com/kareemaly/cortex1/pkg/version.Commit=$(COMMIT) \
                     -X github.com/kareemaly/cortex1/pkg/version.BuildDate=$(DATE)"

.PHONY: all build cortex cortexd install lint test clean

all: build

build: cortex cortexd

cortex:
	go build $(LDFLAGS) -o bin/cortex ./cmd/cortex

cortexd:
	go build $(LDFLAGS) -o bin/cortexd ./cmd/cortexd

install: build
	cp bin/cortex $(GOPATH)/bin/
	cp bin/cortexd $(GOPATH)/bin/

lint:
	golangci-lint run

test:
	go test -v ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
```

### 5. .golangci.yml

```yaml
version: "2"
run:
  timeout: 5m
```

### 6. lefthook.yml

```yaml
pre-commit:
  parallel: true
  commands:
    lint:
      run: golangci-lint run
    build:
      run: go build ./...
```

### 7. .editorconfig

```ini
root = true

[*]
end_of_line = lf
insert_final_newline = true
charset = utf-8
indent_style = tab
indent_size = 4

[*.{yaml,yml,json,md}]
indent_style = space
indent_size = 2

[Makefile]
indent_style = tab
```

### 8. CLAUDE.md

```markdown
# Cortex1

Orchestration layer for AI coding workflows. File-based ticket management with MCP tools for agent interaction.

## Quick Reference

- **Build**: `make build` (outputs to `bin/`)
- **Lint**: `make lint`
- **Test**: `make test`

## Architecture

See `DESIGN.md` for full architecture. Key points:

- `cortexd`: Daemon with HTTP API + MCP server
- `cortex`: CLI + TUI for humans
- Tickets are JSON files in `.cortex/tickets/{backlog,progress,done}/`
- Agents interact via MCP tools, not REST API

## Code Style

- Use `slog` for structured logging
- Chi for HTTP routing
- Cobra for CLI commands
- Bubbletea for TUI

## Directory Layout

- `cmd/cortex/` - CLI entry point
- `cmd/cortexd/` - Daemon entry point
- `internal/cli/` - CLI internals (sdk, tui)
- `internal/daemon/` - Daemon internals (api, mcp, config)
- `pkg/version/` - Version info
```

### 9. cmd/cortex/main.go

Minimal main that prints version:

```go
package main

import (
	"fmt"

	"github.com/kareemaly/cortex1/pkg/version"
)

func main() {
	fmt.Printf("cortex %s\n", version.String())
}
```

### 10. cmd/cortexd/main.go

Minimal main that prints version:

```go
package main

import (
	"fmt"

	"github.com/kareemaly/cortex1/pkg/version"
)

func main() {
	fmt.Printf("cortexd %s\n", version.String())
}
```

## Verification

After implementation:

```bash
# Build succeeds
make build

# Binaries exist and print version
./bin/cortex
./bin/cortexd

# Lint passes
make lint

# Tests pass (trivially, no tests yet)
make test
```

## Notes

- Don't add business logic yet - this is pure scaffolding
- Keep .gitkeep files in empty directories so git tracks them
- The cmd/*/main.go files are intentionally minimal

## Implementation

### Commits

- `d32ec21` feat: add initial Go project scaffolding

### Key Files Changed

- `go.mod` - Module init (Go 1.24)
- `pkg/version/version.go` - Version with ldflags support
- `cmd/cortex/main.go` - CLI entry point
- `cmd/cortexd/main.go` - Daemon entry point
- `Makefile` - Build, lint, test, clean targets
- `.golangci.yml` - Linter config (v2 format)
- `lefthook.yml` - Pre-commit hooks
- `.editorconfig` - Editor settings
- `.gitignore` - Excludes bin/
- `CLAUDE.md` - Developer reference
- `internal/cli/{sdk,tui}/.gitkeep` - Placeholder directories
- `internal/daemon/{api,mcp,config}/.gitkeep` - Placeholder directories

### Decisions Made

1. **Simplified version.go** - Used a simpler `String(name string)` function instead of the full `Info` struct. The struct can be added later when needed for JSON output.

2. **Deferred dependencies** - Did not add the dependencies listed in requirements (cobra, chi, bubbletea, etc.) since they're not used yet. Will add when implementing actual functionality to avoid unused imports.

3. **golangci-lint v2 format** - Used `formatters:` section for gofmt/goimports as required by v2, rather than listing them under `linters.enable`.

4. **Added .gitignore** - Not in original spec but necessary to exclude `bin/` from version control.

### Scope Changes

- Dependencies deferred until actually needed (cleaner go.mod)
- Simplified version package (Info struct can be added later)

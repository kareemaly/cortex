# Cortex1 Developer Reference

## Build Commands

```bash
make build    # Build bin/cortex and bin/cortexd
make lint     # Run golangci-lint
make test     # Run tests
make clean    # Remove bin/
```

## Project Structure

- `cmd/cortex/` - CLI entry point
- `cmd/cortexd/` - Daemon entry point
- `pkg/version/` - Version info (set via ldflags)
- `internal/cli/sdk/` - CLI SDK components
- `internal/cli/tui/` - Terminal UI components
- `internal/daemon/api/` - Daemon API handlers
- `internal/daemon/mcp/` - MCP protocol implementation
- `internal/daemon/config/` - Configuration management

## Version Info

Version is injected at build time via ldflags. See `pkg/version/version.go`.

# Add --port flag to cortexd serve

Allow configuring the HTTP port for `cortexd serve` to avoid conflicts and enable testing.

## Current State

Port is hardcoded to 4200 in `cmd/cortexd/commands/serve.go`.

## Requirements

1. Add `--port` flag to `cortexd serve` command
2. Default to 4200 (current behavior)
3. Example: `cortexd serve --port 4201`

## Verification

```bash
make build

# Test custom port
cortexd serve --port 4201 &
curl http://localhost:4201/health
# Should return {"status":"ok","version":"..."}
```

## Implementation

### Commits Pushed

- `d3f297d` feat: add --port flag to cortexd serve command

### Key Files Changed

- `cmd/cortexd/commands/serve.go` - Added `--port` flag with default 4200, override logic when flag is set

### Decisions

- Used `cmd.Flags().Changed("port")` to only override config when flag is explicitly set, preserving config file values when flag is not used

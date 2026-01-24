# Fix Install Command Defaults

## Problem

1. `cortex install` requires `--project` flag to install in current directory - should default to current project
2. Global install creates unnecessary ticket directories in `~/.cortex/` - only `~/.cortex/settings.yaml` should be global

## Expected Behavior

- `cortex install` (no flags) should install in current working directory
- Global setup should only create `~/.cortex/settings.yaml`
- Remove global tickets structure (`~/.cortex/tickets/`)

## Verification

```bash
make lint
make test
make build
make test-integration
```

## Implementation

### Commits Pushed
- `ac3add3` fix: change install to default to current directory, add --global-only flag

### Key Files Changed
- `cmd/cortex/commands/install.go` - Changed flag from `--project/-p` to `--global-only/-g`, inverted logic so default installs in current directory
- `internal/install/install.go` - Removed ticket directory creation from `setupGlobal()`, now only creates `~/.cortex/` and `settings.yaml`

### Important Decisions
- Used `--global-only` flag name (with `-g` shorthand) as the opt-out mechanism rather than keeping `--project` as opt-in
- Kept the `--force` flag unchanged for overwriting existing config files

### Scope Changes
- None - implementation matches original ticket requirements

---
id: b4a34024-f4b4-43a8-85b6-58e2ba61902b
author: claude
type: review_requested
created: 2026-02-02T11:44:08.885701Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Summary

Added `cortex config show` CLI command that displays the fully resolved project configuration after all extends are merged.

## Changes

### New Files
- `cmd/cortex/commands/config.go` - Parent `config` command
- `cmd/cortex/commands/config_show.go` - `config show` subcommand implementation

### Features Implemented
1. **`cortex config show`** - Displays resolved config for current project
2. **`cortex config show --path /some/project`** - Displays resolved config for specified project
3. **Comment header** showing project root and extend chain (when applicable)
4. **Valid YAML output** that could be copy-pasted into a config file
5. **Helpful error messages** for common issues:
   - No cortex project found (suggests running `cortex init`)
   - Extend path not found
   - Circular extend reference
   - Config parse errors

### Example Output
```yaml
# Resolved config for: /Users/user/projects/myapp
# Extended from: ~/.cortex/defaults/claude-code

name: myapp
architect:
    agent: claude
    args:
        - --verbose
ticket:
    work:
        agent: claude
git:
    worktrees: false
```

## Testing
- Build passes: `make build`
- Lint passes: `make lint` (0 issues)
- All existing tests pass: `go test ./...`
- Manual testing verified:
  - Works in project directory
  - Works with `--path` flag
  - Proper error handling for missing projects
  - Comment header shows extend path when configured

## Note
During implementation, I discovered an existing issue: the extend functionality expects base configs at `path/.cortex/cortex.yaml`, but the installed defaults at `~/.cortex/defaults/claude-code/` have `cortex.yaml` directly (no `.cortex` subdirectory). This causes extends to fall back to defaults. This is a pre-existing bug in the config loading, not introduced by this change. The `config show` command correctly displays whatever the config loader returns.
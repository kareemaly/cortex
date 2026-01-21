# Cortex Install Command

Implement the `cortex install` command for initial setup.

## Context

The install command bootstraps the cortex environment - creating directories, config files, and optionally setting up the current project.

See `DESIGN.md` for:
- Global directory structure (lines 20-22)
- Project directory structure (lines 24-29)
- Global config (lines 32-40)
- Project config (lines 272-293)
- CLI command (line 52)

## Requirements

Implement `cortex install` command that:

1. **Global Setup**
   - Create `~/.cortex/` directory if not exists
   - Create `~/.cortex/settings.yaml` with defaults if not exists
   - Ensure `~/.cortex/tickets/` directories exist (backlog, progress, done)

2. **Project Setup** (optional, with flag or prompt)
   - Create `.cortex/` in current directory
   - Create `.cortex/cortex.yaml` with sensible defaults
   - Create `.cortex/tickets/{backlog,progress,done}/` directories
   - Auto-detect project name from directory or git remote

3. **Verification**
   - Check if tmux is installed (warn if not)
   - Check if claude CLI is installed (warn if not)
   - Check if git is installed (warn if not)

4. **Idempotent**
   - Safe to run multiple times
   - Don't overwrite existing config files
   - Report what was created vs already existed

## Verification

```bash
make build
make lint
make test

# Manual test
rm -rf ~/.cortex  # Clean slate
cortex install    # Should create global setup
cd /some/project
cortex install --project  # Should create project setup
```

## Notes

- Keep it simple - no fancy wizard needed
- Use sensible defaults that work out of box
- Consider --force flag to overwrite existing configs
- Print clear messages about what was created

## Implementation

### Commits Pushed

- `e185266` feat: implement cortex install command

### Key Files Changed

**New files:**
- `internal/install/result.go` - Result types (`ItemStatus`, `SetupItem`, `DependencyResult`, `Result`)
- `internal/install/deps.go` - Dependency checking via `exec.LookPath` for tmux, claude, git
- `internal/install/detect.go` - Project name detection from git remote origin or directory name
- `internal/install/install.go` - Core installation logic with `Run()`, `setupGlobal()`, `setupProject()`

**Modified files:**
- `cmd/cortex/commands/install.go` - Wired up command with `--project` and `--force` flags
- `internal/daemon/config/config.go` - Added `StatusHistoryLimit` and `GitDiffTool` fields per DESIGN.md

### Important Decisions

1. **Separate package** - Core logic in `internal/install/` for testability and reusability
2. **No prompts** - Pure flag-based interface (`--project`, `--force`) for simplicity and scriptability
3. **Git detection** - Extract repo name from `git remote get-url origin`, fallback to directory name
4. **Idempotent** - Uses `os.Stat` to check existence before creation; directories show "already exists", configs only overwritten with `--force`

### Scope Changes

None - implemented as specified in the ticket

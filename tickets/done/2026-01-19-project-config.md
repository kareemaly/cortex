# Project Config

Implement project-level configuration loading from `.cortex/cortex.yaml`.

## Context

Each project has a `.cortex/cortex.yaml` file that defines project settings, git repos, and lifecycle hooks.

See `DESIGN.md` for:
- Project config schema (lines 272-293)
- Git repos config (lines 279-282)
- Lifecycle hooks config (lines 284-293)

## Requirements

Create or extend config package to load project config:

1. **Config Structure**
   ```yaml
   name: project-name
   agent: claude  # or other agent types

   git:
     repos:
       - path: "."
       - path: "packages/shared"

   lifecycle:
     on_pickup:
       - run: "echo 'Starting work on {{ticket_slug}}'"
     on_submit:
       - run: "npm run lint"
       - run: "npm run test"
     on_approve:
       - run: "git add -A"
       - run: "git commit -m '{{commit_message}}'"
   ```

2. **Loading**
   - Find `.cortex/cortex.yaml` from current or parent directories
   - Parse YAML into Go structs
   - Validate required fields
   - Return sensible defaults for optional fields

3. **Discovery**
   - `FindProjectRoot(path)` - walk up to find `.cortex/` directory
   - `Load(projectPath)` - load config from project root

4. **Integration Points**
   - Lifecycle hooks use this for hook definitions
   - Git discovery uses this for repo paths
   - Daemon uses this for project detection

## Verification

```bash
make build   # Builds successfully
make test    # Tests pass
make lint    # No lint errors
```

## Notes

- This is separate from daemon's global config (~/.cortex/settings.yaml)
- Project config is per-project, global config is per-user
- Handle missing config file gracefully (use defaults)
- Consider caching loaded config

## Implementation

### Commits Pushed
- `2c14940` feat: add project config package for loading .cortex/cortex.yaml
- `6913e2d` Merge branch 'ticket/2026-01-19-project-config'

### Key Files Changed
- `internal/project/config/config.go` - Config structs (Config, GitConfig, RepoConfig, LifecycleConfig, HookConfig) and functions (DefaultConfig, FindProjectRoot, Load, LoadFromPath, Validate)
- `internal/project/config/errors.go` - Error types (ProjectNotFoundError, ConfigParseError, ValidationError) with Is* helpers
- `internal/project/config/config_test.go` - 12 test cases covering all functionality

### Important Decisions
- Package placed at `internal/project/config/` to separate from daemon config (`internal/daemon/config/`)
- `FindProjectRoot` walks up directory tree using `os.Stat` to find `.cortex/` directory
- Default config: agent=`claude`, repos=[`"."`], no lifecycle hooks
- Missing config file returns defaults without error (graceful handling)
- Agent type restricted to `claude` or `opencode`
- Validation ensures non-empty repo paths and hook run commands

### Scope Changes
- None - implemented as specified in the plan

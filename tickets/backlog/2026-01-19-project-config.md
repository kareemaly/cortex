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

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

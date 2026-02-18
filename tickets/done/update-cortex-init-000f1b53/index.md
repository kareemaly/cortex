---
id: 000f1b53-9d4d-4613-9265-5f50733daae8
title: Update cortex init with agent detection and migrate existing projects
type: work
tags:
    - configuration
    - cleanup
    - v1.1.0
references:
    - ticket:0ccfed2a-371a-4819-b65c-cca8e8da64d8
created: 2026-02-14T09:24:21.354904Z
updated: 2026-02-14T09:38:08.688509Z
---
## Problem

`cortex init` currently requires the user to manually specify `--agent claude` or `--agent opencode` with no validation that the agent binary is actually installed. Additionally, existing projects still have `extend` pointing to old `~/.cortex/defaults/claude-code` or `~/.cortex/defaults/opencode` paths that no longer exist after the defaults unification (ticket 0ccfed2a).

## What needs to change

### 1. Agent detection in `cortex init`

When running `cortex init`, the CLI should detect which agent binaries are installed:

- Check if `claude` binary is available on PATH
- Check if `opencode` binary is available on PATH
- **Neither installed** → error with clear message telling user to install one
- **Only one installed** → auto-select that agent, inform user
- **Both installed** → prompt user to choose (interactive selection)
- `--agent` flag overrides auto-detection but **must validate the binary exists** — fail with error if specified agent binary is not installed

### 2. Migrate existing projects

For each project registered in `~/.cortex/settings.yaml`:

- Read the project's `.cortex/cortex.yaml`
- If `extend` points to an old path (`~/.cortex/defaults/claude-code` or `~/.cortex/defaults/opencode`), update it to `~/.cortex/defaults/main`
- If the project config is minimal (just `name` + `extend`, relying on old merge behavior), rewrite it as a complete self-contained config with all defaults inlined for the appropriate agent type (detect from the old extend path or from the existing `agent` field)
- This migration should run as part of `cortex defaults upgrade` (since that's already cleaning up old directories)

### 3. Clean up old defaults directories

`cortex defaults upgrade` already removes old `claude-code/` and `opencode/` directories (from ticket 0ccfed2a). The migration of project configs should happen **before** those directories are removed, so we can read the old base configs if needed to determine what agent type a project was using.

## Acceptance criteria

- `cortex init` with no `--agent` flag auto-detects installed agents
- If both `claude` and `opencode` are installed, user is prompted to choose
- If only one is installed, it's auto-selected
- `--agent` flag works but fails if specified binary is not on PATH
- `cortex defaults upgrade` migrates existing projects' cortex.yaml files (updates extend path, inlines full config if needed)
- After migration, all projects have self-contained cortex.yaml with `extend: ~/.cortex/defaults/main`
- Existing projects continue to work correctly after migration
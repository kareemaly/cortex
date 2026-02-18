---
id: 0ccfed2a-371a-4819-b65c-cca8e8da64d8
title: Unify defaults directory and decouple extend from cortex.yaml merging
type: work
tags:
    - configuration
    - cleanup
    - v1.1.0
created: 2026-02-14T09:00:52.009632Z
updated: 2026-02-14T09:20:57.789031Z
---
## Problem

Cortex currently maintains two identical copies of prompts under `~/.cortex/defaults/claude-code/` and `~/.cortex/defaults/opencode/`. The prompts are byte-for-byte identical — the only difference between the two directories is `cortex.yaml`. This is unnecessary duplication. Additionally, the `extend` field in project config currently merges both cortex.yaml AND resolves prompts, which adds complexity.

## What needs to change

### 1. Unify defaults into `~/.cortex/defaults/main/`

- Replace `internal/install/defaults/claude-code/` and `internal/install/defaults/opencode/` with a single `internal/install/defaults/main/` directory
- This directory contains **only prompts** — no `cortex.yaml`, no `CONFIG_DOCS.md`
- Structure: `internal/install/defaults/main/prompts/{architect,meta,ticket}/{SYSTEM,KICKOFF,APPROVE}.md`
- Update the Go embed directive and all install/copy logic to work with the single `main/` directory
- `CopyEmbeddedDefaults` should create `~/.cortex/defaults/main/` on disk
- Remove `DefaultsDirForAgent()` or equivalent agent-to-directory mapping — there's only one defaults directory now

### 2. Decouple `extend` from cortex.yaml merging

- The `extend` field in `.cortex/cortex.yaml` should **only** control prompt resolution path
- Remove the cortex.yaml merge logic: `MergeConfigs()`, recursive config loading from base, circular reference detection for config
- `Config.Load()` should read the project's `.cortex/cortex.yaml` as the complete, self-contained configuration
- The `extend` path is still validated (directory must exist) and stored for `PromptResolver` to use
- Prompt resolution waterfall remains: project `.cortex/prompts/` → `extend` path prompts → not found

### 3. Update `cortex defaults upgrade` command

- Should create/update `~/.cortex/defaults/main/` from embedded defaults
- Should remove old `~/.cortex/defaults/claude-code/` and `~/.cortex/defaults/opencode/` directories if they exist
- Comparison logic should work with the single `main/` directory

## Acceptance criteria

- Only `~/.cortex/defaults/main/prompts/` exists after init or defaults upgrade (no claude-code/opencode directories)
- `extend` field only affects prompt resolution, not cortex.yaml loading
- `Config.Load()` treats project cortex.yaml as complete config (no merging from base)
- Prompt resolution still works: project prompts → extend path prompts → not found
- `cortex eject` still works (copies from extend path to project)
- `cortex defaults upgrade` cleans up old directories and creates new structure
- All existing tests pass or are updated accordingly
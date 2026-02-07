---
id: 1eb7f570-a8eb-41c5-8d1b-69ff46334f0a
title: Configuration Extension System with Deep Merge and Prompt Inheritance
type: work
created: 2026-01-29T08:27:32.946334Z
updated: 2026-01-29T08:47:55.111555Z
---
## Summary

Add an `extend` attribute to project configuration that allows inheriting from a base config folder. This enables minimal project setup (single `cortex.yaml` with `extend` pointing to shared defaults) and lays groundwork for future plugin/config sharing ecosystem.

## Requirements

### 1. `extend` attribute in project config
- New optional field `extend` in `.cortex/cortex.yaml` accepting a path to a base config folder
- Single path only (no arrays, no chaining for now)

### 2. Path resolution semantics
| Path type | Behavior |
|-----------|----------|
| Absolute (`/opt/cortex/...`) | Use as-is |
| Tilde (`~/.cortex/...`) | Expand to current user's home directory |
| Relative (`./foo`, `../foo`) | Resolve from **project root**, not cwd |

- Tilde paths are user-specific by design — each developer can have their own defaults
- Teams wanting identical configs across users/CI should use absolute paths or commit the base config within the repo
- Relative paths resolved from project root ensure consistent behavior regardless of where `cortex` is invoked

### 3. Deep merge semantics
- Base config loaded first, then project config deep-merged on top
- Project values win on any conflict
- Example: project defines only `architect.agent`, inherits `architect.args` from base

### 4. Prompt inheritance
- If project has no `prompts/` folder, use prompts from extended base entirely
- If project defines specific prompt files, those override the corresponding base prompts
- Unoverridden prompts still inherited from base

### 5. Updated `cortex init` flow
- Creates `~/.cortex/defaults/basic/` with full default config:
  - `cortex.yaml` (default project config without `extend`)
  - `prompts/` folder with default prompts
- Creates `./.cortex/cortex.yaml` in project with minimal content:
  ```yaml
  extend: ~/.cortex/defaults/basic
  ```
- If `~/.cortex/defaults/basic/` already exists, skip creating it (don't overwrite user customizations)

### 6. Fail-fast on missing base
- If `extend` path does not exist (after resolution), fail immediately with clear error message
- Validate at config load time, not lazily

## Acceptance Criteria

- [ ] `extend` attribute parsed from project config
- [ ] Absolute paths used as-is
- [ ] Tilde paths expanded to current user's home
- [ ] Relative paths resolved from project root (not cwd)
- [ ] Config values deep-merged (base first, project overlays)
- [ ] Prompts inherited from base when project doesn't define them
- [ ] Individual prompt files in project override corresponding base prompts
- [ ] `cortex init` creates `~/.cortex/defaults/basic/**` on first run
- [ ] `cortex init` creates minimal `.cortex/cortex.yaml` with `extend` attribute
- [ ] Clear error when extended path doesn't exist
- [ ] Existing projects without `extend` continue working (backward compatible)
---
id: 88af3f2b-2f45-414a-9d56-1237d8d4d262
title: Fix Extended Base Prompt Path Construction
type: work
created: 2026-01-30T09:07:21.361846Z
updated: 2026-01-30T09:15:23.142049Z
---
## Summary

The prompt resolver added in ticket `3eaaf1ec` constructs incorrect paths for extended bases. It's appending `.cortex/prompts/` to the extended path, but the extended path already points to a cortex config directory structure.

**Current (broken):**
```
~/.cortex/defaults/claude-code/.cortex/prompts/architect/SYSTEM.md
```

**Expected:**
```
~/.cortex/defaults/claude-code/prompts/architect/SYSTEM.md
```

## Requirements

- When resolving prompts from an extended base, use `{extend}/prompts/{role}/SYSTEM.md`
- Do not add extra `.cortex/` segment to extended paths
- The `extend` value points directly to a cortex config directory (equivalent to `.cortex/`)

## Acceptance Criteria

- [ ] Extended base paths resolve correctly without duplicate `.cortex/` segment
- [ ] Spawning architect works for projects that extend `~/.cortex/defaults/claude-code`
- [ ] Project-local prompts still resolve correctly (`.cortex/prompts/...`)
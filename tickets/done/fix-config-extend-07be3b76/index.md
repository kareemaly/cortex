---
id: 07be3b76-cdf1-4467-9e78-ca71c6c79c25
title: Fix Config Extend Path Resolution for Base Configs
type: work
created: 2026-02-02T11:46:17.967739Z
updated: 2026-02-02T11:55:02.934366Z
---
## Summary

Config extension fails silently because the loader expects `.cortex/cortex.yaml` inside the extend path, but installed defaults have `cortex.yaml` directly at the root.

## Problem

When a project config has:
```yaml
extend: ~/.cortex/defaults/claude-code
```

The config loader looks for:
```
~/.cortex/defaults/claude-code/.cortex/cortex.yaml  ← NOT FOUND
```

But the installed defaults structure is:
```
~/.cortex/defaults/claude-code/
├── cortex.yaml          ← FILE IS HERE
└── prompts/
```

This causes extends to silently fail, falling back to empty defaults. Users don't get inherited args (like `--permission-mode plan`).

## Solution

Change the config loader to look for `cortex.yaml` directly in the extend path, not inside a `.cortex` subdirectory. Base configs are not full projects — they're config bundles.

**In `internal/project/config/config.go`:**

Current (line ~154):
```go
configPath := filepath.Join(resolvedExtendPath, ".cortex", "cortex.yaml")
```

Should be:
```go
configPath := filepath.Join(resolvedExtendPath, "cortex.yaml")
```

## Acceptance Criteria
- [ ] Extend paths resolve `cortex.yaml` directly (not `.cortex/cortex.yaml`)
- [ ] Existing tests updated to reflect new behavior
- [ ] `cortex config show` displays correctly merged config when extending defaults
- [ ] Plan mode args are inherited when extending `~/.cortex/defaults/claude-code`
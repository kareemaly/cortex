---
id: 4064b6ad-3e13-40ae-982e-0eb6342550b9
title: Add version command to cortexd
type: ""
created: 2026-01-21T10:01:15Z
updated: 2026-01-21T10:01:15Z
---
`cortexd version` is missing. Only `cortex version` exists.

## Current State

```bash
$ cortex version
cortex dev
  Commit:     ea41837
  Built:      2026-01-20T16:44:34Z
  Go version: go1.25.6
  Platform:   darwin/arm64

$ cortexd version
Error: unknown command "version" for "cortexd"
```

## Requirements

Add `cortexd version` command that shows the same info as `cortex version`.

## Files to reference

- `cmd/cortex/commands/version.go` - existing implementation to copy
- `pkg/version/version.go` - shared version package

## Verification

```bash
make build
cortexd version  # Should show version info
```

## Implementation

### Commits
- `7a17456` feat: add cortexd version command

### Key Files Changed
- `cmd/cortexd/commands/version.go` (new) - Version command for cortexd

### Decisions
- Omitted daemon status check that exists in `cortex version` since `cortexd` IS the daemon
- Used identical output format as `cortex version` for consistency

### Scope
No scope changes from original ticket
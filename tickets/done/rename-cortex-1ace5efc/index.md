---
id: 1ace5efc-26d3-4cfc-96ce-6c5cf2c46d52
title: Rename cortex install to cortex init
type: ""
created: 2026-01-27T10:42:30.303431Z
updated: 2026-01-27T11:05:29.690555Z
---
## Problem

The project setup command is `cortex install`, but `init` is the universally recognized convention (`git init`, `npm init`, `go mod init`, etc.).

## Solution

Rename the `install` command to `init` across the codebase.

## Scope

- Rename `cmd/cortex/commands/install.go` → `cmd/cortex/commands/init.go`
- Rename the cobra command from `install` to `init`
- Update command description/long text
- Rename `internal/install/` package → `internal/init/` (or keep as `install` internally if `init` is a Go keyword conflict — may need `initializer` or similar)
- Update all references: imports, tests, docs, CLAUDE.md, prompt files
- Update variable names: `installCmd` → `initCmd`, `runInstall` → `runInit`, `installGlobalOnly` → `initGlobalOnly`, `installForce` → `initForce`

## Acceptance Criteria

- [ ] `cortex init` works as the setup command
- [ ] `cortex install` no longer exists
- [ ] All references updated (imports, tests, docs)
- [ ] `make build && make test` pass
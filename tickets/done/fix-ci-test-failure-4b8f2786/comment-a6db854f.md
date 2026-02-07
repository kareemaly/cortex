---
id: a6db854f-6462-470f-9bb1-37ce49893de4
author: claude
type: review_requested
created: 2026-02-04T12:14:48.707037Z
action:
    type: git_diff
    args:
        commit: 42d0049
        repo_path: /Users/kareemelbahrawy/.cortex/worktrees/eddb5946-ceef-462a-97f3-e588b7cad607
---
## Summary

Fixed CI test failure caused by incorrect dependency classification in `go.mod`.

## Root Cause

The `github.com/charmbracelet/x/ansi` package is directly imported in `internal/cli/tui/ticket/model.go:14` but was marked as `// indirect` in `go.mod`. Go 1.24 (used in CI) has stricter module verification than Go 1.25.6 (local), causing test failures with exit code 2.

## Changes Made

**go.mod** (1 change):
- Moved `github.com/charmbracelet/x/ansi v0.10.1` from the indirect require block to the direct require block

## Verification

- `go mod tidy -diff` returns no changes (exit 0)
- `make test` passes all tests
- `make lint` passes with 0 issues

## Commit

`42d0049` - fix: correct go.mod indirect dependency marking for charmbracelet/x/ansi
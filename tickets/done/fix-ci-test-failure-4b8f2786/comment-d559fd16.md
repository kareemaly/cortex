---
id: d559fd16-fa46-4fd2-a496-d6ebae26366a
author: claude
type: done
created: 2026-02-04T12:16:28.365164Z
---
## Summary

Fixed CI test failure caused by incorrect Go module dependency classification.

## Root Cause

The `github.com/charmbracelet/x/ansi` package was directly imported in `internal/cli/tui/ticket/model.go:14` but was marked as `// indirect` in `go.mod:24`. Go 1.24 (used in CI) has stricter module verification than Go 1.25.6 (local), causing `make test` to fail with exit code 2 during module resolution.

## Fix Applied

Ran `go mod tidy` which moved `github.com/charmbracelet/x/ansi v0.10.1` from the indirect require block to the direct require block in `go.mod`.

## Files Changed

- `go.mod` - 1 line change (moved dependency from indirect to direct)

## Verification

- `go mod tidy -diff` exits cleanly (no further changes needed)
- `make test` passes all tests
- `make lint` passes with 0 issues

## Commit

`42d0049` - fix: correct go.mod indirect dependency marking for charmbracelet/x/ansi

Merged to main and pushed to origin.
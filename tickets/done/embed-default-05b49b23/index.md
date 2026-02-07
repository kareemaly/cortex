---
id: 05b49b23-9138-458a-b8ea-9fcd297f1262
title: Embed Default Config Folders and Copy on Init
type: work
created: 2026-01-30T09:17:20.703149Z
updated: 2026-01-30T09:35:11.535541Z
---
## Summary

Refactor how default configurations (like `claude-code`) are installed. Instead of generating files programmatically, embed the actual folder structure in the codebase and copy it to `~/.cortex/defaults/` on init.

## Requirements

- Create an embedded defaults directory in the codebase (e.g., `internal/install/defaults/claude-code/`) containing:
  - `cortex.yaml`
  - `prompts/architect/SYSTEM.md`
  - `prompts/ticket/SYSTEM.md`
  - Any other default files

- On `cortex init`, copy the embedded folder to `~/.cortex/defaults/claude-code/` if it doesn't already exist

- Structure should support adding more default configurations later (just add another folder like `defaults/cursor/`, `defaults/windsurf/`, etc.)

- Use Go embed (`//go:embed`) to bundle the defaults into the binary

## Acceptance Criteria

- [ ] Default config files live in source tree (not generated programmatically)
- [ ] `cortex init` copies embedded defaults to `~/.cortex/defaults/` when missing
- [ ] Existing defaults are not overwritten (preserve user customizations)
- [ ] Easy to add new default configurations by adding folders
- [ ] Binary is self-contained (no external file dependencies)
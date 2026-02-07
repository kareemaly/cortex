---
id: 3eaaf1ec-c3e7-4992-ba90-a54ddc2e59c1
title: Abstract Prompt Resolution with Extension Fallback
type: work
created: 2026-01-30T08:50:59.366925Z
updated: 2026-01-30T09:05:38.322791Z
---
## Summary

Prompt loading doesn't respect the `extend` configuration. When a project extends a base (e.g., `extend: ~/.cortex/defaults/claude-code`), the system should fall back to the extended path's prompts when project-specific prompts don't exist. Currently it fails immediately if the project prompt is missing.

## Requirements

- Create an abstracted prompt resolution function that checks paths in order:
  1. Project path (`.cortex/prompts/{role}/SYSTEM.md`)
  2. Extended base path (if `extend` is configured)
  3. Fail with clear error if neither exists

- Update all prompt loading in the daemon to use this abstraction

- All clients (TUIs, CLI) must go through the daemon HTTP API for prompt resolution — no direct file access

## Acceptance Criteria

- [ ] Prompt resolution respects `extend` fallback chain
- [ ] Spawning architect/ticket agents works when project has no local prompts but extends a base that does
- [ ] All daemon prompt loading uses the new abstracted resolver
- [ ] TUIs and CLI do not read prompt files directly — all access goes through daemon API
- [ ] Clear error message when prompt not found in any location
---
id: 0428bc92-e8a6-4952-98f6-d2f70e7097da
title: Add diff preview to `cortex defaults upgrade`
type: work
created: 2026-02-05T11:31:09.260033Z
updated: 2026-02-05T11:39:56.507194Z
---
Before upgrading defaults, show a diff of what will change and require user confirmation (y/n). This prevents unexpected overwrites of customizations.

**Location**: `cmd/cortex/commands/defaults.go` or similar
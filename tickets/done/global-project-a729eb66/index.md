---
id: a729eb66-41ef-48b1-9422-23252a8940eb
title: Global Project Registry in settings.yaml
type: ""
created: 2026-01-27T10:40:00.454421Z
updated: 2026-01-27T11:29:58.702315Z
---
## Problem

There is no global registry of projects. Projects are discovered ad-hoc via upward directory traversal (`FindProjectRoot`). The daemon has no awareness of which projects exist on the machine until they're explicitly accessed via the `X-Cortex-Project` header.

This blocks cross-project visibility features like the daemon dashboard TUI.

## Solution

Add a `projects` list to `~/.cortex/settings.yaml` that tracks all registered projects:

```yaml
port: 4200
log_level: info

projects:
  - path: /Users/kareem/projects/cortex1
    title: Cortex
  - path: /Users/kareem/projects/myapp
    title: My App
```

### Auto-registration

When `cortex install` sets up `.cortex/` in a project, it should also append the project to `~/.cortex/settings.yaml` automatically.

### CLI commands

- `cortex projects` — list all registered projects with basic stats (ticket counts)
- Consider `cortex register` / `cortex unregister` for manual management

## Scope

- Extend `internal/daemon/config/config.go` to include `Projects` field in the global `Config` struct
- Add project entry struct: `path` (required, absolute), `title` (optional, defaults to directory name)
- Auto-register on `cortex install`
- Validate on load: skip/warn for paths that no longer exist or lack `.cortex/`
- Add `cortex projects` CLI command
- Add API endpoint `GET /projects` for the daemon to serve registered projects

## Acceptance Criteria

- [ ] `~/.cortex/settings.yaml` supports a `projects` list
- [ ] `cortex install` auto-registers the project in global settings
- [ ] `cortex projects` lists all registered projects
- [ ] Stale paths (missing `.cortex/`) are handled gracefully
- [ ] `GET /projects` API endpoint returns registered projects
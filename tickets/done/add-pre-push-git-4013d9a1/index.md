---
id: 4013d9a1-73b7-4d2d-a9fd-53781a7e473c
title: Add pre-push git hook for lint and build
type: work
created: 2026-02-04T13:17:26.820813Z
updated: 2026-02-04T13:25:12.268547Z
---
## Goal

Prevent pushing code that fails lint or build by adding a git pre-push hook.

## Requirements

### Hook script

Create `.githooks/pre-push` that runs:
1. `make lint` - fail push if lint errors
2. `make build` - fail push if build fails

### Setup

- Add `.githooks/` directory to repo
- Update `cortex init` or add Makefile target to configure git: `git config core.hooksPath .githooks`
- Document in README or CONTRIBUTING

### Considerations

- Hook should be fast (lint + build is ~15-20s, acceptable)
- Clear error output so developer knows what failed
- Optional: add `--no-verify` bypass documentation for emergencies
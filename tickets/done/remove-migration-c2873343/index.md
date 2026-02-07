---
id: c2873343-0fad-40e0-9761-d5f2f2bcde3f
title: Remove migration scripts
type: chore
created: 2026-02-07T11:04:42.328401Z
updated: 2026-02-07T11:06:27.712636Z
---
## Overview

Remove all migration scripts from the codebase. They've served their purpose — all projects are migrated.

## Scripts to remove

1. **`cmd/migrate/main.go`** — JSON → YAML frontmatter migration (just created this session)
2. **Old Python migration script** — migrated from cortex0. Find it and remove it.

Search the codebase for any other migration-related files or scripts.

## Branch

Working on `main` branch.
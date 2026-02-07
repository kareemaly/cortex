---
id: 45b4ec7d-6925-48a9-b4cd-44cedcdfd628
title: Research cross-project ticketing for architects
type: research
created: 2026-02-05T09:11:32.086772Z
updated: 2026-02-05T09:31:29.83921Z
---
## Research Question

Is it feasible for architects to access, create, and spawn tickets across different projects from a single session?

## Context

Currently, project context is scoped via:
- `X-Cortex-Project` header (HTTP)
- `CORTEX_PROJECT_PATH` env (MCP)

Architects are bound to one project at a time.

## Areas to Investigate

1. **Current Architecture Constraints**
   - How is project context threaded through MCP tools?
   - What would need to change in the daemon API?
   - How does StoreManager handle multi-project access?

2. **Use Cases**
   - Spawning work in a dependency project
   - Viewing tickets across a monorepo or related projects
   - Coordinating work that spans multiple codebases

3. **Design Options**
   - Explicit project parameter on each tool call
   - Project switching within architect session
   - Global architect mode vs project-scoped architect

4. **Security/Isolation Concerns**
   - Should all projects be accessible or only registered ones?
   - Permission model for cross-project operations

## Deliverables

- Summary of architectural implications
- Recommended approach (if feasible)
- List of files/components that would need modification
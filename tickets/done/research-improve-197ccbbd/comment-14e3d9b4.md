---
id: 14e3d9b4-792f-4d63-9ea9-4cb79bddfc04
author: claude
type: comment
created: 2026-02-09T16:27:26.778715Z
---
## Finding 4: Concrete Examples — The Biggest Missing Piece

Claude's prompting guide is emphatic: **"Be vigilant with examples & details — Claude pays close attention to details and examples as part of its precise instruction following."** The current prompt has zero examples. This is the highest-impact addition.

### Recommended: Good vs Bad ticket examples

```markdown
<examples>
<example type="bad" label="Implementation assumptions">
## Add health check endpoint

Add a `/health` endpoint to `internal/daemon/api/server.go`:

1. Create `handleHealth()` function that returns JSON `{"status": "ok"}`
2. Register route in `setupRoutes()` at line ~45
3. Add corresponding test in `server_test.go`
4. Estimated effort: Low (~30 min)
</example>

<example type="good" label="Requirements-focused">
## Add health check endpoint

The daemon should expose a health check endpoint for monitoring tools and load balancers.

**Requirements:**
- GET endpoint that returns service health status
- Response should include: daemon uptime, whether the store is accessible, and version info
- Should work without authentication (monitoring tools need unauthenticated access)

**Acceptance criteria:**
- Endpoint returns 200 when healthy, 503 when degraded
- Response includes uptime, version, and store connectivity
- Works without project context header
</example>
</examples>
```

### Why this pair works

The **bad example** shows exactly the anti-pattern we want to eliminate:
- Assumes specific file paths (`internal/daemon/api/server.go`)
- Assumes specific function names (`handleHealth()`, `setupRoutes()`)
- Includes guessed line numbers (`line ~45`)
- Includes a time estimate (`~30 min`)
- Tells the agent HOW instead of WHAT

The **good example** shows the target behavior:
- Describes the WHAT (health check for monitoring)
- Explains WHY (monitoring tools, load balancers)
- Specifies acceptance criteria (200/503 status codes)
- Includes design constraints from context (no auth needed)
- Leaves all implementation decisions to the ticket agent

From the best practices guide: showing both good and bad examples is more effective than just good examples, because Claude can see the contrast and internalize the difference.
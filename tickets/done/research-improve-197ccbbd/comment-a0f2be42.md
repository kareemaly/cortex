---
id: a0f2be42-5440-4c99-80f3-4dc97eccc64c
author: claude
type: comment
created: 2026-02-09T16:26:55.56411Z
---
## Finding 2: Current SYSTEM.md Analysis — Critical Weaknesses

The prompt has significant gaps that directly cause the behavioral issues identified in the ticket:

### Gap 1: No ticket quality guidelines (THE critical gap)
The prompt says "create a well-scoped ticket" but provides zero guidance on what "well-scoped" means. No mention of:
- What to include (requirements, acceptance criteria, design constraints)
- What NOT to include (implementation details, file paths, effort estimates)
- When to research first vs. delegate immediately

This is the root cause of the "assumptions" problem — the architect has no instruction telling it what a good ticket looks like, so it fills tickets with whatever seems helpful, including wrong implementation guesses.

### Gap 2: No "WHAT not HOW" principle
The #1 behavioral issue. The architect currently writes tickets like:
> "Update `internal/daemon/api/server.go` to add a new handler function `handleFoo()` that calls `store.GetFoo()`..."

This is wrong in two ways: (a) the file paths and function names may be wrong, (b) the ticket agent will explore the codebase and figure out implementation anyway. Wrong assumptions actively mislead.

### Gap 3: No examples (good or bad)
Claude's prompting guide specifically warns: "Be vigilant with examples & details — Claude pays close attention to details and examples." The current prompt has zero examples of good or bad tickets. This is a huge missed opportunity — examples are the single most effective way to shape output quality.

### Gap 4: "No time estimates" doesn't extend to tickets
Line 44 says "Do not give time estimates" for communication, but nothing prevents the architect from writing "Estimated effort: Medium" or "Complexity: Low" in ticket bodies. This needs to be explicitly called out in the ticket-writing guidelines.

### Gap 5: Missing "explore before assuming" workflow
When an architectural decision requires technical details (e.g., "should we use a wrapper pattern or peer models?"), the architect should spawn an explore agent first, get accurate facts, THEN write the ticket. Currently there's no guidance on this workflow — the architect just guesses.

### Gap 6: Missing ticket type guidance
The architect can create tickets with types (work, debug, research, chore) but has no guidance on when to use which type or how to adjust ticket structure per type.

### Gap 7: Incomplete pre-spawn workflow
Steps 1-4 in "After Spawning" describe post-spawn flow. But there's nothing about the pre-spawn workflow: how to decompose user requests, when to create multiple tickets vs. one, how to sequence tickets with dependencies.

### Gap 8: No docs workflow
The architect has access to docs tools (createDoc, readDoc, listDocs, etc.) but the SYSTEM.md says nothing about when/how to use docs. Session docs are mentioned in KICKOFF.md but not in the system prompt.

### Gap 9: No cross-project awareness
The architect tools include optional `project_path` parameters for cross-project operations, and `listProjects` is available. The SYSTEM.md says nothing about this capability.
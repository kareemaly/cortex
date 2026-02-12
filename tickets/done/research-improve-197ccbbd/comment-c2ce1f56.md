---
id: c2ce1f56-0053-43b2-9048-69bc0d1ea29c
author: claude
type: comment
created: 2026-02-09T16:27:13.068797Z
---
## Finding 3: Best Practices Applied — The "WHAT not HOW" Principle

This is the most important change. Here's the analysis of why it matters and how to implement it, grounded in prompting best practices.

### Why assumptions are actively harmful

From Claude's prompting guide: **"Add context to improve performance — providing context or motivation behind your instructions helps Claude better understand your goals."**

The motivation here is clear: when the architect writes "Update `internal/daemon/api/server.go` line 145 to add handler X", and the ticket agent finds that file doesn't have a handler at line 145, the agent either:
1. Wastes time reconciling the wrong assumption with reality
2. Follows the bad assumption and creates a mess
3. Ignores the ticket details entirely (defeating the purpose)

All three outcomes are worse than simply not including the assumption.

### Recommended approach: XML-tagged section with examples

From Claude's prompting guide: **"Be vigilant with examples & details"** and **"Use XML format indicators."**

I recommend a new section using an XML tag with paired good/bad examples:

```markdown
<ticket_quality>
Tickets define WHAT needs to be done, not HOW to implement it. The ticket agent will explore the codebase, understand patterns, and determine the right approach. Your job is to give them clear requirements.

**Include:**
- Clear problem statement or feature description
- Acceptance criteria (what "done" looks like)
- Design constraints and decisions (when the user has expressed a preference)
- References to related tickets or docs
- Relevant user context or background

**Never include:**
- Assumed file paths or function names (you haven't read the code)
- Guessed implementation steps
- Time estimates, effort sizing, or complexity ratings
- Speculative code patterns or signatures

When an architectural decision requires technical accuracy (e.g., choosing between patterns), use an explore agent to gather facts first. Never guess.
</ticket_quality>
```

### Why this works per best practices

1. **Explicit instructions** — "Tickets define WHAT, not HOW" is a clear directive
2. **Motivation** — "you haven't read the code" explains WHY file paths are wrong
3. **Positive framing** — "Include" list tells what TO do, not just what to avoid
4. **Concrete lists** — Both include/exclude are specific enough to act on
5. **Escape hatch** — "When architectural decision requires technical accuracy, use an explore agent" gives the right workflow for edge cases
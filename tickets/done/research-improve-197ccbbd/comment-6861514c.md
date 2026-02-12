---
id: 6861514c-81be-44fb-9293-cf7a7f642d02
author: claude
type: comment
created: 2026-02-09T16:28:00.062013Z
---
## Finding 6: Additional Improvements — Smaller but Valuable

### 1. Ticket type guidance
The architect can create tickets with types but has no guidance. Recommended addition:

```markdown
### Ticket Types
- **work** — feature implementation, enhancements, refactoring
- **debug** — bug investigation and fixing (include reproduction steps)
- **research** — exploration, analysis, documentation (read-only, no code changes)
- **chore** — maintenance tasks, dependency updates, cleanup
```

### 2. Pre-spawn workflow
Currently missing. Add guidance on decomposition:

```markdown
### Before Creating Tickets
- Break large requests into independent, well-scoped tickets
- Each ticket should be completable by one agent in one session
- If tickets have dependencies, note them in the ticket body — but prefer independent tickets when possible
```

### 3. Explore-first workflow
When architectural decisions need accuracy:

```markdown
### When Technical Details Matter
If a design decision requires knowing how the codebase currently works (e.g., "should we extend the existing handler or create a new one?"), spawn an explore agent first. Get accurate technical context, then write the ticket with facts — not guesses.
```

### 4. Review workflow clarification
The current prompt says "User reviews and approves directly (you do not have an approval tool)" but could be clearer about the architect's role during review:

```markdown
### During Review
When a ticket moves to review, the user will review the agent's work directly. You may be asked to look at the review results or help resolve issues. Wait for the user's input rather than proactively checking on spawned sessions.
```

### 5. Docs workflow (minor)
At minimum, mention docs exist:

```markdown
### Documentation
Use `createDoc` to capture session decisions, architectural context, or research findings that should persist beyond ticket comments. Docs survive across sessions; ticket comments may not be revisited.
```

### 6. Claude Opus 4.6 tuning
From the prompting best practices, Opus 4.6 is more responsive to system prompts than previous models. The guide warns: "If your prompts were designed to reduce undertriggering on tools or skills, these models may now overtrigger." 

This means we should:
- Use normal language instead of CAPS/CRITICAL markers for tool usage
- Be precise rather than aggressive with constraints
- Trust that the model will follow instructions without heavy emphasis

The current prompt is already fairly restrained in tone, which is good. The new sections should maintain this natural tone.
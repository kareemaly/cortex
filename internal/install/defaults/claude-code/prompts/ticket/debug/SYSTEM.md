## Cortex Debug Workflow

Use Cortex MCP tools: `readTicket`, `addComment`, `addBlocker`, `requestReview`, `concludeSession`.

1. **Reproduce**: Reproduce the issue and document steps
2. **Investigate**: Isolate the root cause systematically
3. **Document**: Add comment with findings BEFORE making changes
4. **Fix**: Implement the fix only after documenting investigation
5. Call `requestReview` with root cause explanation and wait for approval

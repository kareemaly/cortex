## Review Approved

Your changes have been reviewed and approved. Complete the following steps:

1. **Commit all changes**
   - Run `git status` to check for uncommitted changes
   - Commit any remaining changes

2. **Merge to main**
   - Run `cd {{.ProjectPath}} && git merge {{.WorktreeBranch}}`

3. **Call concludeSession**
   - Call `mcp__cortex__concludeSession` with a complete report including:
     - Summary of all changes made
     - Key decisions and their rationale
     - List of files modified
     - Any follow-up tasks or notes

This will mark the ticket as done and end your session.

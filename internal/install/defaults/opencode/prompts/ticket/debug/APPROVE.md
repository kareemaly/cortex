## Approved

{{if .IsWorktree}}
Working in worktree on branch `{{.WorktreeBranch}}`.

1. Run tests to verify the fix
2. Commit with root cause explanation in message
3. Merge to main:
   ```bash
   cd {{.ProjectPath}}
   git merge {{.WorktreeBranch}}
   ```
4. Call `concludeSession` with root cause and resolution summary
{{else}}
1. Run tests to verify the fix
2. Commit with root cause explanation in message
3. Call `concludeSession` with root cause and resolution summary
{{end}}

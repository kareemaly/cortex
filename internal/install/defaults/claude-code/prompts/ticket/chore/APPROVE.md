## Approved

{{if .IsWorktree}}
Working in worktree on branch `{{.WorktreeBranch}}`.

1. Run tests if applicable
2. Commit all changes
3. Merge to main:
   ```bash
   cd {{.ProjectPath}}
   git merge {{.WorktreeBranch}}
   ```
4. Call `concludeSession` with brief summary
{{else}}
1. Run tests if applicable
2. Commit changes
3. Call `concludeSession` with brief summary
{{end}}

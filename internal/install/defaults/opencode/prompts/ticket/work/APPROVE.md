## Approved

{{if .IsWorktree}}
Working in worktree on branch `{{.WorktreeBranch}}`.

1. Run tests to verify changes
2. Commit all changes with descriptive message
3. Merge to main:
   ```bash
   cd {{.ProjectPath}}
   git merge {{.WorktreeBranch}}
   ```
4. Call `concludeSession` with summary
{{else}}
1. Run tests to verify changes
2. Commit with descriptive message
3. Call `concludeSession` with summary
{{end}}

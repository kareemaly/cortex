## Approved

Approved. Commit and conclude.

{{if .IsWorktree}}
You are working in a git worktree on branch `{{.WorktreeBranch}}`.

1. Commit all changes in the worktree
2. Switch to main project and merge:
   ```bash
   cd {{.ProjectPath}}
   git merge {{.WorktreeBranch}}
   git push origin main
   ```
3. Call `concludeSession` with a brief summary
{{else}}
1. Commit your changes
2. Push to origin
3. Call `concludeSession` with a brief summary
{{end}}

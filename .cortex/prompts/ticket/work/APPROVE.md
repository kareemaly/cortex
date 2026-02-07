## Approved

{{ if .IsWorktree }}
You are working in a git worktree on branch `{{ .WorktreeBranch }}`.

1. **Review documentation impact** — If your changes affect:
   - User-facing behavior → Update `README.md`
   - Architecture, code paths, or patterns → Update `CLAUDE.md`

   **CLAUDE.md guidance**: Keep it high-level. Focus on:
   - Architecture and component relationships
   - Key file paths for common tasks
   - Build/test commands
   - Patterns and conventions

2. Commit all changes in the worktree
3. Switch to main project and merge:
   ```bash
   cd {{ .ProjectPath }}
   git merge {{ .WorktreeBranch }}
   git push origin main
   ```
4. Call `concludeSession` with a summary of what was done
{{ else }}
1. **Review documentation impact** — If your changes affect:
   - User-facing behavior → Update `README.md`
   - Architecture, code paths, or patterns → Update `CLAUDE.md`
   - Detailed guides or tutorials → Add to `docs/` (create if needed)

   **CLAUDE.md guidance**: Keep it high-level. Focus on:
   - Architecture and component relationships
   - Key file paths for common tasks
   - Build/test commands
   - Patterns and conventions

   Avoid detailed implementation docs — those belong in `docs/` or code comments.

2. Commit your changes
3. Push to origin
4. Call `concludeSession` with a summary of what was done
{{ end }}

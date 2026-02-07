---
id: fbbc8a62-e99a-46a8-b8b1-390f3c70f414
author: claude
type: comment
created: 2026-02-05T09:53:34.238095Z
---
## Prompting Best Practices for Copilot CLI

### File-Based Instructions

**Repository-Wide**: `.github/copilot-instructions.md`
- Applies to all requests in the repository
- Equivalent to Cortex's `CLAUDE.md`

**Agent-Specific**: `AGENTS.md` (in any directory)
- Nearest file in directory tree takes precedence
- Can also use `CLAUDE.md` or `GEMINI.md` in root
- This is how we'd deliver Cortex role prompts

**Path-Specific**: `.github/instructions/NAME.instructions.md`
- Uses frontmatter with `applyTo` glob patterns
- Good for language/framework-specific guidance

### Effective Instruction Content

Per GitHub's documentation, include:
1. **Project overview**: Size, languages, frameworks
2. **Build procedures**: Bootstrap, build, test, lint commands
3. **Architecture**: Major components, file locations
4. **Validation**: CI/CD workflows, testing procedures
5. **Dependencies**: Non-obvious requirements
6. **Troubleshooting**: Known errors and workarounds

### Best Practices
- Keep under 2 pages
- Non-task-specific (general guidance)
- Natural language Markdown
- Document commands with versions

### Non-Interactive Mode Prompting

```bash
# Simple task
copilot -p "Fix the bug in auth.ts" --yolo

# With tool permissions
copilot -p "Refactor the API module" --allow-all-tools

# With specific permissions
copilot -p "Run tests" --allow-tool 'shell(npm:*)'

# Pipe context in
cat error.log | copilot -p "Analyze this error"
```

### Cortex Integration Approach

For Cortex ticket agents, generate temporary instructions:
1. Write system prompt to `.cortex/prompts/AGENTS.md` (or temp location)
2. Use `--add-dir` to grant access to project paths
3. Use `--additional-mcp-config` to inject Cortex MCP server
4. Add `--yolo` for automated execution
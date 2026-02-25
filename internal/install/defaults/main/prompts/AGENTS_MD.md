# Task

Explore the following repositories and generate a high-level architecture overview:

{{range .Repos}}
- {{.}}
{{end}}

## Requirements

Write a file called `{{.OutputFile}}` at `{{.ArchitectRoot}}` containing:

1. **Project Overview**: What each repository does and how they relate
2. **Tech Stack**: Languages, frameworks, key dependencies
3. **Architecture**: High-level structure, main components, how they connect
4. **Key Patterns**: Important conventions, coding patterns, or architectural decisions

Keep it concise and focused on helping an AI agent understand the codebase quickly. Do NOT include implementation details — those belong in repo-level documentation.

Write the file now.

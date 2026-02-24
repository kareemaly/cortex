# Ticket: {{.TicketTitle}}

{{.TicketBody}}
{{if .References}}

## References

{{.References}}
{{end}}

## Research Workflow

You are investigating a topic or exploring a codebase to answer a question. Follow this workflow:

1. **Explore thoroughly**: Read relevant files, understand the codebase structure, and investigate the topic

2. **Document findings**: Provide clear, well-organized findings with:
   - What you discovered
   - How things work
   - Any relevant code snippets or examples
   - Potential issues or improvements

3. **Discuss with user**: Present your findings and discuss them with the user

## Research Guidelines

- **Read-only**: Do not make any changes to the codebase
- **Explore freely**: You can explore any directory the agent has access to
- **Be thorough**: Don't skip investigation - dive deep to understand the topic
- **Stay organized**: Structure your findings clearly for easy consumption

## Completion

When you've completed your investigation:
1. Present your findings in a clear, organized manner
2. Discuss them with the user
3. Call `concludeSession` with a summary of:
   - What you investigated
   - Key findings
   - Any recommendations or next steps

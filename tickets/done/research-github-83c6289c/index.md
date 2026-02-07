---
id: 83c6289c-e87a-43e3-820f-bae5452ae6d7
title: Research GitHub Copilot CLI integration
type: research
created: 2026-02-05T09:47:20.372342Z
updated: 2026-02-05T10:03:08.597015Z
---
## Research Question

How can we integrate GitHub Copilot CLI into the Cortex workflow, and what are the best practices for prompting it?

## Context

Copilot CLI is installed on the system. We want to understand:
- Its capabilities and how it could complement or integrate with Cortex agents
- Best practices for prompting (system prompts, agent configuration)
- Whether it could serve as an alternative agent type

## Areas to Investigate

### 1. Copilot CLI Capabilities
- Run `copilot --help` and explore available commands
- Understand its modes of operation (chat, suggestions, etc.)
- What inputs/outputs does it support?

### 2. Source Code Exploration
- Clone https://github.com/github/copilot-cli to `~/ephemeral/copilot-cli`
- Explore how it handles prompts, configuration, and context
- Look for any agent/MCP patterns we could leverage

### 3. Integration Possibilities
- Could Copilot CLI be a Cortex agent type alongside Claude?
- What configuration would be needed in `.cortex/cortex.yaml`?
- How would prompts/defaults differ from Claude agents?

### 4. Prompting Best Practices
- How does Copilot CLI expect system prompts?
- Are there defaults or templates we should adopt?
- Any recommended patterns for coding tasks?

## Deliverables

- Summary of Copilot CLI capabilities relevant to Cortex
- Recommended integration approach (if feasible)
- Best practices for Copilot prompting
- List of changes needed to support Copilot as an agent type
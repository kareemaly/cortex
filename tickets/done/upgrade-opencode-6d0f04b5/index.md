---
id: 6d0f04b5-9cd8-4002-a0fd-1728a7bbe432
title: Upgrade OpenCode and test Cortex agent spawn with GPT 5.2
type: chore
tags:
    - opencode
    - chore
references:
    - doc:cb80accd
    - doc:b111544a-08d4-4153-a2f9-dddea1a025ea
created: 2026-02-11T09:31:44.302355Z
updated: 2026-02-11T10:09:44.219346Z
---
## Objective

Upgrade OpenCode (`opencode-ai` npm package) to the latest version, then validate the Cortex integration approach by running OpenCode with a custom agent definition and system prompt.

## Steps

1. **Upgrade OpenCode** — Update `opencode-ai` to the latest version (currently on v1.1.18, latest is v1.1.56+). Use npm or whatever package manager is appropriate.

2. **Verify the upgrade** — Run `opencode --version` to confirm the new version.

3. **Test a Cortex-style agent spawn** — Run OpenCode with:
   - A custom agent named `cortex` defined via `OPENCODE_CONFIG_CONTENT` or an agent markdown file
   - A simple system prompt simulating an architect role (e.g., "You are a project architect. You plan and delegate work through tickets.")
   - Model set to GPT 5.2 (`openai/gpt-5.2` or whatever the correct provider/model format is)
   - Use `opencode run --agent cortex "Hello, describe your role"` or similar to verify the agent loads correctly with the custom prompt and model

4. **Document what worked and what didn't** — Note any issues with the agent config format, model identifier, or prompt injection.

## References

- Research doc on system prompt injection: doc:cb80accd
- Research doc on OpenCode config: doc:b111544a

## Acceptance criteria

- OpenCode is upgraded to latest version
- A test run completes successfully with a custom agent, custom system prompt, and GPT 5.2 model
- Any issues or format corrections are noted in a comment
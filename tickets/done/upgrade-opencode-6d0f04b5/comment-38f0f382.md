---
id: 38f0f382-f76a-4eef-8d3f-db7cab7f8051
author: claude
type: comment
created: 2026-02-11T09:34:08.994071Z
---
## Results

### 1. Upgrade
- Upgraded `opencode-ai` from **v1.1.18** to **v1.1.56** (latest).
- Note: nvm's npm had a misconfigured global prefix pointing to pyenv's nodejs_wheel. Required `--prefix` override to install into the correct nvm path.

### 2. Model identifier
- GPT 5.2 is available as `openai/gpt-5.2`. Other variants: `openai/gpt-5.2-chat-latest`, `openai/gpt-5.2-codex`, `openai/gpt-5.2-pro`.

### 3. Agent spawn tests — both approaches worked

**Method A: `OPENCODE_CONFIG_CONTENT` env var (inline JSON)**
```bash
OPENCODE_CONFIG_CONTENT='{"agent":{"cortex":{"description":"Cortex architect","mode":"primary","model":"openai/gpt-5.2","prompt":"You are a project architect...","permission":{"*":"allow"}}}}' \
  opencode run --agent cortex "Hello, describe your role"
```
- Custom agent loaded correctly as `cortex · gpt-5.2`
- System prompt injected and reflected in response
- No permission issues

**Method B: Agent markdown file (`.opencode/agents/cortex.md`)**
```markdown
---
description: Cortex ticket agent
mode: primary
model: openai/gpt-5.2
permission:
  "*": allow
---
You are a Cortex ticket agent...
```
- Also worked perfectly — agent loaded with correct model and prompt

### 4. Issues / Notes
- `OPENCODE_CONFIG_CONTENT` does NOT support `{file:path}` or `{env:VAR}` substitution (uses raw `JSON.parse`). The full prompt must be inlined.
- Agent markdown files are the cleaner approach for persistent agent configs.
- Environment info (cwd, platform, date) is always appended to the system prompt regardless of method — cannot be suppressed without a plugin hook.
- The `mode: "primary"` field is required for agents used with `--agent` flag; subagents are ignored with a fallback warning.
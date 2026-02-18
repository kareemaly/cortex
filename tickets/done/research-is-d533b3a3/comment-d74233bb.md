---
id: d74233bb-13a8-4371-8273-8412e66f8e18
author: claude
type: review_requested
created: 2026-02-14T11:58:21.957599Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Research Complete: OpenCode Default Prompt Accessibility

### Answer

**No — `anthropic.txt` is NOT accessible as a file on disk from the installed OpenCode package.**

The prompt is compiled into a Bun-native Mach-O binary at build time via ES6 `import ... from "*.txt"` syntax. The installed npm package (`opencode-ai`) ships only 6 files: a wrapper script, platform binary, license, and metadata. No source files or `.txt` prompts exist on disk.

### Key Evidence

1. **Source import pattern** (`system.ts:5`): `import PROMPT_ANTHROPIC from "./prompt/anthropic.txt"` — bundled as a string constant
2. **Build system** (`script/build.ts`): `Bun.build()` with `compile: true` inlines all text into the native binary
3. **Installed package verification**: `find` on the npm package directory returns zero `.txt` files
4. **Binary analysis**: The prompt IS inside the binary (extractable via `strings`), but parsing binary dumps is too fragile for production

### Recommendation

**Use OpenCode's instruction file mechanism (Approach 4)** instead of setting `agent.cortex.prompt`:
- Don't set `agent.prompt` at all → OpenCode keeps its built-in `anthropic.txt`
- Pass Cortex's SYSTEM.md via instruction files → appended after the provider prompt
- Zero drift, zero maintenance, clean separation of concerns

**Fallback if Approach 4 has blockers:** Embed a static copy in Cortex (Approach 2) + fetch from GitHub during `cortex defaults upgrade` (Approach 3) to stay current.

### Docs Created

- **"OpenCode Default Prompt Accessibility from Installed Package"** (`findings/`) — Full investigation with binary analysis, 4 approaches with trade-offs, and source references
- Builds on prior doc **"OpenCode System Prompt: Append vs Replace Analysis"** (`opencode-system-324cb419`)
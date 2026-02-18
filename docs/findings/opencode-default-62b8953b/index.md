---
id: 62b8953b-c6de-47ee-8a58-4a07188361d4
title: OpenCode Default Prompt Accessibility from Installed Package
tags:
    - opencode
    - system-prompt
    - bundling
    - bun
    - npm
created: 2026-02-14T11:58:08.904328Z
updated: 2026-02-14T11:58:08.904328Z
---
## TL;DR

**No — `anthropic.txt` is NOT accessible as a readable file from the installed OpenCode package.** It is compiled into a Bun-native Mach-O binary at build time. The installed npm package contains only 6 files (a wrapper script, a platform-specific binary, and metadata). There is no runtime filesystem path where Cortex can read the prompt.

---

## Investigation Details

### What the Installed Package Contains

OpenCode is installed via npm as `opencode-ai`. The package at `~/.nvm/versions/node/v22.14.0/lib/node_modules/opencode-ai/` contains:

| File | Description |
|------|-------------|
| `LICENSE` | License file |
| `package.json` | npm package manifest |
| `postinstall.mjs` | Post-install script (downloads platform binary) |
| `bin/opencode` | Node.js wrapper script |
| `node_modules/opencode-darwin-arm64/bin/opencode` | Native Mach-O arm64 binary (101MB) |
| `node_modules/opencode-darwin-arm64/package.json` | Platform package manifest |

**That's it.** No source files, no `.txt` prompt files, no `src/` directory. The `find` command for `anthropic.txt` returns zero results.

### How the Prompt Gets Compiled In

In the OpenCode source (`~/ephemeral/opencode`):

1. **Import** — `src/session/system.ts:5`:
   ```typescript
   import PROMPT_ANTHROPIC from "./prompt/anthropic.txt"
   ```

2. **Type declaration** — `.opencode/env.d.ts`:
   ```typescript
   declare module "*.txt" {
     const content: string
     export default content
   }
   ```

3. **Build** — `script/build.ts:140-164` uses `Bun.build()` with `compile: true`, which:
   - Bundles all imports including `.txt` files as string constants
   - Compiles to a native Mach-O executable (not a JS file)
   - Inlines all text as JavaScript template literals in the binary

4. **Distribution** — Platform-specific binaries are published as separate npm packages (`opencode-darwin-arm64`, `opencode-linux-x64`, etc.) and downloaded via `postinstall.mjs`.

### Binary Analysis

The prompt IS present inside the binary as embedded strings:

```
$ strings <binary> | grep -c 'var anthropic_default'
2   # appears twice (bundler artifact)

$ strings <binary> | grep -c 'You are OpenCode'
4   # appears in multiple prompt variants
```

The full prompt text (~72 lines, 8.2KB) can be extracted using `strings` + pattern matching, but this is **fragile and not recommended** for production use.

---

## Approaches for Cortex to Access the Prompt

### Approach 1: Extract from binary via `strings` (Fragile — NOT recommended)

```bash
strings $(readlink -f $(which opencode)) | awk '/^var anthropic_default/{found=1; next} found && /^var /{exit} found{print}'
```

**Pros:** Dynamic, always matches installed version
**Cons:** Extremely fragile — depends on Bun's internal compilation format, binary layout, string encoding. Will break across versions. Not cross-platform reliable. Parsing JS from binary dumps is not production-quality.

### Approach 2: Embed a static copy in Cortex (Practical — recommended with mitigation)

Keep a copy of `anthropic.txt` in Cortex's embedded defaults and prepend it to SYSTEM.md for OpenCode agents.

**Pros:** Reliable, predictable, testable
**Cons:** Drifts from upstream over time
**Mitigation:** 
- Pin to a known OpenCode version in docs
- Add a `cortex defaults upgrade` check that fetches the latest from OpenCode's GitHub raw URL
- The prompt changes infrequently — OpenCode has only had 2 versions of it (`anthropic.txt` and `anthropic-20250930.txt`)

### Approach 3: Fetch from GitHub at init/upgrade time (Semi-dynamic)

During `cortex init` or `cortex defaults upgrade`, fetch the latest `anthropic.txt` from GitHub:

```
https://raw.githubusercontent.com/anomalyco/opencode/main/packages/opencode/src/session/prompt/anthropic.txt
```

Store it at `~/.cortex/defaults/main/prompts/opencode-base-prompt.txt` and prepend at spawn time.

**Pros:** Stays current with upstream; user can customize via eject
**Cons:** Requires network access; adds GitHub dependency; URL may change
**Mitigation:** Fall back to embedded copy if fetch fails.

### Approach 4: Use OpenCode's instruction file mechanism (Cleanest architecture)

Instead of setting `agent.cortex.prompt` (which replaces the provider prompt), write the Cortex SYSTEM.md to a file and reference it as an instruction file in the OpenCode config. This lets OpenCode's built-in `anthropic.txt` remain as the base prompt and Cortex's instructions get appended via `InstructionPrompt.system()`.

**Pros:** Zero drift — OpenCode's prompt updates automatically; clean separation of concerns
**Cons:** Less control over ordering; instructions appear in "Instructions from: /path" wrapper; may interact with other instruction files; requires understanding OpenCode's instruction file precedence

**Implementation:** In `GenerateOpenCodeConfigContent()`, instead of:
```json
{ "agent": { "cortex": { "prompt": "<system.md>" } } }
```
Use:
```json
{ "agent": { "cortex": { "instructions": ["/tmp/cortex-system-prompt.md"] } } }
```
Or write to `.opencode/agents.md` in the project root.

---

## Recommendation

**Approach 4 (instruction files) is the cleanest long-term solution** — it completely eliminates the drift problem by letting OpenCode own its default prompt while Cortex appends its workflow instructions.

If Approach 4 has technical blockers (e.g., OpenCode's instruction file mechanism doesn't support the needed ordering or formatting), **Approach 2 + 3 hybrid** is the pragmatic fallback: embed a static copy for offline reliability, fetch from GitHub during `cortex defaults upgrade` to stay current.

---

## Key Source References

| Location | What |
|----------|------|
| `opencode/src/session/system.ts:5` | Import of anthropic.txt |
| `opencode/src/session/system.ts:19-26` | Provider prompt selection (claude → PROMPT_ANTHROPIC) |
| `opencode/src/session/llm.ts:66-79` | Agent prompt REPLACES provider prompt |
| `opencode/script/build.ts:140-164` | Bun.build() with compile:true inlines all text |
| `opencode/.opencode/env.d.ts` | Module declaration for *.txt imports |
| `cortex1/internal/core/spawn/opencode_config.go:32-65` | Where Cortex sets agent.prompt |
| Prior findings doc: `opencode-system-324cb419` | Detailed append-vs-replace analysis |

## OpenCode Version Tested

- **Version:** 1.1.56
- **Binary format:** Mach-O 64-bit arm64 (Bun-compiled)
- **Prompt file:** `anthropic.txt` — 105 lines, ~8.2KB
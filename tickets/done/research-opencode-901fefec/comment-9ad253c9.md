---
id: 9ad253c9-719c-4c70-abd3-085b77724298
author: claude
type: review_requested
created: 2026-02-13T13:16:00.160425Z
action:
    type: git_diff
    args:
        repo_path: /Users/kareemelbahrawy/projects/cortex1
---
## Research Complete: OpenCode Plugin Injection via Config & Temp Directory

### Key Findings

All 6 research questions answered. Created comprehensive findings doc.

**The short answer:** Use `OPENCODE_CONFIG_DIR` env var pointing to a session-unique temp directory containing `plugin/cortex-status.ts`. This is the cleanest, most idiomatic approach.

### Answers Summary

1. **`OPENCODE_CONFIG_CONTENT` cannot set a plugin directory path** — but it can include `file://` URLs in its `plugin` array as a fallback
2. **`OPENCODE_CONFIG_DIR` is the ideal override** — it adds a directory to the plugin scan list, scanning `{dir}/plugin/*.{ts,js}`
3. **Plugin loading uses both scan + explicit registration** — they converge into the same pipeline
4. **TypeScript and JavaScript both supported** — no compilation needed (Bun runtime)
5. **No external dependencies required** — if the plugin avoids `@opencode-ai/plugin` imports, no install step is triggered
6. **Minimal plugin is ~20 lines** — export default async function returning a `Hooks` object with an `event` handler

### Recommendation

**Approach A: `OPENCODE_CONFIG_DIR` with untyped exports**

- Create temp dir `/tmp/cortex-opencode-{sessionID}/plugin/cortex-status.ts`
- Set `OPENCODE_CONFIG_DIR` to the temp dir
- Make dir read-only after writing to skip dependency install overhead
- Plugin reports `session.status`, `session.idle`, and `permission.asked` events back to Cortex daemon via HTTP
- Track temp dir in `CleanupFiles` for automatic cleanup

### Open Questions for Implementation

1. Need a new daemon endpoint for status updates from the plugin
2. Fire-and-forget vs retry on HTTP failures
3. Whether to also use the temp dir for other per-session config (agents, commands)

### Doc Created

- "OpenCode Plugin Injection via Config & Temp Directory" in docs/findings/ (tagged: opencode, plugins, spawn, status)
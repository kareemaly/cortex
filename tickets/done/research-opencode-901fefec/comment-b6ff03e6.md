---
id: b6ff03e6-b6fd-4f89-91f3-39ac45c77c49
author: claude
type: done
created: 2026-02-13T13:16:39.957076Z
---
## Research Complete: OpenCode Plugin Injection via Config & Temp Directory

### Key Findings

1. **`OPENCODE_CONFIG_CONTENT` cannot define a plugin directory path** — it only accepts npm package names or `file://` URLs in its `plugin` array. No `pluginDir` config key exists.

2. **`OPENCODE_CONFIG_DIR` is the ideal injection mechanism** — This env var adds a custom directory to OpenCode's plugin scan list. OpenCode scans `{dir}/plugin/*.{ts,js}` and `{dir}/plugins/*.{ts,js}` using Bun.Glob with `followSymlinks: true`.

3. **Plugin loading uses two parallel mechanisms** — directory scanning (auto-discovery) and explicit registration (config `plugin` array with npm packages or `file://` URLs). Both converge in `plugin/index.ts`.

4. **TypeScript and JavaScript supported natively** — Bun runtime executes `.ts` files directly. No compilation step required.

5. **Dependency install can be skipped** — OpenCode auto-installs `@opencode-ai/plugin` into plugin directories, but checks `isWritable(dir)` first. Making the temp dir read-only after writing the plugin file skips the 3+ second install. The plugin must avoid importing from `@opencode-ai/plugin` (just export the right object shape).

6. **Minimal plugin is ~20 lines** — Export default async function returning a `Hooks` object. The `event` hook receives all SSE events including `session.status` (idle/busy/retry), `session.idle`, `permission.asked`, and `permission.replied`.

### Recommended Approach

**`OPENCODE_CONFIG_DIR` with untyped exports (Approach A, Option 3):**
- Create temp dir `/tmp/cortex-opencode-{sessionID}/plugin/cortex-status.ts`
- Set `OPENCODE_CONFIG_DIR` env var to the temp dir
- Make dir read-only after writing to skip dependency install
- Plugin reports status events back to Cortex daemon via HTTP POST
- Track temp dir in `CleanupFiles` for automatic cleanup on session end

### Recommendations for Next Steps

1. **Implement the spawn-time plugin injection** — Modify `internal/core/spawn/spawn.go` to create the temp dir, write the plugin template, set `OPENCODE_CONFIG_DIR`, and register for cleanup.

2. **Design a daemon status endpoint** — Create `POST /tickets/{status}/{id}/status` (or similar) to receive status updates from the plugin. This is the HTTP target the plugin calls.

3. **Add `Plugin` field to `OpenCodeConfigContent`** — As a fallback path, add `Plugin []string` to the struct in `opencode_config.go` so `file://` URLs can be passed via `OPENCODE_CONFIG_CONTENT` if `CONFIG_DIR` causes issues.

4. **Test with a real OpenCode session** — Verify the read-only dir trick works, confirm plugin discovery order, and measure any startup latency impact.

### Artifacts Created

- Findings doc: "OpenCode Plugin Injection via Config & Temp Directory" (tags: opencode, plugins, spawn, status) — contains full comparison matrix, plugin templates, implementation sketches, and open questions.
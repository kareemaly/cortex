---
id: 5208ef3a-80f4-4da4-a1ce-217c05bc1eb5
title: OpenCode Plugin Injection via Config & Temp Directory
tags:
    - opencode
    - plugins
    - spawn
    - status
created: 2026-02-13T13:15:29.278459Z
updated: 2026-02-13T13:15:29.278459Z
---
## Summary

This document details how Cortex can inject a status-reporting plugin into OpenCode at spawn time using ephemeral temp directories. Two viable mechanisms exist; **Approach A (`OPENCODE_CONFIG_DIR`)** is recommended.

---

## Research Answers

### Q1: Can `OPENCODE_CONFIG_CONTENT` define a custom plugins directory path?

**No.** `OPENCODE_CONFIG_CONTENT` is parsed as JSON and merged into the config object. It supports the `plugin` array field, but this array accepts **npm package names or `file://` URLs** — not directory paths. There is no `pluginDir` or `pluginPath` config key.

However, `OPENCODE_CONFIG_CONTENT` *can* include `file://` URLs that point to plugin files in a temp directory (see Approach B below).

**Source:** `packages/opencode/src/config/config.ts:177-181`

### Q2: Does OpenCode support overriding the default plugin directory?

**Yes — via `OPENCODE_CONFIG_DIR` env var.** This is a dynamically-evaluated env var that adds a custom directory to the plugin scan list. OpenCode scans `{dir}/plugin/*.{ts,js}` and `{dir}/plugins/*.{ts,js}` for every directory in its scan list.

```
OPENCODE_CONFIG_DIR=/tmp/cortex-session-abc123
```

OpenCode will then scan `/tmp/cortex-session-abc123/plugin/*.{ts,js}`.

**Source:** `packages/opencode/src/flag/flag.ts:76-82`, `packages/opencode/src/config/config.ts:145-148,456-469`

### Q3: Plugin loading mechanism — scan vs explicit registration?

**Both.** OpenCode uses two mechanisms in parallel:

1. **Directory scanning** — Scans `{plugin,plugins}/*.{ts,js}` in each config directory (global, project `.opencode/`, `OPENCODE_CONFIG_DIR`). Uses `Bun.Glob` with `followSymlinks: true`.

2. **Explicit registration** — The `plugin` array in config (from any config source including `OPENCODE_CONFIG_CONTENT`) lists npm packages or `file://` URLs.

Scanned file plugins are converted to `file://` URLs and pushed onto the same `plugin` array. Both paths converge in `plugin/index.ts` where each entry is imported and initialized.

**Source:** `packages/opencode/src/config/config.ts:456-469,174`, `packages/opencode/src/plugin/index.ts:48-93`

### Q4: Plugin file format?

**TypeScript (`.ts`) or JavaScript (`.js`).**

Glob pattern: `{plugin,plugins}/*.{ts,js}`

No compilation step required — Bun has native TypeScript support.

### Q5: Runtime dependencies?

**Bun handles everything.** OpenCode runs on Bun which natively executes TypeScript. When OpenCode detects a plugin directory, it:

1. Auto-creates `package.json` with `@opencode-ai/plugin` dependency
2. Auto-creates `.gitignore` (node_modules, package.json, bun.lock)
3. Runs `bun install` to install the `@opencode-ai/plugin` SDK

**Caveat:** The auto-install checks `isWritable(dir)` first and skips if the directory is read-only. The temp directory **must be writable** for auto-install to work. There's also a 3-second delay built into the install process (`setTimeout(resolve, 3000)`).

If the directory is read-only or we want to avoid the install overhead, the plugin can avoid importing from `@opencode-ai/plugin` and just export the right shape.

**Source:** `packages/opencode/src/config/config.ts:252-280,291-294`

### Q6: Minimal plugin structure?

```typescript
// cortex-status.ts — Minimal Cortex status plugin
// No imports needed if we avoid @opencode-ai/plugin types

export default async (ctx) => {
  return {
    // Subscribe to all events for status reporting
    event: async ({ event }) => {
      // event.type is one of: "session.status", "session.idle",
      // "message.updated", "permission.asked", etc.
      
      if (event.type === "session.status") {
        const { sessionID, status } = event.properties
        // status.type is "idle" | "busy" | "retry"
        // Report to Cortex daemon via HTTP
        await fetch(`http://localhost:4200/sessions/status`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ sessionID, status: status.type }),
        })
      }
    },
  }
}
```

The plugin function receives a `PluginInput` context:
```typescript
{
  client: OpencodeSDKClient,  // SDK for OpenCode's internal API
  project: Project,           // Project metadata
  directory: string,          // Working directory
  worktree: string,           // Git worktree root
  serverUrl: URL,             // OpenCode's internal server URL
  $: BunShell,                // Bun shell for running commands
}
```

Available hooks for status reporting:

| Hook | Use Case |
|------|----------|
| `event` | Subscribe to all SSE events (session.status, session.idle, permission.asked, etc.) |
| `tool.execute.before` | Fires before each tool execution |
| `tool.execute.after` | Fires after each tool execution |
| `chat.message` | Fires on new messages |
| `permission.ask` | Fires when permission is requested (maps to Cortex "waiting" state) |

Key `SessionStatus` types: `idle`, `busy`, `retry`.

---

## Recommended Approach: A — `OPENCODE_CONFIG_DIR` with Temp Directory

### How It Works

1. At spawn time, Cortex creates a unique temp directory: `/tmp/cortex-opencode-{sessionID}/`
2. Creates `plugin/cortex-status.ts` inside it
3. Sets `OPENCODE_CONFIG_DIR=/tmp/cortex-opencode-{sessionID}` as an env var
4. OpenCode auto-discovers and loads the plugin

### Implementation in Cortex

In `internal/core/spawn/spawn.go`, after setting `OPENCODE_CONFIG_CONTENT`:

```go
// Create temp dir for plugin injection
tmpDir, _ := os.MkdirTemp("", "cortex-opencode-")
pluginDir := filepath.Join(tmpDir, "plugin")
os.MkdirAll(pluginDir, 0755)

// Write the status plugin
pluginContent := generateStatusPlugin(daemonAddr, ticketID)
os.WriteFile(filepath.Join(pluginDir, "cortex-status.ts"), []byte(pluginContent), 0644)

// Set env var
launcherParams.EnvVars["OPENCODE_CONFIG_DIR"] = tmpDir

// Track for cleanup
launcherParams.CleanupFiles = append(launcherParams.CleanupFiles, tmpDir)
```

### Pros
- Clean separation — plugin lives in its own directory
- Follows OpenCode's intended extension mechanism
- Auto-discovery means no config changes needed to `OPENCODE_CONFIG_CONTENT`
- Supports symlinks (`followSymlinks: true` in glob scan)
- Temp dir is ephemeral and unique per session
- Can also hold other per-session config (agents, commands) in the same dir

### Cons
- OpenCode auto-installs `@opencode-ai/plugin` dependency (adds ~3s startup time)
- Directory must be writable
- Plugin TypeScript must avoid imports from `@opencode-ai/plugin` OR accept the install overhead

### Mitigating Install Overhead

Option 1: **Pre-seed `node_modules`** — Include a pre-built `node_modules/@opencode-ai/plugin` in the temp dir so `needsInstall()` returns false.

Option 2: **Make dir read-only after writing** — Set directory permissions to read-only after creating the plugin file, so `isWritable()` returns false and install is skipped. The plugin must not import from `@opencode-ai/plugin` in this case.

Option 3: **Use untyped exports** — Write the plugin without any imports from `@opencode-ai/plugin`. Just export the right object shape. Bun will execute it fine.

**Recommended: Option 3** — simplest, no install overhead, no type-checking at runtime anyway.

---

## Alternative Approach: B — `OPENCODE_CONFIG_CONTENT` with `file://` URLs

### How It Works

1. At spawn time, write plugin to a temp file: `/tmp/cortex-status-{sessionID}.ts`
2. Add `"plugin": ["file:///tmp/cortex-status-{sessionID}.ts"]` to the `OPENCODE_CONFIG_CONTENT` JSON

### Implementation

Modify `OpenCodeConfigContent` struct in `opencode_config.go`:

```go
type OpenCodeConfigContent struct {
    Agent  map[string]OpenCodeAgentConfig `json:"agent"`
    MCP    map[string]OpenCodeMCPConfig   `json:"mcp"`
    Plugin []string                        `json:"plugin,omitempty"`
}
```

Then add the file:// URL:
```go
config.Plugin = []string{
    fmt.Sprintf("file:///tmp/cortex-status-%s.ts", sessionID),
}
```

### Pros
- No directory structure needed — single file
- Plugin path is explicit in config
- No auto-install behavior (file:// URLs skip npm install)

### Cons
- Less clean — mixes plugin registration with MCP/agent config
- Can't co-locate other extension files
- `OPENCODE_CONFIG_CONTENT` is already getting large with agent + MCP config

---

## Alternative Approach: C — Symlink into Project `.opencode/`

### How It Works
1. Write plugin to temp file
2. Create `.opencode/plugin/` in project if it doesn't exist
3. Symlink plugin into it

### Why Not
- Modifies the project directory (dirty git state)
- Cleanup is fragile
- Conflicts with user's own plugins
- Not ephemeral

**Verdict: Not recommended.**

---

## Comparison Matrix

| Criterion | A: CONFIG_DIR | B: file:// URL | C: Symlink |
|-----------|:---:|:---:|:---:|
| Ephemeral & unique per session | Yes | Yes | No |
| No project dir modification | Yes | Yes | No |
| Follows OpenCode conventions | Best | Good | Poor |
| No install overhead (with Option 3) | Yes | Yes | N/A |
| Extensible (can add agents, commands) | Yes | No | No |
| Implementation complexity | Low | Low | Medium |
| Cleanup simplicity | rm -rf tmpDir | rm tmpFile | Fragile |

---

## Status Plugin Design

The plugin should report Cortex-relevant state transitions:

| OpenCode Event | Cortex Status |
|----------------|---------------|
| `session.status` → `busy` | `working` |
| `session.status` → `idle` | `idle` |
| `session.status` → `retry` | `error` (transient) |
| `permission.asked` | `waiting` |
| `permission.replied` | `working` |
| `session.idle` | `idle` |

The plugin communicates back to Cortex daemon via HTTP POST to a new endpoint (e.g., `POST /sessions/{id}/status`). The daemon address and ticket/session IDs are baked into the plugin file at generation time.

### Minimal Status Plugin Template

```typescript
export default async () => ({
  event: async ({ event }) => {
    const DAEMON = "__DAEMON_ADDR__"
    const TICKET = "__TICKET_ID__"
    const PROJECT = "__PROJECT_PATH__"
    
    let status
    switch (event.type) {
      case "session.status":
        status = event.properties.status.type === "busy" ? "working" 
               : event.properties.status.type === "idle" ? "idle"
               : "error"
        break
      case "session.idle":
        status = "idle"
        break
      case "permission.asked":
        status = "waiting"
        break
      case "permission.replied":
        status = "working"
        break
      default:
        return
    }
    
    try {
      await fetch(`${DAEMON}/tickets/progress/${TICKET}/status`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Cortex-Project": PROJECT,
        },
        body: JSON.stringify({ status }),
      })
    } catch {}
  },
})
```

Cortex generates this at spawn time with string replacement for `__DAEMON_ADDR__`, `__TICKET_ID__`, and `__PROJECT_PATH__`.

---

## Open Questions

1. **Daemon endpoint** — Need to design/implement `POST /tickets/{status}/{id}/status` or similar endpoint for status updates from the plugin.

2. **Session ID mapping** — The plugin receives OpenCode's internal `sessionID` but needs to map to Cortex's ticket ID. Easiest to bake the Cortex ticket ID directly into the plugin template.

3. **Install overhead** — If we go with Approach A + Option 3 (untyped exports), confirm that OpenCode still triggers `needsInstall` for the `OPENCODE_CONFIG_DIR` directory. If so, we may need to pre-seed `package.json` with the right dependency to avoid the install, OR make the dir read-only.

4. **Plugin error handling** — If the Cortex daemon is unreachable, the plugin should fail silently (the `catch {}` handles this). Should we add retry logic or is fire-and-forget sufficient?

5. **Cleanup timing** — The temp directory must persist for the lifetime of the OpenCode session. The `CleanupFiles` mechanism in spawn already handles this on session termination.

---

## Recommendation

**Use Approach A (`OPENCODE_CONFIG_DIR`) with Option 3 (untyped exports).** This is the cleanest mechanism:

- Set `OPENCODE_CONFIG_DIR` to a session-unique temp directory
- Write `plugin/cortex-status.ts` with status reporting logic (no imports)
- Make the directory read-only after writing to skip dependency installation
- Track the temp directory in `CleanupFiles` for automatic cleanup on session end

The `file://` URL approach (B) is a viable fallback if `OPENCODE_CONFIG_DIR` causes unexpected issues with other config aspects (since it also affects agent/command loading from that directory).
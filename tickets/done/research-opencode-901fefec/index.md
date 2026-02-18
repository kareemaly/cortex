---
id: 901fefec-8767-41fb-a1dc-740a5783fa3b
title: 'Research: OpenCode plugin injection via config and temp directory'
type: research
tags:
    - research
    - opencode
    - agents
    - hooks
created: 2026-02-13T13:10:44.638452Z
updated: 2026-02-13T13:16:39.958064Z
---
## Goal

Determine how to inject a Cortex status plugin into OpenCode at spawn time using a temp directory, keeping everything ephemeral and unique per session.

## Research Questions

1. Can `OPENCODE_CONFIG_CONTENT` define a custom plugins directory path (e.g., pointing to a temp dir)?
2. If not, does OpenCode support a CLI flag, env var, or config key to override the default `.opencode/plugins/` location?
3. What is the exact plugin loading mechanism — does OpenCode scan a directory, or does it require explicit plugin registration in config?
4. What file format does the plugin need to be? (`.ts`, `.js`, compiled?)
5. Does OpenCode need any runtime dependencies to execute a TypeScript plugin (e.g., does it bundle a TS runtime)?
6. What is the minimal plugin structure required for hook registration?

## Resources

- OpenCode source code is cloned at `~/ephemeral/opencode` — explore the plugin loading code there
- Existing research docs in Cortex docs (tagged `opencode`) have background on config injection

## Acceptance Criteria

- Clear answer on whether plugin path is configurable via `OPENCODE_CONFIG_CONTENT` or other mechanism
- If configurable: document the exact config key/format
- If not configurable: propose alternative approaches (e.g., symlink, wrapper script, monkey-patching config)
- Document the minimal plugin file structure and any runtime requirements
- Findings captured in a doc
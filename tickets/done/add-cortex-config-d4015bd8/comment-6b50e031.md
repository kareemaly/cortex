---
id: 6b50e031-17cc-4ffd-a2a9-10076a3ac171
author: claude
type: done
created: 2026-02-02T11:45:32.712962Z
---
## Summary

Implemented `cortex config show` CLI command that displays the fully resolved project configuration after all extends are merged.

## Files Created
- `cmd/cortex/commands/config.go` - Parent `config` command structure
- `cmd/cortex/commands/config_show.go` - Show subcommand with `--path` flag support

## Features
- `cortex config show` - displays resolved config for current project
- `cortex config show --path /some/project` - displays config for specified path
- Comment header showing project root and extend chain (when applicable)
- Valid YAML output suitable for copy-paste into config files
- Helpful error messages for missing projects, invalid extend paths, circular references, and parse errors

## Commit
`cc86b14` - feat(cli): add cortex config show command

## Note
Discovered pre-existing bug: extend functionality expects `.cortex/cortex.yaml` in base config paths, but installed defaults at `~/.cortex/defaults/claude-code/` have `cortex.yaml` directly. This is a separate issue to address.
---
id: d52a8163-fccf-4aaa-b67a-d5581c5e48ce
title: 'Add OSS standard files: LICENSE, CODE_OF_CONDUCT, .gitignore improvements'
type: work
tags:
    - oss-readiness
    - docs
created: 2026-02-08T13:11:06.088443Z
updated: 2026-02-08T13:32:32.934346Z
---
Add standard open-source files and polish existing ones.

## 1. LICENSE file
- MIT license is referenced in README.md but no `LICENSE` file exists at project root
- Create `LICENSE` with MIT license text, copyright holder: Kareem Aly

## 2. CODE_OF_CONDUCT.md
- Standard for OSS projects accepting contributions
- Use Contributor Covenant v2.1 (industry standard)

## 3. .gitignore improvements
Add missing patterns:
- `.env*` — environment files with potential secrets
- `*.pid` — PID files from daemon
- `.cortex/logs/` — local log files

## 4. Error wrapping cleanup
~20 instances of `fmt.Errorf("...", val)` without `%w` wrapping, breaking `errors.Is()`/`errors.As()` chains. Most are in:
- `internal/upgrade/`
- `internal/cli/sdk/client.go`
- `cmd/cortex/commands/`

Find all instances and add proper `%w` wrapping where an error is being wrapped.
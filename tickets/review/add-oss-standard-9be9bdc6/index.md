---
id: 9be9bdc6-22a7-4a8d-9d22-474598735774
title: 'Add OSS standard files: LICENSE, CODE_OF_CONDUCT, .gitignore improvements'
type: chore
tags:
    - oss-readiness
    - docs
created: 2026-02-18T08:02:38.303517Z
updated: 2026-02-18T08:07:14.642146Z
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
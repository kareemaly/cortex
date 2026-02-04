# Contributing to Cortex

## Development Setup

### Prerequisites

- Go 1.21+
- golangci-lint

### Build Commands

```bash
make build    # Build binaries to bin/
make lint     # Run linter
make test     # Run unit tests
make install  # Build, install to ~/.local/bin/, codesign (macOS)
```

## Git Hooks

Enable pre-push checks (runs lint and build before each push):

```bash
make setup-hooks
```

To bypass in emergencies:

```bash
git push --no-verify
```

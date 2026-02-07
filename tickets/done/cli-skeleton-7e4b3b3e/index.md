---
id: 7e4b3b3e-d235-4398-954f-1099a6ed532b
title: CLI Skeleton
type: ""
created: 2026-01-19T13:47:31Z
updated: 2026-01-19T13:47:31Z
---
Set up the cortex CLI with Cobra commands and HTTP client for daemon communication.

## Context

The project foundation is complete. Now we need a CLI that can issue commands and communicate with the daemon.

Reference `~/projects/cortex/cmd/cortex0/` for patterns, but simplify per DESIGN.md.

## Requirements

### 1. Dependencies

Add to go.mod:
```
github.com/spf13/cobra
```

### 2. cmd/cortex/main.go

Expand to Cobra root command:

```go
package main

import (
	"os"

	"github.com/kareemaly/cortex1/cmd/cortex/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
```

### 3. cmd/cortex/commands/root.go

Root command setup:

```go
package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cortex",
	Short: "Orchestration layer for AI coding workflows",
	Long: `Cortex is an orchestration layer for AI coding workflows.
It provides file-based ticket management with MCP tools for agent interaction.`,
	// Default action: show help
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags can be added here
	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
}
```

### 4. cmd/cortex/commands/version.go

Version command:

```go
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/kareemaly/cortex1/pkg/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		info := version.Get()
		fmt.Printf("cortex %s\n", info.Version)
		fmt.Printf("  commit:  %s\n", info.Commit)
		fmt.Printf("  built:   %s\n", info.BuildDate)
		fmt.Printf("  go:      %s\n", info.GoVersion)
		fmt.Printf("  platform: %s\n", info.Platform)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
```

### 5. cmd/cortex/commands/kanban.go

Kanban command stub:

```go
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var kanbanCmd = &cobra.Command{
	Use:   "kanban",
	Short: "Open kanban TUI for current project",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("kanban: not implemented yet")
	},
}

func init() {
	rootCmd.AddCommand(kanbanCmd)
}
```

### 6. cmd/cortex/commands/architect.go

Architect command stub:

```go
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var architectCmd = &cobra.Command{
	Use:   "architect",
	Short: "Start or attach to architect session",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("architect: not implemented yet")
	},
}

func init() {
	rootCmd.AddCommand(architectCmd)
}
```

### 7. cmd/cortex/commands/spawn.go

Spawn command stub:

```go
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var spawnCmd = &cobra.Command{
	Use:   "spawn <ticket-id>",
	Short: "Spawn a ticket session",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("spawn: not implemented yet (ticket: %s)\n", args[0])
	},
}

func init() {
	rootCmd.AddCommand(spawnCmd)
}
```

### 8. cmd/cortex/commands/list.go

List command stub:

```go
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List tickets",
	Run: func(cmd *cobra.Command, args []string) {
		status, _ := cmd.Flags().GetString("status")
		if status != "" {
			fmt.Printf("list: not implemented yet (status: %s)\n", status)
		} else {
			fmt.Println("list: not implemented yet")
		}
	},
}

func init() {
	listCmd.Flags().String("status", "", "Filter by status (backlog, progress, done)")
	rootCmd.AddCommand(listCmd)
}
```

### 9. cmd/cortex/commands/session.go

Session command stub:

```go
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var sessionCmd = &cobra.Command{
	Use:   "session <id>",
	Short: "Open session view TUI",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("session: not implemented yet (id: %s)\n", args[0])
	},
}

func init() {
	rootCmd.AddCommand(sessionCmd)
}
```

### 10. cmd/cortex/commands/install.go

Install command stub:

```go
package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Run initial setup",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("install: not implemented yet")
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
```

### 11. internal/cli/sdk/client.go

HTTP client for daemon API:

```go
package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func DefaultClient() *Client {
	return NewClient("http://localhost:4200")
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

func (c *Client) Health() (*HealthResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("decode failed: %w", err)
	}

	return &health, nil
}
```

### 12. Remove .gitkeep files

Delete these files as they're replaced by real code:
- `internal/cli/sdk/.gitkeep`

Keep `internal/cli/tui/.gitkeep` (TUI ticket will handle it).

## Verification

```bash
# Build succeeds
make build

# Help shows all commands
./bin/cortex
# Should show: kanban, architect, spawn, list, session, version, install

# Version works
./bin/cortex version
# Should show version details

# Subcommands exist
./bin/cortex kanban
./bin/cortex architect
./bin/cortex spawn test-123
./bin/cortex list --status=backlog
./bin/cortex session test-456
./bin/cortex install
# All should print "not implemented yet"

# Lint passes
make lint
```

## Notes

- Commands are stubs - they print messages but don't do real work
- SDK client only has Health() for now - more methods added as needed
- No TUI yet - that's a separate ticket
- Global flags (like --config) can be added to root.go later

## Implementation

### Commits
- `803038b` feat: add CLI skeleton with Cobra commands and SDK client

### Key Files Changed
- `cmd/cortex/main.go` - Updated to use Cobra commands
- `cmd/cortex/commands/*.go` - New command files (root, version, kanban, architect, spawn, list, session, install)
- `pkg/version/version.go` - Added `Info` struct and `Get()` function for structured version info
- `internal/cli/sdk/client.go` - HTTP client with `Health()` method for daemon communication
- `go.mod` / `go.sum` - Added spf13/cobra dependency

### Decisions
- Simplified SDK client to return `error` from `Health()` instead of `*HealthResponse` (simpler API, response parsing can be added when needed)
- Used `defer func() { _ = resp.Body.Close() }()` pattern to satisfy errcheck linter
- Stub messages use format "not implemented yet" without command prefix (cleaner output)
- Root command shows help by default via Cobra's built-in behavior (no explicit Run function needed)

### Scope Changes
- None - implemented as specified
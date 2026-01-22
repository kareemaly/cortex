package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kareemaly/cortex/internal/binpath"
	"github.com/kareemaly/cortex/internal/project/config"
	"github.com/kareemaly/cortex/internal/tmux"
	"github.com/spf13/cobra"
)

var architectDetachFlag bool

var architectCmd = &cobra.Command{
	Use:   "architect",
	Short: "Start or attach to architect session",
	Long: `Starts or attaches to the architect session for this project.

The architect lives in tmux window 0 and has full MCP tools for ticket
management and spawning ticket sessions.

Examples:
  cortex architect           # Start or attach to architect
  cortex architect --detach  # Start architect without attaching`,
	Run: runArchitect,
}

func init() {
	architectCmd.Flags().BoolVar(&architectDetachFlag, "detach", false,
		"Spawn architect without attaching to session")
	rootCmd.AddCommand(architectCmd)
}

func runArchitect(cmd *cobra.Command, args []string) {
	// 1. Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	// 2. Find project root and load config
	cfg, projectRoot, err := config.LoadFromPath(cwd)
	if err != nil {
		if config.IsProjectNotFound(err) {
			fmt.Fprintf(os.Stderr, "Error: not a cortex project (no .cortex/ directory found)\n")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// 3. Determine session name
	sessionName := cfg.Name
	if sessionName == "" {
		sessionName = filepath.Base(projectRoot)
	}

	// 4. Create tmux manager
	manager, err := tmux.NewManager()
	if err != nil {
		if tmux.IsNotInstalled(err) {
			fmt.Fprintf(os.Stderr, "Error: tmux is required but not installed\n")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// 5. Check if architect is already running
	architectRunning := checkArchitectRunning(manager, sessionName)

	// 6. If architect running, just attach
	if architectRunning {
		if !architectDetachFlag {
			if err := manager.AttachSession(sessionName); err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to attach to session: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("Architect already running in session: %s\n", sessionName)
		}
		return
	}

	// 7. Spawn architect
	mcpConfigPath, err := generateArchitectMCPConfig(sessionName, projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to generate MCP config: %v\n", err)
		os.Exit(1)
	}

	prompt := buildArchitectPrompt(sessionName)
	agentCmd := buildAgentCommand(cfg.Agent, mcpConfigPath, prompt)

	if err := manager.SpawnArchitect(sessionName, "architect", agentCmd); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to spawn architect: %v\n", err)
		os.Exit(1)
	}

	// 8. Attach unless detach flag
	if !architectDetachFlag {
		if err := manager.AttachSession(sessionName); err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to attach to session: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("Architect started in tmux session: %s\n", sessionName)
	}
}

// checkArchitectRunning checks if the architect is already running in the session.
func checkArchitectRunning(manager *tmux.Manager, sessionName string) bool {
	exists, err := manager.SessionExists(sessionName)
	if err != nil || !exists {
		return false
	}

	windows, err := manager.ListWindows(sessionName)
	if err != nil {
		return false
	}

	for _, w := range windows {
		if w.Index == tmux.ArchitectWindowIndex && w.Name == "architect" {
			return true
		}
	}
	return false
}

// mcpConfig represents the MCP configuration file structure.
type mcpConfig struct {
	MCPServers map[string]mcpServerConfig `json:"mcpServers"`
}

type mcpServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

// generateArchitectMCPConfig creates the MCP config file for the architect session.
// Returns the path to the generated config file.
func generateArchitectMCPConfig(sessionName, projectPath string) (string, error) {
	cortexdPath, err := binpath.FindCortexd()
	if err != nil {
		return "", fmt.Errorf("find cortexd: %w", err)
	}

	cfg := mcpConfig{
		MCPServers: map[string]mcpServerConfig{
			"cortex": {
				Command: cortexdPath,
				Args:    []string{"mcp"},
				Env: map[string]string{
					"CORTEX_PROJECT_PATH": projectPath,
					"CORTEX_TMUX_SESSION": sessionName,
				},
			},
		},
	}

	configPath := fmt.Sprintf("/tmp/cortex-architect-%s.json", sessionName)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}

	return configPath, nil
}

// buildArchitectPrompt creates the initial prompt for the architect agent.
func buildArchitectPrompt(projectName string) string {
	return fmt.Sprintf(`You are the architect for project: %s

Your role is to manage tickets and orchestrate development work. Use the cortex MCP tools to:
- List tickets with optional status/query filters (listTickets)
- Read full ticket details (readTicket)
- Create and update tickets (createTicket, updateTicket, deleteTicket, moveTicket)
- Spawn agent sessions for tickets (spawnSession)

Start by listing current tickets to understand the project state.`, projectName)
}

// buildAgentCommand constructs the command to run the AI agent.
func buildAgentCommand(agent config.AgentType, mcpConfigPath, prompt string) string {
	switch agent {
	case config.AgentOpenCode:
		return fmt.Sprintf("opencode %q --mcp-config %s", prompt, mcpConfigPath)
	case config.AgentClaude:
		fallthrough
	default:
		return fmt.Sprintf("claude %q --mcp-config %s", prompt, mcpConfigPath)
	}
}

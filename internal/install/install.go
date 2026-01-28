package install

import (
	"os"
	"path/filepath"

	"github.com/kareemaly/cortex/internal/daemon/config"
	"github.com/kareemaly/cortex/internal/prompt"
)

// defaultArchitectSystemPrompt contains the full self-contained system prompt for the architect agent.
// This fully replaces the default Claude Code engineer prompt via --system-prompt.
// Dynamic content (ticket list) is injected via the KICKOFF prompt.
const defaultArchitectSystemPrompt = `# Role

You are a project architect. You orchestrate development by managing tickets
and delegating implementation to ticket agents. You do not write code or
read source files directly.

<do_not_act_before_instructions>
Never implement changes, edit files, or write code yourself. When the user
describes a feature, bug, or improvement, your job is to create a well-scoped
ticket and spawn an agent session to do the work. Default to creating tickets
and delegating rather than taking direct action. Only proceed with spawning
when the user explicitly approves.
</do_not_act_before_instructions>

<stay_high_level>
Do not read source files, explore codebases, or engage with implementation
details directly. This pollutes your context with low-level jargon that
is not your concern. When you need technical context to properly scope a
ticket, spawn an explore agent to investigate and return a high-level
summary. Focus your context on requirements, architecture, and ticket
management — leave implementation details to the ticket agents.
</stay_high_level>

<investigate_before_answering>
Always read ticket details before making decisions about them. Never assume
ticket state or contents — use readTicket to inspect before acting. When
reviewing completed work, read the ticket comments and review history before
approving.
</investigate_before_answering>

## Context Awareness

Your context window will be automatically compacted as it approaches its limit.
Save important decisions and context into ticket bodies and comments so state
persists across compactions. Use ticket comments (type: decision) to record
architectural choices.

## Cortex MCP Tools

### Read Operations (auto-approved)
- ` + "`listTickets`" + ` — List tickets by status (backlog, progress, review, done)
- ` + "`readTicket`" + ` — Read full ticket details by ID

### Write Operations (require approval)
- ` + "`createTicket`" + ` — Create a new ticket with title and body
- ` + "`updateTicket`" + ` — Update ticket title or body
- ` + "`deleteTicket`" + ` — Delete a ticket by ID
- ` + "`moveTicket`" + ` — Move ticket to a different status
- ` + "`addTicketComment`" + ` — Add comments to tickets (types: decision, blocker,
  progress, question, scope_change)
- ` + "`spawnSession`" + ` — Spawn a ticket agent session to do the work

## Workflow

1. Discuss requirements with the user to clarify scope
2. If technical context is needed, spawn an explore agent to investigate — do
   not read source files yourself
3. Create well-scoped tickets with clear requirements and acceptance criteria
4. Use ` + "`spawnSession`" + ` to assign work to a ticket agent
5. Monitor progress by reading ticket comments
6. Review completed work when tickets move to review status
7. Use ` + "`addTicketComment`" + ` to provide feedback or record decisions
8. Approve or request changes

## Writing Good Tickets

A ticket body should contain:
- **Summary** — What needs to change and why
- **Requirements** — Expected behavior and constraints
- **Acceptance criteria** — How to verify the work is complete

Only include implementation guidance when it involves an architectural
decision (e.g., choice of protocol, data model, API design). Leave
implementation details and code-level decisions to the ticket agent.

## Communication Style

Be direct and concise. Provide fact-based assessments. When brainstorming
with the user, focus on trade-offs and constraints rather than code details.
Ask clarifying questions before creating tickets when requirements are
ambiguous.
`

// defaultArchitectKickoffPrompt is the template for the architect kickoff prompt.
// Variables: {{.ProjectName}}, {{.TicketList}}
const defaultArchitectKickoffPrompt = `# Project: {{.ProjectName}}

# Tickets

{{.TicketList}}
`

// Options configures the installation.
type Options struct {
	// ProjectPath is the path for project setup. If empty, project setup is skipped.
	ProjectPath string
	// ProjectName overrides auto-detected project name.
	ProjectName string
	// Force overwrites existing config files.
	Force bool
}

// Run performs the installation with the given options.
func Run(opts Options) (*Result, error) {
	result := &Result{}

	// Always setup global config
	globalItems, err := setupGlobal(opts.Force)
	if err != nil {
		return nil, err
	}
	result.GlobalItems = globalItems

	// Setup project if path is provided
	if opts.ProjectPath != "" {
		name := opts.ProjectName
		if name == "" {
			name = DetectProjectName(opts.ProjectPath)
		}
		result.ProjectName = name

		projectItems, err := setupProject(opts.ProjectPath, name, opts.Force)
		if err != nil {
			return nil, err
		}
		result.ProjectItems = projectItems

		// Auto-register project in global config (non-fatal)
		registered, regErr := registerProject(opts.ProjectPath, name)
		result.Registered = registered
		result.RegistrationError = regErr
	}

	// Check dependencies
	result.Dependencies = CheckDependencies()

	return result, nil
}

// registerProject adds the project to the global settings.yaml registry.
func registerProject(projectPath, name string) (bool, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return false, err
	}

	cfg, err := config.Load()
	if err != nil {
		return false, err
	}

	if !cfg.RegisterProject(absPath, name) {
		return false, nil // already registered
	}

	if err := cfg.Save(); err != nil {
		return false, err
	}
	return true, nil
}

// setupGlobal creates the global ~/.cortex/ directory and config.
func setupGlobal(force bool) ([]SetupItem, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	cortexDir := filepath.Join(homeDir, ".cortex")
	configPath := filepath.Join(cortexDir, "settings.yaml")

	var items []SetupItem

	// Create cortex directory
	item := ensureDir(cortexDir)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	// Create config file
	configContent := `port: 4200
log_level: info
status_history_limit: 10
git_diff_tool: diff
`
	item = ensureConfigFile(configPath, configContent, force)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	return items, nil
}

// setupProject creates the project .cortex/ directory and config.
func setupProject(projectPath, name string, force bool) ([]SetupItem, error) {
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return nil, err
	}

	cortexDir := filepath.Join(absPath, ".cortex")
	ticketsDir := filepath.Join(cortexDir, "tickets")
	backlogDir := filepath.Join(ticketsDir, "backlog")
	progressDir := filepath.Join(ticketsDir, "progress")
	reviewDir := filepath.Join(ticketsDir, "review")
	doneDir := filepath.Join(ticketsDir, "done")
	configPath := filepath.Join(cortexDir, "cortex.yaml")

	var items []SetupItem

	// Create directories
	dirs := []string{cortexDir, ticketsDir, backlogDir, progressDir, reviewDir, doneDir}
	for _, dir := range dirs {
		item := ensureDir(dir)
		items = append(items, item)
		if item.Error != nil {
			return items, item.Error
		}
	}

	// Create config file
	configContent := `name: ` + name + `
architect:
  agent: claude
  args:
    - "--allowedTools"
    - "mcp__cortex__listTickets,mcp__cortex__readTicket"
ticket:
  work:
    agent: claude
    args:
      - "--permission-mode"
      - "plan"
      - "--allow-dangerously-skip-permissions"
      - "--allowedTools"
      - "mcp__cortex__readTicket"
git:
  worktrees: false
`
	item := ensureConfigFile(configPath, configContent, force)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	// Create prompts directory structure
	promptsDir := prompt.PromptsDir(absPath)
	architectPromptsDir := filepath.Join(promptsDir, "architect")
	ticketWorkPromptsDir := filepath.Join(promptsDir, "ticket", "work")

	promptDirs := []string{promptsDir, architectPromptsDir, ticketWorkPromptsDir}
	for _, dir := range promptDirs {
		item = ensureDir(dir)
		items = append(items, item)
		if item.Error != nil {
			return items, item.Error
		}
	}

	// Architect prompts
	architectSystemPath := prompt.ArchitectPromptPath(absPath, prompt.StageSystem)
	item = ensureConfigFile(architectSystemPath, defaultArchitectSystemPrompt, force)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	architectKickoffPath := prompt.ArchitectPromptPath(absPath, prompt.StageKickoff)
	item = ensureConfigFile(architectKickoffPath, defaultArchitectKickoffPrompt, force)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	// Ticket work prompts
	ticketSystemPath := prompt.TicketPromptPath(absPath, "work", prompt.StageSystem)
	item = ensureConfigFile(ticketSystemPath, DefaultTicketSystemPrompt, force)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	ticketKickoffPath := prompt.TicketPromptPath(absPath, "work", prompt.StageKickoff)
	item = ensureConfigFile(ticketKickoffPath, DefaultTicketKickoffPrompt, force)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	ticketApprovePath := prompt.TicketPromptPath(absPath, "work", prompt.StageApprove)
	item = ensureConfigFile(ticketApprovePath, DefaultTicketApprovePrompt, force)
	items = append(items, item)
	if item.Error != nil {
		return items, item.Error
	}

	return items, nil
}

// ensureDir creates a directory if it doesn't exist.
func ensureDir(path string) SetupItem {
	item := SetupItem{Path: path}

	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			item.Status = StatusExists
			return item
		}
		// Path exists but is not a directory
		item.Error = &PathNotDirectoryError{Path: path}
		return item
	}

	if !os.IsNotExist(err) {
		item.Error = err
		return item
	}

	// Create the directory
	if err := os.MkdirAll(path, 0755); err != nil {
		item.Error = err
		return item
	}

	item.Status = StatusCreated
	return item
}

// ensureConfigFile creates a config file if it doesn't exist or force is true.
func ensureConfigFile(path, content string, force bool) SetupItem {
	item := SetupItem{Path: path}

	_, err := os.Stat(path)
	if err == nil {
		if !force {
			item.Status = StatusExists
			return item
		}
		// Force overwrite
	} else if !os.IsNotExist(err) {
		item.Error = err
		return item
	}

	// Write the file
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		item.Error = err
		return item
	}

	item.Status = StatusCreated
	return item
}

// PathNotDirectoryError indicates a path exists but is not a directory.
type PathNotDirectoryError struct {
	Path string
}

func (e *PathNotDirectoryError) Error() string {
	return "path exists but is not a directory: " + e.Path
}

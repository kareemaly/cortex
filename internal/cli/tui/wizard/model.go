package wizard

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/kareemaly/cortex/internal/daemon/autostart"
	"github.com/kareemaly/cortex/internal/install"
	"github.com/kareemaly/cortex/internal/storage"
)

// ---------------------------------------------------------------------------
// Async messages
// ---------------------------------------------------------------------------

type agentsDetectedMsg struct {
	agents install.AgentAvailability
}

type modelsLoadedMsg struct {
	models []string
	err    error
}

type installDoneMsg struct {
	result *install.Result
	err    error
}

type agentStartedMsg struct {
	proc *install.AgentProcess
	err  error
}

type agentEventMsg struct {
	event install.AgentEvent
}

type daemonDoneMsg struct {
	err error
}

// ---------------------------------------------------------------------------
// Config passed from the cobra command
// ---------------------------------------------------------------------------

// Config holds the initial parameters from CLI flags/args.
type Config struct {
	ArgName    string
	FlagAgent  string
	FlagForce  bool
	GlobalOnly bool
	IsTTY      bool
	Cwd        string
}

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

// Model is the bubbletea model for the architect create wizard.
type Model struct {
	cfg Config

	// Window dimensions.
	width, height int
	ready         bool

	// Phase tracking.
	step       stepID
	stepStatus map[stepID]stepStatus
	err        error

	// Detected state.
	agents    install.AgentAvailability
	companion string

	// Collected values.
	agent           string
	name            string
	slug            string
	architectPath   string
	model           string
	models          []string
	repos           []string
	useCompanion    bool
	featureBranches bool
	researchPaths   []string
	generateMD      bool

	// Install result.
	installResult *install.Result

	// AGENTS.md streaming.
	agentProc       *install.AgentProcess
	agentStreamMsgs []agentStreamMsg
	agentStreamMu   *sync.Mutex
	agentTotalCost  float64
	agentTotalTok   agentStreamTokens

	// Daemon.
	daemonErr error

	// Widgets.
	textInput   textinput.Model
	selectIdx   int
	selectOpts  []string
	confirmDflt bool
	spinner     spinner.Model
	repoError   string
}

type agentStreamMsg struct {
	msgType  string
	content  string
	tool     string
	status   string
	subagent *agentStreamSubagent
}

type agentStreamSubagent struct {
	agentType   string
	description string
}

type agentStreamTokens struct {
	input  int64
	output int64
}

// New creates a new wizard model.
func New(cfg Config) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot

	ss := make(map[stepID]stepStatus)
	for _, def := range allSteps {
		ss[def.id] = statusPending
	}

	return Model{
		cfg:           cfg,
		step:          stepDetectAgents,
		stepStatus:    ss,
		spinner:       s,
		agentStreamMu: &sync.Mutex{},
	}
}

// Err returns the wizard's error (if any) after it exits.
func (m Model) Err() error {
	return m.err
}

// ---------------------------------------------------------------------------
// Init
// ---------------------------------------------------------------------------

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.detectAgentsCmd(),
	)
}

func (m Model) detectAgentsCmd() tea.Cmd {
	return func() tea.Msg {
		return agentsDetectedMsg{agents: install.DetectAgents()}
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		if isKey(msg, KeyCtrlC) {
			if m.agentProc != nil {
				m.agentProc.Kill()
			}
			m.err = fmt.Errorf("cancelled")
			return m, tea.Quit
		}
		if m.step == stepDone {
			// Any key exits
			return m, tea.Quit
		}
		return m.handleKey(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case agentsDetectedMsg:
		return m.handleAgentsDetected(msg)
	case modelsLoadedMsg:
		return m.handleModelsLoaded(msg)
	case installDoneMsg:
		return m.handleInstallDone(msg)
	case agentStartedMsg:
		return m.handleAgentStarted(msg)
	case agentEventMsg:
		return m.handleAgentEvent(msg)
	case daemonDoneMsg:
		return m.handleDaemonDone(msg)
	}

	return m, nil
}

// ---------------------------------------------------------------------------
// Step transitions
// ---------------------------------------------------------------------------

func (m Model) markDone(s stepID) Model {
	m.stepStatus[s] = statusDone
	return m
}

func (m Model) markActive(s stepID) Model {
	m.stepStatus[s] = statusActive
	m.step = s
	return m
}

func (m Model) markSkipped(s stepID) Model {
	m.stepStatus[s] = statusSkipped
	return m
}

func (m Model) markError(s stepID) Model {
	m.stepStatus[s] = statusError
	return m
}

func (m Model) handleAgentsDetected(msg agentsDetectedMsg) (tea.Model, tea.Cmd) {
	m.agents = msg.agents
	m.companion = install.DetectCompanion()
	m = m.markDone(stepDetectAgents)

	if m.cfg.GlobalOnly {
		// Skip all interactive steps, go straight to install
		for _, s := range []stepID{stepSelectAgent, stepInputName, stepSelectModel, stepInputRepos, stepCompanion, stepFeatureBranches, stepResearchPaths} {
			m = m.markSkipped(s)
		}
		m = m.markActive(stepInstall)
		return m, m.runInstall()
	}

	if msg.agents.AgentCount() == 0 {
		m.err = fmt.Errorf("no supported agent found in PATH\n\nInstall one of:\n  claude  — https://docs.anthropic.com/en/docs/claude-code\n  opencode — https://opencode.ai")
		m = m.markError(stepDetectAgents)
		m.step = stepDone
		return m, tea.Quit
	}

	// Resolve agent from flag
	if m.cfg.FlagAgent != "" {
		switch m.cfg.FlagAgent {
		case "claude", "opencode":
			if _, err := exec.LookPath(m.cfg.FlagAgent); err != nil {
				m.err = fmt.Errorf("%s binary not found in PATH; install it first", m.cfg.FlagAgent)
				m = m.markError(stepSelectAgent)
				m.step = stepDone
				return m, tea.Quit
			}
			m.agent = m.cfg.FlagAgent
			m = m.markDone(stepSelectAgent)
		default:
			m.err = fmt.Errorf("invalid agent type %q: must be claude or opencode", m.cfg.FlagAgent)
			m = m.markError(stepSelectAgent)
			m.step = stepDone
			return m, tea.Quit
		}
		return m.advanceFromAgent()
	}

	if msg.agents.AgentCount() == 1 {
		m.agent = msg.agents.OnlyAgent()
		m = m.markDone(stepSelectAgent)
		return m.advanceFromAgent()
	}

	// Both available — prompt user
	m = m.markActive(stepSelectAgent)
	m.selectOpts = []string{"opencode", "claude"}
	m.selectIdx = 0
	return m, nil
}

func (m Model) advanceFromAgent() (tea.Model, tea.Cmd) {
	if m.cfg.ArgName != "" {
		m.name = m.cfg.ArgName
		m.slug = storage.GenerateSlug(m.name, m.name)
		m.architectPath = filepath.Join(m.cfg.Cwd, m.slug)
		m = m.markDone(stepInputName)
		return m.advanceFromName()
	}

	if !m.cfg.IsTTY {
		m.err = fmt.Errorf("name argument required in non-interactive mode")
		m.step = stepDone
		return m, tea.Quit
	}

	m = m.markActive(stepInputName)
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	ti.CharLimit = 64
	m.textInput = ti
	return m, nil
}

func (m Model) advanceFromName() (tea.Model, tea.Cmd) {
	if m.agent == "opencode" && m.cfg.IsTTY {
		m = m.markActive(stepSelectModel)
		return m, m.loadModelsCmd()
	}
	m = m.markSkipped(stepSelectModel)
	return m.advanceFromModel()
}

func (m Model) advanceFromModel() (tea.Model, tea.Cmd) {
	if !m.cfg.IsTTY {
		for _, s := range []stepID{stepInputRepos, stepCompanion, stepFeatureBranches, stepResearchPaths} {
			m = m.markSkipped(s)
		}
		m = m.markActive(stepInstall)
		return m, m.runInstall()
	}

	m = m.markActive(stepInputRepos)
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Focus()
	m.textInput = ti
	m.repoError = ""
	return m, nil
}

func (m Model) advanceFromRepos() (tea.Model, tea.Cmd) {
	m = m.markActive(stepCompanion)
	m.confirmDflt = true
	return m, nil
}

func (m Model) advanceFromCompanion() (tea.Model, tea.Cmd) {
	m = m.markActive(stepFeatureBranches)
	m.confirmDflt = true
	return m, nil
}

func (m Model) advanceFromFeatureBranches() (tea.Model, tea.Cmd) {
	m = m.markActive(stepResearchPaths)
	ti := textinput.New()
	ti.Placeholder = "~/**"
	ti.Focus()
	m.textInput = ti
	return m, nil
}

func (m Model) advanceFromResearchPaths() (tea.Model, tea.Cmd) {
	m = m.markActive(stepInstall)
	return m, m.runInstall()
}

func (m Model) loadModelsCmd() tea.Cmd {
	return func() tea.Msg {
		models, err := install.GetOpenCodeModels()
		return modelsLoadedMsg{models: models, err: err}
	}
}

func (m Model) handleModelsLoaded(msg modelsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil || len(msg.models) == 0 {
		m = m.markSkipped(stepSelectModel)
		return m.advanceFromModel()
	}

	m.models = msg.models
	m.selectOpts = msg.models

	defaultModels := []string{"github-copilot/gpt-5.2", "github-copilot/claude-sonnet-4.6", "anthropic/claude-sonnet-4-6"}
	m.selectIdx = 0
	for i, model := range msg.models {
		for _, d := range defaultModels {
			if model == d && m.selectIdx == 0 {
				m.selectIdx = i
				break
			}
		}
	}

	ti := textinput.New()
	ti.Placeholder = "type to filter..."
	ti.Focus()
	m.textInput = ti
	return m, nil
}

func (m Model) runInstall() tea.Cmd {
	opts := install.Options{
		ArchitectPath:   m.architectPath,
		ArchitectName:   m.name,
		Agent:           m.agent,
		Model:           m.model,
		Force:           m.cfg.FlagForce,
		Repos:           m.repos,
		FeatureBranches: m.featureBranches,
		ResearchPaths:   m.researchPaths,
	}
	if m.useCompanion {
		opts.Companion = m.companion
	}
	if m.cfg.GlobalOnly {
		opts.ArchitectPath = m.cfg.Cwd
	}
	return func() tea.Msg {
		result, err := install.Run(opts)
		return installDoneMsg{result: result, err: err}
	}
}

func (m Model) handleInstallDone(msg installDoneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m = m.markError(stepInstall)
		m.step = stepDone
		return m, tea.Quit
	}
	m.installResult = msg.result
	m = m.markDone(stepInstall)

	if m.cfg.GlobalOnly || len(m.repos) == 0 || !m.cfg.IsTTY {
		m = m.markSkipped(stepAgentsMDConfirm)
		m = m.markSkipped(stepAgentsMDRun)
		m = m.markActive(stepDaemon)
		return m, m.runDaemon()
	}

	m = m.markActive(stepAgentsMDConfirm)
	m.confirmDflt = true
	return m, nil
}

func (m Model) runDaemon() tea.Cmd {
	return func() tea.Msg {
		return daemonDoneMsg{err: autostart.EnsureDaemonRunning()}
	}
}

func (m Model) handleDaemonDone(msg daemonDoneMsg) (tea.Model, tea.Cmd) {
	m.daemonErr = msg.err
	if msg.err != nil {
		m = m.markError(stepDaemon)
	} else {
		m = m.markDone(stepDaemon)
	}
	m.step = stepDone
	return m, nil
}

// ---------------------------------------------------------------------------
// Key handling
// ---------------------------------------------------------------------------

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.step {
	case stepSelectAgent:
		return m.handleSelectKey(msg)
	case stepInputName:
		return m.handleNameKey(msg)
	case stepSelectModel:
		return m.handleSearchSelectKey(msg)
	case stepInputRepos:
		return m.handleRepoKey(msg)
	case stepCompanion:
		return m.handleConfirmKey(msg, func(yes bool) (tea.Model, tea.Cmd) {
			m.useCompanion = yes
			m = m.markDone(stepCompanion)
			return m.advanceFromCompanion()
		})
	case stepFeatureBranches:
		return m.handleConfirmKey(msg, func(yes bool) (tea.Model, tea.Cmd) {
			m.featureBranches = yes
			m = m.markDone(stepFeatureBranches)
			return m.advanceFromFeatureBranches()
		})
	case stepResearchPaths:
		return m.handleResearchPathsKey(msg)
	case stepAgentsMDConfirm:
		return m.handleConfirmKey(msg, func(yes bool) (tea.Model, tea.Cmd) {
			m.generateMD = yes
			if !yes {
				m = m.markSkipped(stepAgentsMDConfirm)
				m = m.markSkipped(stepAgentsMDRun)
				m = m.markActive(stepDaemon)
				return m, m.runDaemon()
			}
			m = m.markDone(stepAgentsMDConfirm)
			m = m.markActive(stepAgentsMDRun)
			return m, m.runAgentsMD()
		})
	}
	return m, nil
}

func (m Model) handleSelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, KeyUp, KeyK):
		if m.selectIdx > 0 {
			m.selectIdx--
		}
	case isKey(msg, KeyDown, KeyJ):
		if m.selectIdx < len(m.selectOpts)-1 {
			m.selectIdx++
		}
	case isKey(msg, KeyEnter):
		m.agent = m.selectOpts[m.selectIdx]
		m = m.markDone(stepSelectAgent)
		return m.advanceFromAgent()
	}
	return m, nil
}

func (m Model) handleNameKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		val := m.textInput.Value()
		if val == "" {
			return m, nil // require a name
		}
		m.name = val
		m.slug = storage.GenerateSlug(m.name, m.name)
		m.architectPath = filepath.Join(m.cfg.Cwd, m.slug)
		m = m.markDone(stepInputName)
		return m.advanceFromName()
	default:
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
}

func (m Model) handleSearchSelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, KeyUp):
		if m.selectIdx > 0 {
			m.selectIdx--
		}
		return m, nil
	case isKey(msg, KeyDown):
		filtered := m.getFilteredOpts()
		if m.selectIdx < len(filtered)-1 {
			m.selectIdx++
		}
		return m, nil
	case isKey(msg, KeyEnter):
		filtered := m.getFilteredOpts()
		if len(filtered) > 0 && m.selectIdx < len(filtered) {
			m.model = filtered[m.selectIdx]
		} else if len(m.selectOpts) > 0 {
			m.model = m.selectOpts[0]
		}
		m = m.markDone(stepSelectModel)
		return m.advanceFromModel()
	default:
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		m.selectIdx = 0
		return m, cmd
	}
}

func (m Model) handleRepoKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		input := m.textInput.Value()
		if input == "" {
			m.repoError = "Repos are required"
			return m, nil
		}
		repos := parseCommaList(input)
		for _, repo := range repos {
			expanded := storage.ExpandHome(repo)
			if err := validateGitRepo(expanded); err != nil {
				m.repoError = fmt.Sprintf("Invalid repo %q: %v", repo, err)
				return m, nil
			}
		}
		m.repos = repos
		m.repoError = ""
		m = m.markDone(stepInputRepos)
		return m.advanceFromRepos()
	default:
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		m.repoError = ""
		return m, cmd
	}
}

func (m Model) handleResearchPathsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		val := m.textInput.Value()
		if val == "" && m.textInput.Placeholder != "" {
			val = m.textInput.Placeholder
		}
		m.researchPaths = parseCommaList(val)
		m = m.markDone(stepResearchPaths)
		return m.advanceFromResearchPaths()
	default:
		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)
		return m, cmd
	}
}

func (m Model) handleConfirmKey(msg tea.KeyMsg, onDone func(bool) (tea.Model, tea.Cmd)) (tea.Model, tea.Cmd) {
	switch {
	case isKey(msg, KeyEnter):
		return onDone(m.confirmDflt)
	case isKey(msg, KeyY):
		return onDone(true)
	case isKey(msg, KeyN):
		return onDone(false)
	}
	return m, nil
}

func (m Model) runAgentsMD() tea.Cmd {
	agent := m.agent
	repos := m.repos
	architectPath := m.architectPath
	mdl := m.model
	return func() tea.Msg {
		proc, err := install.StartAgentsMDProcess(architectPath, repos, agent, mdl)
		return agentStartedMsg{proc: proc, err: err}
	}
}

func (m Model) handleAgentStarted(msg agentStartedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m = m.markError(stepAgentsMDRun)
		m = m.markActive(stepDaemon)
		return m, m.runDaemon()
	}
	m.agentProc = msg.proc
	return m, m.readNextEvent()
}

func (m Model) handleAgentEvent(msg agentEventMsg) (tea.Model, tea.Cmd) {
	ev := msg.event

	switch ev.Type {
	case "error":
		m = m.markError(stepAgentsMDRun)
		if m.agentProc != nil {
			m.agentProc.Kill()
			_ = m.agentProc.Wait()
		}
		m = m.markActive(stepDaemon)
		return m, m.runDaemon()

	case "done":
		if m.agentProc != nil {
			_ = m.agentProc.Wait()
		}
		m = m.markDone(stepAgentsMDRun)
		m = m.markActive(stepDaemon)
		return m, m.runDaemon()

	case "result":
		// Result event carries final cost/tokens — record and wait for "done".
		if ev.Cost > 0 {
			m.agentTotalCost += ev.Cost
		}
		if ev.Tokens.Input > 0 {
			m.agentTotalTok.input += ev.Tokens.Input
			m.agentTotalTok.output += ev.Tokens.Output
		}
		return m, m.readNextEvent()

	case "step_finish":
		if ev.Cost > 0 {
			m.agentTotalCost += ev.Cost
		}
		if ev.Tokens.Input > 0 {
			m.agentTotalTok.input += ev.Tokens.Input
			m.agentTotalTok.output += ev.Tokens.Output
		}
		return m, m.readNextEvent()
	}

	// Tool, subagent, text, reasoning — add to display list.
	streamMsg := agentStreamMsg{
		msgType: ev.Type,
		tool:    ev.Tool,
		status:  ev.Status,
		content: ev.Content,
	}
	if ev.Subagent != nil {
		streamMsg.subagent = &agentStreamSubagent{
			agentType:   ev.Subagent.AgentType,
			description: ev.Subagent.Description,
		}
	}

	m.agentStreamMu.Lock()
	m.agentStreamMsgs = append(m.agentStreamMsgs, streamMsg)
	if len(m.agentStreamMsgs) > 15 {
		m.agentStreamMsgs = m.agentStreamMsgs[len(m.agentStreamMsgs)-15:]
	}
	m.agentStreamMu.Unlock()

	return m, m.readNextEvent()
}

func (m Model) readNextEvent() tea.Cmd {
	proc := m.agentProc
	if proc == nil {
		return nil
	}
	return func() tea.Msg {
		ev := install.ReadNextEvent(proc)
		return agentEventMsg{event: ev}
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (m Model) getFilteredOpts() []string {
	if m.textInput.Value() == "" {
		return m.selectOpts
	}
	query := strings.ToLower(m.textInput.Value())
	var filtered []string
	for _, opt := range m.selectOpts {
		if strings.Contains(strings.ToLower(opt), query) {
			filtered = append(filtered, opt)
		}
	}
	return filtered
}

func parseCommaList(input string) []string {
	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func validateGitRepo(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist")
		}
		return fmt.Errorf("failed to access directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory")
	}
	gitDir := filepath.Join(path, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return fmt.Errorf("not a git repository (no .git found)")
	}
	return nil
}

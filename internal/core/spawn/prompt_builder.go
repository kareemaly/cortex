package spawn

import (
	"fmt"
	"sort"
	"strings"
	"time"

	architectconfig "github.com/kareemaly/cortex/internal/architect/config"
	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/prompt"
	"github.com/kareemaly/cortex/internal/ticket"
)

// promptInfo contains both dynamic prompt text and the static system prompt content.
type promptInfo struct {
	PromptText          string
	SystemPromptContent string
}

// buildPrompt builds the dynamic prompt and returns the system prompt content.
// Dynamic content (ticket details, ticket lists) is embedded in the prompt.
// Static instructions are loaded from file via --system-prompt (architect, full replace)
// or --append-system-prompt (ticket agent, appended to default).
func (s *Spawner) buildPrompt(req SpawnRequest, workingDir string) (*promptInfo, error) {
	switch req.AgentType {
	case AgentTypeTicketAgent:
		return s.buildTicketAgentPrompt(req, workingDir)
	case AgentTypeArchitect:
		return s.buildArchitectPrompt(req)
	case AgentTypeCollabAgent:
		return &promptInfo{PromptText: req.Prompt}, nil
	default:
		return nil, &ConfigError{Field: "AgentType", Message: "unknown agent type: " + string(req.AgentType)}
	}
}

// buildTicketAgentPrompt creates the dynamic ticket prompt.
func (s *Spawner) buildTicketAgentPrompt(req SpawnRequest, workingDir string) (*promptInfo, error) {
	ticketType := ticket.DefaultTicketType

	resolver := prompt.NewPromptResolver(req.ArchitectPath, s.deps.DefaultsDir)

	s.logWarn("buildTicketAgentPrompt: resolving prompts",
		"ticketType", ticketType,
		"architectPath", req.ArchitectPath,
		"defaultsDir", s.deps.DefaultsDir)

	systemPromptContent, systemErr := resolver.ResolveTicketPrompt(ticketType, prompt.StageSystem)
	s.logWarn("buildTicketAgentPrompt: system prompt resolved",
		"systemPromptLen", len(systemPromptContent),
		"systemErr", systemErr)

	kickoffTemplate, err := resolver.ResolveTicketPrompt(ticketType, prompt.StageKickoff)
	s.logWarn("buildTicketAgentPrompt: kickoff template resolved",
		"kickoffTemplateLen", len(kickoffTemplate),
		"err", err)

	if err != nil {
		s.logWarn("buildTicketAgentPrompt: using fallback prompt due to error", "err", err)
		promptText := fmt.Sprintf("# Ticket: %s\n\n%s", req.Ticket.Title, req.Ticket.Body)
		return &promptInfo{
			PromptText:          promptText,
			SystemPromptContent: systemPromptContent,
		}, nil
	}

	vars := prompt.TicketVars{
		ProjectPath: req.ArchitectPath,
		TicketID:    req.TicketID,
		TicketTitle: req.Ticket.Title,
		TicketBody:  req.Ticket.Body,
		References:  formatTicketReferences(req.Ticket.References),
		Repo:        req.Ticket.Repo,
		RepoPath:    workingDir,
	}

	if cfg, cfgErr := architectconfig.Load(req.ArchitectPath); cfgErr == nil {
		vars.ArchitectName = cfg.Name
		vars.Repos = formatOtherRepos(cfg, req.Ticket.Repo)
	} else if req.Ticket.Repo != "" {
		return nil, cfgErr
	}

	promptText, err := prompt.RenderTemplate(kickoffTemplate, vars)
	if err != nil {
		return nil, err
	}

	return &promptInfo{
		PromptText:          promptText,
		SystemPromptContent: systemPromptContent,
	}, nil
}

// buildArchitectPrompt creates the dynamic architect prompt with ticket list.
func (s *Spawner) buildArchitectPrompt(req SpawnRequest) (*promptInfo, error) {
	resolver := prompt.NewPromptResolver(req.ArchitectPath, s.deps.DefaultsDir)

	var systemPromptContent string
	{
		var err error
		systemPromptContent, err = resolver.ResolveArchitectPrompt(prompt.StageSystem)
		if err != nil {
			return nil, err
		}
	}

	client := sdk.DefaultClient(req.ArchitectPath)
	tickets, err := client.ListAllTickets("", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list tickets: %w", err)
	}

	var sb strings.Builder

	writeSection := func(name string, items []sdk.TicketSummary) {
		sb.WriteString(fmt.Sprintf("## %s\n", name))
		if len(items) == 0 {
			sb.WriteString("(none)\n")
		} else {
			for _, t := range items {
				dueStr := ""
				if t.Due != nil {
					dueStr = fmt.Sprintf(" (due: %s)", t.Due.Format(time.DateOnly))
				}
				sb.WriteString(fmt.Sprintf("- [%s] %s%s (updated: %s)\n", t.ID, t.Title, dueStr, t.Updated.Format(time.DateOnly)))
			}
		}
		sb.WriteString("\n")
	}

	writeSection("Backlog", tickets.Backlog)
	writeSection("In Progress", tickets.Progress)
	doneTickets := tickets.Done
	if len(doneTickets) > 10 {
		doneTickets = doneTickets[:10]
	}
	writeSection("Done", doneTickets)

	ticketList := sb.String()

	var sessionsList string
	conclusionsResp, conclusionsErr := client.ListConclusions(sdk.ListConclusionsParams{Limit: 10})
	if conclusionsErr == nil && len(conclusionsResp.Conclusions) > 0 {
		var sessionsSB strings.Builder
		for _, c := range conclusionsResp.Conclusions {
			sessionsSB.WriteString(fmt.Sprintf("- [%s] %s (%s)\n", c.ID, c.TicketID, c.Agent))
		}
		sessionsList = sessionsSB.String()
	}

	var lastConclusionID string
	archConclusionsResp, archConclusionsErr := client.ListConclusions(sdk.ListConclusionsParams{
		Type:  "architect",
		Limit: 1,
	})
	if archConclusionsErr == nil && len(archConclusionsResp.Conclusions) > 0 {
		lastConclusionID = archConclusionsResp.Conclusions[0].ID
	}

	var reposList string
	var variantsList string
	projectCfg, cfgErr := architectconfig.Load(req.ArchitectPath)
	if cfgErr == nil {
		if len(projectCfg.Repos) > 0 {
			var reposSB strings.Builder
			for _, key := range projectCfg.RepoKeys() {
				repoPath, err := projectCfg.ResolveRepoPath(key)
				if err != nil {
					return nil, err
				}
				reposSB.WriteString(fmt.Sprintf("- %s: %s\n", key, repoPath))
			}
			reposList = reposSB.String()
		}
		if names := projectCfg.VariantNames(); len(names) > 0 {
			variantsList = strings.Join(names, ", ")
		}
	}

	kickoffTemplate, kickoffErr := resolver.ResolveArchitectPrompt(prompt.StageKickoff)
	if kickoffErr == nil {
		vars := prompt.ArchitectKickoffVars{
			ArchitectName:    req.ArchitectName,
			TicketList:       ticketList,
			CurrentDate:      time.Now().Format("2006-01-02 15:04 MST"),
			Sessions:         sessionsList,
			Repos:            reposList,
			LastConclusionID: lastConclusionID,
			Variants:         variantsList,
		}
		rendered, renderErr := prompt.RenderTemplate(kickoffTemplate, vars)
		if renderErr == nil {
			return &promptInfo{
				PromptText:          rendered,
				SystemPromptContent: systemPromptContent,
			}, nil
		}
	}

	promptText := fmt.Sprintf("# Project: %s\n\n# Tickets\n\n%s", req.ArchitectName, ticketList)

	return &promptInfo{
		PromptText:          promptText,
		SystemPromptContent: systemPromptContent,
	}, nil
}

// formatTicketReferences formats ticket references into a bulleted markdown list.
func formatTicketReferences(refs []string) string {
	if len(refs) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, ref := range refs {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("- ")
		sb.WriteString(ref)
	}
	return sb.String()
}

// formatOtherRepos formats repos into a bulleted markdown list, excluding the current ticket's repo key.
func formatOtherRepos(cfg *architectconfig.Config, currentRepo string) string {
	keys := cfg.RepoKeys()
	sort.Strings(keys)
	var sb strings.Builder
	first := true
	for _, key := range keys {
		if key == currentRepo {
			continue
		}
		repoPath, err := cfg.ResolveRepoPath(key)
		if err != nil {
			continue
		}
		if !first {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("- %s: %s", key, repoPath))
		first = false
	}
	return sb.String()
}

package mcp

import (
	"github.com/kareemaly/cortex/internal/lifecycle"
	"github.com/kareemaly/cortex/internal/project/config"
	"github.com/kareemaly/cortex/internal/ticket"
)

// convertHookConfigs converts project config hooks to lifecycle hook definitions.
func convertHookConfigs(hooks []config.HookConfig) []lifecycle.HookDefinition {
	result := make([]lifecycle.HookDefinition, len(hooks))
	for i, h := range hooks {
		result[i] = lifecycle.HookDefinition{Run: h.Run}
	}
	return result
}

// buildTemplateVars creates template variables from a ticket.
func buildTemplateVars(t *ticket.Ticket) lifecycle.TemplateVars {
	return lifecycle.NewTemplateVars(
		t.ID,
		ticket.GenerateSlug(t.Title),
		t.Title,
	)
}

// getHooksForType returns the hook definitions for a given hook type.
func (s *Server) getHooksForType(hookType lifecycle.HookType) []lifecycle.HookDefinition {
	if s.projectConfig == nil {
		return nil
	}

	switch hookType {
	case lifecycle.HookOnPickup:
		return convertHookConfigs(s.projectConfig.Lifecycle.OnPickup)
	case lifecycle.HookOnSubmit:
		return convertHookConfigs(s.projectConfig.Lifecycle.OnSubmit)
	case lifecycle.HookOnApprove:
		return convertHookConfigs(s.projectConfig.Lifecycle.OnApprove)
	default:
		return nil
	}
}

// convertExecutionResult converts a lifecycle execution result to hook output.
func convertExecutionResult(result *lifecycle.ExecutionResult) *HooksExecutionOutput {
	if result == nil {
		return &HooksExecutionOutput{
			Executed: false,
			Success:  true,
		}
	}

	hooks := make([]HookResultOutput, len(result.Hooks))
	for i, h := range result.Hooks {
		hooks[i] = HookResultOutput{
			Command:  h.Command,
			Stdout:   h.Stdout,
			ExitCode: h.ExitCode,
		}
	}

	return &HooksExecutionOutput{
		Executed: true,
		Success:  result.Success,
		Hooks:    hooks,
	}
}

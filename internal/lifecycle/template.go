package lifecycle

import (
	"regexp"
	"strings"
)

// templateVarRegex matches template variables like {{variable_name}}.
var templateVarRegex = regexp.MustCompile(`\{\{(\w+)\}\}`)

// validVarNames maps variable names to their availability in hook types.
// true means available in all hook types, false means only in on_approve.
var validVarNames = map[string]bool{
	"ticket_id":      true,
	"ticket_slug":    true,
	"ticket_title":   true,
	"commit_message": false, // Only available in on_approve
}

// expandTemplate replaces template variables in a command string.
// Unknown variables are left unchanged (graceful degradation).
func expandTemplate(command string, vars TemplateVars) (string, error) {
	result := templateVarRegex.ReplaceAllStringFunc(command, func(match string) string {
		// Extract variable name from {{variable_name}}
		varName := match[2 : len(match)-2]

		switch varName {
		case "ticket_id":
			return vars.TicketID
		case "ticket_slug":
			return vars.TicketSlug
		case "ticket_title":
			return vars.TicketTitle
		case "commit_message":
			return vars.CommitMessage
		default:
			// Leave unknown variables unchanged
			return match
		}
	})

	return result, nil
}

// ValidateHookType checks if the command uses variables appropriate for the hook type.
// Returns InvalidVariableError if commit_message is used outside on_approve hooks.
func ValidateHookType(hookType HookType, command string) error {
	matches := templateVarRegex.FindAllStringSubmatch(command, -1)

	for _, match := range matches {
		varName := match[1]

		// Check if commit_message is used in non-approve hooks
		if varName == "commit_message" && hookType != HookOnApprove {
			return &InvalidVariableError{
				Variable: varName,
				HookType: hookType,
			}
		}
	}

	return nil
}

// ContainsTemplateVars checks if a string contains any template variables.
func ContainsTemplateVars(s string) bool {
	return templateVarRegex.MatchString(s)
}

// ExtractTemplateVars returns all template variable names found in a string.
func ExtractTemplateVars(s string) []string {
	matches := templateVarRegex.FindAllStringSubmatch(s, -1)
	vars := make([]string, 0, len(matches))

	seen := make(map[string]bool)
	for _, match := range matches {
		varName := match[1]
		if !seen[varName] {
			vars = append(vars, varName)
			seen[varName] = true
		}
	}

	return vars
}

// IsKnownVariable returns true if the variable name is a known template variable.
func IsKnownVariable(varName string) bool {
	_, ok := validVarNames[varName]
	return ok
}

// IsAvailableInHookType returns true if the variable is available in the given hook type.
func IsAvailableInHookType(varName string, hookType HookType) bool {
	availableInAll, ok := validVarNames[varName]
	if !ok {
		return false
	}

	if availableInAll {
		return true
	}

	// commit_message is only available in on_approve
	return hookType == HookOnApprove
}

// EscapeForShell escapes special characters for safe shell usage.
// This is a simple escape that wraps the string in single quotes.
func EscapeForShell(s string) string {
	// Replace single quotes with '\'' (end quote, escaped quote, start quote)
	escaped := strings.ReplaceAll(s, "'", "'\\''")
	return "'" + escaped + "'"
}

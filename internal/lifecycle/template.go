package lifecycle

import (
	"regexp"
	"strings"
)

// templateVarRegex matches template variables like {{variable_name}}.
var templateVarRegex = regexp.MustCompile(`\{\{(\w+)\}\}`)

// validVarNames maps variable names to their availability in hook types.
// true means available in all hook types, false means limited availability.
var validVarNames = map[string]bool{
	"ticket_id":      true,
	"ticket_slug":    true,
	"ticket_title":   true,
	"ticket_body":    true,
	"session_id":     true,
	"agent":          true,
	"commit_message": false, // Only available in on_approve/moved_to_done
	"comment_type":   false, // Only available in comment_added
	"comment":        false, // Only available in comment_added
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
		case "ticket_body":
			return vars.TicketBody
		case "session_id":
			return vars.SessionID
		case "agent":
			return vars.Agent
		case "commit_message":
			return vars.CommitMessage
		case "comment_type":
			return vars.CommentType
		case "comment":
			return vars.Comment
		default:
			// Leave unknown variables unchanged
			return match
		}
	})

	return result, nil
}

// ValidateHookType checks if the command uses variables appropriate for the hook type.
// Returns InvalidVariableError if a variable is used in an inappropriate hook type.
func ValidateHookType(hookType HookType, command string) error {
	matches := templateVarRegex.FindAllStringSubmatch(command, -1)

	for _, match := range matches {
		varName := match[1]

		// Check if commit_message is used in inappropriate hooks
		if varName == "commit_message" {
			if hookType != HookOnApprove && hookType != HookMovedToDone {
				return &InvalidVariableError{
					Variable: varName,
					HookType: hookType,
				}
			}
		}

		// Check if comment variables are used in non-comment hooks
		if varName == "comment_type" || varName == "comment" {
			if hookType != HookCommentAdded {
				return &InvalidVariableError{
					Variable: varName,
					HookType: hookType,
				}
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

	// commit_message is available in on_approve and moved_to_done
	if varName == "commit_message" {
		return hookType == HookOnApprove || hookType == HookMovedToDone
	}

	// comment_type and comment are only available in comment_added
	if varName == "comment_type" || varName == "comment" {
		return hookType == HookCommentAdded
	}

	return false
}

// EscapeForShell escapes special characters for safe shell usage.
// This is a simple escape that wraps the string in single quotes.
func EscapeForShell(s string) string {
	// Replace single quotes with '\'' (end quote, escaped quote, start quote)
	escaped := strings.ReplaceAll(s, "'", "'\\''")
	return "'" + escaped + "'"
}

package config

// MergeConfigs merges a project config onto a base config.
// Rules:
// - Scalars: project wins if non-zero
// - Args slices: project replaces entirely (no append)
// - Ticket map: merge entries, project wins on conflict
func MergeConfigs(base, project *Config) *Config {
	if base == nil {
		return project
	}
	if project == nil {
		return base
	}

	result := &Config{}

	// Name: project wins if set
	result.Name = base.Name
	if project.Name != "" {
		result.Name = project.Name
	}

	// Extend: always use project's extend (resolved in Load)
	result.Extend = project.Extend

	// Architect: merge role config
	result.Architect = mergeRoleConfig(base.Architect, project.Architect)

	// Ticket: merge maps
	result.Ticket = mergeTicketConfig(base.Ticket, project.Ticket)

	// Git: project wins if set
	result.Git = mergeGitConfig(base.Git, project.Git)

	return result
}

// mergeRoleConfig merges two RoleConfigs, with project taking precedence.
func mergeRoleConfig(base, project RoleConfig) RoleConfig {
	result := RoleConfig{}

	// Agent: project wins if set
	result.Agent = base.Agent
	if project.Agent != "" {
		result.Agent = project.Agent
	}

	// Args: project replaces entirely if set (no append)
	if len(project.Args) > 0 {
		result.Args = make([]string, len(project.Args))
		copy(result.Args, project.Args)
	} else if len(base.Args) > 0 {
		result.Args = make([]string, len(base.Args))
		copy(result.Args, base.Args)
	}

	return result
}

// mergeTicketConfig merges two TicketConfigs, with project entries taking precedence.
func mergeTicketConfig(base, project TicketConfig) TicketConfig {
	if base == nil && project == nil {
		return nil
	}

	result := make(TicketConfig)

	// Copy base entries
	for k, v := range base {
		result[k] = v
	}

	// Overlay project entries (project wins on conflict)
	for k, v := range project {
		if baseRole, exists := result[k]; exists {
			// Merge at role level
			result[k] = mergeRoleConfig(baseRole, v)
		} else {
			result[k] = v
		}
	}

	return result
}

// mergeGitConfig merges two GitConfigs, with project taking precedence.
func mergeGitConfig(base, project GitConfig) GitConfig {
	result := GitConfig{}

	// Worktrees: project wins (including false)
	// Since we can't distinguish "not set" from "set to false" with bool,
	// project always overrides base for GitConfig
	result.Worktrees = project.Worktrees
	if !project.Worktrees {
		result.Worktrees = base.Worktrees
	}

	return result
}

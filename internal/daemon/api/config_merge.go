package api

import (
	architectconfig "github.com/kareemaly/cortex/internal/architect/config"
	daemonconfig "github.com/kareemaly/cortex/internal/daemon/config"
)

// mergeProjectConfig loads the project config and merges global agent variants
// from settings.yaml. Global entries serve as base; project-level entries win.
func mergeProjectConfig(projectPath string) (*architectconfig.Config, error) {
	cfg, err := architectconfig.Load(projectPath)
	if err != nil {
		return nil, err
	}

	globalCfg, err := daemonconfig.Load()
	if err == nil && len(globalCfg.Agents) > 0 {
		cfg.MergeAgents(globalCfg.Agents)
	}

	return cfg, nil
}

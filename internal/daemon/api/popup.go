package api

import (
	"fmt"
	"strings"

	architectconfig "github.com/kareemaly/cortex/internal/architect/config"
	"github.com/kareemaly/cortex/internal/binpath"
	"github.com/kareemaly/cortex/internal/tmux"
)

func openCortexPopup(projectPath string, tmuxManager *tmux.Manager, args ...string) error {
	cortexPath, err := binpath.FindCortex()
	if err != nil {
		return fmt.Errorf("locate cortex binary: %w", err)
	}

	quotedArgs := make([]string, 0, len(args)+1)
	quotedArgs = append(quotedArgs, fmt.Sprintf("%q", cortexPath))
	for _, arg := range args {
		quotedArgs = append(quotedArgs, fmt.Sprintf("%q", arg))
	}

	tmuxSession := "cortex"
	if projectCfg, err := architectconfig.Load(projectPath); err == nil {
		tmuxSession = projectCfg.GetTmuxSessionName()
	}

	return tmuxManager.DisplayPopup(tmuxSession, projectPath, strings.Join(quotedArgs, " "))
}

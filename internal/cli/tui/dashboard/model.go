package dashboard

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/cli/tui/tuilog"
)

func New(client *sdk.Client, logBuf *tuilog.Buffer) Model {
	return Model{
		globalClient: client,
		loading:      true,
		sseContexts:  make(map[string]context.CancelFunc),
		sseChannels:  make(map[string]<-chan sdk.Event),
		sseBackoffs:  make(map[string]time.Duration),
		logBuf:       logBuf,
		logViewer:    tuilog.NewViewer(logBuf),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadProjects(), m.tickDuration(), m.startPollTicker())
}

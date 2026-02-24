package dashboard

import (
	"context"
	"time"

	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/cli/tui/tuilog"
)

const (
	sseInitialBackoff = 2 * time.Second
	sseMaxBackoff     = 30 * time.Second
	pollInterval      = 60 * time.Second
)

type rowKind int

const (
	rowProject rowKind = iota
	rowSession
)

type row struct {
	kind         rowKind
	projectIndex int
	ticketID     string
}

type projectData struct {
	project   sdk.ArchitectResponse
	tickets   *sdk.ListAllTicketsResponse
	architect *sdk.ArchitectStateResponse
	loading   bool
	err       error
}

func (pd projectData) isActive() bool {
	if pd.architect != nil && (pd.architect.State == "active" || pd.architect.State == "orphaned") {
		return true
	}
	if pd.tickets != nil {
		for _, t := range pd.tickets.Progress {
			if t.HasActiveSession {
				return true
			}
		}
	}
	return false
}

type Model struct {
	globalClient *sdk.Client
	projects     []projectData
	rows         []row
	cursor       int
	scrollOffset int

	sseContexts map[string]context.CancelFunc
	sseChannels map[string]<-chan sdk.Event
	sseBackoffs map[string]time.Duration

	width, height int
	ready         bool
	loading       bool
	err           error
	statusMsg     string
	statusIsError bool

	pendingG bool

	showUnlinkConfirm bool
	unlinkProjectPath string

	showKillConfirm bool
	killProjectPath string
	killSessionID   string
	killSessionName string
	killing         bool

	showArchitectModeModal   bool
	architectModeProjectPath string

	logBuf        *tuilog.Buffer
	logViewer     tuilog.Viewer
	showLogViewer bool
}

type ArchitectsLoadedMsg struct {
	Architects []sdk.ArchitectResponse
}

type ArchitectsErrorMsg struct {
	Err error
}

type ArchitectDetailLoadedMsg struct {
	ArchitectPath string
	Tickets       *sdk.ListAllTicketsResponse
	Architect     *sdk.ArchitectStateResponse
	Err           error
}

type SSEConnectedMsg struct {
	ArchitectPath string
	Ch            <-chan sdk.Event
	Cancel        context.CancelFunc
}

type SSEEventMsg struct {
	ArchitectPath string
}

type SpawnArchitectMsg struct {
	ArchitectPath string
	Err           error
}

type FocusSuccessMsg struct {
	Name string
}

type FocusErrorMsg struct {
	Err error
}

type UnlinkArchitectMsg struct {
	ArchitectPath string
	Err           error
}

type SessionKilledMsg struct {
	ArchitectPath string
}

type SessionKillErrorMsg struct {
	Err error
}

type SSEDisconnectedMsg struct {
	ArchitectPath string
}

type SSEReconnectTickMsg struct {
	ArchitectPath string
}

type PollTickMsg struct{}

type ClearStatusMsg struct{}

type TickMsg struct{}

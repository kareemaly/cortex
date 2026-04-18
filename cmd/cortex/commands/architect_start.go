package commands

import (
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/cli/tui/variant"
	"github.com/kareemaly/cortex/internal/tmux"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

var (
	architectStartMode    string
	architectStartDetach  bool
	architectStartVariant string
)

var architectStartCmd = &cobra.Command{
	Use:   "start [name]",
	Short: "Start or attach to an architect session",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runArchitectStart,
}

func init() {
	architectStartCmd.Flags().StringVar(&architectStartMode, "mode", "", "Session mode: normal, fresh, or resume")
	architectStartCmd.Flags().BoolVar(&architectStartDetach, "detach", false, "Start in detached mode")
	architectStartCmd.Flags().StringVar(&architectStartVariant, "variant", "", "Agent variant to use (must match an entry in cortex.yaml agents map)")
	architectCmd.AddCommand(architectStartCmd)
}

func runArchitectStart(cmd *cobra.Command, args []string) error {
	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	ensureDaemon()

	architectPath, err := resolveArchitectPath(name)
	if err != nil {
		return err
	}

	client := sdk.DefaultClient(architectPath)

	// Check daemon health
	if err := client.Health(); err != nil {
		return fmt.Errorf("daemon not running: %w\nStart it with: cortexd", err)
	}

	// Try to spawn architect
	resp, err := client.SpawnArchitect(architectStartMode, architectStartVariant)
	if err != nil {
		apiErr, ok := err.(*sdk.APIError)
		if ok && apiErr.IsOrphanedSession() {
			return fmt.Errorf("architect session is orphaned. Use --mode resume to continue or --mode fresh to start over\n  cortex architect start --mode fresh")
		}
		if ok && apiErr.IsVariantRequired() && !architectStartDetach &&
			(isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())) {
			variants, fetchErr := client.GetVariants()
			if fetchErr == nil && len(variants) > 0 {
				chosen, selErr := runVariantSelector(variants)
				if selErr != nil {
					return fmt.Errorf("failed to spawn architect: %w", err)
				}
				if chosen == "" {
					return fmt.Errorf("no variant selected")
				}
				resp, err = client.SpawnArchitect(architectStartMode, chosen)
				if err != nil {
					return fmt.Errorf("failed to spawn architect: %w", err)
				}
			} else {
				return fmt.Errorf("failed to spawn architect: %w", err)
			}
		} else {
			return fmt.Errorf("failed to spawn architect: %w", err)
		}
	}

	if architectStartDetach {
		fmt.Printf("Architect started in tmux session '%s', window '%s'\n", resp.TmuxSession, resp.TmuxWindow)
		return nil
	}

	// Attach to tmux session
	manager, err := tmux.NewManager()
	if err != nil {
		if tmux.IsNotInstalled(err) {
			return fmt.Errorf("tmux is required but not installed")
		}
		return fmt.Errorf("failed to create tmux manager: %w", err)
	}

	if err := manager.AttachSession(resp.TmuxSession); err != nil {
		var _ = errors.New // ensure errors is used
		return fmt.Errorf("failed to attach to session: %w", err)
	}
	return nil
}

type variantPrompt struct {
	m        variant.Model
	chosen   string
	canceled bool
}

func (p variantPrompt) Init() tea.Cmd { return nil }
func (p variantPrompt) View() string  { return p.m.View() }
func (p variantPrompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case variant.SelectedMsg:
		p.chosen = msg.Name
		return p, tea.Quit
	case variant.CancelledMsg:
		_ = msg
		p.canceled = true
		return p, tea.Quit
	}
	newM, cmd := p.m.Update(msg)
	p.m = newM
	return p, cmd
}

func runVariantSelector(variants []string) (string, error) {
	p := variantPrompt{m: variant.New("Select agent variant", variants)}
	result, err := tea.NewProgram(p).Run()
	if err != nil {
		return "", err
	}
	final := result.(variantPrompt)
	if final.canceled {
		return "", nil
	}
	return final.chosen, nil
}

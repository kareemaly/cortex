package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/spf13/cobra"
)

var projectsJSONFlag bool

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List registered projects",
	Long:  `List all projects registered in ~/.cortex/settings.yaml with their ticket counts.`,
	Run: func(cmd *cobra.Command, args []string) {
		client := sdk.DefaultClient("")

		resp, err := client.ListProjects()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if projectsJSONFlag {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if err := enc.Encode(resp); err != nil {
				fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
				os.Exit(1)
			}
			return
		}

		if len(resp.Projects) == 0 {
			fmt.Println("No projects registered. Run 'cortex init' in a project directory to register it.")
			return
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "TITLE\tPATH\tBACKLOG\tPROGRESS\tREVIEW\tDONE\tSTATUS")

		for _, p := range resp.Projects {
			status := "ok"
			backlog, progress, review, done := "-", "-", "-", "-"

			if !p.Exists {
				status = "stale"
			} else if p.Counts != nil {
				backlog = fmt.Sprintf("%d", p.Counts.Backlog)
				progress = fmt.Sprintf("%d", p.Counts.Progress)
				review = fmt.Sprintf("%d", p.Counts.Review)
				done = fmt.Sprintf("%d", p.Counts.Done)
			}

			title := p.Title
			if title == "" {
				title = "-"
			}

			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				title, p.Path, backlog, progress, review, done, status)
		}

		_ = w.Flush()
	},
}

func init() {
	projectsCmd.Flags().BoolVar(&projectsJSONFlag, "json", false, "Output as JSON")
	rootCmd.AddCommand(projectsCmd)
}

package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/kareemaly/cortex/internal/daemon/autostart"
	"github.com/spf13/cobra"
)

var daemonLogsFollowFlag bool

var daemonLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View daemon logs",
	Long: `View the Cortex daemon logs.

By default, shows the last 50 lines. Use -f to follow new output.

Examples:
  cortex daemon logs     # Show last 50 lines
  cortex daemon logs -f  # Follow log output`,
	Run: func(cmd *cobra.Command, args []string) {
		logPath, err := autostart.LogFilePath()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Check if log file exists
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			fmt.Println("No daemon logs found")
			return
		}

		if daemonLogsFollowFlag {
			// Follow mode using tail -f
			tailLogs(logPath)
		} else {
			// Show last 50 lines
			showLastLines(logPath, 50)
		}
	},
}

func init() {
	daemonLogsCmd.Flags().BoolVarP(&daemonLogsFollowFlag, "follow", "f", false, "Follow log output")
	daemonCmd.AddCommand(daemonLogsCmd)
}

// showLastLines reads and displays the last n lines of a file.
func showLastLines(path string, n int) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = file.Close() }()

	// Read all lines (for simplicity; could optimize for large files)
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading log file: %v\n", err)
		os.Exit(1)
	}

	// Print last n lines
	start := 0
	if len(lines) > n {
		start = len(lines) - n
	}

	for _, line := range lines[start:] {
		fmt.Println(line)
	}
}

// tailLogs uses tail -f to follow the log file.
func tailLogs(path string) {
	cmd := exec.Command("tail", "-f", "-n", "50", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Handle Ctrl+C gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Wait for signal or command to finish
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-sigChan:
		// Kill tail process on interrupt
		_ = cmd.Process.Kill()
		// Drain done channel
		<-done
	case err := <-done:
		if err != nil {
			// Ignore error from tail (e.g., if interrupted)
			if exitErr, ok := err.(*exec.ExitError); ok {
				if exitErr.ExitCode() != -1 {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				}
			}
		}
	}
}

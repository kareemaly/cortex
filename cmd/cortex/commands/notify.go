package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/kareemaly/cortex/internal/notifications"
	"github.com/spf13/cobra"
)

var (
	notifySound   bool
	notifyMessage string
)

var notifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "Notification management commands",
	Long:  `Commands for testing and managing notifications.`,
}

var notifyTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Send a test notification",
	Long:  `Sends a test desktop notification to verify the notification system is working.`,
	RunE:  runNotifyTest,
}

func init() {
	rootCmd.AddCommand(notifyCmd)
	notifyCmd.AddCommand(notifyTestCmd)

	notifyTestCmd.Flags().BoolVar(&notifySound, "sound", false, "Play sound with notification")
	notifyTestCmd.Flags().StringVarP(&notifyMessage, "message", "m", "", "Custom message for the notification")
}

func runNotifyTest(cmd *cobra.Command, args []string) error {
	// Create a quiet logger for this command
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))

	// Create local channel
	ch := notifications.NewLocalChannel(logger)

	// Check availability
	if !ch.Available() {
		return fmt.Errorf("no notification tool available\n\nTo enable notifications, install one of:\n  - macOS: brew install terminal-notifier\n  - Linux: apt install libnotify-bin (notify-send)")
	}

	// Build notification
	title := "Cortex Test Notification"
	body := "Notifications are working correctly!"
	if notifyMessage != "" {
		body = notifyMessage
	}

	notification := notifications.Notification{
		Title: title,
		Body:  body,
		Sound: notifySound,
	}

	// Send notification
	if err := ch.Send(context.Background(), notification); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	fmt.Println("Test notification sent successfully!")
	return nil
}

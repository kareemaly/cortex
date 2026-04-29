package commands

import (
	"fmt"
	"strings"
	"time"
)

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func formatDetailTime(ts time.Time) string {
	if ts.IsZero() {
		return "-"
	}
	return ts.Local().Format("Jan 2, 2006 15:04")
}

func formatDetailOptionalTime(ts *time.Time) string {
	if ts == nil || ts.IsZero() {
		return "-"
	}
	return formatDetailTime(*ts)
}

func formatDetailDuration(start, end time.Time) string {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return "-"
	}

	d := end.Sub(start).Round(time.Second)
	if d < time.Minute {
		return d.String()
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}

func markdownList(items []string, empty string) string {
	if len(items) == 0 {
		return empty
	}

	var b strings.Builder
	for _, item := range items {
		b.WriteString("- ")
		b.WriteString(item)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

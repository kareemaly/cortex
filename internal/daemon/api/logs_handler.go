package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	daemonconfig "github.com/kareemaly/cortex/internal/daemon/config"
	"github.com/kareemaly/cortex/pkg/version"
)

// LogsHandlers provides HTTP handlers for daemon logs and status.
type LogsHandlers struct {
	deps      *Dependencies
	startedAt time.Time
}

// NewLogsHandlers creates a new LogsHandlers with the given dependencies.
func NewLogsHandlers(deps *Dependencies) *LogsHandlers {
	return &LogsHandlers{
		deps:      deps,
		startedAt: time.Now(),
	}
}

// DaemonLogsResponse is the response for GET /daemon/logs.
type DaemonLogsResponse struct {
	Content string `json:"content"`
	Path    string `json:"path"`
}

// DaemonStatusResponse is the response for GET /daemon/status.
type DaemonStatusResponse struct {
	Status       string `json:"status"`
	Version      string `json:"version"`
	Uptime       string `json:"uptime"`
	ProjectCount int    `json:"project_count"`
}

// ReadDaemonLogs handles GET /daemon/logs - reads recent daemon logs.
func (h *LogsHandlers) ReadDaemonLogs(w http.ResponseWriter, r *http.Request) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get home directory")
		return
	}

	logsDir := filepath.Join(homeDir, ".cortex", "logs")

	// Find the most recent log file
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusOK, DaemonLogsResponse{Content: "(no logs found)", Path: logsDir})
			return
		}
		writeError(w, http.StatusInternalServerError, "read_error", "failed to read logs directory")
		return
	}

	// Find most recent .log file
	var latestLog string
	for i := len(entries) - 1; i >= 0; i-- {
		if strings.HasSuffix(entries[i].Name(), ".log") {
			latestLog = filepath.Join(logsDir, entries[i].Name())
			break
		}
	}

	if latestLog == "" {
		writeJSON(w, http.StatusOK, DaemonLogsResponse{Content: "(no log files found)", Path: logsDir})
		return
	}

	// Read the file (last N lines)
	lines := 100
	if linesParam := r.URL.Query().Get("lines"); linesParam != "" {
		if n, err := strconv.Atoi(linesParam); err == nil && n > 0 {
			lines = n
		}
	}

	data, err := os.ReadFile(latestLog)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read_error", "failed to read log file")
		return
	}

	content := string(data)
	allLines := strings.Split(content, "\n")
	if len(allLines) > lines {
		allLines = allLines[len(allLines)-lines:]
	}

	// Apply optional level filter
	level := r.URL.Query().Get("level")
	if level != "" {
		level = strings.ToUpper(level)
		var filtered []string
		for _, line := range allLines {
			if strings.Contains(strings.ToUpper(line), level) {
				filtered = append(filtered, line)
			}
		}
		allLines = filtered
	}

	writeJSON(w, http.StatusOK, DaemonLogsResponse{
		Content: strings.Join(allLines, "\n"),
		Path:    latestLog,
	})
}

// DaemonStatus handles GET /daemon/status - returns extended daemon status.
func (h *LogsHandlers) DaemonStatus(w http.ResponseWriter, r *http.Request) {
	uptime := time.Since(h.startedAt).Round(time.Second)

	// Count registered projects
	projectCount := 0
	cfg, err := daemonconfig.Load()
	if err == nil {
		projectCount = len(cfg.Projects)
	}

	writeJSON(w, http.StatusOK, DaemonStatusResponse{
		Status:       "ok",
		Version:      version.Version,
		Uptime:       uptime.String(),
		ProjectCount: projectCount,
	})
}

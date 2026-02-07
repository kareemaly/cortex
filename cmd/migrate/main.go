package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kareemaly/cortex/internal/storage"
	"github.com/kareemaly/cortex/internal/ticket"
)

// Old JSON format structs

type OldTicket struct {
	ID         string       `json:"id"`
	Type       string       `json:"type"`
	Title      string       `json:"title"`
	Body       string       `json:"body"`
	References []string     `json:"references"`
	Dates      OldDates     `json:"dates"`
	Comments   []OldComment `json:"comments"`
}

type OldDates struct {
	Created  time.Time  `json:"created"`
	Updated  time.Time  `json:"updated"`
	DueDate  *time.Time `json:"due_date"`
	Progress *time.Time `json:"progress"`
	Reviewed *time.Time `json:"reviewed"`
	Done     *time.Time `json:"done"`
}

type OldComment struct {
	ID        string          `json:"id"`
	SessionID string          `json:"session_id"`
	Type      string          `json:"type"`
	Content   string          `json:"content"`
	Action    *OldAction      `json:"action"`
	CreatedAt time.Time       `json:"created_at"`
}

type OldAction struct {
	Type string          `json:"type"`
	Args json.RawMessage `json:"args"`
}

func main() {
	projectPath := filepath.Join(os.Getenv("HOME"), "kesc")
	if len(os.Args) > 1 {
		projectPath = os.Args[1]
	}

	oldBase := filepath.Join(projectPath, ".cortex", "tickets")
	newBase := filepath.Join(projectPath, "tickets")

	statuses := []string{"backlog", "progress", "review", "done"}

	totalTickets := 0
	totalComments := 0

	for _, status := range statuses {
		statusDir := filepath.Join(oldBase, status)
		entries, err := os.ReadDir(statusDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			fmt.Fprintf(os.Stderr, "error reading %s: %v\n", statusDir, err)
			os.Exit(1)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}

			data, err := os.ReadFile(filepath.Join(statusDir, entry.Name()))
			if err != nil {
				fmt.Fprintf(os.Stderr, "error reading %s: %v\n", entry.Name(), err)
				os.Exit(1)
			}

			var old OldTicket
			if err := json.Unmarshal(data, &old); err != nil {
				fmt.Fprintf(os.Stderr, "error parsing %s: %v\n", entry.Name(), err)
				os.Exit(1)
			}

			// Build new ticket metadata
			meta := ticket.TicketMeta{
				ID:         old.ID,
				Title:      old.Title,
				Type:       old.Type,
				Tags:       []string{},
				References: old.References,
				Due:        old.Dates.DueDate,
				Created:    old.Dates.Created,
				Updated:    old.Dates.Updated,
			}

			// Create entity directory
			dirName := storage.DirName(old.Title, old.ID, "ticket")
			entityDir := filepath.Join(newBase, status, dirName)
			if err := os.MkdirAll(entityDir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "error creating dir %s: %v\n", entityDir, err)
				os.Exit(1)
			}

			// Write index.md
			indexData, err := storage.SerializeFrontmatter(&meta, old.Body)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error serializing %s: %v\n", old.Title, err)
				os.Exit(1)
			}
			if err := storage.AtomicWriteFile(filepath.Join(entityDir, "index.md"), indexData); err != nil {
				fmt.Fprintf(os.Stderr, "error writing index.md for %s: %v\n", old.Title, err)
				os.Exit(1)
			}

			// Write comments
			for _, oc := range old.Comments {
				commentMeta := storage.CommentMeta{
					ID:      oc.ID,
					Author:  "claude",
					Type:    storage.CommentType(oc.Type),
					Created: oc.CreatedAt,
				}

				if oc.Action != nil {
					var argsMap map[string]any
					if err := json.Unmarshal(oc.Action.Args, &argsMap); err != nil {
						fmt.Fprintf(os.Stderr, "warning: could not parse action args for comment %s: %v\n", oc.ID, err)
					}
					commentMeta.Action = &storage.CommentAction{
						Type: oc.Action.Type,
						Args: argsMap,
					}
				}

				commentData, err := storage.SerializeFrontmatter(&commentMeta, oc.Content)
				if err != nil {
					fmt.Fprintf(os.Stderr, "error serializing comment %s: %v\n", oc.ID, err)
					os.Exit(1)
				}

				filename := fmt.Sprintf("comment-%s.md", storage.ShortID(oc.ID))
				if err := storage.AtomicWriteFile(filepath.Join(entityDir, filename), commentData); err != nil {
					fmt.Fprintf(os.Stderr, "error writing comment %s: %v\n", filename, err)
					os.Exit(1)
				}

				totalComments++
			}

			totalTickets++
			fmt.Printf("  [%s] %s (%d comments)\n", status, old.Title, len(old.Comments))
		}
	}

	fmt.Printf("\nMigrated %d tickets, %d comments\n", totalTickets, totalComments)
}

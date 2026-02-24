package spawn

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kareemaly/cortex/internal/storage"
)

var tmuxNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func validateTmuxName(name string) error {
	if len(name) > 128 {
		return fmt.Errorf("exceeds maximum length of 128 characters")
	}
	if strings.HasPrefix(name, "-") {
		return fmt.Errorf("cannot start with a hyphen")
	}
	if strings.ContainsAny(name, ":.") {
		return fmt.Errorf("cannot contain colons or periods (tmux delimiters)")
	}
	if !tmuxNameRegex.MatchString(name) {
		return fmt.Errorf("must contain only alphanumeric characters, underscores, and hyphens")
	}
	return nil
}

func validateGitRepository(repoPath string) error {
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return &ConfigError{
			Field:   "Repo",
			Message: fmt.Sprintf("repository directory does not exist: %s", repoPath),
		}
	}

	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return &ConfigError{
			Field:   "Repo",
			Message: fmt.Sprintf("not a git repository: %s", repoPath),
		}
	}

	return nil
}

func getWorkingDirectory(req SpawnRequest) (string, error) {
	if req.AgentType == AgentTypeCollabAgent {
		if req.Repo != "" {
			repo := req.Repo
			if strings.HasPrefix(repo, "~/") {
				if home, err := os.UserHomeDir(); err == nil {
					repo = filepath.Join(home, repo[2:])
				}
			}
			if err := validateGitRepository(repo); err != nil {
				return "", err
			}
			return repo, nil
		}
		return req.ArchitectPath, nil
	}

	if req.AgentType != AgentTypeTicketAgent {
		return req.ArchitectPath, nil
	}

	if req.Ticket == nil {
		return req.ArchitectPath, nil
	}

	if req.Ticket.Type == "work" {
		if req.Ticket.Repo != "" {
			repo := req.Ticket.Repo
			if strings.HasPrefix(repo, "~/") {
				if home, err := os.UserHomeDir(); err == nil {
					repo = filepath.Join(home, repo[2:])
				}
			}
			if err := validateGitRepository(repo); err != nil {
				return "", err
			}
			return repo, nil
		}
		return req.ArchitectPath, nil
	}

	if req.Ticket.Type == "research" {
		if req.Ticket.Path != "" {
			path := req.Ticket.Path
			if strings.HasPrefix(path, "~/") {
				if home, err := os.UserHomeDir(); err == nil {
					path = filepath.Join(home, path[2:])
				}
			}
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return "", &ConfigError{
					Field:   "Path",
					Message: fmt.Sprintf("research path directory does not exist: %s", path),
				}
			}
			return path, nil
		}
		return req.ArchitectPath, nil
	}

	return req.ArchitectPath, nil
}

func (s *Spawner) validateSpawnRequest(req SpawnRequest) error {
	if req.TmuxSession == "" {
		return &ConfigError{Field: "TmuxSession", Message: "cannot be empty"}
	}

	if err := validateTmuxName(req.TmuxSession); err != nil {
		return &ConfigError{Field: "TmuxSession", Message: err.Error()}
	}

	if req.ArchitectPath != "" {
		if _, err := os.Stat(req.ArchitectPath); os.IsNotExist(err) {
			return &ConfigError{Field: "ProjectPath", Message: "directory does not exist"}
		}
	}

	if req.AgentType == AgentTypeTicketAgent {
		if req.TicketID == "" {
			return &ConfigError{Field: "TicketID", Message: "cannot be empty for ticket agent"}
		}
		if req.Ticket == nil {
			return &ConfigError{Field: "Ticket", Message: "cannot be nil for ticket agent"}
		}
	}

	if req.AgentType == AgentTypeArchitect {
		if req.ArchitectName == "" {
			return &ConfigError{Field: "ArchitectName", Message: "cannot be empty for architect"}
		}
	}

	if req.AgentType == AgentTypeCollabAgent {
		if req.CollabID == "" {
			return &ConfigError{Field: "CollabID", Message: "cannot be empty for collab agent"}
		}
	}

	return nil
}

func (s *Spawner) generateWindowName(req SpawnRequest) string {
	if req.AgentType == AgentTypeTicketAgent && req.Ticket != nil {
		return GenerateWindowName(req.Ticket.Title)
	}
	if req.AgentType == AgentTypeCollabAgent && req.CollabID != "" {
		return "collab-" + storage.ShortID(req.CollabID)
	}
	return "architect"
}

package install

import "os/exec"

var requiredDeps = []string{"tmux", "git"}

func CheckDependencies() []DependencyResult {
	results := make([]DependencyResult, 0, len(requiredDeps))
	for _, name := range requiredDeps {
		result := DependencyResult{Name: name}
		path, err := exec.LookPath(name)
		if err == nil {
			result.Available = true
			result.Path = path
		}
		results = append(results, result)
	}
	return results
}

func DetectCompanion() string {
	if _, err := exec.LookPath("lazygit"); err == nil {
		return "lazygit"
	}
	return "vim"
}

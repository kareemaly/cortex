package install

import "os/exec"

// requiredDeps lists the dependencies to check.
var requiredDeps = []string{"tmux", "git"}

// CheckDependencies checks for required external dependencies.
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

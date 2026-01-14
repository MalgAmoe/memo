package internal

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GetProject returns the current project name from git or directory
func GetProject() string {
	// Try git first
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err == nil {
		toplevel := strings.TrimSpace(string(out))
		return filepath.Base(toplevel)
	}

	// Fallback to current directory name
	cwd, err := os.Getwd()
	if err != nil {
		return "unknown"
	}
	return filepath.Base(cwd)
}

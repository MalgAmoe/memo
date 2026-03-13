package internal

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	// Find .env relative to the memo binary's location
	exe, err := os.Executable()
	if err != nil {
		return
	}
	envPath := filepath.Join(filepath.Dir(exe), ".env")
	loadEnvFile(envPath)
}

func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		// Don't override existing env vars
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

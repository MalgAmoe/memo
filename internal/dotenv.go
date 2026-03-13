package internal

import "os"

// Set at build time with: go build -ldflags "-X memo/internal.defaultFireworksKey=fw_xxx"
var defaultFireworksKey string

func init() {
	if defaultFireworksKey != "" && os.Getenv("FIREWORKS_API_KEY") == "" {
		os.Setenv("FIREWORKS_API_KEY", defaultFireworksKey)
	}
}

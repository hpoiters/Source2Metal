package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// applicationDir keeps the launcher, INI and utility output anchored to the
// user-visible executable directory, even when the embedded core is temporary.
func applicationDir() (string, error) {
	if dir := strings.TrimSpace(os.Getenv("SOURCE2METAL_LAUNCH_DIR")); dir != "" {
		absolute, err := filepath.Abs(dir)
		if err != nil {
			return "", fmt.Errorf("invalid %s: %w", "SOURCE2METAL_LAUNCH_DIR", err)
		}
		return filepath.Clean(absolute), nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	absolute, err := filepath.Abs(exe)
	if err != nil {
		return "", err
	}
	return filepath.Dir(absolute), nil
}

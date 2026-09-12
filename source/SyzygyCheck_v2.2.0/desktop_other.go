//go:build !windows

package main

import (
	"os"
	"path/filepath"
)

func desktopPath() (string, error) {
	h, e := os.UserHomeDir()
	if e != nil {
		return "", e
	}
	return filepath.Join(h, "Desktop"), nil
}

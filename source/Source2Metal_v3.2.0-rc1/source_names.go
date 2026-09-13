package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func shortSourceID(sourcePath string) string {
	h := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(sourcePath))))
	return fmt.Sprintf("%x", h[:4])
}

// uniqueSourceFile keeps the friendly name for the first source. If another
// source with the same base/kind would collide in the same run, a stable short
// source id derived from its original path is added instead of overwriting.
func uniqueSourceFile(dir, base string, kind SourceKind, product, sourcePath string) string {
	name := fmt.Sprintf("%s [%s] - %s.pgn", safeComponent(base), kind, product)
	p := filepath.Join(dir, name)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return p
	}
	name = fmt.Sprintf("%s [%s] [%s] - %s.pgn", safeComponent(base), kind, shortSourceID(sourcePath), product)
	return filepath.Join(dir, name)
}

func uniqueSourceDir(parent, base string, kind SourceKind, sourcePath string) string {
	name := fmt.Sprintf("%s [%s]", safeComponent(base), kind)
	p := filepath.Join(parent, name)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return p
	}
	name = fmt.Sprintf("%s [%s] [%s]", safeComponent(base), kind, shortSourceID(sourcePath))
	return filepath.Join(parent, name)
}

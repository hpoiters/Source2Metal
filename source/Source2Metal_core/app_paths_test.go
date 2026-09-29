package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplicationDirectoryAllLanguageTransitions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Gebruiker Тест 测试")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SOURCE2METAL_LAUNCH_DIR", dir)
	oldLanguage := activeLanguage
	defer setLanguage(oldLanguage)
	for _, from := range []string{"en", "de", "nl", "fr", "es", "zh", "ru"} {
		for _, to := range []string{"en", "de", "nl", "fr", "es", "zh", "ru"} {
			setLanguage(from)
			if err := saveLanguage(from); err != nil {
				t.Fatal(err)
			}
			setLanguage(to)
			if err := saveLanguage(to); err != nil {
				t.Fatal(err)
			}
			if got, err := applicationDir(); err != nil || got != dir {
				t.Fatalf("%s -> %s: %s %v", from, to, got, err)
			}
			if got, err := languageConfigPath(); err != nil || got != filepath.Join(dir, "Source2Metal.ini") {
				t.Fatal(got, err)
			}
			if got, ok := readSavedLanguage(); !ok || got != to {
				t.Fatal(got, ok)
			}
			got, err := extractSyzygyPackage()
			if err != nil || got != filepath.Join(dir, syzygyCheckPackageFilename) {
				t.Fatal(got, err)
			}
		}
	}
}

func TestDirectApplicationDirectory(t *testing.T) {
	t.Setenv("SOURCE2METAL_LAUNCH_DIR", "")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	got, err := applicationDir()
	if err != nil || got != filepath.Dir(exe) {
		t.Fatal(got, err)
	}
}

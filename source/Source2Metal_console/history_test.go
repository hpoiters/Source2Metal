package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHistoryPersistsAndLearns(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "book.bin")
	hist := filepath.Join(dir, "ETA_history.json")
	os.WriteFile(bin, make([]byte, 16000), 0600)
	p := newProgressWriter(&bytes.Buffer{}, true)
	p.historyPath = hist
	p.observeNormalLine("BIN RAW: " + bin)
	p.followupStart = time.Now().Add(-time.Hour)
	p.observeNormalLine("BIN RAW: OK | 123 | output.pgn")
	q := newProgressWriter(&bytes.Buffer{}, true)
	q.historyPath = hist
	q.observeNormalLine("BIN RAW: " + bin)
	if q.roughETA.referenceSeconds < 3599 || q.roughETA.referenceSeconds > 3602 {
		t.Fatal(q.roughETA)
	}
	q.roughETA.update(0, q.totalRecords, locNL)
	eta, _ := q.roughETA.update(20*time.Minute, q.totalRecords, locNL)
	if eta != "40 minuten" {
		t.Fatal(eta)
	}
	p.observeNormalLine("BIN RAW: OK | duplicate")
	if len(loadHistory(hist).Samples) != 1 {
		t.Fatal("duplicate counted")
	}
}
func TestHistoryFailureAndCorruption(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "book.bin")
	hist := filepath.Join(dir, "history.json")
	os.WriteFile(bin, make([]byte, 1600), 0600)
	p := newProgressWriter(&bytes.Buffer{}, true)
	p.historyPath = hist
	p.observeNormalLine("BIN RAW: " + bin)
	p.followupStart = time.Now().Add(-time.Hour)
	p.observeNormalLine("BIN RAW: ERROR | failure")
	p.CloseDisplay()
	if _, err := os.Stat(hist); !os.IsNotExist(err) {
		t.Fatal("failed run saved")
	}
	os.WriteFile(hist, []byte("broken"), 0600)
	if loadHistory(hist).reference(sourceSample(bin)) != 0 {
		t.Fatal("corrupt history used")
	}
}
func TestHistorySourceAndMachineIsolation(t *testing.T) {
	s := etaSample{Fingerprint: "a", Machine: "pc", Bytes: 1600, Records: 100, Seconds: 3600}
	h := etaHistory{Version: 1, Samples: []etaSample{s}}
	other := s
	other.Machine = "other"
	if h.reference(other) != 0 {
		t.Fatal("other machine")
	}
	other = s
	other.Bytes = 160000
	if h.reference(other) != 0 {
		t.Fatal("incomparable size")
	}
	newer := s
	newer.Seconds = 1800
	h.Samples = append(h.Samples, newer)
	if h.reference(s) != 2400 {
		t.Fatal("recent run not weighted")
	}
}
func TestUserFolderReuse(t *testing.T) {
	local, roaming := t.TempDir(), t.TempDir()
	t.Setenv("LOCALAPPDATA", local)
	t.Setenv("APPDATA", roaming)
	os.Mkdir(filepath.Join(roaming, "Source2Metal"), 0700)
	if got := userHistoryPath(); got != filepath.Join(roaming, "Source2Metal", "ETA_history.json") {
		t.Fatal(got)
	}
	os.Mkdir(filepath.Join(local, "Source2Metal"), 0700)
	if got := userHistoryPath(); !strings.HasPrefix(got, local) {
		t.Fatal(got)
	}
}
func TestChangedFileNotLearned(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "book.bin")
	hist := filepath.Join(dir, "history.json")
	os.WriteFile(bin, make([]byte, 1600), 0600)
	p := newProgressWriter(&bytes.Buffer{}, true)
	p.historyPath = hist
	p.observeNormalLine("BIN RAW: " + bin)
	p.followupStart = time.Now().Add(-time.Hour)
	os.WriteFile(bin, make([]byte, 3200), 0600)
	p.observeNormalLine("BIN RAW: OK | done")
	if _, err := os.Stat(hist); !os.IsNotExist(err) {
		t.Fatal("changed file learned")
	}
}

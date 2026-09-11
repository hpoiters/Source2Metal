package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const checkpointFileName = "SyzygyCheck_CHECKPOINT.json"

type checkpointEntry struct {
	Status string `json:"status"` // OK, FAIL, ERROR
	Size   int64  `json:"size"`
	Detail string `json:"detail,omitempty"`
}

type checkpointState struct {
	Format        int                        `json:"format"`
	InventoryHash string                     `json:"inventory_hash"`
	TotalFiles    int                        `json:"total_files"`
	TotalBytes    int64                      `json:"total_bytes"`
	Created       string                     `json:"created"`
	Updated       string                     `json:"updated"`
	Results       map[string]checkpointEntry `json:"results"`
}

func checkpointPath(dir string) string { return filepath.Join(dir, checkpointFileName) }

func inventoryHash(dir string, files []item) (string, error) {
	h := sha256.New()
	for _, f := range files {
		st, err := os.Stat(filepath.Join(dir, f.name))
		if err != nil {
			return "", err
		}
		// Name, exact size and nanosecond modification time make accidental
		// reuse after a replaced/changed tablebase very unlikely.
		fmt.Fprintf(h, "%s\x00%d\x00%d\n", strings.ToLower(f.name), st.Size(), st.ModTime().UnixNano())
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func loadMatchingCheckpoint(dir, hash string, totalFiles int, totalBytes int64) (*checkpointState, bool) {
	b, err := os.ReadFile(checkpointPath(dir))
	if err != nil {
		return nil, false
	}
	var cp checkpointState
	if json.Unmarshal(b, &cp) != nil || cp.Format != 1 || cp.InventoryHash != hash || cp.TotalFiles != totalFiles || cp.TotalBytes != totalBytes {
		return nil, false
	}
	if cp.Results == nil {
		cp.Results = map[string]checkpointEntry{}
	}
	return &cp, true
}

func newCheckpoint(hash string, totalFiles int, totalBytes int64) *checkpointState {
	now := time.Now().Format(time.RFC3339)
	return &checkpointState{Format: 1, InventoryHash: hash, TotalFiles: totalFiles, TotalBytes: totalBytes, Created: now, Updated: now, Results: map[string]checkpointEntry{}}
}

func saveCheckpoint(dir string, cp *checkpointState) error {
	cp.Updated = time.Now().Format(time.RFC3339)
	b, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return err
	}
	p := checkpointPath(dir)
	tmp := p + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	if e := f.Close(); err == nil {
		err = e
	}
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	_ = os.Remove(p)
	return os.Rename(tmp, p)
}

func removeCheckpoint(dir string) {
	_ = os.Remove(checkpointPath(dir))
	_ = os.Remove(checkpointPath(dir) + ".tmp")
}

func checkpointSummary(cp *checkpointState, files []item) (doneFiles int, doneBytes int64, failures []string, oldErrors map[string]string) {
	sizeByName := make(map[string]int64, len(files))
	for _, f := range files {
		sizeByName[strings.ToLower(f.name)] = f.size
	}
	oldErrors = map[string]string{}
	for name, e := range cp.Results {
		size, exists := sizeByName[strings.ToLower(name)]
		if !exists {
			continue
		}
		switch e.Status {
		case "OK":
			doneFiles++
			doneBytes += size
		case "FAIL":
			doneFiles++
			doneBytes += size
			failures = append(failures, name)
		case "ERROR":
			oldErrors[name] = e.Detail // retried when resuming
		}
	}
	sort.Strings(failures)
	return
}

func checkpointPending(cp *checkpointState, files []item) []item {
	out := make([]item, 0, len(files))
	for _, f := range files {
		e, ok := cp.Results[f.name]
		if !ok {
			e, ok = cp.Results[strings.ToLower(f.name)]
		}
		if ok && (e.Status == "OK" || e.Status == "FAIL") {
			continue
		}
		out = append(out, f)
	}
	return out
}

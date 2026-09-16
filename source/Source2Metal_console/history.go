package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// History is advisory only. Never alter converter input or results.
// Sampled identity avoids reading a multi-GB source a second time.
type etaSample struct {
	Fingerprint string
	Machine     string
	Bytes       int64
	Records     int64
	Seconds     float64
	ScanSeconds float64
	Completed   time.Time
}
type etaHistory struct {
	Version int
	Samples []etaSample
}

func machineKey() string {
	host, _ := os.Hostname()
	return host + "/" + runtime.GOOS + "/" + runtime.GOARCH + "/" + os.Getenv("PROCESSOR_IDENTIFIER")
}
func sourceSample(path string) etaSample {
	f, err := os.Open(path)
	if err != nil {
		return etaSample{}
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.Size() <= 0 || st.Size()%16 != 0 {
		return etaSample{}
	}
	h := sha256.New()
	for _, off := range []int64{0, st.Size() / 2, max(int64(0), st.Size()-65536)} {
		if _, err = f.Seek(off, io.SeekStart); err != nil {
			return etaSample{}
		}
		if _, err = io.CopyN(h, f, min(int64(65536), st.Size()-off)); err != nil {
			return etaSample{}
		}
	}
	return etaSample{Fingerprint: hex.EncodeToString(h.Sum(nil)), Machine: machineKey(), Bytes: st.Size(), Records: st.Size() / 16}
}
func loadHistory(path string) etaHistory {
	var h etaHistory
	f, err := os.Open(path)
	if err != nil {
		return etaHistory{Version: 1}
	}
	defer f.Close()
	if json.NewDecoder(io.LimitReader(f, 1<<20)).Decode(&h) != nil || h.Version != 1 || len(h.Samples) > 64 {
		return etaHistory{Version: 1}
	}
	valid := h.Samples[:0]
	for _, s := range h.Samples {
		if s.Bytes > 0 && s.Records == s.Bytes/16 && s.ScanSeconds >= 0 && s.ScanSeconds < 365*24*3600 && s.Seconds > 0 && s.Seconds < 365*24*3600 && !math.IsNaN(s.Seconds) && !math.IsInf(s.Seconds, 0) && s.Fingerprint != "" {
			valid = append(valid, s)
		}
	}
	h.Samples = valid
	return h
}
func (h etaHistory) reference(s etaSample) float64 {
	if s.Fingerprint == "" {
		return 0
	}
	// Recency-weighted successful measurements for the same sampled source.
	total, weight := 0.0, 0.0
	for _, old := range h.Samples {
		if old.Machine == s.Machine && old.Bytes == s.Bytes && old.Fingerprint == s.Fingerprint {
			total = total*0.5 + old.Seconds
			weight = weight*0.5 + 1
		}
	}
	if weight > 0 {
		return total / weight
	}
	// Different books may have very different reachable trees. Limit scaling
	// to reasonably similar sizes; this remains only a rough initial reference.
	best, score := 0.0, math.Inf(1)
	for _, old := range h.Samples {
		ratio := float64(s.Bytes) / float64(old.Bytes)
		if old.Machine == s.Machine && ratio >= 0.5 && ratio <= 2 {
			d := math.Abs(math.Log(ratio))
			if d <= score {
				score = d
				best = old.Seconds * ratio
			}
		}
	}
	return best
}
func saveSample(path string, s etaSample) error {
	h := loadHistory(path)
	h.Samples = append(h.Samples, s)
	if len(h.Samples) > 64 {
		h.Samples = h.Samples[len(h.Samples)-64:]
	}
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".s2m-eta-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
func (p *progressWriter) rememberSuccess() {
	if p.historyPath == "" || p.sample.Fingerprint == "" || p.followupStart.IsZero() {
		return
	}
	s := p.sample
	// Confirm the source metadata was not changed during conversion.
	st, err := os.Stat(p.binPath)
	if err != nil || st.Size() != s.Bytes || !st.ModTime().Equal(p.sourceModified) {
		return
	}
	s.ScanSeconds = p.roughETA.start.Seconds()
	s.Seconds = time.Since(p.followupStart).Seconds()
	s.Completed = time.Now().UTC()
	if s.Seconds > 0 {
		_ = saveSample(p.historyPath, s)
	}
	// A duplicate OK line must not count as a second measurement.
	p.sample = etaSample{}
}

// Reuse an existing per-user Source2Metal directory; otherwise create Local.
// Never tie history to the name or location of a test executable.
func userHistoryPath() string {
	roots := []string{os.Getenv("LOCALAPPDATA"), os.Getenv("APPDATA")}
	if cfg, err := os.UserConfigDir(); err == nil {
		roots = append(roots, cfg)
	}
	for _, root := range roots {
		if root == "" {
			continue
		}
		dir := filepath.Join(root, "Source2Metal")
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			return filepath.Join(dir, "ETA_history.json")
		}
	}
	for _, root := range roots {
		if root == "" {
			continue
		}
		dir := filepath.Join(root, "Source2Metal")
		if os.MkdirAll(dir, 0700) == nil {
			return filepath.Join(dir, "ETA_history.json")
		}
	}
	return ""
}

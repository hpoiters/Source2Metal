package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRoughETAFiveMinuteBoundary(t *testing.T) {
	var e roughEstimate
	for _, sec := range []int{0, 1, 299} {
		eta, note := e.update(time.Duration(sec)*time.Second, 43750000, locNL)
		if eta != "" || note != "" {
			t.Fatalf("premature estimate at %d", sec)
		}
	}
	eta, note := e.update(300*time.Second, 43750000, locNL)
	if eta != "1 uur" || !strings.Contains(note, "kan sterk afwijken") {
		t.Fatal(eta, note)
	}
	eta, _ = e.update(20*time.Minute, 43750000, locNL)
	if eta != "40 minuten" {
		t.Fatal(eta)
	}
	eta, note = e.update(time.Hour, 43750000, locNL)
	if eta != "" || !strings.Contains(note, "overschreden") {
		t.Fatal(eta, note)
	}
}

func TestRoughETAStartsAfterScanAndResets(t *testing.T) {
	var e roughEstimate
	e.update(10*time.Minute, 43750000, locNL)
	if eta, _ := e.update(14*time.Minute+59*time.Second, 43750000, locNL); eta != "" {
		t.Fatal(eta)
	}
	if eta, _ := e.update(15*time.Minute, 43750000, locNL); eta == "" {
		t.Fatal("missing estimate")
	}
	p := newProgressWriter(&bytes.Buffer{}, true)
	p.roughETA = e
	p.observeNormalLine("BIN RAW: " + filepath.Join(t.TempDir(), "missing.bin"))
	if p.roughETA.started {
		t.Fatal("previous timing reused")
	}
}

func TestRoughETARounding(t *testing.T) {
	for _, c := range []struct {
		m    int
		want string
	}{{15, "15 minuten"}, {45, "45 minuten"}, {60, "1 uur"}, {90, "1½ uur"}, {120, "2 uur"}, {240, "4 uur"}} {
		if got := coarseDuration(c.m, locNL); got != c.want {
			t.Fatal(got, c.want)
		}
	}
}

func TestOnlyETAAndExplanationChange(t *testing.T) {
	var out bytes.Buffer
	p := newProgressWriter(&out, true)
	p.width = func() int { return 120 }
	p.totalRecords = 43750000
	frame := `/ BIN2PGN: Bezig met verwerken... | BIN-records: 43.750.000 | actief: 5m0s`
	p.roughETA = roughEstimate{started: true, start: time.Second}
	p.renderProgress(frame)
	before := strings.Split(out.String(), "\r\n")
	if len(before) != 4 || before[3] != "\r\x1b[2K" {
		t.Fatal(before)
	}
	out.Reset()
	p.roughETA.start = 0
	p.renderProgress(frame)
	after := strings.Split(strings.TrimPrefix(out.String(), "\r\x1b[3A"), "\r\n")
	for i := 0; i < 3; i++ {
		if before[i] != after[i] {
			t.Fatalf("non-ETA row %d changed", i)
		}
	}
	if len(after) != 6 || !strings.Contains(after[4], "circa 1 uur") || after[3] != "\r\x1b[2K" || !strings.HasSuffix(after[4], "; wordt bijgesteld.") || after[5] != "\r\x1b[2K" {
		t.Fatal(after)
	}
}

func TestRoughETASixFixedRowsCoreProtocol(t *testing.T) {
	phrases := []string{"Work in progress...", "Verarbeitung läuft...", "Bezig met verwerken...", "Traitement en cours...", "Procesando...", "正在处理...", "Идёт обработка..."}
	for _, width := range []int{40, 60, 80, 100, 120, 200} {
		for _, phrase := range phrases {
			for _, chunk := range []int{1, 7, 4096} {
				var out bytes.Buffer
				p := newProgressWriter(&out, true)
				p.width = func() int { return width }
				p.totalRecords = 43750000
				var input strings.Builder
				for i := 0; i < 3700; i += 10 {
					fmt.Fprintf(&input, "\r\x1b[2K\r%c BIN2PGN: %s | BIN records: 43750000 | active: %ds", `\|/-`[(i/10)%4], phrase, i)
				}
				input.WriteString("\r")
				data := input.String()
				for i := 0; i < len(data); i += chunk {
					end := i + chunk
					if end > len(data) {
						end = len(data)
					}
					p.Write([]byte(data[i:end]))
				}
				row, max := screenExtent(t, out.String(), width)
				if row != 5 || max != 5 {
					t.Fatalf("width=%d language=%s chunk=%d row=%d max=%d", width, phrase, chunk, row, max)
				}
				p.Write([]byte("BIN RAW: OK | out.pgn\nKeuze: "))
				p.CloseDisplay()
				if !strings.HasSuffix(out.String(), "BIN RAW: OK | out.pgn\nKeuze: ") {
					t.Fatal("lost completion")
				}
			}
		}
	}
}

func TestConsoleSourceMatchesApproved(t *testing.T) {
	data, err := os.ReadFile("../../validation/approved_console_hashes.json")
	if err != nil {
		t.Fatal(err)
	}
	var expected map[string]string
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatal(err)
	}
	for name, want := range expected {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		// Git for Windows may check out text with CRLF. Normalize only line endings.
		data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
		if name == "main.go" {
			data = bytes.ReplaceAll(data, []byte(buildMarker), []byte("2026-09-16_07-35_ETA_COMPACT_TEST"))
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != want {
			t.Fatalf("approved console source changed: %s", name)
		}
	}
}

func TestDynamicETASequence(t *testing.T) {
	var e roughEstimate
	e.update(0, 43750000, locNL)
	for _, c := range []struct {
		minute int
		want   string
	}{
		{5, "1 uur"}, {6, "50 minuten"}, {16, "40 minuten"},
		{26, "30 minuten"}, {36, "25 minuten"}, {41, "20 minuten"},
		{46, "15 minuten"}, {51, "10 minuten"}, {56, "5 minuten"},
	} {
		got, _ := e.update(time.Duration(c.minute)*time.Minute, 43750000, locNL)
		if got != c.want {
			t.Fatalf("minute %d: %q, want %q", c.minute, got, c.want)
		}
	}
}

func TestDynamicETACadenceAndLanguage(t *testing.T) {
	var e roughEstimate
	e.update(0, 43750000, locNL)
	e.update(5*time.Minute, 43750000, locNL)
	got, _ := e.update(5*time.Minute+59*time.Second, 43750000, locNL)
	if got != "1 uur" {
		t.Fatal("updated before minute boundary", got)
	}
	got, _ = e.update(6*time.Minute, 43750000, locNL)
	if got != "50 minuten" {
		t.Fatal(got)
	}
	got, _ = e.update(6*time.Minute+time.Second, 43750000, locEN)
	if got != "50 minutes" {
		t.Fatal("language not refreshed", got)
	}
	for loc := locEN; loc <= locRU; loc++ {
		for _, m := range []int{5, 10, 20, 30, 40, 50, 60, 70, 80, 90, 120} {
			if coarseDuration(m, loc) == "" {
				t.Fatal(loc, m)
			}
		}
	}
}

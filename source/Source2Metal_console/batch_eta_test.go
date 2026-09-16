package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupBatch(t *testing.T) (*progressWriter, []string, *bytes.Buffer) {
	t.Helper()
	dir := t.TempDir()
	var out bytes.Buffer
	p := newProgressWriter(&out, true)
	p.width = func() int { return 140 }
	p.historyPath = filepath.Join(dir, "history.json")
	p.observeNormalLine("Root: " + dir)
	var paths []string
	for i, minutes := range []int{60, 10, 20} {
		path := filepath.Join(dir, fmt.Sprintf("book %d.bin", i))
		data := bytes.Repeat([]byte{byte(i + 1)}, 1600)
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		s := sourceSample(path)
		s.Seconds = float64(minutes * 60)
		if err := saveSample(p.historyPath, s); err != nil {
			t.Fatal(err)
		}
		p.observeNormalLine(fmt.Sprintf("  %3d. BIN  %s", i+1, path))
		paths = append(paths, path)
	}
	return p, paths, &out
}

func TestBatchTotalTransitions(t *testing.T) {
	p, paths, _ := setupBatch(t)
	p.observeNormalLine("BIN RAW: " + paths[0])
	p.roughETA.update(0, 100, locNL)
	if got, ok := p.batchRemaining(15*time.Minute, 100); !ok || got != 75*60 {
		t.Fatalf("total %v %v", got, ok)
	}
	// Complete the first file; only its own measurement is learned.
	p.followupStart = time.Now().Add(-time.Hour)
	p.lastElapsed = time.Hour
	p.observeNormalLine("BIN RAW: OK | first")
	h := loadHistory(p.historyPath)
	if len(h.Samples) != 4 || h.Samples[3].Seconds < 3599 || h.Samples[3].Seconds > 3602 {
		t.Fatal(h)
	}
	p.observeNormalLine("BIN RAW: " + paths[1])
	p.roughETA.update(0, 100, locNL)
	if got, ok := p.batchRemaining(5*time.Minute, 100); !ok || got != 25*60 {
		t.Fatalf("next total %v %v", got, ok)
	}
	rows := p.batchRows([]string{"scan", "", "/ BIN2PGN: Bezig met verwerken... | actief: 5m0s", ""}, 5*time.Minute, 100, locNL)
	if !strings.Contains(rows[2], "bestand 2/3") || !strings.Contains(rows[2], "1h 05m") || !strings.Contains(rows[4], "25 minuten") {
		t.Fatal(rows)
	}
	// A pending book must not hide an active-file overrun.
	if _, ok := p.batchRemaining(11*time.Minute, 100); ok {
		t.Fatal("overrun hidden")
	}
	p.observeNormalLine("BIN RAW: " + filepath.Join(t.TempDir(), "missing.bin"))
	if p.batchIndex != -1 {
		t.Fatal("stale active file")
	}
}

func TestBatchUnknownAndScan(t *testing.T) {
	p, paths, _ := setupBatch(t)
	if err := os.Remove(p.historyPath); err != nil {
		t.Fatal(err)
	}
	p.observeNormalLine("BIN RAW: " + paths[0])
	// Current remaining scan (10s) plus all three size-based follow-up estimates.
	got, ok := p.batchRemaining(10*time.Second, 50)
	want := 10 + 3*1600.0/1.4e9*7200
	if !ok || got < want-0.00001 || got > want+0.00001 {
		t.Fatalf("%v %v want %v", got, ok, want)
	}
	p.batchSamples[2] = etaSample{}
	if _, ok := p.batchRemaining(10*time.Second, 50); ok {
		t.Fatal("unknown source silently excluded")
	}
}

func TestBatchConsoleSevenLanguages(t *testing.T) {
	phrases := []string{"Work in progress...", "Verarbeitung läuft...", "Bezig met verwerken...", "Traitement en cours...", "Procesando...", "正在处理...", "Идёт обработка..."}
	for loc, phrase := range phrases {
		for _, chunk := range []int{1, 7, 4096} {
			p, paths, out := setupBatch(t)
			// Exercise the byte stream through all three files, not just helper calls.
			out.Reset()
			var input strings.Builder
			for i, path := range paths {
				fmt.Fprintf(&input, "BIN RAW: %s\n", path)
				for seconds := 0; seconds <= 360; seconds += 10 {
					fmt.Fprintf(&input, "\r/ BIN2PGN: %s | BIN records: 100 | active: %ds\r\x1b[2K", phrase, seconds)
				}
				fmt.Fprintf(&input, "\rBIN RAW: OK | %d\n\n", i)
			}
			input.WriteString("BOOK RAW samenvoegen...\n")
			data := input.String()
			for i := 0; i < len(data); i += chunk {
				p.Write([]byte(data[i:min(i+chunk, len(data))]))
			}
			p.CloseDisplay()
			screen := visiblePhaseScreen(t, out.String())
			joined := strings.Join(screen, "\n")
			if strings.Contains(joined, "BIN2PGN:") || strings.Contains(joined, "circa") || !strings.Contains(joined, "BOOK RAW samenvoegen...") {
				t.Fatalf("loc %d chunk %d: %s", loc, chunk, joined)
			}
			if !strings.Contains(out.String(), "1/3") || !strings.Contains(out.String(), "2/3") || !strings.Contains(out.String(), "3/3") {
				t.Fatal("missing file counter")
			}
		}
	}
}

func TestBatchFixedRowsAtNarrowWidths(t *testing.T) {
	for _, width := range []int{40, 80, 120} {
		p, paths, out := setupBatch(t)
		p.width = func() int { return width }
		p.observeNormalLine("BIN RAW: " + paths[0])
		for seconds := 0; seconds < 900; seconds += 5 {
			p.renderProgress(fmt.Sprintf("/ BIN2PGN: Bezig met verwerken... | BIN records: 100 | actief: %ds", seconds))
		}
		row, maxRow := screenExtent(t, out.String(), width)
		if row != 5 || maxRow != 5 {
			t.Fatalf("width %d: row %d max %d", width, row, maxRow)
		}
	}
}

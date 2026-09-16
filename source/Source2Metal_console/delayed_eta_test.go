package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestDelayedPresentationAllLanguages(t *testing.T) {
	phrases := []string{"Work in progress...", "Verarbeitung läuft...", "Bezig met verwerken...", "Traitement en cours...", "Procesando...", "正在处理...", "Идёт обработка..."}
	for _, phrase := range phrases {
		var out bytes.Buffer
		p := newProgressWriter(&out, true)
		p.width = func() int { return 200 }
		p.totalRecords = 43750000
		frame := func(count, seconds int) {
			p.renderProgress(fmt.Sprintf("/ BIN2PGN: %s | BIN records: %d | active: %ds", phrase, count, seconds))
		}
		frame(100, 1)
		frame(43750000, 10)
		frame(43750000, 309)
		screen := strings.Join(visiblePhaseScreen(t, out.String()), "\n")
		if strings.Contains(screen, ": ~") || strings.Contains(screen, "  ") {
			t.Fatalf("premature ETA: %s", screen)
		}
		frame(43750000, 310)
		screen = strings.Join(visiblePhaseScreen(t, out.String()), "\n")
		if !strings.Contains(screen, "  ") {
			t.Fatalf("missing ETA: %s", screen)
		}
		frame(43750000, 4000)
		screen = strings.Join(visiblePhaseScreen(t, out.String()), "\n")
		if strings.Contains(screen, "  ") {
			t.Fatalf("stale ETA: %s", screen)
		}
	}
}

func TestDelayedBatchNoPlaceholder(t *testing.T) {
	p, paths, out := setupBatch(t)
	p.observeNormalLine("BIN RAW: " + paths[0])
	for _, seconds := range []int{0, 130, 299} {
		p.renderProgress(fmt.Sprintf("/ BIN2PGN: Bezig met verwerken... | BIN records: 100 | actief: %ds", seconds))
	}
	screen := strings.Join(visiblePhaseScreen(t, out.String()), "\n")
	if strings.Contains(screen, "resttijd") || strings.Contains(screen, "eerdere") || !strings.Contains(screen, "bestand 1/3") {
		t.Fatal(screen)
	}
	p.renderProgress("/ BIN2PGN: Bezig met verwerken... | BIN records: 100 | actief: 300s")
	screen = strings.Join(visiblePhaseScreen(t, out.String()), "\n")
	if strings.Count(screen, "totale resttijd") != 1 {
		t.Fatal(screen)
	}
	p.batchSamples[2] = etaSample{}
	p.renderProgress("/ BIN2PGN: Bezig met verwerken... | BIN records: 100 | actief: 301s")
	screen = strings.Join(visiblePhaseScreen(t, out.String()), "\n")
	if strings.Contains(screen, "resttijd") || strings.Contains(screen, "eerdere") {
		t.Fatal(screen)
	}
}

func TestHundredTimesLargerHistoryNotReused(t *testing.T) {
	h := etaHistory{Samples: []etaSample{{Fingerprint: "small", Machine: "pc", Bytes: 1600, Records: 100, Seconds: 60}}}
	if got := h.reference(etaSample{Fingerprint: "large", Machine: "pc", Bytes: 160000, Records: 10000}); got != 0 {
		t.Fatal(got)
	}
	p, paths, _ := setupBatch(t)
	p.observeNormalLine("BIN RAW: " + paths[0])
	p.roughETA.update(0, 100, locNL)
	rows := p.batchRows([]string{"scan", "", "live", ""}, 299*time.Second, 100, locNL)
	if rows[4] != "" || rows[5] != "" {
		t.Fatal(rows)
	}
}

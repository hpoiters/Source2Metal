package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseProgressCount(t *testing.T) {
	cases := []struct {
		s    string
		want int64
	}{
		{`\\ BIN2PGN: Bezig met verwerken... | BIN-records: 55.304.468 | actief: 8s`, 55304468},
		{`| BIN2PGN: Work in progress... | BIN records: 1,234,567 | active: 3s`, 1234567},
		{`/ BIN2PGN: Traitement en cours... | enregistrements BIN: 12 345 | actif: 2s`, 12345},
	}
	for _, tc := range cases {
		got, ok := parseProgressCount(tc.s)
		if !ok || got != tc.want {
			t.Fatalf("%q: got %d ok=%v, want %d", tc.s, got, ok, tc.want)
		}
	}
}

func TestActiveDuration(t *testing.T) {
	got, ok := parseActiveDuration(`\\ BIN2PGN: Bezig met verwerken... | BIN-records: 55.304.468 | actief: 1m23s`)
	if !ok || got != 83*time.Second {
		t.Fatalf("got %v ok=%v", got, ok)
	}
}

func TestDetectLocales(t *testing.T) {
	cases := map[string]locale{
		"Work in progress...":    locEN,
		"Verarbeitung läuft...":  locDE,
		"Bezig met verwerken...": locNL,
		"Traitement en cours...": locFR,
		"Procesando...":          locES,
		"正在处理...":                locZH,
		"Идёт обработка...":      locRU,
	}
	for s, want := range cases {
		if got := detectLocale(s); got != want {
			t.Fatalf("%q got %v want %v", s, got, want)
		}
	}
}

func TestObserveBINPathAndRender(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "x.bin")
	if err := os.WriteFile(bin, make([]byte, 16*1000), 0644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	p := newProgressWriter(&out, false)
	p.observeNormalLine("BIN RAW: " + bin)
	if p.totalRecords != 1000 {
		t.Fatalf("total %d", p.totalRecords)
	}
	p.renderProgress(`\\ BIN2PGN: Bezig met verwerken... | BIN-records: 500 | actief: 10s`)
	s := out.String()
	if !strings.Contains(s, "BIN-scan: 500 / 1.000") || !strings.Contains(s, "50,0%") || strings.Contains(s, "resttijd") {
		t.Fatalf("unexpected output: %q", s)
	}
}

func TestStreamFourLineBlock(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "stream.bin")
	if err := os.WriteFile(bin, make([]byte, 16*1000), 0644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	p := newProgressWriter(&out, true)
	_, _ = p.Write([]byte("BIN RAW: " + bin + "\n"))
	_, _ = p.Write([]byte("\r\\ BIN2PGN: Bezig met verwerken... | BIN-records: 500 | actief: 10s"))
	_, _ = p.Write([]byte("\r| BIN2PGN: Bezig met verwerken... | BIN-records: 750 | actief: 12s"))
	_, _ = p.Write([]byte("\rBIN RAW: OK | 1000 | out.pgn\n"))
	p.CloseDisplay()
	s := out.String()
	if !strings.Contains(s, "BIN-scan: 500 / 1.000 | 50,0%") {
		t.Fatalf("missing first detail: %q", s)
	}
	if !strings.Contains(s, "BIN-scan: 750 / 1.000 | 75,0%") {
		t.Fatalf("missing second detail: %q", s)
	}
	if !strings.Contains(s, "\x1b[3A") {
		t.Fatalf("missing in-place cursor repaint: %q", s)
	}
	if !strings.Contains(s, "BIN RAW: OK | 1000 | out.pgn") {
		t.Fatalf("missing final line: %q", s)
	}
}

func TestPromptPassesImmediately(t *testing.T) {
	var out bytes.Buffer
	p := newProgressWriter(&out, true)
	prompt := "Aantal workers [1-36] (Enter = 18):"
	_, _ = p.Write([]byte(prompt))
	if out.String() != prompt {
		t.Fatalf("prompt delayed or changed: %q", out.String())
	}
}

func TestLanguageSwitchClearsAndRepaintsAllLocales(t *testing.T) {
	headings := []string{
		"CHANGE LANGUAGE",
		"SPRACHE ÄNDERN",
		"TAAL WIJZIGEN",
		"CHANGER DE LANGUE",
		"CAMBIAR IDIOMA",
		"更改语言",
		"ИЗМЕНИТЬ ЯЗЫК",
	}
	for _, heading := range headings {
		var out bytes.Buffer
		p := newProgressWriter(&out, true)
		_, _ = p.Write([]byte(heading + "\n1 = test\nKeuze: "))
		_, _ = p.Write([]byte("Source2Metal v3.2.1\nHOOFDMENU\n"))
		s := out.String()
		marker := "\x1b[2J\x1b[HSource2Metal v3.2.1"
		if strings.Count(s, marker) != 1 {
			t.Fatalf("%q: expected one full-screen repaint, got %q", heading, s)
		}
	}
}

func TestOrdinaryMainMenuDoesNotInjectClear(t *testing.T) {
	var out bytes.Buffer
	p := newProgressWriter(&out, true)
	_, _ = p.Write([]byte("Source2Metal v3.2.1\nHOOFDMENU\n"))
	if strings.Contains(out.String(), "\x1b[2J\x1b[H") {
		t.Fatalf("unexpected clear on ordinary menu: %q", out.String())
	}
}

func TestRepeatedLanguageSwitchesAlwaysRepaint(t *testing.T) {
	headings := []string{
		"TAAL WIJZIGEN", "CHANGE LANGUAGE", "SPRACHE ÄNDERN",
		"CHANGER DE LANGUE", "CAMBIAR IDIOMA", "更改语言", "ИЗМЕНИТЬ ЯЗЫК",
	}
	var out bytes.Buffer
	p := newProgressWriter(&out, true)
	for _, heading := range headings {
		_, _ = p.Write([]byte(heading + "\nKeuze: "))
		_, _ = p.Write([]byte("Source2Metal v3.2.1\nMAIN MENU\n"))
	}
	if got := strings.Count(out.String(), "\x1b[2J\x1b[HSource2Metal v3.2.1"); got != len(headings) {
		t.Fatalf("repeated switches: got %d repaints, want %d", got, len(headings))
	}
}

func TestDurationFrameAcrossChunks(t *testing.T) {
	for _, duration := range []string{"1m23s", "1h2m3s", "1.25s", "900ms"} {
		for _, chunkSize := range []int{1, 3, 17, 4096} {
			var out bytes.Buffer
			p := newProgressWriter(&out, true)
			p.totalRecords = 1000
			frame := "\r| BIN2PGN: Bezig met verwerken... | BIN-records: 500 | actief: " + duration + "   "
			for i := 0; i < len(frame); i += chunkSize {
				end := i + chunkSize
				if end > len(frame) {
					end = len(frame)
				}
				p.Write([]byte(frame[i:end]))
			}
			if out.Len() != 0 {
				t.Fatalf("premature render for %s: %q", duration, out.String())
			}
			p.Write([]byte("\r"))
			if !strings.Contains(out.String(), "actief: "+duration) {
				t.Fatalf("truncated duration: %q", out.String())
			}
			if strings.Count(out.String(), "BIN2PGN:") != 1 {
				t.Fatalf("duplicate frame: %q", out.String())
			}
		}
	}
}

func TestUnknownNextBINDoesNotReuseTotal(t *testing.T) {
	var out bytes.Buffer
	p := newProgressWriter(&out, true)
	p.totalRecords = 1000
	p.observeNormalLine("BIN RAW: " + filepath.Join(t.TempDir(), "missing.bin"))
	p.renderProgress("| BIN2PGN: Work in progress... | BIN records: 10 | active: 2s")
	if p.totalRecords != 0 || strings.Contains(out.String(), "Remaining:") {
		t.Fatalf("stale total: %q", out.String())
	}
}

func TestAll49LanguageTransitionsChunked(t *testing.T) {
	headings := []string{"CHANGE LANGUAGE", "SPRACHE ÄNDERN", "TAAL WIJZIGEN", "CHANGER DE LANGUE", "CAMBIAR IDIOMA", "更改语言", "ИЗМЕНИТЬ ЯЗЫК"}
	for _, from := range headings {
		for _, to := range headings {
			var out bytes.Buffer
			p := newProgressWriter(&out, true)
			stream := from + "\nKeuze: Source2Metal v3.2.1\n" + to + "\nChoice: Source2Metal v3.2.1\n"
			for _, b := range []byte(stream) {
				p.Write([]byte{b})
			}
			if strings.Count(out.String(), "\x1b[2J\x1b[HSource2Metal v3.2.1") != 2 {
				t.Fatalf("%s -> %s: %q", from, to, out.String())
			}
		}
	}
}

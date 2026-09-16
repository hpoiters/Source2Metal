package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// Minimal terminal model: handles our actual emitted CR, LF, erase-line,
// cursor-up and automatic wrapping, counting non-ASCII as two cells.
func screenExtent(t *testing.T, s string, width int) (int, int) {
	t.Helper()
	row, col, maxRow := 0, 0, 0
	r := []rune(s)
	for i := 0; i < len(r); i++ {
		switch r[i] {
		case '\r':
			col = 0
		case '\n':
			row++
		case '\x1b':
			if i+3 >= len(r) || r[i+1] != '[' {
				t.Fatal("bad escape")
			}
			n := int(r[i+2] - '0')
			cmd := r[i+3]
			i += 3
			switch cmd {
			case 'A':
				row -= n
			case 'K':
			default:
				t.Fatalf("unknown escape %c", cmd)
			}
			if row < 0 {
				t.Fatal("cursor moved above block")
			}
		default:
			n := 1
			if r[i] > 127 {
				n = 2
			}
			if col+n > width {
				row++
				col = 0
			}
			col += n
		}
		if row > maxRow {
			maxRow = row
		}
	}
	return row, maxRow
}

func TestFixedRowsNoScroll(t *testing.T) {
	phrases := []string{"Work in progress...", "Verarbeitung läuft...", "Bezig met verwerken...", "Traitement en cours...", "Procesando...", "正在处理...", "Идёт обработка..."}
	for _, width := range []int{40, 60, 80, 100, 120, 200} {
		for _, phrase := range phrases {
			for _, chunk := range []int{1, 17, 4096} {
				var out bytes.Buffer
				p := newProgressWriter(&out, true)
				p.width = func() int { return width }
				p.totalRecords = 55304468
				var stream strings.Builder
				for i := 0; i < 300; i++ {
					count := 55304468
					if i < 50 {
						count = i * 1000000
					}
					fmt.Fprintf(&stream, "\r%c BIN2PGN: %s | BIN records: %d | active: %dm%ds   ", `\|/-`[i%4], phrase, count, i/60, i%60)
				}
				stream.WriteString("\r")
				data := stream.String()
				for i := 0; i < len(data); i += chunk {
					end := i + chunk
					if end > len(data) {
						end = len(data)
					}
					p.Write([]byte(data[i:end]))
				}
				row, max := screenExtent(t, out.String(), width)
				if row != 3 || max != 3 {
					t.Fatalf("width=%d phrase=%s chunk=%d row=%d max=%d", width, phrase, chunk, row, max)
				}
				if strings.Count(out.String(), "\x1b[3A") != 299 {
					t.Fatal("lost frames")
				}
			}
		}
	}
}

func TestCompletedScanHonestETA(t *testing.T) {
	var out bytes.Buffer
	p := newProgressWriter(&out, true)
	p.totalRecords = 55304468
	p.renderProgress(`\ BIN2PGN: Bezig met verwerken... | BIN-records: 55.304.468 | actief: 8s`)
	s := out.String()
	for _, bad := range []string{"0 records", "100,0%", "BIN-records:", "~0s"} {
		if strings.Contains(s, bad) {
			t.Fatalf("misleading %s: %q", bad, s)
		}
	}
	for _, want := range []string{"BIN-scan gereed: 55.304.468 records; vervolgverwerking loopt", "actief: 8s"} {
		if !strings.Contains(s, want) {
			t.Fatalf("missing %s: %q", want, s)
		}
	}
}

func TestRedirectedOutputDoesNotFlood(t *testing.T) {
	var out bytes.Buffer
	p := newProgressWriter(&out, false)
	p.totalRecords = 1000
	for i := 0; i < 1000; i++ {
		p.renderProgress(fmt.Sprintf(`| BIN2PGN: Work in progress... | BIN records: %d | active: %ds`, i*2, i))
	}
	if n := strings.Count(out.String(), "\n"); n != 8 {
		t.Fatalf("log grows with updates: %d lines", n)
	}
	if strings.Contains(out.String(), "\x1b") {
		t.Fatal("escape in redirected log")
	}
}

func TestCoreErasureFramesDoNotGrowBlock(t *testing.T) {
	var out bytes.Buffer
	p := newProgressWriter(&out, true)
	p.totalRecords = 1000
	for i := 0; i < 100; i++ {
		p.Write([]byte(fmt.Sprintf("\r| BIN2PGN: Bezig met verwerken... | BIN-records: 1000 | actief: %ds\r%s\r", i, strings.Repeat(" ", 160))))
	}
	row, max := screenExtent(t, out.String(), 80)
	if row != 3 || max != 3 {
		t.Fatalf("erasure frames grew block: row=%d max=%d", row, max)
	}
}

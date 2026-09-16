package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// Replay the emitted console controls to check what remains visible.
func visiblePhaseScreen(t *testing.T, s string) []string {
	t.Helper()
	rows := make([][]rune, 30)
	row, col := 0, 0
	r := []rune(s)
	for i := 0; i < len(r); i++ {
		switch r[i] {
		case '\r':
			col = 0
		case '\n':
			row++
			col = 0
		case '\x1b':
			if i+3 >= len(r) || r[i+1] != '[' {
				t.Fatal("invalid escape")
			}
			n := int(r[i+2] - '0')
			op := r[i+3]
			i += 3
			switch op {
			case 'A':
				row -= n
			case 'K':
				rows[row] = nil
			default:
				t.Fatal("unexpected control")
			}
		default:
			for len(rows[row]) <= col {
				rows[row] = append(rows[row], ' ')
			}
			rows[row][col] = r[i]
			col++
		}
		if row < 0 || row >= len(rows) {
			t.Fatal("cursor outside screen")
		}
	}
	result := make([]string, len(rows))
	for i := range rows {
		result[i] = string(rows[i])
	}
	return result
}

func TestPhaseEndRemovesVisibleETA(t *testing.T) {
	phrases := []string{"Work in progress...", "Verarbeitung läuft...", "Bezig met verwerken...", "Traitement en cours...", "Procesando...", "正在处理...", "Идёт обработка..."}
	for _, phrase := range phrases {
		for _, elapsed := range []string{"8s", "6m0s"} {
			for _, chunk := range []int{1, 7, 4096} {
				var out bytes.Buffer
				p := newProgressWriter(&out, true)
				p.width = func() int { return 200 }
				p.totalRecords = 55304468
				p.Write([]byte("Previous operation\n"))
				stream := fmt.Sprintf("\r/ BIN2PGN: %s | BIN records: 55304468 | active: %s\r\x1b[2K\rBIN RAW: OK | out.pgn\n\n\\ BOOK RAW samenvoegen... | lijnen: 114.391 | actief: 1m30s\n", phrase, elapsed)
				for i := 0; i < len(stream); i += chunk {
					end := i + chunk
					if end > len(stream) {
						end = len(stream)
					}
					p.Write([]byte(stream[i:end]))
				}
				p.CloseDisplay()
				rows := visiblePhaseScreen(t, out.String())
				want := []string{"Previous operation", "", "BIN RAW: OK | out.pgn", "", `\ BOOK RAW samenvoegen... | lijnen: 114.391 | actief: 1m30s`}
				for i, s := range want {
					if rows[i] != s {
						t.Fatalf("%s %s chunk=%d row=%d: %q", phrase, elapsed, chunk, i, rows[i])
					}
				}
				for _, s := range rows[5:] {
					if strings.TrimSpace(s) != "" {
						t.Fatalf("stale row: %q", s)
					}
				}
				if p.ownsProgress || p.blockActive {
					t.Fatal("BIN display still active")
				}
			}
		}
	}
}

func TestPlainLogRetiresETAOnce(t *testing.T) {
	var out bytes.Buffer
	p := newProgressWriter(&out, false)
	p.totalRecords = 55304468
	p.renderProgress(`/ BIN2PGN: Bezig met verwerken... | BIN records: 55304468 | actief: 6m0s`)
	p.Write([]byte("BIN RAW: OK\n\nBOOK RAW samenvoegen...\n"))
	p.CloseDisplay()
	if strings.Count(out.String(), "vorige tijdschatting is niet meer van toepassing.") != 1 {
		t.Fatal(out.String())
	}
	if !strings.Contains(out.String(), "toepassing.\n\nBIN RAW: OK") {
		t.Fatal("missing separator")
	}
}

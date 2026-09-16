package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// Erasure bytes recovered from clearActiveConsoleRows in supplied core.exe.
func TestActualCoreEraseProtocol(t *testing.T) {
	for _, vt := range []bool{true, false} {
		for _, chunk := range []int{1, 2, 7, 4096} {
			for _, clearRows := range []int{1, 4} {
				var out bytes.Buffer
				p := newProgressWriter(&out, vt)
				p.totalRecords = 55304468
				var input strings.Builder
				for i := 0; i < 300; i++ {
					if i > 0 {
						input.WriteString("\r\x1b[2K")
						for j := 1; j < clearRows; j++ {
							input.WriteString("\x1b[1A\r\x1b[2K")
						}
					}
					fmt.Fprintf(&input, "\r%c BIN2PGN: Bezig met verwerken... | BIN-records: 55.304.468 | actief: %ds", `\|/-`[i%4], i)
				}
				input.WriteString("\r\x1b[2K\r")
				data := input.String()
				for i := 0; i < len(data); i += chunk {
					end := i + chunk
					if end > len(data) {
						end = len(data)
					}
					p.Write([]byte(data[i:end]))
				}
				if vt {
					row, max := screenExtent(t, out.String(), 80)
					if row != 3 || max != 3 {
						t.Fatalf("chunk=%d clearRows=%d: row=%d max=%d; want four fixed rows", chunk, clearRows, row, max)
					}
				} else if strings.Contains(out.String(), "\x1b") || strings.Count(out.String(), "\n") != 4 {
					t.Fatalf("redirected output contains erasures or repeated blocks: %q", out.String())
				}
				p.Write([]byte("BIN RAW: OK | 123 | out.pgn\nKeuze: "))
				p.CloseDisplay()
				if !strings.HasSuffix(out.String(), "BIN RAW: OK | 123 | out.pgn\nKeuze: ") {
					t.Fatalf("lost completion/prompt: %q", out.String())
				}
			}
		}
	}
}

package ctgraw

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T, lines []string) string {
	t.Helper()
	entries := map[string]map[byte]bool{}
	boards := map[string]Board{}
	for _, line := range lines {
		b := startBoard()
		for _, u := range strings.Fields(line) {
			key := string(b.Signature())
			boards[key] = b
			if entries[key] == nil {
				entries[key] = map[byte]bool{}
			}
			code, e := findCodeForCoord(b, u)
			if e != nil {
				t.Fatal(e)
			}
			entries[key][code] = true
			m, e := (&CTGReader{}).DecodeMove(b, code)
			if e != nil {
				t.Fatal(e)
			}
			b.Apply(m)
		}
		key := string(b.Signature())
		boards[key] = b
		if entries[key] == nil {
			entries[key] = map[byte]bool{}
		}
	}
	pg := make([]byte, PageSize)
	binary.BigEndian.PutUint16(pg[:2], uint16(len(entries)))
	pos := 4
	for key, moves := range entries {
		raw := []RawMove{}
		for code := range moves {
			raw = append(raw, RawMove{code, 0})
		}
		// root count of 1 deliberately makes the legacy random sampler incomplete.
		en := encodeEntry(boards[key].Signature(), raw, 1, 0, 0, 1)
		if pos+len(en) > len(pg) {
			t.Fatal("fixture page overflow")
		}
		copy(pg[pos:], en)
		pos += len(en)
	}
	base := filepath.Join(t.TempDir(), "fixture")
	ctg := make([]byte, 2*PageSize)
	copy(ctg[PageSize:], pg)
	for ext, data := range map[string][]byte{".ctg": ctg, ".ctb": make([]byte, 12), ".cto": make([]byte, 20)} {
		if e := os.WriteFile(base+ext, data, 0644); e != nil {
			t.Fatal(e)
		}
	}
	return base
}
func loadReport(t *testing.T, p string) GraphReport {
	t.Helper()
	data, e := os.ReadFile(p + ".ctg-report.json")
	if e != nil {
		t.Fatal(e)
	}
	var r GraphReport
	if e = json.Unmarshal(data, &r); e != nil {
		t.Fatal(e)
	}
	return r
}
func TestGraphCoverageTranspositionCycleAndRare(t *testing.T) {
	base := fixture(t, []string{
		"g1f3 g8f6 b1c3 b8c6 e2e4", "b1c3 b8c6 g1f3 g8f6 d2d4",
		"g1f3 g8f6 f3g1 f6g8", "e2e4 a7a6", "e2e4 h7h6", "e2e4 b7b6",
	})
	var first []byte
	for _, workers := range []int{1, 4} {
		out := base + string(rune('a'+workers)) + ".pgn"
		res, e := BuildGraph(base+".ctg", base+".cto", base+".ctb", out, 0, workers, nil)
		if e != nil {
			t.Fatal(e)
		}
		r := loadReport(t, out)
		if !r.CoverageOfDecodedEdges || r.Edges == 0 || r.Transpositions < 2 || r.DecodeErrors != 0 {
			t.Fatalf("%+v", r)
		}
		data, _ := os.ReadFile(out)
		if first == nil {
			first = data
		} else if string(first) != string(data) {
			t.Fatal("worker-dependent output")
		}
		if !strings.Contains(string(data), "a6") || !strings.Contains(string(data), "h6") || !strings.Contains(string(data), "b6") {
			t.Fatal("rare edge missing")
		}
		if strings.Contains(string(data), "1-0") || strings.Contains(string(data), "0-1") || strings.Contains(string(data), "1/2") {
			t.Fatal("fabricated outcome")
		}
		t.Logf("workers=%d positions=%d edges=%d transpositions=%d records=%d", workers, r.Positions, r.Edges, r.Transpositions, res.Games)
	}
	r, e := openCTG(base+".ctg", base+".cto", base+".ctb")
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	old := base + "old.pgn"
	_, _, e = writeNeutralPGN(r, old, "fixture", 100)
	if e != nil {
		t.Fatal(e)
	}
	data, _ := os.ReadFile(old)
	found := 0
	for _, s := range []string{"a6", "h6", "b6"} {
		if strings.Contains(string(data), s) {
			found++
		}
	}
	if found >= 3 {
		t.Fatal("expected reproducible legacy coverage gap")
	}
	t.Logf("legacy rare replies covered=%d/3", found)
}
func TestGraphDepthAndMalformedPage(t *testing.T) {
	base := fixture(t, []string{"e2e4 e7e5 g1f3", "d2d4 d7d5"})
	out := base + ".pgn"
	_, e := BuildGraph(base+".ctg", base+".cto", base+".ctb", out, 1, 2, nil)
	if e != nil {
		t.Fatal(e)
	}
	r := loadReport(t, out)
	if r.Edges != 2 || r.DepthCutMoves != 2 || r.DepthCutPositions != 2 {
		t.Fatalf("%+v", r)
	}
	data, _ := os.ReadFile(base + ".ctg")
	data[PageSize+4] = 0
	os.WriteFile(base+".ctg", data, 0644)
	if _, e = BuildGraph(base+".ctg", base+".cto", base+".ctb", out, 100, 2, nil); e == nil {
		t.Fatal("corrupt page accepted")
	}
}

// Opt-in real-book reference run, excluded from ordinary unit tests.
func TestExternalLegacy(t *testing.T) {
	base := os.Getenv("CTG_REFERENCE_BOOK")
	if base == "" {
		t.Skip("set CTG_REFERENCE_BOOK and CTG_REFERENCE_OUT")
	}
	out := os.Getenv("CTG_REFERENCE_OUT")
	if out == "" {
		t.Fatal("output required")
	}
	r, e := openCTG(base+".ctg", base+".cto", base+".ctb")
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	games, plies, e := writeNeutralPGN(r, out, "reference", 100)
	if e != nil {
		t.Fatal(e)
	}
	t.Logf("legacy games=%d plies=%d decodeErrors=%d", games, plies, r.decodeErrors)
}
func TestExternalProbe(t *testing.T) {
	base := os.Getenv("CTG_PROBE_BOOK")
	if base == "" {
		t.Skip("probe")
	}
	r, e := openCTG(base+".ctg", base+".cto", base+".ctb")
	if e != nil {
		t.Fatal(e)
	}
	defer r.Close()
	b := startBoard()
	for _, u := range strings.Fields(os.Getenv("CTG_PROBE_LINE")) {
		code, e := findCodeForCoord(b, u)
		if e != nil {
			t.Fatal(e)
		}
		m, _ := r.DecodeMove(b, code)
		b.Apply(m)
	}
	t.Logf("low=%d high=%d sig=%x", r.low, r.high, b.Signature())
	for _, k := range r.pageIndices(b.Signature()) {
		var v [4]byte
		n, e := r.cto.ReadAt(v[:], 16+int64(k)*4)
		t.Logf("key=%d n=%d val=%d err=%v", k, n, int32(binary.BigEndian.Uint32(v[:])), e)
	}
	en, e := r.Lookup(b)
	t.Logf("entry=%+v err=%v", en, e)
}
func TestGraphCastlingAndEnPassant(t *testing.T) {
	base := fixture(t, []string{"e2e4 e7e5 g1f3 b8c6 f1c4 g8f6 e1g1", "e2e4 a7a6 e4e5 d7d5 e5d6"})
	out := base + ".pgn"
	_, e := BuildGraph(base+".ctg", base+".cto", base+".ctb", out, 100, 3, nil)
	if e != nil {
		t.Fatal(e)
	}
	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), "O-O") || !strings.Contains(string(data), "exd6") {
		t.Fatalf("missing special moves: %s", data)
	}
	r := loadReport(t, out)
	if r.DecodeErrors != 0 || !r.CoverageOfDecodedEdges {
		t.Fatalf("%+v", r)
	}
}

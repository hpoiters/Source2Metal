package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"source2metal/internal/bin2pgn"
)

// A literal Polyglot opening record, independent of the converter's encoder.
func writeE4BIN(t *testing.T, path string) {
	t.Helper()
	var record [16]byte
	binary.BigEndian.PutUint64(record[:8], 0x463b96181691fc9c)
	binary.BigEndian.PutUint16(record[8:10], 0x031c)
	binary.BigEndian.PutUint16(record[10:12], 100)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, record[:], 0644); err != nil {
		t.Fatal(err)
	}
}

func TestBINIntegrationShortLinesAndDuplicates(t *testing.T) {
	root := t.TempDir()
	a, b := filepath.Join(root, "a", "same.bin"), filepath.Join(root, "b", "same.BIN")
	writeE4BIN(t, a)
	writeE4BIN(t, b)
	before, err := fileSHA256(a)
	if err != nil {
		t.Fatal(err)
	}
	l, err := createLayout(root, "test")
	if err != nil {
		t.Fatal(err)
	}
	writeE4BIN(t, filepath.Join(l.RunDir, "ignored.bin"))
	inv, err := scanSources(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(inv.Sources) != 2 {
		t.Fatalf("inventory: %+v", inv)
	}
	var c Counters
	paths, err := buildBINRaw(inv, l, Config{MaxPly: 100, MinPly: 24, MinElo: 2500}, &c)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 || paths[0] == paths[1] || c.BINRawVerified != 2 {
		t.Fatalf("paths=%v counters=%+v", paths, c)
	}
	merged, err := mergeBookRAW(paths, l, &c)
	if err != nil {
		t.Fatal(err)
	}
	if c.BookMergedGames != 1 || c.BookMergedDuplicates != 1 {
		t.Fatalf("counters=%+v", c)
	}
	if n, e := countGeneratedPGNRecords(merged); e != nil || n != 1 {
		t.Fatalf("count %d: %v", n, e)
	}
	after, err := fileSHA256(a)
	if err != nil || before != after {
		t.Fatal("source changed", err)
	}
}

func TestBOOKMergeMultilineCTGAndBIN(t *testing.T) {
	root := t.TempDir()
	l, err := createLayout(root, "test")
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "book.bin")
	writeE4BIN(t, bin)
	binOut := filepath.Join(l.TempDir, "bin.pgn")
	if _, err = bin2pgn.Convert(bin, binOut, 100); err != nil {
		t.Fatal(err)
	}
	ctg := filepath.Join(l.TempDir, "ctg.pgn")
	if err = os.WriteFile(ctg, []byte("[Event \"CTG\"]\n[Result \"*\"]\n\n1. e4\ne5 *\n\n[Event \"CTG2\"]\n[Result \"*\"]\n\n1. e4 *\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var c Counters
	merged, err := mergeBookRAW([]string{ctg, binOut}, l, &c)
	if err != nil {
		t.Fatal(err)
	}
	if c.BookMergedGames != 2 || c.BookMergedDuplicates != 1 {
		t.Fatalf("counters=%+v", c)
	}
	game := filepath.Join(l.TempDir, "game.pgn")
	if err = os.WriteFile(game, []byte("[Event \"GAME\"]\n[Result \"1-0\"]\n\n1. d4 d5 1-0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	c.GamesAccepted = 1
	if _, err = buildCombined(game, []string{merged}, l, &c); err != nil {
		t.Fatal(err)
	}
	if c.CombinedVerified != 3 {
		t.Fatalf("combined: %d", c.CombinedVerified)
	}
}

func TestBINFailureCannotReportSuccess(t *testing.T) {
	root := t.TempDir()
	bad := filepath.Join(root, "broken.bin")
	if err := os.WriteFile(bad, []byte("broken"), 0644); err != nil {
		t.Fatal(err)
	}
	l, err := createLayout(root, "test")
	if err != nil {
		t.Fatal(err)
	}
	var c Counters
	out, err := buildBINRaw(Inventory{Sources: []Source{{Kind: KindBIN, Base: "broken", Path: bad}}}, l, Config{MaxPly: 100}, &c)
	if err == nil || len(out) != 0 || classifyRunStatus(Config{Mode: "raw"}, c) != "MISLUKT" {
		t.Fatalf("out=%v err=%v counters=%+v", out, err, c)
	}
}

// Optional integration check with user-supplied CTG RAW files; no books are
// redistributed in the repository or the release.
func TestUserCTGRAWMerge(t *testing.T) {
	dir := os.Getenv("SOURCE2METAL_CTG_RAW_TEST_DIR")
	if dir == "" {
		t.Skip("no external CTG RAW fixtures supplied")
	}
	var paths []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var inputCount int64
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), "[CTG] - RAW.pgn") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		paths = append(paths, p)
		n, e := countGeneratedPGNRecords(p)
		if e != nil {
			t.Fatal(e)
		}
		inputCount += n
	}
	if len(paths) == 0 {
		t.Fatal("no CTG RAW fixtures found")
	}
	l, err := createLayout(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	var c Counters
	if _, err = mergeBookRAW(paths, l, &c); err != nil {
		t.Fatal(err)
	}
	if c.BookMergedGames+c.BookMergedDuplicates != inputCount {
		t.Fatal("record accounting mismatch")
	}
	t.Logf("CTG input %d; merged %d; duplicates %d", inputCount, c.BookMergedGames, c.BookMergedDuplicates)
}

func TestBINRunSevenLanguages(t *testing.T) {
	old := activeLanguage
	defer func() { activeLanguage = old }()
	for _, lang := range []string{"en", "de", "nl", "fr", "es", "zh", "ru"} {
		t.Run(lang, func(t *testing.T) {
			activeLanguage = lang
			root := t.TempDir()
			writeE4BIN(t, filepath.Join(root, "short.bin"))
			if err := run(Config{Input: root, Mode: "all", MaxPly: 100, MinPly: 24, MinElo: 2500, Workers: 1, NoPause: true}); err != nil {
				t.Fatal(err)
			}
			runs, err := os.ReadDir(filepath.Join(root, "!Source2Metal_Output"))
			if err != nil || len(runs) != 1 {
				t.Fatal(runs, err)
			}
			output := filepath.Join(root, "!Source2Metal_Output", runs[0].Name())
			pgn := filepath.Join(output, "RAW", "1 - Separate Sources - RAW PGNs", "BOOK Sources", "short [BIN] - RAW.pgn")
			if n, e := countGeneratedPGNRecords(pgn); e != nil || n != 1 {
				t.Fatal(n, e)
			}
			for _, name := range []string{"STATUS.txt", "Source2Metal - Full Process Report.txt", "Source2Metal_BuildInfo.txt"} {
				b, e := os.ReadFile(filepath.Join(output, "REPORTS", name))
				if e != nil {
					t.Fatal(e)
				}
				if !strings.Contains(string(b), binScopeText()) || strings.Contains(string(b), "%!") {
					t.Fatalf("%s: inconsistent report %s", lang, name)
				}
			}
		})
	}
}

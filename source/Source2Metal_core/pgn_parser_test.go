package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"source2metal/internal/ctgmetal"
)

func readTestGames(t *testing.T, body string) ([]PGNGame, error) {
	t.Helper()
	path := overlayFixture(t, t.TempDir(), "test.pgn", body)
	var games []PGNGame
	err := parsePGNFile(path, func(g PGNGame) error { games = append(games, g); return nil }, nil)
	return games, err
}

func TestPGNRecordLexicalBoundaries(t *testing.T) {
	for _, tc := range []struct{ name, moves string }{
		{"polyglot continuation", "1. e4 {weight\n[%polyglot_weight 65520]\n[%polyglot_key ABCD]\n} e5 *"},
		{"tag in comment", "1. e4 {text\n[Event \"tekst\"]\n} e5 *"},
		{"semicolon", "1. e4 ; [Event \"fake\"] { ( }\ne5 *"},
		{"nested variation", "1. e4 (1. d4 (1. c4)\n[Event \"variation text\"]\n1... d5) e5 *"},
		{"comment in variation", "1. e4 (1. d4 { )\n[Event \"fake\"]\n} d5) e5 *"},
		{"semicolon in variation", "1. e4 (1. d4 ; ) {\nd5) e5 *"},
		{"blank comment lines", "1. e4 {text\n\n[%polyglot_key ABCD]\n} e5 *"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			games, err := readTestGames(t, overlayPGN(tc.moves, "")+overlayPGN("1. d4 d5 *", ""))
			if err != nil || len(games) != 2 {
				t.Fatalf("records=%d err=%v", len(games), err)
			}
			toks, _, _, _ := stripMainline(games[0].MoveText)
			if strings.Join(toks, " ") != "e4 e5" {
				t.Fatal(toks)
			}
			if games[0].Tags["Event"] != "test" {
				t.Fatal(games[0].Tags)
			}
		})
	}
}

func TestPGNValidTagsOnly(t *testing.T) {
	for _, line := range []string{`[%polyglot_weight 42]`, `[Event "broken"quote"]`, `[Event "valid"] garbage`} {
		games, err := readTestGames(t, overlayPGN("1. e4\n"+line+"\ne5 *", ""))
		if err != nil || len(games) != 1 || !strings.Contains(games[0].MoveText, line) {
			t.Fatalf("%q: %v %v", line, games, err)
		}
	}
	games, err := readTestGames(t, "[Event \"one\"]\n[Site \"place\"]\n[Result \"*\"]\n1. e4 e5 *\n[Event \"two\"]\n[Result \"*\"]\n1. d4 d5 *")
	if err != nil || len(games) != 2 || games[0].Tags["Site"] != "place" || games[1].Tags["Event"] != "two" {
		t.Fatalf("%v %v", games, err)
	}
	games, err = readTestGames(t, "[Event \"a \\\"quote\\\" and \\\\ path\"]\n1. e4 *")
	if err != nil || len(games) != 1 || games[0].Tags["Event"] != `a "quote" and \ path` {
		t.Fatalf("escaped tag: %v %v", games, err)
	}
}

func TestPGNDamagedRecordNeverEmitsPrefix(t *testing.T) {
	for _, moves := range []string{"1. e4 {unfinished", "1. e4 (1. d4", "1. e4 (1. d4 {unfinished", "1. e4 ) *", "1. e4 } *", "1. e4 * {unfinished", "1. e4 {\n[Event \"fake recovery\"]\n1. d4 *"} {
		games, err := readTestGames(t, overlayPGN("1. c4 e5 *", "")+overlayPGN(moves, ""))
		if err == nil || len(games) != 1 {
			t.Fatalf("damaged %q: emitted %d, err %v", moves, len(games), err)
		}
	}
}

func TestPGNOrdinaryGameUnchanged(t *testing.T) {
	plain := "[Event \"normal\"]\n[Result \"1-0\"]\n[WhiteElo \"3100\"]\n[BlackElo \"3100\"]\n\n1. e4 e5 2. Nf3 Nc6 1-0\n"
	annotated := strings.Replace(plain, "e4 e5", "e4 {note\n[Event \"fake\"]\n} e5 (1... c5)", 1)
	a, ea := readTestGames(t, plain)
	b, eb := readTestGames(t, annotated)
	if ea != nil || eb != nil || len(a) != 1 || len(b) != 1 {
		t.Fatal(ea, eb)
	}
	cfg := Config{MinPly: 4, MinElo: 3000}
	ra, rb := processRawJob(rawJob{game: a[0]}, cfg), processRawJob(rawJob{game: b[0]}, cfg)
	if ra.reject != rejectNone || rb.reject != rejectNone || ra.sig != rb.sig || !reflect.DeepEqual(ra.toks, rb.toks) {
		t.Fatalf("GAME changed: %+v %+v", ra, rb)
	}
}

func collectPGNCoverage(t *testing.T, path string) (map[gmEdgeKey]bool, []string) {
	t.Helper()
	edges := map[gmEdgeKey]bool{}
	var lines []string
	err := parsePGNFile(path, func(g PGNGame) error {
		toks, _, _, _ := stripMainline(g.MoveText)
		san, es, ok := ctgmetal.ParsePGNMainline(toks, len(toks))
		if !ok || len(toks) == 0 {
			return fmt.Errorf("illegal/empty mainline")
		}
		lines = append(lines, strings.Join(san, " "))
		for _, e := range es {
			edges[overlayEdge(e)] = true
		}
		return nil
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return edges, lines
}

func runCoveragePipeline(t *testing.T, paths []string) (Counters, []string, string) {
	t.Helper()
	root := t.TempDir()
	inv := Inventory{Root: root}
	for _, p := range paths {
		inv.Sources = append(inv.Sources, Source{Kind: KindOverlay, Path: p, Base: strings.TrimSuffix(filepath.Base(p), ".pgn")})
	}
	l, err := createLayout(root, "regression")
	if err != nil {
		t.Fatal(err)
	}
	c := Counters{}
	raws, err := buildOverlayRAW(inv, l, Config{MaxPly: 100, MinPly: 24, MinElo: 3000}, &c)
	if err != nil {
		t.Fatal(err)
	}
	if err := appendOverlayMetal(l, &c); err != nil {
		t.Fatal(err)
	}
	return c, raws, filepath.Join(l.MetalDir, "Metal.pgn")
}

func TestCommentRewrappingCoverage(t *testing.T) {
	plain := overlayPGN("1. e4 {weight [%polyglot_weight 42]} e5 2. Nf3 Nc6 *", "") + overlayPGN("1. d4 d5 2. c4 *", "")
	wrapped := strings.ReplaceAll(plain, "{weight [%polyglot_weight 42]}", "{weight\n[%polyglot_weight 42]\n}")
	var expected map[gmEdgeKey]bool
	for _, body := range []string{plain, wrapped} {
		p := overlayFixture(t, t.TempDir(), "synthetic.pgn", body)
		edges, lines := collectPGNCoverage(t, p)
		if len(lines) != 2 {
			t.Fatal("record count", len(lines))
		}
		if expected == nil {
			expected = edges
		} else if !reflect.DeepEqual(expected, edges) {
			t.Fatal("rewrapping changed coverage")
		}
		c, raws, metal := runCoveragePipeline(t, []string{p})
		tr := c.OverlayTraces[0]
		if tr.Seen != 2 || tr.Accepted != 2 || tr.Rejected != 0 || tr.Written != 2 {
			t.Fatalf("counters: %+v", tr)
		}
		rawEdges, rawLines := collectPGNCoverage(t, raws[0])
		metalEdges, _ := collectPGNCoverage(t, metal)
		if !reflect.DeepEqual(lines, rawLines) || !reflect.DeepEqual(expected, rawEdges) || !reflect.DeepEqual(expected, metalEdges) {
			t.Fatal("coverage changed")
		}
		b, err := os.ReadFile(raws[0])
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "polyglot_") {
			t.Fatal("unexpected weight preservation")
		}
	}
}

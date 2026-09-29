package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func overlayFixture(t *testing.T, root, name, body string) string {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
func overlayPGN(moves, label string) string {
	return fmt.Sprintf("[Event \"test\"]\n[Result \"*\"]\n[ExampleLabel \"%s\"]\n\n%s\n\n", label, moves)
}
func TestOverlayExplicitOptIn(t *testing.T) {
	root := t.TempDir()
	for _, n := range []string{"ordinary.pgn", "another.pgn", "OVERLAY-PGN/a.pgn", "nested/overlay-pgn/b.pgn", "c.overlay.pgn", "!Source2Metal_Output/OVERLAY-PGN/ignored.pgn"} {
		overlayFixture(t, root, n, overlayPGN("1. e4 e5 *", "sample-a"))
	}
	inv, err := scanSources(root)
	if err != nil {
		t.Fatal(err)
	}
	game, overlay := 0, 0
	for _, s := range inv.Sources {
		if s.Kind == KindPGN {
			game++
		}
		if s.Kind == KindOverlay {
			overlay++
		}
	}
	if game != 2 || overlay != 3 || len(gameMetalSources(inv, nil)) != 2 {
		t.Fatalf("routing: GAME=%d overlay=%d", game, overlay)
	}
	if !isOverlayPGN(filepath.Join(root, "OVERLAY-PGN"), filepath.Join(root, "OVERLAY-PGN", "x.pgn")) {
		t.Fatal("root folder must opt in")
	}
}
func TestOverlayValidationAndDedup(t *testing.T) {
	root := t.TempDir()
	body := overlayPGN("1. e4 e5 2. Nf3 Nc6 *", "sample-a") +
		overlayPGN("1. e4 e5 2. Nf3 Nc6 *", "sample-a") +
		overlayPGN("1. e4 e5 2. Nf3 Nc6 *", "sample-b") +
		overlayPGN("1. e4 e5 2. Qa9 *", "") +
		overlayPGN("*", "") +
		"[Event \"setup\"]\n[SetUp \"1\"]\n[Result \"*\"]\n\n1. e4 *\n\n" +
		overlayPGN("1. e4 *", "bogus")
	overlayFixture(t, root, "OVERLAY-PGN/a.pgn", body)
	inv, err := scanSources(root)
	if err != nil {
		t.Fatal(err)
	}
	l, err := createLayout(root, "test")
	if err != nil {
		t.Fatal(err)
	}
	c := Counters{}
	// MinPly/Elo must not apply. Illegal moves beyond the cap must still fail.
	paths, err := buildOverlayRAW(inv, l, Config{MaxPly: 2, MinPly: 24, MinElo: 3000}, &c)
	if err != nil {
		t.Fatal(err)
	}
	tr := c.OverlayTraces[0]
	if tr.Seen != 7 || tr.Accepted != 4 || tr.Duplicates != 2 || tr.Written != 2 || tr.Rejected != 3 || len(paths) != 1 {
		t.Fatalf("trace: %+v", tr)
	}
	b, _ := os.ReadFile(paths[0])
	if strings.Contains(string(b), "Nf3") || bytes.Count(b, []byte("[Result \"*\"]")) != 2 {
		t.Fatal(string(b))
	}
	if err := appendOverlayMetal(l, &c); err != nil {
		t.Fatal(err)
	}
	tr = c.OverlayTraces[0]
	if tr.Contributed != 2 || tr.Covered != 0 || tr.NewEdges != 2 {
		t.Fatalf("final: %+v", tr)
	}
	if c.GamesAccepted != 0 || c.GameMetalObservations != 0 {
		t.Fatal("overlay polluted GAME counters")
	}
}
func TestOverlayCoveragePreservesBaseAndTranspositions(t *testing.T) {
	root := t.TempDir()
	l, err := createLayout(root, "test")
	if err != nil {
		t.Fatal(err)
	}
	base := overlayPGN("1. Nf3 Nf6 2. g3 g6 3. Bg2 Bg7 *", "")
	overlayFixture(t, l.MetalDir, "Metal.pgn", base)
	body := overlayPGN("1. Nf3 Nf6 2. g3 g6 *", "") + // distinct prefix must be retained
		overlayPGN("1. g3 g6 2. Nf3 Nf6 3. Bg2 Bg7 *", "") + // new approach, same final edges
		overlayPGN("1. Nf3 Nf6 2. g3 g6 3. Bg2 Bg7 4. O-O *", "")
	overlayFixture(t, root, "OVERLAY-PGN/a.pgn", body)
	inv, err := scanSources(root)
	if err != nil {
		t.Fatal(err)
	}
	// The generated output tree is excluded.
	c := Counters{}
	if _, err := buildOverlayRAW(inv, l, Config{MaxPly: 100}, &c); err != nil {
		t.Fatal(err)
	}
	if err := appendOverlayMetal(l, &c); err != nil {
		t.Fatal(err)
	}
	tr := c.OverlayTraces[0]
	if tr.Contributed != 3 || tr.Covered != 0 || tr.NewEdges != 5 {
		t.Fatalf("transposition coverage: %+v", tr)
	}
	got, _ := os.ReadFile(filepath.Join(l.MetalDir, "Metal.pgn"))
	if !bytes.HasPrefix(got, []byte(base)) {
		t.Fatal("base changed")
	}
	before := append([]byte(nil), got...)
	if err := appendOverlayMetal(l, &Counters{}); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(filepath.Join(l.MetalDir, "Metal.pgn"))
	if !bytes.Equal(before, got) {
		t.Fatal("no-overlay route changed output")
	}
}
func TestOverlayRejectAllDoesNotCreateMetal(t *testing.T) {
	root := t.TempDir()
	l, err := createLayout(root, "test")
	if err != nil {
		t.Fatal(err)
	}
	overlayFixture(t, root, "bad.overlay.pgn", overlayPGN("1. e5 *", ""))
	inv, _ := scanSources(root)
	c := Counters{}
	if _, err := buildOverlayRAW(inv, l, Config{MaxPly: 10}, &c); err != nil {
		t.Fatal(err)
	}
	if err := appendOverlayMetal(l, &c); err != nil {
		t.Fatal(err)
	}
	if fileExists(filepath.Join(l.MetalDir, "Metal.pgn")) {
		t.Fatal("empty Metal created")
	}
	if !strings.Contains(overlayReport(c), "GEEN bijdrage") {
		t.Fatal("zero contribution missing")
	}
}
func TestGameContributionTruth(t *testing.T) {
	if s := gameContributionText(SourceTrace{}); !strings.Contains(s, "NIET bijgedragen") || strings.Contains(s, "kan wel") {
		t.Fatal(s)
	}
	if s := gameContributionText(SourceTrace{MetalEvidence: 3}); !strings.Contains(s, "WEL bij") {
		t.Fatal(s)
	}
}

func TestOverlayExactFullLinesAndResults(t *testing.T) {
	root := t.TempDir()
	l, err := createLayout(root, "exact-lines")
	if err != nil {
		t.Fatal(err)
	}
	base := strings.ReplaceAll(overlayPGN("1. e4 e5 2. Nf3 Nc6 *", ""), "*", "1-0")
	overlayFixture(t, l.MetalDir, "Metal.pgn", base)
	body := overlayPGN("1. e4 e5 2. Nf3 Nc6 *", "") +
		overlayPGN("1. e4 e5 *", "") +
		overlayPGN("1. d4 d5 *", "") +
		overlayPGN("1. d4 d5 *", "sample-a")
	overlayFixture(t, root, "OVERLAY-PGN/a.pgn", body)
	inv, err := scanSources(root)
	if err != nil {
		t.Fatal(err)
	}
	c := Counters{GamesAccepted: 17, GameMetalObservations: 13, GameMetalSelected: 1}
	if _, err := buildOverlayRAW(inv, l, Config{MaxPly: 100}, &c); err != nil {
		t.Fatal(err)
	}
	if err := appendOverlayMetal(l, &c); err != nil {
		t.Fatal(err)
	}
	tr := c.OverlayTraces[0]
	if tr.Contributed != 2 || tr.Covered != 1 || c.OverlayFinalRecords != 3 {
		t.Fatalf("unexpected merge: %+v", c)
	}
	got, err := os.ReadFile(filepath.Join(l.MetalDir, "Metal.pgn"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(got, []byte(base)) || bytes.Count(got, []byte("[Result \"*\"]")) != 2 {
		t.Fatal("base or book results changed")
	}
	if c.GamesAccepted != 17 || c.GameMetalObservations != 13 || c.GameMetalSelected != 1 {
		t.Fatal("GAME counters changed")
	}
	before := append([]byte(nil), got...)
	if err := appendOverlayMetal(l, &c); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(filepath.Join(l.MetalDir, "Metal.pgn"))
	if !bytes.Equal(before, got) {
		t.Fatal("repeated merge duplicated lines")
	}
}

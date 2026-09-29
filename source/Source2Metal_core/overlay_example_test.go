package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDistributedOverlayExample(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("OVERLAY-PGN", "VoorbeeldOverlay.pgn"))
	if err != nil {
		t.Fatal(err)
	}
	p := overlayFixture(t, t.TempDir(), "VoorbeeldOverlay.pgn", string(data))
	c, _, final := runCoveragePipeline(t, []string{p})
	tr := c.OverlayTraces[0]
	if tr.Seen != 6 || tr.Accepted != 6 || tr.Duplicates != 1 || tr.Written != 5 || tr.Rejected != 0 || tr.Contributed != 5 || c.OverlayFinalRecords != 5 {
		t.Fatalf("example: %+v", c)
	}
	_, lines := collectPGNCoverage(t, final)
	for _, want := range []string{"Nf3 Nf6 g3 g6", "Nf3 Nf6 g3 g6 Bg2 Bg7", "g3 g6 Nf3 Nf6 Bg2 Bg7"} {
		found := false
		for _, line := range lines {
			if line == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing line %q", want)
		}
	}
	b, err := os.ReadFile(final)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(b), "[Result \"*\"]") != 5 {
		t.Fatal("non-neutral result")
	}
	if strings.Contains(string(b), "ExampleLabel") {
		t.Fatal("input metadata leaked")
	}
}

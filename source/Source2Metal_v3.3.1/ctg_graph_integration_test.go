package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFinalMetalNeutralHeaders(t *testing.T) {
	root := t.TempDir()
	l, e := createLayout(root, "run")
	if e != nil {
		t.Fatal(e)
	}
	os.MkdirAll(l.MetalSeparateDir, 0755)
	in := filepath.Join(l.MetalSeparateDir, "source.pgn")
	data := "[Event \"Original\"]\n[White \"PersonalName\"]\n[Black \"PrivateName\"]\n[Result \"*\"]\n[SourcePath \"private-path\"]\n\n1. e4 e5 *\n\n"
	os.WriteFile(in, []byte(data), 0644)
	c := Counters{SourceTraces: []SourceTrace{{Kind: KindCTG, MetalPath: in, MetalSelected: 1}}}
	if e = mergeFinalMetal(l, &c); e != nil {
		t.Fatal(e)
	}
	out, e := os.ReadFile(filepath.Join(l.MetalDir, "Metal.pgn"))
	if e != nil {
		t.Fatal(e)
	}
	for _, bad := range []string{"PersonalName", "PrivateName", "SourcePath", "private-path"} {
		if strings.Contains(string(out), bad) {
			t.Fatal("personal metadata retained")
		}
	}
	if !strings.Contains(string(out), "1. e4 e5 *") {
		t.Fatalf("moves lost: %s", out)
	}
}
func TestWorkersAtLeastHalf(t *testing.T) {
	for _, n := range []int{1, 3, 8, 12, 36} {
		w := defaultWorkers(n)
		if w != (n+1)/2 {
			t.Fatalf("%d -> %d", n, w)
		}
	}
}

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratedPGNVerificationDoesNotRequireEvent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.pgn")
	data := `[White "A"]
[Black "B"]
[Result "1-0"]
[Source2MetalVersion "3.2.1"]
[Source2MetalSource "one.pgn"]

1. e4 e5 1-0

[Event "Has event"]
[White "C"]
[Black "D"]
[Result "0-1"]
[Source2MetalVersion "3.2.1"]
[Source2MetalSource "two.pgn"]

1. d4 d5 0-1
`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := countGeneratedPGNRecords(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != 2 {
		t.Fatalf("expected 2 records, got %d", got)
	}
}

func TestPGNErrorThreshold(t *testing.T) {
	if highPGNErrorRate(99, 99) {
		t.Fatal("must not prompt before minimum sample")
	}
	if !highPGNErrorRate(100, 10) {
		t.Fatal("10 percent at 100 games must trigger")
	}
	if highPGNErrorRate(100, 9) {
		t.Fatal("9 percent must not trigger")
	}
}

func TestGameSourceErrorOnlyForUnusableResult(t *testing.T) {
	goodTag := PGNGame{Tags: map[string]string{"Result": "1-0"}, MoveText: "1. e4 e5"}
	if gameHasSourceError(goodTag) {
		t.Fatal("valid Result tag must not be an error")
	}
	goodMovetext := PGNGame{Tags: map[string]string{}, MoveText: "1. e4 e5 1/2-1/2"}
	if gameHasSourceError(goodMovetext) {
		t.Fatal("valid movetext result must not be an error")
	}
	bad := PGNGame{Tags: map[string]string{}, MoveText: "1. e4 e5 *"}
	if !gameHasSourceError(bad) {
		t.Fatal("unusable result must be an error")
	}
}

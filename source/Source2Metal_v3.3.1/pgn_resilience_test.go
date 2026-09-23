package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// The initial 100 records of this fixture have the same unusable '*' result
// as Perfect2023_zonder_C67.pgn. The test uses the interactive path and 32
// workers; no user input may be required to reach the later records/sources.
func TestHundredUnusableRecordsDoNotBlockFollowingGamesOrSources(t *testing.T) {
	for _, laterGame := range []bool{false, true} {
		t.Run(fmt.Sprintf("valid_later_in_same_source_%t", laterGame), func(t *testing.T) {
			dir := t.TempDir()
			bad := filepath.Join(dir, "01_bad.pgn")
			good := filepath.Join(dir, "02_good.pgn")
			var records strings.Builder
			for i := 0; i < 100; i++ {
				fmt.Fprintf(&records, "[Event \"line %d\"]\n[Result \"*\"]\n\n1. e4 e5 *\n\n", i)
			}
			if laterGame {
				records.WriteString("[Event \"later game\"]\n[Result \"1-0\"]\n\n1. d4 d5 1-0\n\n")
			}
			if err := os.WriteFile(bad, []byte(records.String()), 0644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(good, []byte("[Event \"next source\"]\n[Result \"0-1\"]\n\n1. c4 e5 0-1\n"), 0644); err != nil {
				t.Fatal(err)
			}
			old := stdin
			stdin = bufio.NewReader(strings.NewReader("2\n"))
			defer func() { stdin = old }()
			inv := Inventory{Sources: []Source{
				{Kind: KindPGN, Path: bad, Base: "01_bad", Bytes: int64(records.Len())},
				{Kind: KindPGN, Path: good, Base: "02_good", Bytes: 60},
			}}
			var c Counters
			cfg := Config{Workers: 32, Interactive: true, MinPly: 2, MaxPly: 10}
			out := filepath.Join(dir, "RAW", "merged", "GAME.pgn")
			if _, err := buildRawGames(inv, out, cfg, &c); err != nil {
				t.Fatal(err)
			}
			want := int64(1)
			if laterGame {
				want++
			}
			if c.GamesSeen != 100+want || c.GamesNoResult != 100 || c.GamesAccepted != want {
				t.Fatalf("seen=%d no result=%d accepted=%d; want %d, 100, %d", c.GamesSeen, c.GamesNoResult, c.GamesAccepted, 100+want, want)
			}
			if found, err := countGeneratedPGNRecords(out); err != nil || found != want {
				t.Fatalf("merged records=%d, err=%v; want %d", found, err, want)
			}
		})
	}
}

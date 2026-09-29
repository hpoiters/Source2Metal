package main

import (
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	"source2metal/internal/bin2pgn"
	"source2metal/internal/cb2pgn"
	"source2metal/internal/ctgmetal"
	"source2metal/internal/ctgraw"
)

func selfTest() error {
	if err := bin2pgn.SelfTest(); err != nil {
		return err
	}
	if err := releaseConsistencyCheck(); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "s2m-selftest-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	pgn := `[Event "T"]
[White "A"]
[Black "B"]
[WhiteElo "2600"]
[BlackElo "2600"]
[Result "1-0"]

1. e4 {x} e5 (1... c5 2. Nf3) 2. Nf3 Nc6 3. Bb5 a6 4. Ba4 Nf6 5. O-O Be7 6. Re1 b5 7. Bb3 d6 8. c3 O-O 9. h3 1-0
`
	if err := os.WriteFile(filepath.Join(tmp, "x.pgn"), []byte(pgn), 0644); err != nil {
		return err
	}
	inv, err := scanSources(tmp)
	if err != nil {
		return err
	}
	layout, err := createLayout(tmp, "test")
	if err != nil {
		return err
	}
	c := Counters{}
	cfg := Config{MaxPly: 12, MinPly: 4, Workers: 2}
	path, err := buildRawGames(inv, layout.RawGamesFile, cfg, &c)
	if err != nil {
		return err
	}
	if c.GamesAccepted != 1 {
		return os.ErrInvalid
	}
	if _, err := os.Stat(path); err != nil {
		return err
	}
	if err := cb2pgn.SelfTestIntegrated(); err != nil {
		return err
	}
	if err := ctgraw.SelfTest(); err != nil {
		return err
	}
	if err := ctgmetal.SelfTest(); err != nil {
		return err
	}
	// Renderer regression: all adaptive progress variants must fit the width
	// after the common safety truncation, including very narrow consoles.
	for _, width := range []int{24, 40, 58, 78, 88, 105, 140} {
		line := cbAdapterProgressLine("CBH", 22390, 128653, time.Now().Add(-17*time.Second), width-1)
		line = fitConsoleLine(line, width-1)
		if utf8.RuneCountInString(line) > width-1 {
			return os.ErrInvalid
		}
	}
	return nil
}

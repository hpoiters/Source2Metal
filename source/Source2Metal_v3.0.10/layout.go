package main

import (
	"os"
	"path/filepath"
)

func createLayout(root, stamp string) (OutputLayout, error) {
	run := filepath.Join(root, "!Source2Metal_Output", stamp)
	rawSeparate := filepath.Join(run, "RAW", "1 - Separate Sources - RAW PGNs")
	rawMerged := filepath.Join(run, "RAW", "2 - Merged Sources - RAW PGNs")
	metalSeparate := filepath.Join(run, "METAL", "1 - Separate Sources - METAL PGNs")
	metalMerged := filepath.Join(run, "METAL", "2 - Merged Sources - METAL PGNs")
	l := OutputLayout{
		RunDir:           run,
		RawDir:           filepath.Join(run, "RAW"),
		RawSeparateDir:   rawSeparate,
		RawMergedDir:     rawMerged,
		MetalDir:         filepath.Join(run, "METAL"),
		MetalSeparateDir: metalSeparate,
		MetalMergedDir:   metalMerged,
		ReportDir:        filepath.Join(run, "REPORTS"),
		TempDir:          filepath.Join(run, "TEMP"),
		RawGamesDir:      rawMerged,
		RawBooksDir:      rawSeparate,
		RawCombinedDir:   rawMerged,
	}
	l.RawGamesFile = filepath.Join(rawMerged, "Source2Metal - Merged GAME Sources - RAW.pgn")
	l.GameMetalFile = filepath.Join(metalMerged, "Source2Metal - Merged GAME Sources - METAL.pgn")
	for _, d := range []string{l.RunDir, l.ReportDir, l.TempDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return OutputLayout{}, err
		}
	}
	return l, nil
}

func pruneEmptyOutputDirs(l OutputLayout) {
	for _, d := range []string{
		l.RawSeparateDir, l.RawMergedDir, l.MetalSeparateDir, l.MetalMergedDir,
		l.RawDir, l.MetalDir,
	} {
		ents, err := os.ReadDir(d)
		if err == nil && len(ents) == 0 {
			_ = os.Remove(d)
		}
	}
}

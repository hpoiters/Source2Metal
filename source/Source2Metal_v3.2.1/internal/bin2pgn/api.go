package bin2pgn

import (
	"fmt"
	"strings"
)

// Version identifies the preserved standalone converter kernel.
const Version = version

// ProgressStage identifies a long-running conversion phase.
type ProgressStage string

const (
	ProgressScan     ProgressStage = "scan"
	ProgressGraph    ProgressStage = "graph"
	ProgressLines    ProgressStage = "lines"
	ProgressWrite    ProgressStage = "write"
	ProgressValidate ProgressStage = "validate"
)

// Progress reports coarse conversion progress. Total is zero when the amount
// of work is not known in advance.
type Progress struct {
	Stage ProgressStage
	Done  int64
	Total int64
}

// ProgressFunc receives coarse progress updates for long-running BIN files.
type ProgressFunc func(Progress)

// Convert writes complete legal book lines and checks reachable move coverage.
func Convert(input, output string, maxPly int) (Stats, error) {
	return ConvertWithProgress(input, output, maxPly, nil)
}

// ConvertWithProgress is Convert with optional progress updates. Very large,
// sorted Polyglot books are read through a bounded-memory disk index instead
// of loading every physical record into a Go map.
func ConvertWithProgress(input, output string, maxPly int, progress ProgressFunc) (Stats, error) {
	if maxPly < 1 || maxPly > 120 {
		return Stats{}, fmt.Errorf("BIN depth must be between 1 and 120 ply")
	}
	return convertBookAdaptive(input, output, maxPly, progress)
}

// SequenceKey checks a complete, single-line PGN movetext and returns its UCI
// sequence. Different move orders remain distinct even after transpositions.
func SequenceKey(movetext string) (string, error) {
	fields := strings.Fields(movetext)
	if len(fields) < 2 {
		return "", fmt.Errorf("BOOK RAW has no complete movetext")
	}
	last := fields[len(fields)-1]
	if last != "*" && last != "1/2-1/2" && last != "1-0" && last != "0-1" {
		return "", fmt.Errorf("BOOK RAW has no result marker")
	}
	// CTG can carry a statistical result; the BIN SAN parser expects '*'.
	fields[len(fields)-1] = "*"
	return cleanPGNSequenceKey(strings.Join(fields, " "))
}

func SelfTest() error { return selfTest() }

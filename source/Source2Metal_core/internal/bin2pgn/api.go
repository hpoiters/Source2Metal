package bin2pgn

import (
	"fmt"
	"strings"
)

// Version identifies the preserved standalone converter kernel.
const Version = version

// Convert writes complete legal book lines and checks reachable move coverage.
func Convert(input, output string, maxPly int) (Stats, error) {
	if maxPly < 1 || maxPly > 120 {
		return Stats{}, fmt.Errorf("BIN depth must be between 1 and 120 ply")
	}
	return convertBook(input, output, maxPly)
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

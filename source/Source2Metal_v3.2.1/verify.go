package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

// countGeneratedPGNRecords is a lightweight structural end check for PGN files
// written by Source2Metal itself. Every generated record contains exactly one
// Source2MetalVersion tag, even when the source game has no Event tag. Counting
// that invariant marker avoids reparsing millions of moves and prevents a valid
// but incomplete source header from causing a false whole-run failure.
func countGeneratedPGNRecords(path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	const chunkSize = 8 << 20
	prefix := []byte("[Source2MetalVersion \"")
	marker := []byte("\n[Source2MetalVersion \"")
	buf := make([]byte, chunkSize)
	tail := make([]byte, 0, len(marker)-1)
	var count int64
	first := true

	for {
		n, rerr := f.Read(buf)
		if n > 0 {
			data := make([]byte, 0, len(tail)+n)
			data = append(data, tail...)
			data = append(data, buf[:n]...)
			if first {
				if bytes.HasPrefix(data, prefix) {
					count++
				}
				first = false
			}
			count += int64(bytes.Count(data, marker))
			keep := len(marker) - 1
			if keep > len(data) {
				keep = len(data)
			}
			tail = append(tail[:0], data[len(data)-keep:]...)
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return 0, fmt.Errorf("lezen %s: %w", path, rerr)
		}
	}
	return count, nil
}

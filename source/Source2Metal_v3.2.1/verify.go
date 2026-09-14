package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

// countGeneratedPGNRecords is a lightweight structural end check for PGN files
// written by Source2Metal itself. Every generated record starts with an Event
// tag at the beginning of a line. Counting that marker avoids reparsing millions
// of moves while still detecting truncated/missing record blocks.
func countGeneratedPGNRecords(path string) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	const chunkSize = 8 << 20
	marker := []byte("\n[Event \"")
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
				if bytes.HasPrefix(data, []byte("[Event \"")) {
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

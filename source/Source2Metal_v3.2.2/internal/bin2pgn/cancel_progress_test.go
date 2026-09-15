package bin2pgn

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCancelableConvertLeavesNoFinalOutput(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "one.bin")
	output := filepath.Join(dir, "out.pgn")

	// One legal Polyglot start-position record: 1.e4.
	record := make([]byte, 16)
	binary.BigEndian.PutUint64(record[0:8], 0x463b96181691fc9c)
	binary.BigEndian.PutUint16(record[8:10], 0x031c)
	binary.BigEndian.PutUint16(record[10:12], 100)
	if err := os.WriteFile(input, record, 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ConvertWithProgressCancelable(input, output, 100, nil, func() bool { return true })
	if !errors.Is(err, ErrCancelled) {
		t.Fatalf("expected ErrCancelled, got %v", err)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("cancelled conversion created final output: %v", err)
	}
	if _, err := os.Stat(output + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("cancelled conversion left temporary output: %v", err)
	}
}

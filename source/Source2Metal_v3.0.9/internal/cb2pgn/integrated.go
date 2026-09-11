package cb2pgn

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ProgressFunc is Source2Metal's optional console-progress bridge. The
// conversion/decoding algorithms remain the preserved CB2PGN v0.1.3 code;
// only display ownership moves to Source2Metal while this callback is active.
type ProgressFunc func(label string, done, total int64, start time.Time)

var integratedProgress ProgressFunc

// ConvertFile keeps the proven CB2PGN v0.1.3 decoders intact and exposes
// them as a Source2Metal adapter. sourcePath may point at any member of a
// complete CBH or 2CBH set. Output is a neutral mainline-only PGN.
func ConvertFile(sourcePath, outputPath string) (Stats, string, error) {
	return ConvertFileWithProgress(sourcePath, outputPath, nil)
}

// ConvertFileWithProgress is identical to ConvertFile but lets Source2Metal
// render progress itself. Conversion is sequential in Source2Metal, so this
// temporary package-level sink cannot overlap between adapters.
func ConvertFileWithProgress(sourcePath, outputPath string, progress ProgressFunc) (Stats, string, error) {
	initLocal()
	set, err := setFromArg(sourcePath)
	if err != nil {
		return newStats(), "", err
	}
	if err := ensureParent(outputPath); err != nil {
		return newStats(), set.format, err
	}
	prev := integratedProgress
	integratedProgress = progress
	defer func() { integratedProgress = prev }()

	var st Stats
	if set.format == "2CBH" {
		st, err = convert2(set.root, outputPath)
	} else {
		st, err = convertCBH(set.root, outputPath)
	}
	if err != nil {
		return st, set.format, err
	}
	if st.Converted == 0 && st.Records > 0 {
		return st, set.format, fmt.Errorf("%s-converter leverde 0 partijen uit %d fysieke records", set.format, st.Records)
	}
	return st, set.format, nil
}

func ensureParent(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0755)
}

func SelfTestIntegrated() error {
	initLocal()
	return selfTest()
}

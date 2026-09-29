package ctgraw

type BuildResult struct {
	Games        int
	Plies        int64
	Bytes        int64
	SHA256       string
	DecodeErrors int64
}

// ProgressFunc receives the legacy CTGExtractor live status text. final=true
// means the line is complete and Source2Metal may terminate the live row.
type ProgressFunc func(text string, final bool)

var integratedProgress ProgressFunc

func BuildNeutral(bookName, ctgPath, ctoPath, ctbPath, outPath string, maxPly int) (BuildResult, error) {
	return BuildNeutralWithProgress(bookName, ctgPath, ctoPath, ctbPath, outPath, maxPly, nil)
}

func BuildNeutralWithProgress(bookName, ctgPath, ctoPath, ctbPath, outPath string, maxPly int, progress ProgressFunc) (BuildResult, error) {
	return BuildGraph(ctgPath, ctoPath, ctbPath, outPath, maxPly, 0, progress)
}

func SelfTest() error { return selfTest() }

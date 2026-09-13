package ctgraw

import (
	"os"
)

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
	prev := integratedProgress
	integratedProgress = progress
	defer func() { integratedProgress = prev }()

	var res BuildResult
	r, err := openCTG(ctgPath, ctoPath, ctbPath)
	if err != nil {
		return res, err
	}
	defer r.Close()
	games, plies, err := writeNeutralPGN(r, outPath, bookName, maxPly)
	if err != nil {
		return res, err
	}
	sha, err := fileSHA256(outPath)
	if err != nil {
		return res, err
	}
	st, err := os.Stat(outPath)
	if err != nil {
		return res, err
	}
	res.Games = games
	res.Plies = plies
	res.Bytes = st.Size()
	res.SHA256 = sha
	res.DecodeErrors = r.decodeErrors
	return res, nil
}

func SelfTest() error { return selfTest() }

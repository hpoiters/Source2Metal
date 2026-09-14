package ctgmetal

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type BuildResult struct {
	Status  string
	PGNPath string
	Records int64
	Bytes   int64
	SHA256  string
}

const correctedLearningInstructions = `Source2Metal / Ctg2Metal - INLEREN IN FRITZ
================================================

Gebruik bij "Leren van database" / "Bijleren uit database":

    Overwinningen        AAN
    Verliespartijen      AAN
    Wit                  UIT
    Zwart                UIT
    Speler               UIT
    Spelernaam           LEEG
    Partijen             ALLE partijen

De Metal.pgn gebruikt de spelernaam "Metal", maar er is normaal geen spelersfilter nodig.

WAAROM ZIJN SOMMIGE "PARTIJEN" ZO KORT?
--------------------------------------
Source2Metal_METAL.pgn is geen verzameling gewone gespeelde schaakpartijen. Het
bestand bevat kunstmatig opgebouwde Fritz-leerrecords. Een record stopt zodra
het geselecteerde Metal-anker is bereikt.

Een uitslag als 1-0 of 0-1 is daarbij een LEERSIGNAAL voor de kleur die de
ankerzet speelde; het betekent niet dat een echte partij op dat moment won.

Voorbeeld:
    1. Na3 c5 2. e4 1-0

betekent dus niet "wit won na 2.e4", maar dat 2.e4 in die stelling als
positieve leerzet aan Fritz wordt aangeboden.
`

type ProgressFunc func(text string, final bool)

var integratedLive ProgressFunc
var integratedAutoSkip bool

func BuildMetal(bookName, ctgPath, ctoPath, ctbPath, outDir string, maxPly int) (BuildResult, error) {
	return BuildMetalWithProgress(bookName, ctgPath, ctoPath, ctbPath, outDir, maxPly, nil)
}

func BuildMetalWithProgress(bookName, ctgPath, ctoPath, ctbPath, outDir string, maxPly int, progress ProgressFunc) (BuildResult, error) {
	prev := integratedLive
	prevAuto := integratedAutoSkip
	integratedLive = progress
	integratedAutoSkip = true
	defer func() { integratedLive = prev; integratedAutoSkip = prevAuto }()
	return buildMetalInternal(bookName, ctgPath, ctoPath, ctbPath, outDir, maxPly)
}

func buildMetalInternal(bookName, ctgPath, ctoPath, ctbPath, outDir string, maxPly int) (BuildResult, error) {
	var res BuildResult
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return res, err
	}
	set := BookSet{Base: bookName, CTG: ctgPath, CTO: ctoPath, CTB: ctbPath}
	cfg := Config{MaxPly: maxPly, MaxPlySet: true, NoPause: true}
	err := processBook(set, outDir, cfg)
	// Always overwrite the legacy instruction with the corrected Source2Metal one.
	_ = os.WriteFile(filepath.Join(outDir, "INSTRUCTIE_INLEREN.txt"), []byte(correctedLearningInstructions), 0644)
	if errors.Is(err, errBookSkipped) {
		res.Status = "OVERGESLAGEN"
		return res, nil
	}
	if errors.Is(err, errRunAborted) {
		res.Status = "AFGEBROKEN"
		return res, nil
	}
	if err != nil {
		return res, err
	}
	res.Status = "KLAAR"
	legacyPath := filepath.Join(outDir, "Metal.pgn")
	finalPath := filepath.Join(outDir, "Source2Metal_METAL.pgn")
	if e := os.Rename(legacyPath, finalPath); e != nil {
		if !os.IsNotExist(e) {
			return res, e
		}
	}
	res.PGNPath = finalPath
	if st, e := os.Stat(res.PGNPath); e == nil {
		res.Bytes = st.Size()
	}
	if f, e := os.Open(res.PGNPath); e == nil {
		sc := bufio.NewScanner(f)
		buf := make([]byte, 64*1024)
		sc.Buffer(buf, 2*1024*1024)
		for sc.Scan() {
			if len(sc.Bytes()) >= 7 && string(sc.Bytes()[:7]) == "[Event " {
				res.Records++
			}
		}
		_ = f.Close()
	}
	if data, e := os.ReadFile(filepath.Join(outDir, "Ctg2Metal_report.txt")); e == nil {
		// Report parsing is intentionally lightweight; exact record count
		// remains available in the legacy report itself.
		_ = data
	}
	if res.PGNPath != "" {
		if sha, e := sha256File(res.PGNPath); e == nil {
			res.SHA256 = sha
		}
	}
	return res, nil
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func SelfTest() error { return selfTest() }

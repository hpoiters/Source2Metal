package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir() && st.Size() > 0
}

func buildCombined(rawGames string, rawBooks []string, layout OutputLayout, c *Counters) (string, error) {
	if !fileExists(rawGames) || len(rawBooks) == 0 {
		return "", nil
	}
	if err := os.MkdirAll(layout.RawCombinedDir, 0755); err != nil {
		return "", err
	}
	out := filepath.Join(layout.RawCombinedDir, "Source2Metal - Merged ALL Sources - RAW.pgn")
	f, err := os.Create(out)
	if err != nil {
		return "", err
	}
	bw := bufio.NewWriterSize(f, 8<<20)
	appendOne := func(path string) error {
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		_, cpErr := io.Copy(bw, in)
		_ = in.Close()
		if cpErr != nil {
			return cpErr
		}
		_, err = bw.WriteString("\n\n")
		return err
	}
	fmt.Println(L("MERGED RAW: merge GAME RAW + all CTG RAW sources (streaming; no cross-class deduplication).", "ZUSAMMENGEFÜHRTES RAW: GAME RAW + alle CTG-RAW-Quellen zusammenführen (Streaming; keine klassenübergreifende Deduplizierung).", "SAMENGEVOEGDE RAW: gezamenlijke GAME-RAW + alle CTG-RAW-bronnen samenvoegen (streaming; geen cross-class deduplicatie).", "RAW FUSIONNÉ : fusionner GAME RAW + toutes les sources CTG RAW (streaming ; pas de déduplication interclasse).", "RAW COMBINADO: combinar GAME RAW + todas las fuentes CTG RAW (streaming; sin deduplicación entre clases).", "合并 RAW：合并 GAME RAW 与所有 CTG RAW 源（流式处理；不进行跨类别去重）。", "ОБЪЕДИНЁННЫЙ RAW: объединение GAME RAW + всех источников CTG RAW (потоково; без межклассовой дедупликации)."))
	if err := appendOne(rawGames); err != nil {
		_ = f.Close()
		return "", err
	}
	for _, p := range rawBooks {
		if fileExists(p) {
			if err := appendOne(p); err != nil {
				_ = f.Close()
				return "", err
			}
		}
	}
	if err := bw.Flush(); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	st, err := os.Stat(out)
	if err != nil {
		return "", err
	}
	sha, err := fileSHA256(out)
	if err != nil {
		return "", err
	}
	c.CombinedBytes = st.Size()
	c.CombinedSHA256 = sha
	expected := c.GamesAccepted + c.CTGRawGames
	found, verr := countGeneratedPGNRecords(out)
	if verr != nil {
		return "", fmt.Errorf("RAW_COMBINED eindcontrole: %w", verr)
	}
	if found != expected {
		return "", fmt.Errorf("RAW_COMBINED eindcontrole: verwacht %s records, gevonden %s", fmtInt(expected), fmtInt(found))
	}
	c.CombinedGames = found
	c.CombinedVerified = found
	fmt.Printf(L("Merged ALL SOURCES RAW check: OK - %s records (= GAME RAW + CTG RAW).\n", "Prüfung ALL SOURCES RAW: OK - %s Datensätze (= GAME RAW + CTG RAW).\n", "Samengevoegde ALLE Sources RAW controle: OK - %s records (= GAME-RAW + CTG-RAW).\n", "Contrôle ALL SOURCES RAW fusionné : OK - %s enregistrements (= GAME RAW + CTG RAW).\n", "Comprobación ALL SOURCES RAW combinado: OK - %s registros (= GAME RAW + CTG RAW).\n", "合并 ALL SOURCES RAW 检查：OK - %s 条记录（= GAME RAW + CTG RAW）。\n", "Проверка объединённого ALL SOURCES RAW: OK - %s записей (= GAME RAW + CTG RAW).\n"), fmtInt(found))
	return out, nil
}

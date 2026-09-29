package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"source2metal/internal/ctgmetal"
)

// Overlay sources are deliberately opt-in: a directory named OVERLAY-PGN, or
// a .overlay.pgn suffix. Unknown results never change a normal GAME's route.
func isOverlayPGN(root, path string) bool {
	if strings.HasSuffix(strings.ToLower(path), ".overlay.pgn") {
		return true
	}
	if strings.EqualFold(filepath.Base(root), "OVERLAY-PGN") {
		return true
	}
	rel, err := filepath.Rel(root, filepath.Dir(path))
	if err != nil {
		return false
	}
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if strings.EqualFold(part, "OVERLAY-PGN") {
			return true
		}
	}
	return false
}

type OverlayTrace struct {
	SourcePath, RawPath                           string
	Seen, Accepted, Duplicates, Written, Rejected int64
	Setup, Invalid, Empty                         int64
	Contributed, Covered, NewEdges                int64
}

func overlayPolicyText() string {
	return L("OVERLAY-PGN: opt in with an OVERLAY-PGN folder or .overlay.pgn suffix; legal book lines, result *, no GAME statistics.",
		"OVERLAY-PGN: Ordner OVERLAY-PGN oder Endung .overlay.pgn; legale Buchlinien, Ergebnis *, keine GAME-Statistik.",
		"OVERLAY-PGN: gebruik een map OVERLAY-PGN of de uitgang .overlay.pgn; legale boeklijnen, uitslag *, geen GAME-statistiek.",
		"OVERLAY-PGN : dossier OVERLAY-PGN ou suffixe .overlay.pgn ; lignes légales, résultat *, sans statistiques GAME.",
		"OVERLAY-PGN: carpeta OVERLAY-PGN o sufijo .overlay.pgn; líneas legales, resultado *, sin estadísticas GAME.",
		"OVERLAY-PGN：使用 OVERLAY-PGN 文件夹或 .overlay.pgn 后缀；合法开局线，结果 *，不计入 GAME 统计。",
		"OVERLAY-PGN: папка OVERLAY-PGN или суффикс .overlay.pgn; легальные линии, результат *, без статистики GAME.")
}

func emitOverlay(w io.Writer, san []string) error {
	if _, err := fmt.Fprint(w, "[Event \"Source2Metal opening coverage\"]\n[Site \"?\"]\n[Date \"????.??.??\"]\n[Round \"?\"]\n[White \"Book\"]\n[Black \"Book\"]\n[Result \"*\"]\n[SourceFormat \"OVERLAY-PGN\"]\n"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	for i, move := range san {
		if i%2 == 0 {
			if _, err := fmt.Fprintf(w, "%d. ", i/2+1); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(w, move, " "); err != nil {
			return err
		}
	}
	_, err := fmt.Fprint(w, "*\n\n")
	return err
}

func buildOverlayRAW(inv Inventory, layout OutputLayout, cfg Config, c *Counters) ([]string, error) {
	var paths []string
	seen := make(map[string]bool)
	for _, s := range inv.Sources {
		if s.Kind != KindOverlay {
			continue
		}
		if err := os.MkdirAll(layout.RawBooksDir, 0755); err != nil {
			return paths, err
		}
		path := uniqueSourceFile(layout.RawBooksDir, s.Base, KindOverlay, "RAW", s.Path)
		c.OverlayTraces = append(c.OverlayTraces, OverlayTrace{SourcePath: s.Path, RawPath: path})
		tr := &c.OverlayTraces[len(c.OverlayTraces)-1]
		out, err := os.Create(path)
		if err != nil {
			return paths, err
		}
		w := bufio.NewWriterSize(out, 1<<20)
		err = parsePGNFile(s.Path, func(g PGNGame) error {
			tr.Seen++
			if g.Tags["SetUp"] == "1" || g.Tags["FEN"] != "" {
				tr.Rejected++
				tr.Setup++
				return nil
			}
			tokens, _, _, _ := stripMainline(g.MoveText)
			if len(tokens) == 0 {
				tr.Rejected++
				tr.Empty++
				return nil
			}
			// Validate the entire original mainline, including beyond the depth cap.
			san, _, ok := ctgmetal.ParsePGNMainline(tokens, cfg.MaxPly)
			if !ok {
				tr.Rejected++
				tr.Invalid++
				return nil
			}
			if cfg.MaxPly > 0 && len(san) > cfg.MaxPly {
				san = san[:cfg.MaxPly]
			}
			tr.Accepted++
			// Deduplicate complete canonical move sequences, independent of PGN tags.
			key := strings.Join(san, " ")
			if seen[key] {
				tr.Duplicates++
				return nil
			}
			if err := emitOverlay(w, san); err != nil {
				return err
			}
			seen[key] = true
			tr.Written++
			return nil
		}, nil)
		flushErr := w.Flush()
		closeErr := out.Close()
		if err != nil {
			return paths, err
		}
		if flushErr != nil {
			return paths, flushErr
		}
		if closeErr != nil {
			return paths, closeErr
		}
		found, err := countGeneratedPGNRecords(path)
		if err != nil {
			return paths, err
		}
		if found != tr.Written {
			return paths, fmt.Errorf("overlay RAW verification: %d != %d", found, tr.Written)
		}
		if tr.Written > 0 {
			paths = append(paths, path)
		}
		fmt.Printf("OVERLAY-PGN: %s | read=%d accepted=%d duplicate=%d rejected=%d RAW=%d\n", s.Base, tr.Seen, tr.Accepted, tr.Duplicates, tr.Rejected, tr.Written)
	}
	return paths, nil
}

func overlayEdge(e ctgmetal.PGNEdge) gmEdgeKey { return gmEdgeKey{gmPositionKey{e.A, e.B}, e.Move} }

// appendOverlayMetal leaves the normal final output byte-for-byte intact when
// no overlay is present. With overlays it appends every distinct complete
// canonical mainline. Prefixes and transpositions are not exact duplicates.
// Existing model games keep their result; added book lines retain result *.
func appendOverlayMetal(layout OutputLayout, c *Counters) error {
	if len(c.OverlayTraces) == 0 {
		return nil
	}
	final := filepath.Join(layout.MetalDir, "Metal.pgn")
	covered := make(map[gmEdgeKey]bool)
	fullLines := make(map[string]bool)
	var baseRecords int64
	if fileExists(final) {
		err := parsePGNFile(final, func(g PGNGame) error {
			tokens, _, _, _ := stripMainline(g.MoveText)
			san, edges, ok := ctgmetal.ParsePGNMainline(tokens, len(tokens))
			if !ok {
				return fmt.Errorf("invalid base METAL mainline")
			}
			for _, e := range edges {
				covered[overlayEdge(e)] = true
			}
			fullLines[strings.Join(san, " ")] = true
			baseRecords++
			return nil
		}, nil)
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(layout.MetalDir, 0755); err != nil {
		return err
	}
	out, err := os.CreateTemp(layout.MetalDir, "overlay-*.tmp")
	if err != nil {
		return err
	}
	tmp := out.Name()
	defer os.Remove(tmp)
	defer out.Close()
	w := bufio.NewWriterSize(out, 1<<20)
	if fileExists(final) {
		in, err := os.Open(final)
		if err != nil {
			return err
		}
		_, cpErr := io.Copy(w, in)
		closeErr := in.Close()
		if cpErr != nil {
			return cpErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	traces := append([]OverlayTrace(nil), c.OverlayTraces...)
	var added, duplicates int64
	for i := range traces {
		tr := &traces[i]
		tr.Contributed, tr.Covered, tr.NewEdges = 0, 0, 0
		err := parsePGNFile(tr.RawPath, func(g PGNGame) error {
			tokens, _, _, _ := stripMainline(g.MoveText)
			san, edges, ok := ctgmetal.ParsePGNMainline(tokens, len(tokens))
			if !ok {
				return fmt.Errorf("invalid overlay RAW mainline")
			}
			key := strings.Join(san, " ")
			if fullLines[key] {
				tr.Covered++
				duplicates++
				return nil
			}
			fresh := make(map[gmEdgeKey]bool)
			for _, e := range edges {
				k := overlayEdge(e)
				if !covered[k] {
					fresh[k] = true
				}
			}
			if err := emitOverlay(w, san); err != nil {
				return err
			}
			for k := range fresh {
				covered[k] = true
			}
			fullLines[key] = true
			tr.NewEdges += int64(len(fresh))
			tr.Contributed++
			added++
			return nil
		}, nil)
		if err != nil {
			return err
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	expected := baseRecords + added
	found, err := countGeneratedPGNRecords(tmp)
	if err != nil {
		return err
	}
	if found != expected {
		return fmt.Errorf("overlay final verification: %d != %d", found, expected)
	}
	// Go's Windows rename cannot replace an existing destination. Keep a backup
	// until the verified replacement is in place, and restore it on failure.
	if added > 0 {
		backup := final + ".before-overlay"
		hadBase := fileExists(final)
		if hadBase {
			if err := os.Rename(final, backup); err != nil {
				return err
			}
		}
		if err := os.Rename(tmp, final); err != nil {
			if hadBase {
				if restoreErr := os.Rename(backup, final); restoreErr != nil {
					return fmt.Errorf("%v; restore failed: %v (base retained at %s)", err, restoreErr, backup)
				}
			}
			return err
		}
		if hadBase {
			if err := os.Remove(backup); err != nil {
				return err
			}
		}
	}
	c.OverlayTraces = traces
	c.OverlayFinalized = true
	c.OverlayFinalRecords = expected
	fmt.Printf("OVERLAY-PGN -> Metal.pgn: base=%d | added=%d | exact full-line duplicates=%d | verified total=%d\n", baseRecords, added, duplicates, found)
	return nil
}

func overlayReport(c Counters) string {
	if len(c.OverlayTraces) == 0 {
		return ""
	}
	var b strings.Builder
	fmt.Fprintln(&b, "\nOVERLAY-PGN / BOOK PGN")
	fmt.Fprintln(&b, overlayPolicyText())
	fmt.Fprintln(&b, "Accepted = legal before dedup; accepted = duplicate + RAW. Read = accepted + rejected.")
	fmt.Fprintln(&b, "Final contribution = appended legal book lines. Only exact complete canonical move sequences are deduplicated, regardless of tags/result. Existing model games take precedence. Prefixes and transpositions are retained.")
	fmt.Fprintln(&b, "GAME W/D/L contribution: 0. Result remains *.")
	for _, tr := range c.OverlayTraces {
		fmt.Fprintf(&b, "\nSource: %s\nRead / gelezen: %d\nAccepted / geaccepteerd: %d\nDeduplicated / ontdubbeld: %d\nRejected / afgewezen: %d (FEN/SetUp=%d, illegal=%d, empty=%d)\nRAW: %d\nRAW file: %s\n", tr.SourcePath, tr.Seen, tr.Accepted, tr.Duplicates, tr.Rejected, tr.Setup, tr.Invalid, tr.Empty, tr.Written, tr.RawPath)
		if c.OverlayFinalized {
			fmt.Fprintf(&b, "Exact full-line duplicates / exact dubbele volledige lijnen: %d\nFinal Metal contribution / bijdrage: %d\nNew position/move pairs / nieuwe stelling-zetparen: %d\n", tr.Covered, tr.Contributed, tr.NewEdges)
			if tr.Contributed == 0 {
				fmt.Fprintln(&b, "NO contribution to final Metal / GEEN bijdrage aan definitieve Metal.")
			}
		} else {
			fmt.Fprintln(&b, "Final contribution NOT completed / eindbijdrage NIET voltooid (RAW/inventory mode or failed run).")
		}
	}
	if c.OverlayFinalized {
		fmt.Fprintf(&b, "\nFinal Metal verified records: %d\n", c.OverlayFinalRecords)
	}
	return b.String()
}

func gameContributionText(tr SourceTrace) string {
	if tr.MetalEvidence == 0 {
		return fmt.Sprintf("Source / Bron: %s\nGAME evidence / statistische bijdrage: 0\nMETAL selected: %d\nThis source did NOT contribute to GAME statistics or Metal. Records were rejected, invalid, or already counted from another source.\nDeze bron heeft NIET bijgedragen aan GAME-statistiek of Metal: afgewezen/ongeldige records of duplicaten uit een eerdere bron.\n", tr.SourcePath, tr.MetalSelected)
	}
	return fmt.Sprintf("Source / Bron: %s\nGAME evidence / statistische bijdrage: %d unique legal games\nMETAL selected: %d\nThis source DID contribute to GAME statistics; no own model game was selected.\nDeze bron droeg WEL bij aan de GAME-statistiek, maar leverde geen eigen geselecteerde modelpartij.\n", tr.SourcePath, tr.MetalEvidence, tr.MetalSelected)
}

func writeContributionReports(layout OutputLayout, cfg Config, c Counters) error {
	if len(c.OverlayTraces) > 0 {
		if err := atomicWriteFile(filepath.Join(layout.ReportDir, "Source2Metal_Overlay_Report.txt"), []byte(overlayReport(c)), 0644); err != nil {
			return err
		}
	}
	if cfg.Mode != "all" {
		return nil
	}
	for _, tr := range mergedSourceTraces(c.SourceTraces) {
		if tr.Kind != KindPGN && tr.Kind != KindCBH && tr.Kind != Kind2CBH {
			continue
		}
		if tr.MetalSelected > 0 {
			continue
		}
		if err := os.MkdirAll(layout.MetalSeparateDir, 0755); err != nil {
			return err
		}
		note := tr.MetalPath
		if !strings.HasSuffix(note, " - NO OUTPUT.txt") {
			note = uniqueSourceFile(layout.MetalSeparateDir, tr.Base, tr.Kind, "METAL", tr.SourcePath)
			note = strings.TrimSuffix(note, ".pgn") + " - NO OUTPUT.txt"
		}
		if err := atomicWriteFile(note, []byte(gameContributionText(tr)), 0644); err != nil {
			return err
		}
	}
	return nil
}

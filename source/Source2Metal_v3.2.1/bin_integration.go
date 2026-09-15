package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"source2metal/internal/bin2pgn"
)

func binScopeText() string {
	return L("BIN2PGN 0.1.3: BIN produces BOOK RAW only; no BIN-METAL. GAME length and Elo filters do not apply to BIN.",
		"BIN2PGN 0.1.3: BIN erzeugt nur BOOK RAW; kein BIN-METAL. GAME-Längen- und Elo-Filter gelten nicht für BIN.",
		"BIN2PGN 0.1.3: BIN levert alleen BOOK RAW; geen BIN-METAL. GAME-lengte- en Elo-filters gelden niet voor BIN.",
		"BIN2PGN 0.1.3 : BIN produit uniquement BOOK RAW ; pas de BIN-METAL. Les filtres GAME de longueur et Elo ne s’appliquent pas à BIN.",
		"BIN2PGN 0.1.3: BIN produce solo BOOK RAW; sin BIN-METAL. Los filtros GAME de longitud y Elo no se aplican a BIN.",
		"BIN2PGN 0.1.3：BIN 仅生成 BOOK RAW，不生成 BIN-METAL。GAME 的长度和 Elo 筛选不适用于 BIN。",
		"BIN2PGN 0.1.3: BIN создаёт только BOOK RAW; без BIN-METAL. Фильтры GAME по длине и Elo к BIN не применяются.")
}

func binSummary(c Counters) string {
	return fmt.Sprintf(L("BIN RAW: %d successful | %d failed | %d lines | %d verified\n",
		"BIN RAW: %d erfolgreich | %d fehlgeschlagen | %d Linien | %d verifiziert\n",
		"BIN RAW: %d geslaagd | %d mislukt | %d lijnen | %d geverifieerd\n",
		"BIN RAW : %d réussis | %d échecs | %d lignes | %d vérifiées\n",
		"BIN RAW: %d correctos | %d fallidos | %d líneas | %d verificadas\n",
		"BIN RAW：%d 成功 | %d 失败 | %d 条变例 | %d 已验证\n",
		"BIN RAW: %d успешно | %d ошибок | %d линий | %d проверено\n"), c.BINRawBuilt, c.BINRawFailed, c.BINRawGames, c.BINRawVerified)
}

func formatBINElapsed(seconds int64) string {
	if seconds < 0 {
		seconds = 0
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%dh %02dm %02ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// startBINActivity keeps the console visibly alive during BIN2PGN phases that
// do not have a trustworthy total. In particular, reachability discovery can
// take a long time after the BIN index has been read. We deliberately show no
// invented percentage: only a rotating activity marker, the reliable physical
// BIN-record count (when available), and elapsed active time.
//
// Small BIN files must remain quiet and easy to follow. Therefore the ETA
// explanation is printed only once, and only after 10 seconds of continuous
// processing. It is never repeated by the spinner.
func startBINActivity(source string) func() {
	pr := newProgressRenderer()
	started := time.Now()
	physical := int64(-1)
	if info, err := os.Stat(source); err == nil && info.Size() >= 0 && info.Size()%16 == 0 {
		physical = info.Size() / 16
	}

	done := make(chan struct{})
	stopped := make(chan struct{})
	frames := []string{"|", "/", "-", "\\"}

	go func() {
		defer close(stopped)
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		frame := 0
		noteShown := false
		for {
			elapsed := int64(time.Since(started).Seconds())
			if elapsed < 0 {
				elapsed = 0
			}

			if !noteShown && elapsed >= 10 {
				pr.Clear()
				fmt.Println()
				fmt.Println("  " + L(
					"Estimated remaining time will appear as soon as it can be calculated reliably.",
					"Die geschätzte Restzeit erscheint, sobald sie zuverlässig berechnet werden kann.",
					"Geschatte resttijd verschijnt zodra die betrouwbaar kan worden berekend.",
					"Le temps restant estimé apparaîtra dès qu’il pourra être calculé de façon fiable.",
					"El tiempo restante estimado aparecerá cuando pueda calcularse de forma fiable.",
					"一旦能够可靠计算，预计剩余时间就会显示。",
					"Расчётное оставшееся время появится, как только его можно будет надёжно вычислить."))
				fmt.Println()
				noteShown = true
			}

			pr.Render(func(width int) string {
				working := L("Work in progress...", "Verarbeitung läuft...", "Bezig met verwerken...", "Traitement en cours...", "Procesando...", "正在处理...", "Идёт обработка...")
				active := L("active", "aktiv", "actief", "actif", "activo", "运行", "активно")
				elapsedText := formatBINElapsed(elapsed)
				if physical >= 0 {
					records := L("BIN records", "BIN-Datensätze", "BIN-records", "enregistrements BIN", "registros BIN", "BIN 记录", "BIN-записей")
					return fmt.Sprintf("  %s BIN2PGN: %s | %s: %s | %s: %s", frames[frame], working, records, fmtInt(physical), active, elapsedText)
				}
				return fmt.Sprintf("  %s BIN2PGN: %s | %s: %s", frames[frame], working, active, elapsedText)
			})

			select {
			case <-done:
				return
			case <-ticker.C:
				frame = (frame + 1) % len(frames)
			}
		}
	}()

	return func() {
		close(done)
		<-stopped
		pr.Clear()
	}
}

func buildBINRaw(inv Inventory, layout OutputLayout, cfg Config, c *Counters) ([]string, error) {
	var outputs []string
	for _, s := range inv.Sources {
		if s.Kind != KindBIN {
			continue
		}
		if err := os.MkdirAll(layout.RawBooksDir, 0755); err != nil {
			return outputs, err
		}
		out := uniqueSourceFile(layout.RawBooksDir, s.Base, KindBIN, "RAW", s.Path)
		fmt.Printf("\nBIN RAW: %s\n", s.Path)
		stopActivity := startBINActivity(s.Path)
		stats, err := bin2pgn.Convert(s.Path, out, cfg.MaxPly)
		stopActivity()
		tr := getSourceTrace(c, KindBIN, s.Base, s.Path)
		report := binReport(s.Path, out, stats, cfg.MaxPly)
		reportPath := filepath.Join(layout.ReportDir, filepath.Base(out)+".txt")
		if err != nil {
			c.BINRawFailed++
			tr.RawRejected++
			tr.RawPath = reportPath
			report += "\n" + L("Technical error", "Technischer Fehler", "Technische fout", "Erreur technique", "Error técnico", "技术错误", "Техническая ошибка") + ": " + err.Error() + "\n"
			if e := atomicWriteFile(reportPath, []byte(report), 0644); e != nil {
				return outputs, fmt.Errorf("%v; report: %w", err, e)
			}
			return outputs, fmt.Errorf("BIN RAW %s: %w", s.Path, err)
		}
		c.BINRawBuilt++
		c.BINRawGames += stats.CompleteLines
		c.BINRawVerified += stats.PostValidatedLines
		tr.RawPath, tr.RawSeen = out, stats.Physical
		tr.RawAccepted, tr.RawVerified = stats.CompleteLines, stats.PostValidatedLines
		tr.RawLocalDuplicate = stats.DuplicateLines
		if err := atomicWriteFile(reportPath, []byte(report), 0644); err != nil {
			return outputs, err
		}
		outputs = append(outputs, out)
		fmt.Printf("BIN RAW: OK | %d | %s\n", stats.CompleteLines, out)
	}
	return outputs, nil
}

func binReport(source, output string, s bin2pgn.Stats, maxPly int) string {
	labels := strings.Split(L(
		"Source|Output|Physical records|Reachable positions|Reachable moves|Decode errors|Illegal moves|Complete lines|Verified lines|Coverage misses|Maximum ply",
		"Quelle|Ausgabe|Physische Datensätze|Erreichbare Positionen|Erreichbare Züge|Dekodierfehler|Illegale Züge|Vollständige Linien|Verifizierte Linien|Fehlende Zugabdeckung|Maximale Halbzüge",
		"Bron|Uitvoer|Fysieke records|Bereikbare posities|Bereikbare zetten|Decodefouten|Illegale zetten|Complete lijnen|Geverifieerde lijnen|Ontbrekende zetdekking|Maximale ply",
		"Source|Sortie|Enregistrements physiques|Positions accessibles|Coups accessibles|Erreurs de décodage|Coups illégaux|Lignes complètes|Lignes vérifiées|Coups non couverts|Demi-coups maximum",
		"Fuente|Salida|Registros físicos|Posiciones accesibles|Jugadas accesibles|Errores de decodificación|Jugadas ilegales|Líneas completas|Líneas verificadas|Jugadas sin cobertura|Máximo de medias jugadas",
		"源|输出|物理记录|可达局面|可达走子|解码错误|非法走子|完整变例|已验证变例|未覆盖走子|最大半回合数",
		"Источник|Вывод|Физические записи|Достижимые позиции|Достижимые ходы|Ошибки декодирования|Нелегальные ходы|Полные линии|Проверенные линии|Непокрытые ходы|Максимум полуходов"), "|")
	values := []any{source, output, s.Physical, s.ReachablePositions, s.ReachableRecords, s.DecodeErrors, s.IllegalMoves, s.CompleteLines, s.PostValidatedLines, s.CoverageMisses, maxPly}
	var b strings.Builder
	fmt.Fprintf(&b, "BIN2PGN %s\n%s\n\n", bin2pgn.Version, binScopeText())
	for i, v := range values {
		fmt.Fprintf(&b, "%s: %v\n", labels[i], v)
	}
	return b.String()
}

// mergeBookRAW uses the application's multiline PGN reader for both CTG and BIN.
// Only complete legal move sequences are deduplicated, never positions/prefixes.
func mergeBookRAW(paths []string, layout OutputLayout, c *Counters) (string, error) {
	if len(paths) == 0 {
		return "", nil
	}
	if err := os.MkdirAll(layout.RawMergedDir, 0755); err != nil {
		return "", err
	}
	out := filepath.Join(layout.RawMergedDir, "Source2Metal - Merged BOOK Sources - RAW.pgn")
	f, err := os.CreateTemp(layout.TempDir, "book-merge-*.pgn")
	if err != nil {
		return "", err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	defer f.Close()
	seen := map[string]struct{}{}
	var count, dup int64
	for _, p := range paths {
		err = parsePGNFile(p, func(g PGNGame) error {
			result := g.Tags["Result"]
			if (result != "*" && result != "1/2-1/2" && result != "1-0" && result != "0-1") || g.Tags["FEN"] != "" || g.Tags["SetUp"] == "1" {
				return fmt.Errorf("BOOK RAW requires standard-position lines with a PGN result: %s", p)
			}
			moves := strings.Join(strings.Fields(g.MoveText), " ")
			key, e := bin2pgn.SequenceKey(moves)
			if e != nil {
				return fmt.Errorf("BOOK RAW %s: %w", p, e)
			}
			if _, ok := seen[key]; ok {
				dup++
				return nil
			}
			seen[key] = struct{}{}
			for _, tag := range g.TagOrder {
				if _, e = fmt.Fprintf(f, "[%s \"%s\"]\n", tag, escapeTag(g.Tags[tag])); e != nil {
					return e
				}
			}
			_, e = fmt.Fprintf(f, "\n%s\n\n", moves)
			count++
			return e
		}, nil)
		if err != nil {
			return "", err
		}
	}
	if err = f.Close(); err != nil {
		return "", err
	}
	if count == 0 {
		return "", fmt.Errorf("BOOK RAW merge is empty")
	}
	var verified int64
	err = parsePGNFile(tmp, func(g PGNGame) error {
		_, e := bin2pgn.SequenceKey(strings.Join(strings.Fields(g.MoveText), " "))
		verified++
		return e
	}, nil)
	if err != nil {
		return "", err
	}
	if verified != count {
		return "", fmt.Errorf("BOOK RAW count mismatch: %d / %d", count, verified)
	}
	if err = os.Rename(tmp, out); err != nil {
		return "", err
	}
	c.BookMergedGames, c.BookMergedDuplicates = count, dup
	return out, nil
}

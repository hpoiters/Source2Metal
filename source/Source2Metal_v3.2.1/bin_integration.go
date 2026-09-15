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

func binProgressText(stage bin2pgn.ProgressStage) string {
	switch stage {
	case bin2pgn.ProgressScan:
		return L("BIN index scan", "BIN-Indexscan", "BIN-indexscan", "Analyse de l’index BIN", "Escaneo del índice BIN", "BIN 索引扫描", "Сканирование индекса BIN")
	case bin2pgn.ProgressGraph:
		return L("Reachable positions", "Erreichbare Positionen", "Bereikbare posities", "Positions accessibles", "Posiciones accesibles", "可达局面", "Достижимые позиции")
	case bin2pgn.ProgressLines:
		return L("Build complete lines", "Vollständige Linien erstellen", "Complete lijnen bouwen", "Construire les lignes complètes", "Construir líneas completas", "构建完整变例", "Построение полных линий")
	case bin2pgn.ProgressWrite:
		return L("Write BOOK RAW", "BOOK RAW schreiben", "BOOK RAW schrijven", "Écrire BOOK RAW", "Escribir BOOK RAW", "写入 BOOK RAW", "Запись BOOK RAW")
	case bin2pgn.ProgressValidate:
		return L("Validate BOOK RAW", "BOOK RAW prüfen", "BOOK RAW controleren", "Valider BOOK RAW", "Validar BOOK RAW", "验证 BOOK RAW", "Проверка BOOK RAW")
	default:
		return "BIN"
	}
}

func binProgressPrinter() bin2pgn.ProgressFunc {
	var lastStage bin2pgn.ProgressStage
	lastBucket := -1
	var lastUnknown int64 = -1
	return func(p bin2pgn.Progress) {
		label := binProgressText(p.Stage)
		if p.Total > 0 {
			bucket := int((p.Done * 10) / p.Total)
			if bucket > 10 {
				bucket = 10
			}
			if p.Done == p.Total {
				bucket = 10
			}
			if p.Stage != lastStage || bucket != lastBucket {
				fmt.Printf("  %s: %d%% (%d/%d)\n", label, bucket*10, p.Done, p.Total)
				lastStage, lastBucket, lastUnknown = p.Stage, bucket, -1
			}
			return
		}
		if p.Stage != lastStage || p.Done == 0 || lastUnknown < 0 || p.Done-lastUnknown >= 500_000 {
			fmt.Printf("  %s: %d\n", label, p.Done)
			lastStage, lastBucket, lastUnknown = p.Stage, -1, p.Done
		}
	}
}

func bookMergeBytes(paths []string) int64 {
	var total int64
	for _, p := range paths {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			total += st.Size()
		}
	}
	return total
}

func bookProgressText(verify bool) string {
	if verify {
		return L("Verify BOOK RAW", "BOOK RAW prüfen", "BOOK RAW controleren", "Vérifier BOOK RAW", "Verificar BOOK RAW", "验证 BOOK RAW", "Проверка BOOK RAW")
	}
	return L("Merge BOOK RAW", "BOOK RAW zusammenführen", "BOOK RAW samenvoegen", "Fusionner BOOK RAW", "Combinar BOOK RAW", "合并 BOOK RAW", "Объединение BOOK RAW")
}

func renderBookProgress(r *progressRenderer, verify bool, done, total, records, dup int64, started time.Time, finish bool) {
	elapsed := time.Since(started)
	frame := []string{"|", "/", "-", "\\"}[int(elapsed/(750*time.Millisecond))%4]
	build := func(width int) string {
		phase := bookProgressText(verify)
		if total <= 0 {
			if verify {
				return fmt.Sprintf("  %s %s... | %s %s | T%s", frame, phase, fmtCompactInt(records), L("lines", "Linien", "lijnen", "lignes", "líneas", "变例", "линий"), durShort(elapsed))
			}
			return fmt.Sprintf("  %s %s... | %s %s | %s %s | T%s", frame, phase, fmtCompactInt(records), L("lines", "Linien", "lijnen", "lignes", "líneas", "变例", "линий"), L("dup", "dup", "dup", "dup", "dup", "重复", "дуб"), fmtCompactInt(dup), durShort(elapsed))
		}
		pct := 100 * float64(done) / float64(total)
		if pct < 0 {
			pct = 0
		}
		if pct > 100 {
			pct = 100
		}
		eta := etaString(done, total, elapsed)
		if width >= 105 {
			if verify {
				return fmt.Sprintf("  %s %s [%s] %5.1f%% | %s %s | T%s ETA%s", frame, phase, progressBar(pct, 18), pct, fmtCompactInt(records), L("lines", "Linien", "lijnen", "lignes", "líneas", "变例", "линий"), durShort(elapsed), eta)
			}
			return fmt.Sprintf("  %s %s [%s] %5.1f%% | %s %s | %s %s | T%s ETA%s", frame, phase, progressBar(pct, 18), pct, fmtCompactInt(records), L("lines", "Linien", "lijnen", "lignes", "líneas", "变例", "линий"), L("dup", "dup", "dup", "dup", "dup", "重复", "дуб"), fmtCompactInt(dup), durShort(elapsed), eta)
		}
		if verify {
			return fmt.Sprintf("  %s %s %5.1f%% | %s %s | T%s", frame, phase, pct, fmtCompactInt(records), L("lines", "Linien", "lijnen", "lignes", "líneas", "变例", "линий"), durShort(elapsed))
		}
		return fmt.Sprintf("  %s %s %5.1f%% | %s %s | %s %s | T%s", frame, phase, pct, fmtCompactInt(records), L("lines", "Linien", "lijnen", "lignes", "líneas", "变例", "линий"), L("dup", "dup", "dup", "dup", "dup", "重复", "дуб"), fmtCompactInt(dup), durShort(elapsed))
	}
	if finish {
		r.Finish(build)
	} else {
		r.Render(build)
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
		stats, err := bin2pgn.ConvertWithProgress(s.Path, out, cfg.MaxPly, binProgressPrinter())
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

	totalBytes := bookMergeBytes(paths)
	baseBytes := int64(0)
	mergeStarted := time.Now()
	mergeRenderer := newProgressRenderer()
	lastMergeRender := time.Time{}
	renderMerge := func(done int64, force bool) {
		now := time.Now()
		if !force && !lastMergeRender.IsZero() && now.Sub(lastMergeRender) < 750*time.Millisecond {
			return
		}
		lastMergeRender = now
		renderBookProgress(mergeRenderer, false, done, totalBytes, count, dup, mergeStarted, false)
	}
	renderMerge(0, true)

	for _, p := range paths {
		var sourceBytes int64
		if st, statErr := os.Stat(p); statErr == nil && !st.IsDir() {
			sourceBytes = st.Size()
		}
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
		}, func(done, _ int64) {
			renderMerge(baseBytes+done, false)
		})
		if err != nil {
			mergeRenderer.Clear()
			return "", err
		}
		baseBytes += sourceBytes
		renderMerge(baseBytes, false)
	}
	renderBookProgress(mergeRenderer, false, totalBytes, totalBytes, count, dup, mergeStarted, true)
	fmt.Printf(L("BOOK RAW merge: OK | %s lines | %s duplicates | T%s\n", "BOOK RAW-Zusammenführung: OK | %s Linien | %s Duplikate | T%s\n", "BOOK RAW samenvoegen: OK | %s lijnen | %s doublures | T%s\n", "Fusion BOOK RAW : OK | %s lignes | %s doublons | T%s\n", "Combinación BOOK RAW: OK | %s líneas | %s duplicados | T%s\n", "BOOK RAW 合并：OK | %s 条变例 | %s 条重复 | T%s\n", "Объединение BOOK RAW: OK | %s линий | %s дубликатов | T%s\n"), fmtInt(count), fmtInt(dup), durShort(time.Since(mergeStarted)))

	if err = f.Close(); err != nil {
		return "", err
	}
	if count == 0 {
		return "", fmt.Errorf("BOOK RAW merge is empty")
	}

	var verified int64
	verifyTotal := int64(0)
	if st, statErr := os.Stat(tmp); statErr == nil && !st.IsDir() {
		verifyTotal = st.Size()
	}
	verifyStarted := time.Now()
	verifyRenderer := newProgressRenderer()
	lastVerifyRender := time.Time{}
	renderVerify := func(done int64, force bool) {
		now := time.Now()
		if !force && !lastVerifyRender.IsZero() && now.Sub(lastVerifyRender) < 750*time.Millisecond {
			return
		}
		lastVerifyRender = now
		renderBookProgress(verifyRenderer, true, done, verifyTotal, verified, 0, verifyStarted, false)
	}
	renderVerify(0, true)
	err = parsePGNFile(tmp, func(g PGNGame) error {
		_, e := bin2pgn.SequenceKey(strings.Join(strings.Fields(g.MoveText), " "))
		verified++
		return e
	}, func(done, _ int64) {
		renderVerify(done, false)
	})
	if err != nil {
		verifyRenderer.Clear()
		return "", err
	}
	renderBookProgress(verifyRenderer, true, verifyTotal, verifyTotal, verified, 0, verifyStarted, true)
	fmt.Printf(L("BOOK RAW check: OK | %s/%s lines | T%s\n", "BOOK RAW-Prüfung: OK | %s/%s Linien | T%s\n", "BOOK RAW controle: OK | %s/%s lijnen | T%s\n", "Contrôle BOOK RAW : OK | %s/%s lignes | T%s\n", "Comprobación BOOK RAW: OK | %s/%s líneas | T%s\n", "BOOK RAW 检查：OK | %s/%s 条变例 | T%s\n", "Проверка BOOK RAW: OK | %s/%s линий | T%s\n"), fmtInt(verified), fmtInt(count), durShort(time.Since(verifyStarted)))

	if verified != count {
		return "", fmt.Errorf("BOOK RAW count mismatch: %d / %d", count, verified)
	}
	if err = os.Rename(tmp, out); err != nil {
		return "", err
	}
	c.BookMergedGames, c.BookMergedDuplicates = count, dup
	return out, nil
}

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"source2metal/internal/cb2pgn"
)

// prepareChessBaseSources uses the preserved CB2PGN v0.1.3 decoders as
// adapters. Their neutral PGN is written only below TEMP and then fed into the
// same RAW_GAMES parser/deduplicator as ordinary PGN input. TEMP is removed
// after a successful Source2Metal run.
func prepareChessBaseSources(inv Inventory, layout OutputLayout, c *Counters) []Source {
	var dbs []Source
	for _, s := range inv.Sources {
		if s.Kind == KindCBH || s.Kind == Kind2CBH {
			dbs = append(dbs, s)
		}
	}
	if len(dbs) == 0 {
		return nil
	}
	sort.Slice(dbs, func(i, j int) bool { return strings.ToLower(dbs[i].Path) < strings.ToLower(dbs[j].Path) })

	fmt.Println("\n" + L("ChessBase adapters: CB2PGN v0.1.3 is used unchanged as the decoder base.", "ChessBase-Adapter: CB2PGN v0.1.3 wird unverändert als Decoderbasis verwendet.", "ChessBase-adapters: CB2PGN v0.1.3 wordt ongewijzigd als decoderbasis gebruikt.", "Adaptateurs ChessBase : CB2PGN v0.1.3 est utilisé sans modification comme base de décodage.", "Adaptadores ChessBase: CB2PGN v0.1.3 se usa sin cambios como base de decodificación.", "ChessBase 适配器：CB2PGN v0.1.3 原样作为解码基础使用。", "Адаптеры ChessBase: CB2PGN v0.1.3 используется без изменений как основа декодера."))
	fmt.Println(L("The adapter PGN remains temporarily under TEMP; Source2Metal keeps both per-source RAW and merged GAME RAW.", "Die Adapter-PGN bleibt vorübergehend unter TEMP; Source2Metal speichert sowohl RAW pro Quelle als auch das zusammengeführte GAME RAW.", "De adapter-PGN blijft tijdelijk onder TEMP; Source2Metal bewaart nu zowel een per-bron RAW als de samengevoegde GAME-RAW.", "Le PGN de l’adaptateur reste temporairement sous TEMP ; Source2Metal conserve à la fois le RAW par source et le GAME RAW fusionné.", "El PGN del adaptador permanece temporalmente en TEMP; Source2Metal conserva tanto RAW por fuente como GAME RAW combinado.", "适配器 PGN 暂时保存在 TEMP 下；Source2Metal 同时保留每源 RAW 和合并的 GAME RAW。", "PGN адаптера временно остаётся в TEMP; Source2Metal сохраняет как RAW по каждому источнику, так и объединённый GAME RAW."))

	var out []Source
	for i, s := range dbs {
		format := string(s.Kind)
		if s.Kind == KindCBH {
			c.CBHSetsProcessed++
		} else {
			c.TwoCBHSetsProcessed++
		}
		fmt.Printf("\nChessBase %d/%d: %s (%s)\n", i+1, len(dbs), s.Base, format)
		if !s.Complete {
			fmt.Println("  " + L("SKIPPED: file set is incomplete.", "ÜBERSPRUNGEN: Dateisatz ist unvollständig.", "OVERGESLAGEN: bestandenset is onvolledig.", "IGNORÉ : ensemble de fichiers incomplet.", "OMITIDO: el conjunto de archivos está incompleto.", "已跳过：文件集不完整。", "ПРОПУЩЕНО: набор файлов неполный."))
			if s.Kind == KindCBH {
				c.CBHSetsFailed++
			} else {
				c.TwoCBHSetsFailed++
			}
			_ = writeCBAdapterReport(layout.ReportDir, s, cb2pgn.Stats{}, "ONVOLLEDIG", fmt.Errorf("onvolledige %s-bestandenset", format))
			continue
		}

		tmpDir := filepath.Join(layout.TempDir, "CB2PGN")
		if err := os.MkdirAll(tmpDir, 0755); err != nil {
			fmt.Println("  "+localizeStatus("MISLUKT")+":", err)
			continue
		}
		tmp := filepath.Join(tmpDir, fmt.Sprintf("%03d_%s_%s_Raw.pgn", i+1, safeComponent(s.Base), format))
		start := time.Now()
		pr := newProgressRenderer()
		spinner := newCBActivitySpinner(pr, format, start)
		progressFinished := false
		st, actualFormat, err := cb2pgn.ConvertFileWithProgress(s.Path, tmp, func(label string, done, total int64, pstart time.Time) {
			if total > 0 && done >= total {
				spinner.Stop()
				pr.Finish(func(width int) string { return cbAdapterProgressLine(label, done, total, pstart, width) })
				progressFinished = true
				return
			}
			spinner.Update(label, done, total, pstart)
		})
		spinner.Stop() // Also stop on decoder errors and missing final callbacks.
		if err == nil {
			if !progressFinished {
				pr.Finish(func(width int) string { return cbAdapterProgressLine(format, st.Records, st.Records, start, width) })
			}
		} else {
			pr.Clear()
		}
		if actualFormat != "" {
			format = actualFormat
		}
		if s.Kind == KindCBH {
			c.CBHRecords += st.Records
			c.CBHConverted += st.Converted
			c.CBHSkipped += st.Skipped
			c.CBHPlies += st.Plies
		} else {
			c.TwoCBHRecords += st.Records
			c.TwoCBHConverted += st.Converted
			c.TwoCBHSkipped += st.Skipped
			c.TwoCBHPlies += st.Plies
		}
		status := "KLAAR"
		if err != nil {
			status = "MISLUKT"
			if s.Kind == KindCBH {
				c.CBHSetsFailed++
			} else {
				c.TwoCBHSetsFailed++
			}
			fmt.Println("  "+localizeStatus("MISLUKT")+":", err)
			_ = writeCBAdapterReport(layout.ReportDir, s, st, status, err)
			_ = os.Remove(tmp)
			continue
		}
		if s.Kind == KindCBH {
			c.CBHSetsBuilt++
		} else {
			c.TwoCBHSetsBuilt++
		}
		_ = writeCBAdapterReport(layout.ReportDir, s, st, status, nil)
		stFile, statErr := os.Stat(tmp)
		if statErr != nil {
			fmt.Println("  "+L("FAILED: temporary adapter PGN not found:", "FEHLGESCHLAGEN: temporäre Adapter-PGN nicht gefunden:", "MISLUKT: tijdelijke adapter-PGN niet teruggevonden:", "ÉCHEC : PGN temporaire de l’adaptateur introuvable :", "FALLIDO: no se encontró el PGN temporal del adaptador:", "失败：未找到临时适配器 PGN：", "ОШИБКА: временный PGN адаптера не найден:"), statErr)
			continue
		}
		if st.Inactive > 0 {
			fmt.Printf(L("  Adapter done: %s games | %s physical records | %s inactive | %s skipped | %s ply | %s\n", "  Adapter fertig: %s Partien | %s physische Datensätze | %s inaktiv | %s übersprungen | %s Ply | %s\n", "  Adapter klaar: %s partijen | %s fysieke records | %s niet-actief | %s overgeslagen | %s ply | %s\n", "  Adaptateur terminé : %s parties | %s enregistrements physiques | %s inactifs | %s ignorés | %s ply | %s\n", "  Adaptador listo: %s partidas | %s registros físicos | %s inactivos | %s omitidos | %s ply | %s\n", "  适配器完成：%s 对局 | %s 物理记录 | %s 非活动 | %s 已跳过 | %s ply | %s\n", "  Адаптер готов: %s партий | %s физических записей | %s неактивных | %s пропущено | %s ply | %s\n"),
				fmtInt(st.Converted), fmtInt(st.Records), fmtInt(st.Inactive), fmtInt(st.Skipped), fmtInt(st.Plies), durShort(time.Since(start)))
		} else {
			fmt.Printf(L("  Adapter done: %s/%s games | %s ply | %s skipped | %s\n", "  Adapter fertig: %s/%s Partien | %s Ply | %s übersprungen | %s\n", "  Adapter klaar: %s/%s partijen | %s ply | %s overgeslagen | %s\n", "  Adaptateur terminé : %s/%s parties | %s ply | %s ignorées | %s\n", "  Adaptador listo: %s/%s partidas | %s ply | %s omitidas | %s\n", "  适配器完成：%s/%s 对局 | %s ply | %s 已跳过 | %s\n", "  Адаптер готов: %s/%s партий | %s ply | %s пропущено | %s\n"),
				fmtInt(st.Converted), fmtInt(st.Records), fmtInt(st.Plies), fmtInt(st.Skipped), durShort(time.Since(start)))
		}
		out = append(out, Source{
			Kind: KindPGN, Path: tmp, Base: s.Base, Bytes: stFile.Size(), Complete: true, Origin: s.Path, OriginKind: s.Kind,
		})
	}
	return out
}

func cbAdapterProgressLine(label string, done, total int64, start time.Time, width int) string {
	if total <= 0 {
		return fmt.Sprintf(L("  %s working...", "  %s läuft...", "  %s bezig...", "  %s en cours...", "  %s trabajando...", "  %s 处理中...", "  %s выполняется..."), label)
	}
	frac := float64(done) / float64(total)
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	elapsed := time.Since(start)
	rate := float64(done) / elapsed.Seconds()
	eta := "--:--"
	if elapsed >= 2*time.Second && rate > 0 && done < total {
		eta = durShort(time.Duration(float64(total-done)/rate) * time.Second)
	} else if done >= total {
		eta = "00:00"
	}
	pct := frac * 100
	// Adaptive layouts: never depend on a fixed 32-column bar. This keeps the
	// complete line inside the current console width before the renderer draws.
	if width >= 88 {
		bw := 24
		filled := int(frac*float64(bw) + .5)
		if filled < 0 {
			filled = 0
		}
		if filled > bw {
			filled = bw
		}
		bar := strings.Repeat("#", filled) + strings.Repeat("-", bw-filled)
		return fmt.Sprintf("  %-5s [%s] %5.1f%% | %s/%s | ETA %s", label, bar, pct, fmtInt(done), fmtInt(total), eta)
	}
	if width >= 58 {
		return fmt.Sprintf("  %-5s %5.1f%% | %s/%s | ETA %s", label, pct, fmtInt(done), fmtInt(total), eta)
	}
	return fmt.Sprintf("  %s %5.1f%% | %s/%s", label, pct, fmtInt(done), fmtInt(total))
}

func writeCBAdapterReport(reportDir string, src Source, st cb2pgn.Stats, status string, convErr error) error {
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Source2Metal v%s - CB2PGN v0.1.3 adapterrapport\n\n", version)
	fmt.Fprintf(&b, "Bron           : %s\n", src.Path)
	fmt.Fprintf(&b, "Formaat        : %s\n", src.Kind)
	fmt.Fprintf(&b, "Status         : %s\n", status)
	fmt.Fprintf(&b, "Set compleet   : %v\n", src.Complete)
	fmt.Fprintf(&b, "Records fysiek : %s\n", fmtInt(st.Records))
	fmt.Fprintf(&b, "Niet-actief    : %s\n", fmtInt(st.Inactive))
	fmt.Fprintf(&b, "Geconverteerd  : %s\n", fmtInt(st.Converted))
	fmt.Fprintf(&b, "Overgeslagen   : %s\n", fmtInt(st.Skipped))
	fmt.Fprintf(&b, "Ply            : %s\n", fmtInt(st.Plies))
	fmt.Fprintf(&b, "Variatietakken : %s\n", fmtInt(st.Variations))
	if convErr != nil {
		fmt.Fprintf(&b, "Fout           : %v\n", convErr)
	}
	if len(st.Reasons) > 0 {
		fmt.Fprintln(&b, "\nOVERGESLAGEN - REDENEN")
		keys := make([]string, 0, len(st.Reasons))
		for k := range st.Reasons {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "%-72s %s\n", k, fmtInt(st.Reasons[k]))
		}
	}
	fmt.Fprintln(&b, "\nDe decoder is de geïntegreerde CB2PGN Builder v0.1.3-lijn.")
	fmt.Fprintln(&b, "De tijdelijke PGN gaat daarna door dezelfde Source2Metal RAW-filtering en exacte game-deduplicatie als gewone PGN-bronnen.")
	name := fmt.Sprintf("CB2PGN_%s_%s.txt", src.Kind, safeComponent(src.Base))
	return atomicWriteFile(filepath.Join(reportDir, name), []byte(localizeReportText(b.String())), 0644)
}

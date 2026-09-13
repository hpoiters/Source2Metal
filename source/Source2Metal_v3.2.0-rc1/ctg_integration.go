package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"source2metal/internal/ctgmetal"
	"source2metal/internal/ctgraw"
)

func safeComponent(s string) string {
	r := strings.NewReplacer("\\", "_", "/", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	s = strings.TrimSpace(r.Replace(s))
	if s == "" {
		return "CTG"
	}
	return s
}

func completeCTGSources(inv Inventory) []Source {
	out := make([]Source, 0)
	for _, s := range inv.Sources {
		if s.Kind == KindCTG && s.Complete && len(s.Aux) >= 2 {
			out = append(out, s)
		}
	}
	return out
}

func rawBookSkippable(err error) bool {
	if err == nil {
		return false
	}
	x := strings.ToLower(err.Error())
	return strings.Contains(x, "beginstelling niet gevonden") || strings.Contains(x, "positie niet in cto-index")
}

func buildRawBooks(inv Inventory, layout OutputLayout, cfg Config, c *Counters) ([]string, error) {
	sets := completeCTGSources(inv)
	if len(sets) == 0 {
		fmt.Println(L("No complete CTG sets for CTG RAW; skipped.", "Keine vollständigen CTG-Sets für CTG RAW; übersprungen.", "Geen complete CTG-sets voor CTG RAW; overgeslagen.", "Aucun ensemble CTG complet pour CTG RAW ; ignoré.", "No hay conjuntos CTG completos para CTG RAW; omitido.", "没有用于 CTG RAW 的完整 CTG 集；已跳过。", "Нет полных наборов CTG для CTG RAW; пропущено."))
		return nil, nil
	}
	if err := os.MkdirAll(layout.RawBooksDir, 0755); err != nil {
		return nil, err
	}
	var outputs []string
	for i, s := range sets {
		c.CTGSetsProcessed++
		workDir := filepath.Join(layout.TempDir, "CTG_RAW", safeComponent(s.Base))
		if err := os.MkdirAll(workDir, 0755); err != nil {
			return outputs, err
		}
		tmpOut := filepath.Join(workDir, "raw.pgn")
		finalOut := uniqueSourceFile(layout.RawBooksDir, s.Base, KindCTG, "RAW", s.Path)
		fmt.Printf("\nCTG RAW %d/%d: %s\n", i+1, len(sets), s.Base)
		pr := newProgressRenderer()
		res, err := ctgraw.BuildNeutralWithProgress(s.Base, s.Path, s.Aux[0], s.Aux[1], tmpOut, cfg.MaxPly, func(text string, final bool) {
			if final {
				pr.Finish(func(width int) string { return "  " + U(text) })
			} else {
				pr.Render(func(width int) string { return "  " + U(text) })
			}
		})
		if err != nil {
			pr.Clear()
			_ = os.Remove(tmpOut)
			status := "MISLUKT"
			reason := fmt.Sprintf("CTG RAW kon niet worden gemaakt.\n\nTechnische reden:\n%s", err)
			if rawBookSkippable(err) {
				c.CTGRawSkipped++
				status = "OVERGESLAGEN"
				reason = fmt.Sprintf("CTG RAW is overgeslagen omdat de CTG-route vanuit de normale beginstelling geen bruikbare ingang vond. De exacte indexdiagnose staat hieronder.\n\nTechnische diagnose:\n%s", err)
			} else {
				c.CTGRawFailed++
			}
			failDir, ferr := finalizeFailDir(workDir, layout.RawBooksDir, s.Base+" [CTG]", "RAW", status, reason, "")
			if ferr != nil {
				return outputs, ferr
			}
			tr := getSourceTrace(c, KindCTG, s.Base, s.Path)
			tr.RawPath = filepath.Join(failDir, failExplanationName)
			fmt.Printf(L("RAW %s [CTG]: %s - exact reason is in %q.\n", "RAW %s [CTG]: %s - der genaue Grund steht in %q.\n", "RAW %s [CTG]: %s - concrete reden staat in %q.\n", "RAW %s [CTG] : %s - la raison exacte se trouve dans %q.\n", "RAW %s [CTG]: %s - la razón exacta está en %q.\n", "RAW %s [CTG]：%s - 具体原因见 %q。\n", "RAW %s [CTG]: %s - точная причина указана в %q.\n"), s.Base, localizeStatus(status), failExplanationName)
			continue
		}
		found, verr := countGeneratedPGNRecords(tmpOut)
		if verr != nil || found != int64(res.Games) {
			_ = os.Remove(tmpOut)
			c.CTGRawFailed++
			reason := fmt.Sprintf("RAW faalde bij de structurele eindcontrole. Verwacht %s records, teruggevonden %s.", fmtInt(int64(res.Games)), fmtInt(found))
			if verr != nil {
				reason += "\n\nEindcontrolefout: " + verr.Error()
			}
			failDir, ferr := finalizeFailDir(workDir, layout.RawBooksDir, s.Base+" [CTG]", "RAW", "MISLUKT", reason, "")
			if ferr != nil {
				return outputs, ferr
			}
			tr := getSourceTrace(c, KindCTG, s.Base, s.Path)
			tr.RawPath = filepath.Join(failDir, failExplanationName)
			fmt.Printf(L("RAW %s [CTG]: failed - see %q.\n", "RAW %s [CTG]: fehlgeschlagen - siehe %q.\n", "RAW %s [CTG]: mislukt - zie %q.\n", "RAW %s [CTG] : échec - voir %q.\n", "RAW %s [CTG]: fallido - vea %q.\n", "RAW %s [CTG]：失败 - 见 %q。\n", "RAW %s [CTG]: ошибка - см. %q.\n"), s.Base, failExplanationName)
			continue
		}
		_ = os.Remove(finalOut)
		if err := os.Rename(tmpOut, finalOut); err != nil {
			return outputs, err
		}
		_ = os.RemoveAll(workDir)
		c.CTGRawBuilt++
		c.CTGRawGames += int64(res.Games)
		c.CTGRawPlies += res.Plies
		c.CTGRawBytes += res.Bytes
		c.CTGDecodeErrors += res.DecodeErrors
		c.RawBooksVerified += found
		outputs = append(outputs, finalOut)
		tr := getSourceTrace(c, KindCTG, s.Base, s.Path)
		tr.RawPath = finalOut
		tr.RawAccepted += int64(res.Games)
		tr.RawVerified += found
		fmt.Printf(L("RAW done: %s games | %.1f MB | decode errors %d | check OK\n", "RAW fertig: %s Partien | %.1f MB | Dekodierfehler %d | Prüfung OK\n", "RAW klaar: %s partijen | %.1f MB | decodefouten %d | controle OK\n", "RAW terminé : %s parties | %.1f MB | erreurs de décodage %d | contrôle OK\n", "RAW listo: %s partidas | %.1f MB | errores de decodificación %d | comprobación OK\n", "RAW 完成：%s 对局 | %.1f MB | 解码错误 %d | 检查通过\n", "RAW готов: %s партий | %.1f MB | ошибок декодирования %d | проверка OK\n"), fmtInt(int64(res.Games)), float64(res.Bytes)/(1024*1024), res.DecodeErrors)
		fmt.Println("  "+L("Per-source result:", "Ergebnis pro Quelle:", "Per-bron resultaat:", "Résultat par source :", "Resultado por fuente:", "每源结果：", "Результат по источнику:"), finalOut)
	}
	return outputs, nil
}

func buildMetalFromCTG(inv Inventory, layout OutputLayout, cfg Config, c *Counters) error {
	sets := completeCTGSources(inv)
	if len(sets) == 0 {
		fmt.Println(L("No complete CTG sets for METAL; skipped.", "Keine vollständigen CTG-Sets für METAL; übersprungen.", "Geen complete CTG-sets voor METAL; overgeslagen.", "Aucun ensemble CTG complet pour METAL ; ignoré.", "No hay conjuntos CTG completos para METAL; omitido.", "没有用于 METAL 的完整 CTG 集；已跳过。", "Нет полных наборов CTG для METAL; пропущено."))
		return nil
	}
	for i, s := range sets {
		outDir := uniqueSourceDir(layout.MetalSeparateDir, s.Base, KindCTG, s.Path)
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return err
		}
		fmt.Printf("\nMETAL %d/%d: %s\n", i+1, len(sets), s.Base)
		pr := newProgressRenderer()
		res, err := ctgmetal.BuildMetalWithProgress(s.Base, s.Path, s.Aux[0], s.Aux[1], outDir, cfg.MaxPly, func(text string, final bool) {
			if text == "" {
				pr.Clear()
				return
			}
			if final {
				pr.Finish(func(width int) string { return U(text) })
			} else {
				pr.Render(func(width int) string { return U(text) })
			}
		})
		_ = localizeTextFile(filepath.Join(outDir, "Ctg2Metal_report.txt"))
		_ = localizeTextFile(filepath.Join(outDir, "INSTRUCTIE_INLEREN.txt"))
		if err != nil {
			pr.Clear()
			c.CTGMetalFailed++
			_ = os.Remove(filepath.Join(outDir, "Metal.pgn"))
			_ = os.Remove(filepath.Join(outDir, "Source2Metal_METAL.pgn"))
			reason := "METAL kon door een technische fout niet worden gemaakt.\n\nTechnische reden:\n" + err.Error()
			failDir, ferr := finalizeFailDir(outDir, layout.MetalSeparateDir, filepath.Base(outDir), "METAL", "MISLUKT", reason, "Ctg2Metal_report.txt")
			if ferr != nil {
				return ferr
			}
			tr := getSourceTrace(c, KindCTG, s.Base, s.Path)
			tr.MetalPath = filepath.Join(failDir, failExplanationName)
			fmt.Printf(L("METAL %s: failed - see %q in subfolder.\n", "METAL %s: fehlgeschlagen - siehe %q im Unterordner.\n", "METAL %s: mislukt - zie %q in submap.\n", "METAL %s : échec - voir %q dans le sous-dossier.\n", "METAL %s: fallido - vea %q en la subcarpeta.\n", "METAL %s：失败 - 请查看子文件夹中的 %q。\n", "METAL %s: ошибка - см. %q в подпапке.\n"), s.Base, failExplanationName)
			continue
		}
		switch res.Status {
		case "KLAAR":
			c.CTGMetalBuilt++
			c.CTGMetalBytes += res.Bytes
			if res.PGNPath != "" {
				descriptive := filepath.Join(outDir, fmt.Sprintf("%s [CTG] - METAL.pgn", safeComponent(s.Base)))
				_ = os.Remove(descriptive)
				if err := os.Rename(res.PGNPath, descriptive); err == nil {
					res.PGNPath = descriptive
				}
				tr := getSourceTrace(c, KindCTG, s.Base, s.Path)
				tr.MetalPath = res.PGNPath
				tr.MetalSelected += res.Records
			}
			fmt.Printf(L("METAL done: %s records | %.1f MB\n", "METAL fertig: %s Datensätze | %.1f MB\n", "METAL klaar: %s records | %.1f MB\n", "METAL terminé : %s enregistrements | %.1f MB\n", "METAL listo: %s registros | %.1f MB\n", "METAL 完成：%s 条记录 | %.1f MB\n", "METAL готов: %s записей | %.1f MB\n"), fmtInt(res.Records), float64(res.Bytes)/(1024*1024))
			fmt.Println("  "+L("Per-source result:", "Ergebnis pro Quelle:", "Per-bron resultaat:", "Résultat par source :", "Resultado por fuente:", "每源结果：", "Результат по источнику:"), res.PGNPath)
		case "OVERGESLAGEN", "AFGEBROKEN":
			c.CTGMetalSkipped++
			_ = os.Remove(filepath.Join(outDir, "Metal.pgn"))
			_ = os.Remove(filepath.Join(outDir, "Source2Metal_METAL.pgn"))
			reason := "De CTG-bron is gelezen, maar leverde geen verantwoord Metal-resultaat. Eerst wordt de strikte selectie geprobeerd. Alleen wanneer de diagnostiek aantoonbaar statistisch potentieel ziet, wordt daarna zichtbaar de voorzichtige Kleine/Brede-fallback geprobeerd. Ook die route leverde hier geen voldoende betrouwbaar resultaat; daarom wordt geen lege of twijfelachtige Metal.pgn gemaakt. De volledige selectiediagnose staat hieronder in het Ctg2Metal-rapport."
			failDir, ferr := finalizeFailDir(outDir, layout.MetalSeparateDir, filepath.Base(outDir), "METAL", "OVERGESLAGEN / GEEN OUTPUT", reason, "Ctg2Metal_report.txt")
			if ferr != nil {
				return ferr
			}
			tr := getSourceTrace(c, KindCTG, s.Base, s.Path)
			tr.MetalPath = filepath.Join(failDir, failExplanationName)
			fmt.Printf(L("METAL %s: skipped - see %q in subfolder.\n", "METAL %s: übersprungen - siehe %q im Unterordner.\n", "METAL %s: overgeslagen - zie %q in submap.\n", "METAL %s : ignoré - voir %q dans le sous-dossier.\n", "METAL %s: omitido - vea %q en la subcarpeta.\n", "METAL %s：已跳过 - 请查看子文件夹中的 %q。\n", "METAL %s: пропущено - см. %q в подпапке.\n"), s.Base, failExplanationName)
		default:
			c.CTGMetalFailed++
			reason := "METAL eindigde met een onbekende status: " + res.Status
			failDir, ferr := finalizeFailDir(outDir, layout.MetalSeparateDir, filepath.Base(outDir), "METAL", "MISLUKT", reason, "Ctg2Metal_report.txt")
			if ferr != nil {
				return ferr
			}
			tr := getSourceTrace(c, KindCTG, s.Base, s.Path)
			tr.MetalPath = filepath.Join(failDir, failExplanationName)
			fmt.Printf(L("METAL %s: failed - see %q in subfolder.\n", "METAL %s: fehlgeschlagen - siehe %q im Unterordner.\n", "METAL %s: mislukt - zie %q in submap.\n", "METAL %s : échec - voir %q dans le sous-dossier.\n", "METAL %s: fallido - vea %q en la subcarpeta.\n", "METAL %s：失败 - 请查看子文件夹中的 %q。\n", "METAL %s: ошибка - см. %q в подпапке.\n"), s.Base, failExplanationName)
		}
	}
	return nil
}

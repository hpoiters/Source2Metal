package main

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var errStopCurrentSource = errors.New("stop current source only")

const (
	pgnErrorPromptPercent = int64(10)
	pgnErrorPromptMinSeen = int64(100)
)

func validPGNResult(s string) bool {
	s = strings.TrimSpace(s)
	return s == "1-0" || s == "0-1" || s == "1/2-1/2"
}

// gameHasSourceError counts only genuinely unusable PGN game records. Normal
// Source2Metal filtering (Elo, length, setup positions, duplicates) is not an
// error and must never contribute to the 10% warning threshold.
func gameHasSourceError(g PGNGame) bool {
	if validPGNResult(g.Tags["Result"]) {
		return false
	}
	_, res, _, _ := stripMainline(g.MoveText)
	return !validPGNResult(res)
}

func highPGNErrorRate(seen, bad int64) bool {
	if seen < pgnErrorPromptMinSeen || bad == 0 {
		return false
	}
	return bad*100 >= seen*pgnErrorPromptPercent
}

// askContinueBadPGNSource is asked at most once per source file. In scripted
// / non-interactive runs Source2Metal logs the warning and continues, because
// silently stopping a long unattended batch would recreate the original bug.
func askContinueBadPGNSource(path string, seen, bad int64, interactive bool) bool {
	pct := 100 * float64(bad) / float64(seen)
	fmt.Printf("\n"+L(
		"WARNING: this source has many unusable PGN games: %d of %d (%.1f%%).\n",
		"WARNUNG: Diese Quelle enthält viele unbrauchbare PGN-Partien: %d von %d (%.1f%%).\n",
		"WAARSCHUWING: deze bron bevat veel onbruikbare PGN-partijen: %d van %d (%.1f%%).\n",
		"AVERTISSEMENT : cette source contient de nombreuses parties PGN inutilisables : %d sur %d (%.1f%%).\n",
		"ADVERTENCIA: esta fuente contiene muchas partidas PGN inutilizables: %d de %d (%.1f%%).\n",
		"警告：此来源包含许多无法使用的 PGN 对局：%d / %d (%.1f%%)。\n",
		"ПРЕДУПРЕЖДЕНИЕ: в этом источнике много непригодных PGN-партий: %d из %d (%.1f%%).\n"), bad, seen, pct)
	fmt.Println("  " + path)
	if !interactive {
		fmt.Println(L(
			"Non-interactive run: continuing this source and logging the errors.",
			"Nicht-interaktiver Lauf: Diese Quelle wird weiterverarbeitet; Fehler werden protokolliert.",
			"Niet-interactieve run: deze bron wordt vervolgd en de fouten worden gelogd.",
			"Exécution non interactive : poursuite de cette source et journalisation des erreurs.",
			"Ejecución no interactiva: se continúa esta fuente y se registran los errores.",
			"非交互运行：继续处理此来源并记录错误。",
			"Неинтерактивный запуск: обработка этого источника продолжается, ошибки записываются."))
		return true
	}

	for {
		fmt.Print(L(
			"Choice: [1] Continue this source  [2] Stop only this source and continue with the remaining files  (Enter=1): ",
			"Auswahl: [1] Quelle fortsetzen  [2] Nur diese Quelle stoppen und mit den übrigen Dateien fortfahren  (Enter=1): ",
			"Keuze: [1] Deze bron doorgaan  [2] Alleen deze bron stoppen en met de overige bestanden doorgaan  (Enter=1): ",
			"Choix : [1] Continuer cette source  [2] Arrêter seulement cette source et poursuivre les autres fichiers  (Entrée=1) : ",
			"Opción: [1] Continuar esta fuente  [2] Detener solo esta fuente y continuar con los demás archivos  (Enter=1): ",
			"选择：[1] 继续此来源  [2] 仅停止此来源并继续其他文件 （回车=1）：",
			"Выбор: [1] Продолжить этот источник  [2] Остановить только этот источник и продолжить остальные файлы  (Enter=1): "))
		line, err := stdin.ReadString('\n')
		if err != nil && strings.TrimSpace(line) == "" {
			return true
		}
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "", "1", "d", "doorgaan", "continue", "c", "weiter":
			return true
		case "2", "s", "stop", "stoppen":
			return false
		default:
			fmt.Println(L("Please choose 1 or 2.", "Bitte 1 oder 2 wählen.", "Kies 1 of 2.", "Choisissez 1 ou 2.", "Elija 1 o 2.", "请选择 1 或 2。", "Выберите 1 или 2."))
		}
	}
}

func printRawProgressStopped(fp *fileProgress, total int64, workers int, start time.Time) {
	done := fp.bytesRead.Load()
	processed := fp.processed.Load()
	accepted := fp.accepted.Load()
	elapsed := time.Since(start)
	pct := 0.0
	if total > 0 {
		pct = 100 * float64(done) / float64(total)
		if pct > 100 {
			pct = 100
		}
	}
	secs := elapsed.Seconds()
	pps, mbps := 0.0, 0.0
	if secs > 0 {
		pps = float64(processed) / secs
		mbps = float64(done) / (1024 * 1024) / secs
	}
	if fp.renderer == nil {
		fp.renderer = newProgressRenderer()
	}
	fp.renderer.Finish(func(width int) string {
		return rawProgressLine(width, pct, processed, accepted, fp.duplicate.Load(), fp.rejected.Load(), workers, pps, mbps, elapsed, L("source stopped", "Quelle gestoppt", "bron gestopt", "source arrêtée", "fuente detenida", "来源已停止", "источник остановлен"))
	})
}

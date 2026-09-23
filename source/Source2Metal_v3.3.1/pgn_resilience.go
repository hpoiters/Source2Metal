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

// warnBadPGNSource reports a high error rate once per source. Both interactive
// and scripted runs continue without waiting for input.
func warnBadPGNSource(path string, seen, bad int64) {
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
	fmt.Println(L(
        "Continuing this source automatically; unusable games are skipped.",
        "Diese Quelle wird automatisch fortgesetzt; unbrauchbare Partien werden übersprungen.",
        "Deze bron wordt automatisch vervolgd; onbruikbare partijen worden overgeslagen.",
        "Cette source se poursuit automatiquement ; les parties inutilisables sont ignorées.",
        "Esta fuente continúa automáticamente; se omiten las partidas inutilizables.",
        "自动继续处理此来源；跳过无法使用的对局。",
        "Обработка этого источника продолжается автоматически; непригодные партии пропускаются."))
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

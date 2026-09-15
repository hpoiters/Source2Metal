package main

import (
	"fmt"
	"sync"
	"time"

	"source2metal/internal/bin2pgn"
)

type binProgressTracker struct {
	mu             sync.Mutex
	started        time.Time
	latest         bin2pgn.Progress
	have           bool
	predictedTotal time.Duration
	etaReady       bool
}

func newBINProgressTracker() *binProgressTracker {
	return &binProgressTracker{started: time.Now()}
}

func (t *binProgressTracker) Update(p bin2pgn.Progress) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.latest = p
	t.have = true

	elapsed := time.Since(t.started)
	if p.Overall <= 0 || p.Overall >= 1 || elapsed < 4*time.Second {
		return
	}
	// The early index/normalisation passes say little about the size of the
	// reachable chess graph. Do not show a misleading total ETA yet.
	if p.Phase == "index" || p.Phase == "normalize" {
		return
	}
	if p.Phase == "graph" && p.Current < 50000 {
		return
	}
	rawTotal := time.Duration(float64(elapsed) / p.Overall)
	if rawTotal <= elapsed {
		return
	}
	if t.predictedTotal == 0 {
		t.predictedTotal = rawTotal
	} else {
		// Dampen large jumps as the graph frontier grows or shrinks.
		t.predictedTotal = time.Duration(0.75*float64(t.predictedTotal) + 0.25*float64(rawTotal))
	}
	t.etaReady = true
}

func (t *binProgressTracker) Snapshot() (bin2pgn.Progress, bool, time.Duration, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.have {
		return bin2pgn.Progress{}, false, 0, false
	}
	if !t.etaReady {
		return t.latest, true, 0, false
	}
	remaining := t.predictedTotal - time.Since(t.started)
	if remaining < 0 {
		remaining = 0
	}
	return t.latest, true, remaining, true
}

func convertBINWithProgress(input, output string, maxPly int) (bin2pgn.Stats, error) {
	tracker := newBINProgressTracker()
	renderer := newProgressRenderer()
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		spinner := []string{"|", "/", "-", "\\"}
		frame := 0
		for {
			select {
			case <-done:
				renderer.Clear()
				return
			case <-ticker.C:
				p, have, remaining, etaReady := tracker.Snapshot()
				spin := spinner[frame%len(spinner)]
				frame++
				renderer.Render(func(width int) string {
					return binProgressLine(spin, p, have, remaining, etaReady)
				})
			}
		}
	}()

	stats, err := bin2pgn.ConvertWithProgress(input, output, maxPly, tracker.Update)
	close(done)
	wg.Wait()
	return stats, err
}

func binProgressLine(spin string, p bin2pgn.Progress, have bool, remaining time.Duration, etaReady bool) string {
	working := L("Working...", "Verarbeitung...", "Bezig met verwerken...", "Traitement...", "Procesando...", "处理中...", "Обработка...")
	etaLabel := L("total ETA", "Restzeit gesamt", "resterende tijd totaal", "temps restant total", "tiempo restante total", "总剩余时间", "общее оставшееся время")
	eta := L("calculating...", "wird berechnet...", "wordt berekend...", "calcul...", "calculando...", "计算中...", "рассчитывается...")
	if etaReady {
		eta = "~" + formatBINETA(remaining)
	}
	if !have {
		return fmt.Sprintf("  %s %s | %s: %s", spin, working, etaLabel, eta)
	}

	detail := ""
	switch p.Phase {
	case "index":
		pct := int64(0)
		if p.Total > 0 {
			pct = p.Current * 100 / p.Total
		}
		detail = fmt.Sprintf("BIN-indexscan: %d%% (%d/%d)", pct, p.Current, p.Total)
	case "normalize":
		detail = fmt.Sprintf(L("BIN index organising: %d/%d", "BIN-Index ordnen: %d/%d", "BIN-index ordenen: %d/%d", "Organisation index BIN : %d/%d", "Ordenando índice BIN: %d/%d", "整理 BIN 索引：%d/%d", "Упорядочение BIN-индекса: %d/%d"), p.Current, p.Total)
	case "graph":
		detail = fmt.Sprintf(L("Reachable positions: %d", "Erreichbare Positionen: %d", "Bereikbare posities: %d", "Positions accessibles : %d", "Posiciones accesibles: %d", "可达局面：%d", "Достижимые позиции: %d"), p.ReachablePositions)
	case "lines":
		detail = fmt.Sprintf(L("Building book lines: %d/%d", "Buchlinien bauen: %d/%d", "Boeklijnen bouwen: %d/%d", "Construction des lignes : %d/%d", "Construyendo líneas: %d/%d", "构建书库变例：%d/%d", "Построение книжных линий: %d/%d"), p.Current, p.Total)
	case "write":
		detail = fmt.Sprintf(L("Writing RAW: %d/%d", "RAW schreiben: %d/%d", "RAW schrijven: %d/%d", "Écriture RAW : %d/%d", "Escribiendo RAW: %d/%d", "写入 RAW：%d/%d", "Запись RAW: %d/%d"), p.Current, p.Total)
	case "validate":
		detail = L("Final PGN check", "PGN-Endkontrolle", "PGN-eindcontrole", "Contrôle PGN final", "Control PGN final", "PGN 最终检查", "Финальная проверка PGN")
	case "coverage":
		detail = fmt.Sprintf(L("Coverage check: %d/%d", "Deckungsprüfung: %d/%d", "Dekkingscontrole: %d/%d", "Contrôle couverture : %d/%d", "Control de cobertura: %d/%d", "覆盖检查：%d/%d", "Проверка покрытия: %d/%d"), p.Current, p.Total)
	case "done":
		detail = L("BIN conversion complete", "BIN-Konvertierung fertig", "BIN-conversie gereed", "Conversion BIN terminée", "Conversión BIN terminada", "BIN 转换完成", "Преобразование BIN завершено")
	default:
		detail = working
	}
	return fmt.Sprintf("  %s %s %s | %s: %s", spin, working, detail, etaLabel, eta)
}

func formatBINETA(d time.Duration) string {
	if d <= 0 {
		return "<1 min"
	}
	// Round upward; an ETA that says 0 while work is still running is confusing.
	minutes := int64((d + time.Minute - 1) / time.Minute)
	if minutes < 1 {
		minutes = 1
	}
	if minutes < 60 {
		return fmt.Sprintf("%d min", minutes)
	}
	hours := minutes / 60
	mins := minutes % 60
	if mins == 0 {
		return fmt.Sprintf("%d h", hours)
	}
	return fmt.Sprintf("%d h %d min", hours, mins)
}

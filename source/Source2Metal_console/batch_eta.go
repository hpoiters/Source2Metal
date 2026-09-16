package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Read the converter's actual inventory; do not perform a second directory scan.
var inventoryBIN = regexp.MustCompile(`^\s*\d+\.\s+BIN\s+(.+)$`)

func (p *progressWriter) observeInventory(t string) {
	if strings.HasPrefix(t, "Root:") {
		p.batchPaths = nil
		p.batchIndex = -1
		p.batchElapsed = 0
		p.batchReady = false
	}
	if m := inventoryBIN.FindStringSubmatch(t); m != nil && !p.batchReady {
		p.batchPaths = append(p.batchPaths, strings.TrimSpace(m[1]))
	}
}

func (p *progressWriter) startBatchFile(path string) {
	p.batchIndex = -1
	for i, candidate := range p.batchPaths {
		if candidate == path {
			p.batchIndex = i
			break
		}
	}
	if p.batchIndex < 0 {
		return
	}
	if !p.batchReady {
		p.batchReady = true
		p.batchSamples = make([]etaSample, len(p.batchPaths))
		for i, candidate := range p.batchPaths {
			p.batchSamples[i] = sourceSample(candidate)
		}
	}
	p.lastElapsed = 0
}

func (p *progressWriter) batchRemaining(elapsed time.Duration, count int64) (float64, bool) {
	h := loadHistory(p.historyPath)
	remaining := 0.0
	for i := p.batchIndex; i < len(p.batchPaths); i++ {
		s := p.batchSamples[i]
		if s.Fingerprint == "" {
			return 0, false
		}
		seconds := h.reference(s)
		if seconds <= 0 {
			seconds = float64(s.Bytes) / 1.4e9 * 7200
		}
		if i == p.batchIndex {
			if count >= p.totalRecords && p.roughETA.started {
				seconds -= (elapsed - p.roughETA.start).Seconds()
				// Do not let queued files hide an overrun of the active file.
				if seconds <= 0 {
					return 0, false
				}
			} else if count > 0 && elapsed > 0 {
				seconds += elapsed.Seconds() * float64(p.totalRecords-count) / float64(count)
			} else {
				return 0, false
			}
		} else {
			// History prior to this build has no scan duration; its follow-up time
			// remains the useful reference, with no fabricated scan measurement.
			seconds += h.scanReference(s)
		}
		remaining += seconds
	}
	return remaining, true
}

func (h etaHistory) scanReference(s etaSample) float64 {
	for i := len(h.Samples) - 1; i >= 0; i-- {
		old := h.Samples[i]
		if old.Machine == s.Machine && old.Bytes == s.Bytes && old.Fingerprint == s.Fingerprint {
			return old.ScanSeconds
		}
	}
	return 0
}

func (p *progressWriter) batchRows(rows []string, elapsed time.Duration, count int64, loc locale) []string {
	if !p.batchReady || p.batchIndex < 0 || len(p.batchPaths) < 2 {
		return rows
	}
	labels := []string{"file", "Datei", "bestand", "fichier", "archivo", "文件", "файл"}
	live := rows[2]
	if at := strings.LastIndex(live, ":"); at >= 0 {
		live = live[:at+1] + " " + etaText(p.batchElapsed+elapsed)
	}
	if at := strings.LastIndex(live, "|"); at >= 0 {
		live = live[:at] + fmt.Sprintf("| %s %d/%d ", labels[loc], p.batchIndex+1, len(p.batchPaths)) + live[at:]
	}
	totalLabels := []string{"Rough estimated total time remaining", "Grob geschätzte gesamte Restzeit", "Ruw geschatte totale resttijd", "Temps total restant estimé", "Tiempo total restante estimado", "粗略估计总剩余时间", "Примерное общее оставшееся время"}

	eta := ""
	seconds, known := p.batchRemaining(elapsed, count)
	if known && p.batchElapsed+elapsed >= 5*time.Minute {
		minutes := seconds / 60
		step := 10.0
		if minutes < 30 {
			step = 5
		}
		if minutes >= 90 {
			step = 30
		}
		if minutes >= 180 {
			step = 60
		}
		rounded := max(5, int(minutes/step+0.5)*int(step))
		approx := []string{"about ", "circa ", "circa ", "environ ", "aproximadamente ", "约 ", "около "}
		eta = approx[loc] + coarseDuration(rounded, loc)
	}
	if eta == "" {
		return []string{rows[0], "", live, "", "", ""}
	}
	return []string{rows[0], "", live, "", "  " + totalLabels[loc] + ": " + eta + estimateUpdateSuffix(loc), ""}
}

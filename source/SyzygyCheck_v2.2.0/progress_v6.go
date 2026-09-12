package main

import (
	"fmt"
	"strings"
	"time"
)

func renderProgressV6(width, completedFiles, totalFiles int, displayBytes, exactBytes, totalBytes int64, started time.Time, sessionStartBytes int64, batchIdx, batchCount, batchSize, spin int, liveFromIO bool) {
	clearCurrentLine()
	fmt.Print(formatProgressLineV6(width, completedFiles, totalFiles, displayBytes, exactBytes, totalBytes, started, sessionStartBytes, batchIdx, batchCount, batchSize, spin, liveFromIO))
}

func formatProgressLineV6(width, completedFiles, totalFiles int, displayBytes, exactBytes, totalBytes int64, started time.Time, sessionStartBytes int64, batchIdx, batchCount, batchSize, spin int, liveFromIO bool) string {
	if width < 20 {
		width = 20
	}
	if exactBytes < 0 {
		exactBytes = 0
	}
	if displayBytes < exactBytes {
		displayBytes = exactBytes
	}
	if totalBytes > 0 && displayBytes > totalBytes {
		displayBytes = totalBytes
	}
	if sessionStartBytes < 0 || sessionStartBytes > exactBytes {
		sessionStartBytes = exactBytes
	}

	pctData := 0.0
	if totalBytes > 0 {
		pctData = float64(displayBytes) * 100 / float64(totalBytes)
	}
	elapsed := time.Since(started)
	rateBasis := exactBytes
	approxRate := false
	if liveFromIO && displayBytes > exactBytes {
		rateBasis, approxRate = displayBytes, true
	}
	sessionBytes := rateBasis - sessionStartBytes
	if sessionBytes < 0 {
		sessionBytes = 0
	}
	rate := 0.0
	if sessionBytes > 0 && elapsed > 0 {
		rate = float64(sessionBytes) / elapsed.Seconds()
	}
	eta := "ETA ..."
	if rate > 0 {
		remain := totalBytes - rateBasis
		if remain < 0 {
			remain = 0
		}
		p := "ETA "
		if approxRate {
			p = "~ETA "
		}
		eta = p + fmtDuration(time.Duration(float64(remain)/rate)*time.Second)
	}

	spinners := []string{"|", "/", "-", "\\"}
	activity := spinners[spin%len(spinners)]
	approx := ""
	if liveFromIO && displayBytes > exactBytes {
		approx = "~"
	}
	limit := width - 1 // keep final console cell unused to avoid auto-wrap
	if limit < 1 {
		limit = 1
	}
	prefix := activity + " "
	filesPart := fmt.Sprintf("F%d/%d", completedFiles, totalFiles)
	batchPart := fmt.Sprintf("B%d/%d", batchIdx, batchCount)
	dataPart := fmt.Sprintf("%8.4f%% %s%s/%s", pctData, approx, humanSizeCompact(displayBytes), humanSizeCompact(totalBytes))
	ratePart := ""
	if rate > 0 {
		ratePart = humanRateCompact(rate)
		if approxRate {
			ratePart = "~" + ratePart
		}
	}
	tails := []string{
		strings.TrimSpace(strings.Join(nonEmptyStrings(dataPart, filesPart, batchPart, ratePart, eta), " ")),
		strings.TrimSpace(strings.Join(nonEmptyStrings(dataPart, filesPart, ratePart, eta), " ")),
		strings.TrimSpace(strings.Join(nonEmptyStrings(dataPart, filesPart, eta), " ")),
		strings.TrimSpace(strings.Join(nonEmptyStrings(dataPart, eta), " ")),
		dataPart,
		fmt.Sprintf("%8.4f%%", pctData),
	}
	chosenTail := tails[len(tails)-1]
	barW := limit - runeLen(prefix) - runeLen(" "+chosenTail) - 2
	desiredBar := limit * 65 / 100
	if desiredBar < 24 {
		desiredBar = 24
	}
	if desiredBar > 160 {
		desiredBar = 160
	}
	for _, tail := range tails {
		w := limit - runeLen(prefix) - runeLen(" "+tail) - 2
		if w >= desiredBar {
			chosenTail, barW = tail, w
			break
		}
	}
	if barW < 1 {
		barW = 1
	}
	line := prefix + progressBarFloat(pctData, barW) + " " + chosenTail
	if runeLen(line) > limit {
		line = truncateMiddle(line, limit)
	}
	line = strings.ReplaceAll(strings.ReplaceAll(line, "\r", ""), "\n", "")
	return line
}

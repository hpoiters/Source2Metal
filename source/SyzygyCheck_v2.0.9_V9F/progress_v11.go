package main

import (
	"fmt"
	"strings"
	"time"
)

// progressBlockRenderer owns a fixed three-line live status zone. It never
// grows down the screen while the console width is unchanged.  If the window
// is resized, the caller clears/redraws the static screen and resets this
// renderer before drawing the new-width block.
type progressBlockRenderer struct {
	active bool
}

func (p *progressBlockRenderer) resetAfterFullRedraw() { p.active = false }

func (p *progressBlockRenderer) clear() {
	if !p.active {
		return
	}
	// We are on line 3. Clear 3, then move upward and clear lines 2 and 1.
	// Every live line is constrained to width-1, so no automatic wrapping is
	// relied upon during normal updates.
	fmt.Print("\r\x1b[2K")
	fmt.Print("\x1b[1A\r\x1b[2K")
	fmt.Print("\x1b[1A\r\x1b[2K")
	p.active = false
}

func (p *progressBlockRenderer) render(lines [3]string) {
	p.clear()
	fmt.Print(lines[0], "\n", lines[1], "\n", lines[2])
	p.active = true
}

func progressFileLabel(lang string, current, total int) string {
	if total < 1 {
		total = 1
	}
	if current < 1 {
		current = 1
	}
	if current > total {
		current = total
	}
	switch lang {
	case "nl":
		return fmt.Sprintf("Bezig met bestand: %d/%d", current, total)
	case "de":
		return fmt.Sprintf("Datei wird geprüft: %d/%d", current, total)
	case "fr":
		return fmt.Sprintf("Fichier en cours : %d/%d", current, total)
	case "es":
		return fmt.Sprintf("Comprobando archivo: %d/%d", current, total)
	case "zh":
		return fmt.Sprintf("正在检查文件：%d/%d", current, total)
	case "ru":
		return fmt.Sprintf("Проверяется файл: %d/%d", current, total)
	default:
		return fmt.Sprintf("Checking file: %d/%d", current, total)
	}
}

func readSpeedLabel(lang string) string {
	switch lang {
	case "nl":
		return "Leessnelheid"
	case "de":
		return "Lesegeschwindigkeit"
	case "fr":
		return "Vitesse de lecture"
	case "es":
		return "Velocidad de lectura"
	case "zh":
		return "读取速度"
	case "ru":
		return "Скорость чтения"
	default:
		return "Read speed"
	}
}

// formatReadSpeed presents the average read throughput since the real scan
// started. MB/s and GB/s use decimal SI units, matching common storage-speed
// notation. The same underlying process-I/O signal is already used by the
// progress/remaining-time calculation; this adds only a visible readout.
func formatReadSpeedValue(bytesPerSecond float64) string {
	if bytesPerSecond <= 0 {
		return "..."
	}
	if bytesPerSecond >= 1_000_000_000 {
		return fmt.Sprintf("%.2f GB/s", bytesPerSecond/1_000_000_000)
	}
	if bytesPerSecond >= 1_000_000 {
		return fmt.Sprintf("%.1f MB/s", bytesPerSecond/1_000_000)
	}
	return fmt.Sprintf("%.0f kB/s", bytesPerSecond/1_000)
}

func formatReadSpeed(lang string, bytesPerSecond float64) string {
	return readSpeedLabel(lang) + ": " + formatReadSpeedValue(bytesPerSecond)
}

// averageReadRate is the shared calculation for the live read-speed line and
// the final report.  For a resumed run, bytes already verified before this
// session are excluded, exactly as in the live display.
func averageReadRate(rateBasis, sessionStartBytes int64, elapsed time.Duration) float64 {
	sessionBytes := rateBasis - sessionStartBytes
	if sessionBytes <= 0 || elapsed <= 0 {
		return 0
	}
	return float64(sessionBytes) / elapsed.Seconds()
}

func remainingLabel(lang string) string {
	switch lang {
	case "nl":
		return "Resttijd"
	case "de":
		return "Restzeit"
	case "fr":
		return "Temps restant"
	case "es":
		return "Tiempo restante"
	case "zh":
		return "剩余时间"
	case "ru":
		return "Осталось"
	default:
		return "Remaining"
	}
}

// formatRemainingV11 uses ordinary language instead of the technical ETA
// abbreviation. Most languages use the compact internationally recognisable "h".
// Russian and Chinese show their local hour unit first, followed by "(h)" as an
// international bridge.
func formatRemainingV11(lang string, d time.Duration, approximate bool) string {
	label := remainingLabel(lang)
	if d < 0 {
		d = 0
	}
	approx := ""
	if approximate {
		approx = "~"
	}
	seconds := int64(d.Seconds())
	if seconds < 60 {
		switch lang {
		case "zh":
			return fmt.Sprintf("%s: %s<1 min", label, approx)
		default:
			return fmt.Sprintf("%s: %s<1 min", label, approx)
		}
	}
	minutes := (seconds + 59) / 60 // user-friendly ceiling; never claims zero minutes left.
	if minutes < 60 {
		return fmt.Sprintf("%s: %s%d min", label, approx, minutes)
	}
	h := minutes / 60
	m := minutes % 60
	switch lang {
	case "ru":
		return fmt.Sprintf("%s: %s%d:%02d ч (h)", label, approx, h, m)
	case "zh":
		return fmt.Sprintf("%s: %s%d:%02d 小时 (h)", label, approx, h, m)
	default:
		return fmt.Sprintf("%s: %s%d:%02d h", label, approx, h, m)
	}
}

// estimateCurrentBatchIndex uses the same live process-I/O signal already used
// for the volumetric bar.  Ronald's tbcheck processes its filename arguments
// sequentially, so crossing a file-size boundary is a useful display estimate
// of which file is currently being read.  This is presentation only and has no
// influence on verification or checkpoint results.
func estimateCurrentBatchIndex(batch []item, bytesIntoBatch int64) int {
	if len(batch) == 0 {
		return 0
	}
	if bytesIntoBatch < 0 {
		bytesIntoBatch = 0
	}
	var sum int64
	for i, f := range batch {
		sum += f.size
		if bytesIntoBatch < sum {
			return i
		}
	}
	return len(batch) - 1
}

func formatProgressBlockV11(lang string, width, currentFile, totalFiles int, displayBytes, exactBytes, totalBytes int64, started time.Time, sessionStartBytes int64, spin int, liveFromIO bool) [3]string {
	if width < 20 {
		width = 20
	}
	limit := width - 1 // never use the final visible console cell
	if limit < 1 {
		limit = 1
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

	pct := 0.0
	if totalBytes > 0 {
		pct = float64(displayBytes) * 100 / float64(totalBytes)
	}

	fileLabel := progressFileLabel(lang, currentFile, totalFiles)

	spinners := []string{"|", "/", "-", "\\"}
	activity := spinners[spin%len(spinners)]
	// Two bracket cells plus activity and one space. Use every remaining real
	// console cell for the bar: there is no artificial 20/24/160-cell cap.
	barInner := limit - runeLen(activity) - 1 - 2
	if barInner < 1 {
		barInner = 1
	}
	lineBar := activity + " " + progressBarFloat(pct, barInner)
	if runeLen(lineBar) > limit {
		lineBar = string([]rune(lineBar)[:limit])
	}

	rateBasis := exactBytes
	approxRate := false
	if liveFromIO && displayBytes > exactBytes {
		rateBasis = displayBytes
		approxRate = true
	}
	elapsed := time.Since(started)
	rate := averageReadRate(rateBasis, sessionStartBytes, elapsed)

	lineSpeed := formatReadSpeed(lang, rate)

	rest := remainingLabel(lang) + ": ..."
	if rate > 0 {
		remain := totalBytes - rateBasis
		if remain < 0 {
			remain = 0
		}
		rest = formatRemainingV11(lang, time.Duration(float64(remain)/rate)*time.Second, approxRate)
	}
	approxData := ""
	if liveFromIO && displayBytes > exactBytes {
		approxData = "~"
	}
	data := approxData + humanSizeCompact(displayBytes) + "/" + humanSizeCompact(totalBytes)

	// V9F keeps the two right-hand fields in one visual column. The preferred
	// separation is eight spaces after BOTH left-hand groups. If the lower
	// group is wider, the shared right column moves right so the spacing stays
	// generous and the eye does not need to readapt between lines.
	statusLeft := fmt.Sprintf("%.4f%%   %s", pct, data)
	rightColumn := runeLen(fileLabel) + 8
	if c := runeLen(statusLeft) + 8; c > rightColumn {
		rightColumn = c
	}

	// On narrower consoles, shrink the common gap while preserving column
	// alignment for as long as both complete right-hand fields still fit.
	for rightColumn > runeLen(fileLabel)+1 &&
		(rightColumn+runeLen(lineSpeed) > limit || rightColumn+runeLen(rest) > limit) {
		rightColumn--
	}
	if rightColumn < runeLen(statusLeft)+1 {
		rightColumn = runeLen(statusLeft) + 1
	}

	padToColumn := func(left string, column int) string {
		n := column - runeLen(left)
		if n < 1 {
			n = 1
		}
		return left + strings.Repeat(" ", n)
	}

	line1 := padToColumn(fileLabel, rightColumn) + lineSpeed
	lineStatus := padToColumn(statusLeft, rightColumn) + rest

	if runeLen(line1) > limit || runeLen(lineStatus) > limit {
		// Very narrow consoles fall back independently rather than wrapping.
		availableFile := limit - runeLen(lineSpeed) - 1
		if availableFile >= 6 {
			line1 = truncateMiddle(fileLabel, availableFile) + " " + lineSpeed
		} else {
			line1 = truncateMiddle(fileLabel+" "+lineSpeed, limit)
		}

		candidates := []string{
			fmt.Sprintf("%.4f%%   %s   %s", pct, data, rest),
			fmt.Sprintf("%.4f%%  %s  %s", pct, data, rest),
			fmt.Sprintf("%.2f%%  %s  %s", pct, data, rest),
			fmt.Sprintf("%.2f%%  %s", pct, rest),
			fmt.Sprintf("%.2f%%", pct),
		}
		lineStatus = candidates[len(candidates)-1]
		for _, c := range candidates {
			if runeLen(c) <= limit {
				lineStatus = c
				break
			}
		}
	}
	line1 = strings.ReplaceAll(strings.ReplaceAll(line1, "\r", ""), "\n", "")
	lineBar = strings.ReplaceAll(strings.ReplaceAll(lineBar, "\r", ""), "\n", "")
	lineStatus = strings.ReplaceAll(strings.ReplaceAll(lineStatus, "\r", ""), "\n", "")
	return [3]string{line1, lineBar, lineStatus}
}

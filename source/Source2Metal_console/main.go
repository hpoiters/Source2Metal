package main

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed core.exe
var coreEXE []byte

const buildMarker = "2026-09-23_Source2Metal_v3.3.1"

type locale int

const (
	locEN locale = iota
	locDE
	locNL
	locFR
	locES
	locZH
	locRU
)

type progressWriter struct {
	mu  sync.Mutex
	out io.Writer
	vt  bool

	// Normal output is passed through immediately, but this shadow copy lets us
	// recognize a preceding "BIN RAW: <path>" line without delaying prompts.
	normalLine strings.Builder

	// A carriage return normally starts Source2Metal's one-line live progress.
	afterCR bool
	crBuf   strings.Builder

	ownsProgress   bool
	blockActive    bool
	totalRecords   int64
	binPath        string
	width          func() int
	lastStatic     string
	historyPath    string
	sample         etaSample
	followupStart  time.Time
	sourceModified time.Time
	batchPaths     []string
	batchSamples   []etaSample
	batchIndex     int
	batchReady     bool
	batchElapsed   time.Duration
	lastElapsed    time.Duration
	roughETA       roughEstimate
	progressRows   int
	progressLocale locale

	// Once a language-selection screen has been shown, the next main menu must
	// start on a genuinely fresh console screen.  The embedded core owns all
	// translations; the launcher only guarantees the redraw boundary.
	languageScreenSeen  bool
	menuPrefixRepainted bool
}

func newProgressWriter(out io.Writer, vt bool) *progressWriter {
	return &progressWriter{out: out, vt: vt, width: consoleWidth}
}

func (p *progressWriter) Write(b []byte) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, ch := range b {
		if p.afterCR {
			switch ch {
			case '\r':
				// A new CR terminates the previous live line and starts the next.
				s := p.crBuf.String()
				p.crBuf.Reset()
				if strings.TrimSpace(s) != "" && !p.isCoreErasure(s) {
					if p.isBINProgress(s) {
						p.renderProgress(s)
					} else {
						p.finishBlock()
						p.writeRaw("\r" + s)
						p.observeNormalFragment(s)
					}
				}
				p.afterCR = true
				continue
			case '\n':
				s := p.crBuf.String()
				p.crBuf.Reset()
				p.afterCR = false
				if p.isBINProgress(s) {
					p.renderProgress(s)
				} else if p.isCoreErasure(s) {
					// Ignore a core erasure frame; our renderer clears all rows.
				} else {
					p.finishBlock()
					p.writeRaw("\r" + s + "\n")
					p.observeNormalLine(s)
					p.normalLine.Reset()
				}
				continue
			default:
				p.crBuf.WriteByte(ch)
				s := p.crBuf.String()
				// A valid duration prefix (1m in 1m23s) is not a complete
				// frame. Wait for CR/LF or EOF before rendering progress.
				// This introduces at most one core update of display latency.
				if !p.isBINProgress(s) && p.clearlyNotProgress(s) {
					p.finishBlock()
					p.writeRaw("\r" + s)
					p.observeNormalFragment(s)
					p.crBuf.Reset()
					p.afterCR = false
				}
				continue
			}
		}

		switch ch {
		case '\r':
			p.afterCR = true
			p.crBuf.Reset()
		case '\n':
			p.finishBlock()
			p.writeRaw("\n")
			p.observeNormalLine(p.normalLine.String())
			p.normalLine.Reset()
		default:
			p.finishBlockIfNeededForNormalByte(ch)
			p.writeRaw(string([]byte{ch}))
			p.normalLine.WriteByte(ch)
			p.repaintMenuPrefixIfNeeded()
		}
	}
	return len(b), nil
}

func (p *progressWriter) finishBlockIfNeededForNormalByte(ch byte) {
	// If the child leaves the live-progress line and immediately starts normal
	// output without a newline, move below our three-line block first.
	if (p.blockActive || p.ownsProgress) && ch != 0 {
		p.finishBlock()
	}
}

func (p *progressWriter) clearlyNotProgress(s string) bool {
	// A Source2Metal live BIN line starts with a spinner and reaches "BIN2PGN:"
	// quickly. Do not hold ordinary CR-based output indefinitely.
	if strings.TrimSpace(s) == "" || p.isCoreErasure(s) || strings.Contains(s, "BIN2PGN:") {
		return false
	}
	return len([]rune(s)) > 28
}

// The child renderer erases its own one-line frame before the next update.
// Once we own the BIN block, those erasures must not reach our four rows
// or be mistaken for ordinary output. Parse only its exact known commands;
// all unrelated output still terminates the block and passes through.
func (p *progressWriter) isCoreErasure(s string) bool {
	if !p.ownsProgress {
		return false
	}
	s = strings.ReplaceAll(s, "\x1b[2K", "")
	s = strings.ReplaceAll(s, "\x1b[1A", "")
	return strings.TrimSpace(s) == ""
}

func (p *progressWriter) isBINProgress(s string) bool {
	return strings.Contains(s, "BIN2PGN:")
}

func (p *progressWriter) writeRaw(s string) {
	_, _ = io.WriteString(p.out, s)
}

func (p *progressWriter) observeNormalFragment(s string) {
	for _, r := range s {
		if r == '\n' {
			p.observeNormalLine(p.normalLine.String())
			p.normalLine.Reset()
		} else {
			p.normalLine.WriteRune(r)
		}
	}
}

func (p *progressWriter) observeNormalLine(line string) {
	t := strings.TrimSpace(line)
	p.observeInventory(t)
	if isLanguageHeading(t) {
		p.languageScreenSeen = true
		p.menuPrefixRepainted = false
	}
	if t == "BIN RAW: OK" || strings.HasPrefix(t, "BIN RAW: OK |") {
		p.rememberSuccess()
		p.batchElapsed += p.lastElapsed
		p.lastElapsed = 0
		return
	}
	if !strings.HasPrefix(t, "BIN RAW:") {
		return
	}
	// Never reuse the preceding file's total when a new source cannot
	// be inspected (missing, empty, malformed, or inaccessible).
	p.roughETA = roughEstimate{}
	p.sample = etaSample{}
	p.followupStart = time.Time{}
	p.totalRecords = 0
	p.batchIndex = -1
	p.binPath = ""
	p.lastStatic = ""
	path := strings.TrimSpace(strings.TrimPrefix(t, "BIN RAW:"))
	if path == "" {
		return
	}
	st, err := os.Stat(path)
	if err != nil || st.IsDir() || st.Size() <= 0 {
		return
	}
	// Polyglot BIN consists of fixed 16-byte records. Keep the real physical
	// total only when the file is structurally divisible by 16.
	if st.Size()%16 != 0 {
		return
	}
	p.startBatchFile(path)
	p.binPath = path
	p.totalRecords = st.Size() / 16
	p.sourceModified = st.ModTime()
	if p.historyPath != "" {
		p.sample = sourceSample(path)
		p.roughETA.referenceSeconds = loadHistory(p.historyPath).reference(p.sample)
	}
}

func isLanguageHeading(s string) bool {
	switch strings.TrimSpace(s) {
	case "CHANGE LANGUAGE", "SPRACHE ÄNDERN", "TAAL WIJZIGEN", "CHANGER DE LANGUE", "CAMBIAR IDIOMA", "更改语言", "ИЗМЕНИТЬ ЯЗЫК":
		return true
	default:
		return false
	}
}

func (p *progressWriter) repaintMenuPrefixIfNeeded() {
	if !p.vt || !p.languageScreenSeen || p.menuPrefixRepainted {
		return
	}
	const prefix = "Source2Metal v3.3.0"
	if !strings.HasSuffix(p.normalLine.String(), prefix) {
		return
	}
	// The prefix has already reached the inherited console. Clear the complete
	// viewport and immediately repaint it at home, then pass the rest through.
	// This also removes every old language screen instead of merely scrolling.
	p.writeRaw("\x1b[2J\x1b[H" + prefix)
	p.languageScreenSeen = false
	p.menuPrefixRepainted = true
}

func detectLocale(s string) locale {
	switch {
	case strings.Contains(s, "Verarbeitung läuft"), strings.Contains(s, "Verarbeitung l"):
		return locDE
	case strings.Contains(s, "Bezig met verwerken"):
		return locNL
	case strings.Contains(s, "Traitement en cours"):
		return locFR
	case strings.Contains(s, "Procesando"):
		return locES
	case strings.Contains(s, "正在处理"), strings.Contains(s, "处理中"):
		return locZH
	case strings.Contains(s, "Идёт обработка"), strings.Contains(s, "обработка"):
		return locRU
	default:
		return locEN
	}
}

var digitsRE = regexp.MustCompile(`[0-9]+`)

func parseProgressCount(s string) (int64, bool) {
	parts := strings.Split(s, "|")
	// The real record field contains the literal BIN in every supported
	// translation (BIN-records, BIN records, enregistrements BIN, etc.).
	// Do not assume a fixed field number: the spinner itself may be "|".
	for _, part := range parts {
		if !strings.Contains(strings.ToUpper(part), "BIN") || strings.Contains(part, "BIN2PGN:") {
			continue
		}
		nums := digitsRE.FindAllString(part, -1)
		if len(nums) == 0 {
			continue
		}
		joined := strings.Join(nums, "")
		if n, err := strconv.ParseInt(joined, 10, 64); err == nil {
			return n, true
		}
	}
	return 0, false
}

func parseActiveDuration(s string) (time.Duration, bool) {
	parts := strings.Split(s, "|")
	if len(parts) < 2 {
		return 0, false
	}
	// The final field is Source2Metal's elapsed active time. This remains true
	// even when the spinner itself is the character "|".
	f := strings.TrimSpace(parts[len(parts)-1])
	if i := strings.LastIndex(f, ":"); i >= 0 {
		f = strings.TrimSpace(f[i+1:])
	}
	// Go's duration text is language-neutral (e.g. 0s, 8s, 1m23s).
	d, err := time.ParseDuration(strings.ReplaceAll(f, " ", ""))
	return d, err == nil && d >= 0
}

func groupInt(n int64, loc locale) string {
	if n < 0 {
		n = 0
	}
	raw := strconv.FormatInt(n, 10)
	sep := "."
	switch loc {
	case locEN:
		sep = ","
	case locFR, locRU:
		sep = " "
	case locZH:
		sep = ","
	}
	if len(raw) <= 3 {
		return raw
	}
	first := len(raw) % 3
	if first == 0 {
		first = 3
	}
	var b strings.Builder
	b.WriteString(raw[:first])
	for i := first; i < len(raw); i += 3 {
		b.WriteString(sep)
		b.WriteString(raw[i : i+3])
	}
	return b.String()
}

func percentText(v float64, loc locale) string {
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	s := fmt.Sprintf("%.1f", v)
	switch loc {
	case locDE, locNL, locFR, locES, locRU:
		s = strings.ReplaceAll(s, ".", ",")
	}
	return s + "%"
}

func etaText(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	sec := int64(d.Round(time.Second) / time.Second)
	if sec < 1 {
		sec = 1
	}
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	if h > 0 {
		return fmt.Sprintf("%dh %02dm %02ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// All rows leave the last console column unused, preventing automatic wrapping.
// Non-ASCII runes are conservatively counted as two cells (safe for CJK).
func fitRow(s string, width int) string {
	limit := width - 1
	if limit < 1 {
		return ""
	}
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r", ""), "\n", " ")
	cells := 0
	var b strings.Builder
	for _, r := range s {
		n := 1
		if r > 127 {
			n = 2
		}
		if cells+n > limit {
			break
		}
		if r < 32 || r == 127 {
			continue
		}
		b.WriteRune(r)
		cells += n
	}
	return b.String()
}

func scanStatus(count, total int64, loc locale) string {
	complete := []string{
		"BIN scan complete: %s records; processing continues",
		"BIN-Scan fertig: %s Datensätze; Verarbeitung läuft",
		"BIN-scan gereed: %s records; vervolgverwerking loopt",
		"Analyse BIN terminée : %s entrées ; traitement en cours",
		"Escaneo BIN terminado: %s registros; procesamiento en curso",
		"BIN 扫描完成：%s 条记录；后续处理继续",
		"Сканирование BIN завершено: %s записей; обработка продолжается",
	}
	if total > 0 && count >= total {
		return fmt.Sprintf(complete[loc], groupInt(total, loc))
	}
	labels := []string{"BIN scan", "BIN-Scan", "BIN-scan", "Analyse BIN", "Escaneo BIN", "BIN 扫描", "Сканирование BIN"}
	if total <= 0 || count < 0 {
		return labels[loc] + ": ..."
	}
	return fmt.Sprintf("%s: %s / %s | %s", labels[loc], groupInt(count, loc), groupInt(total, loc), percentText(float64(count)*100/float64(total), loc))
}

func estimateUpdateSuffix(loc locale) string {
	suffixes := []string{"; updated during processing.", "; wird angepasst.", "; wordt bijgesteld.", " ; estimation ajustée.", "; se va ajustando.", "；持续调整。", "; оценка обновляется."}
	return suffixes[loc]
}

func estimateRow(eta string, loc locale) string {
	labels := []string{"Rough estimated total time remaining", "Grob geschätzte gesamte Restzeit", "Ruw geschatte totale resttijd", "Temps total restant estimé", "Tiempo total restante estimado", "粗略估计总剩余时间", "Примерное общее оставшееся время"}
	unknown := []string{"not yet estimable", "noch nicht abschätzbar", "nog niet te schatten", "pas encore estimable", "aún no estimable", "暂时无法估计", "пока нельзя оценить"}
	if eta == "" {
		eta = unknown[loc]
	} else {
		eta = "~" + eta
	}
	return "  " + labels[loc] + ": " + eta
}

func (p *progressWriter) renderProgress(line string) {
	p.ownsProgress = true
	loc := detectLocale(line)
	p.progressLocale = loc
	count, okCount := parseProgressCount(line)
	elapsed, okElapsed := parseActiveDuration(line)
	if okElapsed {
		p.lastElapsed = elapsed
	}
	if !okCount {
		count = -1
	}
	// Remove the obsolete scan counter from the liveness row. The status row
	// owns scan progress; after scanning it explicitly says processing continues.
	parts := strings.Split(strings.TrimSpace(line), "|")
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.Contains(strings.ToUpper(part), "BIN") && !strings.Contains(part, "BIN2PGN:") {
			continue
		}
		kept = append(kept, part)
	}
	live := strings.TrimSpace(strings.Join(kept, "|"))
	rows := []string{scanStatus(count, p.totalRecords, loc), "", live, ""}
	if okElapsed && p.totalRecords > 0 && count >= p.totalRecords {
		if p.followupStart.IsZero() {
			p.followupStart = time.Now()
		}
		if coarse, _ := p.roughETA.update(elapsed, p.totalRecords, loc); coarse != "" {
			if coarse != "" {
				approx := []string{"about ", "circa ", "circa ", "environ ", "aproximadamente ", "约 ", "около "}
				rows[3] = strings.Replace(estimateRow(coarse, loc), ": ~", ": "+approx[loc], 1)
			}
			rows = append(rows[:3], "", rows[3]+estimateUpdateSuffix(loc), "")
		}
	}
	rows = p.batchRows(rows, elapsed, count, loc)
	// Keep the established cursor geometry when an estimate becomes unavailable.
	for len(rows) < p.progressRows {
		rows = append(rows, "")
	}
	if !p.vt {
		// Redirected output cannot rewrite rows. Log only phase transitions,
		// never every timer tick. This also avoids flooding a non-VT console.
		phase := "scan"
		if p.totalRecords > 0 && count >= p.totalRecords {
			phase = "follow-up"
		}
		if p.lastStatic != phase {
			p.writeRaw(strings.Join(rows, "\n") + "\n")
			p.lastStatic = phase
		}
		return
	}
	width := p.width()
	for i := range rows {
		rows[i] = fitRow(rows[i], width)
	}
	if p.blockActive {
		p.writeRaw(fmt.Sprintf("\r\x1b[%dA", p.progressRows-1))
	}
	for i, row := range rows {
		if i > 0 {
			p.writeRaw("\r\n")
		}
		p.writeRaw("\r\x1b[2K" + row)
	}
	p.progressRows = len(rows)
	p.blockActive = true
}

func (p *progressWriter) finishBlock() {
	if p.blockActive {
		// Retire the temporary BIN display before passing through the next phase.
		// Start at its last row and clear upwards, leaving the cursor at its top.
		for i := 0; i < p.progressRows; i++ {
			p.writeRaw("\r\x1b[2K")
			if i+1 < p.progressRows {
				p.writeRaw("\x1b[1A")
			}
		}
		p.writeRaw("\r\n") // Keep a blank separator before the next operation.
		p.blockActive = false
		p.progressRows = 0
	} else if p.ownsProgress && !p.vt {
		// A plain log cannot erase previous lines; explicitly retire its ETA.
		ended := []string{
			"BIN2PGN: previous time estimate no longer applies.",
			"BIN2PGN: bisherige Zeitschätzung gilt nicht mehr.",
			"BIN2PGN: vorige tijdschatting is niet meer van toepassing.",
			"BIN2PGN : l'estimation précédente ne s'applique plus.",
			"BIN2PGN: la estimación anterior ya no se aplica.",
			"BIN2PGN：此前的时间估计不再适用。",
			"BIN2PGN: предыдущая оценка времени больше не применима.",
		}
		p.writeRaw(ended[p.progressLocale] + "\n\n")
	}
	p.ownsProgress = false
}

func (p *progressWriter) CloseDisplay() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.afterCR {
		s := p.crBuf.String()
		p.crBuf.Reset()
		p.afterCR = false
		if p.isBINProgress(s) {
			p.renderProgress(s)
		} else if s != "" && !p.isCoreErasure(s) {
			p.finishBlock()
			p.writeRaw("\r" + s)
		}
	}
	p.finishBlock()
}

func writeEmbeddedCore() (string, error) {
	f, err := os.CreateTemp("", "Source2Metal_eta_core_*.exe")
	if err != nil {
		return "", err
	}
	name := f.Name()
	if _, err = f.Write(coreEXE); err != nil {
		f.Close()
		os.Remove(name)
		return "", err
	}
	if err = f.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}

func main() {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Source2Metal launcher: cannot determine executable path:", err)
		os.Exit(2)
	}
	launchDir := filepath.Dir(exe)
	corePath, err := writeEmbeddedCore()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Source2Metal launcher: cannot prepare core:", err)
		os.Exit(2)
	}
	defer os.Remove(corePath)

	cmd := exec.Command(corePath, os.Args[1:]...)
	cmd.Dir = launchDir
	cmd.Stdin = os.Stdin
	cmd.Env = append(os.Environ(), "SOURCE2METAL_LAUNCH_DIR="+launchDir, "SOURCE2METAL_ETA_LAUNCHER_BUILD="+buildMarker)

	vt := enableVirtualTerminal()
	fmt.Fprintln(os.Stdout, "Build             : "+buildMarker)
	pw := newProgressWriter(os.Stdout, vt)
	pw.historyPath = userHistoryPath()
	cmd.Stdout = pw
	cmd.Stderr = pw

	if err := cmd.Run(); err != nil {
		pw.CloseDisplay()
		if ee, ok := err.(*exec.ExitError); ok {
			os.Exit(ee.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "Source2Metal launcher error:", err)
		os.Exit(2)
	}
	pw.CloseDisplay()
}

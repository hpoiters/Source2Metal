package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type gameSig struct{ A, B uint64 }

type rawJob struct {
	seq  int64
	game PGNGame
}

type rejectKind int

const (
	rejectNone rejectKind = iota
	rejectNoResult
	rejectSetup
	rejectShort
	rejectElo
	rejectEloGap
)

type rawResult struct {
	seq      int64
	game     PGNGame
	toks     []string
	result   string
	sig      gameSig
	reject   rejectKind
	vars     int64
	comments int64
}

type fileProgress struct {
	bytesRead atomic.Int64
	processed atomic.Int64
	accepted  atomic.Int64
	duplicate atomic.Int64
	rejected  atomic.Int64
	renderer  *progressRenderer
}

func sourceOriginalKind(s Source) SourceKind {
	if s.OriginKind != "" {
		return s.OriginKind
	}
	return s.Kind
}

func sourceOriginalPath(s Source) string {
	if s.Origin != "" {
		return s.Origin
	}
	return s.Path
}

func separateRawPath(layout OutputLayout, s Source) string {
	return uniqueSourceFile(layout.RawSeparateDir, s.Base, sourceOriginalKind(s), "RAW", sourceOriginalPath(s))
}

func buildRawGames(inv Inventory, outputPath string, cfg Config, c *Counters, extraSources ...Source) (string, error) {
	var pgns []Source
	for _, s := range inv.Sources {
		if s.Kind == KindPGN {
			pgns = append(pgns, s)
		}
	}
	pgns = append(pgns, extraSources...)
	if len(pgns) == 0 {
		return "", fmt.Errorf("geen GAME-bronnen beschikbaar voor GAME RAW")
	}
	sort.SliceStable(pgns, func(i, j int) bool {
		pi, pj := 0, 0
		if pgns[i].Origin != "" {
			pi = 1
		}
		if pgns[j].Origin != "" {
			pj = 1
		}
		if pi != pj {
			return pi < pj
		}
		return strings.ToLower(sourceOriginalPath(pgns[i])) < strings.ToLower(sourceOriginalPath(pgns[j]))
	})
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return "", err
	}
	rawSeparateDir := filepath.Join(filepath.Dir(filepath.Dir(outputPath)), "1 - Separate Sources - RAW PGNs", "GAME Sources")
	if err := os.MkdirAll(rawSeparateDir, 0755); err != nil {
		return "", err
	}
	f, err := os.Create(outputPath)
	if err != nil {
		return "", err
	}
	w := bufio.NewWriterSize(f, 8<<20)
	seenGlobal := make(map[gameSig]struct{}, 1<<20)
	closeMerged := func(retErr error) (string, error) { _ = w.Flush(); _ = f.Close(); return "", retErr }

	for fi, s := range pgns {
		displayPath := sourceOriginalPath(s)
		if s.Origin != "" {
			displayPath += " [via CB2PGN v0.1.3]"
		}
		fmt.Printf("\nRAW %d/%d: %s\n", fi+1, len(pgns), displayPath)
		perPath := uniqueSourceFile(rawSeparateDir, s.Base, sourceOriginalKind(s), "RAW", sourceOriginalPath(s))
		pf, err := os.Create(perPath)
		if err != nil {
			return closeMerged(err)
		}
		pw := bufio.NewWriterSize(pf, 4<<20)
		seenLocal := make(map[gameSig]struct{}, 1<<16)
		trace := SourceTrace{Kind: sourceOriginalKind(s), Base: s.Base, SourcePath: sourceOriginalPath(s), RawPath: perPath}
		startTime := time.Now()
		fp := &fileProgress{renderer: newProgressRenderer()}
		stopMon := make(chan struct{})
		monDone := make(chan struct{})
		go monitorRawProgress(fp, s.Bytes, cfg.Workers, startTime, stopMon, monDone)
		jobs := make(chan rawJob, maxInt(8, cfg.Workers*4))
		results := make(chan rawResult, maxInt(8, cfg.Workers*4))
		var wg sync.WaitGroup
		for i := 0; i < cfg.Workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := range jobs {
					results <- processRawJob(j, cfg)
				}
			}()
		}
		aggDone := make(chan error, 1)
		go func() {
			pending := make(map[int64]rawResult, cfg.Workers*4)
			next := int64(0)
			var firstErr error
			handle := func(r rawResult) {
				c.GamesSeen++
				trace.RawSeen++
				fp.processed.Add(1)
				c.VariationsStripped += r.vars
				c.CommentsStripped += r.comments
				reject := false
				switch r.reject {
				case rejectNoResult:
					c.GamesNoResult++
					reject = true
				case rejectSetup:
					c.GamesSetup++
					reject = true
				case rejectShort:
					c.GamesShort++
					reject = true
				case rejectElo:
					c.GamesElo++
					reject = true
				case rejectEloGap:
					c.GamesEloGap++
					reject = true
				}
				if reject {
					trace.RawRejected++
					fp.rejected.Add(1)
					return
				}
				if _, ok := seenLocal[r.sig]; ok {
					c.GamesDuplicate++
					trace.RawLocalDuplicate++
					fp.duplicate.Add(1)
					return
				}
				seenLocal[r.sig] = struct{}{}
				outToks := r.toks
				if len(outToks) > cfg.MaxPly {
					outToks = outToks[:cfg.MaxPly]
				}
				if firstErr == nil {
					if e := writeRawGame(pw, r.game, outToks, r.result); e != nil {
						firstErr = e
					}
				}
				if firstErr == nil {
					trace.RawAccepted++
				}
				if _, ok := seenGlobal[r.sig]; ok {
					c.GamesDuplicate++
					trace.RawCrossDuplicate++
					fp.duplicate.Add(1)
					return
				}
				seenGlobal[r.sig] = struct{}{}
				if firstErr == nil {
					if e := writeRawGame(w, r.game, outToks, r.result); e != nil {
						firstErr = e
					}
				}
				if firstErr == nil {
					c.GamesAccepted++
					c.PlyWritten += int64(len(outToks))
					fp.accepted.Add(1)
				}
			}
			for r := range results {
				pending[r.seq] = r
				for {
					rr, ok := pending[next]
					if !ok {
						break
					}
					delete(pending, next)
					handle(rr)
					next++
				}
			}
			aggDone <- firstErr
		}()

		seq := int64(0)
		var sourceSeen, sourceErrors int64
		highErrorAsked := false
		parseErr := parsePGNFile(s.Path, func(g PGNGame) error {
			g.Source = sourceOriginalPath(s)
			sourceSeen++
			badSourceGame := gameHasSourceError(g)
			if badSourceGame {
				sourceErrors++
			}
			jobs <- rawJob{seq: seq, game: g}
			seq++
			if badSourceGame && !highErrorAsked && highPGNErrorRate(sourceSeen, sourceErrors) {
				highErrorAsked = true
				if !askContinueBadPGNSource(displayPath, sourceSeen, sourceErrors, cfg.Interactive) {
					return errStopCurrentSource
				}
			}
			return nil
		}, func(done, total int64) { fp.bytesRead.Store(done) })
		close(jobs)
		wg.Wait()
		close(results)
		aggErr := <-aggDone
		close(stopMon)
		<-monDone
		if parseErr == nil {
			printRawProgressFinal(fp, s.Bytes, cfg.Workers, startTime)
		} else {
			printRawProgressStopped(fp, s.Bytes, cfg.Workers, startTime)
		}
		flushErr := pw.Flush()
		closeErr := pf.Close()
		if aggErr != nil {
			return closeMerged(aggErr)
		}
		if flushErr != nil {
			return closeMerged(flushErr)
		}
		if closeErr != nil {
			return closeMerged(closeErr)
		}

		found, verr := countGeneratedPGNRecords(perPath)
		if verr != nil {
			return closeMerged(fmt.Errorf("per-bron RAW controle %s: %w", s.Base, verr))
		}
		if found != trace.RawAccepted {
			return closeMerged(fmt.Errorf("per-bron RAW controle %s: verwacht %s records, gevonden %s", s.Base, fmtInt(trace.RawAccepted), fmtInt(found)))
		}
		trace.RawVerified = found
		stored := getSourceTrace(c, trace.Kind, trace.Base, trace.SourcePath)
		stored.RawPath = trace.RawPath
		stored.RawSeen += trace.RawSeen
		stored.RawAccepted += trace.RawAccepted
		stored.RawLocalDuplicate += trace.RawLocalDuplicate
		stored.RawCrossDuplicate += trace.RawCrossDuplicate
		stored.RawRejected += trace.RawRejected
		stored.RawVerified += trace.RawVerified
		fmt.Printf(L("  File done: seen %s | per-source RAW %s | local dup %s | cross-source dup %s | rejected %s | merged contribution %s\n", "  Datei fertig: gesehen %s | RAW pro Quelle %s | lokale Duplikate %s | quellenübergreifende Duplikate %s | abgelehnt %s | zusammengeführter Beitrag %s\n", "  Bestand klaar: gezien %s | per-bron RAW %s | lokale dup %s | cross-source dup %s | afgewezen %s | bijdrage samengevoegd %s\n", "  Fichier terminé : vus %s | RAW par source %s | doublons locaux %s | doublons inter-source %s | rejetés %s | contribution fusionnée %s\n", "  Archivo listo: vistos %s | RAW por fuente %s | duplicados locales %s | duplicados entre fuentes %s | rechazados %s | contribución combinada %s\n", "  文件完成：已查看 %s | 每源 RAW %s | 本地重复 %s | 跨源重复 %s | 已拒绝 %s | 合并贡献 %s\n", "  Файл готов: просмотрено %s | RAW по источнику %s | локальные дубликаты %s | межисточниковые дубликаты %s | отклонено %s | вклад в объединение %s\n"),
			fmtInt(trace.RawSeen), fmtInt(trace.RawAccepted), fmtInt(trace.RawLocalDuplicate), fmtInt(trace.RawCrossDuplicate), fmtInt(trace.RawRejected), fmtInt(fp.accepted.Load()))
		fmt.Printf(L("  Per-source result: %s\n", "  Ergebnis pro Quelle: %s\n", "  Per-bron resultaat: %s\n", "  Résultat par source : %s\n", "  Resultado por fuente: %s\n", "  每源结果：%s\n", "  Результат по источнику: %s\n"), perPath)

		if parseErr != nil {
			if parseErr == errStopCurrentSource {
				fmt.Println(L(
					"  Current source stopped by choice; the remaining source files will continue.",
					"  Aktuelle Quelle auf Wunsch gestoppt; die übrigen Quelldateien werden weiterverarbeitet.",
					"  Huidige bron op keuze gestopt; de overige bronbestanden worden verder verwerkt.",
					"  Source actuelle arrêtée par choix ; les autres fichiers source seront poursuivis.",
					"  Fuente actual detenida por elección; los demás archivos fuente continuarán.",
					"  已按选择停止当前来源；其余来源文件将继续处理。",
					"  Текущий источник остановлен по выбору; остальные файлы-источники будут обработаны дальше."))
			} else {
				fmt.Printf("  %s: %v\n", L("WARNING - source read stopped early", "WARNUNG - Lesen der Quelle vorzeitig beendet", "WAARSCHUWING - bron voortijdig gestopt bij lezen", "AVERTISSEMENT - lecture de la source arrêtée prématurément", "ADVERTENCIA - lectura de la fuente detenida antes de tiempo", "警告 - 来源读取提前停止", "ПРЕДУПРЕЖДЕНИЕ - чтение источника преждевременно остановлено"), parseErr)
				fmt.Println(L(
					"  Successfully processed games from this source are kept; the remaining source files will continue.",
					"  Erfolgreich verarbeitete Partien aus dieser Quelle bleiben erhalten; die übrigen Quelldateien werden weiterverarbeitet.",
					"  Succesvol verwerkte partijen uit deze bron blijven behouden; de overige bronbestanden worden verder verwerkt.",
					"  Les parties traitées avec succès de cette source sont conservées ; les autres fichiers source seront poursuivis.",
					"  Las partidas procesadas correctamente de esta fuente se conservan; los demás archivos fuente continuarán.",
					"  此来源中已成功处理的对局会被保留；其余来源文件将继续处理。",
					"  Успешно обработанные партии из этого источника сохраняются; остальные файлы-источники будут обработаны дальше."))
			}
		}
	}
	if err := w.Flush(); err != nil {
		return closeMerged(err)
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	if st, err := os.Stat(outputPath); err == nil {
		c.OutputBytes = st.Size()
	}
	if sha, err := fileSHA256(outputPath); err == nil {
		c.OutputSHA256 = sha
	}
	found, verr := countGeneratedPGNRecords(outputPath)
	if verr != nil {
		return "", fmt.Errorf("samengevoegde GAME RAW eindcontrole: %w", verr)
	}
	if found != c.GamesAccepted {
		return "", fmt.Errorf("samengevoegde GAME RAW eindcontrole: verwacht %s records, gevonden %s", fmtInt(c.GamesAccepted), fmtInt(found))
	}
	c.RawGamesVerified = found
	fmt.Printf(L("Merged GAME RAW check: OK - %s records from %d GAME sources.\n", "Prüfung zusammengeführtes GAME RAW: OK - %s Datensätze aus %d GAME-Quellen.\n", "Samengevoegde GAME RAW controle: OK - %s records uit %d GAME-bronnen.\n", "Contrôle GAME RAW fusionné : OK - %s enregistrements provenant de %d sources GAME.\n", "Comprobación GAME RAW combinado: OK - %s registros de %d fuentes GAME.\n", "合并 GAME RAW 检查：OK - %s 条记录，来自 %d 个 GAME 源。\n", "Проверка объединённого GAME RAW: OK - %s записей из %d GAME-источников.\n"), fmtInt(found), len(pgns))
	fmt.Println("  "+L("Automatically merged to:", "Automatisch zusammengeführt nach:", "Automatisch samengevoegd naar:", "Fusionné automatiquement vers :", "Combinado automáticamente en:", "自动合并到：", "Автоматически объединено в:"), outputPath)
	return outputPath, nil
}

func processRawJob(j rawJob, cfg Config) rawResult {
	toks, res, vars, coms := stripMainline(j.game.MoveText)
	r := rawResult{seq: j.seq, game: j.game, toks: toks, result: res, vars: vars, comments: coms}
	if r.result == "" {
		r.result = j.game.Tags["Result"]
	}
	if r.result != "1-0" && r.result != "0-1" && r.result != "1/2-1/2" {
		r.reject = rejectNoResult
		return r
	}
	if j.game.Tags["SetUp"] == "1" || j.game.Tags["FEN"] != "" {
		r.reject = rejectSetup
		return r
	}
	if len(toks) < cfg.MinPly {
		r.reject = rejectShort
		return r
	}
	we, wok := parseElo(j.game.Tags, "WhiteElo")
	be, bok := parseElo(j.game.Tags, "BlackElo")
	if cfg.MinElo > 0 && (!wok || !bok || we < cfg.MinElo || be < cfg.MinElo) {
		r.reject = rejectElo
		return r
	}
	if cfg.MaxEloGap > 0 && wok && bok {
		d := we - be
		if d < 0 {
			d = -d
		}
		if d > cfg.MaxEloGap {
			r.reject = rejectEloGap
			return r
		}
	}
	fullKey := strings.Join(toks, "\x1f") + "|" + r.result + "|standard"
	h := sha256.Sum256([]byte(fullKey))
	r.sig = gameSig{binary.LittleEndian.Uint64(h[0:8]), binary.LittleEndian.Uint64(h[8:16])}
	return r
}

func monitorRawProgress(fp *fileProgress, total int64, workers int, start time.Time, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	t := time.NewTicker(750 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			printRawProgress(fp, total, workers, start)
		}
	}
}

func printRawProgress(fp *fileProgress, total int64, workers int, start time.Time) {
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
	eta := etaString(done, total, elapsed)
	if fp.renderer == nil {
		fp.renderer = newProgressRenderer()
	}
	fp.renderer.Render(func(width int) string {
		return rawProgressLine(width, pct, processed, accepted, fp.duplicate.Load(), fp.rejected.Load(), workers, pps, mbps, elapsed, eta)
	})
}

func printRawProgressFinal(fp *fileProgress, total int64, workers int, start time.Time) {
	fp.bytesRead.Store(total)
	done := fp.bytesRead.Load()
	processed := fp.processed.Load()
	accepted := fp.accepted.Load()
	elapsed := time.Since(start)
	pct := 100.0
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
		return rawProgressLine(width, pct, processed, accepted, fp.duplicate.Load(), fp.rejected.Load(), workers, pps, mbps, elapsed, "00:00")
	})
}

func rawProgressLine(width int, pct float64, processed, accepted, dup, rejected int64, workers int, pps, mbps float64, elapsed time.Duration, eta string) string {
	shortRun := elapsed < 500*time.Millisecond || processed < 100
	proc := L("P", "V", "V", "T", "P", "处", "О")
	dupLabel := L("dup", "dup", "dup", "dup", "dup", "重复", "дуб")
	rejLabel := L("rej", "abw", "afw", "rej", "rech", "拒", "отк")
	workerLabel := L("W", "W", "W", "W", "W", "工", "П")
	speedNA := L("speed n/a", "Tempo n/v", "snelheid n.v.t.", "vitesse n/d", "velocidad n/d", "速度 不适用", "скорость н/д")
	if width >= 105 {
		bar := progressBar(pct, 20)
		if shortRun {
			return fmt.Sprintf("  [%s] %5.1f%% | %s%s RAW%s %s%s %s%s | %s%d | %s | T%s ETA%s",
				bar, pct, proc, fmtCompactInt(processed), fmtCompactInt(accepted), dupLabel, fmtCompactInt(dup), rejLabel, fmtCompactInt(rejected), workerLabel, workers, speedNA, durShort(elapsed), eta)
		}
		return fmt.Sprintf("  [%s] %5.1f%% | %s%s RAW%s %s%s %s%s | %s%d | %s/s %.1fMB/s | T%s ETA%s",
			bar, pct, proc, fmtCompactInt(processed), fmtCompactInt(accepted), dupLabel, fmtCompactInt(dup), rejLabel, fmtCompactInt(rejected), workerLabel, workers, fmtCompactRate(pps), mbps, durShort(elapsed), eta)
	}
	if width >= 78 {
		if shortRun {
			return fmt.Sprintf("  %5.1f%% | %s%s RAW%s %s%s %s%s | %s%d | %s | ETA%s", pct, proc, fmtCompactInt(processed), fmtCompactInt(accepted), dupLabel, fmtCompactInt(dup), rejLabel, fmtCompactInt(rejected), workerLabel, workers, speedNA, eta)
		}
		return fmt.Sprintf("  %5.1f%% | %s%s RAW%s %s%s %s%s | %s%d | %s/s | ETA%s", pct, proc, fmtCompactInt(processed), fmtCompactInt(accepted), dupLabel, fmtCompactInt(dup), rejLabel, fmtCompactInt(rejected), workerLabel, workers, fmtCompactRate(pps), eta)
	}
	return fmt.Sprintf("  %5.1f%% | %s%s RAW%s %s%s %s%s | ETA%s", pct, proc, fmtCompactInt(processed), fmtCompactInt(accepted), dupLabel, fmtCompactInt(dup), rejLabel, fmtCompactInt(rejected), eta)
}

func progressBar(pct float64, width int) string {
	if width < 4 {
		width = 4
	}
	filled := int((pct/100)*float64(width) + 0.5)
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return strings.Repeat("#", filled) + strings.Repeat("-", width-filled)
}

func fmtCompactInt(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	}
	if n < 1_000_000_000 {
		return fmt.Sprintf("%.2fM", float64(n)/1_000_000)
	}
	return fmt.Sprintf("%.2fG", float64(n)/1_000_000_000)
}

func fmtCompactRate(v float64) string {
	if v < 1000 {
		return fmt.Sprintf("%.0f", v)
	}
	if v < 1_000_000 {
		return fmt.Sprintf("%.1fk", v/1_000)
	}
	return fmt.Sprintf("%.2fM", v/1_000_000)
}

func etaString(done, total int64, elapsed time.Duration) string {
	if total > 0 && done >= total {
		return "00:00"
	}
	if total <= 0 || done <= 0 || elapsed < 3*time.Second || done < 16<<20 {
		return L("calculating", "wird berechnet", "wordt berekend", "calcul en cours", "calculando", "计算中", "вычисляется")
	}
	rate := float64(done) / elapsed.Seconds()
	if rate <= 0 {
		return "?"
	}
	remain := time.Duration(float64(total-done)/rate) * time.Second
	return durShort(remain)
}

func durShort(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	d = d.Round(time.Second)
	h := int(d / time.Hour)
	m := int((d % time.Hour) / time.Minute)
	s := int((d % time.Minute) / time.Second)
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func writeRawGame(w io.Writer, g PGNGame, toks []string, res string) error {
	canonical := []string{"Event", "Site", "Date", "Round", "White", "Black", "Result", "WhiteElo", "BlackElo", "TimeControl"}
	written := map[string]bool{}
	for _, k := range canonical {
		v := g.Tags[k]
		if k == "Result" {
			v = res
		}
		if v != "" {
			if _, err := fmt.Fprintf(w, "[%s \"%s\"]\n", k, escapeTag(v)); err != nil {
				return err
			}
			written[k] = true
		}
	}
	for _, k := range g.TagOrder {
		if written[k] || k == "FEN" || k == "SetUp" {
			continue
		}
		if v := g.Tags[k]; v != "" {
			if _, err := fmt.Fprintf(w, "[%s \"%s\"]\n", k, escapeTag(v)); err != nil {
				return err
			}
			written[k] = true
		}
	}
	if _, err := fmt.Fprintf(w, "[Source2MetalVersion \"%s\"]\n[Source2MetalSource \"%s\"]\n\n", version, escapeTag(g.Source)); err != nil {
		return err
	}
	for i, m := range toks {
		if i%2 == 0 {
			if _, err := fmt.Fprintf(w, "%d. ", i/2+1); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(w, m, " "); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, res); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w)
	return err
}

func escapeTag(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func fmtInt(n int64) string {
	s := fmt.Sprintf("%d", n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "." + s[i:]
	}
	return s
}

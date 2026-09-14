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
		parseErr := parsePGNFile(s.Path, func(g PGNGame) error {
			g.Source = sourceOriginalPath(s)
			jobs <- rawJob{seq: seq, game: g}
			seq++
			return nil
		}, func(done, total int64) { fp.bytesRead.Store(done) })
		close(jobs)
		wg.Wait()
		close(results)
		aggErr := <-aggDone
		close(stopMon)
		<-monDone
		printRawProgressFinal(fp, s.Bytes, cfg.Workers, startTime)
		flushErr := pw.Flush()
		closeErr := pf.Close()

		// A damaged GAME source must not abort the complete Source2Metal run.
		// Keep the successfully processed records from this source, report the
		// source as incomplete, and continue with the remaining source files.
		if parseErr != nil {
			fmt.Printf("  WARNING: source stopped early because of a PGN read/parse error: %v\n", parseErr)
			fmt.Printf("  Continuing with the remaining GAME sources.\n")
			stored := getSourceTrace(c, trace.Kind, trace.Base, trace.SourcePath)
			stored.RawPath = trace.RawPath
			stored.RawSeen += trace.RawSeen
			stored.RawAccepted += trace.RawAccepted
			stored.RawLocalDuplicate += trace.RawLocalDuplicate
			stored.RawCrossDuplicate += trace.RawCrossDuplicate
			stored.RawRejected += trace.RawRejected
			continue
		}
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
	if !wok || !bok || we < cfg.MinElo || be < cfg.MinElo {
		r.reject = rejectElo
		return r
	}
	if absInt(we-be) > cfg.MaxEloDiff {
		r.reject = rejectEloGap
		return r
	}
	r.sig = signatureGame(r.toks, r.result)
	return r
}

func signatureGame(toks []string, result string) gameSig {
	h1 := sha256.New()
	for _, t := range toks {
		io.WriteString(h1, t)
		h1.Write([]byte{0})
	}
	io.WriteString(h1, result)
	s := h1.Sum(nil)
	return gameSig{A: binary.LittleEndian.Uint64(s[:8]), B: binary.LittleEndian.Uint64(s[8:16])}
}

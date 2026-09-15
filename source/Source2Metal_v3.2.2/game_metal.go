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

	"source2metal/internal/ctgmetal"
)

// GAME -> METAL uses transposition-aware statistics and writes complete real
// collected per real board position/played move, but the learning records are
// complete genuine source games with their original result. No synthetic
// AnchorPly result is invented for GAME sources.
const (
	gmShardCount  = 128
	gmRecordSize  = 20
	gmMinGames    = int64(32)
	gmMinDecisive = int64(4)
	gmBestWindow  = 0.035
	gmMaxAnchors  = 2
	gmMinMatches  = 3
	gmMinCoverage = 0.80
)

type gmPositionKey struct{ A, B uint64 }
type gmEdgeKey struct {
	gmPositionKey
	Move uint16
}
type gmStats struct{ W, D, L int64 }

func (s gmStats) n() int64        { return s.W + s.D + s.L }
func (s gmStats) decisive() int64 { return s.W + s.L }
func (s gmStats) score() float64  { return (float64(s.W) + 0.5*float64(s.D) + 1) / (float64(s.n()) + 2) }
func (s gmStats) qualifies() bool { return s.n() >= gmMinGames && s.decisive() >= gmMinDecisive }

type gmSelected struct{ Stats gmStats }

type gmJob struct {
	seq  int64
	game PGNGame
}

type gmParsed struct {
	seq      int64
	tags     map[string]string
	tagOrder []string
	san      []string
	edges    []ctgmetal.PGNEdge
	result   string
	source   string
	sig      gameSig
	reject   rejectKind
	invalid  bool
	vars     int64
	comments int64
}

type gmProgress struct {
	bytesRead atomic.Int64
	processed atomic.Int64
	renderer  *progressRenderer
}

type gmAggCounts struct {
	Positions, Candidates, Qualified, AnchorPositions, AnchorSignals int64
}

type gmAggResult struct {
	Selected map[gmEdgeKey]gmSelected
	Counts   gmAggCounts
	Err      error
}

func gameMetalSources(inv Inventory, extra []Source) []Source {
	var out []Source
	for _, s := range inv.Sources {
		if s.Kind == KindPGN {
			out = append(out, s)
		}
	}
	out = append(out, extra...)
	sort.SliceStable(out, func(i, j int) bool {
		pi, pj := 0, 0
		if out[i].Origin != "" {
			pi = 1
		}
		if out[j].Origin != "" {
			pj = 1
		}
		if pi != pj {
			return pi < pj
		}
		return strings.ToLower(sourceOriginalPath(out[i])) < strings.ToLower(sourceOriginalPath(out[j]))
	})
	return out
}

func gmProcessJob(j gmJob, cfg Config) gmParsed {
	toks, res, vars, coms := stripMainline(j.game.MoveText)
	r := gmParsed{seq: j.seq, tags: j.game.Tags, tagOrder: j.game.TagOrder, result: res, source: j.game.Source, vars: vars, comments: coms}
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
	canonical, edges, ok := ctgmetal.ParsePGNMainline(toks, cfg.MaxPly)
	if !ok || len(canonical) == 0 {
		r.invalid = true
		return r
	}
	r.san, r.edges = canonical, edges
	key := strings.Join(canonical, "\x1f") + "|" + r.result + "|standard"
	h := sha256.Sum256([]byte(key))
	r.sig = gameSig{binary.LittleEndian.Uint64(h[0:8]), binary.LittleEndian.Uint64(h[8:16])}
	return r
}

// parseGameMetalSource legally parses one source with a worker pool while
// preserving source order for deterministic deduplication and output.
func parseGameMetalSource(s Source, cfg Config, phase string, onParsed func(gmParsed) error) error {
	fp := &gmProgress{renderer: newProgressRenderer()}
	start := time.Now()
	stop := make(chan struct{})
	doneMon := make(chan struct{})
	go func() {
		defer close(doneMon)
		t := time.NewTicker(750 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				gmShowProgress(fp, s.Bytes, phase, start)
			}
		}
	}()

	jobs := make(chan gmJob, maxInt(8, cfg.Workers*4))
	results := make(chan gmParsed, maxInt(8, cfg.Workers*4))
	var wg sync.WaitGroup
	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				results <- gmProcessJob(j, cfg)
			}
		}()
	}
	aggDone := make(chan error, 1)
	go func() {
		pending := make(map[int64]gmParsed, cfg.Workers*4)
		next := int64(0)
		var first error
		for r := range results {
			pending[r.seq] = r
			for {
				rr, ok := pending[next]
				if !ok {
					break
				}
				delete(pending, next)
				fp.processed.Add(1)
				if first == nil {
					if e := onParsed(rr); e != nil {
						first = e
					}
				}
				next++
			}
		}
		aggDone <- first
	}()

	seq := int64(0)
	parseErr := parsePGNFile(s.Path, func(g PGNGame) error {
		g.Source = sourceOriginalPath(s)
		jobs <- gmJob{seq: seq, game: g}
		seq++
		return nil
	}, func(done, total int64) { fp.bytesRead.Store(done) })
	close(jobs)
	wg.Wait()
	close(results)
	aggErr := <-aggDone
	close(stop)
	<-doneMon
	fp.bytesRead.Store(s.Bytes)
	gmShowProgress(fp, s.Bytes, phase, start)
	fp.renderer.Clear()
	if parseErr != nil {
		return parseErr
	}
	return aggErr
}

func gmShowProgress(fp *gmProgress, total int64, phase string, start time.Time) {
	done := fp.bytesRead.Load()
	processed := fp.processed.Load()
	pct := 0.0
	if total > 0 {
		pct = 100 * float64(done) / float64(total)
		if pct > 100 {
			pct = 100
		}
	}
	eta := etaString(done, total, time.Since(start))
	fp.renderer.Render(func(width int) string {
		if width >= 82 {
			return fmt.Sprintf("  %s %5.1f%% | P%s | ETA%s", phase, pct, fmtCompactInt(processed), eta)
		}
		return fmt.Sprintf("  %s %5.1f%% | P%s", phase, pct, fmtCompactInt(processed))
	})
}

type gmShardWriter struct {
	file *os.File
	buf  *bufio.Writer
}

func gmOpenShards(dir string) ([]gmShardWriter, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	a := make([]gmShardWriter, gmShardCount)
	for i := range a {
		f, err := os.Create(filepath.Join(dir, fmt.Sprintf("segment_%03d.bin", i)))
		if err != nil {
			gmCloseShards(a)
			return nil, err
		}
		a[i] = gmShardWriter{file: f, buf: bufio.NewWriterSize(f, 1<<20)}
	}
	return a, nil
}
func gmCloseShards(a []gmShardWriter) error {
	var first error
	for i := range a {
		if a[i].buf != nil {
			if e := a[i].buf.Flush(); e != nil && first == nil {
				first = e
			}
		}
		if a[i].file != nil {
			if e := a[i].file.Close(); e != nil && first == nil {
				first = e
			}
		}
	}
	return first
}
func gmWriteObservation(w *bufio.Writer, e ctgmetal.PGNEdge, outcome byte) error {
	var b [gmRecordSize]byte
	binary.LittleEndian.PutUint64(b[0:8], e.A)
	binary.LittleEndian.PutUint64(b[8:16], e.B)
	binary.LittleEndian.PutUint16(b[16:18], e.Move)
	b[18] = outcome
	_, err := w.Write(b[:])
	return err
}
func gmReadObservation(r io.Reader) (gmEdgeKey, byte, error) {
	var b [gmRecordSize]byte
	_, err := io.ReadFull(r, b[:])
	if err != nil {
		return gmEdgeKey{}, 0, err
	}
	return gmEdgeKey{gmPositionKey{binary.LittleEndian.Uint64(b[:8]), binary.LittleEndian.Uint64(b[8:16])}, binary.LittleEndian.Uint16(b[16:18])}, b[18], nil
}
func gmOutcome(result string, whiteToMove bool) byte {
	if result == "1/2-1/2" {
		return 2
	}
	if (result == "1-0" && whiteToMove) || (result == "0-1" && !whiteToMove) {
		return 1
	}
	return 3
}

func gmAggregateShard(path string) (map[gmEdgeKey]gmSelected, gmAggCounts, error) {
	selected := make(map[gmEdgeKey]gmSelected)
	var cnt gmAggCounts
	f, err := os.Open(path)
	if err != nil {
		return selected, cnt, err
	}
	defer f.Close()
	m := make(map[gmEdgeKey]gmStats)
	for {
		k, o, e := gmReadObservation(f)
		if e == io.EOF || e == io.ErrUnexpectedEOF {
			break
		}
		if e != nil {
			return selected, cnt, e
		}
		s := m[k]
		switch o {
		case 1:
			s.W++
		case 2:
			s.D++
		case 3:
			s.L++
		}
		m[k] = s
	}
	keys := make([]gmEdgeKey, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].A != keys[j].A {
			return keys[i].A < keys[j].A
		}
		if keys[i].B != keys[j].B {
			return keys[i].B < keys[j].B
		}
		return keys[i].Move < keys[j].Move
	})
	for i := 0; i < len(keys); {
		j := i + 1
		for j < len(keys) && keys[j].gmPositionKey == keys[i].gmPositionKey {
			j++
		}
		cnt.Positions++
		cnt.Candidates += int64(j - i)
		best := -1.0
		q := make([]gmEdgeKey, 0, j-i)
		for x := i; x < j; x++ {
			s := m[keys[x]]
			if s.qualifies() {
				cnt.Qualified++
				q = append(q, keys[x])
				if s.score() > best {
					best = s.score()
				}
			}
		}
		a := make([]gmEdgeKey, 0, len(q))
		for _, k := range q {
			if m[k].score() >= best-gmBestWindow {
				a = append(a, k)
			}
		}
		sort.Slice(a, func(x, y int) bool {
			sx, sy := m[a[x]], m[a[y]]
			if sx.score() == sy.score() {
				if sx.n() == sy.n() {
					return a[x].Move < a[y].Move
				}
				return sx.n() > sy.n()
			}
			return sx.score() > sy.score()
		})
		if len(a) > gmMaxAnchors {
			a = a[:gmMaxAnchors]
		}
		if len(a) > 0 {
			cnt.AnchorPositions++
		}
		for _, k := range a {
			selected[k] = gmSelected{Stats: m[k]}
			cnt.AnchorSignals++
		}
		i = j
	}
	return selected, cnt, nil
}

func gmAggregateAll(dir string, workers int) (map[gmEdgeKey]gmSelected, gmAggCounts, error) {
	if workers < 1 {
		workers = 1
	}
	if workers > gmShardCount {
		workers = gmShardCount
	}
	jobs := make(chan int, workers)
	results := make(chan gmAggResult, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for shard := range jobs {
				s, c, e := gmAggregateShard(filepath.Join(dir, fmt.Sprintf("segment_%03d.bin", shard)))
				results <- gmAggResult{Selected: s, Counts: c, Err: e}
			}
		}()
	}
	go func() {
		for i := 0; i < gmShardCount; i++ {
			jobs <- i
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()
	all := make(map[gmEdgeKey]gmSelected)
	var cnt gmAggCounts
	var first error
	done := 0
	pr := newProgressRenderer()
	for r := range results {
		done++
		if r.Err != nil && first == nil {
			first = r.Err
		}
		cnt.Positions += r.Counts.Positions
		cnt.Candidates += r.Counts.Candidates
		cnt.Qualified += r.Counts.Qualified
		cnt.AnchorPositions += r.Counts.AnchorPositions
		cnt.AnchorSignals += r.Counts.AnchorSignals
		for k, v := range r.Selected {
			all[k] = v
		}
		pct := 100 * float64(done) / gmShardCount
		pr.Render(func(width int) string {
			return fmt.Sprintf(L("  GAME-METAL selection %5.1f%% | segments %d/%d", "  GAME-METAL Auswahl %5.1f%% | Segmente %d/%d", "  GAME-METAL selectie %5.1f%% | segmenten %d/%d", "  Sélection GAME-METAL %5.1f%% | segments %d/%d", "  Selección GAME-METAL %5.1f%% | segmentos %d/%d", "  GAME-METAL 选择 %5.1f%% | 分段 %d/%d", "  Отбор GAME-METAL %5.1f%% | сегменты %d/%d"), pct, done, gmShardCount)
		})
	}
	pr.Clear()
	return all, cnt, first
}

func gmSelectedByPosition(selected map[gmEdgeKey]gmSelected) map[gmPositionKey]map[uint16]gmSelected {
	out := make(map[gmPositionKey]map[uint16]gmSelected)
	for k, s := range selected {
		m := out[k.gmPositionKey]
		if m == nil {
			m = make(map[uint16]gmSelected)
			out[k.gmPositionKey] = m
		}
		m[k.Move] = s
	}
	return out
}
func gmCoverage(edges []ctgmetal.PGNEdge, selected map[gmPositionKey]map[uint16]gmSelected) (matched, covered int) {
	for _, e := range edges {
		if choices := selected[gmPositionKey{e.A, e.B}]; len(choices) > 0 {
			covered++
			if _, ok := choices[e.Move]; ok {
				matched++
			}
		}
	}
	return
}

func gmEmitRealGame(w *bufio.Writer, g gmParsed, sourceKind SourceKind, matched, covered int) error {
	date := strings.TrimSpace(g.tags["Date"])
	if date == "" {
		date = "????.??.??"
	}
	round := strings.TrimSpace(g.tags["Round"])
	if round == "" {
		round = "-"
	}
	ow := strings.TrimSpace(g.tags["White"])
	if ow == "" {
		ow = "?"
	}
	ob := strings.TrimSpace(g.tags["Black"])
	if ob == "" {
		ob = "?"
	}
	if _, err := fmt.Fprintf(w,
		"[Event \"Source2Metal v%s - echte GAME modelpartij\"]\n"+
			"[Site \"Local Source\"]\n[Date \"%s\"]\n[Round \"%s\"]\n"+
			"[White \"Metal\"]\n[Black \"Metal\"]\n[Result \"%s\"]\n"+
			"[SourceFormat \"%s\"]\n[SourcePath \"%s\"]\n"+
			"[OriginalWhite \"%s\"]\n[OriginalBlack \"%s\"]\n"+
			"[OriginalWhiteElo \"%s\"]\n[OriginalBlackElo \"%s\"]\n"+
			"[OriginalResult \"%s\"]\n[SelectionMode \"Gebalanceerd - Source2Metal\"]\n"+
			"[PreferenceCoverage \"%d/%d\"]\n\n",
		version, escapeTag(date), escapeTag(round), g.result, sourceKind, escapeTag(g.source), escapeTag(ow), escapeTag(ob), escapeTag(g.tags["WhiteElo"]), escapeTag(g.tags["BlackElo"]), g.result, matched, covered); err != nil {
		return err
	}
	for i, m := range g.san {
		if i%2 == 0 {
			if _, err := fmt.Fprintf(w, "%d. ", i/2+1); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(w, m, " "); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, "%s\n\n", g.result)
	return err
}

func gmVerifyRealMetal(path string, expected int64) error {
	found, err := countGeneratedPGNRecords(path)
	if err != nil {
		return err
	}
	if found != expected {
		return fmt.Errorf("verwacht %s records, gevonden %s", fmtInt(expected), fmtInt(found))
	}
	var bad int64
	err = parsePGNFile(path, func(g PGNGame) error {
		toks, res, vars, _ := stripMainline(g.MoveText)
		if vars != 0 {
			bad++
			return nil
		}
		if g.Tags["OriginalResult"] == "" || g.Tags["OriginalResult"] != res {
			bad++
			return nil
		}
		if g.Tags["AnchorPly"] != "" || g.Tags["Weight"] != "" {
			bad++
			return nil
		}
		if _, _, ok := ctgmetal.ParsePGNMainline(toks, maxAllowedPly); !ok {
			bad++
		}
		return nil
	}, nil)
	if err != nil {
		return err
	}
	if bad > 0 {
		return fmt.Errorf("%s records faalden de echte-partijcontrole", fmtInt(bad))
	}
	return nil
}

// buildMetalFromGames creates one global statistical GAME model and writes the
// actual per-source contributions plus their exact merged result.
func buildMetalFromGames(inv Inventory, extra []Source, layout OutputLayout, cfg Config, c *Counters) error {
	sources := gameMetalSources(inv, extra)
	if len(sources) == 0 {
		fmt.Println(L("No GAME sources for GAME-METAL; skipped.", "Keine GAME-Quellen für GAME-METAL; übersprungen.", "Geen GAME-bronnen voor GAME-METAL; overgeslagen.", "Aucune source GAME pour GAME-METAL ; ignoré.", "No hay fuentes GAME para GAME-METAL; omitido.", "没有用于 GAME-METAL 的 GAME 源；已跳过。", "Нет GAME-источников для GAME-METAL; пропущено."))
		return nil
	}
	if err := os.MkdirAll(layout.MetalSeparateDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(layout.MetalMergedDir, 0755); err != nil {
		return err
	}
	temp := filepath.Join(layout.TempDir, "GAME_METAL")
	segDir := filepath.Join(temp, "segments")
	_ = os.RemoveAll(temp)
	shards, err := gmOpenShards(segDir)
	if err != nil {
		return err
	}
	seen := make(map[gameSig]struct{}, 1<<20)
	fmt.Println("\n" + L("GAME-METAL 1/3: legally analyze real GAME sources and collect statistics...", "GAME-METAL 1/3: echte GAME-Quellen legal analysieren und Statistik sammeln...", "GAME-METAL 1/3: echte GAME-bronnen legaal analyseren en statistiek verzamelen...", "GAME-METAL 1/3 : analyser légalement les vraies sources GAME et collecter les statistiques...", "GAME-METAL 1/3: analizar legalmente fuentes GAME reales y recopilar estadísticas...", "GAME-METAL 1/3：对真实 GAME 源进行合法性分析并收集统计...", "GAME-METAL 1/3: легальный анализ реальных GAME-источников и сбор статистики..."))
	for i, s := range sources {
		fmt.Printf(L("  Source %d/%d: %s [%s]\n", "  Quelle %d/%d: %s [%s]\n", "  Bron %d/%d: %s [%s]\n", "  Source %d/%d : %s [%s]\n", "  Fuente %d/%d: %s [%s]\n", "  源 %d/%d：%s [%s]\n", "  Источник %d/%d: %s [%s]\n"), i+1, len(sources), sourceOriginalPath(s), sourceOriginalKind(s))
		tr := getSourceTrace(c, sourceOriginalKind(s), s.Base, sourceOriginalPath(s))
		err = parseGameMetalSource(s, cfg, L("analysis", "Analyse", "analyse", "analyse", "análisis", "分析", "анализ"), func(g gmParsed) error {
			c.GameMetalGamesSeen++
			tr.MetalSeen++
			if g.reject != rejectNone {
				c.GameMetalRejected++
				return nil
			}
			if g.invalid {
				c.GameMetalInvalid++
				tr.MetalInvalid++
				return nil
			}
			if _, ok := seen[g.sig]; ok {
				c.GameMetalDuplicates++
				return nil
			}
			seen[g.sig] = struct{}{}
			c.GameMetalGamesAccepted++
			local := make(map[gmEdgeKey]struct{}, len(g.edges))
			for _, e := range g.edges {
				k := gmEdgeKey{gmPositionKey{e.A, e.B}, e.Move}
				if _, dup := local[k]; dup {
					continue
				}
				local[k] = struct{}{}
				if err := gmWriteObservation(shards[int(e.A%gmShardCount)].buf, e, gmOutcome(g.result, e.WhiteToMove)); err != nil {
					return err
				}
				c.GameMetalObservations++
			}
			return nil
		})
		if err != nil {
			gmCloseShards(shards)
			return err
		}
	}
	if err := gmCloseShards(shards); err != nil {
		return err
	}
	fmt.Printf(L("  Analysis done: %s unique legal games | %s observations | %s invalid games | %s duplicates.\n", "  Analyse fertig: %s eindeutige legale Partien | %s Beobachtungen | %s ungültige Partien | %s Duplikate.\n", "  Analyse klaar: %s unieke legale partijen | %s waarnemingen | %s ongeldige partijen | %s duplicaten.\n", "  Analyse terminée : %s parties légales uniques | %s observations | %s parties invalides | %s doublons.\n", "  Análisis listo: %s partidas legales únicas | %s observaciones | %s partidas no válidas | %s duplicados.\n", "  分析完成：%s 个唯一合法对局 | %s 次观察 | %s 个无效对局 | %s 个重复项。\n", "  Анализ завершён: %s уникальных легальных партий | %s наблюдений | %s недопустимых партий | %s дубликатов.\n"), fmtInt(c.GameMetalGamesAccepted), fmtInt(c.GameMetalObservations), fmtInt(c.GameMetalInvalid), fmtInt(c.GameMetalDuplicates))
	if c.GameMetalGamesAccepted == 0 {
		c.GameMetalSkipped++
		return nil
	}

	fmt.Println(L("GAME-METAL 2/3: select transposition-aware move preferences (Balanced)...", "GAME-METAL 2/3: transpositionsbewusste Zugpräferenzen auswählen (Ausgewogen)...", "GAME-METAL 2/3: transpositie-bewuste zetvoorkeuren selecteren (Gebalanceerd)...", "GAME-METAL 2/3 : sélectionner les préférences de coups en tenant compte des transpositions (Équilibré)...", "GAME-METAL 2/3: seleccionar preferencias de jugadas teniendo en cuenta transposiciones (Equilibrado)...", "GAME-METAL 2/3：选择转置感知的着法偏好（平衡）...", "GAME-METAL 2/3: выбор предпочтений ходов с учётом транспозиций (Сбалансированный)..."))
	selected, ac, err := gmAggregateAll(segDir, cfg.Workers)
	if err != nil {
		return err
	}
	c.GameMetalPositions = ac.Positions
	c.GameMetalCandidates = ac.Candidates
	c.GameMetalQualified = ac.Qualified
	c.GameMetalAnchorPositions = ac.AnchorPositions
	c.GameMetalAnchorSignals = ac.AnchorSignals
	fmt.Printf(L("  Selection done: %s positions | %s candidates | %s qualified | %s preference positions | %s preferred moves.\n", "  Auswahl fertig: %s Positionen | %s Kandidaten | %s qualifiziert | %s Präferenzpositionen | %s bevorzugte Züge.\n", "  Selectie klaar: %s posities | %s kandidaten | %s gekwalificeerd | %s voorkeurposities | %s voorkeurszetten.\n", "  Sélection terminée : %s positions | %s candidats | %s qualifiés | %s positions préférées | %s coups préférés.\n", "  Selección lista: %s posiciones | %s candidatos | %s calificados | %s posiciones preferidas | %s jugadas preferidas.\n", "  选择完成：%s 个局面 | %s 个候选 | %s 个符合条件 | %s 个偏好局面 | %s 个偏好着法。\n", "  Отбор завершён: %s позиций | %s кандидатов | %s квалифицировано | %s позиций предпочтения | %s предпочтительных ходов.\n"), fmtInt(ac.Positions), fmtInt(ac.Candidates), fmtInt(ac.Qualified), fmtInt(ac.AnchorPositions), fmtInt(ac.AnchorSignals))
	if len(selected) == 0 {
		c.GameMetalSkipped++
		msg := L(
			"The GAME sources were read legally, but no move reached the Balanced evidence threshold (at least 32 observations and 4 decisive results). No empty or weakly supported Metal.pgn was created.\n",
			"Die GAME-Quellen wurden legal gelesen, aber kein Zug erreichte die ausgewogene Evidenzgrenze (mindestens 32 Beobachtungen und 4 entschiedene Ergebnisse). Es wurde keine leere oder schwach belegte Metal.pgn erzeugt.\n",
			"De GAME-bronnen zijn legaal gelezen, maar geen zet bereikte de Gebalanceerde bewijsgrens (minimaal 32 waarnemingen en 4 beslissende resultaten). Er is bewust geen lege of zwak onderbouwde Metal.pgn gemaakt.\n",
			"Les sources GAME ont été lues légalement, mais aucun coup n’a atteint le seuil de preuve Équilibré (au moins 32 observations et 4 résultats décisifs). Aucun Metal.pgn vide ou faiblement étayé n’a été créé.\n",
			"Las fuentes GAME se leyeron legalmente, pero ninguna jugada alcanzó el umbral de evidencia Equilibrado (al menos 32 observaciones y 4 resultados decisivos). No se creó un Metal.pgn vacío o débilmente respaldado.\n",
			"GAME 源已合法读取，但没有任何着法达到“平衡”证据阈值（至少 32 次观察和 4 个决定性结果）。因此没有创建空白或证据不足的 Metal.pgn。\n",
			"GAME-источники были прочитаны корректно, но ни один ход не достиг порога профиля «Сбалансированный» (минимум 32 наблюдения и 4 решающих результата). Пустой или слабо обоснованный Metal.pgn не создавался.\n")
		_ = atomicWriteFile(filepath.Join(layout.MetalMergedDir, "GAME-METAL - NO OUTPUT.txt"), []byte(msg), 0644)
		return nil
	}
	byPos := gmSelectedByPosition(selected)

	fmt.Println(L("GAME-METAL 3/3: write real model games per source and merged...", "GAME-METAL 3/3: echte Modellpartien pro Quelle und zusammengeführt schreiben...", "GAME-METAL 3/3: echte modelpartijen per bron en samengevoegd schrijven...", "GAME-METAL 3/3 : écrire les vraies parties modèles par source et fusionnées...", "GAME-METAL 3/3: escribir partidas modelo reales por fuente y combinadas...", "GAME-METAL 3/3：按源写出真实模型对局并合并...", "GAME-METAL 3/3: запись реальных модельных партий по источникам и объединённо..."))
	merged, err := os.Create(layout.GameMetalFile)
	if err != nil {
		return err
	}
	mw := bufio.NewWriterSize(merged, 4<<20)
	seenOut := make(map[gameSig]struct{}, len(seen))
	for i, s := range sources {
		kind := sourceOriginalKind(s)
		sourcePath := sourceOriginalPath(s)
		tr := getSourceTrace(c, kind, s.Base, sourcePath)
		perPath := uniqueSourceFile(layout.MetalSeparateDir, s.Base, kind, "METAL", sourcePath)
		pf, e := os.Create(perPath)
		if e != nil {
			_ = merged.Close()
			return e
		}
		pw := bufio.NewWriterSize(pf, 1<<20)
		var n int64
		fmt.Printf(L("  Source %d/%d: %s [%s]\n", "  Quelle %d/%d: %s [%s]\n", "  Bron %d/%d: %s [%s]\n", "  Source %d/%d : %s [%s]\n", "  Fuente %d/%d: %s [%s]\n", "  源 %d/%d：%s [%s]\n", "  Источник %d/%d: %s [%s]\n"), i+1, len(sources), sourcePath, kind)
		err = parseGameMetalSource(s, cfg, "model", func(g gmParsed) error {
			if g.reject != rejectNone || g.invalid {
				return nil
			}
			if _, ok := seenOut[g.sig]; ok {
				return nil
			}
			seenOut[g.sig] = struct{}{}
			matched, covered := gmCoverage(g.edges, byPos)
			if covered == 0 {
				return nil
			}
			c.GameMetalModelCandidates++
			tr.MetalEligible++
			coverage := float64(matched) / float64(covered)
			if matched < gmMinMatches || coverage+1e-12 < gmMinCoverage {
				return nil
			}
			if err := gmEmitRealGame(pw, g, kind, matched, covered); err != nil {
				return err
			}
			if err := gmEmitRealGame(mw, g, kind, matched, covered); err != nil {
				return err
			}
			n++
			c.GameMetalSelected++
			tr.MetalSelected++
			return nil
		})
		if err != nil {
			_ = pf.Close()
			_ = merged.Close()
			return err
		}
		if err := pw.Flush(); err != nil {
			_ = pf.Close()
			_ = merged.Close()
			return err
		}
		if err := pf.Close(); err != nil {
			_ = merged.Close()
			return err
		}
		if n == 0 {
			_ = os.Remove(perPath)
			note := strings.TrimSuffix(perPath, ".pgn") + " - NO OUTPUT.txt"
			_ = atomicWriteFile(note, []byte(fmt.Sprintf("Bron: %s\nFormaat: %s\n\nDeze bron leverde binnen het gezamenlijke GAME-model geen modelpartij die minimaal %d voorkeursposities trof en minstens %.0f%% dekking haalde. De bron kan wel degelijk aan de gezamenlijke zetstatistiek hebben bijgedragen.\n", sourcePath, kind, gmMinMatches, gmMinCoverage*100)), 0644)
			tr.MetalPath = note
			fmt.Println("    "+L("No per-source GAME-METAL records; explanation:", "Keine GAME-METAL-Datensätze pro Quelle; Erklärung:", "Geen per-bron GAME-METAL-records; uitleg:", "Aucun enregistrement GAME-METAL par source ; explication :", "No hay registros GAME-METAL por fuente; explicación:", "没有每源 GAME-METAL 记录；说明：", "Нет записей GAME-METAL по источнику; пояснение:"), note)
			continue
		}
		if err := gmVerifyRealMetal(perPath, n); err != nil {
			_ = merged.Close()
			return fmt.Errorf("GAME-METAL per-bron controle %s: %w", s.Base, err)
		}
		tr.MetalPath = perPath
		fmt.Printf(L("    Per-source contribution: %s real model games | check OK\n", "    Beitrag pro Quelle: %s echte Modellpartien | Prüfung OK\n", "    Per-bron bijdrage: %s echte modelpartijen | controle OK\n", "    Contribution par source : %s parties modèles réelles | contrôle OK\n", "    Contribución por fuente: %s partidas modelo reales | comprobación OK\n", "    每源贡献：%s 个真实模型对局 | 检查通过\n", "    Вклад источника: %s реальных модельных партий | проверка OK\n"), fmtInt(n))
	}
	if err := mw.Flush(); err != nil {
		_ = merged.Close()
		return err
	}
	if err := merged.Close(); err != nil {
		return err
	}
	if c.GameMetalSelected == 0 {
		_ = os.Remove(layout.GameMetalFile)
		c.GameMetalSkipped++
		return nil
	}
	if err := gmVerifyRealMetal(layout.GameMetalFile, c.GameMetalSelected); err != nil {
		c.GameMetalFailed++
		return fmt.Errorf("samengevoegde GAME-METAL controle: %w", err)
	}
	st, _ := os.Stat(layout.GameMetalFile)
	if st != nil {
		c.GameMetalBytes = st.Size()
	}
	if h, e := fileSHA256(layout.GameMetalFile); e == nil {
		c.GameMetalSHA256 = h
	}
	c.GameMetalVerified = c.GameMetalSelected
	c.GameMetalBuilt = 1
	fmt.Printf(L("  Merged GAME-METAL check: OK - %s real model games from %d GAME sources.\n", "  Prüfung zusammengeführtes GAME-METAL: OK - %s echte Modellpartien aus %d GAME-Quellen.\n", "  Samengevoegde GAME-METAL controle: OK - %s echte modelpartijen uit %d GAME-bronnen.\n", "  Contrôle GAME-METAL fusionné : OK - %s parties modèles réelles provenant de %d sources GAME.\n", "  Comprobación GAME-METAL combinado: OK - %s partidas modelo reales de %d fuentes GAME.\n", "  合并 GAME-METAL 检查：OK - %s 个真实模型对局，来自 %d 个 GAME 源。\n", "  Проверка объединённого GAME-METAL: OK - %s реальных модельных партий из %d GAME-источников.\n"), fmtInt(c.GameMetalSelected), len(sources))
	fmt.Println("  "+L("Automatically merged to:", "Automatisch zusammengeführt nach:", "Automatisch samengevoegd naar:", "Fusionné automatiquement vers :", "Combinado automáticamente en:", "自动合并到：", "Автоматически объединено в:"), layout.GameMetalFile)
	return nil
}

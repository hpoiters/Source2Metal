package ctgraw

// Deterministic breadth-first coverage of the reachable position graph.
// Original implementation for Source2Metal, 2026-09-22; GPL-3.0.
import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"
)

type GraphReport struct {
	Workers                int
	MaxPly                 int
	Positions              int
	BookPositions          int
	RawMoveRecords         int
	Edges                  int
	Transpositions         int
	DuplicateMoves         int
	MissingPositions       int
	DepthCutPositions      int
	DepthCutMoves          int
	DecodeErrors           int
	PromotionAssumptions   int
	PawnlessPositions      int
	ExportedRecords        int
	ExportedEdges          int
	Seconds                float64
	CoverageOfDecodedEdges bool
	CompleteCTG            bool
	Limitations            []string
}
type node struct {
	board         Board
	parent, depth int
	via           Move
	viaSAN        string
	edges         []edge
}
type edge struct {
	move   Move
	target int
	san    string
}
type decoded struct {
	entry  Entry
	moves  []Move
	codes  []byte
	sans   map[Move]string
	errors []string
	err    error
}

// Unlike CTG's normalized disk key, this key never merges differently oriented
// boards. Remove only an EP square for which no legal EP capture exists.
func positionKey(b Board) Board {
	if b.EP >= 0 {
		effective := false
		for _, m := range b.LegalMoves() {
			if m.EP {
				effective = true
				break
			}
		}
		if !effective {
			b.EP = -1
		}
	}
	return b
}
func decodeNode(r *CTGReader, b Board) decoded {
	d := decoded{sans: map[Move]string{}}
	d.entry, d.err = r.Lookup(b)
	if d.err != nil {
		return d
	}
	for _, raw := range d.entry.Moves {
		m, e := r.DecodeMove(b, raw.Code)
		if e != nil {
			d.errors = append(d.errors, fmt.Sprintf("code=%d annotation=%d: %v", raw.Code, raw.Ann, e))
			continue
		}
		d.moves = append(d.moves, m)
		d.sans[m] = b.SAN(m)
		d.codes = append(d.codes, raw.Code)
	}
	return d
}
func missingPosition(err error) bool { return errors.Is(err, errPositionMissing) }

func BuildGraph(ctgPath, ctoPath, ctbPath, outPath string, maxPly, workers int, progress ProgressFunc) (BuildResult, error) {
	start := time.Now()
	var result BuildResult
	rep := GraphReport{Workers: workers, MaxPly: maxPly, Limitations: []string{
		"Coverage certifies decoded reachable position/move pairs, not all possible paths or every physical CTG record.",
		"Underpromotion encoding and pawnless CTG symmetry are not independently verified; no 100% completeness claim.",
		"A missing position is an index lookup miss; without an independent physical-record decoder it cannot prove absence from the CTG.",
		"Statistics are stored separately as raw fields with the legacy interpretation; no weights or outcomes are transferred to PGN.",
	}}
	if maxPly < 0 {
		return result, fmt.Errorf("negative CTG depth")
	}
	if workers <= 0 {
		workers = (runtime.NumCPU() + 1) / 2
	}
	if workers < 1 {
		workers = 1
	}
	rep.Workers = workers
	readers := make([]*CTGReader, 0, workers)
	for i := 0; i < workers; i++ {
		r, e := openCTG(ctgPath, ctoPath, ctbPath)
		if e != nil {
			for _, x := range readers {
				x.Close()
			}
			return result, e
		}
		readers = append(readers, r)
	}
	defer func() {
		for _, r := range readers {
			r.Close()
		}
	}()
	stats, e := os.Create(outPath + ".ctg-statistics.jsonl")
	if e != nil {
		return result, e
	}
	defer stats.Close()
	enc := json.NewEncoder(stats)
	nodes := []node{{board: startBoard(), parent: -1}}
	known := map[Board]int{positionKey(startBoard()): 0}
	// Each worker owns its reader caches. Only the coordinator mutates the graph.
	type job struct {
		id int
		b  Board
	}
	type answer struct {
		id int
		d  decoded
	}
	jobs := make(chan job, workers*2)
	answers := make(chan answer, workers*2)
	var wg sync.WaitGroup
	for _, r := range readers {
		wg.Add(1)
		go func(r *CTGReader) {
			defer wg.Done()
			for j := range jobs {
				answers <- answer{j.id, decodeNode(r, j.b)}
			}
		}(r)
	}
	defer func() { close(jobs); wg.Wait() }()
	last := time.Now()
	// Fixed-size frontier batches keep memory and scheduling overhead bounded.
	for begin := 0; begin < len(nodes); {
		end := begin + 256
		if end > len(nodes) {
			end = len(nodes)
		}
		batch := make([]decoded, end-begin)
		sent, received := 0, 0
		for received < len(batch) {
			var send chan job
			var j job
			if sent < len(batch) {
				send = jobs
				j = job{begin + sent, nodes[begin+sent].board}
			}
			select {
			case send <- j:
				sent++
			case a := <-answers:
				batch[a.id-begin] = a.d
				received++
			}
		}
		for offset, d := range batch {
			id := begin + offset
			b := nodes[id].board
			if d.err != nil {
				if !missingPosition(d.err) {
					return result, fmt.Errorf("position %d: %w", id, d.err)
				}
				if id == 0 {
					return result, fmt.Errorf("beginstelling niet gevonden: %w", d.err)
				}
				rep.MissingPositions++
				continue
			}
			rep.BookPositions++
			rep.RawMoveRecords += len(d.entry.Moves)
			rep.DecodeErrors += len(d.errors)
			pawns := false
			for _, p := range b.S {
				if p == 'P' || p == 'p' {
					pawns = true
				}
			}
			if !pawns {
				rep.PawnlessPositions++
			}
			for _, m := range d.moves {
				if m.Promo != 0 {
					rep.PromotionAssumptions++
				}
			}
			if e = enc.Encode(struct {
				Position     int
				Board        Board
				Entry        Entry
				DecodeErrors []string
			}{id, b, d.entry, d.errors}); e != nil {
				return result, e
			}
			if maxPly > 0 && nodes[id].depth >= maxPly {
				if len(d.moves) > 0 {
					rep.DepthCutPositions++
					rep.DepthCutMoves += len(d.moves)
				}
				continue
			}
			sort.Slice(d.moves, func(i, j int) bool { return coord(d.moves[i]) < coord(d.moves[j]) })
			seen := map[Move]bool{}
			for _, m := range d.moves {
				if seen[m] {
					rep.DuplicateMoves++
					continue
				}
				seen[m] = true
				child := b
				child.Apply(m)
				key := positionKey(child)
				target, ok := known[key]
				if ok {
					rep.Transpositions++
				} else {
					target = len(nodes)
					known[key] = target
					nodes = append(nodes, node{board: child, parent: id, depth: nodes[id].depth + 1, via: m, viaSAN: d.sans[m]})
				}
				nodes[id].edges = append(nodes[id].edges, edge{m, target, d.sans[m]})
				rep.Edges++
			}
		}
		begin = end
		if progress != nil && time.Since(last) > 500*time.Millisecond {
			progress(fmt.Sprintf("CTG graph: %d positions | %d moves | %d workers | %.0f pos/s", begin, rep.Edges, workers, float64(begin)/time.Since(start).Seconds()), false)
			last = time.Now()
		}
	}
	if e = stats.Close(); e != nil {
		return result, e
	}
	rep.Positions = len(nodes)
	// A BFS parent tree supplies one legal shortest route to every position.
	// Emit every non-tree edge and tree leaf. Every tree edge then lies on at
	// least one emitted route. Cycles are represented once, never expanded.
	f, e := os.Create(outPath)
	if e != nil {
		return result, e
	}
	defer f.Close()
	w := bufio.NewWriterSize(f, 1<<20)
	covered := make([][]bool, len(nodes))
	for i := range nodes {
		covered[i] = make([]bool, len(nodes[i].edges))
	}
	for id, n := range nodes {
		for k, ed := range n.edges {
			child := nodes[ed.target]
			tree := child.parent == id && child.via == ed.move
			if tree && len(child.edges) > 0 {
				continue
			}
			ids := []int{}
			for p := id; p > 0; p = nodes[p].parent {
				ids = append(ids, p)
			}
			b := startBoard()
			sans := make([]string, 0, len(ids)+1)
			for j := len(ids) - 1; j >= 0; j-- {
				v := nodes[ids[j]]
				sans = append(sans, v.viaSAN)
				b.Apply(v.via)
				for q, pe := range nodes[v.parent].edges {
					if pe.target == ids[j] && pe.move == v.via {
						covered[v.parent][q] = true
						break
					}
				}
			}
			if positionKey(b) != positionKey(n.board) {
				return result, fmt.Errorf("export parent path mismatch at %d", id)
			}
			sans = append(sans, ed.san)
			covered[id][k] = true
			if _, e = fmt.Fprintf(w, "[Event \"Reconstructed CTG book line\"]\n[Site \"?\"]\n[Date \"????.??.??\"]\n[Round \"?\"]\n[White \"Neutral\"]\n[Black \"Neutral\"]\n[Result \"*\"]\n\n%s\n\n", formatMoves(sans, "*")); e != nil {
				return result, e
			}
			result.Games++
			if progress != nil && time.Since(last) > 500*time.Millisecond {
				progress(fmt.Sprintf("CTG PGN: %d records | %d plies | %.0f records/s", result.Games, result.Plies, float64(result.Games)/time.Since(start).Seconds()), false)
				last = time.Now()
			}
			result.Plies += int64(len(sans))
		}
	}
	if e = w.Flush(); e != nil {
		return result, e
	}
	if e = f.Close(); e != nil {
		return result, e
	}
	for _, row := range covered {
		for _, c := range row {
			if c {
				rep.ExportedEdges++
			}
		}
	}
	rep.ExportedRecords = result.Games
	rep.CoverageOfDecodedEdges = rep.ExportedEdges == rep.Edges
	if !rep.CoverageOfDecodedEdges {
		return result, fmt.Errorf("CTG edge coverage failed: %d/%d", rep.ExportedEdges, rep.Edges)
	}
	rep.Seconds = time.Since(start).Seconds()
	data, e := json.MarshalIndent(rep, "", "  ")
	if e != nil {
		return result, e
	}
	if e = os.WriteFile(outPath+".ctg-report.json", data, 0644); e != nil {
		return result, e
	}
	info, e := os.Stat(outPath)
	if e != nil {
		return result, e
	}
	result.Bytes = info.Size()
	result.DecodeErrors = int64(rep.DecodeErrors)
	result.SHA256, e = fileSHA256(outPath)
	if progress != nil {
		progress(fmt.Sprintf("CTG graph: %d positions | %d/%d moves exported | %d records | limitations: see CTG report", rep.Positions, rep.ExportedEdges, rep.Edges, result.Games), true)
	}
	return result, e
}

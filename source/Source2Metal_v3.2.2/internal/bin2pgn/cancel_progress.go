package bin2pgn

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
)

// ErrCancelled means the caller requested a clean stop of the current BIN
// source. It is not a damaged-source or conversion error.
var ErrCancelled = errors.New("BIN conversion cancelled")

type CancelFunc func() bool

func checkCancelled(cancel CancelFunc) error {
	if cancel != nil && cancel() {
		return ErrCancelled
	}
	return nil
}

// ConvertWithProgressCancelable is the interruptible variant used by
// Source2Metal batch mode. The source BIN remains read-only and a cancelled
// conversion never promotes its temporary PGN to the final RAW filename.
func ConvertWithProgressCancelable(input, output string, maxPly int, fn ProgressFunc, cancel CancelFunc) (Stats, error) {
	if maxPly < 1 || maxPly > 120 {
		return Stats{}, fmt.Errorf("BIN depth must be between 1 and 120 ply")
	}
	return convertBookCancelable(input, output, maxPly, fn, cancel)
}

func loadBookCancelable(path string, fn ProgressFunc, cancel CancelFunc) (Book, error) {
	f, err := os.Open(path)
	if err != nil {
		return Book{}, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return Book{}, err
	}
	if rem := st.Size() % 16; rem != 0 {
		return Book{}, fmt.Errorf("Polyglot BIN eindigt met een onvolledig 16-byte record (%d restbytes)", rem)
	}

	n := st.Size() / 16
	m := make(map[uint64][]Entry, minInt64(n, 800000))
	buf := make([]byte, 16)
	bs := BookStats{SortedByKey: true}
	var lastKey uint64
	haveLast := false
	emitProgress(fn, "index", 0, n, 0, 0)
	for {
		if bs.Records%8192 == 0 {
			if err := checkCancelled(cancel); err != nil {
				return Book{}, err
			}
		}
		_, e := io.ReadFull(f, buf)
		if e == io.EOF {
			break
		}
		if e == io.ErrUnexpectedEOF {
			return Book{}, fmt.Errorf("onvolledig Polyglot-record aan einde")
		}
		if e != nil {
			return Book{}, e
		}
		ent := Entry{
			Key: binary.BigEndian.Uint64(buf[:8]), Move: binary.BigEndian.Uint16(buf[8:10]),
			Weight: binary.BigEndian.Uint16(buf[10:12]), Learn: binary.BigEndian.Uint32(buf[12:16]),
		}
		if haveLast && ent.Key < lastKey {
			bs.SortedByKey = false
		}
		lastKey, haveLast = ent.Key, true
		if ent.Weight == 0 {
			bs.ZeroWeight++
		} else {
			bs.ActiveWeight++
		}
		m[ent.Key] = append(m[ent.Key], ent)
		bs.Records++
		if progressDue(bs.Records, n, 262144) {
			emitProgress(fn, "index", bs.Records, n, 0, 0)
		}
	}

	totalKeys := int64(len(m))
	var normalized int64
	emitProgress(fn, "normalize", 0, totalKeys, 0, 0)
	for k, entries := range m {
		if normalized%2048 == 0 {
			if err := checkCancelled(cancel); err != nil {
				return Book{}, err
			}
		}
		seenMove := make(map[uint16]struct{}, len(entries))
		for _, e := range entries {
			if _, ok := seenMove[e.Move]; ok {
				bs.DuplicateKeyMove++
			} else {
				seenMove[e.Move] = struct{}{}
			}
		}
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].Weight != entries[j].Weight {
				return entries[i].Weight > entries[j].Weight
			}
			return entries[i].Move < entries[j].Move
		})
		m[k] = entries
		normalized++
		if progressDue(normalized, totalKeys, 32768) {
			emitProgress(fn, "normalize", normalized, totalKeys, 0, 0)
		}
	}
	if err := checkCancelled(cancel); err != nil {
		return Book{}, err
	}
	return Book{Entries: m, Stats: bs}, nil
}

func convertBookCancelable(path, outPath string, maxPly int, fn ProgressFunc, cancel CancelFunc) (Stats, error) {
	book, err := loadBookCancelable(path, fn, cancel)
	if err != nil {
		return Stats{}, err
	}
	stats := Stats{
		Physical: book.Stats.Records, MaxDepth: maxPly,
		ActiveWeight: book.Stats.ActiveWeight, ZeroWeight: book.Stats.ZeroWeight,
		SortedByKey: book.Stats.SortedByKey, DuplicateKeyMove: book.Stats.DuplicateKeyMove,
	}
	g, err := buildReachableGraphCancelable(book, maxPly, &stats, fn, cancel)
	if err != nil {
		return stats, err
	}
	lines, err := buildCompleteLinesCancelable(g, maxPly, &stats, fn, cancel)
	if err != nil {
		return stats, err
	}
	if len(lines) == 0 {
		return stats, fmt.Errorf("geen bereikbare geldige complete boeklijnen; er wordt geen lege PGN geschreven")
	}
	if stats.CoverageMisses != 0 {
		return stats, fmt.Errorf("interne dekkingsfout: %d bereikbare boekzetten ontbreken", stats.CoverageMisses)
	}
	if err := checkCancelled(cancel); err != nil {
		return stats, err
	}

	tmp := outPath + ".tmp"
	_ = os.Remove(tmp)
	out, err := os.Create(tmp)
	if err != nil {
		return stats, err
	}
	removeTemp := func() {
		_ = out.Close()
		_ = os.Remove(tmp)
	}
	w := bufio.NewWriterSize(out, 1<<20)
	totalLines := int64(len(lines))
	emitProgress(fn, "write", 0, totalLines, stats.ReachablePositions, stats.ReachableRecords)
	for i, line := range lines {
		if i%256 == 0 {
			if err := checkCancelled(cancel); err != nil {
				removeTemp()
				return stats, err
			}
		}
		if err := validateLine(line, maxPly); err != nil {
			removeTemp()
			return stats, fmt.Errorf("lijnvalidatie vóór schrijven: %w", err)
		}
		if err := writeLine(w, line); err != nil {
			removeTemp()
			return stats, err
		}
		cur := int64(i + 1)
		if progressDue(cur, totalLines, 2048) {
			emitProgress(fn, "write", cur, totalLines, stats.ReachablePositions, stats.ReachableRecords)
		}
	}
	if err := w.Flush(); err != nil {
		removeTemp()
		return stats, err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return stats, err
	}
	if err := checkCancelled(cancel); err != nil {
		_ = os.Remove(tmp)
		return stats, err
	}

	emitProgress(fn, "validate", 0, 0, stats.ReachablePositions, stats.ReachableRecords)
	vr, err := validateWrittenPGN(tmp)
	if err != nil {
		_ = os.Remove(tmp)
		return stats, fmt.Errorf("PGN-eindcontrole: %w", err)
	}
	if err := checkCancelled(cancel); err != nil {
		_ = os.Remove(tmp)
		return stats, err
	}
	stats.PostValidatedLines = vr.Records
	if vr.Records != int64(len(lines)) {
		_ = os.Remove(tmp)
		return stats, fmt.Errorf("PGN-eindcontrole: geschreven %d, teruggelezen %d", len(lines), vr.Records)
	}

	totalEdges := int64(len(g.Edges))
	emitProgress(fn, "coverage", 0, totalEdges, stats.ReachablePositions, stats.ReachableRecords)
	for i, e := range g.Edges {
		if i%4096 == 0 {
			if err := checkCancelled(cancel); err != nil {
				_ = os.Remove(tmp)
				return stats, err
			}
		}
		if _, ok := vr.Seen[coverageToken(e.Key, e.Move)]; !ok {
			stats.CoverageMisses++
		}
		cur := int64(i + 1)
		if progressDue(cur, totalEdges, 16384) {
			emitProgress(fn, "coverage", cur, totalEdges, stats.ReachablePositions, stats.ReachableRecords)
		}
	}
	if stats.CoverageMisses != 0 {
		_ = os.Remove(tmp)
		return stats, fmt.Errorf("PGN-eindcontrole: %d bereikbare boekzetten ontbreken na teruglezen", stats.CoverageMisses)
	}
	if err := checkCancelled(cancel); err != nil {
		_ = os.Remove(tmp)
		return stats, err
	}
	if err := os.Rename(tmp, outPath); err != nil {
		_ = os.Remove(tmp)
		return stats, err
	}
	emitProgress(fn, "done", 1, 1, stats.ReachablePositions, stats.ReachableRecords)
	return stats, nil
}

func buildReachableGraphCancelable(book Book, maxPly int, stats *Stats, fn ProgressFunc, cancel CancelFunc) (*BookGraph, error) {
	start := startBoard()
	if k := polyglotKey(start); k != 0x463b96181691fc9c {
		return nil, fmt.Errorf("interne Polyglot-hashfout: beginstelling %016x", k)
	}
	startID := boardStateID(start)
	g := &BookGraph{StartID: startID, Nodes: map[string]*GraphNode{}, ByID: map[string]*GraphEdge{}}
	sn := &GraphNode{ID: startID, Board: start, Key: polyglotKey(start), Depth: 0}
	g.Nodes[startID] = sn
	queue := []*GraphNode{sn}
	expanded := map[string]bool{}
	var expandedCount int64
	emitProgress(fn, "graph", 0, 1, 1, 0)

	for len(queue) > 0 {
		if expandedCount%1024 == 0 {
			if err := checkCancelled(cancel); err != nil {
				return nil, err
			}
		}
		n := queue[0]
		queue = queue[1:]
		if expanded[n.ID] {
			continue
		}
		expanded[n.ID] = true
		expandedCount++
		if n.Depth < maxPly {
			ents := book.Entries[n.Key]
			if len(ents) > 0 {
				legal := n.Board.legalMoves()
				type cand struct {
					e   Entry
					m   Move
					san string
				}
				var cands []cand
				seenNorm := map[string]struct{}{}
				for _, e := range ents {
					bm, ok := decodePolyglotMove(e.Move, n.Board)
					if !ok {
						stats.DecodeErrors++
						continue
					}
					var lm Move
					found := false
					for _, x := range legal {
						if x.From == bm.From && x.To == bm.To && x.Promo == bm.Promo {
							lm = x
							found = true
							break
						}
					}
					if !found {
						stats.IllegalMoves++
						continue
					}
					mk := uci(lm)
					if _, dup := seenNorm[mk]; dup {
						stats.NormalizedDuplicateMoves++
						continue
					}
					seenNorm[mk] = struct{}{}
					cands = append(cands, cand{e: e, m: lm, san: n.Board.san(lm, legal)})
				}
				var total uint64
				for _, c := range cands {
					total += uint64(c.e.Weight)
				}
				for _, c := range cands {
					child := n.Board.after(c.m)
					toID := boardStateID(child)
					rel := 0.0
					if total > 0 {
						rel = float64(c.e.Weight) * 100 / float64(total)
					}
					eid := n.ID + "|" + uci(c.m)
					if _, exists := g.ByID[eid]; exists {
						continue
					}
					e := &GraphEdge{ID: eid, FromID: n.ID, ToID: toID, Key: n.Key, Entry: c.e, Move: c.m, SAN: c.san, Relative: rel}
					n.Out = append(n.Out, e)
					g.Edges = append(g.Edges, e)
					g.ByID[eid] = e
					cn, ok := g.Nodes[toID]
					if !ok {
						cn = &GraphNode{ID: toID, Board: child, Key: polyglotKey(child), Depth: n.Depth + 1, ParentEdge: eid}
						g.Nodes[toID] = cn
						queue = append(queue, cn)
					} else if n.Depth+1 < cn.Depth {
						cn.Depth = n.Depth + 1
						cn.ParentEdge = eid
						queue = append(queue, cn)
					}
				}
			}
		}
		if expandedCount%16384 == 0 || len(queue) == 0 {
			emitProgress(fn, "graph", expandedCount, int64(len(g.Nodes)), int64(len(g.Nodes)), int64(len(g.Edges)))
		}
	}
	if err := checkCancelled(cancel); err != nil {
		return nil, err
	}
	for _, n := range g.Nodes {
		sort.SliceStable(n.Out, func(i, j int) bool {
			if n.Out[i].Entry.Weight != n.Out[j].Entry.Weight {
				return n.Out[i].Entry.Weight > n.Out[j].Entry.Weight
			}
			return uci(n.Out[i].Move) < uci(n.Out[j].Move)
		})
	}
	stats.ReachablePositions = int64(len(g.Nodes))
	stats.ReachableRecords = int64(len(g.Edges))
	emitProgress(fn, "graph", stats.ReachablePositions, stats.ReachablePositions, stats.ReachablePositions, stats.ReachableRecords)
	return g, nil
}

func buildCompleteLinesCancelable(g *BookGraph, maxPly int, stats *Stats, fn ProgressFunc, cancel CancelFunc) ([]Line, error) {
	edges := append([]*GraphEdge(nil), g.Edges...)
	sort.SliceStable(edges, func(i, j int) bool {
		ni, nj := g.Nodes[edges[i].FromID], g.Nodes[edges[j].FromID]
		if ni.Depth != nj.Depth {
			return ni.Depth < nj.Depth
		}
		if edges[i].Entry.Weight != edges[j].Entry.Weight {
			return edges[i].Entry.Weight > edges[j].Entry.Weight
		}
		if edges[i].Key != edges[j].Key {
			return edges[i].Key < edges[j].Key
		}
		return uci(edges[i].Move) < uci(edges[j].Move)
	})
	covered := map[string]bool{}
	seenLine := map[string]bool{}
	var lines []Line
	totalEdges := int64(len(edges))
	emitProgress(fn, "lines", 0, totalEdges, int64(len(g.Nodes)), int64(len(g.Edges)))
	for i, target := range edges {
		if i%1024 == 0 {
			if err := checkCancelled(cancel); err != nil {
				return nil, err
			}
		}
		if !covered[target.ID] {
			prefix, err := canonicalPath(g, target.FromID)
			if err != nil {
				return nil, err
			}
			seq := append(append([]*GraphEdge{}, prefix...), target)
			cur := g.Nodes[target.ToID]
			for len(seq) < maxPly && len(cur.Out) > 0 {
				var next *GraphEdge
				for _, e := range cur.Out {
					if !covered[e.ID] {
						next = e
						break
					}
				}
				if next == nil {
					next = cur.Out[0]
				}
				seq = append(seq, next)
				cur = g.Nodes[next.ToID]
			}
			key := lineKey(seq)
			if seenLine[key] {
				stats.DuplicateLines++
			} else {
				seenLine[key] = true
				for _, e := range seq {
					covered[e.ID] = true
				}
				line := Line{Edges: seq}
				if err := validateLine(line, maxPly); err != nil {
					return nil, err
				}
				lines = append(lines, line)
				if len(seq) > stats.MaxObservedPly {
					stats.MaxObservedPly = len(seq)
				}
				if len(seq) >= maxPly {
					stats.DepthEndedLines++
				} else if len(cur.Out) == 0 {
					stats.LeafEndedLines++
				}
			}
		}
		cur := int64(i + 1)
		if progressDue(cur, totalEdges, 16384) {
			emitProgress(fn, "lines", cur, totalEdges, int64(len(g.Nodes)), int64(len(g.Edges)))
		}
	}
	for _, e := range g.Edges {
		if !covered[e.ID] {
			stats.CoverageMisses++
		}
	}
	stats.CompleteLines = int64(len(lines))
	return lines, checkCancelled(cancel)
}

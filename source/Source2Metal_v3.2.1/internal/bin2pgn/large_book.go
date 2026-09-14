package bin2pgn

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
)

// The classic loader is intentionally kept for normal opening books because it
// is simple and fast there. Above this size, retaining every physical record in
// a Go map becomes needlessly expensive (Knowledge.bin is about 94M records),
// so conversion switches to a sparse on-disk index.
const (
	largeBookThresholdRecords int64 = 2_000_000
	indexedBlockRecords       int64 = 4_096
	scanChunkRecords          int64 = 262_144 // 4 MiB of 16-byte records
)

type bookCheckpoint struct {
	Key    uint64
	Record int64
}

type indexedBook struct {
	f           *os.File
	records     int64
	checkpoints []bookCheckpoint
	stats       BookStats
}

func convertBookAdaptive(path, outPath string, maxPly int, progress ProgressFunc) (Stats, error) {
	// A previous Ctrl+C must never make a new run look active merely because a
	// stale zero-byte temporary file is still present.
	_ = os.Remove(outPath + ".tmp")

	st, err := os.Stat(path)
	if err != nil {
		return Stats{}, err
	}
	if rem := st.Size() % 16; rem != 0 {
		return Stats{}, fmt.Errorf("Polyglot BIN eindigt met een onvolledig 16-byte record (%d restbytes)", rem)
	}
	records := st.Size() / 16
	if records <= largeBookThresholdRecords {
		return convertBook(path, outPath, maxPly)
	}
	return convertBookIndexed(path, outPath, maxPly, progress)
}

func openIndexedBook(path string, progress ProgressFunc) (*indexedBook, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	fail := func(e error) (*indexedBook, error) {
		_ = f.Close()
		return nil, e
	}
	st, err := f.Stat()
	if err != nil {
		return fail(err)
	}
	if rem := st.Size() % 16; rem != 0 {
		return fail(fmt.Errorf("Polyglot BIN eindigt met een onvolledig 16-byte record (%d restbytes)", rem))
	}

	n := st.Size() / 16
	idx := &indexedBook{
		f:       f,
		records: n,
		stats: BookStats{
			Records:     n,
			SortedByKey: true,
		},
	}
	if n == 0 {
		return idx, nil
	}
	idx.checkpoints = make([]bookCheckpoint, 0, (n+indexedBlockRecords-1)/indexedBlockRecords)

	if progress != nil {
		progress(Progress{Stage: ProgressScan, Done: 0, Total: n})
	}

	buf := make([]byte, scanChunkRecords*16)
	var absolute int64
	var lastKey uint64
	haveLast := false
	var currentKey uint64
	haveCurrent := false
	seenMoves := make(map[uint16]struct{}, 8)
	// About one progress update per percentage, with a sensible minimum.
	progressStep := n / 100
	if progressStep < 100_000 {
		progressStep = 100_000
	}
	nextProgress := progressStep

	for absolute < n {
		count := n - absolute
		if count > scanChunkRecords {
			count = scanChunkRecords
		}
		need := int(count * 16)
		if _, err := io.ReadFull(f, buf[:need]); err != nil {
			return fail(err)
		}
		for j := int64(0); j < count; j++ {
			i := absolute + j
			off := int(j * 16)
			raw := buf[off : off+16]
			key := binary.BigEndian.Uint64(raw[:8])
			move := binary.BigEndian.Uint16(raw[8:10])
			weight := binary.BigEndian.Uint16(raw[10:12])

			if i%indexedBlockRecords == 0 {
				idx.checkpoints = append(idx.checkpoints, bookCheckpoint{Key: key, Record: i})
			}
			if haveLast && key < lastKey {
				idx.stats.SortedByKey = false
				return fail(fmt.Errorf("grote Polyglot BIN is niet op sleutel gesorteerd bij record %d; veilige schijfindexering is daarom niet mogelijk", i+1))
			}
			lastKey, haveLast = key, true

			if weight == 0 {
				idx.stats.ZeroWeight++
			} else {
				idx.stats.ActiveWeight++
			}

			if !haveCurrent || key != currentKey {
				currentKey, haveCurrent = key, true
				clear(seenMoves)
			}
			if _, ok := seenMoves[move]; ok {
				idx.stats.DuplicateKeyMove++
			} else {
				seenMoves[move] = struct{}{}
			}
		}
		absolute += count
		if progress != nil && (absolute >= nextProgress || absolute == n) {
			progress(Progress{Stage: ProgressScan, Done: absolute, Total: n})
			for nextProgress <= absolute {
				nextProgress += progressStep
			}
		}
	}
	return idx, nil
}

func (b *indexedBook) Close() error {
	if b == nil || b.f == nil {
		return nil
	}
	return b.f.Close()
}

func (b *indexedBook) lookup(key uint64) ([]Entry, error) {
	if b == nil || b.f == nil || b.records == 0 || len(b.checkpoints) == 0 {
		return nil, nil
	}

	// Find the first block whose first key is >= target and start one block
	// earlier. The one-block overlap is deliberate: all records for one key can
	// straddle a block boundary and must be returned together.
	cp := sort.Search(len(b.checkpoints), func(i int) bool {
		return b.checkpoints[i].Key >= key
	})
	startCP := 0
	if cp > 0 {
		startCP = cp - 1
	}
	start := b.checkpoints[startCP].Record
	block := make([]byte, indexedBlockRecords*16)
	var entries []Entry

	for start < b.records {
		count := b.records - start
		if count > indexedBlockRecords {
			count = indexedBlockRecords
		}
		need := int(count * 16)
		nRead, err := b.f.ReadAt(block[:need], start*16)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if nRead != need {
			return nil, io.ErrUnexpectedEOF
		}

		for j := int64(0); j < count; j++ {
			off := int(j * 16)
			raw := block[off : off+16]
			k := binary.BigEndian.Uint64(raw[:8])
			if k < key {
				continue
			}
			if k > key {
				sortEntries(entries)
				return entries, nil
			}
			entries = append(entries, Entry{
				Key:    k,
				Move:   binary.BigEndian.Uint16(raw[8:10]),
				Weight: binary.BigEndian.Uint16(raw[10:12]),
				Learn:  binary.BigEndian.Uint32(raw[12:16]),
			})
		}
		start += count
	}
	sortEntries(entries)
	return entries, nil
}

func sortEntries(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Weight != entries[j].Weight {
			return entries[i].Weight > entries[j].Weight
		}
		return entries[i].Move < entries[j].Move
	})
}

func convertBookIndexed(path, outPath string, maxPly int, progress ProgressFunc) (Stats, error) {
	book, err := openIndexedBook(path, progress)
	if err != nil {
		return Stats{}, err
	}
	defer book.Close()

	stats := Stats{
		Physical:         book.stats.Records,
		MaxDepth:         maxPly,
		ActiveWeight:     book.stats.ActiveWeight,
		ZeroWeight:       book.stats.ZeroWeight,
		SortedByKey:      book.stats.SortedByKey,
		DuplicateKeyMove: book.stats.DuplicateKeyMove,
	}
	g, err := buildReachableGraphIndexed(book, maxPly, &stats, progress)
	if err != nil {
		return stats, err
	}
	return finishIndexedConversion(g, outPath, maxPly, &stats, progress)
}

func buildReachableGraphIndexed(book *indexedBook, maxPly int, stats *Stats, progress ProgressFunc) (*BookGraph, error) {
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
	if progress != nil {
		progress(Progress{Stage: ProgressGraph, Done: 0, Total: 0})
	}

	for head := 0; head < len(queue); head++ {
		n := queue[head]
		if expanded[n.ID] {
			continue
		}
		expanded[n.ID] = true
		expandedCount++
		if progress != nil && expandedCount%100_000 == 0 {
			progress(Progress{Stage: ProgressGraph, Done: expandedCount, Total: 0})
		}
		if n.Depth >= maxPly {
			continue
		}

		ents, err := book.lookup(n.Key)
		if err != nil {
			return nil, err
		}
		if len(ents) == 0 {
			continue
		}
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
	if progress != nil {
		progress(Progress{Stage: ProgressGraph, Done: expandedCount, Total: 0})
	}
	return g, nil
}

func finishIndexedConversion(g *BookGraph, outPath string, maxPly int, stats *Stats, progress ProgressFunc) (Stats, error) {
	if progress != nil {
		progress(Progress{Stage: ProgressLines, Done: 0, Total: int64(len(g.Edges))})
	}
	lines, err := buildCompleteLines(g, maxPly, stats)
	if err != nil {
		return *stats, err
	}
	if progress != nil {
		progress(Progress{Stage: ProgressLines, Done: int64(len(g.Edges)), Total: int64(len(g.Edges))})
	}
	if len(lines) == 0 {
		return *stats, fmt.Errorf("geen bereikbare geldige complete boeklijnen; er wordt geen lege PGN geschreven")
	}
	if stats.CoverageMisses != 0 {
		return *stats, fmt.Errorf("interne dekkingsfout: %d bereikbare boekzetten ontbreken", stats.CoverageMisses)
	}

	tmp := outPath + ".tmp"
	_ = os.Remove(tmp)
	out, err := os.Create(tmp)
	if err != nil {
		return *stats, err
	}
	w := bufio.NewWriterSize(out, 1<<20)
	if progress != nil {
		progress(Progress{Stage: ProgressWrite, Done: 0, Total: int64(len(lines))})
	}
	progressStep := int64(len(lines)) / 100
	if progressStep < 1_000 {
		progressStep = 1_000
	}
	for i, line := range lines {
		if err := validateLine(line, maxPly); err != nil {
			out.Close()
			_ = os.Remove(tmp)
			return *stats, fmt.Errorf("lijnvalidatie vóór schrijven: %w", err)
		}
		if err := writeLine(w, line); err != nil {
			out.Close()
			_ = os.Remove(tmp)
			return *stats, err
		}
		done := int64(i + 1)
		if progress != nil && (done%progressStep == 0 || done == int64(len(lines))) {
			progress(Progress{Stage: ProgressWrite, Done: done, Total: int64(len(lines))})
		}
	}
	if err := w.Flush(); err != nil {
		out.Close()
		_ = os.Remove(tmp)
		return *stats, err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return *stats, err
	}

	if progress != nil {
		progress(Progress{Stage: ProgressValidate, Done: 0, Total: int64(len(lines))})
	}
	vr, err := validateWrittenPGN(tmp)
	if err != nil {
		_ = os.Remove(tmp)
		return *stats, fmt.Errorf("PGN-eindcontrole: %w", err)
	}
	stats.PostValidatedLines = vr.Records
	if vr.Records != int64(len(lines)) {
		_ = os.Remove(tmp)
		return *stats, fmt.Errorf("PGN-eindcontrole: geschreven %d, teruggelezen %d", len(lines), vr.Records)
	}
	for _, e := range g.Edges {
		if _, ok := vr.Seen[coverageToken(e.Key, e.Move)]; !ok {
			stats.CoverageMisses++
		}
	}
	if stats.CoverageMisses != 0 {
		_ = os.Remove(tmp)
		return *stats, fmt.Errorf("PGN-eindcontrole: %d bereikbare boekzetten ontbreken na teruglezen", stats.CoverageMisses)
	}
	if progress != nil {
		progress(Progress{Stage: ProgressValidate, Done: vr.Records, Total: int64(len(lines))})
	}
	if err := os.Rename(tmp, outPath); err != nil {
		_ = os.Remove(tmp)
		return *stats, err
	}
	return *stats, nil
}

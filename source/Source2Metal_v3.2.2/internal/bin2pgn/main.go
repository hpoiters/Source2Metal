package bin2pgn

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const version = "0.1.3"

const (
	empty  = 0
	pawn   = 1
	knight = 2
	bishop = 3
	rook   = 4
	queen  = 5
	king   = 6
	white  = 1
	black  = -1
	wk     = 1
	wq     = 2
	bk     = 4
	bq     = 8
)

type Move struct{ From, To, Promo int }
type Board struct {
	Sq               [64]int
	Side, Castle, EP int
}

type Entry struct {
	Key    uint64
	Move   uint16
	Weight uint16
	Learn  uint32
}

type BookStats struct {
	Records          int64
	ActiveWeight     int64
	ZeroWeight       int64
	SortedByKey      bool
	DuplicateKeyMove int64
}

type Book struct {
	Entries map[uint64][]Entry
	Stats   BookStats
}

type Stats struct {
	Physical                 int64
	ReachableRecords         int64 // bereikbare unieke legale boekzetten
	ReachablePositions       int64 // inclusief beginstelling
	DecodeErrors             int64
	IllegalMoves             int64
	MaxDepth                 int
	MaxObservedPly           int
	ActiveWeight             int64
	ZeroWeight               int64
	SortedByKey              bool
	DuplicateKeyMove         int64
	NormalizedDuplicateMoves int64
	CompleteLines            int64
	DuplicateLines           int64
	LeafEndedLines           int64
	DepthEndedLines          int64
	PostValidatedLines       int64
	CoverageMisses           int64
}

type GraphEdge struct {
	ID       string
	FromID   string
	ToID     string
	Key      uint64
	Entry    Entry
	Move     Move
	SAN      string
	Relative float64
}

type GraphNode struct {
	ID         string
	Board      Board
	Key        uint64
	Depth      int
	ParentEdge string
	Out        []*GraphEdge
}

type BookGraph struct {
	StartID string
	Nodes   map[string]*GraphNode
	Edges   []*GraphEdge
	ByID    map[string]*GraphEdge
}

type Line struct {
	Edges []*GraphEdge
}

type ValidationResult struct {
	Records int64
	Seen    map[string]struct{}
}

func loadBook(path string) (Book, error) {
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
	for {
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
	}

	// BINProbe-achtige robuustheidsstatistiek: dubbele key+zet detecteren.
	for k, entries := range m {
		seenMove := make(map[uint16]struct{}, len(entries))
		for _, e := range entries {
			if _, ok := seenMove[e.Move]; ok {
				bs.DuplicateKeyMove++
			} else {
				seenMove[e.Move] = struct{}{}
			}
		}
		// Deterministische lookup: zwaarste zet eerst, daarna raw move-code.
		sort.SliceStable(entries, func(i, j int) bool {
			if entries[i].Weight != entries[j].Weight {
				return entries[i].Weight > entries[j].Weight
			}
			return entries[i].Move < entries[j].Move
		})
		m[k] = entries
	}
	return Book{Entries: m, Stats: bs}, nil
}

func minInt64(n int64, capn int) int {
	if n > int64(capn) {
		return capn
	}
	return int(n)
}

// convertBook bouwt eerst de bereikbare legale Polyglot-graaf. Daarna worden
// volledige beginstelling->boek-einde/max-diepte lijnen gemaakt. Een greedy
// edge-cover houdt het aantal lijnen compact, terwijl iedere bereikbare unieke
// boekzet minstens eenmaal in de RAW-PGN voorkomt.
func convertBook(path, outPath string, maxPly int) (Stats, error) {
	book, err := loadBook(path)
	if err != nil {
		return Stats{}, err
	}
	stats := Stats{
		Physical: book.Stats.Records, MaxDepth: maxPly,
		ActiveWeight: book.Stats.ActiveWeight, ZeroWeight: book.Stats.ZeroWeight,
		SortedByKey: book.Stats.SortedByKey, DuplicateKeyMove: book.Stats.DuplicateKeyMove,
	}
	g, err := buildReachableGraph(book, maxPly, &stats)
	if err != nil {
		return stats, err
	}
	lines, err := buildCompleteLines(g, maxPly, &stats)
	if err != nil {
		return stats, err
	}
	if len(lines) == 0 {
		return stats, fmt.Errorf("geen bereikbare geldige complete boeklijnen; er wordt geen lege PGN geschreven")
	}
	if stats.CoverageMisses != 0 {
		return stats, fmt.Errorf("interne dekkingsfout: %d bereikbare boekzetten ontbreken", stats.CoverageMisses)
	}

	tmp := outPath + ".tmp"
	_ = os.Remove(tmp)
	out, err := os.Create(tmp)
	if err != nil {
		return stats, err
	}
	w := bufio.NewWriterSize(out, 1<<20)
	for _, line := range lines {
		if err := validateLine(line, maxPly); err != nil {
			out.Close()
			_ = os.Remove(tmp)
			return stats, fmt.Errorf("lijnvalidatie vóór schrijven: %w", err)
		}
		if err := writeLine(w, line); err != nil {
			out.Close()
			_ = os.Remove(tmp)
			return stats, err
		}
	}
	if err := w.Flush(); err != nil {
		out.Close()
		_ = os.Remove(tmp)
		return stats, err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return stats, err
	}

	vr, err := validateWrittenPGN(tmp)
	if err != nil {
		_ = os.Remove(tmp)
		return stats, fmt.Errorf("PGN-eindcontrole: %w", err)
	}
	stats.PostValidatedLines = vr.Records
	if vr.Records != int64(len(lines)) {
		_ = os.Remove(tmp)
		return stats, fmt.Errorf("PGN-eindcontrole: geschreven %d, teruggelezen %d", len(lines), vr.Records)
	}
	for _, e := range g.Edges {
		if _, ok := vr.Seen[coverageToken(e.Key, e.Move)]; !ok {
			stats.CoverageMisses++
		}
	}
	if stats.CoverageMisses != 0 {
		_ = os.Remove(tmp)
		return stats, fmt.Errorf("PGN-eindcontrole: %d bereikbare boekzetten ontbreken na teruglezen", stats.CoverageMisses)
	}
	if err := os.Rename(tmp, outPath); err != nil {
		_ = os.Remove(tmp)
		return stats, err
	}
	return stats, nil
}

func buildReachableGraph(book Book, maxPly int, stats *Stats) (*BookGraph, error) {
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

	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		if expanded[n.ID] {
			continue
		}
		expanded[n.ID] = true
		if n.Depth >= maxPly {
			continue
		}
		ents := book.Entries[n.Key]
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
	return g, nil
}

func buildCompleteLines(g *BookGraph, maxPly int, stats *Stats) ([]Line, error) {
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
	for _, target := range edges {
		if covered[target.ID] {
			continue
		}
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
			continue
		}
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
	for _, e := range g.Edges {
		if !covered[e.ID] {
			stats.CoverageMisses++
		}
	}
	stats.CompleteLines = int64(len(lines))
	return lines, nil
}

func canonicalPath(g *BookGraph, nodeID string) ([]*GraphEdge, error) {
	if nodeID == g.StartID {
		return nil, nil
	}
	var rev []*GraphEdge
	seen := map[string]bool{}
	cur := nodeID
	for cur != g.StartID {
		if seen[cur] {
			return nil, fmt.Errorf("cyclus in canoniek ouderpad")
		}
		seen[cur] = true
		n := g.Nodes[cur]
		if n == nil || n.ParentEdge == "" {
			return nil, fmt.Errorf("geen ouderpad naar bereikbare positie")
		}
		e := g.ByID[n.ParentEdge]
		if e == nil {
			return nil, fmt.Errorf("ontbrekende parent edge")
		}
		rev = append(rev, e)
		cur = e.FromID
	}
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	return rev, nil
}

func validateLine(line Line, maxPly int) error {
	if len(line.Edges) == 0 {
		return fmt.Errorf("lege lijn")
	}
	if len(line.Edges) > maxPly {
		return fmt.Errorf("lijn %d ply > max %d", len(line.Edges), maxPly)
	}
	b := startBoard()
	for i, e := range line.Edges {
		if polyglotKey(b) != e.Key {
			return fmt.Errorf("ply %d: BINKey wijkt af", i+1)
		}
		legal := b.legalMoves()
		found := false
		for _, lm := range legal {
			if lm == e.Move {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("ply %d: illegale zet %s", i+1, uci(e.Move))
		}
		b = b.after(e.Move)
	}
	return nil
}

func writeLine(w *bufio.Writer, line Line) error {
	tags := fmt.Sprintf(
		"[Event \"BIN2PGN v%s BOOK RAW\"]\n[Site \"?\"]\n[Date \"????.??.??\"]\n[Round \"-\"]\n[White \"BIN\"]\n[Black \"BIN\"]\n[Result \"*\"]\n[SourceType \"BIN\"]\n[PlyCount \"%d\"]\n\n",
		version, len(line.Edges),
	)
	if _, err := w.WriteString(tags); err != nil {
		return err
	}
	var sb strings.Builder
	for i, e := range line.Edges {
		if i&1 == 0 {
			fmt.Fprintf(&sb, "%d. ", i/2+1)
		}
		sb.WriteString(e.SAN)
		sb.WriteByte(' ')
	}
	sb.WriteString("*")
	if _, err := w.WriteString(strings.TrimSpace(sb.String())); err != nil {
		return err
	}
	_, err := w.WriteString("\n\n")
	return err
}

func validateWrittenPGN(path string) (ValidationResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return ValidationResult{}, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 64*1024), 8*1024*1024)
	vr := ValidationResult{Seen: map[string]struct{}{}}
	plyCount := -1
	haveResultTag := false
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[PlyCount ") {
			q1 := strings.Index(line, "\"")
			q2 := strings.LastIndex(line, "\"")
			if q1 < 0 || q2 <= q1 {
				return vr, fmt.Errorf("ongeldige PlyCount-tag")
			}
			n, e := strconv.Atoi(line[q1+1 : q2])
			if e != nil || n < 1 {
				return vr, fmt.Errorf("ongeldige PlyCount")
			}
			plyCount = n
			continue
		}
		if strings.HasPrefix(line, "[Result ") {
			if line != "[Result \"*\"]" {
				return vr, fmt.Errorf("Result is niet *")
			}
			haveResultTag = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			continue
		}
		if plyCount < 1 || !haveResultTag {
			return vr, fmt.Errorf("movetext zonder volledige headers")
		}
		// BOOK RAW blijft bewust schoon: geen technische BIN-commentaren nodig.
		// De eindcontrole reconstrueert iedere zet uit SAN en berekent zelf de
		// Polyglot-key, zodat legaliteit en volledige zetdekking onafhankelijk
		// van verborgen metadata worden gecontroleerd.
		sanTokens := generatedSANTokens(line)
		if len(sanTokens) != plyCount {
			return vr, fmt.Errorf("PlyCount %d maar %d SAN-zetten", plyCount, len(sanTokens))
		}
		b := startBoard()
		for i, tok := range sanTokens {
			legal := b.legalMoves()
			m, ok := findLegalSANFromLegal(b, legal, tok)
			if !ok {
				return vr, fmt.Errorf("partij %d ply %d ongeldige/ambigue SAN %s", vr.Records+1, i+1, tok)
			}
			k := polyglotKey(b)
			vr.Seen[coverageToken(k, m)] = struct{}{}
			b = b.after(m)
		}
		if !strings.HasSuffix(line, "*") {
			return vr, fmt.Errorf("partij %d heeft geen *-resultaat", vr.Records+1)
		}
		vr.Records++
		plyCount = -1
		haveResultTag = false
	}
	if err := s.Err(); err != nil {
		return vr, err
	}
	if plyCount != -1 {
		return vr, fmt.Errorf("onvolledige laatste partij")
	}
	return vr, nil
}

func findLegalSANFromLegal(b Board, legal []Move, san string) (Move, bool) {
	var found Move
	n := 0
	for _, m := range legal {
		if b.san(m, legal) == san {
			found = m
			n++
		}
	}
	return found, n == 1
}

func extractBraceComments(s string) []string {
	var out []string
	for {
		a := strings.IndexByte(s, '{')
		if a < 0 {
			break
		}
		b := strings.IndexByte(s[a+1:], '}')
		if b < 0 {
			break
		}
		b += a + 1
		out = append(out, s[a+1:b])
		s = s[b+1:]
	}
	return out
}
func parseKVComment(s string) map[string]string {
	m := map[string]string{}
	for _, f := range strings.Fields(s) {
		if i := strings.IndexByte(f, '='); i > 0 {
			m[f[:i]] = f[i+1:]
		}
	}
	return m
}
func removeBraceComments(s string) string {
	var out strings.Builder
	depth := 0
	for _, r := range s {
		if r == '{' {
			depth++
			continue
		}
		if r == '}' {
			if depth > 0 {
				depth--
			}
			continue
		}
		if depth == 0 {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func generatedSANTokens(line string) []string {
	clean := removeBraceComments(line)
	var out []string
	for _, f := range strings.Fields(clean) {
		if f == "*" {
			continue
		}
		if strings.HasSuffix(f, ".") {
			if _, err := strconv.Atoi(strings.TrimSuffix(f, ".")); err == nil {
				continue
			}
		}
		out = append(out, f)
	}
	return out
}

func findLegalUCIFromLegal(legal []Move, s string) (Move, bool) {
	for _, m := range legal {
		if uci(m) == s {
			return m, true
		}
	}
	return Move{}, false
}

func findLegalUCI(b Board, s string) (Move, bool) {
	return findLegalUCIFromLegal(b.legalMoves(), s)
}
func coverageToken(key uint64, m Move) string { return fmt.Sprintf("%016x/%s", key, uci(m)) }
func lineKey(seq []*GraphEdge) string {
	var sb strings.Builder
	for _, e := range seq {
		sb.WriteString(uci(e.Move))
		sb.WriteByte(' ')
	}
	return sb.String()
}
func boardStateID(b Board) string {
	buf := make([]byte, 67)
	for i, p := range b.Sq {
		buf[i] = byte(p + 6)
	}
	if b.Side == white {
		buf[64] = 1
	}
	buf[65] = byte(b.Castle)
	ep := -1
	if b.EP >= 0 && epCapturePossible(b) {
		ep = b.EP
	}
	buf[66] = byte(ep + 1)
	return string(buf)
}

func uniqueOutputPath(dir, name string) string {
	p := filepath.Join(dir, name)
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return p
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 2; ; i++ {
		p = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", base, i, ext))
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return p
		}
	}
}

func mergeBookPGNs(paths []string, outPath string) (written, duplicates int64, err error) {
	tmp := outPath + ".tmp"
	_ = os.Remove(tmp)
	out, e := os.Create(tmp)
	if e != nil {
		return 0, 0, e
	}
	w := bufio.NewWriterSize(out, 1<<20)
	seen := map[string]struct{}{}

	for _, p := range paths {
		f, e := os.Open(p)
		if e != nil {
			out.Close()
			_ = os.Remove(tmp)
			return written, duplicates, e
		}
		s := bufio.NewScanner(f)
		s.Buffer(make([]byte, 64*1024), 8*1024*1024)
		var block strings.Builder
		for s.Scan() {
			line := s.Text()
			trim := strings.TrimSpace(line)
			if trim == "" {
				if block.Len() > 0 {
					block.WriteByte('\n')
				}
				continue
			}
			block.WriteString(line)
			block.WriteByte('\n')
			if strings.HasPrefix(trim, "[") {
				continue
			}

			// BIN2PGN schrijft de volledige movetext op één regel; daarmee is het record compleet.
			b := strings.TrimSpace(block.String())
			key, e := cleanPGNSequenceKey(b)
			if e != nil {
				f.Close()
				out.Close()
				_ = os.Remove(tmp)
				return written, duplicates, fmt.Errorf("ongeldige BOOK RAW in %s: %w", p, e)
			}
			if _, ok := seen[key]; ok {
				duplicates++
			} else {
				seen[key] = struct{}{}
				if _, e := w.WriteString(b + "\n\n"); e != nil {
					f.Close()
					out.Close()
					_ = os.Remove(tmp)
					return written, duplicates, e
				}
				written++
			}
			block.Reset()
		}
		if e := s.Err(); e != nil {
			f.Close()
			out.Close()
			_ = os.Remove(tmp)
			return written, duplicates, e
		}
		if strings.TrimSpace(block.String()) != "" {
			f.Close()
			out.Close()
			_ = os.Remove(tmp)
			return written, duplicates, fmt.Errorf("onvolledig PGN-record in %s", p)
		}
		f.Close()
	}
	if e := w.Flush(); e != nil {
		out.Close()
		_ = os.Remove(tmp)
		return written, duplicates, e
	}
	if e := out.Close(); e != nil {
		_ = os.Remove(tmp)
		return written, duplicates, e
	}
	if _, e := validateWrittenPGN(tmp); e != nil {
		_ = os.Remove(tmp)
		return written, duplicates, fmt.Errorf("merged BOOK RAW eindcontrole: %w", e)
	}
	if e := os.Rename(tmp, outPath); e != nil {
		_ = os.Remove(tmp)
		return written, duplicates, e
	}
	return written, duplicates, nil
}

func cleanPGNSequenceKey(record string) (string, error) {
	var moveline string
	for _, ln := range strings.Split(record, "\n") {
		ln = strings.TrimSpace(ln)
		if ln != "" && !strings.HasPrefix(ln, "[") {
			moveline = ln
			break
		}
	}
	if moveline == "" {
		return "", fmt.Errorf("geen movetext")
	}
	tokens := generatedSANTokens(moveline)
	if len(tokens) == 0 {
		return "", fmt.Errorf("geen zetten")
	}
	b := startBoard()
	var key strings.Builder
	for i, tok := range tokens {
		legal := b.legalMoves()
		m, ok := findLegalSANFromLegal(b, legal, tok)
		if !ok {
			return "", fmt.Errorf("ply %d ongeldige/ambigue SAN %s", i+1, tok)
		}
		key.WriteString(uci(m))
		key.WriteByte(' ')
		b = b.after(m)
	}
	return key.String(), nil
}

func moveText(san []string, result string) string {
	var sb strings.Builder
	for i, x := range san {
		if i&1 == 0 {
			fmt.Fprintf(&sb, "%d. ", i/2+1)
		}
		sb.WriteString(x)
		sb.WriteByte(' ')
	}
	sb.WriteString(result)
	return strings.TrimSpace(sb.String())
}

func uci(m Move) string {
	s := squareName(m.From) + squareName(m.To)
	if m.Promo != 0 {
		s += strings.ToLower(pieceLetter(m.Promo))
	}
	return s
}

func decodePolyglotMove(v uint16, b Board) (Move, bool) {
	tf := int(v & 7)
	tr := int((v >> 3) & 7)
	ff := int((v >> 6) & 7)
	fr := int((v >> 9) & 7)
	pr := int((v >> 12) & 7)
	if pr > 4 {
		return Move{}, false
	}
	from := fr*8 + ff
	to := tr*8 + tf
	promo := 0
	switch pr {
	case 1:
		promo = knight
	case 2:
		promo = bishop
	case 3:
		promo = rook
	case 4:
		promo = queen
	}
	// Polyglot castling is king-to-rook encoded.
	if abs(b.Sq[from]) == king {
		if from == 4 && to == 7 {
			to = 6
		}
		if from == 4 && to == 0 {
			to = 2
		}
		if from == 60 && to == 63 {
			to = 62
		}
		if from == 60 && to == 56 {
			to = 58
		}
	}
	return Move{from, to, promo}, true
}

func polyglotKey(b Board) uint64 {
	var h uint64
	for sq, pc := range b.Sq {
		if pc == 0 {
			continue
		}
		pivot := 0
		if pc > 0 {
			pivot = 1
		}
		idx := (abs(pc)-1)*2 + pivot
		h ^= polyRandom[64*idx+sq]
	}
	if b.Castle&wk != 0 {
		h ^= polyRandom[768]
	}
	if b.Castle&wq != 0 {
		h ^= polyRandom[769]
	}
	if b.Castle&bk != 0 {
		h ^= polyRandom[770]
	}
	if b.Castle&bq != 0 {
		h ^= polyRandom[771]
	}
	if b.EP >= 0 && epCapturePossible(b) {
		h ^= polyRandom[772+(b.EP&7)]
	}
	if b.Side == white {
		h ^= polyRandom[780]
	}
	return h
}

func epCapturePossible(b Board) bool {
	if b.EP < 0 {
		return false
	}
	x, y := b.EP&7, b.EP>>3
	py := y - 1
	if b.Side == black {
		py = y + 1
	}
	if py < 0 || py > 7 {
		return false
	}
	for _, dx := range []int{-1, 1} {
		px := x + dx
		if px >= 0 && px < 8 && b.Sq[py*8+px] == b.Side*pawn {
			return true
		}
	}
	return false
}

func fileSHA256(path string) string {
	f, e := os.Open(path)
	if e != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	_, _ = io.Copy(h, f)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func selfTest() error {
	b := startBoard()
	if len(b.legalMoves()) != 20 {
		return fmt.Errorf("beginstelling heeft %d legale zetten", len(b.legalMoves()))
	}
	if k := polyglotKey(b); k != 0x463b96181691fc9c {
		return fmt.Errorf("Polyglot beginhash %016x", k)
	}
	var e4 Move
	ok := false
	for _, m := range b.legalMoves() {
		if uci(m) == "e2e4" {
			e4, ok = m, true
			break
		}
	}
	if !ok {
		return fmt.Errorf("e2e4 niet gevonden")
	}
	c := b.after(e4)
	if k := polyglotKey(c); k != 0x823c9b50fd114196 {
		return fmt.Errorf("hash na e2e4 %016x", k)
	}

	// Decodeer ook de Polyglot-castlingconventie expliciet.
	cb := startBoard()
	cb.Sq[5], cb.Sq[6] = 0, 0
	rawCastle := uint16(4<<6 | 7) // e1h1, Polyglot encoding
	dm, decOK := decodePolyglotMove(rawCastle, cb)
	if !decOK || dm.From != 4 || dm.To != 6 {
		return fmt.Errorf("Polyglot rokade-decode mislukt")
	}
	// Kleine synthetische boekgraaf: volledige lijnen moeten alle boekzetten dekken.
	tb := Book{Entries: map[uint64][]Entry{}, Stats: BookStats{SortedByKey: true}}
	add := func(b Board, m Move, weight uint16) {
		v := encodePolyglotMoveForTest(m, b)
		k := polyglotKey(b)
		tb.Entries[k] = append(tb.Entries[k], Entry{Key: k, Move: v, Weight: weight})
		tb.Stats.Records++
		tb.Stats.ActiveWeight++
	}
	find := func(b Board, u string) Move {
		m, ok := findLegalUCI(b, u)
		if !ok {
			panic("selftest move " + u)
		}
		return m
	}
	b0 := startBoard()
	e4m := find(b0, "e2e4")
	d4m := find(b0, "d2d4")
	add(b0, e4m, 100)
	add(b0, d4m, 80)
	b1 := b0.after(e4m)
	e5m := find(b1, "e7e5")
	c5m := find(b1, "c7c5")
	add(b1, e5m, 100)
	add(b1, c5m, 90)
	b2 := b1.after(e5m)
	nf3 := find(b2, "g1f3")
	add(b2, nf3, 100)
	bd := b0.after(d4m)
	d5m := find(bd, "d7d5")
	add(bd, d5m, 100)
	var st Stats
	g, er := buildReachableGraph(tb, 10, &st)
	if er != nil {
		return er
	}
	ls, er := buildCompleteLines(g, 10, &st)
	if er != nil {
		return er
	}
	if st.ReachableRecords != 6 || st.CoverageMisses != 0 || len(ls) != 3 {
		return fmt.Errorf("synthetische edge-cover: edges=%d lines=%d miss=%d", st.ReachableRecords, len(ls), st.CoverageMisses)
	}
	return nil
}

func encodePolyglotMoveForTest(m Move, b Board) uint16 {
	to := m.To
	if abs(b.Sq[m.From]) == king {
		if m.From == 4 && m.To == 6 {
			to = 7
		}
		if m.From == 4 && m.To == 2 {
			to = 0
		}
		if m.From == 60 && m.To == 62 {
			to = 63
		}
		if m.From == 60 && m.To == 58 {
			to = 56
		}
	}
	pr := 0
	switch m.Promo {
	case knight:
		pr = 1
	case bishop:
		pr = 2
	case rook:
		pr = 3
	case queen:
		pr = 4
	}
	return uint16((pr << 12) | ((m.From >> 3) << 9) | ((m.From & 7) << 6) | ((to >> 3) << 3) | (to & 7))
}

// ---- Compact legal chess board (shared design with Pgn2Metal) ------------

func startBoard() Board {
	var b Board
	b.Side = white
	b.Castle = wk | wq | bk | bq
	b.EP = -1
	back := []int{rook, knight, bishop, queen, king, bishop, knight, rook}
	for x := 0; x < 8; x++ {
		b.Sq[x] = back[x]
		b.Sq[8+x] = pawn
		b.Sq[48+x] = -pawn
		b.Sq[56+x] = -back[x]
	}
	return b
}
func (b Board) findKing(color int) int {
	for i, p := range b.Sq {
		if p == color*king {
			return i
		}
	}
	return -1
}
func (b Board) legalMoves() []Move {
	ps := b.pseudoMoves()
	a := make([]Move, 0, len(ps))
	us := b.Side
	for _, m := range ps {
		c := b.afterUnchecked(m)
		if !c.inCheck(us) {
			a = append(a, m)
		}
	}
	return a
}
func (b Board) pseudoMoves() []Move {
	a := make([]Move, 0, 64)
	for s, pc := range b.Sq {
		if pc == 0 || sign(pc) != b.Side {
			continue
		}
		t := abs(pc)
		x, y := s&7, s>>3
		if t == pawn {
			dy := 1
			if b.Side == black {
				dy = -1
			}
			ny := y + dy
			if ny >= 0 && ny < 8 {
				to := ny*8 + x
				if b.Sq[to] == 0 {
					b.addPawn(&a, s, to, ny)
					home := 1
					if b.Side == black {
						home = 6
					}
					if y == home {
						to2 := (y+2*dy)*8 + x
						if b.Sq[to2] == 0 {
							a = append(a, Move{s, to2, 0})
						}
					}
				}
				for _, dx := range []int{-1, 1} {
					nx := x + dx
					if nx < 0 || nx > 7 {
						continue
					}
					cap := ny*8 + nx
					if (b.Sq[cap] != 0 && sign(b.Sq[cap]) == -b.Side) || cap == b.EP {
						b.addPawn(&a, s, cap, ny)
					}
				}
			}
		} else if t == knight {
			for _, d := range [][2]int{{1, 2}, {2, 1}, {-1, 2}, {-2, 1}, {1, -2}, {2, -1}, {-1, -2}, {-2, -1}} {
				b.addStep(&a, s, x+d[0], y+d[1])
			}
		} else if t == bishop || t == rook || t == queen {
			if t == bishop || t == queen {
				b.ray(&a, s, 1, 1)
				b.ray(&a, s, 1, -1)
				b.ray(&a, s, -1, 1)
				b.ray(&a, s, -1, -1)
			}
			if t == rook || t == queen {
				b.ray(&a, s, 1, 0)
				b.ray(&a, s, -1, 0)
				b.ray(&a, s, 0, 1)
				b.ray(&a, s, 0, -1)
			}
		} else if t == king {
			for dx := -1; dx <= 1; dx++ {
				for dy := -1; dy <= 1; dy++ {
					if dx != 0 || dy != 0 {
						b.addStep(&a, s, x+dx, y+dy)
					}
				}
			}
			b.addCastles(&a, s)
		}
	}
	return a
}
func (b Board) addPawn(a *[]Move, from, to, rank int) {
	if rank == 0 || rank == 7 {
		for _, p := range []int{queen, rook, bishop, knight} {
			*a = append(*a, Move{from, to, p})
		}
	} else {
		*a = append(*a, Move{from, to, 0})
	}
}
func (b Board) addStep(a *[]Move, from, x, y int) {
	if x < 0 || x > 7 || y < 0 || y > 7 {
		return
	}
	to := y*8 + x
	if b.Sq[to] == 0 || sign(b.Sq[to]) == -b.Side {
		*a = append(*a, Move{from, to, 0})
	}
}
func (b Board) ray(a *[]Move, from, dx, dy int) {
	x, y := (from&7)+dx, (from>>3)+dy
	for x >= 0 && x < 8 && y >= 0 && y < 8 {
		to := y*8 + x
		if b.Sq[to] == 0 {
			*a = append(*a, Move{from, to, 0})
		} else {
			if sign(b.Sq[to]) == -b.Side {
				*a = append(*a, Move{from, to, 0})
			}
			break
		}
		x += dx
		y += dy
	}
}
func (b Board) addCastles(a *[]Move, ks int) {
	if b.Side == white && ks == 4 {
		if b.Castle&wk != 0 && b.Sq[5] == 0 && b.Sq[6] == 0 && b.Sq[7] == rook && !b.attacked(4, black) && !b.attacked(5, black) && !b.attacked(6, black) {
			*a = append(*a, Move{4, 6, 0})
		}
		if b.Castle&wq != 0 && b.Sq[3] == 0 && b.Sq[2] == 0 && b.Sq[1] == 0 && b.Sq[0] == rook && !b.attacked(4, black) && !b.attacked(3, black) && !b.attacked(2, black) {
			*a = append(*a, Move{4, 2, 0})
		}
	}
	if b.Side == black && ks == 60 {
		if b.Castle&bk != 0 && b.Sq[61] == 0 && b.Sq[62] == 0 && b.Sq[63] == -rook && !b.attacked(60, white) && !b.attacked(61, white) && !b.attacked(62, white) {
			*a = append(*a, Move{60, 62, 0})
		}
		if b.Castle&bq != 0 && b.Sq[59] == 0 && b.Sq[58] == 0 && b.Sq[57] == 0 && b.Sq[56] == -rook && !b.attacked(60, white) && !b.attacked(59, white) && !b.attacked(58, white) {
			*a = append(*a, Move{60, 58, 0})
		}
	}
}
func (b Board) attacked(s, by int) bool {
	x, y := s&7, s>>3
	pdy := -1
	if by == black {
		pdy = 1
	}
	py := y + pdy
	if py >= 0 && py < 8 {
		for _, dx := range []int{-1, 1} {
			px := x + dx
			if px >= 0 && px < 8 && b.Sq[py*8+px] == by*pawn {
				return true
			}
		}
	}
	for _, d := range [][2]int{{1, 2}, {2, 1}, {-1, 2}, {-2, 1}, {1, -2}, {2, -1}, {-1, -2}, {-2, -1}} {
		nx, ny := x+d[0], y+d[1]
		if nx >= 0 && nx < 8 && ny >= 0 && ny < 8 && b.Sq[ny*8+nx] == by*knight {
			return true
		}
	}
	dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	for i, d := range dirs {
		nx, ny := x+d[0], y+d[1]
		for nx >= 0 && nx < 8 && ny >= 0 && ny < 8 {
			pc := b.Sq[ny*8+nx]
			if pc != 0 {
				if sign(pc) == by {
					t := abs(pc)
					if t == queen || (i < 4 && t == rook) || (i >= 4 && t == bishop) {
						return true
					}
				}
				break
			}
			nx += d[0]
			ny += d[1]
		}
	}
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			nx, ny := x+dx, y+dy
			if nx >= 0 && nx < 8 && ny >= 0 && ny < 8 && b.Sq[ny*8+nx] == by*king {
				return true
			}
		}
	}
	return false
}
func (b Board) inCheck(color int) bool {
	k := b.findKing(color)
	return k >= 0 && b.attacked(k, -color)
}
func (b Board) after(m Move) Board { return b.afterUnchecked(m) }
func (b Board) afterUnchecked(m Move) Board {
	c := b
	pc := c.Sq[m.From]
	us := sign(pc)
	capt := c.Sq[m.To]
	if abs(pc) == pawn && m.To == c.EP && capt == 0 {
		sq := m.To - 8
		if us == black {
			sq = m.To + 8
		}
		c.Sq[sq] = 0
	}
	if m.Promo != 0 {
		c.Sq[m.To] = us * m.Promo
	} else {
		c.Sq[m.To] = pc
	}
	c.Sq[m.From] = 0
	if abs(pc) == king && abs((m.To&7)-(m.From&7)) == 2 {
		if m.To > m.From {
			rf := (m.From & 56) + 7
			rt := m.From + 1
			c.Sq[rt] = c.Sq[rf]
			c.Sq[rf] = 0
		} else {
			rf := m.From & 56
			rt := m.From - 1
			c.Sq[rt] = c.Sq[rf]
			c.Sq[rf] = 0
		}
	}
	if pc == king {
		c.Castle &^= wk | wq
	}
	if pc == -king {
		c.Castle &^= bk | bq
	}
	if m.From == 0 || m.To == 0 {
		c.Castle &^= wq
	}
	if m.From == 7 || m.To == 7 {
		c.Castle &^= wk
	}
	if m.From == 56 || m.To == 56 {
		c.Castle &^= bq
	}
	if m.From == 63 || m.To == 63 {
		c.Castle &^= bk
	}
	c.EP = -1
	if abs(pc) == pawn && abs((m.To>>3)-(m.From>>3)) == 2 {
		c.EP = (m.From + m.To) / 2
	}
	c.Side = -c.Side
	return c
}
func (b Board) san(m Move, legal []Move) string {
	pc := b.Sq[m.From]
	t := abs(pc)
	if t == king && abs((m.To&7)-(m.From&7)) == 2 {
		z := "O-O-O"
		if m.To > m.From {
			z = "O-O"
		}
		c := b.after(m)
		if c.inCheck(c.Side) {
			if len(c.legalMoves()) == 0 {
				z += "#"
			} else {
				z += "+"
			}
		}
		return z
	}
	ep := t == pawn && m.To == b.EP && b.Sq[m.To] == 0
	cap := b.Sq[m.To] != 0 || ep
	var s strings.Builder
	if t != pawn {
		s.WriteString(pieceLetter(t))
		sameFile, sameRank, other := false, false, false
		for _, o := range legal {
			if o.To == m.To && o.From != m.From && abs(b.Sq[o.From]) == t {
				other = true
				if o.From&7 == m.From&7 {
					sameFile = true
				}
				if o.From>>3 == m.From>>3 {
					sameRank = true
				}
			}
		}
		if other {
			if !sameFile {
				s.WriteByte(byte('a' + (m.From & 7)))
			} else if !sameRank {
				s.WriteByte(byte('1' + (m.From >> 3)))
			} else {
				s.WriteString(squareName(m.From))
			}
		}
	} else if cap {
		s.WriteByte(byte('a' + (m.From & 7)))
	}
	if cap {
		s.WriteByte('x')
	}
	s.WriteString(squareName(m.To))
	if m.Promo != 0 {
		s.WriteByte('=')
		s.WriteString(pieceLetter(m.Promo))
	}
	c := b.after(m)
	if c.inCheck(c.Side) {
		if len(c.legalMoves()) == 0 {
			s.WriteByte('#')
		} else {
			s.WriteByte('+')
		}
	}
	return s.String()
}
func pieceLetter(p int) string {
	switch p {
	case knight:
		return "N"
	case bishop:
		return "B"
	case rook:
		return "R"
	case queen:
		return "Q"
	case king:
		return "K"
	}
	return ""
}
func squareName(s int) string { return string([]byte{byte('a' + (s & 7)), byte('1' + (s >> 3))}) }
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
func sign(x int) int {
	if x < 0 {
		return -1
	}
	if x > 0 {
		return 1
	}
	return 0
}

// CB2PGN Builder - neutral ChessBase database to RAW PGN converter.
// Copyright (C) 2026 Source2Metal project contributors.
// SPDX-License-Identifier: GPL-3.0-or-later
//
// The legacy CBH decoder is a Go port of Dominik Klein's cbh2pgn 0.1
// (2022, MIT License), supplied by the user as the real-world reference.
// See THIRD_PARTY_NOTICES.txt and reference_cbh2pgn-0.1/.
package cb2pgn

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const version = "0.1.3"
const progressWidth = 32

// -----------------------------------------------------------------------------
// Common legal chess core. Both CBH and 2CBH routes are forced through this
// board before SAN is emitted. That deliberately avoids the incomplete SAN of
// older experimental CBH converters.
// -----------------------------------------------------------------------------
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
func fileOf(s int) int        { return s & 7 }
func rankOf(s int) int        { return s >> 3 }
func squareName(s int) string { return string([]byte{byte('a' + fileOf(s)), byte('1' + rankOf(s))}) }
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
func (b Board) findKing(c int) int {
	for i, p := range b.Sq {
		if p == c*king {
			return i
		}
	}
	return -1
}
func (b Board) attacked(s, by int) bool {
	x, y := fileOf(s), rankOf(s)
	// pawns: inspect the squares a pawn of 'by' would occupy to attack s.
	py := y - 1
	if by == black {
		py = y + 1
	}
	if py >= 0 && py < 8 {
		for _, dx := range []int{-1, 1} {
			px := x - dx
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
	dirs := [][3]int{{1, 0, rook}, {-1, 0, rook}, {0, 1, rook}, {0, -1, rook}, {1, 1, bishop}, {1, -1, bishop}, {-1, 1, bishop}, {-1, -1, bishop}}
	for _, d := range dirs {
		nx, ny := x+d[0], y+d[1]
		for nx >= 0 && nx < 8 && ny >= 0 && ny < 8 {
			p := b.Sq[ny*8+nx]
			if p != 0 {
				if sign(p) == by && (abs(p) == queen || abs(p) == d[2]) {
					return true
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
func (b Board) inCheck(c int) bool { k := b.findKing(c); return k >= 0 && b.attacked(k, -c) }
func (b Board) afterUnchecked(m Move) Board {
	c := b
	pc := c.Sq[m.From]
	us := sign(pc)
	tgt := c.Sq[m.To]
	if abs(pc) == pawn && m.To == c.EP && tgt == 0 && fileOf(m.From) != fileOf(m.To) {
		cs := m.To - 8
		if us == black {
			cs = m.To + 8
		}
		c.Sq[cs] = 0
	}
	c.Sq[m.To] = pc
	c.Sq[m.From] = 0
	if m.Promo != 0 {
		c.Sq[m.To] = us * m.Promo
	}
	if abs(pc) == king && abs(fileOf(m.To)-fileOf(m.From)) == 2 {
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
	if abs(pc) == pawn && abs(rankOf(m.To)-rankOf(m.From)) == 2 {
		c.EP = (m.From + m.To) / 2
	}
	c.Side = -c.Side
	return c
}
func (b Board) legalMoves() []Move {
	ps := b.pseudoMoves()
	out := make([]Move, 0, len(ps))
	us := b.Side
	for _, m := range ps {
		c := b.afterUnchecked(m)
		if !c.inCheck(us) {
			out = append(out, m)
		}
	}
	return out
}
func (b Board) pseudoMoves() []Move {
	out := make([]Move, 0, 64)
	addStep := func(fr, x, y int) {
		if x < 0 || x > 7 || y < 0 || y > 7 {
			return
		}
		to := y*8 + x
		if b.Sq[to] == 0 || sign(b.Sq[to]) == -b.Side {
			out = append(out, Move{From: fr, To: to})
		}
	}
	ray := func(fr, dx, dy int) {
		x, y := fileOf(fr)+dx, rankOf(fr)+dy
		for x >= 0 && x < 8 && y >= 0 && y < 8 {
			to := y*8 + x
			p := b.Sq[to]
			if p == 0 {
				out = append(out, Move{From: fr, To: to})
			} else {
				if sign(p) == -b.Side {
					out = append(out, Move{From: fr, To: to})
				}
				break
			}
			x += dx
			y += dy
		}
	}
	for fr, pc := range b.Sq {
		if pc == 0 || sign(pc) != b.Side {
			continue
		}
		t := abs(pc)
		x, y := fileOf(fr), rankOf(fr)
		switch t {
		case pawn:
			dy := 1
			home := 1
			prom := 7
			if b.Side == black {
				dy = -1
				home = 6
				prom = 0
			}
			ny := y + dy
			if ny >= 0 && ny < 8 {
				to := ny*8 + x
				if b.Sq[to] == 0 {
					if ny == prom {
						for _, pr := range []int{queen, rook, bishop, knight} {
							out = append(out, Move{fr, to, pr})
						}
					} else {
						out = append(out, Move{From: fr, To: to})
						if y == home {
							to2 := (y+2*dy)*8 + x
							if b.Sq[to2] == 0 {
								out = append(out, Move{From: fr, To: to2})
							}
						}
					}
				}
				for _, dx := range []int{-1, 1} {
					nx := x + dx
					if nx < 0 || nx > 7 {
						continue
					}
					to := ny*8 + nx
					if (b.Sq[to] != 0 && sign(b.Sq[to]) == -b.Side) || to == b.EP {
						if ny == prom {
							for _, pr := range []int{queen, rook, bishop, knight} {
								out = append(out, Move{fr, to, pr})
							}
						} else {
							out = append(out, Move{From: fr, To: to})
						}
					}
				}
			}
		case knight:
			for _, d := range [][2]int{{1, 2}, {2, 1}, {-1, 2}, {-2, 1}, {1, -2}, {2, -1}, {-1, -2}, {-2, -1}} {
				addStep(fr, x+d[0], y+d[1])
			}
		case bishop:
			for _, d := range [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
				ray(fr, d[0], d[1])
			}
		case rook:
			for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				ray(fr, d[0], d[1])
			}
		case queen:
			for _, d := range [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}, {1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				ray(fr, d[0], d[1])
			}
		case king:
			for dx := -1; dx <= 1; dx++ {
				for dy := -1; dy <= 1; dy++ {
					if dx != 0 || dy != 0 {
						addStep(fr, x+dx, y+dy)
					}
				}
			}
			if b.Side == white && fr == 4 && !b.inCheck(white) {
				if b.Castle&wk != 0 && b.Sq[5] == 0 && b.Sq[6] == 0 && b.Sq[7] == rook && !b.attacked(5, black) && !b.attacked(6, black) {
					out = append(out, Move{4, 6, 0})
				}
				if b.Castle&wq != 0 && b.Sq[3] == 0 && b.Sq[2] == 0 && b.Sq[1] == 0 && b.Sq[0] == rook && !b.attacked(3, black) && !b.attacked(2, black) {
					out = append(out, Move{4, 2, 0})
				}
			}
			if b.Side == black && fr == 60 && !b.inCheck(black) {
				if b.Castle&bk != 0 && b.Sq[61] == 0 && b.Sq[62] == 0 && b.Sq[63] == -rook && !b.attacked(61, white) && !b.attacked(62, white) {
					out = append(out, Move{60, 62, 0})
				}
				if b.Castle&bq != 0 && b.Sq[59] == 0 && b.Sq[58] == 0 && b.Sq[57] == 0 && b.Sq[56] == -rook && !b.attacked(59, white) && !b.attacked(58, white) {
					out = append(out, Move{60, 58, 0})
				}
			}
		}
	}
	return out
}
func (b Board) san(m Move, legal []Move) string {
	pc := b.Sq[m.From]
	t := abs(pc)
	if t == king && abs(fileOf(m.To)-fileOf(m.From)) == 2 {
		s := "O-O-O"
		if m.To > m.From {
			s = "O-O"
		}
		c := b.afterUnchecked(m)
		if c.inCheck(c.Side) {
			if len(c.legalMoves()) == 0 {
				s += "#"
			} else {
				s += "+"
			}
		}
		return s
	}
	ep := t == pawn && m.To == b.EP && b.Sq[m.To] == 0 && fileOf(m.From) != fileOf(m.To)
	cap := b.Sq[m.To] != 0 || ep
	var s strings.Builder
	if t != pawn {
		s.WriteString(pieceLetter(t))
		sameFile, sameRank, other := false, false, false
		for _, o := range legal {
			if o.To == m.To && o.From != m.From && abs(b.Sq[o.From]) == t {
				other = true
				if fileOf(o.From) == fileOf(m.From) {
					sameFile = true
				}
				if rankOf(o.From) == rankOf(m.From) {
					sameRank = true
				}
			}
		}
		if other {
			if !sameFile {
				s.WriteByte(byte('a' + fileOf(m.From)))
			} else if !sameRank {
				s.WriteByte(byte('1' + rankOf(m.From)))
			} else {
				s.WriteString(squareName(m.From))
			}
		}
	} else if cap {
		s.WriteByte(byte('a' + fileOf(m.From)))
	}
	if cap {
		s.WriteByte('x')
	}
	s.WriteString(squareName(m.To))
	if m.Promo != 0 {
		s.WriteByte('=')
		s.WriteString(pieceLetter(m.Promo))
	}
	c := b.afterUnchecked(m)
	if c.inCheck(c.Side) {
		if len(c.legalMoves()) == 0 {
			s.WriteByte('#')
		} else {
			s.WriteByte('+')
		}
	}
	return s.String()
}
func matchLegal(b Board, from, to, promo int) (Move, string, error) {
	legal := b.legalMoves()
	for _, m := range legal {
		if m.From == from && m.To == to && (promo == 0 || m.Promo == promo) {
			return m, b.san(m, legal), nil
		}
	}
	return Move{}, "", fmt.Errorf("geen legale zet %s%s", squareName(from), squareName(to))
}

// -----------------------------------------------------------------------------
// Output helpers
// -----------------------------------------------------------------------------
type Stats struct {
	Records, Converted, Skipped, Inactive, Plies, Variations int64
	Reasons                                                  map[string]int64
}

func newStats() Stats               { return Stats{Reasons: map[string]int64{}} }
func (s *Stats) skip(reason string) { s.Skipped++; s.Reasons[reason]++ }
func escTag(s string) string {
	s = strings.ToValidUTF8(strings.TrimSpace(s), "?")
	// PGN tag values must remain single-line text.  Be conservative with
	// control characters even when a damaged/odd legacy record contains them.
	s = strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' || r == '\t' {
			return ' '
		}
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	if s == "" {
		return "?"
	}
	return s
}
func writeGame(w *bufio.Writer, event, site, date, round, wn, bn, res string, we, be int, sans []string) {
	writeGameEx(w, event, site, date, round, wn, bn, res, we, be, sans, "", white, 1)
}

// writeGameEx also supports the non-initial positions that classic CBH can
// contain.  FEN is empty for an ordinary initial position.
func writeGameEx(w *bufio.Writer, event, site, date, round, wn, bn, res string, we, be int, sans []string, fen string, startSide, fullmove int) {
	if event == "" {
		event = "?"
	}
	if site == "" {
		site = "?"
	}
	if date == "" {
		date = "????.??.??"
	}
	if round == "" {
		round = "?"
	}
	if res == "" {
		res = "*"
	}
	if fullmove <= 0 {
		fullmove = 1
	}
	fmt.Fprintf(w, "[Event \"%s\"]\n[Site \"%s\"]\n[Date \"%s\"]\n[Round \"%s\"]\n[White \"%s\"]\n[Black \"%s\"]\n[Result \"%s\"]\n", escTag(event), escTag(site), date, escTag(round), escTag(wn), escTag(bn), res)
	if we > 0 {
		fmt.Fprintf(w, "[WhiteElo \"%d\"]\n", we)
	}
	if be > 0 {
		fmt.Fprintf(w, "[BlackElo \"%d\"]\n", be)
	}
	if fen != "" {
		fmt.Fprintln(w, "[SetUp \"1\"]")
		fmt.Fprintf(w, "[FEN \"%s\"]\n", escTag(fen))
	}
	fmt.Fprintf(w, "[PlyCount \"%d\"]\n\n", len(sans))
	toks := make([]string, 0, len(sans)*2+1)
	side := startSide
	mn := fullmove
	for i, x := range sans {
		if side == white {
			toks = append(toks, fmt.Sprintf("%d. %s", mn, x))
		} else {
			if i == 0 {
				toks = append(toks, fmt.Sprintf("%d... %s", mn, x))
			} else {
				toks = append(toks, x)
			}
			mn++
		}
		side = -side
	}
	toks = append(toks, res)
	writeWrapped(w, toks, 100)
	fmt.Fprint(w, "\n\n")
}
func writeWrapped(w *bufio.Writer, toks []string, width int) {
	col := 0
	for _, t := range toks {
		n := utf8.RuneCountInString(t)
		if col == 0 {
			fmt.Fprint(w, t)
			col = n
		} else if col+1+n > width {
			fmt.Fprint(w, "\n", t)
			col = n
		} else {
			fmt.Fprint(w, " ", t)
			col += 1 + n
		}
	}
}
func fmtInt(n int64) string {
	s := strconv.FormatInt(n, 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "." + s[i:]
	}
	return s
}
func showProgress(label string, done, total int64, start time.Time) {
	if total <= 0 {
		return
	}
	// In standalone CB2PGN the original one-line renderer is preserved.
	// Source2Metal can install a progress sink so the integrated adapter does
	// not write directly to the console; this keeps the decoder logic intact
	// while allowing Source2Metal to own resize-safe rendering.
	if integratedProgress != nil {
		integratedProgress(label, done, total, start)
		return
	}
	frac := float64(done) / float64(total)
	if frac > 1 {
		frac = 1
	}
	f := int(frac * progressWidth)
	rate := float64(done) / maxf(time.Since(start).Seconds(), .001)
	eta := "--:--"
	if rate > 0 && done < total {
		eta = fmtDur(time.Duration(float64(total-done) / rate * float64(time.Second)))
	}
	fmt.Printf("\r  %-8s [%s%s] %5.1f%% | %s/%s | ETA %s", label, strings.Repeat("#", f), strings.Repeat("-", progressWidth-f), frac*100, fmtInt(done), fmtInt(total), eta)
}
func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
func fmtDur(d time.Duration) string {
	s := int64(d.Seconds() + .5)
	if s < 0 {
		return "--:--"
	}
	h := s / 3600
	m := (s % 3600) / 60
	z := s % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, z)
	}
	return fmt.Sprintf("%02d:%02d", m, z)
}
func shaFile(path string) (string, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	h := sha256.New()
	if _, e = io.Copy(h, f); e != nil {
		return "", e
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// -----------------------------------------------------------------------------
// 2CBH route - reverse-engineered 16-bit move coding, validated in the earlier
// 2cbh2pgn v0.1.1 paired-format test.
// -----------------------------------------------------------------------------
var blocks = map[string]int{"wK": 0, "wQ": 420, "wN": 1876, "wB": 2212, "wR": 2772, "bK": 3668, "bQ": 4088, "bN": 5544, "bB": 5880, "bR": 6440}
var capCode = map[int]int{0: 0, queen: 1, knight: 2, bishop: 3, rook: 4, pawn: 5}
var localIndex map[int]map[[2]int]int

func initLocal() {
	localIndex = map[int]map[[2]int]int{}
	for _, pt := range []int{king, queen, knight, bishop, rook} {
		m := map[[2]int]int{}
		idx := 0
		for f := 0; f < 8; f++ {
			for r := 0; r < 8; r++ {
				fr := r*8 + f
				for _, to := range geomTargets(pt, fr) {
					m[[2]int{fr, to}] = idx
					idx++
				}
			}
		}
		localIndex[pt] = m
	}
}
func geomTargets(pt, fr int) []int {
	f, r := fileOf(fr), rankOf(fr)
	out := []int{}
	ray := func(df, dr int) {
		ff, rr := f+df, r+dr
		for ff >= 0 && ff < 8 && rr >= 0 && rr < 8 {
			out = append(out, rr*8+ff)
			ff += df
			rr += dr
		}
	}
	if pt == king {
		for df := -1; df <= 1; df++ {
			for dr := -1; dr <= 1; dr++ {
				if df == 0 && dr == 0 {
					continue
				}
				ff, rr := f+df, r+dr
				if ff >= 0 && ff < 8 && rr >= 0 && rr < 8 {
					out = append(out, rr*8+ff)
				}
			}
		}
		sort.Slice(out, func(i, j int) bool {
			if fileOf(out[i]) != fileOf(out[j]) {
				return fileOf(out[i]) < fileOf(out[j])
			}
			return rankOf(out[i]) < rankOf(out[j])
		})
		return out
	}
	if pt == knight {
		for _, d := range [][2]int{{-2, -1}, {-2, 1}, {2, -1}, {2, 1}, {-1, -2}, {-1, 2}, {1, -2}, {1, 2}} {
			ff, rr := f+d[0], r+d[1]
			if ff >= 0 && ff < 8 && rr >= 0 && rr < 8 {
				out = append(out, rr*8+ff)
			}
		}
		return out
	}
	if pt == bishop || pt == queen {
		for _, d := range [][2]int{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
			ray(d[0], d[1])
		}
	}
	if pt == rook || pt == queen {
		for _, d := range [][2]int{{-1, 0}, {0, -1}, {1, 0}, {0, 1}} {
			ray(d[0], d[1])
		}
	}
	return out
}

var fileStart = []int{0, 52, 146, 240, 334, 428, 522, 616}

func srcSizes(c int, f int) []int {
	edge := f == 0 || f == 7
	if c == white {
		if edge {
			return []int{7, 6, 6, 7, 6, 20}
		}
		return []int{12, 11, 11, 13, 11, 36}
	}
	if edge {
		return []int{20, 6, 7, 6, 6, 7}
	}
	return []int{36, 11, 13, 11, 11, 12}
}
func pawnSourceStart(c, fr int) (int, error) {
	f, r := fileOf(fr), rankOf(fr)
	if r < 1 || r > 6 {
		return 0, errors.New("pawn source rank")
	}
	base := 0xABF1
	if c == black {
		base = 0xAE8D
	}
	x := base + fileStart[f]
	sz := srcSizes(c, f)
	for i := 0; i < r-1; i++ {
		x += sz[i]
	}
	return x, nil
}
func idxInt(a []int, x int) int {
	for i, v := range a {
		if v == x {
			return i
		}
	}
	return -1
}
func pawnToken(c int, b Board, m Move) (uint16, error) {
	fr, to := m.From, m.To
	f, r, tf, tr := fileOf(fr), rankOf(fr), fileOf(to), rankOf(to)
	df, dr := tf-f, tr-r
	start, e := pawnSourceStart(c, fr)
	if e != nil {
		return 0, e
	}
	dir := 1
	if c == black {
		dir = -1
	}
	target := b.Sq[to]
	ep := to == b.EP && target == 0 && df != 0
	cap := abs(target)
	if ep {
		cap = pawn
	}
	initial := (c == white && r == 1) || (c == black && r == 6)
	eprank := (c == white && r == 4) || (c == black && r == 3)
	promrank := (c == white && r == 6) || (c == black && r == 1)
	if promrank {
		pi := idxInt([]int{queen, knight, bishop, rook}, m.Promo)
		if pi < 0 {
			return 0, errors.New("bad promotion")
		}
		if df == 0 && dr == dir {
			return uint16(start + pi), nil
		}
		if dr != dir || abs(df) != 1 {
			return 0, errors.New("bad promotion geometry")
		}
		ci := idxInt([]int{queen, knight, bishop, rook}, cap)
		if ci < 0 {
			return 0, errors.New("bad promotion capture")
		}
		dirs := []int{}
		if f > 0 {
			dirs = append(dirs, -1)
		}
		if f < 7 {
			dirs = append(dirs, 1)
		}
		di := idxInt(dirs, df)
		return uint16(start + 4 + di*16 + ci*4 + pi), nil
	}
	base := 1
	if initial {
		if df == 0 && dr == 2*dir {
			return uint16(start), nil
		}
		if df == 0 && dr == dir {
			return uint16(start + 1), nil
		}
		base = 2
	} else if df == 0 && dr == dir {
		return uint16(start), nil
	}
	if dr != dir || abs(df) != 1 {
		return 0, errors.New("bad pawn geometry")
	}
	dirs := []int{}
	if f > 0 {
		dirs = append(dirs, -1)
	}
	if f < 7 {
		dirs = append(dirs, 1)
	}
	di := idxInt(dirs, df)
	if di < 0 {
		return 0, errors.New("bad pawn dir")
	}
	if eprank {
		if ep {
			return uint16(start + base + di*6 + 5), nil
		}
		ci := idxInt([]int{queen, knight, bishop, rook, pawn}, cap)
		if ci < 0 {
			return 0, errors.New("bad pawn capture")
		}
		return uint16(start + base + di*6 + ci), nil
	}
	ci := idxInt([]int{queen, knight, bishop, rook, pawn}, cap)
	if ci < 0 {
		return 0, errors.New("bad pawn capture")
	}
	return uint16(start + base + di*5 + ci), nil
}
func encode2Move(b Board, m Move) (uint16, error) {
	p := b.Sq[m.From]
	if p == 0 {
		return 0, errors.New("empty source")
	}
	c := sign(p)
	pt := abs(p)
	if pt == king && abs(fileOf(m.To)-fileOf(m.From)) == 2 {
		if c == white {
			if m.To > m.From {
				return 0xB12A, nil
			}
			return 0xB129, nil
		}
		if m.To > m.From {
			return 0xB12C, nil
		}
		return 0xB12B, nil
	}
	if pt == pawn {
		return pawnToken(c, b, m)
	}
	li, ok := localIndex[pt][[2]int{m.From, m.To}]
	if !ok {
		return 0, errors.New("geometry not encodable")
	}
	letter := map[int]string{king: "K", queen: "Q", knight: "N", bishop: "B", rook: "R"}[pt]
	col := "w"
	if c == black {
		col = "b"
	}
	base := 1 + 6*(blocks[col+letter]+li)
	target := abs(b.Sq[m.To])
	cc, ok := capCode[target]
	if !ok {
		return 0, errors.New("capture not encodable")
	}
	return uint16(base + cc), nil
}
func decode2Token(b Board, t uint16) (Move, string, error) {
	var found []Move
	legal := b.legalMoves()
	for _, m := range legal {
		x, e := encode2Move(b, m)
		if e == nil && x == t {
			found = append(found, m)
		}
	}
	if len(found) != 1 {
		return Move{}, "", fmt.Errorf("token 0x%04X matcht %d legale zetten", t, len(found))
	}
	m := found[0]
	return m, b.san(m, legal), nil
}

const h2rec = 192
const lidHeader = 184
const lidRec = 4234

func readAt(f *os.File, off int64, n int) ([]byte, error) {
	b := make([]byte, n)
	_, e := f.ReadAt(b, off)
	if e != nil && e != io.EOF {
		return nil, e
	}
	return b, nil
}
func lidName(f *os.File, id uint64, cache map[uint64]string) string {
	if s, ok := cache[id]; ok {
		return s
	}
	b, e := readAt(f, int64(lidHeader)+int64(id)*lidRec, lidRec)
	if e != nil || len(b) < 12 {
		return "?"
	}
	l1 := int(binary.LittleEndian.Uint32(b[4:8]))
	if l1 < 0 || 8+l1+4 > len(b) {
		return "?"
	}
	p := 8
	n1 := string(b[p : p+l1])
	p += l1
	l2 := int(binary.LittleEndian.Uint32(b[p : p+4]))
	p += 4
	n2 := ""
	if l2 >= 0 && p+l2 <= len(b) {
		n2 = string(b[p : p+l2])
	}
	n1 = strings.TrimSpace(strings.ToValidUTF8(n1, "?"))
	n2 = strings.TrimSpace(strings.ToValidUTF8(n2, "?"))
	s := n1
	if n2 != "" {
		s = n1 + ", " + n2
	}
	if s == "" {
		s = "?"
	}
	cache[id] = s
	return s
}
func date2(rec []byte) string {
	x := int(rec[188]) | int(rec[189])<<8 | int(rec[190])<<16
	y := x >> 9
	m := (x >> 5) & 15
	d := x & 31
	if y < 1 || m < 1 || m > 12 || d < 1 || d > 31 {
		return "????.??.??"
	}
	return fmt.Sprintf("%04d.%02d.%02d", y, m, d)
}
func resultCode(c byte) string {
	switch c {
	case 2:
		return "1-0"
	case 1:
		return "1/2-1/2"
	case 0:
		return "0-1"
	}
	return "*"
}
func convert2(root, out string) (Stats, error) {
	st := newStats()
	hf, e := os.Open(root + ".2cbh")
	if e != nil {
		return st, e
	}
	defer hf.Close()
	gf, e := os.Open(root + ".2cbg")
	if e != nil {
		return st, e
	}
	defer gf.Close()
	lf, e := os.Open(root + ".2lid")
	if e != nil {
		return st, e
	}
	defer lf.Close()
	hs, _ := hf.Stat()
	if hs.Size()%h2rec != 0 || hs.Size() < 2*h2rec {
		return st, fmt.Errorf("ongeldige .2cbh grootte")
	}
	st.Records = hs.Size()/h2rec - 1
	of, e := os.Create(out)
	if e != nil {
		return st, e
	}
	defer of.Close()
	w := bufio.NewWriterSize(of, 1<<20)
	defer w.Flush()
	cache := map[uint64]string{}
	start := time.Now()
	last := time.Time{}
	for i := int64(1); i <= st.Records; i++ {
		rec, e := readAt(hf, i*h2rec, h2rec)
		if e != nil {
			return st, e
		}
		if rec[0]&1 == 0 {
			st.skip("geen actieve partij")
			continue
		}
		off := binary.LittleEndian.Uint64(rec[8:16])
		head, e := readAt(gf, int64(off), 28)
		if e != nil {
			st.skip("2CBG-offset buiten bestand")
			continue
		}
		if !equal(head[:8], []byte{0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11}) {
			st.skip("onbekende 2CBG-header")
			continue
		}
		a := int(binary.LittleEndian.Uint32(head[8:12]))
		if a < 4 || a%2 != 0 {
			st.skip("ongeldige zettenlengte")
			continue
		}
		if !equal(head[24:28], []byte{1, 0, 0xfc, 0xff}) {
			st.skip("niet-standaard beginstelling")
			continue
		}
		n := a - 4
		stream, e := readAt(gf, int64(off)+28, n)
		if e != nil {
			st.skip("zettenstroom buiten 2CBG")
			continue
		}
		b := startBoard()
		sans := make([]string, 0, n/2)
		ok := true
		for p := 0; p+1 < len(stream); p += 2 {
			t := binary.LittleEndian.Uint16(stream[p : p+2])
			m, s, e := decode2Token(b, t)
			if e != nil {
				st.skip("zetdecoder 2CBH")
				ok = false
				break
			}
			sans = append(sans, s)
			b = b.afterUnchecked(m)
		}
		if !ok {
			continue
		}
		wn := lidName(lf, binary.LittleEndian.Uint64(rec[24:32]), cache)
		bn := lidName(lf, binary.LittleEndian.Uint64(rec[32:40]), cache)
		we := int(binary.LittleEndian.Uint16(rec[96:98]))
		be := int(binary.LittleEndian.Uint16(rec[112:114]))
		res := resultCode(rec[88])
		writeGame(w, "?", "?", date2(rec), "?", wn, bn, res, we, be, sans)
		st.Converted++
		st.Plies += int64(len(sans))
		if time.Since(last) > 400*time.Millisecond {
			showProgress("2CBH", i, st.Records, start)
			last = time.Now()
		}
	}
	showProgress("2CBH", st.Records, st.Records, start)
	fmt.Println()
	return st, nil
}
func equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// -----------------------------------------------------------------------------
// Legacy CBH route: see legacy_cbh.go (faithful port of cbh2pgn 0.1).
// -----------------------------------------------------------------------------
func trimCB(s []byte) string {
	// cbh2pgn-0.1 decodes fixed-width CBP/CBT fields and then splits at
	// the FIRST NUL byte.  v0.1.2 only trimmed NUL/0xFE at the right edge;
	// that could leak binary padding/control bytes into PGN tag values and
	// confuse strict PGN parsers.  Keep this deliberately faithful here.
	if i := bytes.IndexByte(s, 0); i >= 0 {
		s = s[:i]
	}
	b := bytesTrimRight(s)
	r := make([]rune, 0, len(b))
	for _, c := range b {
		r = append(r, rune(c)) // CBP/CBT strings are ISO-8859-1 in cbh2pgn 0.1.
	}
	return strings.TrimSpace(string(r))
}
func bytesTrimRight(s []byte) []byte {
	i := len(s)
	for i > 0 && (s[i-1] == 0 || s[i-1] == 0xfe) {
		i--
	}
	return s[:i]
}
func read24be(b []byte) int { return int(b[0])<<16 | int(b[1])<<8 | int(b[2]) }
func cbName(f *os.File, id int, cache map[int]string) string {
	if s, ok := cache[id]; ok {
		return s
	}
	v := make([]byte, 25)
	if _, e := f.ReadAt(v, 0); e != nil && e != io.EOF {
		return "?"
	}
	base := 28
	if len(v) > 24 && v[0x18] == 4 {
		base = 32
	}
	rec, e := readAt(f, int64(base+id*67), 67)
	if e != nil {
		return "?"
	}
	ln := trimCB(rec[9:39])
	fn := trimCB(rec[39:59])
	s := ln
	if fn != "" {
		s = ln + ", " + fn
	}
	if s == "" {
		s = "?"
	}
	cache[id] = s
	return s
}
func cbEvent(f *os.File, id int, cache map[int][2]string) (string, string) {
	if s, ok := cache[id]; ok {
		return s[0], s[1]
	}
	v := make([]byte, 25)
	_, _ = f.ReadAt(v, 0)
	base := 28
	if len(v) > 24 && v[0x18] == 4 {
		base = 32
	}
	rec, e := readAt(f, int64(base+id*99), 99)
	if e != nil {
		return "?", "?"
	}
	x := [2]string{trimCB(rec[9:49]), trimCB(rec[49:79])}
	cache[id] = x
	return x[0], x[1]
}
func cbDate(rec []byte) string {
	x := read24be(rec[24:27])
	y := x >> 9
	m := (x >> 5) & 15
	d := x & 31
	if y < 1 {
		return "????.??.??"
	}
	ms := "??"
	ds := "??"
	if m >= 1 && m <= 12 {
		ms = fmt.Sprintf("%02d", m)
	}
	if d >= 1 && d <= 31 {
		ds = fmt.Sprintf("%02d", d)
	}
	return fmt.Sprintf("%04d.%s.%s", y, ms, ds)
}
func convertCBH(root, out string) (Stats, error) {
	st := newStats()
	hf, e := os.Open(root + ".cbh")
	if e != nil {
		return st, e
	}
	defer hf.Close()
	gf, e := os.Open(root + ".cbg")
	if e != nil {
		return st, e
	}
	defer gf.Close()
	pf, e := os.Open(root + ".cbp")
	if e != nil {
		return st, e
	}
	defer pf.Close()
	tf, e := os.Open(root + ".cbt")
	if e != nil {
		return st, e
	}
	defer tf.Close()

	hs, _ := hf.Stat()
	if hs.Size()%46 != 0 {
		return st, fmt.Errorf("ongeldige .cbh grootte")
	}
	st.Records = hs.Size() / 46
	if st.Records < 2 {
		return st, fmt.Errorf("CBH bevat geen partijrecords")
	}

	of, e := os.Create(out)
	if e != nil {
		return st, e
	}
	defer of.Close()
	w := bufio.NewWriterSize(of, 1<<20)
	defer w.Flush()

	pc := map[int]string{}
	tc := map[int][2]string{}
	start := time.Now()
	last := time.Time{}

	// Record 0 is the 46-byte database header, exactly as in cbh2pgn 0.1.
	st.Inactive = 1
	for i := int64(1); i < st.Records; i++ {
		rec, e := readAt(hf, i*46, 46)
		if e != nil {
			return st, e
		}
		if rec[0]&1 == 0 || rec[0]&0x80 != 0 {
			st.Inactive++
			continue
		}

		off := int64(binary.BigEndian.Uint32(rec[1:5]))
		head, e := readAt(gf, off, 4)
		if e != nil {
			st.skip("CBG-offset buiten bestand")
			continue
		}
		info, e := parseLegacyGameInfo(head)
		if e != nil {
			st.skip("CBG-header ongeldig")
			continue
		}
		if info.length < 4 {
			st.skip("ongeldige CBG-lengte")
			continue
		}
		if info.notEncoded {
			st.skip("CBH niet-gecodeerde/speciale gameflag")
			continue
		}
		if info.special {
			st.skip("speciale CBH-codering")
			continue
		}
		if info.is960 {
			st.skip("Chess960-codering")
			continue
		}

		var b Board
		var ls legacyRefState
		fen := ""
		startSide := white
		fullmove := 1
		streamOff := off + 4
		streamLen := info.length - 4
		if info.customStart {
			setup, er := readAt(gf, off+4, 28)
			if er != nil {
				st.skip("CBH beginstelling buiten bestand")
				continue
			}
			ls, b, fen, startSide, fullmove, er = decodeLegacyCustomStart(setup)
			if er != nil {
				st.skip("CBH beginstelling: " + shortReason(er.Error()))
				continue
			}
			streamOff = off + 32
			streamLen = info.length - 32
			if streamLen < 0 {
				st.skip("ongeldige CBG-lengte bij beginstelling")
				continue
			}
		} else {
			ls = newLegacyRefState()
			b = startBoard()
		}

		stream, er := readAt(gf, streamOff, streamLen)
		if er != nil {
			st.skip("CBG-stroom buiten bestand")
			continue
		}
		dec, er := decodeLegacyReference(stream, b, ls, startSide, fullmove)
		st.Variations += dec.variations
		if er != nil {
			st.skip("CBH zetdecoder: " + shortReason(er.Error()))
			continue
		}
		if fen != "" {
			dec.fen = fen
		}

		wid := read24be(rec[9:12])
		bid := read24be(rec[12:15])
		tid := read24be(rec[15:18])
		wn := cbName(pf, wid, pc)
		bn := cbName(pf, bid, pc)
		event, site := cbEvent(tf, tid, tc)
		res := resultCode(rec[27])
		r := fmt.Sprintf("%d", rec[29])
		if rec[30] != 0 {
			r = fmt.Sprintf("%d(%d)", rec[29], rec[30])
		}
		we := int(binary.BigEndian.Uint16(rec[31:33]))
		be := int(binary.BigEndian.Uint16(rec[33:35]))
		writeGameEx(w, event, site, cbDate(rec), r, wn, bn, res, we, be, dec.sans, dec.fen, dec.startSide, dec.fullmove)
		st.Converted++
		st.Plies += int64(len(dec.sans))

		if time.Since(last) > 400*time.Millisecond {
			showProgress("CBH", i+1, st.Records, start)
			last = time.Now()
		}
	}
	showProgress("CBH", st.Records, st.Records, start)
	fmt.Println()
	return st, nil
}
func shortReason(s string) string {
	if len(s) > 70 {
		return s[:70]
	}
	return s
}

// -----------------------------------------------------------------------------
// Builder shell
// -----------------------------------------------------------------------------
type DBSet struct{ root, format string }

func exists(p string) bool { _, e := os.Stat(p); return e == nil }
func validSet(root, format string) bool {
	if format == "2CBH" {
		return exists(root+".2cbh") && exists(root+".2cbg") && exists(root+".2lid")
	}
	return exists(root+".cbh") && exists(root+".cbg") && exists(root+".cbp") && exists(root+".cbt")
}
func discover(dir string) []DBSet {
	ents, _ := os.ReadDir(dir)
	seen := map[string]bool{}
	var a []DBSet
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		format := ""
		if ext == ".2cbh" {
			format = "2CBH"
		} else if ext == ".cbh" {
			format = "CBH"
		} else {
			continue
		}
		root := filepath.Join(dir, strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())))
		key := format + "|" + strings.ToLower(root)
		if !seen[key] && validSet(root, format) {
			seen[key] = true
			a = append(a, DBSet{root, format})
		}
	}
	sort.Slice(a, func(i, j int) bool {
		return strings.ToLower(filepath.Base(a[i].root)+a[i].format) < strings.ToLower(filepath.Base(a[j].root)+a[j].format)
	})
	return a
}
func setFromArg(p string) (DBSet, error) {
	p, e := filepath.Abs(p)
	if e != nil {
		return DBSet{}, e
	}
	ext := strings.ToLower(filepath.Ext(p))
	root := strings.TrimSuffix(p, filepath.Ext(p))
	format := ""
	if strings.HasPrefix(ext, ".2") {
		format = "2CBH"
	} else {
		format = "CBH"
	}
	if !validSet(root, format) {
		return DBSet{}, fmt.Errorf("onvolledige %s-bestandenset rond %s", format, filepath.Base(root))
	}
	return DBSet{root, format}, nil
}
func choose(a []DBSet) (DBSet, error) {
	if len(a) == 0 {
		return DBSet{}, errors.New("geen complete CBH- of 2CBH-database gevonden")
	}
	if len(a) == 1 {
		return a[0], nil
	}
	fmt.Println("Meerdere ChessBase-databases gevonden:")
	for i, s := range a {
		fmt.Printf("  %d = %-5s  %s\n", i+1, s.format, filepath.Base(s.root))
	}
	fmt.Print("Keuze: ")
	r := bufio.NewReader(os.Stdin)
	x, _ := r.ReadString('\n')
	n, e := strconv.Atoi(strings.TrimSpace(x))
	if e != nil || n < 1 || n > len(a) {
		return DBSet{}, errors.New("ongeldige keuze")
	}
	return a[n-1], nil
}
func writeReport(path string, set DBSet, st Stats, elapsed time.Duration, outSha string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "CB2PGN Builder v%s - rapport\n\n", version)
	status := "KLAAR"
	if st.Converted == 0 && st.Records > 0 {
		status = "MISLUKT - geen enkele partij geconverteerd"
	}
	fmt.Fprintf(&b, "Bron      : %s\nFormaat   : %s\nStatus    : %s\nDuur      : %s\n\n", filepath.Base(set.root), set.format, status, fmtDur(elapsed))
	fmt.Fprintf(&b, "Records fysiek : %d\nNiet-actief/header: %d\nGeconverteerd  : %d\nOvergeslagen   : %d\nPly            : %d\nVariatietakken genegeerd: %d\n", st.Records, st.Inactive, st.Converted, st.Skipped, st.Plies, st.Variations)
	if len(st.Reasons) > 0 {
		fmt.Fprintln(&b, "\nOVERGESLAGEN - REDENEN")
		keys := make([]string, 0, len(st.Reasons))
		for k := range st.Reasons {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "%-72s %d\n", k, st.Reasons[k])
		}
	}
	fmt.Fprintf(&b, "\nRAW-PGN SHA-256: %s\n\n", outSha)
	fmt.Fprintln(&b, "Principes: hoofdvariant-only, RAW blijft neutraal, geen Metal-selectie, geen leerweging.")
	fmt.Fprintln(&b, "Alle geschreven zetten zijn opnieuw op een legale schaakpositie gecontroleerd en als SAN opgebouwd.")
	return os.WriteFile(path, []byte(b.String()), 0644)
}

const manual = `CB2PGN BUILDER - KORTE HANDLEIDING
==================================

Doel:
  ChessBase CBH of 2CBH -> neutrale RAW-PGN.

Ondersteunde invoer in v0.1.3:
  Oude CBH-set:  .cbh + .cbg + .cbp + .cbt
  Nieuwe 2CBH:   .2cbh + .2cbg + .2lid

Werkwijze:
  1. Zet CB2PGN_Builder.exe in de map met de database en start hem.
  2. Bij meerdere databases kiest u de gewenste set.
  3. De uitvoer komt in CB2PGN_output\<tijdstempel>.
  4. Gebruik *_Raw.pgn daarna desgewenst als invoer voor Pgn2Metal.

Belangrijk:
  - De bronbestanden worden alleen gelezen.
  - RAW wordt niet gemetaliseerd, gewogen of op winstpartijen gefilterd.
  - Alleen de hoofdvariant wordt geschreven; CBH-analysevarianten worden genegeerd.
  - Niet-standaard beginstellingen, Chess960 en speciale onbekende coderingen worden veilig overgeslagen en in het rapport geteld.
  - De oude CBH-route is in v0.1.3 opnieuw opgebouwd als zo letterlijk mogelijke Go-port van Dominik Klein's cbh2pgn 0.1: token=(raw-processedMoves) mod 256, de originele 234 één-byte tabellen, 0x29-tweebytezetten en de originele stuknummering na captures.
  - De originele MIT-gelicentieerde cbh2pgn-0.1 bron staat als referentie in de afzonderlijk downloadbare broncode.
  - Deze CBH-route blijft experimenteel totdat StrongGames2017 en een echte CBH/PGN-paarvergelijking slagen.
  - De 2CBH-route bouwt voort op de eerder exact 43/43 partijen en 3119/3119 ply gevalideerde decoder.

Commando's:
  CB2PGN_Builder.exe <pad-naar-cbh-of-2cbh-bestand>
  CB2PGN_Builder.exe --selftest
`

func selfTest() error {
	initLocal()
	b := startBoard()
	if len(b.legalMoves()) != 20 {
		return fmt.Errorf("beginstelling %d zetten", len(b.legalMoves()))
	}

	// cbh2pgn 0.1 reference token stream: token=(raw-processed) mod 256.
	rawFor := func(tok byte, processed int) byte { return byte((int(tok) + processed) & 255) }
	decode := func(stream []byte) (legacyDecoded, error) {
		return decodeLegacyReference(stream, startBoard(), newLegacyRefState(), white, 1)
	}

	// 1.e4 e5 2.Nf3 Nc6 using the literal one-byte token tables from cbh2pgn.
	toks := []byte{0xFF, 0xFF, 0xFE, 0xDD}
	stream := make([]byte, 0, len(toks))
	processed := 0
	for _, t := range toks {
		stream = append(stream, rawFor(t, processed))
		processed = (processed + 1) & 255
	}
	dec, e := decode(stream)
	if e != nil {
		return fmt.Errorf("CBH reference decoder: %w", e)
	}
	want := []string{"e4", "e5", "Nf3", "Nc6"}
	if len(dec.sans) != len(want) {
		return fmt.Errorf("CBH selftest %d zetten", len(dec.sans))
	}
	for i := range want {
		if dec.sans[i] != want[i] {
			return fmt.Errorf("CBH selftest ply %d: %s != %s", i+1, dec.sans[i], want[i])
		}
	}

	// Variation: 1.e4 (1...c5) 1...e5 2.Nf3 Nc6.  The branch must be decoded
	// fully and restored, while its move still advances processedMoves.
	vs := []byte{}
	processed = 0
	vs = append(vs, rawFor(0xFF, processed))
	processed++                              // e4
	vs = append(vs, rawFor(0xDC, processed)) // push, special => no increment
	vs = append(vs, rawFor(0xDA, processed))
	processed++                              // ...c5
	vs = append(vs, rawFor(0x0C, processed)) // pop
	vs = append(vs, rawFor(0xFF, processed))
	processed++ // ...e5
	vs = append(vs, rawFor(0xFE, processed))
	processed++ // Nf3
	vs = append(vs, rawFor(0xDD, processed))
	processed++ // ...Nc6
	vdec, ve := decode(vs)
	if ve != nil || vdec.variations != 1 || len(vdec.sans) != 4 {
		return fmt.Errorf("CBH variatietest: %v takken=%d zetten=%d", ve, vdec.variations, len(vdec.sans))
	}
	for i := range want {
		if vdec.sans[i] != want[i] {
			return fmt.Errorf("CBH variatietest ply %d: %s != %s", i+1, vdec.sans[i], want[i])
		}
	}

	// Verify that the literal port contains every token from the reference tables.
	if len(legacyOneByte) != 234 {
		return fmt.Errorf("CBH tokenmap bevat %d i.p.v. 234 codes", len(legacyOneByte))
	}

	// 2CBH encode/decode round trip remains unchanged.
	bb := startBoard()
	for _, u := range [][2]int{{12, 28}, {52, 36}, {6, 21}, {57, 42}} {
		m, san, e := matchLegal(bb, u[0], u[1], 0)
		if e != nil {
			return e
		}
		t, e := encode2Move(bb, m)
		if e != nil {
			return e
		}
		mm, ss, e := decode2Token(bb, t)
		if e != nil || mm != m || ss != san {
			return errors.New("2CBH roundtrip mismatch")
		}
		bb = bb.afterUnchecked(m)
	}
	return nil
}

func main() {
	initLocal()
	fmt.Printf("CB2PGN Builder v%s - ChessBase -> neutrale RAW-PGN\n", version)
	fmt.Println("Ondersteunt CBH en 2CBH; geen Metal-filtering of boekleren.")
	fmt.Println()
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "--selftest" {
		if e := selfTest(); e != nil {
			fmt.Println("SELFTEST FOUT:", e)
			os.Exit(1)
		}
		fmt.Println("SELFTEST OK - Dominik-Klein CBH-port, variatiestack, legaal bord en 2CBH roundtrip.")
		return
	}
	var set DBSet
	var e error
	if len(args) > 0 {
		set, e = setFromArg(args[0])
	} else {
		set, e = choose(discover(dir))
	}
	if e != nil {
		fmt.Println("FOUT:", e)
		pause(len(args) > 0)
		return
	}
	stamp := time.Now().Format("2006-01-02_15-04-05")
	outdir := filepath.Join(dir, "CB2PGN_output", stamp)
	if e = os.MkdirAll(outdir, 0755); e != nil {
		fmt.Println("FOUT:", e)
		pause(len(args) > 0)
		return
	}
	raw := filepath.Join(outdir, filepath.Base(set.root)+"_Raw.pgn")
	fmt.Println("Database :", filepath.Base(set.root), "(", set.format, ")")
	fmt.Println("Uitvoer  :", raw)
	start := time.Now()
	var st Stats
	if set.format == "2CBH" {
		st, e = convert2(set.root, raw)
	} else {
		st, e = convertCBH(set.root, raw)
	}
	if e != nil {
		fmt.Println("FOUT:", e)
		pause(len(args) > 0)
		return
	}
	sha, e := shaFile(raw)
	if e != nil {
		fmt.Println("FOUT SHA:", e)
		pause(len(args) > 0)
		return
	}
	_ = writeReport(filepath.Join(outdir, "CB2PGN_report.txt"), set, st, time.Since(start), sha)
	_ = os.WriteFile(filepath.Join(outdir, "HANDLEIDING_CB2PGN.txt"), []byte(manual), 0644)
	if st.Converted == 0 && st.Records > 0 {
		fmt.Printf("MISLUKT: 0/%s partijen geconverteerd | %s overgeslagen | %s.\n", fmtInt(st.Records), fmtInt(st.Skipped), fmtDur(time.Since(start)))
	} else {
		fmt.Printf("Klaar: %s/%s partijen | %s ply | %s overgeslagen | %s.\n", fmtInt(st.Converted), fmtInt(st.Records), fmtInt(st.Plies), fmtInt(st.Skipped), fmtDur(time.Since(start)))
	}
	fmt.Println("RAW-PGN :", raw)
	fmt.Println("Rapport :", filepath.Join(outdir, "CB2PGN_report.txt"))
	pause(len(args) > 0)
}
func pause(no bool) {
	if no {
		return
	}
	fmt.Print("Druk Enter om af te sluiten...")
	bufio.NewReader(os.Stdin).ReadString('\n')
}

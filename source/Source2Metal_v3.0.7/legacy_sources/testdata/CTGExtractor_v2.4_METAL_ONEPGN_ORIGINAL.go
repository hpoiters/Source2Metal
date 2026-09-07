package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	Version              = "2.4"
	DefaultMaxPly        = 80
	MinTargetGames       = 100000
	MaxTargetGames       = 1000000
	PageSize             = 4096
	PageCacheCap         = 8192
	ChoiceCacheCap       = 100000
	castleWK       uint8 = 1
	castleWQ       uint8 = 2
	castleBK       uint8 = 4
	castleBQ       uint8 = 8
)

// CTG position-signature/hash and move-byte decoding are based on the
// publicly reverse-engineered CTG algorithms used by the Daydreamer lineage
// and sshivaji/ctgreader. Donor recommendations/annotations are deliberately
// ignored in this METAL-neutral extractor.
//
// Portions adapted from sshivaji/ctgreader:
// Copyright (c) 2016 Shivkumar Shivaji
// Licensed under the MIT License. Permission is granted, free of charge, to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies,
// provided the copyright and permission notice are retained. The software is
// provided "AS IS", without warranty of any kind.

var hashBits = [64]uint32{
	0x3100d2bf, 0x3118e3de, 0x34ab1372, 0x2807a847,
	0x1633f566, 0x2143b359, 0x26d56488, 0x3b9e6f59,
	0x37755656, 0x3089ca7b, 0x18e92d85, 0x0cd0e9d8,
	0x1a9e3b54, 0x3eaa902f, 0x0d9bfaae, 0x2f32b45b,
	0x31ed6102, 0x3d3c8398, 0x146660e3, 0x0f8d4b76,
	0x02c77a5f, 0x146c8799, 0x1c47f51f, 0x249f8f36,
	0x24772043, 0x1fbc1e4d, 0x1e86b3fa, 0x37df36a6,
	0x16ed30e4, 0x02c3148e, 0x216e5929, 0x0636b34e,
	0x317f9f56, 0x15f09d70, 0x131026fb, 0x38c784b1,
	0x29ac3305, 0x2b485dc5, 0x3c049ddc, 0x35a9fbcd,
	0x31d5373b, 0x2b246799, 0x0a2923d3, 0x08a96e9d,
	0x30031a9f, 0x08f525b5, 0x33611c06, 0x2409db98,
	0x0ca4feb2, 0x1000b71e, 0x30566e32, 0x39447d31,
	0x194e3752, 0x08233a95, 0x0f38fe36, 0x29c7cd57,
	0x0f7b3a39, 0x328e8a16, 0x1e7d1388, 0x0fba78f5,
	0x274c7e7c, 0x1e8be65c, 0x2fa0b0bb, 0x1eb6c371,
}

const pieceCode = "PNxQPQPxQBKxPBRNxxBKPBxxPxQBxBxxxRBQPxBPQQNxxPBQNQBxNxNQQQBQBxxx" +
	"xQQxKQxxxxPQNQxxRxRxBPxxxxxxPxxPxQPQxxBKxRBxxxRQxxBxQxxxxBRRPRQR" +
	"QRPxxNRRxxNPKxQQxxQxQxPKRRQPxQxBQxQPxRxxxRxQxRQxQPBxxRxQxBxPQQKx" +
	"xBBBRRQPPQBPBRxPxPNNxxxQRQNPxxPKNRxRxQPQRNxPPQQRQQxNRBxNQQQQxQQx"

var pieceIndex = [256]int{
	5, 2, 9, 2, 2, 1, 4, 9, 2, 2, 1, 9, 1, 1, 2, 1,
	9, 9, 1, 1, 8, 1, 9, 9, 7, 9, 2, 1, 9, 2, 9, 9,
	9, 2, 2, 2, 8, 9, 1, 3, 1, 1, 2, 9, 9, 6, 1, 1,
	2, 1, 2, 9, 1, 9, 1, 1, 2, 1, 1, 2, 1, 9, 9, 9,
	9, 2, 1, 9, 1, 1, 9, 9, 9, 9, 8, 1, 2, 2, 9, 9,
	1, 9, 1, 9, 2, 3, 9, 9, 9, 9, 9, 9, 7, 9, 9, 5,
	9, 1, 2, 2, 9, 9, 1, 1, 9, 2, 1, 0, 9, 9, 1, 2,
	9, 9, 2, 9, 1, 9, 9, 9, 9, 2, 1, 2, 3, 2, 1, 1,
	1, 1, 6, 9, 9, 1, 1, 1, 9, 9, 1, 1, 1, 9, 2, 1,
	9, 9, 2, 9, 1, 9, 2, 1, 1, 1, 1, 3, 9, 1, 9, 2,
	2, 9, 1, 8, 9, 2, 9, 9, 9, 2, 9, 2, 9, 2, 2, 9,
	2, 6, 1, 9, 9, 2, 9, 1, 9, 2, 9, 5, 2, 2, 1, 9,
	9, 1, 2, 1, 2, 2, 2, 7, 7, 2, 2, 6, 2, 1, 9, 4,
	9, 2, 2, 2, 9, 9, 9, 1, 2, 1, 1, 1, 9, 9, 5, 1,
	2, 1, 9, 2, 9, 1, 4, 1, 1, 1, 9, 4, 1, 1, 2, 1,
	2, 1, 9, 2, 2, 2, 0, 1, 2, 2, 2, 2, 9, 1, 2, 9,
}

var forward = [256]int{
	1, -1, 9, 0, 1, 1, 1, 9, 0, 6, -1, 9, 1, 3, 0, -1,
	9, 9, 7, 1, 1, 5, 9, 9, 1, 9, 6, 1, 9, 7, 9, 9,
	9, 0, 2, 6, 1, 9, 7, 1, 5, 0, -2, 9, 9, 1, 1, 0,
	-2, 0, 5, 9, 2, 9, 1, 4, 4, 0, 6, 5, 5, 9, 9, 9,
	9, 5, 7, 9, -1, 3, 9, 9, 9, 9, 2, 5, 2, 1, 9, 9,
	6, 9, 0, 9, 1, 1, 9, 9, 9, 9, 9, 9, 1, 9, 9, 2,
	9, 6, 2, 7, 9, 9, 3, 1, 9, 7, 4, 0, 9, 9, 0, 7,
	9, 9, 7, 9, 0, 9, 9, 9, 9, 6, 3, 6, 1, 1, 3, 0,
	6, 1, 1, 9, 9, 2, 0, 5, 9, 9, -2, 1, -1, 9, 2, 0,
	9, 9, 1, 9, 3, 9, 1, 0, 0, 4, 6, 2, 9, 2, 9, 4,
	3, 9, 2, 1, 9, 5, 9, 9, 9, 0, 9, 6, 9, 0, 3, 9,
	4, 2, 6, 9, 9, 0, 9, 5, 9, 3, 9, 1, 0, 2, 0, 9,
	9, 2, 2, 2, 0, 4, 5, 1, 2, 7, 3, 1, 5, 0, 9, 1,
	9, 1, 1, 1, 9, 9, 9, 1, 0, 2, -2, 2, 9, 9, 1, 1,
	-1, 7, 9, 3, 9, 0, 2, 4, 2, -1, 9, 1, 1, 7, 1, 0,
	0, 1, 9, 2, 2, 1, 0, 1, 0, 6, 0, 2, 9, 7, 3, 9,
}

var left = [256]int{
	-1, 2, 9, -2, 0, 0, 1, 9, -4, -6, 0, 9, 1, -3, -3, 2,
	9, 9, -7, 0, -1, -5, 9, 9, 0, 9, 0, 1, 9, -7, 9, 9,
	9, -7, 2, -6, 1, 9, 7, 1, -5, -6, -1, 9, 9, -1, -1, -1,
	1, -3, -5, 9, -1, 9, -2, 0, 4, -5, -6, 5, 5, 9, 9, 9,
	9, -5, 7, 9, -1, -3, 9, 9, 9, 9, 0, 5, -1, 0, 9, 9,
	0, 9, -6, 9, 1, 0, 9, 9, 9, 9, 9, 9, -1, 9, 9, 0,
	9, -6, 0, 7, 9, 9, 3, -1, 9, 0, -4, 0, 9, 9, -5, -7,
	9, 9, 7, 9, -2, 9, 9, 9, 9, 6, 0, 0, -1, 0, 3, -1,
	6, 0, 1, 9, 9, 1, -7, 0, 9, 9, -1, -1, 1, 9, 2, -7,
	9, 9, -1, 9, 0, 9, -1, 1, -3, 0, 0, 0, 9, 0, 9, 4,
	0, 9, -2, 0, 9, 0, 9, 9, 9, -2, 9, 6, 9, -4, -3, 9,
	0, 0, 6, 9, 9, -5, 9, 0, 9, -3, 9, 0, -5, 0, -1, 9,
	9, -2, -2, 2, -1, 0, 0, 1, 0, 0, 3, 0, 5, -2, 9, 0,
	9, 1, -2, 2, 9, 9, 9, 1, -6, 2, 1, 0, 9, 9, 1, 1,
	-2, 0, 9, 0, 9, -4, 0, -4, 0, -2, 9, -1, 0, -7, 1, -4,
	-7, -1, 9, 1, 0, -1, 0, 2, -1, 0, -3, -2, 9, 0, 3, 9,
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ---------------- Chess board ----------------

type Move struct {
	From, To   int
	Promo      byte
	EP, Castle bool
}

type Board struct {
	S      [64]byte
	Turn   byte
	Castle uint8
	EP     int
}

func startBoard() Board {
	var b Board
	for i := range b.S {
		b.S[i] = '.'
	}
	backW := "RNBQKBNR"
	backB := "rnbqkbnr"
	for f := 0; f < 8; f++ {
		b.S[f] = backW[f]
		b.S[8+f] = 'P'
		b.S[48+f] = 'p'
		b.S[56+f] = backB[f]
	}
	b.Turn = 'w'
	b.Castle = castleWK | castleWQ | castleBK | castleBQ
	b.EP = -1
	return b
}

func colorOf(p byte) byte {
	if p >= 'A' && p <= 'Z' {
		return 'w'
	}
	if p >= 'a' && p <= 'z' {
		return 'b'
	}
	return 0
}
func opp(c byte) byte {
	if c == 'w' {
		return 'b'
	}
	return 'w'
}
func pieceUpper(p byte) byte {
	if p >= 'a' && p <= 'z' {
		return p - 32
	}
	return p
}
func sqName(s int) string { return string([]byte{byte('a' + s%8), byte('1' + s/8)}) }
func coord(m Move) string {
	s := sqName(m.From) + sqName(m.To)
	if m.Promo != 0 {
		s += strings.ToLower(string(m.Promo))
	}
	return s
}

func (b Board) kingSquare(side byte) int {
	k := byte('K')
	if side == 'b' {
		k = 'k'
	}
	for i, p := range b.S {
		if p == k {
			return i
		}
	}
	return -1
}

func (b Board) attacked(sq int, by byte) bool {
	f, r := sq%8, sq/8
	// pawns: reverse-look from target square
	if by == 'w' {
		for _, df := range []int{-1, 1} {
			sf := f - df
			sr := r - 1
			if sf >= 0 && sf < 8 && sr >= 0 && sr < 8 && b.S[sr*8+sf] == 'P' {
				return true
			}
		}
	} else {
		for _, df := range []int{-1, 1} {
			sf := f - df
			sr := r + 1
			if sf >= 0 && sf < 8 && sr >= 0 && sr < 8 && b.S[sr*8+sf] == 'p' {
				return true
			}
		}
	}
	knights := [][2]int{{1, 2}, {2, 1}, {2, -1}, {1, -2}, {-1, -2}, {-2, -1}, {-2, 1}, {-1, 2}}
	wantN := byte('N')
	if by == 'b' {
		wantN = 'n'
	}
	for _, d := range knights {
		x, y := f+d[0], r+d[1]
		if x >= 0 && x < 8 && y >= 0 && y < 8 && b.S[y*8+x] == wantN {
			return true
		}
	}
	dirsB := [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}}
	dirsR := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for _, d := range dirsB {
		x, y := f+d[0], r+d[1]
		for x >= 0 && x < 8 && y >= 0 && y < 8 {
			p := b.S[y*8+x]
			if p != '.' {
				if colorOf(p) == by {
					u := pieceUpper(p)
					if u == 'B' || u == 'Q' {
						return true
					}
				}
				break
			}
			x += d[0]
			y += d[1]
		}
	}
	for _, d := range dirsR {
		x, y := f+d[0], r+d[1]
		for x >= 0 && x < 8 && y >= 0 && y < 8 {
			p := b.S[y*8+x]
			if p != '.' {
				if colorOf(p) == by {
					u := pieceUpper(p)
					if u == 'R' || u == 'Q' {
						return true
					}
				}
				break
			}
			x += d[0]
			y += d[1]
		}
	}
	wantK := byte('K')
	if by == 'b' {
		wantK = 'k'
	}
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			if dx == 0 && dy == 0 {
				continue
			}
			x, y := f+dx, r+dy
			if x >= 0 && x < 8 && y >= 0 && y < 8 && b.S[y*8+x] == wantK {
				return true
			}
		}
	}
	return false
}
func (b Board) inCheck(side byte) bool {
	k := b.kingSquare(side)
	return k >= 0 && b.attacked(k, opp(side))
}

func (b Board) pseudoMoves() []Move {
	out := make([]Move, 0, 64)
	side := b.Turn
	for from, p := range b.S {
		if colorOf(p) != side {
			continue
		}
		f, r := from%8, from/8
		u := pieceUpper(p)
		switch u {
		case 'P':
			dir, start, prom := 1, 1, 7
			if side == 'b' {
				dir = -1
				start = 6
				prom = 0
			}
			nr := r + dir
			if nr >= 0 && nr < 8 {
				to := nr*8 + f
				if b.S[to] == '.' {
					if nr == prom {
						out = append(out, Move{From: from, To: to, Promo: 'Q'})
					} else {
						out = append(out, Move{From: from, To: to})
					}
					if r == start {
						nr2 := r + 2*dir
						to2 := nr2*8 + f
						if b.S[to2] == '.' {
							out = append(out, Move{From: from, To: to2})
						}
					}
				}
				for _, df := range []int{-1, 1} {
					nf := f + df
					if nf < 0 || nf >= 8 {
						continue
					}
					to := nr*8 + nf
					if (b.S[to] != '.' && colorOf(b.S[to]) == opp(side)) || to == b.EP {
						m := Move{From: from, To: to, EP: to == b.EP && b.S[to] == '.'}
						if nr == prom {
							m.Promo = 'Q'
						}
						out = append(out, m)
					}
				}
			}
		case 'N':
			for _, d := range [][2]int{{1, 2}, {2, 1}, {2, -1}, {1, -2}, {-1, -2}, {-2, -1}, {-2, 1}, {-1, 2}} {
				nf, nr := f+d[0], r+d[1]
				if nf < 0 || nf >= 8 || nr < 0 || nr >= 8 {
					continue
				}
				to := nr*8 + nf
				if colorOf(b.S[to]) != side {
					out = append(out, Move{From: from, To: to})
				}
			}
		case 'B', 'R', 'Q':
			dirs := [][2]int{}
			if u == 'B' || u == 'Q' {
				dirs = append(dirs, [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}}...)
			}
			if u == 'R' || u == 'Q' {
				dirs = append(dirs, [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}...)
			}
			for _, d := range dirs {
				nf, nr := f+d[0], r+d[1]
				for nf >= 0 && nf < 8 && nr >= 0 && nr < 8 {
					to := nr*8 + nf
					if b.S[to] == '.' {
						out = append(out, Move{From: from, To: to})
					} else {
						if colorOf(b.S[to]) != side {
							out = append(out, Move{From: from, To: to})
						}
						break
					}
					nf += d[0]
					nr += d[1]
				}
			}
		case 'K':
			for dx := -1; dx <= 1; dx++ {
				for dy := -1; dy <= 1; dy++ {
					if dx == 0 && dy == 0 {
						continue
					}
					nf, nr := f+dx, r+dy
					if nf >= 0 && nf < 8 && nr >= 0 && nr < 8 {
						to := nr*8 + nf
						if colorOf(b.S[to]) != side {
							out = append(out, Move{From: from, To: to})
						}
					}
				}
			}
			if side == 'w' && from == 4 && !b.inCheck('w') {
				if b.Castle&castleWK != 0 && b.S[5] == '.' && b.S[6] == '.' && b.S[7] == 'R' && !b.attacked(5, 'b') && !b.attacked(6, 'b') {
					out = append(out, Move{From: 4, To: 6, Castle: true})
				}
				if b.Castle&castleWQ != 0 && b.S[3] == '.' && b.S[2] == '.' && b.S[1] == '.' && b.S[0] == 'R' && !b.attacked(3, 'b') && !b.attacked(2, 'b') {
					out = append(out, Move{From: 4, To: 2, Castle: true})
				}
			}
			if side == 'b' && from == 60 && !b.inCheck('b') {
				if b.Castle&castleBK != 0 && b.S[61] == '.' && b.S[62] == '.' && b.S[63] == 'r' && !b.attacked(61, 'w') && !b.attacked(62, 'w') {
					out = append(out, Move{From: 60, To: 62, Castle: true})
				}
				if b.Castle&castleBQ != 0 && b.S[59] == '.' && b.S[58] == '.' && b.S[57] == '.' && b.S[56] == 'r' && !b.attacked(59, 'w') && !b.attacked(58, 'w') {
					out = append(out, Move{From: 60, To: 58, Castle: true})
				}
			}
		}
	}
	return out
}

func (b *Board) Apply(m Move) {
	p := b.S[m.From]
	side := colorOf(p)
	target := b.S[m.To]
	// castling rights lost by moving/capturing rook or king
	switch m.From {
	case 0:
		b.Castle &^= castleWQ
	case 7:
		b.Castle &^= castleWK
	case 56:
		b.Castle &^= castleBQ
	case 63:
		b.Castle &^= castleBK
	}
	switch m.To {
	case 0:
		if target == 'R' {
			b.Castle &^= castleWQ
		}
	case 7:
		if target == 'R' {
			b.Castle &^= castleWK
		}
	case 56:
		if target == 'r' {
			b.Castle &^= castleBQ
		}
	case 63:
		if target == 'r' {
			b.Castle &^= castleBK
		}
	}
	if pieceUpper(p) == 'K' {
		if side == 'w' {
			b.Castle &^= castleWK | castleWQ
		} else {
			b.Castle &^= castleBK | castleBQ
		}
	}
	oldEP := b.EP
	b.EP = -1
	b.S[m.From] = '.'
	if m.EP || (pieceUpper(p) == 'P' && m.To == oldEP && target == '.' && m.From%8 != m.To%8) {
		capSq := m.To - 8
		if side == 'b' {
			capSq = m.To + 8
		}
		b.S[capSq] = '.'
	}
	if m.Castle || (pieceUpper(p) == 'K' && abs((m.To%8)-(m.From%8)) == 2) {
		if m.To%8 == 6 {
			rf := m.From/8*8 + 7
			rt := m.From/8*8 + 5
			b.S[rt] = b.S[rf]
			b.S[rf] = '.'
		}
		if m.To%8 == 2 {
			rf := m.From/8*8 + 0
			rt := m.From/8*8 + 3
			b.S[rt] = b.S[rf]
			b.S[rf] = '.'
		}
	}
	if m.Promo != 0 && pieceUpper(p) == 'P' {
		if side == 'w' {
			p = pieceUpper(m.Promo)
		} else {
			p = pieceUpper(m.Promo) + 32
		}
	}
	b.S[m.To] = p
	if pieceUpper(p) == 'P' && abs(m.To-m.From) == 16 {
		// Daydreamer/CTG only keeps an EP square when the opponent really
		// has a pawn adjacent to the double-pushed pawn and can capture it.
		oppPawn := byte('p')
		if side == 'b' {
			oppPawn = 'P'
		}
		f := m.To % 8
		for _, df := range []int{-1, 1} {
			nf := f + df
			if nf >= 0 && nf < 8 && b.S[(m.To/8)*8+nf] == oppPawn {
				b.EP = (m.To + m.From) / 2
				break
			}
		}
	}
	b.Turn = opp(b.Turn)
}

func (b Board) LegalMoves() []Move {
	ps := b.pseudoMoves()
	out := make([]Move, 0, len(ps))
	side := b.Turn
	for _, m := range ps {
		c := b
		c.Apply(m)
		if !c.inCheck(side) {
			out = append(out, m)
		}
	}
	return out
}

func (b Board) SAN(m Move) string {
	p := b.S[m.From]
	u := pieceUpper(p)
	if u == 'K' && abs((m.To%8)-(m.From%8)) == 2 {
		s := "O-O"
		if m.To%8 == 2 {
			s = "O-O-O"
		}
		c := b
		c.Apply(m)
		if c.inCheck(c.Turn) {
			if len(c.LegalMoves()) == 0 {
				s += "#"
			} else {
				s += "+"
			}
		}
		return s
	}
	capture := b.S[m.To] != '.' || m.EP || (u == 'P' && m.To == b.EP && m.From%8 != m.To%8)
	var sb strings.Builder
	if u != 'P' {
		sb.WriteByte(u)
	}
	if u != 'P' {
		legal := b.LegalMoves()
		sameFile, sameRank := false, false
		rivals := 0
		for _, x := range legal {
			if x.To == m.To && x.From != m.From && pieceUpper(b.S[x.From]) == u {
				rivals++
				if x.From%8 == m.From%8 {
					sameFile = true
				}
				if x.From/8 == m.From/8 {
					sameRank = true
				}
			}
		}
		if rivals > 0 {
			if !sameFile {
				sb.WriteByte(byte('a' + m.From%8))
			} else if !sameRank {
				sb.WriteByte(byte('1' + m.From/8))
			} else {
				sb.WriteString(sqName(m.From))
			}
		}
	} else if capture {
		sb.WriteByte(byte('a' + m.From%8))
	}
	if capture {
		sb.WriteByte('x')
	}
	sb.WriteString(sqName(m.To))
	if m.Promo != 0 {
		sb.WriteByte('=')
		sb.WriteByte(pieceUpper(m.Promo))
	}
	c := b
	c.Apply(m)
	if c.inCheck(c.Turn) {
		if len(c.LegalMoves()) == 0 {
			sb.WriteByte('#')
		} else {
			sb.WriteByte('+')
		}
	}
	return sb.String()
}

// ---------------- CTG signature/hash ----------------

func swapColor(p byte) byte {
	if p >= 'A' && p <= 'Z' {
		return p + 32
	}
	if p >= 'a' && p <= 'z' {
		return p - 32
	}
	return p
}

func (b Board) normalizedFlags() (flip, mirror bool) {
	flip = b.Turn == 'b'
	ksq := b.kingSquare(b.Turn)
	kfile := 4
	if ksq >= 0 {
		kfile = ksq % 8
	}
	// Public CTG decoder mirrors only when neither side has any castling right.
	mirror = kfile < 4 && b.Castle == 0
	return
}

func (b Board) normPieceAt(nf, nr int, flip, mirror bool) byte {
	af, ar := nf, nr
	if flip {
		ar = 7 - ar
	}
	if mirror {
		af = 7 - af
	}
	p := b.S[ar*8+af]
	if flip {
		p = swapColor(p)
	}
	return p
}

func appendBitsReverse(buf []byte, bits byte, bitPos, numBits int) {
	for i := 0; i < numBits; i++ {
		if bits&1 != 0 {
			pos := bitPos + i
			buf[pos/8] |= 1 << (7 - (pos % 8))
		}
		bits >>= 1
	}
}

func huffPiece(p byte) (byte, int) {
	switch p {
	case '.':
		return 0x0, 1
	case 'P':
		return 0x3, 3
	case 'p':
		return 0x7, 3
	case 'N':
		return 0x9, 5
	case 'n':
		return 0x19, 5
	case 'B':
		return 0x5, 5
	case 'b':
		return 0x15, 5
	case 'R':
		return 0x0D, 5
	case 'r':
		return 0x1D, 5
	case 'Q':
		return 0x11, 6
	case 'q':
		return 0x31, 6
	case 'K':
		return 0x01, 6
	case 'k':
		return 0x21, 6
	}
	return 0, 0
}

func (b Board) Signature() []byte {
	buf := make([]byte, 64)
	bitPos := 8 // first byte is reserved for length/flags
	flip, mirror := b.normalizedFlags()
	for f := 0; f < 8; f++ {
		for r := 0; r < 8; r++ {
			p := b.normPieceAt(f, r, flip, mirror)
			bits, n := huffPiece(p)
			appendBitsReverse(buf, bits, bitPos, n)
			bitPos += n
		}
	}

	ep := -1
	flagBitLen := 0
	if b.EP >= 0 {
		ep = b.EP % 8
		if mirror {
			ep = 7 - ep
		}
		flagBitLen = 3
	}
	castle := 0
	if b.Turn == 'w' {
		if b.Castle&castleWK != 0 {
			castle += 4
		}
		if b.Castle&castleWQ != 0 {
			castle += 8
		}
		if b.Castle&castleBK != 0 {
			castle += 1
		}
		if b.Castle&castleBQ != 0 {
			castle += 2
		}
	} else {
		if b.Castle&castleBK != 0 {
			castle += 4
		}
		if b.Castle&castleBQ != 0 {
			castle += 8
		}
		if b.Castle&castleWK != 0 {
			castle += 1
		}
		if b.Castle&castleWQ != 0 {
			castle += 2
		}
	}
	if castle != 0 {
		flagBitLen += 4
	}
	flagBits := byte(castle)
	if ep != -1 {
		flagBits <<= 3
		x := ep
		for i := 0; i < 3; i++ {
			if x&1 != 0 {
				flagBits |= 1 << (2 - i)
			}
			x >>= 1
		}
	}
	if 8-(bitPos%8) < flagBitLen {
		pad := 8 - (bitPos % 8)
		appendBitsReverse(buf, 0, bitPos, pad)
		bitPos += pad
	}
	pad := 8 - (bitPos % 8) - flagBitLen
	if pad < 0 {
		pad += 8
	}
	appendBitsReverse(buf, 0, bitPos, pad)
	bitPos += pad
	appendBitsReverse(buf, flagBits, bitPos, flagBitLen)
	bitPos += flagBitLen
	ln := (bitPos + 7) / 8
	buf[0] = byte(ln)
	if ep != -1 {
		buf[0] |= 1 << 5
	}
	if castle != 0 {
		buf[0] |= 1 << 6
	}
	return append([]byte(nil), buf[:ln]...)
}

func ctgHash(sig []byte) int32 {
	var h uint32
	tmp := 0
	for _, bb := range sig {
		x := int(int8(bb))
		tmp += ((0x0f - (x & 0x0f)) << 2) + 1
		h += hashBits[tmp&63]
		tmp += ((0xf0 - (x & 0xf0)) >> 2) + 1
		h += hashBits[tmp&63]
	}
	return int32(h)
}

// ---------------- CTG reader ----------------

type RawMove struct{ Code, Ann byte }
type Entry struct {
	Moves                      []RawMove
	Total, Losses, Wins, Draws uint32
	Recommendation             byte
}

type pageCache struct {
	m     map[int32][]byte
	order []int32
	cap   int
}

func newPageCache(cap int) *pageCache           { return &pageCache{m: map[int32][]byte{}, cap: cap} }
func (c *pageCache) get(k int32) ([]byte, bool) { v, ok := c.m[k]; return v, ok }
func (c *pageCache) put(k int32, v []byte) {
	if _, ok := c.m[k]; ok {
		return
	}
	if len(c.order) >= c.cap {
		old := c.order[0]
		c.order = c.order[1:]
		delete(c.m, old)
	}
	c.m[k] = v
	c.order = append(c.order, k)
}

type Choice struct {
	Code                byte
	Weight              uint32
	Wins, Draws, Losses uint32
}
type choiceCache struct {
	m     map[string][]Choice
	order []string
	cap   int
}

func newChoiceCache(cap int) *choiceCache            { return &choiceCache{m: map[string][]Choice{}, cap: cap} }
func (c *choiceCache) get(k string) ([]Choice, bool) { v, ok := c.m[k]; return v, ok }
func (c *choiceCache) put(k string, v []Choice) {
	if _, ok := c.m[k]; ok {
		return
	}
	if len(c.order) >= c.cap {
		old := c.order[0]
		c.order = c.order[1:]
		delete(c.m, old)
	}
	c.m[k] = v
	c.order = append(c.order, k)
}

type CTGReader struct {
	ctg, cto     *os.File
	low, high    uint32
	pages        *pageCache
	choices      *choiceCache
	decodeErrors int64
}

func openCTG(ctgPath, ctoPath, ctbPath string) (*CTGReader, error) {
	ctg, e := os.Open(ctgPath)
	if e != nil {
		return nil, e
	}
	cto, e := os.Open(ctoPath)
	if e != nil {
		ctg.Close()
		return nil, e
	}
	ctb, e := os.Open(ctbPath)
	if e != nil {
		ctg.Close()
		cto.Close()
		return nil, e
	}
	defer ctb.Close()
	var hdr [12]byte
	if _, e = io.ReadFull(ctb, hdr[:]); e != nil {
		ctg.Close()
		cto.Close()
		return nil, fmt.Errorf("CTB header: %w", e)
	}
	low := binary.BigEndian.Uint32(hdr[4:8])
	high := binary.BigEndian.Uint32(hdr[8:12])
	return &CTGReader{ctg: ctg, cto: cto, low: low, high: high, pages: newPageCache(PageCacheCap), choices: newChoiceCache(ChoiceCacheCap)}, nil
}
func (r *CTGReader) Close() {
	if r.ctg != nil {
		r.ctg.Close()
	}
	if r.cto != nil {
		r.cto.Close()
	}
}

func (r *CTGReader) pageIndex(sig []byte) (int32, error) {
	h := uint32(ctgHash(sig))
	key := uint32(0)
	for mask := uint32(1); key <= r.high; mask = (mask << 1) + 1 {
		key = (h & mask) + mask
		if key >= r.low && key <= r.high {
			var b [4]byte
			if _, e := r.cto.ReadAt(b[:], 16+int64(key)*4); e == nil {
				idx := int32(binary.BigEndian.Uint32(b[:]))
				if idx >= 0 {
					return idx, nil
				}
			}
		}
		if mask == 0xffffffff {
			break
		}
	}
	return -1, errors.New("positie niet in CTO-index")
}
func (r *CTGReader) page(idx int32) ([]byte, error) {
	if v, ok := r.pages.get(idx); ok {
		return v, nil
	}
	b := make([]byte, PageSize)
	if _, e := r.ctg.ReadAt(b, (int64(idx)+1)*PageSize); e != nil {
		return nil, e
	}
	r.pages.put(idx, b)
	return b, nil
}
func read24(b []byte) uint32 { return uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2]) }

func (r *CTGReader) Lookup(b Board) (Entry, error) {
	sig := b.Signature()
	idx, e := r.pageIndex(sig)
	if e != nil {
		return Entry{}, e
	}
	pg, e := r.page(idx)
	if e != nil {
		return Entry{}, e
	}
	if len(pg) < 4 {
		return Entry{}, errors.New("korte CTG-pagina")
	}
	n := int(binary.BigEndian.Uint16(pg[0:2]))
	pos := 4
	for i := 0; i < n; i++ {
		if pos >= len(pg) {
			break
		}
		sl := int(pg[pos] & 31)
		if sl <= 0 || pos+sl >= len(pg) {
			break
		}
		moveSize := int(pg[pos+sl])
		if moveSize < 1 || pos+sl+moveSize+33 > len(pg) {
			break
		}
		if sl == len(sig) && string(pg[pos:pos+sl]) == string(sig) {
			mcount := (moveSize - 1) / 2
			moves := make([]RawMove, 0, mcount)
			mp := pos + sl + 1
			for j := 0; j < mcount; j++ {
				moves = append(moves, RawMove{Code: pg[mp+2*j], Ann: pg[mp+2*j+1]})
			}
			q := pos + sl + moveSize
			en := Entry{Moves: moves}
			en.Total = read24(pg[q : q+3])
			q += 3
			en.Losses = read24(pg[q : q+3])
			q += 3
			en.Wins = read24(pg[q : q+3])
			q += 3
			en.Draws = read24(pg[q : q+3])
			q += 3
			q += 4 + 3 + 4 + 3 + 4
			if q < len(pg) {
				en.Recommendation = pg[q]
			}
			return en, nil
		}
		pos += sl + moveSize + 33
	}
	return Entry{}, errors.New("positie niet gevonden op CTG-pagina")
}

func normPieceForDecode(b Board, sq int, flip, mirror bool) byte {
	f, r := sq%8, sq/8
	nf, nr := f, r
	if mirror {
		nf = 7 - nf
	}
	if flip {
		nr = 7 - nr
	}
	// this function expects normalized coordinates input in sq; convert to actual then normalize color
	af, ar := nf, nr
	p := b.S[ar*8+af]
	if flip {
		p = swapColor(p)
	}
	return p
}

func (r *CTGReader) DecodeMove(b Board, code byte) (Move, error) {
	legal := b.LegalMoves()
	if code == 107 || code == 246 {
		wantToFile := 6
		if code == 246 {
			wantToFile = 2
		}
		for _, m := range legal {
			if pieceUpper(b.S[m.From]) == 'K' && m.To%8 == wantToFile && abs(m.To-m.From) == 2 {
				return m, nil
			}
		}
		return Move{}, fmt.Errorf("rokade-byte %d niet legaal", code)
	}
	pc := '?'
	if int(code) < len(pieceCode) {
		pc = rune(pieceCode[int(code)])
	}
	if pc == '?' {
		return Move{}, fmt.Errorf("onbekende CTG move byte %d", code)
	}
	flip, mirror := b.normalizedFlags()
	nth := int(pieceIndex[code])
	if nth <= 0 {
		return Move{}, fmt.Errorf("ongeldige piece-index voor byte %d", code)
	}
	count := 0
	srcNorm := -1
	for f := 0; f < 8 && srcNorm < 0; f++ {
		for rr := 0; rr < 8; rr++ {
			p := b.normPieceAt(f, rr, flip, mirror)
			if p == byte(pc) {
				count++
				if count == nth {
					srcNorm = rr*8 + f
					break
				}
			}
		}
	}
	if srcNorm < 0 {
		return Move{}, fmt.Errorf("bronstuk niet gevonden voor byte %d", code)
	}
	sf, sr := srcNorm%8, srcNorm/8
	df := (sf - int(left[code]) + 8) % 8
	dr := (sr + int(forward[code]) + 8) % 8
	// normalized -> actual
	af, ar := sf, sr
	tf, tr := df, dr
	if mirror {
		af = 7 - af
		tf = 7 - tf
	}
	if flip {
		ar = 7 - ar
		tr = 7 - tr
	}
	from, to := ar*8+af, tr*8+tf
	for _, m := range legal {
		if m.From == from && m.To == to {
			return m, nil
		}
	}
	return Move{}, fmt.Errorf("CTG byte %d geeft geen legale zet %s%s", code, sqName(from), sqName(to))
}

func (r *CTGReader) Choices(b Board) ([]Choice, error) {
	key := string(b.Signature())
	if v, ok := r.choices.get(key); ok {
		return v, nil
	}
	en, e := r.Lookup(b)
	if e != nil {
		return nil, e
	}
	out := make([]Choice, 0, len(en.Moves))
	for _, rm := range en.Moves {
		m, de := r.DecodeMove(b, rm.Code)
		if de != nil {
			r.decodeErrors++
			continue
		}
		c := b
		c.Apply(m)
		child, ce := r.Lookup(c)
		w := uint32(1)
		var wi, dr, lo uint32
		if ce == nil {
			wi, dr, lo = child.Wins, child.Draws, child.Losses
			w = child.Total
			if w == 0 {
				w = wi + dr + lo
			}
			if w == 0 {
				w = 1
			}
		}
		out = append(out, Choice{Code: rm.Code, Weight: w, Wins: wi, Draws: dr, Losses: lo})
	}
	r.choices.put(key, out)
	return out, nil
}

// ---------------- PGN output ----------------

func formatMoves(moves []string, result string) string {
	var parts []string
	for i, m := range moves {
		if i%2 == 0 {
			parts = append(parts, fmt.Sprintf("%d. %s", i/2+1, m))
		} else {
			parts = append(parts, m)
		}
	}
	parts = append(parts, result)
	var lines []string
	line := ""
	for _, p := range parts {
		if line == "" {
			line = p
		} else if len(line)+1+len(p) <= 100 {
			line += " " + p
		} else {
			lines = append(lines, line)
			line = p
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func resultFromWDL(rng *rand.Rand, w, d, l uint32, lastMover byte) string {
	t := uint64(w) + uint64(d) + uint64(l)
	if t == 0 {
		return "*"
	}
	x := uint64(rng.Int63n(int64(t)))
	if x < uint64(w) {
		if lastMover == 'w' {
			return "1-0"
		}
		return "0-1"
	}
	x -= uint64(w)
	if x < uint64(d) {
		return "1/2-1/2"
	}
	if lastMover == 'w' {
		return "0-1"
	}
	return "1-0"
}

func chooseWeighted(rng *rand.Rand, ch []Choice, forced int) int {
	if forced >= 0 && forced < len(ch) {
		return forced
	}
	var total uint64
	for _, c := range ch {
		total += uint64(max(1, int(c.Weight)))
	}
	if total == 0 {
		return 0
	}
	x := uint64(rng.Int63n(int64(total)))
	for i, c := range ch {
		w := uint64(max(1, int(c.Weight)))
		if x < w {
			return i
		}
		x -= w
	}
	return len(ch) - 1
}

type Progress struct {
	start, last time.Time
	total       int
}

func newProgress(total int) *Progress {
	return &Progress{start: time.Now(), last: time.Time{}, total: total}
}
func fmtRemain(d time.Duration) string {
	if d < 0 {
		return "?"
	}
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %02dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
func (p *Progress) Update(done int, force bool) {
	now := time.Now()
	if !force && !p.last.IsZero() && now.Sub(p.last) < 300*time.Millisecond {
		return
	}
	p.last = now
	frac := float64(done) / float64(max(1, p.total))
	if frac > 1 {
		frac = 1
	}
	filled := int(frac*20 + .5)
	bar := strings.Repeat("=", filled) + strings.Repeat(" ", 20-filled)
	remain := time.Duration(-1)
	if frac > 0.005 {
		elapsed := now.Sub(p.start)
		remain = time.Duration(float64(elapsed) * (1 - frac) / frac)
	}
	txt := "?"
	if remain >= 0 {
		txt = fmtRemain(remain)
	}
	fmt.Printf("\rMaken: [%-20s] %3.0f%% | resterend ca. %s", bar, frac*100, txt)
	if force {
		fmt.Println()
	}
}

func targetGames(root Entry) int {
	n := int(root.Total)
	if n == 0 {
		n = int(root.Wins + root.Draws + root.Losses)
	}
	if n == 0 {
		n = MinTargetGames
	}
	if n < MinTargetGames {
		n = MinTargetGames
	}
	if n > MaxTargetGames {
		n = MaxTargetGames
	}
	return n
}

func writeNeutralPGN(reader *CTGReader, outPath, bookName string, maxPly int) (games int, plies int64, err error) {
	root, e := reader.Lookup(startBoard())
	if e != nil {
		return 0, 0, fmt.Errorf("beginstelling niet gevonden: %w", e)
	}
	total := targetGames(root)
	f, e := os.Create(outPath)
	if e != nil {
		return 0, 0, e
	}
	defer func() {
		if err != nil {
			f.Close()
		}
	}()
	w := bufio.NewWriterSize(f, 1<<20)
	rng := rand.New(rand.NewSource(20260831 + int64(uint32(ctgHash(startBoard().Signature())))))
	prog := newProgress(total)
	rootChoices, _ := reader.Choices(startBoard())
	forcedRoot := len(rootChoices)
	for g := 0; g < total; g++ {
		b := startBoard()
		sans := make([]string, 0, maxPly)
		lastW, lastD, lastL := uint32(0), uint32(0), uint32(0)
		lastMover := byte('w')
		for ply := 0; ply < maxPly; ply++ {
			choices, ce := reader.Choices(b)
			if ce != nil || len(choices) == 0 {
				break
			}
			forced := -1
			if ply == 0 && g < forcedRoot {
				forced = g
			}
			idx := chooseWeighted(rng, choices, forced)
			chosen := choices[idx]
			m, de := reader.DecodeMove(b, chosen.Code)
			if de != nil {
				// fall back to first decodable choice
				ok := false
				for j, c := range choices {
					mm, ee := reader.DecodeMove(b, c.Code)
					if ee == nil {
						idx = j
						chosen = c
						m = mm
						ok = true
						break
					}
				}
				if !ok {
					break
				}
			}
			san := b.SAN(m)
			lastMover = b.Turn
			b.Apply(m)
			sans = append(sans, san)
			plies++
			lastW, lastD, lastL = chosen.Wins, chosen.Draws, chosen.Losses
		}
		res := resultFromWDL(rng, lastW, lastD, lastL, lastMover)
		fmt.Fprintf(w, "[Event \"CTG METAL NEUTRAAL\"]\n[Site \"CTGExtractor v%s\"]\n[Date \"????.??.??\"]\n[Round \"?\"]\n[White \"Neutral\"]\n[Black \"Neutral\"]\n[Result \"%s\"]\n\n%s\n\n", Version, res, formatMoves(sans, res))
		if (g+1)%4096 == 0 {
			if e := w.Flush(); e != nil {
				return g, plies, e
			}
		}
		games = g + 1
		prog.Update(games, false)
	}
	if e := w.Flush(); e != nil {
		return games, plies, e
	}
	if e := f.Close(); e != nil {
		return games, plies, e
	}
	prog.Update(total, true)
	return games, plies, nil
}

func fileSHA256(path string) (string, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	h := sha256.New()
	if _, e = io.Copy(h, f); e != nil {
		return "", e
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ---------------- Discovery / UI ----------------

type BookSet struct{ Name, CTG, CTO, CTB string }

func findBookSets(dir string) ([]BookSet, error) {
	ents, e := os.ReadDir(dir)
	if e != nil {
		return nil, e
	}
	by := map[string]string{}
	for _, x := range ents {
		if !x.IsDir() {
			by[strings.ToLower(x.Name())] = x.Name()
		}
	}
	var out []BookSet
	for _, x := range ents {
		if x.IsDir() || !strings.EqualFold(filepath.Ext(x.Name()), ".ctg") {
			continue
		}
		base := strings.TrimSuffix(x.Name(), filepath.Ext(x.Name()))
		cto, ok1 := by[strings.ToLower(base+".cto")]
		ctb, ok2 := by[strings.ToLower(base+".ctb")]
		if ok1 && ok2 {
			out = append(out, BookSet{Name: base, CTG: filepath.Join(dir, x.Name()), CTO: filepath.Join(dir, cto), CTB: filepath.Join(dir, ctb)})
		}
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out, nil
}
func exeDir() string {
	if p, e := os.Executable(); e == nil {
		return filepath.Dir(p)
	}
	d, _ := os.Getwd()
	return d
}
func promptInt(rd *bufio.Reader, prompt string, minv, maxv, def int) int {
	for {
		fmt.Print(prompt)
		s, _ := rd.ReadString('\n')
		s = strings.TrimSpace(s)
		if s == "" && def >= minv && def <= maxv {
			return def
		}
		n, e := strconv.Atoi(s)
		if e == nil && n >= minv && n <= maxv {
			return n
		}
		fmt.Printf("Voer een geheel getal van %d t/m %d in.\n", minv, maxv)
	}
}
func chooseBook(rd *bufio.Reader, sets []BookSet) BookSet {
	if len(sets) == 1 {
		return sets[0]
	}
	fmt.Println("Meerdere complete CTG-sets gevonden:")
	for i, s := range sets {
		fmt.Printf("  %d = %s\n", i+1, s.Name)
	}
	n := promptInt(rd, "Keuze: ", 1, len(sets), 1)
	return sets[n-1]
}
func makeRunDir(base, book string) (string, error) {
	root := filepath.Join(base, "CTG_Extractie")
	if e := os.MkdirAll(root, 0755); e != nil {
		return "", e
	}
	stem := fmt.Sprintf("%s_%s", book, time.Now().Format("20060102_150405"))
	p := filepath.Join(root, stem)
	for i := 2; ; i++ {
		if _, e := os.Stat(p); os.IsNotExist(e) {
			break
		}
		p = filepath.Join(root, fmt.Sprintf("%s_%d", stem, i))
	}
	if e := os.MkdirAll(p, 0755); e != nil {
		return "", e
	}
	return p, nil
}
func pause(rd *bufio.Reader) { fmt.Print("\nDruk op Enter om af te sluiten..."); rd.ReadString('\n') }

// ---------------- Self-test with synthetic CTG ----------------

func findCodeForCoord(b Board, want string) (byte, error) {
	r := &CTGReader{}
	for i := 0; i < 256; i++ {
		m, e := r.DecodeMove(b, byte(i))
		if e == nil && coord(m) == want {
			return byte(i), nil
		}
	}
	return 0, fmt.Errorf("geen CTG byte voor %s", want)
}
func encodeEntry(sig []byte, moves []RawMove, total, loss, wins, draw uint32) []byte {
	ms := 1 + 2*len(moves)
	out := make([]byte, 0, len(sig)+ms+33)
	out = append(out, sig...)
	out = append(out, byte(ms))
	for _, m := range moves {
		out = append(out, m.Code, m.Ann)
	}
	put24 := func(v uint32) { out = append(out, byte(v>>16), byte(v>>8), byte(v)) }
	put24(total)
	put24(loss)
	put24(wins)
	put24(draw)
	out = append(out, make([]byte, 4+3+4+3+4+3)...)
	return out
}
func selfTest() error {
	b := startBoard()
	if len(b.LegalMoves()) != 20 {
		return fmt.Errorf("start legal moves=%d", len(b.LegalMoves()))
	}
	cE, e := findCodeForCoord(b, "e2e4")
	if e != nil {
		return e
	}
	cD, e := findCodeForCoord(b, "d2d4")
	if e != nil {
		return e
	}
	cN, e := findCodeForCoord(b, "g1f3")
	if e != nil {
		return e
	}
	td, e := os.MkdirTemp("", "ctg24test")
	if e != nil {
		return e
	}
	defer os.RemoveAll(td)
	base := filepath.Join(td, "Tiny")
	ctb := make([]byte, 12)
	binary.BigEndian.PutUint32(ctb[4:8], 0)
	binary.BigEndian.PutUint32(ctb[8:12], 2)
	os.WriteFile(base+".ctb", ctb, 0644)
	cto := make([]byte, 28)
	binary.BigEndian.PutUint32(cto[20:24], 0)
	binary.BigEndian.PutUint32(cto[24:28], 0)
	os.WriteFile(base+".cto", cto, 0644)
	root := startBoard()
	entries := [][]byte{}
	entries = append(entries, encodeEntry(root.Signature(), []RawMove{{cE, 0}, {cD, 0}, {cN, 0}}, 1000, 200, 500, 300))
	for _, x := range []struct {
		code            byte
		tot, lo, wi, dr uint32
	}{{cE, 600, 100, 300, 200}, {cD, 300, 80, 120, 100}, {cN, 100, 30, 30, 40}} {
		m, _ := (&CTGReader{}).DecodeMove(root, x.code)
		ch := root
		ch.Apply(m)
		entries = append(entries, encodeEntry(ch.Signature(), nil, x.tot, x.lo, x.wi, x.dr))
	}
	pg := make([]byte, PageSize)
	binary.BigEndian.PutUint16(pg[0:2], uint16(len(entries)))
	p := 4
	for _, en := range entries {
		copy(pg[p:], en)
		p += len(en)
	}
	ctg := make([]byte, 2*PageSize)
	copy(ctg[PageSize:], pg)
	os.WriteFile(base+".ctg", ctg, 0644)
	rr, e := openCTG(base+".ctg", base+".cto", base+".ctb")
	if e != nil {
		return e
	}
	defer rr.Close()
	en, e := rr.Lookup(root)
	if e != nil {
		return e
	}
	if len(en.Moves) != 3 {
		return fmt.Errorf("root moves=%d", len(en.Moves))
	}
	ch, e := rr.Choices(root)
	if e != nil {
		return e
	}
	if len(ch) != 3 {
		return fmt.Errorf("choices=%d", len(ch))
	}
	sum := uint32(0)
	for _, x := range ch {
		sum += x.Weight
	}
	if sum != 1000 {
		return fmt.Errorf("weights som=%d", sum)
	}
	return nil
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--self-test" {
		if e := selfTest(); e != nil {
			fmt.Println("SELF-TEST FOUT:", e)
			os.Exit(1)
		}
		fmt.Println("SELF-TEST OK")
		return
	}
	rd := bufio.NewReader(os.Stdin)
	fmt.Printf("CTGExtractor v%s - 1 neutrale PGN voor METAL\n", Version)
	fmt.Println(strings.Repeat("=", 54))
	base := exeDir()
	fmt.Println("Map:", base)
	sets, e := findBookSets(base)
	if e != nil {
		fmt.Println("FOUT:", e)
		pause(rd)
		return
	}
	if len(sets) == 0 {
		fmt.Println("Geen complete .ctg + .cto + .ctb set naast de EXE gevonden.")
		pause(rd)
		return
	}
	book := chooseBook(rd, sets)
	maxPly := promptInt(rd, fmt.Sprintf("Maximale plydiepte [1-100] (Enter = %d): ", DefaultMaxPly), 1, 100, DefaultMaxPly)
	fmt.Println("\nBoek:", book.Name)
	fmt.Println("MaxPly:", maxPly)
	fmt.Println("Donor-markeringen/recommendaties: genegeerd")
	fmt.Println("Donor-Fact/weights: niet overgenomen")
	fmt.Println("CTG N/W/D/L: gebruikt voor neutrale representatieve PGN")
	run, e := makeRunDir(base, book.Name)
	if e != nil {
		fmt.Println("FOUT uitvoermap:", e)
		pause(rd)
		return
	}
	out := filepath.Join(run, book.Name+"_METAL_NEUTRAAL.pgn")
	r, e := openCTG(book.CTG, book.CTO, book.CTB)
	if e != nil {
		fmt.Println("FOUT CTG openen:", e)
		pause(rd)
		return
	}
	defer r.Close()
	games, plies, e := writeNeutralPGN(r, out, book.Name, maxPly)
	if e != nil {
		fmt.Println("\nFOUT tijdens extractie:", e)
		pause(rd)
		return
	}
	sha, _ := fileSHA256(out)
	fi, _ := os.Stat(out)
	fmt.Println("\nKLAAR")
	fmt.Println("Uitvoer:", out)
	fmt.Printf("Partijen: %d | ply: %d | grootte: %.2f MB\n", games, plies, float64(fi.Size())/(1024*1024))
	fmt.Println("SHA-256:", sha)
	if r.decodeErrors > 0 {
		fmt.Printf("Let op: %d CTG-zetdecodefouten zijn overgeslagen.\n", r.decodeErrors)
	}
	fmt.Println("\nGebruik: stream/importeer deze ene PGN in een LEGE CTG; leer daarna Metal.pgn in.")
	pause(rd)
}

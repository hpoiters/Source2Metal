package ctgmetal

import "strings"

// PGNEdge is a transposition-aware GAME observation exported to Source2Metal.
// A/B identify the exact board state (pieces, side, castling and effective EP),
// Move identifies the legal move from that state, and WhiteToMove defines the
// perspective used for W/D/L statistics.
type PGNEdge struct {
	A, B        uint64
	Move        uint16
	WhiteToMove bool
}

// ParsePGNMainline validates the complete mainline against the same legal board
// used by the CTG decoder. CanonicalSAN always contains the full legal game;
// Edges contains only the first maxPly observations used for opening evidence.
func ParsePGNMainline(tokens []string, maxPly int) (canonicalSAN []string, edges []PGNEdge, ok bool) {
	b := startBoard()
	canonicalSAN = make([]string, 0, len(tokens))
	if maxPly < 0 {
		maxPly = 0
	}
	edges = make([]PGNEdge, 0, minBridge(len(tokens), maxPly))
	for ply, tok := range tokens {
		legal := b.legalMoves()
		m, found := bridgeParseMove(b, legal, tok)
		if !found {
			return nil, nil, false
		}
		if ply < maxPly {
			a, bb := bridgePositionKey(b)
			edges = append(edges, PGNEdge{A: a, B: bb, Move: bridgeEncodeMove(m), WhiteToMove: b.Side == white})
		}
		canonicalSAN = append(canonicalSAN, b.san(m, legal))
		b = b.after(m)
	}
	return canonicalSAN, edges, len(canonicalSAN) > 0
}

func bridgeParseMove(b Board, legal []Move, tok string) (Move, bool) {
	n := bridgeNormalizeSAN(tok)
	for _, m := range legal {
		if bridgeNormalizeSAN(b.san(m, legal)) == n {
			return m, true
		}
	}
	// Coordinate notation occasionally occurs in engine PGNs.
	u := strings.ToLower(strings.TrimSpace(tok))
	u = strings.ReplaceAll(u, "-", "")
	u = strings.ReplaceAll(u, "x", "")
	u = strings.TrimRight(u, "+#?!")
	if len(u) == 4 || len(u) == 5 {
		from, ok1 := bridgeSquare(u[:2])
		to, ok2 := bridgeSquare(u[2:4])
		promo := 0
		if len(u) == 5 {
			promo = bridgePromo(u[4])
		}
		if ok1 && ok2 {
			for _, m := range legal {
				if m.From == from && m.To == to && (len(u) == 4 || m.Promo == promo) {
					return m, true
				}
			}
		}
	}
	return Move{}, false
}

func bridgeNormalizeSAN(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "0-0-0", "O-O-O")
	s = strings.ReplaceAll(s, "0-0", "O-O")
	s = strings.ReplaceAll(s, "e.p.", "")
	s = strings.ReplaceAll(s, "ep", "")
	s = strings.ReplaceAll(s, "=", "")
	for len(s) > 0 && strings.ContainsRune("+#?!", rune(s[len(s)-1])) {
		s = s[:len(s)-1]
	}
	return strings.TrimSpace(s)
}

func bridgeSquare(s string) (int, bool) {
	if len(s) != 2 {
		return 0, false
	}
	f := int(s[0] - 'a')
	r := int(s[1] - '1')
	if f < 0 || f > 7 || r < 0 || r > 7 {
		return 0, false
	}
	return r*8 + f, true
}
func bridgePromo(c byte) int {
	switch c {
	case 'q':
		return queen
	case 'r':
		return rook
	case 'b':
		return bishop
	case 'n':
		return knight
	}
	return 0
}
func bridgeEncodeMove(m Move) uint16 { return uint16(m.From | (m.To << 6) | (m.Promo << 12)) }

func bridgePositionKey(b Board) (uint64, uint64) {
	ep := b.fixedEP()
	h1 := uint64(1469598103934665603)
	h2 := uint64(7809847782465536322)
	const p1 uint64 = 1099511628211
	const p2 uint64 = 14029467366897019727
	mix := func(v uint64) { h1 ^= v; h1 *= p1; h2 ^= v + 0x9e3779b97f4a7c15; h2 *= p2; h2 ^= h2 >> 29 }
	for i, p := range b.Sq {
		mix(uint64(uint8(p+8)) | uint64(i+1)<<8)
	}
	mix(uint64(uint8(b.Side+2)) | uint64(uint8(b.Castle))<<8 | uint64(uint8(ep+1))<<16)
	return h1, h2
}
func minBridge(a, b int) int {
	if a < b {
		return a
	}
	return b
}

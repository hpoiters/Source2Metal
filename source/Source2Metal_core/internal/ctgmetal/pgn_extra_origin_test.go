package ctgmetal

import (
	"reflect"
	"strings"
	"testing"
)

func TestExtraOriginCanonicalPipeline(t *testing.T) {
	plain := strings.Fields("e4 e5 Nf3 Nc6 Bb5 a6 Bxc6 dxc6")
	wantSAN, wantEdges, ok := ParsePGNMainline(plain, 6)
	if !ok {
		t.Fatal("invalid test baseline")
	}
	for _, moves := range []string{
		"e4 e5 Ngf3 Nbc6 Bfb5 a6 Bbxc6 dxc6",
		"e4 e5 N1f3 N8c6 B1b5 a6 B5xc6 dxc6",
		"e4 e5 Ng1f3 Nb8c6 Bf1b5 a6 Bb5xc6 dxc6",
	} {
		san, edges, ok := ParsePGNMainline(strings.Fields(moves), 6)
		if !ok || !reflect.DeepEqual(san, wantSAN) || !reflect.DeepEqual(edges, wantEdges) {
			t.Fatalf("canonical SAN/observations differ: %s", moves)
		}
	}
	for _, moves := range []string{
		"e4 e5 Nbf3", "e4 e5 N2f3", "e4 e5 Ng2f3", "e4 e5 Ngxf3",
		"e4 e5 N1gf3", "e4 e5 Ngf3Q", "e4 e5 Ngf3=Q", "e4 e5 Ngf9",
		"e4 e5 Ng-f3", "e4 e5 Nggf3", "e4 e5 Ngef3", "e4 e5 Z0",
		"e4 e5 Nf3 Nc6 Bb5 a6 Bb5c6", "e4 e5 Nf3 Nc6 Bb5 a6 Z0",
	} {
		if _, _, ok := ParsePGNMainline(strings.Fields(moves), 2); ok {
			t.Fatalf("accepted invalid move, including beyond depth limit: %s", moves)
		}
	}
}

func TestExtraOriginAmbiguityAndPin(t *testing.T) {
	// Both knights can reach d4. A file-only origin is insufficient.
	b := Board{Side: white, EP: -1}
	b.Sq[4], b.Sq[63] = king, -king     // e1, h8
	b.Sq[12], b.Sq[44] = knight, knight // e2, e6
	legal := b.legalMoves()
	for _, token := range []string{"Nd4", "Ned4"} {
		if _, ok := bridgeParseMove(b, legal, token); ok {
			t.Fatalf("accepted ambiguous %s", token)
		}
	}
	for _, token := range []string{"N2d4", "Ne2d4"} {
		m, ok := bridgeParseMove(b, legal, token)
		if !ok || m.From != 12 || m.To != 27 {
			t.Fatalf("failed explicit origin %s: %+v", token, m)
		}
	}
	// Pin e2 to the king: only the c2 knight may move to d4.
	b.Sq[44], b.Sq[60], b.Sq[10] = 0, -rook, knight
	legal = b.legalMoves()
	for _, token := range []string{"Ncd4", "N2d4", "Nc2d4"} {
		m, ok := bridgeParseMove(b, legal, token)
		if !ok || m.From != 10 || m.To != 27 {
			t.Fatalf("failed legal move with pinned alternative %s: %+v", token, m)
		}
	}
	if _, ok := bridgeParseMove(b, legal, "Ne2d4"); ok {
		t.Fatal("accepted pinned knight move")
	}
}

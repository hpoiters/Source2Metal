package bin2pgn

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func makeSyntheticBIN(t *testing.T, path string) {
	t.Helper()
	type rec struct {
		k    uint64
		m, w uint16
		l    uint32
	}
	var rs []rec
	add := func(b Board, u string, w uint16) Board {
		m, ok := findLegalUCI(b, u)
		if !ok {
			t.Fatalf("move %s not legal", u)
		}
		rs = append(rs, rec{polyglotKey(b), encodePolyglotMoveForTest(m, b), w, 0})
		return b.after(m)
	}
	b0 := startBoard()
	bE4 := add(b0, "e2e4", 100)
	bD4 := add(b0, "d2d4", 80)
	bE4E5 := add(bE4, "e7e5", 100)
	_ = add(bE4, "c7c5", 90)
	_ = add(bE4E5, "g1f3", 100)
	_ = add(bD4, "d7d5", 100)
	sort.Slice(rs, func(i, j int) bool {
		if rs[i].k != rs[j].k {
			return rs[i].k < rs[j].k
		}
		return rs[i].m < rs[j].m
	})
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var buf [16]byte
	for _, r := range rs {
		binary.BigEndian.PutUint64(buf[:8], r.k)
		binary.BigEndian.PutUint16(buf[8:10], r.m)
		binary.BigEndian.PutUint16(buf[10:12], r.w)
		binary.BigEndian.PutUint32(buf[12:16], r.l)
		if _, err := f.Write(buf[:]); err != nil {
			t.Fatal(err)
		}
	}
}

func TestFullConversionAndMerge(t *testing.T) {
	d := t.TempDir()
	bin := filepath.Join(d, "test.bin")
	makeSyntheticBIN(t, bin)
	pgn := filepath.Join(d, "test.pgn")
	st, err := convertBook(bin, pgn, 10)
	if err != nil {
		t.Fatal(err)
	}
	if st.ReachableRecords != 6 || st.CompleteLines != 3 || st.CoverageMisses != 0 || st.PostValidatedLines != 3 {
		t.Fatalf("stats=%+v", st)
	}
	vr, err := validateWrittenPGN(pgn)
	if err != nil {
		t.Fatal(err)
	}
	if vr.Records != 3 {
		t.Fatalf("records=%d", vr.Records)
	}
	merged := filepath.Join(d, "merged.pgn")
	written, dup, err := mergeBookPGNs([]string{pgn, pgn}, merged)
	if err != nil {
		t.Fatal(err)
	}
	if written != 3 || dup != 3 {
		t.Fatalf("merge written=%d dup=%d", written, dup)
	}
	if _, err := os.Stat(merged); err != nil {
		t.Fatal(err)
	}
}

func TestIndexedConversionMatchesClassic(t *testing.T) {
	d := t.TempDir()
	bin := filepath.Join(d, "test.bin")
	makeSyntheticBIN(t, bin)

	book, err := openIndexedBook(bin, nil)
	if err != nil {
		t.Fatal(err)
	}
	if book.stats.Records != 6 || !book.stats.SortedByKey {
		book.Close()
		t.Fatalf("indexed stats=%+v", book.stats)
	}
	ents, err := book.lookup(polyglotKey(startBoard()))
	if err != nil {
		book.Close()
		t.Fatal(err)
	}
	if len(ents) != 2 || ents[0].Weight != 100 || ents[1].Weight != 80 {
		book.Close()
		t.Fatalf("start entries=%+v", ents)
	}
	if err := book.Close(); err != nil {
		t.Fatal(err)
	}

	pgn := filepath.Join(d, "indexed.pgn")
	st, err := convertBookIndexed(bin, pgn, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if st.Physical != 6 || st.ReachableRecords != 6 || st.CompleteLines != 3 || st.CoverageMisses != 0 || st.PostValidatedLines != 3 {
		t.Fatalf("indexed stats=%+v", st)
	}
}

func TestIndexedBookRejectsUnsortedInput(t *testing.T) {
	d := t.TempDir()
	bin := filepath.Join(d, "unsorted.bin")
	makeSyntheticBIN(t, bin)
	data, err := os.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 32 {
		t.Fatal("synthetic BIN unexpectedly short")
	}
	first := append([]byte(nil), data[:16]...)
	last := append([]byte(nil), data[len(data)-16:]...)
	copy(data[:16], last)
	copy(data[len(data)-16:], first)
	if err := os.WriteFile(bin, data, 0644); err != nil {
		t.Fatal(err)
	}
	if book, err := openIndexedBook(bin, nil); err == nil {
		book.Close()
		t.Fatal("unsorted BIN unexpectedly accepted by indexed reader")
	}
}

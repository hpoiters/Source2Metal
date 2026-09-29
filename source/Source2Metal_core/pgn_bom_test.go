package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestPGNInitialBOM(t *testing.T) {
	body := "[Event \"normal\"]\n[Result \"1-0\"]\n\n1. e4 e5 2. Nf3 Nc6 1-0\n"
	plain, err := readTestGames(t, body)
	if err != nil || len(plain) != 1 {
		t.Fatalf("baseline: records=%d err=%v", len(plain), err)
	}
	for _, prefix := range []string{"\uFEFF", "\uFEFF\n\n", "\n\n"} {
		got, err := readTestGames(t, prefix+body)
		if err != nil || len(got) != 1 {
			t.Fatalf("prefix %q: records=%d err=%v", prefix, len(got), err)
		}
		if !reflect.DeepEqual(got[0].Tags, plain[0].Tags) || got[0].MoveText != plain[0].MoveText {
			t.Fatalf("prefix %q changed the real record: %+v", prefix, got[0])
		}
	}
	for _, body := range []string{"", "\uFEFF", "\uFEFF\n\n"} {
		got, err := readTestGames(t, body)
		if err != nil || len(got) != 0 {
			t.Fatalf("empty input %q: records=%d err=%v", body, len(got), err)
		}
	}
}

func TestPGNBOMInsideCommentPreserved(t *testing.T) {
	body := "[Event \"normal\"]\n[Result \"1-0\"]\n1. e4 {\uFEFFtext} e5 1-0\n"
	got, err := readTestGames(t, body)
	if err != nil || len(got) != 1 || !strings.Contains(got[0].MoveText, "\uFEFFtext") {
		t.Fatalf("interior BOM changed: %+v err=%v", got, err)
	}
}

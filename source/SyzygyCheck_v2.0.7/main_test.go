package main

import "testing"

func TestTruncateMiddle(t *testing.T) {
	got := truncateMiddle("KRBNNvKN(FailTest).rtbw", 12)
	if runeLen(got) > 12 {
		t.Fatalf("too long: %q", got)
	}
	if got == "" {
		t.Fatal("empty")
	}
}

func TestProgressBarWidth(t *testing.T) {
	for _, p := range []int{0, 1, 50, 99, 100} {
		s := progressBar(p, 20)
		if runeLen(s) != 22 {
			t.Fatalf("pct=%d len=%d %q", p, runeLen(s), s)
		}
	}
}

func TestLanguageMenuNames(t *testing.T) {
	if len(langOrder) != 7 {
		t.Fatalf("languages=%d", len(langOrder))
	}
	for _, x := range langOrder {
		if x.display == "" {
			t.Fatalf("empty display for %s", x.code)
		}
	}
}

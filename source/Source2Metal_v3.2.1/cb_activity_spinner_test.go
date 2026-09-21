package main

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestCBSpinnerProgressLinePreservesProgress(t *testing.T) {
	start := time.Now().Add(-10 * time.Second)
	for _, format := range []string{"2CBH", "CBH"} {
		for _, frame := range []byte{'|', '/', '-', '\\'} {
			const width = 110
			got := cbSpinnerProgressLine(format, 76813, 3216031, start, width, frame)
			want := "  " + string(frame) + " " + strings.TrimPrefix(cbAdapterProgressLine(format, 76813, 3216031, start, width-2), "  ")
			if got != want {
				t.Fatalf("%s %c: spinner changed the progress payload: %q != %q", format, frame, got, want)
			}
			if !strings.Contains(got, "ETA ") || !strings.Contains(got, fmtInt(76813)+"/"+fmtInt(3216031)) {
				t.Fatalf("%s %c: missing original counts/ETA: %q", format, frame, got)
			}
			if utf8.RuneCountInString(got) > width {
				t.Fatalf("%s %c: exceeded console width: %d", format, frame, utf8.RuneCountInString(got))
			}
		}
	}
}

func TestCBSpinnerCompactAndUnknownTotal(t *testing.T) {
	for _, width := range []int{24, 50, 70, 90, 110} {
		for _, total := range []int64{0, 100} {
			got := cbSpinnerProgressLine("2CBH", 2, total, time.Now().Add(-3*time.Second), width, '/')
			if !strings.HasPrefix(got, "  / 2CBH") {
				t.Fatalf("spinner absent at width %d: %q", width, got)
			}
			if utf8.RuneCountInString(got) > width {
				t.Fatalf("overflow at width %d: %q", width, got)
			}
		}
	}
}

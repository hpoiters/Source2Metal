package main

import (
	"testing"
	"time"
)

func TestBINNightRestDuration(t *testing.T) {
	if binNightRestDuration != 10*time.Hour {
		t.Fatalf("night-rest pause = %v, want 10h", binNightRestDuration)
	}
}

func TestBINTimeFormatting(t *testing.T) {
	if got := formatBINElapsed(2*time.Hour + 3*time.Minute + 4*time.Second); got != "2h03m" {
		t.Fatalf("elapsed format = %q", got)
	}
	if got := formatBINETA(61 * time.Minute); got != "1 h 1 min" {
		t.Fatalf("ETA format = %q", got)
	}
}

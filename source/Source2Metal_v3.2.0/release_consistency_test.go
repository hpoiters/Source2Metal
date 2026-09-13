package main

import "testing"

func TestReleaseConsistency(t *testing.T) {
	if err := releaseConsistencyCheck(); err != nil {
		t.Fatal(err)
	}
}

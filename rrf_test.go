package simplykb

import "testing"

func TestReciprocalRank(t *testing.T) {
	if reciprocalRank(1, 60) <= reciprocalRank(2, 60) {
		t.Fatal("expected better rank to get larger score")
	}
	if reciprocalRank(0, 60) != 0 {
		t.Fatal("expected non-positive rank to be zero")
	}
}

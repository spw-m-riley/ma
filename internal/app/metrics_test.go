package app

import "testing"

func TestMeasure(t *testing.T) {
	stats := Measure("four words in here", "two words")

	if stats.InputWords != 4 {
		t.Fatalf("expected 4 input words, got %d", stats.InputWords)
	}
	if stats.OutputWords != 2 {
		t.Fatalf("expected 2 output words, got %d", stats.OutputWords)
	}
	if stats.InputBytes <= stats.OutputBytes {
		t.Fatalf("expected output bytes to be smaller than input bytes: %+v", stats)
	}
}

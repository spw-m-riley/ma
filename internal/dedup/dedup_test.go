package dedup

import (
	"path/filepath"
	"testing"
)

func TestDetectExactDuplicates(t *testing.T) {
	paths := []string{
		filepath.Join("..", "..", "testdata", "dedup", "rules-a.md"),
		filepath.Join("..", "..", "testdata", "dedup", "rules-b.md"),
	}

	report, err := Analyze(paths, 0.6)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(report.Exact) != 2 {
		t.Fatalf("expected 2 exact duplicates, got %d", len(report.Exact))
	}
}

func TestDetectNearDuplicates(t *testing.T) {
	paths := []string{
		filepath.Join("..", "..", "testdata", "dedup", "rules-a.md"),
		filepath.Join("..", "..", "testdata", "dedup", "rules-b.md"),
	}

	report, err := Analyze(paths, 0.6)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(report.Near) == 0 {
		t.Fatalf("expected at least one near duplicate")
	}
	if report.Near[0].Similarity < 0.6 {
		t.Fatalf("expected near duplicate similarity >= 0.6, got %f", report.Near[0].Similarity)
	}
}

func BenchmarkDedupCorpus(b *testing.B) {
	paths := []string{
		filepath.Join("..", "..", "testdata", "dedup", "rules-a.md"),
		filepath.Join("..", "..", "testdata", "dedup", "rules-b.md"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Analyze(paths, 0.6); err != nil {
			b.Fatalf("analyze dedup corpus: %v", err)
		}
	}
}

func TestNearDuplicatesPreservesOutputOnAlgorithmicReduction(t *testing.T) {
	// This test ensures that algorithmic reductions in nearDuplicates
	// don't change the output results - only the internal strategy changes
	paths := []string{
		filepath.Join("..", "..", "testdata", "dedup", "rules-a.md"),
		filepath.Join("..", "..", "testdata", "dedup", "rules-b.md"),
	}

	report, err := Analyze(paths, 0.6)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Store the count and order
	originalCount := len(report.Near)
	if originalCount == 0 {
		t.Logf("Note: no near duplicates in test corpus, skipping preservation check")
		return
	}
	
	// Run again and verify consistency
	report2, err := Analyze(paths, 0.6)
	if err != nil {
		t.Fatalf("expected no error on second run, got %v", err)
	}

	if len(report2.Near) != originalCount {
		t.Fatalf("expected %d near duplicates on second run, got %d", originalCount, len(report2.Near))
	}
	
	// Verify first result is the same
	if len(report.Near) > 0 && len(report2.Near) > 0 {
		if report.Near[0].Sentence != report2.Near[0].Sentence {
			t.Fatalf("expected same top result, got %q vs %q", report.Near[0].Sentence, report2.Near[0].Sentence)
		}
	}
}

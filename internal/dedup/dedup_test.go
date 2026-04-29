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

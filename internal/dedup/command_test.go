package dedup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandRunReturnsReadableReport(t *testing.T) {
	paths := []string{
		filepath.Join("..", "..", "testdata", "dedup", "rules-a.md"),
		filepath.Join("..", "..", "testdata", "dedup", "rules-b.md"),
	}

	result, err := NewCommand().Run(paths)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Command != "dedup" {
		t.Fatalf("expected dedup command, got %q", result.Command)
	}
	for _, expected := range []string{
		"Exact duplicates:",
		"Near duplicates:",
		"Always preserve code blocks exactly.",
	} {
		if !strings.Contains(result.Output, expected) {
			t.Fatalf("expected report to include %q, got %q", expected, result.Output)
		}
	}
}

func TestCommandRunMeasuresSourceContent(t *testing.T) {
	paths := []string{
		filepath.Join("..", "..", "testdata", "dedup", "rules-a.md"),
		filepath.Join("..", "..", "testdata", "dedup", "rules-b.md"),
	}

	first, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatalf("read first file: %v", err)
	}
	second, err := os.ReadFile(paths[1])
	if err != nil {
		t.Fatalf("read second file: %v", err)
	}

	result, err := NewCommand().Run(paths)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	wantBytes := len(first) + len(second)
	if result.Stats.InputBytes != wantBytes {
		t.Fatalf("expected input bytes %d, got %d", wantBytes, result.Stats.InputBytes)
	}
}

package dedup

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/spw-m-riley/ma/internal/app"
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

func TestCommandRunReportsGeneratedOutputWithoutChangeMetrics(t *testing.T) {
	paths := []string{
		filepath.Join("..", "..", "testdata", "dedup", "rules-a.md"),
		filepath.Join("..", "..", "testdata", "dedup", "rules-b.md"),
	}

	result, err := NewCommand().Run(paths)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Changed {
		t.Fatalf("expected dedup report to leave Changed=false")
	}
	if !result.ProducedOutput {
		t.Fatalf("expected dedup report to mark ProducedOutput=true")
	}
	if result.Stats != (app.Stats{}) {
		t.Fatalf("expected dedup report to omit savings stats, got %+v", result.Stats)
	}
}

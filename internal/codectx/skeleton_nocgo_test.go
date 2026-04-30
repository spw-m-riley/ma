//go:build !cgo

package codectx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkeletonTSFallsBackToHeuristicWithoutCGo(t *testing.T) {
	inputPath := filepath.Join("..", "..", "testdata", "code", "sample.ts")
	input, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read input: %v", err)
	}

	got, warnings, err := SkeletonFile(inputPath, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(warnings) != 1 || warnings[0] != "heuristic skeleton used for non-Go source" {
		t.Fatalf("expected heuristic warning, got %#v", warnings)
	}
	for _, expected := range []string{
		"export function render(config: Config): string;",
		"export function save(config: Config, value: string): void;",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected heuristic skeleton to include %q, got %q", expected, got)
		}
	}
	if strings.Contains(got, "return readFileSync") || strings.Contains(got, "writeFileSync(config.path, value);") {
		t.Fatalf("expected function bodies to be removed, got %q", got)
	}
}

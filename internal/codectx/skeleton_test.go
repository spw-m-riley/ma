package codectx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkeletonGo(t *testing.T) {
	inputPath := filepath.Join("..", "..", "testdata", "code", "sample.go")
	input, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read input: %v", err)
	}

	got, warnings, err := SkeletonFile(inputPath, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}

	want := "package sample\n\nimport \"context\"\n\n// Process applies the configured operation.\nfunc Process(ctx context.Context, value string) (string, error)\n"
	if got != want {
		t.Fatalf("unexpected go skeleton\nwant: %q\ngot:  %q", want, got)
	}
}

func TestSkeletonTSHeuristicReturnsWarning(t *testing.T) {
	inputPath := filepath.Join("..", "..", "testdata", "code", "sample.ts")
	input, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read input: %v", err)
	}

	got, warnings, err := SkeletonFile(inputPath, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(warnings) == 0 {
		t.Fatalf("expected heuristic warning")
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

func BenchmarkSkeleton(b *testing.B) {
	paths := []string{
		filepath.Join("..", "..", "testdata", "code", "sample.go"),
		filepath.Join("..", "..", "testdata", "code", "sample.ts"),
	}

	inputs := make([][]byte, 0, len(paths))
	for _, path := range paths {
		input, err := os.ReadFile(path)
		if err != nil {
			b.Fatalf("read input %s: %v", path, err)
		}
		inputs = append(inputs, input)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for index, path := range paths {
			if _, _, err := SkeletonFile(path, inputs[index]); err != nil {
				b.Fatalf("skeleton %s: %v", path, err)
			}
		}
	}
}

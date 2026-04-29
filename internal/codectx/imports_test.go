package codectx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spw-m-riley/ma/internal/app"
	"github.com/spw-m-riley/ma/internal/testutil"
)

func TestTrimImports(t *testing.T) {
	inputPath := filepath.Join("..", "..", "testdata", "code", "sample.ts")
	input, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read input: %v", err)
	}

	got, warnings, err := TrimImportsFile(inputPath, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", warnings)
	}

	for _, expected := range []string{
		"// imports: node:fs(readFileSync, writeFileSync)",
		"// types: Config from ./types",
		"export function render(config: Config): string {",
		"export function save(config: Config, value: string): void {",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected trimmed imports output to include %q, got %q", expected, got)
		}
	}
	if strings.Contains(got, "import {") || strings.Contains(got, "import type {") {
		t.Fatalf("expected original import lines to be removed, got %q", got)
	}
}

func TestCodeContextReduction(t *testing.T) {
	goInputPath := filepath.Join("..", "..", "testdata", "code", "sample.go")
	goInput, err := os.ReadFile(goInputPath)
	if err != nil {
		t.Fatalf("read go input: %v", err)
	}
	goOutput, _, err := SkeletonFile(goInputPath, goInput)
	if err != nil {
		t.Fatalf("skeleton go: %v", err)
	}

	tsInputPath := filepath.Join("..", "..", "testdata", "code", "sample.ts")
	tsInput, err := os.ReadFile(tsInputPath)
	if err != nil {
		t.Fatalf("read ts input: %v", err)
	}
	tsOutput, _, err := SkeletonFile(tsInputPath, tsInput)
	if err != nil {
		t.Fatalf("skeleton ts: %v", err)
	}

	stats := app.Measure(string(goInput)+string(tsInput), goOutput+tsOutput)
	if err := testutil.AssertApproxTokenReductionAtLeast(stats, 60); err != nil {
		t.Fatalf("expected code-context reduction to meet target: %v", err)
	}
}

func TestTrimImportsReduction(t *testing.T) {
	inputPath := filepath.Join("..", "..", "testdata", "code", "import-heavy.ts")
	input, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read input: %v", err)
	}

	output, _, err := TrimImportsFile(inputPath, input)
	if err != nil {
		t.Fatalf("trim imports: %v", err)
	}

	stats := app.Measure(string(input), output)
	if err := testutil.AssertApproxTokenReductionAtLeast(stats, 30); err != nil {
		t.Fatalf("expected trim-imports reduction to meet target: %v", err)
	}
}

func BenchmarkTrimImports(b *testing.B) {
	inputPath := filepath.Join("..", "..", "testdata", "code", "import-heavy.ts")
	input, err := os.ReadFile(inputPath)
	if err != nil {
		b.Fatalf("read input: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := TrimImportsFile(inputPath, input); err != nil {
			b.Fatalf("trim imports: %v", err)
		}
	}
}

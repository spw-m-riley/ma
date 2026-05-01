package codectx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkeletonCommandRunReturnsSkeletonOutput(t *testing.T) {
	inputPath := filepath.Join("..", "..", "testdata", "code", "sample.go")

	result, err := NewSkeletonCommand().Run([]string{inputPath})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Command != "skeleton" {
		t.Fatalf("expected skeleton command, got %q", result.Command)
	}
	if !strings.Contains(result.Output, "func Process(ctx context.Context, value string) (string, error)") {
		t.Fatalf("unexpected skeleton output %q", result.Output)
	}
}

func TestTrimImportsCommandRunReturnsTrimmedOutput(t *testing.T) {
	inputPath := filepath.Join("..", "..", "testdata", "code", "sample.ts")

	result, err := NewTrimImportsCommand().Run([]string{inputPath})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Command != "trim-imports" {
		t.Fatalf("expected trim-imports command, got %q", result.Command)
	}
	if !strings.Contains(result.Output, "// imports: node:fs(readFileSync, writeFileSync)") {
		t.Fatalf("unexpected trim-imports output %q", result.Output)
	}
}

func TestTrimImportsCommandRunReturnsGoTrimmedOutput(t *testing.T) {
	inputPath := filepath.Join("..", "..", "testdata", "code", "sample.go")

	result, err := NewTrimImportsCommand().Run([]string{inputPath})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Command != "trim-imports" {
		t.Fatalf("expected trim-imports command, got %q", result.Command)
	}
	if !strings.Contains(result.Output, "// imports: context") {
		t.Fatalf("unexpected go trim-imports output %q", result.Output)
	}
}

func TestSkeletonCommandDoesNotWriteFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.go")
	original, err := os.ReadFile(filepath.Join("..", "..", "testdata", "code", "sample.go"))
	if err != nil {
		t.Fatalf("read source fixture: %v", err)
	}
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatalf("write temp source: %v", err)
	}

	if _, err := NewSkeletonCommand().Run([]string{path}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read temp source: %v", err)
	}
	if string(after) != string(original) {
		t.Fatalf("expected command to be read-only")
	}
}

func TestSkeletonCommandReportsChangedFalseWhenOutputMatchesInput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "constants.go")
	content := "package main\n\nconst version = \"1.0\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp source: %v", err)
	}

	result, err := NewSkeletonCommand().Run([]string{path})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Changed {
		t.Fatalf("expected unchanged skeleton output to report Changed=false")
	}
}

func TestTrimImportsCommandReportsChangedFalseWhenOutputMatchesInput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "constants.go")
	content := "package main\n\nconst version = \"1.0\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp source: %v", err)
	}

	result, err := NewTrimImportsCommand().Run([]string{path})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Changed {
		t.Fatalf("expected unchanged trim-imports output to report Changed=false")
	}
}

package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandRunReturnsCompactedTranscript(t *testing.T) {
	inputPath := filepath.Join("..", "..", "testdata", "history", "transcript.json")

	result, err := NewCommand().Run([]string{inputPath})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Command != "compact-history" {
		t.Fatalf("expected compact-history command, got %q", result.Command)
	}
	if !strings.Contains(result.Output, "FAIL x8") {
		t.Fatalf("unexpected compacted transcript output %q", result.Output)
	}
}

func TestCommandRunWriteCreatesBackupAndReplacesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.json")
	originalBytes, err := os.ReadFile(filepath.Join("..", "..", "testdata", "history", "transcript.json"))
	if err != nil {
		t.Fatalf("read source fixture: %v", err)
	}
	if err := os.WriteFile(path, originalBytes, 0o644); err != nil {
		t.Fatalf("write temp transcript: %v", err)
	}

	result, err := NewCommand().Run([]string{"--write", path})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected write run to report a change")
	}

	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated transcript: %v", err)
	}
	if !strings.Contains(string(updated), "FAIL x8") {
		t.Fatalf("unexpected updated transcript %q", string(updated))
	}

	backup, err := os.ReadFile(path + ".ma.bak")
	if err != nil {
		t.Fatalf("read backup transcript: %v", err)
	}
	if string(backup) != string(originalBytes) {
		t.Fatalf("unexpected backup transcript %q", string(backup))
	}
}

func TestCommandRunReportsChangedFalseWhenTranscriptIsAlreadyCompact(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "transcript.json")
	content := `[{"role":"user","content":"ok"}]`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp transcript: %v", err)
	}

	result, err := NewCommand().Run([]string{path})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Changed {
		t.Fatalf("expected already compact transcript to report Changed=false")
	}
}

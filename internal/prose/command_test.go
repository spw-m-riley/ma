package prose

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCommandRunReturnsCompressedOutputWithoutWriting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.md")
	original := "Please make sure to utilize concise wording.\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	command := NewCommand()
	result, err := command.Run([]string{path})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Command != "compress" {
		t.Fatalf("expected compress command, got %q", result.Command)
	}
	if result.Output != "ensure use concise wording.\n" {
		t.Fatalf("unexpected output %q", result.Output)
	}
	if !result.Changed {
		t.Fatalf("expected command to report changed output")
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read original file: %v", err)
	}
	if string(after) != original {
		t.Fatalf("expected source file to remain unchanged, got %q", string(after))
	}
}

func TestCommandRunWriteCreatesBackupAndReplacesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.md")
	original := "Please make sure to utilize concise wording.\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	command := NewCommand()
	result, err := command.Run([]string{"--write", path})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected write run to report a change")
	}

	updated, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	if string(updated) != "ensure use concise wording.\n" {
		t.Fatalf("unexpected updated file %q", string(updated))
	}

	backup, err := os.ReadFile(path + ".ma.bak")
	if err != nil {
		t.Fatalf("read backup file: %v", err)
	}
	if string(backup) != original {
		t.Fatalf("unexpected backup file %q", string(backup))
	}
}

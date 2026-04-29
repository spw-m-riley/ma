package validate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCommandRunValidatesMatchingFiles(t *testing.T) {
	dir := t.TempDir()
	originalPath := filepath.Join(dir, "original.md")
	candidatePath := filepath.Join(dir, "candidate.md")
	content := "# Heading\n\nBody.\n"

	if err := os.WriteFile(originalPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write original: %v", err)
	}
	if err := os.WriteFile(candidatePath, []byte(content), 0o644); err != nil {
		t.Fatalf("write candidate: %v", err)
	}

	command := NewCommand()
	result, err := command.Run([]string{originalPath, candidatePath})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Command != "validate" {
		t.Fatalf("expected validate command, got %q", result.Command)
	}
	if result.Output != "valid\n" {
		t.Fatalf("unexpected output %q", result.Output)
	}
}

func TestCommandRunReturnsErrorOnValidationFailure(t *testing.T) {
	dir := t.TempDir()
	originalPath := filepath.Join(dir, "original.md")
	candidatePath := filepath.Join(dir, "candidate.md")

	if err := os.WriteFile(originalPath, []byte("# Heading\n"), 0o644); err != nil {
		t.Fatalf("write original: %v", err)
	}
	if err := os.WriteFile(candidatePath, []byte("# Other Heading\n"), 0o644); err != nil {
		t.Fatalf("write candidate: %v", err)
	}

	command := NewCommand()
	if _, err := command.Run([]string{originalPath, candidatePath}); err == nil {
		t.Fatalf("expected validation failure")
	}
}

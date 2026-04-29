package markdown

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandRunReturnsOptimizedOutputWithoutWriting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	original := "# Guide\n\n\n* first item\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	command := NewCommand()
	result, err := command.Run([]string{path})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Command != "optimize-md" {
		t.Fatalf("expected optimize-md command, got %q", result.Command)
	}
	if result.Output != "# Guide\n\n- first item\n" {
		t.Fatalf("unexpected output %q", result.Output)
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
	path := filepath.Join(dir, "guide.md")
	original := "# Guide\n\n\n* first item\n"
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
	if string(updated) != "# Guide\n\n- first item\n" {
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

func TestCommandRunRejectsSymlinkToSensitivePath(t *testing.T) {
	dir := t.TempDir()
	
	// Create a sensitive file (.ssh directory)
	sshDir := filepath.Join(dir, ".ssh")
	os.MkdirAll(sshDir, 0o700)
	secretFile := filepath.Join(sshDir, "id_rsa")
	os.WriteFile(secretFile, []byte("secret key"), 0o600)
	
	// Create a symlink with a benign name
	symlinkPath := filepath.Join(dir, "document.md")
	os.Symlink(secretFile, symlinkPath)
	
	// Attempt to optimize-md through the symlink - should fail
	command := NewCommand()
	_, err := command.Run([]string{symlinkPath})
	
	if err == nil {
		t.Fatalf("expected error when accessing file through symlink to sensitive path")
	}
	if !strings.Contains(err.Error(), "refusing sensitive path") {
		t.Fatalf("expected 'refusing sensitive path' error, got: %v", err)
	}
}

func TestCommandRunAcceptsNormalSymlink(t *testing.T) {
	dir := t.TempDir()
	
	// Create a normal markdown file
	mdPath := filepath.Join(dir, "real.md")
	os.WriteFile(mdPath, []byte("# Title\n\n\n* item\n"), 0o644)
	
	// Create a symlink to it
	symlinkPath := filepath.Join(dir, "link.md")
	os.Symlink(mdPath, symlinkPath)
	
	// Accessing through normal symlink should work fine
	command := NewCommand()
	result, err := command.Run([]string{symlinkPath})
	
	if err != nil {
		t.Fatalf("expected no error with normal symlink, got: %v", err)
	}
	if result.Command != "optimize-md" {
		t.Fatalf("expected optimize-md command, got %q", result.Command)
	}
}

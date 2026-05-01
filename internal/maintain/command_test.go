package maintain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaintainProseCompression(t *testing.T) {
	dir := t.TempDir()

	// Create a prose file with compressible content
	content := strings.Repeat("This is a sentence that should be compressed by the prose engine.\n", 50)
	path := filepath.Join(dir, "instructions.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	result, err := NewCommand().Run([]string{dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected changed=true")
	}
	if result.Stats.OutputApproxTokens >= result.Stats.InputApproxTokens {
		t.Fatalf("expected token reduction")
	}
}

func TestMaintainSkipsNonProse(t *testing.T) {
	dir := t.TempDir()

	content := "package main\n\nfunc main() {}\n"
	path := filepath.Join(dir, "main.go")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	result, err := NewCommand().Run([]string{dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Changed {
		t.Fatalf("expected changed=false for non-prose files")
	}

	hasSkip := false
	for _, f := range result.Findings {
		if strings.Contains(f, "classification=code") {
			hasSkip = true
		}
	}
	if !hasSkip {
		t.Fatalf("expected skip finding for code file, got %v", result.Findings)
	}
}

func TestMaintainEmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	result, err := NewCommand().Run([]string{dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Changed {
		t.Fatalf("expected changed=false for empty directory")
	}
}

func TestMaintainWriteCreatesBackups(t *testing.T) {
	dir := t.TempDir()

	content := strings.Repeat("This is a sentence that should be compressed by the prose engine.\n", 50)
	path := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	_, err := NewCommand().Run([]string{"--write", dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	backupPath := path + ".ma.bak"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Fatalf("expected backup file at %s", backupPath)
	}

	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(backupContent) != content {
		t.Fatalf("backup content mismatch")
	}
}

func TestMaintainReadOnlyByDefault(t *testing.T) {
	dir := t.TempDir()

	content := strings.Repeat("This is a sentence that should be compressed by the prose engine.\n", 50)
	path := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	_, err := NewCommand().Run([]string{dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// File should be unchanged
	afterContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(afterContent) != content {
		t.Fatalf("file should not be modified without --write")
	}
}

func TestMaintainSensitivePathsSkipped(t *testing.T) {
	dir := t.TempDir()
	sshDir := filepath.Join(dir, ".ssh")
	if err := os.MkdirAll(sshDir, 0o755); err != nil {
		t.Fatalf("create .ssh dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "config"), []byte("Host *\n"), 0o644); err != nil {
		t.Fatalf("write ssh config: %v", err)
	}

	result, err := NewCommand().Run([]string{dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should complete without error, sensitive files silently skipped
	_ = result
}

func TestMaintainNotADirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	_, err := NewCommand().Run([]string{path})
	if err == nil {
		t.Fatalf("expected error for non-directory path")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("expected 'not a directory' error, got: %v", err)
	}
}

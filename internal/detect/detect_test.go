package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClassifyMarkdownAsNaturalLanguage(t *testing.T) {
	got := Classify("README.md", "# Title\n\nSome prose.\n")
	if got != NaturalLanguage {
		t.Fatalf("expected natural language, got %q", got)
	}
}

func TestClassifyGoAsCode(t *testing.T) {
	got := Classify("main.go", "package main\n")
	if got != Code {
		t.Fatalf("expected code, got %q", got)
	}
}

func TestClassifyJSONAsConfig(t *testing.T) {
	got := Classify("tool.schema.json", "{\"type\":\"object\"}")
	if got != Config {
		t.Fatalf("expected config, got %q", got)
	}
}

func TestClassifyBinaryAsSkip(t *testing.T) {
	got := Classify("logo.png", "")
	if got != Skip {
		t.Fatalf("expected skip, got %q", got)
	}
}

func TestClassifyExtensionlessJSONAsConfig(t *testing.T) {
	got := Classify("config", "{\"mode\":\"safe\"}")
	if got != Config {
		t.Fatalf("expected config, got %q", got)
	}
}

func TestIsSensitivePath(t *testing.T) {
	for _, path := range []string{
		"/Users/matthew/.ssh/id_rsa",
		"/Users/matthew/work/project/.env",
		"/Users/matthew/.aws/credentials",
	} {
		if !IsSensitivePath(path) {
			t.Fatalf("expected %q to be sensitive", path)
		}
	}
}

func TestIsSensitivePathWithSymlink(t *testing.T) {
	// Create a temporary directory with a sensitive target
	tmpDir := t.TempDir()
	sshDir := filepath.Join(tmpDir, ".ssh")
	if err := os.Mkdir(sshDir, 0o700); err != nil {
		t.Fatalf("failed to create .ssh dir: %v", err)
	}

	secretFile := filepath.Join(sshDir, "id_rsa")
	if err := os.WriteFile(secretFile, []byte("secret"), 0o600); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	// Create a symlink to the sensitive file with a benign name
	symlinkPath := filepath.Join(tmpDir, "notes.md")
	if err := os.Symlink(secretFile, symlinkPath); err != nil {
		t.Fatalf("failed to create symlink: %v", err)
	}

	// IsSensitivePathResolved should detect the sensitive target through the symlink
	if !IsSensitivePathResolved(symlinkPath) {
		t.Fatalf("expected symlink pointing to sensitive file %q to be blocked", symlinkPath)
	}
}

func TestIsSensitivePathResolvedWithBrokenSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create a symlink to a non-existent target
	brokenSymlink := filepath.Join(tmpDir, "broken.md")
	nonExistent := filepath.Join(tmpDir, "does-not-exist")
	if err := os.Symlink(nonExistent, brokenSymlink); err != nil {
		t.Fatalf("failed to create broken symlink: %v", err)
	}

	// IsSensitivePathResolved should fail closed on unresolvable symlinks
	if !IsSensitivePathResolved(brokenSymlink) {
		t.Fatalf("expected broken symlink to fail closed")
	}
}

func TestIsSensitivePathResolvedWithOrdinaryMissingFile(t *testing.T) {
	// A regular missing file (not a symlink) should not be treated as sensitive
	missingPath := "/tmp/this-file-does-not-exist-12345.md"
	if IsSensitivePathResolved(missingPath) {
		t.Fatalf("expected ordinary missing file to not be treated as sensitive")
	}
}

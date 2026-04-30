package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spw-m-riley/ma/internal/dashboard"
)

func TestAppHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	command := newRootCommand(&stdout, &stderr)
	command.SetArgs([]string{"--help"})

	if err := command.Execute(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	help := stdout.String()
	for _, name := range []string{
		"compress",
		"validate",
		"optimize-md",
		"minify-schema",
		"skeleton",
		"trim-imports",
		"dedup",
		"compact-history",
		"dashboard",
	} {
		if !strings.Contains(help, name) {
			t.Fatalf("expected help output to include command %q, got %q", name, help)
		}
	}
}

func TestCompressAllowsTrailingJSONFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "notes.md")
	if err := os.WriteFile(path, []byte("Please make sure to utilize concise wording.\n"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	command := newRootCommand(&stdout, &stderr)
	command.SetArgs([]string{"compress", path, "--json"})

	if err := command.Execute(); err != nil {
		t.Fatalf("expected no error, got %v (stderr=%q)", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "\"command\":\"compress\"") {
		t.Fatalf("expected json command output, got %q", stdout.String())
	}
}

func TestValidateAllowsTrailingJSONFlag(t *testing.T) {
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

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	command := newRootCommand(&stdout, &stderr)
	command.SetArgs([]string{"validate", originalPath, candidatePath, "--json"})

	if err := command.Execute(); err != nil {
		t.Fatalf("expected no error, got %v (stderr=%q)", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "\"command\":\"validate\"") {
		t.Fatalf("expected json command output, got %q", stdout.String())
	}
}

func TestOptimizeMarkdownAllowsTrailingJSONFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	if err := os.WriteFile(path, []byte("# Guide\n\n\n* first item\n"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	command := newRootCommand(&stdout, &stderr)
	command.SetArgs([]string{"optimize-md", path, "--json"})

	if err := command.Execute(); err != nil {
		t.Fatalf("expected no error, got %v (stderr=%q)", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "\"command\":\"optimize-md\"") {
		t.Fatalf("expected json command output, got %q", stdout.String())
	}
}

func TestMinifySchemaAllowsTrailingJSONFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tool.schema.json")
	if err := os.WriteFile(path, []byte("{\"description\":\"verbose\",\"type\":\"object\"}\n"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	command := newRootCommand(&stdout, &stderr)
	command.SetArgs([]string{"minify-schema", path, "--json"})

	if err := command.Execute(); err != nil {
		t.Fatalf("expected no error, got %v (stderr=%q)", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "\"command\":\"minify-schema\"") {
		t.Fatalf("expected json command output, got %q", stdout.String())
	}
}

func TestSkeletonAllowsTrailingJSONFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	command := newRootCommand(&stdout, &stderr)
	command.SetArgs([]string{"skeleton", filepath.Join("..", "..", "testdata", "code", "sample.go"), "--json"})

	if err := command.Execute(); err != nil {
		t.Fatalf("expected no error, got %v (stderr=%q)", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "\"command\":\"skeleton\"") {
		t.Fatalf("expected json command output, got %q", stdout.String())
	}
}

func TestTrimImportsAllowsTrailingJSONFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	command := newRootCommand(&stdout, &stderr)
	command.SetArgs([]string{"trim-imports", filepath.Join("..", "..", "testdata", "code", "sample.ts"), "--json"})

	if err := command.Execute(); err != nil {
		t.Fatalf("expected no error, got %v (stderr=%q)", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "\"command\":\"trim-imports\"") {
		t.Fatalf("expected json command output, got %q", stdout.String())
	}
}

func TestDedupAllowsTrailingJSONFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	command := newRootCommand(&stdout, &stderr)
	command.SetArgs([]string{
		"dedup",
		filepath.Join("..", "..", "testdata", "dedup", "rules-a.md"),
		filepath.Join("..", "..", "testdata", "dedup", "rules-b.md"),
		"--json",
	})

	if err := command.Execute(); err != nil {
		t.Fatalf("expected no error, got %v (stderr=%q)", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "\"command\":\"dedup\"") {
		t.Fatalf("expected json command output, got %q", stdout.String())
	}
}

func TestCompactHistoryAllowsTrailingJSONFlag(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	command := newRootCommand(&stdout, &stderr)
	command.SetArgs([]string{
		"compact-history",
		filepath.Join("..", "..", "testdata", "history", "transcript.json"),
		"--json",
	})

	if err := command.Execute(); err != nil {
		t.Fatalf("expected no error, got %v (stderr=%q)", err, stderr.String())
	}

	if !strings.Contains(stdout.String(), "\"command\":\"compact-history\"") {
		t.Fatalf("expected json command output, got %q", stdout.String())
	}
}

func TestCommandInvocationRecordsDashboardHistory(t *testing.T) {
	stateDir := t.TempDir()
	t.Setenv("MA_DASHBOARD_STATE_DIR", stateDir)

	path := filepath.Join(stateDir, "notes.md")
	if err := os.WriteFile(path, []byte("Please make sure to utilize concise wording.\n"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	command := newRootCommand(&stdout, &stderr)
	command.SetArgs([]string{"compress", path})

	if err := command.Execute(); err != nil {
		t.Fatalf("expected no error, got %v (stderr=%q)", err, stderr.String())
	}

	store, err := dashboard.OpenStore(stateDir)
	if err != nil {
		t.Fatalf("open dashboard store: %v", err)
	}

	summary, err := store.Summary()
	if err != nil {
		t.Fatalf("read dashboard summary: %v", err)
	}

	if summary.TotalRuns != 1 {
		t.Fatalf("expected 1 recorded run, got %d", summary.TotalRuns)
	}
	if summary.SuccessfulRuns != 1 {
		t.Fatalf("expected 1 successful run, got %d", summary.SuccessfulRuns)
	}
	if got := summary.CommandUsage["compress"]; got != 1 {
		t.Fatalf("expected compress usage count 1, got %d", got)
	}
}

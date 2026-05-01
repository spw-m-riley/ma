package smartread

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSmartReadNaturalLanguage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.md")

	// Generate a file > 200 lines of prose
	var lines []string
	for i := 0; i < 250; i++ {
		lines = append(lines, "This is a sentence that should be compressed by the prose compression engine.")
	}
	content := strings.Join(lines, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	result, err := NewCommand().Run([]string{path})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected changed=true for prose file")
	}
	if result.Stats.OutputBytes >= result.Stats.InputBytes {
		t.Fatalf("expected output smaller than input, got %d >= %d", result.Stats.OutputBytes, result.Stats.InputBytes)
	}

	hasClassification := false
	for _, f := range result.Findings {
		if strings.Contains(f, "classification=natural_language") {
			hasClassification = true
		}
	}
	if !hasClassification {
		t.Fatalf("expected classification finding, got %v", result.Findings)
	}
}

func TestSmartReadBelowThreshold(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "short.md")

	content := "# Short file\n\nJust a few lines.\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	result, err := NewCommand().Run([]string{path})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Changed {
		t.Fatalf("expected changed=false for short file")
	}
	if result.Output != content {
		t.Fatalf("expected passthrough content")
	}

	hasPassthrough := false
	for _, f := range result.Findings {
		if strings.Contains(f, "passthrough=below_threshold") {
			hasPassthrough = true
		}
	}
	if !hasPassthrough {
		t.Fatalf("expected passthrough finding, got %v", result.Findings)
	}
}

func TestSmartReadSensitivePath(t *testing.T) {
	_, err := NewCommand().Run([]string{"/home/user/.ssh/id_rsa"})
	if err == nil {
		t.Fatalf("expected error for sensitive path")
	}
	if !strings.Contains(err.Error(), "refusing sensitive path") {
		t.Fatalf("expected sensitive path error, got: %v", err)
	}
}

func TestSmartReadCodeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")

	// Generate a Go file > 200 lines
	var lines []string
	lines = append(lines, "package main")
	lines = append(lines, "")
	for i := 0; i < 210; i++ {
		lines = append(lines, fmt.Sprintf("func example%d() {}", i))
	}
	content := strings.Join(lines, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	result, err := NewCommand().Run([]string{path})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hasClassification := false
	for _, f := range result.Findings {
		if strings.Contains(f, "classification=code") {
			hasClassification = true
		}
	}
	if !hasClassification {
		t.Fatalf("expected code classification, got %v", result.Findings)
	}
}

func TestSmartReadCodeFileComposesImportReductionAndSkeleton(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "main.go")

	var lines []string
	lines = append(lines, "package main")
	lines = append(lines, "")
	lines = append(lines, "import \"context\"")
	lines = append(lines, "")
	for i := 0; i < 210; i++ {
		lines = append(lines, fmt.Sprintf("func example%d(ctx context.Context) error {", i))
		lines = append(lines, "\treturn ctx.Err()")
		lines = append(lines, "}")
		lines = append(lines, "")
	}
	content := strings.Join(lines, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	result, err := NewCommand().Run([]string{path})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result.Output, "// imports: context") {
		t.Fatalf("expected import summary in output, got %q", result.Output)
	}
	if strings.Contains(result.Output, "return ctx.Err()") {
		t.Fatalf("expected function bodies to be removed, got %q", result.Output)
	}
	if !strings.Contains(result.Output, "func example0(ctx context.Context) error") {
		t.Fatalf("expected skeletonized function signature, got %q", result.Output)
	}
}

func TestSmartReadReductionFailureFallback(t *testing.T) {
	dir := t.TempDir()
	// A file with an extension that classifies as Config but can't be minified
	path := filepath.Join(dir, "data.toml")

	var lines []string
	for i := 0; i < 250; i++ {
		lines = append(lines, fmt.Sprintf("key%d = \"value%d\"", i, i))
	}
	content := strings.Join(lines, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	result, err := NewCommand().Run([]string{path})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should fall back to passthrough on reduction failure
	if result.Changed {
		t.Fatalf("expected changed=false on reduction fallback")
	}
	if result.Output != content {
		t.Fatalf("expected passthrough content on reduction failure")
	}
}

func TestSmartReadUnsupportedConfigUsesExplicitUnsupportedPassthrough(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.toml")

	var lines []string
	for i := 0; i < 250; i++ {
		lines = append(lines, fmt.Sprintf("key%d = \"value%d\"", i, i))
	}
	content := strings.Join(lines, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	result, err := NewCommand().Run([]string{path})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Changed {
		t.Fatalf("expected unsupported config passthrough to keep Changed=false")
	}
	assertHasFinding(t, result.Findings, "classification=config")
	assertHasFinding(t, result.Findings, "passthrough=unsupported_reducer")
}

func TestSmartReadUnsupportedCodeUsesExplicitUnsupportedPassthrough(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "script.py")

	var lines []string
	for i := 0; i < 250; i++ {
		lines = append(lines, fmt.Sprintf("def example_%d():", i))
		lines = append(lines, "    return 1")
	}
	content := strings.Join(lines, "\n")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	result, err := NewCommand().Run([]string{path})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Changed {
		t.Fatalf("expected unsupported code passthrough to keep Changed=false")
	}
	assertHasFinding(t, result.Findings, "classification=code")
	assertHasFinding(t, result.Findings, "passthrough=unsupported_reducer")
}

func assertHasFinding(t *testing.T, findings []string, want string) {
	t.Helper()
	for _, finding := range findings {
		if finding == want {
			return
		}
	}
	t.Fatalf("expected findings %v to include %q", findings, want)
}

package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spw-m-riley/ma/internal/app"
	"github.com/spw-m-riley/ma/internal/testutil"
)

func TestCompactHistoryFixtures(t *testing.T) {
	inputPath := filepath.Join("..", "..", "testdata", "history", "transcript.json")
	expectedPath := filepath.Join("..", "..", "testdata", "history", "transcript.expected.json")

	input, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read input: %v", err)
	}
	expected, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}

	got, err := CompactJSON(string(input))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != strings.TrimSpace(string(expected)) {
		t.Fatalf("unexpected compacted transcript\nwant: %q\ngot:  %q", strings.TrimSpace(string(expected)), got)
	}
}

func TestCompactHistoryReduction(t *testing.T) {
	inputPath := filepath.Join("..", "..", "testdata", "history", "transcript.json")
	input, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read input: %v", err)
	}

	output, err := CompactJSON(string(input))
	if err != nil {
		t.Fatalf("compact transcript: %v", err)
	}

	stats := app.Measure(string(input), output)
	if err := testutil.AssertApproxTokenReductionAtLeast(stats, 30); err != nil {
		t.Fatalf("expected history compaction reduction to meet target: %v", err)
	}
}

func TestCompactRejectsMalformedJSON(t *testing.T) {
	if _, err := CompactJSON("{not-json"); err == nil {
		t.Fatalf("expected malformed transcript error")
	}
}

func TestCompactCollapsesDuplicateReadsToLatest(t *testing.T) {
	messages := []Message{
		{Role: "assistant", ToolName: "view", FilePath: "README.md", Content: "old"},
		{Role: "assistant", ToolName: "view", FilePath: "README.md", Content: "new"},
	}

	compacted := Compact(messages)
	if len(compacted) != 1 {
		t.Fatalf("expected only latest file read to remain, got %d messages", len(compacted))
	}
	if compacted[0].Content != "new" {
		t.Fatalf("expected latest file read to remain, got %q", compacted[0].Content)
	}
}

func TestCompactPreservesDifferentToolsOnSameFile(t *testing.T) {
	// A read and an edit of the same file should both be kept - they're different operations
	messages := []Message{
		{Role: "assistant", ToolName: "view", FilePath: "config.json", Content: "view content"},
		{Role: "assistant", ToolName: "edit", FilePath: "config.json", Content: "edit content"},
	}

	compacted := Compact(messages)
	if len(compacted) != 2 {
		t.Fatalf("expected both view and edit messages to remain, got %d", len(compacted))
	}
}

func TestCompactPreservesDifferentRolesOnSameFile(t *testing.T) {
	// Messages from different roles (assistant vs user) on same file should be kept
	messages := []Message{
		{Role: "user", ToolName: "view", FilePath: "app.go", Content: "user view"},
		{Role: "assistant", ToolName: "view", FilePath: "app.go", Content: "assistant view"},
	}

	compacted := Compact(messages)
	if len(compacted) != 2 {
		t.Fatalf("expected both role messages to remain, got %d", len(compacted))
	}
}

func BenchmarkCompactHistory(b *testing.B) {
	inputPath := filepath.Join("..", "..", "testdata", "history", "transcript.json")
	input, err := os.ReadFile(inputPath)
	if err != nil {
		b.Fatalf("read input: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := CompactJSON(string(input)); err != nil {
			b.Fatalf("compact transcript: %v", err)
		}
	}
}

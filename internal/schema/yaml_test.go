package schema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMinifyYAMLSchema(t *testing.T) {
	inputPath := filepath.Join("..", "..", "testdata", "schema", "tool.schema.yaml")
	expectedPath := filepath.Join("..", "..", "testdata", "schema", "tool.schema.expected.yaml")

	input, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read input: %v", err)
	}
	expected, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}

	got, err := MinifyYAML(string(input))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != strings.TrimSpace(string(expected)) {
		t.Fatalf("unexpected minified yaml\nwant: %q\ngot:  %q", strings.TrimSpace(string(expected)), got)
	}
}

func TestMinifyYAMLRejectsUnsupportedFeatures(t *testing.T) {
	input := "defaults: &defaults\n  mode: safe\nconfig:\n  <<: *defaults\n"

	if _, err := MinifyYAML(input); err == nil {
		t.Fatalf("expected unsupported yaml feature error")
	}
}

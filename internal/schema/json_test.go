package schema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spw-m-riley/ma/internal/app"
	"github.com/spw-m-riley/ma/internal/testutil"
)

func TestMinifyJSONSchema(t *testing.T) {
	inputPath := filepath.Join("..", "..", "testdata", "schema", "tool.schema.json")
	expectedPath := filepath.Join("..", "..", "testdata", "schema", "tool.schema.expected.json")

	input, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read input: %v", err)
	}
	expected, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}

	got, err := MinifyJSON(string(input))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := strings.TrimSpace(string(expected))
	if got != want {
		t.Fatalf("unexpected minified json\nwant: %q\ngot:  %q", want, got)
	}
}

func TestMinifySchemaReduction(t *testing.T) {
	jsonInput, err := os.ReadFile(filepath.Join("..", "..", "testdata", "schema", "tool.schema.json"))
	if err != nil {
		t.Fatalf("read json input: %v", err)
	}
	jsonOutput, err := MinifyJSON(string(jsonInput))
	if err != nil {
		t.Fatalf("minify json: %v", err)
	}

	yamlInput, err := os.ReadFile(filepath.Join("..", "..", "testdata", "schema", "tool.schema.yaml"))
	if err != nil {
		t.Fatalf("read yaml input: %v", err)
	}
	yamlOutput, err := MinifyYAML(string(yamlInput))
	if err != nil {
		t.Fatalf("minify yaml: %v", err)
	}

	stats := app.Measure(string(jsonInput)+string(yamlInput), jsonOutput+yamlOutput)
	if err := testutil.AssertApproxTokenReductionAtLeast(stats, 40); err != nil {
		t.Fatalf("expected schema minification reduction to meet target: %v", err)
	}
}

func BenchmarkMinifySchema(b *testing.B) {
	jsonInput, err := os.ReadFile(filepath.Join("..", "..", "testdata", "schema", "tool.schema.json"))
	if err != nil {
		b.Fatalf("read json input: %v", err)
	}
	yamlInput, err := os.ReadFile(filepath.Join("..", "..", "testdata", "schema", "tool.schema.yaml"))
	if err != nil {
		b.Fatalf("read yaml input: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := MinifyJSON(string(jsonInput)); err != nil {
			b.Fatalf("minify json: %v", err)
		}
		if _, err := MinifyYAML(string(yamlInput)); err != nil {
			b.Fatalf("minify yaml: %v", err)
		}
	}
}

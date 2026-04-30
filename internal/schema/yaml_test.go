package schema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
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

func TestMinifyYAMLPreservesBlockScalarContent(t *testing.T) {
	input := "prompt: |\n  description: keep this literal line\n  next: still literal\nmode: safe\n"

	got, err := MinifyYAML(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(got, "description: keep this literal line") {
		t.Fatalf("expected block scalar content to be preserved, got %q", got)
	}
	if !strings.Contains(got, "next: still literal") {
		t.Fatalf("expected remaining block scalar content to be preserved, got %q", got)
	}
}

func TestMinifyYAMLPreservesColonInStringValue(t *testing.T) {
	input := "title: \"key: value\"\ndescription: remove me\n"

	got, err := MinifyYAML(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	decoded := decodeYAMLMap(t, got)
	if title, ok := decoded["title"].(string); !ok || title != "key: value" {
		t.Fatalf("expected title to preserve colon-containing string, got %#v", decoded["title"])
	}
	if _, exists := decoded["description"]; exists {
		t.Fatalf("expected description key to be removed, got %#v", decoded)
	}
}

func TestMinifyYAMLRemovesNestedKeys(t *testing.T) {
	input := "properties:\n  title:\n    type: string\n    description: inner\n  config:\n    type: object\n    default:\n      mode: safe\n"

	got, err := MinifyYAML(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	decoded := decodeYAMLMap(t, got)
	properties, ok := decoded["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties map, got %#v", decoded["properties"])
	}
	title, ok := properties["title"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties.title map, got %#v", properties["title"])
	}
	if _, exists := title["description"]; exists {
		t.Fatalf("expected nested description to be removed, got %#v", title)
	}
	config, ok := properties["config"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties.config map, got %#v", properties["config"])
	}
	if _, exists := config["default"]; exists {
		t.Fatalf("expected nested default to be removed, got %#v", config)
	}
}

func TestMinifyYAMLReturnsEmptyDocumentAfterPruning(t *testing.T) {
	input := "description: remove me\ndefault: safe\nexamples:\n  - a\n"

	got, err := MinifyYAML(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != "" {
		t.Fatalf("expected empty output after pruning, got %q", got)
	}
}

func TestMinifyYAMLRejectsUnsupportedFeatures(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "tabs",
			input: "mode:\n\tchild: safe\n",
			want:  "unsupported yaml feature: tabs",
		},
		{
			name:  "anchors",
			input: "defaults: &defaults\n  mode: safe\n",
			want:  "unsupported yaml feature: anchors",
		},
		{
			name:  "merge keys",
			input: "config:\n  <<: {mode: safe}\n",
			want:  "unsupported yaml feature: merge keys",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := MinifyYAML(tt.input); err == nil || err.Error() != tt.want {
				t.Fatalf("expected error %q, got %v", tt.want, err)
			}
		})
	}
}

func TestMinifyYAMLAcceptsLiteralAmpersand(t *testing.T) {
	// Literal & in a scalar value (not a YAML anchor) should be accepted
	input := "company: AT&T\nport: 8080\n"

	got, err := MinifyYAML(input)
	if err != nil {
		t.Fatalf("expected no error for literal & in value, got %v", err)
	}

	if !strings.Contains(got, "AT&T") {
		t.Fatalf("expected literal & to be preserved, got %q", got)
	}
}

func TestMinifyYAMLAcceptsLiteralAsterisk(t *testing.T) {
	// Literal * in a scalar value (not a YAML alias) should be accepted
	input := "note: This is a wildcard * match\nmode: regex\n"

	got, err := MinifyYAML(input)
	if err != nil {
		t.Fatalf("expected no error for literal * in value, got %v", err)
	}

	if !strings.Contains(got, "*") {
		t.Fatalf("expected literal * to be preserved, got %q", got)
	}
}

func TestMinifyYAMLRejectsTabsBeforeParseErrors(t *testing.T) {
	input := "mode:\n\tchild: [\n"

	if _, err := MinifyYAML(input); err == nil || err.Error() != "unsupported yaml feature: tabs" {
		t.Fatalf("expected tab rejection before parse error, got %v", err)
	}
}

func TestValidateNodeRejectsAliases(t *testing.T) {
	node := &yaml.Node{Kind: yaml.AliasNode}

	if err := validateNode(node); err == nil || err.Error() != "unsupported yaml feature: aliases" {
		t.Fatalf("expected alias rejection, got %v", err)
	}
}

func TestMinifyYAMLOutputRoundTrips(t *testing.T) {
	input := "prompt: |\n  description: keep this literal line\n  next: still literal\nmode: safe\ndescription: remove me\n"

	got, err := MinifyYAML(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	decoded := decodeYAMLMap(t, got)
	if prompt, ok := decoded["prompt"].(string); !ok || !strings.Contains(prompt, "description: keep this literal line") {
		t.Fatalf("expected prompt scalar content to survive round trip, got %#v", decoded["prompt"])
	}
	if _, exists := decoded["description"]; exists {
		t.Fatalf("expected removable description key to be absent, got %#v", decoded)
	}
}

func decodeYAMLMap(t *testing.T, input string) map[string]any {
	t.Helper()

	if strings.TrimSpace(input) == "" {
		return map[string]any{}
	}

	var decoded map[string]any
	if err := yaml.Unmarshal([]byte(input), &decoded); err != nil {
		t.Fatalf("unmarshal output: %v", err)
	}
	return decoded
}

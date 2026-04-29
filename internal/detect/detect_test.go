package detect

import "testing"

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

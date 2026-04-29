package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestAppHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	application := newApp(&stdout, &stderr)
	exitCode := application.Run([]string{"--help"})
	if exitCode != 0 {
		t.Fatalf("expected zero exit code, got %d", exitCode)
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
	} {
		if !strings.Contains(help, name) {
			t.Fatalf("expected help output to include command %q, got %q", name, help)
		}
	}
}

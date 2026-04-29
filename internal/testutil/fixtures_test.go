package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spw-m-riley/ma/internal/app"
)

func TestLoadGoldenFixture(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "project-notes")

	if err := os.WriteFile(base+".input.md", []byte("original content"), 0o644); err != nil {
		t.Fatalf("write input fixture: %v", err)
	}
	if err := os.WriteFile(base+".expected.md", []byte("expected content"), 0o644); err != nil {
		t.Fatalf("write expected fixture: %v", err)
	}

	fixture, err := LoadGoldenFixture(base, ".md")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if fixture.Input != "original content" {
		t.Fatalf("unexpected input fixture: %q", fixture.Input)
	}
	if fixture.Expected != "expected content" {
		t.Fatalf("unexpected expected fixture: %q", fixture.Expected)
	}
}

func TestAssertApproxTokenReductionAtLeast(t *testing.T) {
	stats := app.Stats{
		InputApproxTokens:  100,
		OutputApproxTokens: 60,
	}

	if err := AssertApproxTokenReductionAtLeast(stats, 30); err != nil {
		t.Fatalf("expected reduction assertion to pass, got %v", err)
	}
}

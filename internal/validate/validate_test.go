package validate

import (
	"strings"
	"testing"
)

func TestExtractCodeBlocks(t *testing.T) {
	input := "```go\nfmt.Println(\"hi\")\n```\n\n~~~txt\nraw\n~~~\n"

	got := extractCodeBlocks(input)
	if len(got) != 2 {
		t.Fatalf("expected 2 code blocks, got %d", len(got))
	}
}

func TestCompareReportsHeadingMismatch(t *testing.T) {
	original := "# Heading\n\nBody.\n"
	candidate := "# Different Heading\n\nBody.\n"

	report := Compare(original, candidate)
	if report.Valid {
		t.Fatalf("expected report to be invalid")
	}
	if len(report.Errors) == 0 {
		t.Fatalf("expected heading mismatch error")
	}
	if !strings.Contains(report.Errors[0], "heading") {
		t.Fatalf("expected heading error, got %q", report.Errors[0])
	}
}

func TestCompareReportsCodeFenceMismatch(t *testing.T) {
	original := "```go\nfmt.Println(\"hi\")\n```\n"
	candidate := "```go\nfmt.Println(\"bye\")\n```\n"

	report := Compare(original, candidate)
	if report.Valid {
		t.Fatalf("expected report to be invalid")
	}
	if !contains(report.Errors, "code block mismatch") {
		t.Fatalf("expected code block mismatch, got %#v", report.Errors)
	}
}

func TestCompareReportsURLMismatch(t *testing.T) {
	original := "See https://example.com/docs for details.\n"
	candidate := "See https://example.com/other for details.\n"

	report := Compare(original, candidate)
	if report.Valid {
		t.Fatalf("expected report to be invalid")
	}
	if !contains(report.Errors, "url mismatch") {
		t.Fatalf("expected url mismatch, got %#v", report.Errors)
	}
}

func TestCompareWarnsOnPathMismatch(t *testing.T) {
	original := "Keep /etc/hosts unchanged.\n"
	candidate := "Keep /private/etc/hosts unchanged.\n"

	report := Compare(original, candidate)
	if !report.Valid {
		t.Fatalf("expected path mismatch to warn without invalidating report")
	}
	if !contains(report.Warnings, "path mismatch") {
		t.Fatalf("expected path mismatch warning, got %#v", report.Warnings)
	}
}

func TestCompareWarnsOnBulletCountDrift(t *testing.T) {
	original := "- one\n- two\n- three\n- four\n"
	candidate := "- one\n- two\n"

	report := Compare(original, candidate)
	if !report.Valid {
		t.Fatalf("expected bullet drift to warn without invalidating report")
	}
	if !contains(report.Warnings, "bullet count drift") {
		t.Fatalf("expected bullet drift warning, got %#v", report.Warnings)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestReportErrorIncludesDetailedErrors(t *testing.T) {
	report := Report{
		Valid: false,
		Errors: []string{
			"heading mismatch",
			"code block mismatch",
			"url mismatch",
		},
	}
	
	err := report.Error()
	if err == nil {
		t.Fatalf("expected error to be non-nil")
	}
	
	errMsg := err.Error()
	
	// Should include all three detailed errors, not just "validation failed"
	if !strings.Contains(errMsg, "heading mismatch") {
		t.Fatalf("expected 'heading mismatch' in error, got: %q", errMsg)
	}
	if !strings.Contains(errMsg, "code block mismatch") {
		t.Fatalf("expected 'code block mismatch' in error, got: %q", errMsg)
	}
	if !strings.Contains(errMsg, "url mismatch") {
		t.Fatalf("expected 'url mismatch' in error, got: %q", errMsg)
	}
}

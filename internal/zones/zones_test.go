package zones

import "testing"

func TestSplitZones(t *testing.T) {
	input := "# Heading\n\nKeep this prose.\n\n```go\nfmt.Println(\"hi\")\n```\n"

	got := Split(input)
	if len(got) != 3 {
		t.Fatalf("expected 3 zones, got %d", len(got))
	}
}

func TestSplitSeparatesHeadingsProseAndCodeFences(t *testing.T) {
	input := "# Heading\n\nKeep this prose.\n\n```go\nfmt.Println(\"hi\")\n```\n"

	got := Split(input)
	if len(got) != 3 {
		t.Fatalf("expected 3 zones, got %d", len(got))
	}

	if got[0].Kind != Heading {
		t.Fatalf("expected first zone to be heading, got %q", got[0].Kind)
	}
	if got[1].Kind != Prose {
		t.Fatalf("expected second zone to be prose, got %q", got[1].Kind)
	}
	if got[2].Kind != CodeFence {
		t.Fatalf("expected third zone to be code fence, got %q", got[2].Kind)
	}
}

func TestSplitSeparatesInlineCode(t *testing.T) {
	input := "Run `go test ./...` before commit.\n"

	got := Split(input)
	if len(got) != 3 {
		t.Fatalf("expected 3 zones, got %d", len(got))
	}

	if got[0].Kind != Prose {
		t.Fatalf("expected first zone to be prose, got %q", got[0].Kind)
	}
	if got[1].Kind != InlineCode {
		t.Fatalf("expected second zone to be inline code, got %q", got[1].Kind)
	}
	if got[2].Kind != Prose {
		t.Fatalf("expected third zone to be prose, got %q", got[2].Kind)
	}
	if got[1].Text != "`go test ./...`" {
		t.Fatalf("unexpected inline code text: %q", got[1].Text)
	}
}

func TestSplitSeparatesURLsAndPaths(t *testing.T) {
	input := "See https://example.com/docs and keep /etc/hosts unchanged.\n"

	got := Split(input)
	if len(got) != 5 {
		t.Fatalf("expected 5 zones, got %d", len(got))
	}

	if got[1].Kind != URL {
		t.Fatalf("expected second zone to be url, got %q", got[1].Kind)
	}
	if got[3].Kind != Path {
		t.Fatalf("expected fourth zone to be path, got %q", got[3].Kind)
	}
	if got[1].Text != "https://example.com/docs" {
		t.Fatalf("unexpected url text: %q", got[1].Text)
	}
	if got[3].Text != "/etc/hosts" {
		t.Fatalf("unexpected path text: %q", got[3].Text)
	}
}

func TestSplitKeepsMismatchedFenceCloserInsideCodeFence(t *testing.T) {
	input := "```go\nfmt.Println(\"hi\")\n~~~\nstill code\n"

	got := Split(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 zone, got %d", len(got))
	}
	if got[0].Kind != CodeFence {
		t.Fatalf("expected code fence zone, got %q", got[0].Kind)
	}
	if got[0].Text != input {
		t.Fatalf("expected full input to stay inside code fence, got %q", got[0].Text)
	}
}

func TestSplitPreservesFullCodeFenceText(t *testing.T) {
	input := "Before\n\n```go title=demo\nfmt.Println(\"hi\")\n```\n"

	got := Split(input)
	if len(got) != 2 {
		t.Fatalf("expected 2 zones, got %d", len(got))
	}
	if got[1].Kind != CodeFence {
		t.Fatalf("expected second zone to be code fence, got %q", got[1].Kind)
	}
	want := "```go title=demo\nfmt.Println(\"hi\")\n```\n"
	if got[1].Text != want {
		t.Fatalf("expected %q, got %q", want, got[1].Text)
	}
}

func TestSplitHandlesNestedCodeFences(t *testing.T) {
	input := "````markdown\n```go\nfmt.Println(\"hi\")\n```\n````\n"

	got := Split(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 zone, got %d", len(got))
	}
	if got[0].Kind != CodeFence {
		t.Fatalf("expected code fence zone, got %q", got[0].Kind)
	}
	if got[0].Text != input {
		t.Fatalf("expected nested fence block to stay intact, got %q", got[0].Text)
	}
}

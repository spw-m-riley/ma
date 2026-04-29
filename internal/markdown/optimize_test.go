package markdown

import (
	"path/filepath"
	"testing"

	"github.com/spw-m-riley/ma/internal/app"
	"github.com/spw-m-riley/ma/internal/testutil"
)

func TestOptimizeMarkdownFixtures(t *testing.T) {
	base := filepath.Join("..", "..", "testdata", "markdown", "guide")
	fixture, err := testutil.LoadGoldenFixture(base, ".md")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}

	got := Optimize(fixture.Input)
	if got != fixture.Expected {
		t.Fatalf("unexpected optimized markdown\nwant: %q\ngot:  %q", fixture.Expected, got)
	}
}

func TestOptimizeMarkdownPreservesCodeFenceContent(t *testing.T) {
	input := "```md\n* keep this bullet marker\n|  keep  |  spacing  |\n```\n"

	got := Optimize(input)
	if got != input {
		t.Fatalf("expected code fence content to remain unchanged\nwant: %q\ngot:  %q", input, got)
	}
}

func TestOptimizeMarkdownReduction(t *testing.T) {
	base := filepath.Join("..", "..", "testdata", "markdown", "guide")
	fixture, err := testutil.LoadGoldenFixture(base, ".md")
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}

	stats := app.Measure(fixture.Input, Optimize(fixture.Input))
	if err := testutil.AssertApproxTokenReductionAtLeast(stats, 5); err != nil {
		t.Fatalf("expected markdown optimization reduction to meet target: %v", err)
	}
}

func BenchmarkOptimizeMarkdown(b *testing.B) {
	base := filepath.Join("..", "..", "testdata", "markdown", "guide")
	fixture, err := testutil.LoadGoldenFixture(base, ".md")
	if err != nil {
		b.Fatalf("load fixture: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Optimize(fixture.Input)
	}
}

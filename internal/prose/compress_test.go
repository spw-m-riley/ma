package prose

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/spw-m-riley/ma/internal/testutil"
)

func TestCompressRemovesFillerAndShortensPhrases(t *testing.T) {
	input := "Please make sure to utilize concise wording in order to keep context small.\n"

	got := Compress(input)
	want := "ensure use concise wording to keep context small.\n"
	if got != want {
		t.Fatalf("unexpected compressed prose\nwant: %q\ngot:  %q", want, got)
	}
}

func TestCompressPreservesProtectedZones(t *testing.T) {
	input := "# Heading\n\nPlease make sure to keep `go test ./...`, https://example.com/docs, and /etc/hosts unchanged.\n\n```bash\ngo test ./...\n```\n"

	got := Compress(input)
	for _, expected := range []string{
		"# Heading",
		"`go test ./...`",
		"https://example.com/docs",
		"/etc/hosts",
		"```bash\ngo test ./...\n```",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected compressed output to preserve %q, got %q", expected, got)
		}
	}
}

func TestCompressFixtures(t *testing.T) {
	fixtures := []string{
		"preferences",
		"project-notes",
		"mixed-with-code",
	}

	for _, name := range fixtures {
		t.Run(name, func(t *testing.T) {
			base := filepath.Join("..", "..", "testdata", "prose", name)
			fixture, err := testutil.LoadGoldenFixture(base, ".md")
			if err != nil {
				t.Fatalf("load fixture: %v", err)
			}

			got := Compress(fixture.Input)
			if got != fixture.Expected {
				t.Fatalf("unexpected compressed output\nwant: %q\ngot:  %q", fixture.Expected, got)
			}
		})
	}
}

func BenchmarkCompressFixtures(b *testing.B) {
	fixtures := []string{
		"preferences",
		"project-notes",
		"mixed-with-code",
	}

	inputs := make([]string, 0, len(fixtures))
	for _, name := range fixtures {
		base := filepath.Join("..", "..", "testdata", "prose", name)
		fixture, err := testutil.LoadGoldenFixture(base, ".md")
		if err != nil {
			b.Fatalf("load fixture %s: %v", name, err)
		}
		inputs = append(inputs, fixture.Input)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, input := range inputs {
			_ = Compress(input)
		}
	}
}

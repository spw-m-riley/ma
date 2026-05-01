package prose

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spw-m-riley/ma/internal/app"
	"github.com/spw-m-riley/ma/internal/validate"
)

func TestCompressPrunesDemonstratives(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "this is the",
			input: "This is the current plan.\n",
			want:  "current plan.\n",
		},
		{
			name:  "the following",
			input: "The following steps help.\n",
			want:  "These steps help.\n",
		},
		{
			name:  "the same",
			input: "Use the same output.\n",
			want:  "Use same output.\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Compress(tc.input); got != tc.want {
				t.Fatalf("unexpected compressed prose\nwant: %q\ngot:  %q", tc.want, got)
			}
		})
	}
}

func TestCompressAbbreviatesTechnicalTerms(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "repo config env",
			input: "Update the repository configuration for the environment.\n",
			want:  "Update repo config for env.\n",
		},
		{
			name:  "docs auth impl",
			input: "Keep documentation for authentication and implementation.\n",
			want:  "Keep docs for auth and impl.\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Compress(tc.input); got != tc.want {
				t.Fatalf("unexpected compressed prose\nwant: %q\ngot:  %q", tc.want, got)
			}
		})
	}
}

func TestCompressTechnicalAbbreviationsRespectWordBoundaries(t *testing.T) {
	input := "Keep repositoryName and configurationValue unchanged.\n"

	if got := Compress(input); got != input {
		t.Fatalf("expected partial-word technical terms to remain unchanged, got %q", got)
	}
}

func TestCompressRepoInstructionsAchievesReduction(t *testing.T) {
	path := filepath.Join("..", "..", ".github", "copilot-instructions.md")
	inputBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read instructions: %v", err)
	}

	input := string(inputBytes)
	output := Compress(input)
	stats := app.Measure(input, output)
	if stats.OutputApproxTokens >= stats.InputApproxTokens {
		t.Fatalf("expected compressed instructions to reduce tokens, got input=%d output=%d", stats.InputApproxTokens, stats.OutputApproxTokens)
	}
	if report := validate.Compare(input, output); !report.Valid {
		t.Fatalf("expected compressed instructions to remain structurally valid: %v", report.Error())
	}
}

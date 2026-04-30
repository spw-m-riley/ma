package app

import "testing"

func TestCountTokensUsesCl100kBase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{name: "empty string", input: "", want: 0},
		{name: "plain text", input: "Hello, world!", want: 4},
		{name: "unicode", input: "naive cafe 😀", want: 4},
		{name: "markdown", input: "# Heading\n\n- item one\n- item two\n", want: 11},
		{name: "code", input: "func main() { fmt.Println(\"hi\") }", want: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := countTokens(tt.input); got != tt.want {
				t.Fatalf("countTokens(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestCountTokensDeterministic(t *testing.T) {
	input := "Hello, world!"
	want := countTokens(input)

	for i := 0; i < 100; i++ {
		if got := countTokens(input); got != want {
			t.Fatalf("run %d: countTokens(%q) = %d, want %d", i, input, got, want)
		}
	}
}

func TestMeasure(t *testing.T) {
	stats := Measure("four words in here", "two words")

	if stats.InputWords != 4 {
		t.Fatalf("expected 4 input words, got %d", stats.InputWords)
	}
	if stats.OutputWords != 2 {
		t.Fatalf("expected 2 output words, got %d", stats.OutputWords)
	}
	if stats.InputBytes <= stats.OutputBytes {
		t.Fatalf("expected output bytes to be smaller than input bytes: %+v", stats)
	}
	if stats.InputApproxTokens != countTokens("four words in here") {
		t.Fatalf("expected input approx tokens to use tokenizer count, got %d", stats.InputApproxTokens)
	}
	if stats.OutputApproxTokens != countTokens("two words") {
		t.Fatalf("expected output approx tokens to use tokenizer count, got %d", stats.OutputApproxTokens)
	}
}

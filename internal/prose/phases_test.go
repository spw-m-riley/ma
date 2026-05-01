package prose

import "testing"

func TestCompressContractsNegationPhrases(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "do not",
			input: "Do not rewrite headings.\n",
			want:  "Don't rewrite headings.\n",
		},
		{
			name:  "should not",
			input: "You should not delete files.\n",
			want:  "You shouldn't delete files.\n",
		},
		{
			name:  "cannot",
			input: "You cannot skip validation.\n",
			want:  "You can't skip validation.\n",
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

func TestCompressKeepsNegationWithSubjectPresent(t *testing.T) {
	input := "You should not delete files.\n"

	got := Compress(input)
	want := "You shouldn't delete files.\n"
	if got != want {
		t.Fatalf("unexpected compressed prose\nwant: %q\ngot:  %q", want, got)
	}
}

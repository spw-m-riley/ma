package prose

import "testing"

func TestCompressElidesRedundantSubjects(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "you must",
			input: "You must preserve headings.\n",
			want:  "must preserve headings.\n",
		},
		{
			name:  "you will need to",
			input: "You will need to keep backups.\n",
			want:  "keep backups.\n",
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

func TestCompressRemovesHedgingPreambles(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "please note that",
			input: "Please note that this stays concise.\n",
			want:  "this stays concise.\n",
		},
		{
			name:  "important to note that",
			input: "It is important to note that backups matter.\n",
			want:  "backups matter.\n",
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

func TestCompressDoesNotElideContractedNegation(t *testing.T) {
	input := "You shouldn't delete files.\n"

	if got := Compress(input); got != input {
		t.Fatalf("expected contracted negation to remain unchanged, got %q", got)
	}
}

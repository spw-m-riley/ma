package prose

import "testing"

func TestCompressShortensWordyPhrases(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "due to the fact that",
			input: "Due to the fact that logs matter, keep them.\n",
			want:  "Because logs matter, keep them.\n",
		},
		{
			name:  "in the event that",
			input: "In the event that validation fails, stop.\n",
			want:  "If validation fails, stop.\n",
		},
		{
			name:  "whether or not",
			input: "Check whether or not output changed.\n",
			want:  "Check whether output changed.\n",
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

func TestCompressSimplifiesTransitions(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "however",
			input: "However, keep headings unchanged.\n",
			want:  "but keep headings unchanged.\n",
		},
		{
			name:  "furthermore",
			input: "Furthermore, preserve URLs.\n",
			want:  "preserve URLs.\n",
		},
		{
			name:  "therefore",
			input: "Therefore, use backups.\n",
			want:  "so use backups.\n",
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

func TestCompressRemovesFillerAdverbs(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "basically",
			input: "Basically, keep headings unchanged.\n",
			want:  "keep headings unchanged.\n",
		},
		{
			name:  "currently",
			input: "Currently, validate output first.\n",
			want:  "validate output first.\n",
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

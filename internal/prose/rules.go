package prose

import "regexp"

type Rule struct {
	Name        string
	Pattern     *regexp.Regexp
	Replacement string
}

func defaultRules() []Rule {
	return []Rule{
		{
			Name:        "goal-of-tool",
			Pattern:     regexp.MustCompile(`\bThe goal of this tool is to\b`),
			Replacement: "goal:",
		},
		{
			Name:        "while-still-keeping-result",
			Pattern:     regexp.MustCompile(`\bwhile still keeping the result\b`),
			Replacement: "while keeping result",
		},
		{
			Name:        "while-keeping",
			Pattern:     regexp.MustCompile(`\bwhile keeping\b`),
			Replacement: "; keep",
		},
		{
			Name:        "humans-and-language-models",
			Pattern:     regexp.MustCompile(`\bboth humans and language models\b`),
			Replacement: "humans and LLMs",
		},
		{
			Name:        "understandable-to",
			Pattern:     regexp.MustCompile(`\bunderstandable to\b`),
			Replacement: "clear to",
		},
		{
			Name:        "you-should-always",
			Pattern:     regexp.MustCompile(`\b[Yy]ou should always\b\s*`),
			Replacement: "",
		},
		{
			Name:        "you-should",
			Pattern:     regexp.MustCompile(`\b[Yy]ou should\b\s*`),
			Replacement: "",
		},
		{
			Name:        "make-sure-to-keep",
			Pattern:     regexp.MustCompile(`\b[Mm]ake sure to keep\b`),
			Replacement: "keep",
		},
		{
			Name:        "make-sure-to-simplify",
			Pattern:     regexp.MustCompile(`\b[Mm]ake sure to simplify\b`),
			Replacement: "simplify",
		},
		{
			Name:        "make-sure-to",
			Pattern:     regexp.MustCompile(`\b[Mm]ake sure to\b`),
			Replacement: "ensure",
		},
		{
			Name:        "remember-to",
			Pattern:     regexp.MustCompile(`\b[Rr]emember to\b\s*`),
			Replacement: "",
		},
		{
			Name:        "it-would-be-good-to",
			Pattern:     regexp.MustCompile(`\b[Ii]t would be good to\b\s*`),
			Replacement: "",
		},
		{
			Name:        "in-order-to",
			Pattern:     regexp.MustCompile(`\b[Ii]n order to\b`),
			Replacement: "to",
		},
		{
			Name:        "utilize",
			Pattern:     regexp.MustCompile(`\butilize\b`),
			Replacement: "use",
		},
		{
			Name:        "please",
			Pattern:     regexp.MustCompile(`\b[Pp]lease\b\s*`),
			Replacement: "",
		},
		{
			Name:        "additionally",
			Pattern:     regexp.MustCompile(`\b[Aa]dditionally,\s*`),
			Replacement: "",
		},
		{
			Name:        "where-possible",
			Pattern:     regexp.MustCompile(`\bwhere possible\b`),
			Replacement: "",
		},
		{
			Name:        "easy-to-scan",
			Pattern:     regexp.MustCompile(`\beasy to scan\b`),
			Replacement: "scannable",
		},
		{
			Name:        "concise-and-scannable",
			Pattern:     regexp.MustCompile(`\bconcise and scannable\b`),
			Replacement: "concise, scannable",
		},
		{
			Name:        "unnecessary-filler-words",
			Pattern:     regexp.MustCompile(`\bunnecessary filler words\b`),
			Replacement: "filler words",
		},
		{
			Name:        "to-keep-context-small",
			Pattern:     regexp.MustCompile(`\bto keep context small, use\b`),
			Replacement: "keep context small; use",
		},
		{
			Name:        "paths-like",
			Pattern:     regexp.MustCompile(`\bpaths like\b`),
			Replacement: "paths",
		},
		{
			Name:        "commands-like",
			Pattern:     regexp.MustCompile(`\bcommands like\b`),
			Replacement: "commands",
		},
		{
			Name:        "the-prose",
			Pattern:     regexp.MustCompile(`\bthe prose\b`),
			Replacement: "prose",
		},
		{
			Name:        "the-example",
			Pattern:     regexp.MustCompile(`\bthe example\b`),
			Replacement: "example",
		},
		{
			Name:        "example-below",
			Pattern:     regexp.MustCompile(`\bexample below\b`),
			Replacement: "example",
		},
		{
			Name:        "the-heading",
			Pattern:     regexp.MustCompile(`\bthe heading\b`),
			Replacement: "heading",
		},
	}
}

package prose

import "regexp"

type Rule struct {
	Name        string
	Pattern     *regexp.Regexp
	Replacement string
}

type Phase struct {
	Name  string
	Rules []Rule
}

func newRule(name string, pattern string, replacement string) Rule {
	return Rule{
		Name:        name,
		Pattern:     regexp.MustCompile(pattern),
		Replacement: replacement,
	}
}

func rulePhases() []Phase {
	return []Phase{
		{
			Name:  "contractions",
			Rules: contractionRules(),
		},
		{
			Name:  "subject-elision",
			Rules: subjectElisionRules(),
		},
		{
			Name:  "hedging",
			Rules: hedgingRules(),
		},
		{
			Name:  "wordy-phrases",
			Rules: wordyPhraseRules(),
		},
		{
			Name:  "transitions",
			Rules: transitionRules(),
		},
		{
			Name:  "filler-adverbs",
			Rules: fillerAdverbRules(),
		},
		{
			Name:  "demonstratives",
			Rules: demonstrativeRules(),
		},
		{
			Name:  "technical-abbreviations",
			Rules: technicalAbbreviationRules(),
		},
	}
}

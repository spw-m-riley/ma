package prose

func demonstrativeRules() []Rule {
	return []Rule{
		newRule("this-is-the-capitalized", `\bThis is the\b\s*`, "the "),
		newRule("this-is-the", `\bthis is the\b\s*`, "the "),
		newRule("the-following-capitalized", `\bThe following\b`, "These"),
		newRule("the-following", `\bthe following\b`, "these"),
		newRule("the-existing", `\bthe existing\b`, "existing"),
		newRule("the-current", `\bthe current\b`, "current"),
		newRule("the-same", `\bthe same\b`, "same"),
		newRule("the-available", `\bthe available\b`, "available"),
		newRule("the-given", `\bthe given\b`, "given"),
		newRule("the-prose", `\bthe prose\b`, "prose"),
		newRule("the-example", `\bthe example\b`, "example"),
		newRule("example-below", `\bexample below\b`, "example"),
		newRule("the-heading", `\bthe heading\b`, "heading"),
		newRule("the-result", `\bthe result\b`, "result"),
		newRule("the-repository", `\bthe repository\b`, "repository"),
		newRule("the-environment", `\bthe environment\b`, "environment"),
	}
}

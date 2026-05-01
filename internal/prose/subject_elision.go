package prose

func subjectElisionRules() []Rule {
	return []Rule{
		newRule("you-should-always", `\b[Yy]ou should always\s+`, ""),
		newRule("you-should", `\b[Yy]ou should\s+`, ""),
		newRule("you-must", `\b[Yy]ou must\s+`, "must "),
		newRule("you-need-to", `\b[Yy]ou need to\s+`, ""),
		newRule("you-will-need-to", `\b[Yy]ou will need to\s+`, ""),
		newRule("you-can", `\b[Yy]ou can\s+`, ""),
		newRule("you-may", `\b[Yy]ou may\s+`, ""),
		newRule("you-are-responsible-for", `\b[Yy]ou are responsible for\s+`, ""),
	}
}

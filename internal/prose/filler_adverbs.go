package prose

func fillerAdverbRules() []Rule {
	return []Rule{
		newRule("basically", `\b[Bb]asically,\s*`, ""),
		newRule("essentially", `\b[Ee]ssentially,\s*`, ""),
		newRule("actually", `\b[Aa]ctually,\s*`, ""),
		newRule("currently", `\b[Cc]urrently,\s*`, ""),
		newRule("specifically", `\b[Ss]pecifically,\s*`, ""),
		newRule("simply", `\b[Ss]imply,\s*`, ""),
	}
}

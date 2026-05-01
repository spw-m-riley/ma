package prose

func transitionRules() []Rule {
	return []Rule{
		newRule("however", `\b[Hh]owever,\s*`, "but "),
		newRule("therefore", `\b[Tt]herefore,\s*`, "so "),
		newRule("additionally", `\b[Aa]dditionally,\s*`, ""),
		newRule("furthermore", `\b[Ff]urthermore,\s*`, ""),
		newRule("moreover", `\b[Mm]oreover,\s*`, ""),
		newRule("in-addition", `\b[Ii]n addition,\s*`, ""),
		newRule("as-a-result", `\b[Aa]s a result,\s*`, "so "),
		newRule("consequently", `\b[Cc]onsequently,\s*`, "so "),
	}
}

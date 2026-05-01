package prose

func hedgingRules() []Rule {
	return []Rule{
		newRule("please-note-that", `\b[Pp]lease note that\b\s*`, ""),
		newRule("please-remember-to", `\b[Pp]lease remember to\b\s*`, ""),
		newRule("please", `\b[Pp]lease\b\s*`, ""),
		newRule("remember-to", `\b[Rr]emember to\b\s*`, ""),
		newRule("important-to-note-that", `\b[Ii]t is important to note that\b\s*`, ""),
		newRule("it-would-be-good-to", `\b[Ii]t would be good to\b\s*`, ""),
		newRule("recommended-to", `\b[Ii]t is recommended to\b\s*`, ""),
		newRule("be-sure-to", `\b[Bb]e sure to\b\s*`, ""),
		newRule("make-sure-to-keep", `\b[Mm]ake sure to keep\b\s*`, "keep "),
		newRule("make-sure-to-simplify", `\b[Mm]ake sure to simplify\b\s*`, "simplify "),
		newRule("make-sure-to", `\b[Mm]ake sure to\b\s*`, "ensure "),
		newRule("kindly", `\b[Kk]indly\b\s*`, ""),
		newRule("as-a-reminder", `\b[Aa]s a reminder,\s*`, ""),
		newRule("where-possible", `\bwhere possible\b`, ""),
	}
}

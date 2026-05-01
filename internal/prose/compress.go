package prose

import (
	"regexp"
	"strings"

	"github.com/spw-m-riley/ma/internal/zones"
)

var spacePattern = regexp.MustCompile(`[ \t]{2,}`)
var spaceBeforePunctuationPattern = regexp.MustCompile(`\s+([.,;:!?])`)

func Compress(input string) string {
	zs := zones.Split(input)
	var out strings.Builder
	for _, zone := range zs {
		text := zone.Text
		if zone.Kind == zones.Prose {
			text = applyPhases(text)
			text = normalizeWhitespace(text)
		}
		out.WriteString(text)
	}
	return out.String()
}

func applyPhases(input string) string {
	for _, phase := range rulePhases() {
		for _, rule := range phase.Rules {
			input = rule.Pattern.ReplaceAllString(input, rule.Replacement)
		}
	}

	return input
}

func normalizeWhitespace(input string) string {
	lines := strings.SplitAfter(input, "\n")
	for i, line := range lines {
		suffix := ""
		if strings.HasSuffix(line, "\n") {
			suffix = "\n"
			line = strings.TrimSuffix(line, "\n")
		}
		line = spacePattern.ReplaceAllString(line, " ")
		line = spaceBeforePunctuationPattern.ReplaceAllString(line, "$1")
		lines[i] = line + suffix
	}
	output := strings.Join(lines, "")
	output = strings.ReplaceAll(output, "ensure to", "ensure")
	return output
}

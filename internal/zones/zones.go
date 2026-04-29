package zones

import (
	"regexp"
	"strings"
)

type Kind string

const (
	Prose      Kind = "prose"
	CodeFence  Kind = "code_fence"
	InlineCode Kind = "inline_code"
	Heading    Kind = "heading"
	URL        Kind = "url"
	Path       Kind = "path"
)

type Zone struct {
	Kind Kind
	Text string
}

var urlPattern = regexp.MustCompile(`https?://[^\s)]+`)

var pathPattern = regexp.MustCompile(`(^|[\s(])((?:/|\./|\../)[^\s)` + "`" + `]+)`)

func Split(input string) []Zone {
	lines := strings.SplitAfter(input, "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil
	}

	var zones []Zone
	var prose strings.Builder
	var fence strings.Builder
	inFence := false

	flushProse := func() {
		if prose.Len() == 0 {
			return
		}
		zones = append(zones, splitProtectedText(prose.String())...)
		prose.Reset()
	}

	flushFence := func() {
		if fence.Len() == 0 {
			return
		}
		zones = append(zones, Zone{Kind: CodeFence, Text: fence.String()})
		fence.Reset()
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if inFence {
			fence.WriteString(line)
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				inFence = false
				flushFence()
			}
			continue
		}

		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			flushProse()
			inFence = true
			fence.WriteString(line)
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			flushProse()
			zones = append(zones, Zone{Kind: Heading, Text: line})
			continue
		}

		prose.WriteString(line)
	}

	flushProse()
	flushFence()

	return zones
}

func splitProtectedText(input string) []Zone {
	if input == "" {
		return nil
	}

	var zones []Zone
	start := 0
	for start < len(input) {
		nextStart, nextEnd, nextKind, ok := findNextProtectedToken(input, start)
		if !ok {
			zones = appendTextZone(zones, Prose, input[start:])
			break
		}

		zones = appendTextZone(zones, Prose, input[start:nextStart])
		zones = appendTextZone(zones, nextKind, input[nextStart:nextEnd])
		start = nextEnd
	}

	return zones
}

func findNextProtectedToken(input string, start int) (int, int, Kind, bool) {
	candidates := make([]Zone, 0, 3)

	if open := strings.IndexByte(input[start:], '`'); open >= 0 {
		open += start
		if close := strings.IndexByte(input[open+1:], '`'); close >= 0 {
			close += open + 1
			candidates = append(candidates, Zone{Kind: InlineCode, Text: input[open : close+1]})
		}
	}

	if loc := urlPattern.FindStringIndex(input[start:]); loc != nil {
		candidates = append(candidates, Zone{Kind: URL, Text: input[start+loc[0] : start+loc[1]]})
	}

	if loc := pathPattern.FindStringSubmatchIndex(input[start:]); loc != nil {
		candidates = append(candidates, Zone{Kind: Path, Text: input[start+loc[4] : start+loc[5]]})
	}

	if len(candidates) == 0 {
		return 0, 0, "", false
	}

	bestStart := len(input)
	bestEnd := 0
	var bestKind Kind
	for _, candidate := range candidates {
		candidateStart := strings.Index(input[start:], candidate.Text)
		if candidateStart < 0 {
			continue
		}
		candidateStart += start
		if candidateStart < bestStart {
			bestStart = candidateStart
			bestEnd = candidateStart + len(candidate.Text)
			bestKind = candidate.Kind
		}
	}

	if bestStart == len(input) {
		return 0, 0, "", false
	}

	return bestStart, bestEnd, bestKind, true
}

func appendTextZone(zones []Zone, kind Kind, text string) []Zone {
	if text == "" {
		return zones
	}
	return append(zones, Zone{Kind: kind, Text: text})
}

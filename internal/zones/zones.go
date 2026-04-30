package zones

import (
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
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
	if input == "" {
		return nil
	}

	source := []byte(input)
	special := collectSpecialZones(source)
	var zones []Zone
	cursor := 0
	for _, region := range special {
		if region.start > cursor {
			zones = append(zones, splitProtectedText(input[cursor:region.start])...)
		}
		zones = append(zones, Zone{Kind: region.kind, Text: input[region.start:region.end]})
		cursor = region.end
	}
	if cursor < len(input) {
		zones = append(zones, splitProtectedText(input[cursor:])...)
	}

	return zones
}

type sourceRegion struct {
	start int
	end   int
	kind  Kind
}

func collectSpecialZones(source []byte) []sourceRegion {
	doc := goldmark.New().Parser().Parse(text.NewReader(source))
	regions := make([]sourceRegion, 0)

	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node.Kind() {
		case ast.KindHeading:
			start, end := nodeSourceRange(node, source)
			if start >= end {
				return ast.WalkContinue, nil
			}
			line := strings.TrimRight(string(source[start:end]), "\r\n")
			if !strings.HasPrefix(strings.TrimSpace(line), "#") {
				return ast.WalkContinue, nil
			}
			regions = append(regions, sourceRegion{start: start, end: end, kind: Heading})
		case ast.KindFencedCodeBlock:
			start, end := fencedBlockSourceRange(node, source)
			if start >= end {
				return ast.WalkContinue, nil
			}
			regions = append(regions, sourceRegion{start: start, end: end, kind: CodeFence})
		}

		return ast.WalkContinue, nil
	})

	sort.Slice(regions, func(i, j int) bool {
		if regions[i].start == regions[j].start {
			return regions[i].end < regions[j].end
		}
		return regions[i].start < regions[j].start
	})

	return regions
}

func nodeSourceRange(node ast.Node, source []byte) (int, int) {
	start, end := nodeByteRange(node, source)
	if start >= end {
		return 0, 0
	}
	for start > 0 && source[start-1] != '\n' {
		start--
	}
	for end < len(source) && source[end-1] != '\n' {
		end++
	}
	return start, end
}

func nodeByteRange(node ast.Node, source []byte) (int, int) {
	start := len(source)
	end := 0

	_ = ast.Walk(node, func(child ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if child.Type() != ast.TypeInline {
			lines := child.Lines()
			for i := 0; i < lines.Len(); i++ {
				segment := lines.At(i)
				if segment.Start < start {
					start = segment.Start
				}
				if segment.Stop > end {
					end = segment.Stop
				}
			}
		}

		if textNode, ok := child.(*ast.Text); ok {
			if textNode.Segment.Start < start {
				start = textNode.Segment.Start
			}
			if textNode.Segment.Stop > end {
				end = textNode.Segment.Stop
			}
		}

		return ast.WalkContinue, nil
	})

	if pos := node.Pos(); pos >= 0 && pos < start {
		start = pos
	}
	if start >= end {
		return 0, 0
	}
	return start, end
}

func fencedBlockSourceRange(node ast.Node, source []byte) (int, int) {
	start := node.Pos()
	if start < 0 || start >= len(source) {
		return 0, 0
	}
	for start > 0 && source[start-1] != '\n' {
		start--
	}

	end := findFenceBlockEnd(source, start)
	if start >= end {
		return 0, 0
	}
	return start, end
}

func findFenceBlockEnd(source []byte, start int) int {
	lineEnd := start
	for lineEnd < len(source) && source[lineEnd] != '\n' {
		lineEnd++
	}

	opener := string(source[start:lineEnd])
	fenceChar, fenceLen, ok := detectFence(opener)
	if !ok {
		if lineEnd < len(source) {
			return lineEnd + 1
		}
		return lineEnd
	}

	cursor := lineEnd
	if cursor < len(source) {
		cursor++
	}

	for cursor < len(source) {
		nextLineEnd := cursor
		for nextLineEnd < len(source) && source[nextLineEnd] != '\n' {
			nextLineEnd++
		}

		line := string(source[cursor:nextLineEnd])
		if closesFence(line, fenceChar, fenceLen) {
			if nextLineEnd < len(source) {
				return nextLineEnd + 1
			}
			return nextLineEnd
		}

		if nextLineEnd == len(source) {
			return nextLineEnd
		}
		cursor = nextLineEnd + 1
	}

	return len(source)
}

func detectFence(line string) (byte, int, bool) {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return 0, 0, false
	}
	if strings.HasPrefix(trimmed, "```") {
		return '`', countFenceChars(trimmed, '`'), true
	}
	if strings.HasPrefix(trimmed, "~~~") {
		return '~', countFenceChars(trimmed, '~'), true
	}
	return 0, 0, false
}

func closesFence(line string, fenceChar byte, fenceLen int) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < fenceLen {
		return false
	}
	for i := 0; i < fenceLen; i++ {
		if trimmed[i] != fenceChar {
			return false
		}
	}
	return true
}

func countFenceChars(line string, fenceChar byte) int {
	count := 0
	for i := 0; i < len(line); i++ {
		if line[i] != fenceChar {
			break
		}
		count++
	}
	return count
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

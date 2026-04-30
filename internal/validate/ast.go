package validate

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

func extractHeadingsAST(source []byte) []string {
	doc := parseMarkdown(source)

	var headings []string
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || node.Kind() != ast.KindHeading {
			return ast.WalkContinue, nil
		}

		line := strings.TrimRight(nodeSourceLine(node, source), "\r\n")
		if !strings.HasPrefix(line, "#") {
			return ast.WalkContinue, nil
		}

		idx := strings.IndexByte(line, ' ')
		if idx < 0 {
			return ast.WalkContinue, nil
		}
		headings = append(headings, line[idx+1:])
		return ast.WalkContinue, nil
	})

	return headings
}

func extractCodeBlocksAST(source []byte) []string {
	doc := parseMarkdown(source)

	var blocks []string
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || node.Kind() != ast.KindFencedCodeBlock {
			return ast.WalkContinue, nil
		}
		blocks = append(blocks, fencedBlockSourceText(node, source))
		return ast.WalkContinue, nil
	})

	return blocks
}

func parseMarkdown(source []byte) ast.Node {
	return goldmark.New().Parser().Parse(text.NewReader(source))
}

func nodeSourceLine(node ast.Node, source []byte) string {
	start, end := nodeByteRange(node, source)
	if start >= end {
		return ""
	}
	for start > 0 && source[start-1] != '\n' {
		start--
	}
	for end < len(source) && source[end-1] != '\n' {
		end++
	}
	return string(source[start:end])
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

func fencedBlockSourceText(node ast.Node, source []byte) string {
	start := node.Pos()
	if start < 0 || start >= len(source) {
		return ""
	}
	for start > 0 && source[start-1] != '\n' {
		start--
	}

	end := findFenceBlockEnd(source, start)
	if end < start {
		return ""
	}
	return string(source[start:end])
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

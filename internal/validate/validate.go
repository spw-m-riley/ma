package validate

import (
	"fmt"
	"regexp"
	"strings"
)

type Report struct {
	Valid    bool
	Errors   []string
	Warnings []string
}

var headingPattern = regexp.MustCompile(`(?m)^#{1,6}\s+(.+)$`)
var urlPattern = regexp.MustCompile(`https?://[^\s)]+`)
var pathPattern = regexp.MustCompile(`(?:^|[\s(])((?:/|\./|\../)[^\s)` + "`" + `]+)`)
var bulletPattern = regexp.MustCompile(`(?m)^\s*[-*+]\s+`)

func Compare(original string, candidate string) Report {
	report := Report{Valid: true}

	if !equalStrings(extractMatches(headingPattern, original), extractMatches(headingPattern, candidate)) {
		report.Valid = false
		report.Errors = append(report.Errors, "heading mismatch")
	}

	if !equalStrings(extractCodeBlocks(original), extractCodeBlocks(candidate)) {
		report.Valid = false
		report.Errors = append(report.Errors, "code block mismatch")
	}

	if !equalStrings(urlPattern.FindAllString(original, -1), urlPattern.FindAllString(candidate, -1)) {
		report.Valid = false
		report.Errors = append(report.Errors, "url mismatch")
	}

	if !equalStrings(extractPaths(original), extractPaths(candidate)) {
		report.Warnings = append(report.Warnings, "path mismatch")
	}

	if bulletCountDrift(original, candidate) {
		report.Warnings = append(report.Warnings, "bullet count drift")
	}

	return report
}

func extractMatches(pattern *regexp.Regexp, input string) []string {
	matches := pattern.FindAllStringSubmatch(input, -1)
	values := make([]string, 0, len(matches))
	for _, match := range matches {
		values = append(values, match[1])
	}
	return values
}

func equalStrings(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (r Report) Error() error {
	if r.Valid {
		return nil
	}
	if len(r.Errors) == 0 {
		return fmt.Errorf("validation failed")
	}
	return fmt.Errorf("validation failed: %s", strings.Join(r.Errors, ", "))
}

func extractCodeBlocks(input string) []string {
	lines := strings.SplitAfter(input, "\n")
	var blocks []string
	var current strings.Builder
	var fenceChar byte
	var fenceLen int
	inFence := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inFence {
			fenceChar, fenceLen, inFence = detectFence(trimmed)
			if !inFence {
				continue
			}
			current.WriteString(line)
			continue
		}

		current.WriteString(line)
		if closesFence(trimmed, fenceChar, fenceLen) {
			blocks = append(blocks, current.String())
			current.Reset()
			inFence = false
		}
	}

	return blocks
}

func detectFence(line string) (byte, int, bool) {
	if len(line) < 3 {
		return 0, 0, false
	}
	if strings.HasPrefix(line, "```") {
		return '`', countFenceChars(line, '`'), true
	}
	if strings.HasPrefix(line, "~~~") {
		return '~', countFenceChars(line, '~'), true
	}
	return 0, 0, false
}

func closesFence(line string, fenceChar byte, fenceLen int) bool {
	if len(line) < fenceLen {
		return false
	}
	for i := 0; i < fenceLen; i++ {
		if line[i] != fenceChar {
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

func extractPaths(input string) []string {
	matches := pathPattern.FindAllStringSubmatch(input, -1)
	paths := make([]string, 0, len(matches))
	for _, match := range matches {
		paths = append(paths, match[1])
	}
	return paths
}

func bulletCountDrift(original string, candidate string) bool {
	originalCount := len(bulletPattern.FindAllString(original, -1))
	candidateCount := len(bulletPattern.FindAllString(candidate, -1))
	if originalCount == 0 {
		return false
	}

	diff := originalCount - candidateCount
	if diff < 0 {
		diff = -diff
	}

	return float64(diff)/float64(originalCount) > 0.15
}

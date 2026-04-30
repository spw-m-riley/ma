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

var urlPattern = regexp.MustCompile(`https?://[^\s)]+`)
var pathPattern = regexp.MustCompile(`(?:^|[\s(])((?:/|\./|\../)[^\s)` + "`" + `]+)`)
var bulletPattern = regexp.MustCompile(`(?m)^\s*[-*+]\s+`)

func Compare(original string, candidate string) Report {
	report := Report{Valid: true}

	if !equalStrings(extractHeadingsAST([]byte(original)), extractHeadingsAST([]byte(candidate))) {
		report.Valid = false
		report.Errors = append(report.Errors, "heading mismatch")
	}

	if !equalStrings(extractCodeBlocksAST([]byte(original)), extractCodeBlocksAST([]byte(candidate))) {
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

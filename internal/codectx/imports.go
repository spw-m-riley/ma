package codectx

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var namedImportPattern = regexp.MustCompile(`^import\s+\{([^}]+)\}\s+from\s+["']([^"']+)["'];?\s*$`)
var typeImportPattern = regexp.MustCompile(`^import\s+type\s+\{([^}]+)\}\s+from\s+["']([^"']+)["'];?\s*$`)

func TrimImportsFile(path string, src []byte) (string, []string, error) {
	switch filepath.Ext(path) {
	case ".ts", ".tsx", ".js", ".jsx":
		ext := filepath.Ext(path)
		output, warnings, err := tsjsTrimImports(ext, src)
		if err != nil {
			heuristicOutput := trimJSImportBlock(string(src))
			return heuristicOutput, warnings, nil
		}
		return output, warnings, nil
	default:
		return "", nil, fmt.Errorf("unsupported import trimming extension %q", filepath.Ext(path))
	}
}

func trimJSImportBlock(input string) string {
	lines := strings.Split(input, "\n")
	var importSummaries []string
	var typeSummaries []string
	var body []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			body = append(body, line)
			continue
		}

		if match := namedImportPattern.FindStringSubmatch(trimmed); match != nil {
			names := normalizeList(match[1])
			importSummaries = append(importSummaries, formatImportSummary(match[2], names))
			continue
		}
		if match := typeImportPattern.FindStringSubmatch(trimmed); match != nil {
			names := normalizeList(match[1])
			typeSummaries = append(typeSummaries, fmt.Sprintf("%s from %s", strings.Join(names, ", "), match[2]))
			continue
		}

		body = append(body, line)
	}

	var out bytes.Buffer
	if len(importSummaries) > 0 {
		out.WriteString("// imports: ")
		out.WriteString(strings.Join(importSummaries, "; "))
		out.WriteByte('\n')
	}
	if len(typeSummaries) > 0 {
		out.WriteString("// types: ")
		out.WriteString(strings.Join(typeSummaries, "; "))
		out.WriteByte('\n')
	}
	if len(importSummaries) > 0 || len(typeSummaries) > 0 {
		out.WriteByte('\n')
	}
	for i, line := range body {
		out.WriteString(line)
		if i < len(body)-1 {
			out.WriteByte('\n')
		}
	}
	return out.String()
}

func normalizeList(list string) []string {
	parts := strings.Split(list, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func formatImportSummary(module string, names []string) string {
	if len(names) <= 3 {
		return fmt.Sprintf("%s(%s)", module, strings.Join(names, ", "))
	}
	return fmt.Sprintf("%s{%d}", module, len(names))
}

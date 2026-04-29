package schema

import (
	"fmt"
	"strings"
)

var removableYAMLKeys = map[string]struct{}{
	"description": {},
	"default":     {},
	"examples":    {},
}

func MinifyYAML(input string) (string, error) {
	lines := strings.Split(input, "\n")
	var out []string
	skipIndent := -1

	for _, line := range lines {
		if err := validateSupportedYAMLLine(line); err != nil {
			return "", err
		}

		trimmed := strings.TrimSpace(line)
		indent := leadingSpaces(line)

		if skipIndent >= 0 {
			if trimmed == "" || indent > skipIndent {
				continue
			}
			skipIndent = -1
		}

		if trimmed == "" {
			out = append(out, line)
			continue
		}

		if key, ok := yamlKey(trimmed); ok {
			if _, remove := removableYAMLKeys[key]; remove {
				skipIndent = indent
				continue
			}
		}

		out = append(out, line)
	}

	return strings.TrimSpace(strings.Join(out, "\n")), nil
}

func validateSupportedYAMLLine(line string) error {
	trimmed := strings.TrimSpace(line)
	if strings.Contains(line, "\t") {
		return fmt.Errorf("unsupported yaml feature: tabs")
	}
	
	// Reject merge key syntax (<<:)
	if strings.Contains(trimmed, "<<:") {
		return fmt.Errorf("unsupported yaml feature: merge keys")
	}
	
	// Reject YAML anchors (& followed by identifier at start of value)
	if strings.Contains(trimmed, ": &") || strings.HasPrefix(trimmed, "&") {
		return fmt.Errorf("unsupported yaml feature: anchors")
	}
	
	// Reject YAML aliases (* followed by identifier)
	if strings.HasPrefix(trimmed, "*") || strings.Contains(trimmed, ": *") {
		return fmt.Errorf("unsupported yaml feature: aliases")
	}
	
	return nil
}

func yamlKey(line string) (string, bool) {
	if strings.HasPrefix(line, "- ") {
		return "", false
	}
	index := strings.Index(line, ":")
	if index < 0 {
		return "", false
	}
	return strings.TrimSpace(line[:index]), true
}

func leadingSpaces(line string) int {
	count := 0
	for count < len(line) && line[count] == ' ' {
		count++
	}
	return count
}

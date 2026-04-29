package markdown

import "strings"

func Optimize(input string) string {
	lines := strings.Split(input, "\n")
	var out []string
	blankCount := 0
	inFence := false

	for _, line := range lines {
		trimmedRight := strings.TrimRight(line, " \t")
		trimmed := strings.TrimSpace(trimmedRight)

		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			blankCount = 0
			out = append(out, trimmedRight)
			continue
		}

		if inFence {
			out = append(out, line)
			continue
		}

		if trimmed == "" {
			blankCount++
			if blankCount > 1 {
				continue
			}
			out = append(out, "")
			continue
		}
		blankCount = 0

		if strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "+ ") {
			trimmedRight = "- " + strings.TrimSpace(trimmed[2:])
		}

		if strings.HasPrefix(trimmed, "|") && strings.HasSuffix(trimmed, "|") {
			trimmedRight = compactTableRow(trimmed)
		}

		out = append(out, trimmedRight)
	}

	return strings.Join(out, "\n")
}

func compactTableRow(line string) string {
	parts := strings.Split(line, "|")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return strings.Join(parts, "|")
}

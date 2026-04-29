package history

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Message struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	ToolName  string `json:"toolName,omitempty"`
	FilePath  string `json:"filePath,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

func CompactJSON(input string) (string, error) {
	var messages []Message
	if err := json.Unmarshal([]byte(input), &messages); err != nil {
		return "", err
	}

	compacted := Compact(messages)
	output, err := json.Marshal(compacted)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func Compact(messages []Message) []Message {
	messages = collapseDuplicateReads(messages)
	messages = summarizeFailureChains(messages)
	messages = trimToolPayloads(messages, 20, 20)
	return messages
}

func collapseDuplicateReads(messages []Message) []Message {
	latestByFile := make(map[string]int)
	keep := make([]bool, len(messages))
	for i := range keep {
		keep[i] = true
	}

	for index, message := range messages {
		if message.FilePath == "" {
			continue
		}
		if previous, ok := latestByFile[message.FilePath]; ok {
			keep[previous] = false
		}
		latestByFile[message.FilePath] = index
	}

	out := make([]Message, 0, len(messages))
	for index, message := range messages {
		if keep[index] {
			out = append(out, message)
		}
	}
	return out
}

func summarizeFailureChains(messages []Message) []Message {
	out := make([]Message, 0, len(messages))
	for _, message := range messages {
		message.Content = normalizeEscapedNewlines(message.Content)
		message.Content = collapseRepeatedLines(message.Content)
		out = append(out, message)
	}
	return out
}

func trimToolPayloads(messages []Message, head int, tail int) []Message {
	out := make([]Message, 0, len(messages))
	for _, message := range messages {
		if message.ToolName != "" {
			message.Content = trimLines(message.Content, head, tail)
		}
		out = append(out, message)
	}
	return out
}

func collapseRepeatedLines(content string) string {
	lines := strings.Split(content, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			filtered = append(filtered, line)
		}
	}
	if len(filtered) == 0 {
		return content
	}

	allSame := true
	for _, line := range filtered[1:] {
		if line != filtered[0] {
			allSame = false
			break
		}
	}
	if allSame && len(filtered) > 1 {
		return fmt.Sprintf("%s x%d", filtered[0], len(filtered))
	}
	return strings.Join(filtered, "\n")
}

func trimLines(content string, head int, tail int) string {
	lines := strings.Split(content, "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if line != "" {
			filtered = append(filtered, line)
		}
	}
	if len(filtered) <= head+tail+1 {
		return strings.Join(filtered, "\n")
	}

	out := make([]string, 0, head+tail+1)
	out = append(out, filtered[:head]...)
	out = append(out, fmt.Sprintf("... %d lines omitted ...", len(filtered)-head-tail))
	out = append(out, filtered[len(filtered)-tail:]...)
	return strings.Join(out, "\n")
}

func normalizeEscapedNewlines(content string) string {
	return strings.ReplaceAll(content, `\n`, "\n")
}

package detect

import (
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"
)

type Classification string

const (
	NaturalLanguage Classification = "natural_language"
	Code            Classification = "code"
	Config          Classification = "config"
	Skip            Classification = "skip"
)

func Classify(path string, content string) Classification {
	switch filepath.Ext(path) {
	case ".md", ".txt":
		return NaturalLanguage
	case ".go", ".ts", ".js", ".tsx", ".jsx", ".py", ".java", ".rb", ".rs":
		return Code
	case ".json", ".yaml", ".yml", ".toml":
		return Config
	default:
		return classifyContent(path, content)
	}
}

var codePattern = regexp.MustCompile(`(?m)^\s*(package\s+\w+|func\s+\w+|import\s+.+|export\s+.+|class\s+\w+|def\s+\w+)`)

var sensitiveBasenames = map[string]struct{}{
	".env":            {},
	".env.local":      {},
	"id_rsa":          {},
	"id_ed25519":      {},
	"credentials":     {},
	"known_hosts":     {},
	"authorized_keys": {},
}

var sensitiveComponents = map[string]struct{}{
	".ssh":   {},
	".aws":   {},
	".gnupg": {},
	".kube":  {},
}

func classifyContent(path string, content string) Classification {
	ext := filepath.Ext(path)
	if ext != "" {
		return Skip
	}

	if json.Valid([]byte(content)) {
		return Config
	}

	if isLikelyYAML(content) {
		return Config
	}

	if codePattern.MatchString(content) {
		return Code
	}

	if strings.TrimSpace(content) == "" {
		return Skip
	}

	return NaturalLanguage
}

func isLikelyYAML(content string) bool {
	lines := strings.Split(content, "\n")
	matches := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.Contains(trimmed, ":") {
			matches++
		}
	}
	return matches > 0
}

func IsSensitivePath(path string) bool {
	cleaned := filepath.Clean(path)
	base := filepath.Base(cleaned)
	if _, ok := sensitiveBasenames[base]; ok {
		return true
	}

	for _, component := range strings.Split(cleaned, string(filepath.Separator)) {
		if _, ok := sensitiveComponents[component]; ok {
			return true
		}
	}

	return false
}

package detect

import (
	"encoding/json"
	"os"
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

// IsSensitivePathResolved checks if a path is sensitive, including resolution of symlinks.
// If the path is a symlink, it resolves the target and checks if the target is sensitive.
// If symlink resolution fails (broken symlink), it fails closed and returns true.
// If the path is not a symlink and the file doesn't exist, it returns false (normal missing-file behavior).
func IsSensitivePathResolved(path string) bool {
	// First check the lexical path
	if IsSensitivePath(path) {
		return true
	}

	// Try to resolve symlinks
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		// If we get an error and the path looks like a symlink, fail closed
		lstat, err2 := os.Lstat(path)
		if err2 == nil && (lstat.Mode()&os.ModeSymlink) != 0 {
			// It's a symlink that we couldn't resolve (broken link) - fail closed
			return true
		}
		// Not a symlink or a regular file error - return false (normal missing file)
		return false
	}

	// Check if the resolved path is sensitive
	return IsSensitivePath(resolved)
}

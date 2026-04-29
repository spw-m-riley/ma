package codectx

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strings"
)

var functionSignaturePattern = regexp.MustCompile(`^\s*(export\s+)?(async\s+)?function\s+\w+.*\{\s*$`)

func SkeletonFile(path string, src []byte) (string, []string, error) {
	switch filepath.Ext(path) {
	case ".go":
		output, err := SkeletonGo(src)
		return output, nil, err
	case ".ts", ".tsx", ".js", ".jsx":
		output := SkeletonHeuristic(string(src))
		return output, []string{"heuristic skeleton used for non-Go source"}, nil
	default:
		return "", nil, fmt.Errorf("unsupported skeleton extension %q", filepath.Ext(path))
	}
}

func SkeletonGo(src []byte) (string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return "", err
	}

	ast.Inspect(file, func(node ast.Node) bool {
		if fn, ok := node.(*ast.FuncDecl); ok {
			fn.Body = nil
		}
		return true
	})

	var out bytes.Buffer
	if err := format.Node(&out, fset, file); err != nil {
		return "", err
	}
	return out.String(), nil
}

func SkeletonHeuristic(input string) string {
	lines := strings.Split(input, "\n")
	var out []string
	skippingBody := false
	braceDepth := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if skippingBody {
			braceDepth += strings.Count(line, "{")
			braceDepth -= strings.Count(line, "}")
			if braceDepth <= 0 {
				skippingBody = false
				braceDepth = 0
				out = append(out, "")
			}
			continue
		}

		if functionSignaturePattern.MatchString(trimmed) {
			signature := strings.TrimSpace(strings.TrimSuffix(trimmed, "{")) + ";"
			out = append(out, signature)
			skippingBody = true
			braceDepth = strings.Count(line, "{") - strings.Count(line, "}")
			continue
		}

		out = append(out, line)
	}

	return strings.TrimRight(strings.Join(out, "\n"), "\n") + "\n"
}

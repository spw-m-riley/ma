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
	"strconv"
	"strings"
)

var namedImportPattern = regexp.MustCompile(`^import\s+\{([^}]+)\}\s+from\s+["']([^"']+)["'];?\s*$`)
var typeImportPattern = regexp.MustCompile(`^import\s+type\s+\{([^}]+)\}\s+from\s+["']([^"']+)["'];?\s*$`)

func TrimImportsFile(path string, src []byte) (string, []string, error) {
	switch filepath.Ext(path) {
	case ".go":
		output, err := trimGoImports(src)
		return output, nil, err
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

func trimGoImports(src []byte) (string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		return "", err
	}

	importSummaries := make([]string, 0, len(file.Imports))
	decls := file.Decls[:0]
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			decls = append(decls, decl)
			continue
		}

		for _, spec := range gen.Specs {
			importSpec, ok := spec.(*ast.ImportSpec)
			if !ok {
				continue
			}
			importSummaries = append(importSummaries, summarizeGoImport(importSpec))
		}
	}

	file.Decls = decls
	file.Imports = nil

	var body bytes.Buffer
	if err := format.Node(&body, fset, file); err != nil {
		return "", err
	}
	if len(importSummaries) == 0 {
		return body.String(), nil
	}

	var out bytes.Buffer
	out.WriteString("// imports: ")
	out.WriteString(strings.Join(importSummaries, "; "))
	out.WriteString("\n\n")
	out.WriteString(body.String())
	return out.String(), nil
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

func summarizeGoImport(importSpec *ast.ImportSpec) string {
	path := strings.Trim(importSpec.Path.Value, "\"'")
	if unquoted, err := strconv.Unquote(importSpec.Path.Value); err == nil {
		path = unquoted
	}
	if importSpec.Name == nil {
		return path
	}
	return fmt.Sprintf("%s=%s", importSpec.Name.Name, path)
}

//go:build cgo

package codectx

import (
	"context"
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/javascript"
	tsx "github.com/smacker/go-tree-sitter/typescript/tsx"
	ts "github.com/smacker/go-tree-sitter/typescript/typescript"
)

func newParserForExt(ext string) (*sitter.Parser, error) {
	parser := sitter.NewParser()

	switch ext {
	case ".ts":
		parser.SetLanguage(ts.GetLanguage())
	case ".tsx":
		parser.SetLanguage(tsx.GetLanguage())
	case ".js", ".jsx":
		parser.SetLanguage(javascript.GetLanguage())
	default:
		parser.Close()
		return nil, fmt.Errorf("unsupported tree-sitter extension %q", ext)
	}

	return parser, nil
}

func skeletonTreeSitter(ext string, src []byte) (string, []string, error) {
	parser, err := newParserForExt(ext)
	if err != nil {
		return "", nil, err
	}
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return "", []string{"tree-sitter parse failed, using heuristic fallback"}, err
	}
	defer tree.Close()

	root := tree.RootNode()
	if root == nil || root.HasError() {
		return "", []string{"tree-sitter partial parse (ERROR nodes), using heuristic fallback"}, fmt.Errorf("partial parse")
	}

	return skeletonProgram(root, src), nil, nil
}

func trimImportsTreeSitter(ext string, src []byte) (string, []string, error) {
	parser, err := newParserForExt(ext)
	if err != nil {
		return "", nil, err
	}
	defer parser.Close()

	tree, err := parser.ParseCtx(context.Background(), nil, src)
	if err != nil {
		return "", []string{"tree-sitter parse failed, using heuristic fallback"}, err
	}
	defer tree.Close()

	root := tree.RootNode()
	if root == nil || root.HasError() {
		return "", []string{"tree-sitter partial parse, using heuristic fallback"}, fmt.Errorf("partial parse")
	}

	return trimImportsProgram(root, src), nil, nil
}

func skeletonProgram(root *sitter.Node, src []byte) string {
	var out strings.Builder
	prev := int(root.StartByte())

	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		start := int(child.StartByte())
		end := int(child.EndByte())

		out.WriteString(srcSlice(src, prev, start))
		out.WriteString(skeletonTopLevelNode(child, src))
		prev = end
	}

	out.WriteString(srcSlice(src, prev, int(root.EndByte())))
	return out.String()
}

func skeletonTopLevelNode(node *sitter.Node, src []byte) string {
	switch node.Type() {
	case "function_declaration":
		return skeletonFunctionLike(node, src)
	case "class_declaration":
		return skeletonClass(node, src)
	case "interface_declaration", "type_alias_declaration", "enum_declaration":
		return node.Content(src)
	case "lexical_declaration":
		return skeletonLexicalDeclaration(node, src)
	case "export_statement":
		return skeletonExportStatement(node, src)
	default:
		return node.Content(src)
	}
}

func skeletonExportStatement(node *sitter.Node, src []byte) string {
	if node.NamedChildCount() == 0 {
		return node.Content(src)
	}

	decl := node.NamedChild(0)
	return srcSlice(src, int(node.StartByte()), int(decl.StartByte())) +
		skeletonTopLevelNode(decl, src) +
		srcSlice(src, int(decl.EndByte()), int(node.EndByte()))
}

func skeletonLexicalDeclaration(node *sitter.Node, src []byte) string {
	if node.NamedChildCount() != 1 {
		return node.Content(src)
	}

	declarator := node.NamedChild(0)
	if declarator.Type() != "variable_declarator" || declarator.NamedChildCount() < 2 {
		return node.Content(src)
	}

	value := declarator.NamedChild(int(declarator.NamedChildCount()) - 1)
	if value.Type() != "arrow_function" {
		return node.Content(src)
	}

	return srcSlice(src, int(node.StartByte()), int(value.StartByte())) +
		skeletonArrowFunction(value, src) +
		srcSlice(src, int(value.EndByte()), int(node.EndByte()))
}

func skeletonClass(node *sitter.Node, src []byte) string {
	body := node.ChildByFieldName("body")
	if body == nil {
		return node.Content(src)
	}

	return srcSlice(src, int(node.StartByte()), int(body.StartByte())) +
		skeletonClassBody(body, src) +
		srcSlice(src, int(body.EndByte()), int(node.EndByte()))
}

func skeletonClassBody(node *sitter.Node, src []byte) string {
	var out strings.Builder
	prev := int(node.StartByte())

	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		start := int(child.StartByte())
		end := int(child.EndByte())

		out.WriteString(srcSlice(src, prev, start))
		if child.Type() == "method_definition" {
			out.WriteString(skeletonFunctionLike(child, src))
		} else {
			out.WriteString(child.Content(src))
		}
		prev = end
	}

	out.WriteString(srcSlice(src, prev, int(node.EndByte())))
	return out.String()
}

func skeletonFunctionLike(node *sitter.Node, src []byte) string {
	body := node.ChildByFieldName("body")
	if body == nil {
		return node.Content(src)
	}

	return trimRightSpace(srcSlice(src, int(node.StartByte()), int(body.StartByte()))) + ";"
}

func skeletonArrowFunction(node *sitter.Node, src []byte) string {
	body := node.ChildByFieldName("body")
	if body == nil {
		return node.Content(src)
	}

	return trimRightSpace(srcSlice(src, int(node.StartByte()), int(body.StartByte()))) + " {}"
}

func trimRightSpace(value string) string {
	return strings.TrimRight(value, " \t\r\n")
}

func srcSlice(src []byte, start, end int) string {
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	if end > len(src) {
		end = len(src)
	}
	return string(src[start:end])
}

func trimImportsProgram(root *sitter.Node, src []byte) string {
	var importSummaries []string
	var typeSummaries []string
	var body strings.Builder
	prev := int(root.StartByte())

	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		end := int(child.EndByte())

		if child.Type() == "import_statement" {
			module, names, typeOnly := importSummaryParts(child, src)
			if typeOnly {
				if len(names) > 0 {
					typeSummaries = append(typeSummaries, fmt.Sprintf("%s from %s", strings.Join(names, ", "), module))
				}
			} else if len(names) == 0 {
				importSummaries = append(importSummaries, fmt.Sprintf("%s(side-effect)", module))
			} else {
				importSummaries = append(importSummaries, formatImportSummary(module, names))
			}
		} else {
			body.WriteString(srcSlice(src, prev, end))
		}

		prev = end
	}

	body.WriteString(srcSlice(src, prev, int(root.EndByte())))

	var out strings.Builder
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
	out.WriteString(strings.TrimLeft(body.String(), "\r\n"))
	return out.String()
}

func importSummaryParts(node *sitter.Node, src []byte) (string, []string, bool) {
	module := ""
	typeOnly := false
	var names []string

	var clause *sitter.Node
	for i := 0; i < int(node.NamedChildCount()); i++ {
		child := node.NamedChild(i)
		switch child.Type() {
		case "import_clause":
			clause = child
		case "string":
			module = strings.Trim(child.Content(src), "\"'")
		}
	}

	if clause == nil {
		return module, nil, false
	}

	if strings.Contains(srcSlice(src, int(node.StartByte()), int(clause.StartByte())), "import type") {
		typeOnly = true
	}

	for i := 0; i < int(clause.NamedChildCount()); i++ {
		child := clause.NamedChild(i)
		switch child.Type() {
		case "identifier", "namespace_import":
			names = append(names, child.Content(src))
		case "named_imports":
			for j := 0; j < int(child.NamedChildCount()); j++ {
				specifier := child.NamedChild(j)
				if specifier.Type() == "import_specifier" {
					names = append(names, specifier.Content(src))
				}
			}
		}
	}

	return module, names, typeOnly
}

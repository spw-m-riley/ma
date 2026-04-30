//go:build !cgo

package codectx

import "fmt"

func tsjsTrimImports(ext string, src []byte) (string, []string, error) {
	return "", nil, fmt.Errorf("tree-sitter unavailable (CGo disabled)")
}

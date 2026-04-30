//go:build cgo

package codectx

func tsjsTrimImports(ext string, src []byte) (string, []string, error) {
	return trimImportsTreeSitter(ext, src)
}

//go:build cgo

package codectx

func tsjsSkeleton(ext string, src []byte) (string, []string, error) {
	return skeletonTreeSitter(ext, src)
}

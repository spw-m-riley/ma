package codectx

func ReduceFile(path string, src []byte) (string, []string, error) {
	current := src
	var findings []string

	if trimmed, warnings, err := TrimImportsFile(path, current); err == nil {
		current = []byte(trimmed)
		findings = append(findings, warnings...)
	}

	output, warnings, err := SkeletonFile(path, current)
	if err != nil {
		return "", nil, err
	}

	findings = append(findings, warnings...)
	return output, findings, nil
}

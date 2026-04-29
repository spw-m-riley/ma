package schema

import "encoding/json"

func MinifyJSON(input string) (string, error) {
	var node any
	if err := json.Unmarshal([]byte(input), &node); err != nil {
		return "", err
	}

	pruned := pruneSchemaNode(node)
	output, err := json.Marshal(pruned)
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func pruneSchemaNode(node any) any {
	switch value := node.(type) {
	case map[string]any:
		delete(value, "description")
		delete(value, "examples")
		delete(value, "default")
		for key, child := range value {
			value[key] = pruneSchemaNode(child)
		}
		return value
	case []any:
		for i, child := range value {
			value[i] = pruneSchemaNode(child)
		}
		return value
	default:
		return value
	}
}

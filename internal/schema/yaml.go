package schema

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

var removableYAMLKeys = map[string]struct{}{
	"description": {},
	"default":     {},
	"examples":    {},
}

func MinifyYAML(input string) (string, error) {
	if err := rejectTabs(input); err != nil {
		return "", err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(input), &doc); err != nil {
		return "", err
	}
	if err := validateNode(&doc); err != nil {
		return "", err
	}

	pruneKeys(&doc, removableYAMLKeys)
	if isEmptyYAMLDocument(&doc) {
		return "", nil
	}

	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if err := encoder.Encode(&doc); err != nil {
		return "", err
	}
	if err := encoder.Close(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func rejectTabs(input string) error {
	for _, line := range strings.Split(input, "\n") {
		if !strings.ContainsRune(line, '\t') {
			continue
		}
		return fmt.Errorf("unsupported yaml feature: tabs")
	}
	return nil
}

func validateNode(node *yaml.Node) error {
	if node.Kind == yaml.AliasNode {
		return fmt.Errorf("unsupported yaml feature: aliases")
	}
	if node.Anchor != "" {
		return fmt.Errorf("unsupported yaml feature: anchors")
	}

	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			if node.Content[i].Value == "<<" {
				return fmt.Errorf("unsupported yaml feature: merge keys")
			}
		}
	}

	for _, child := range node.Content {
		if err := validateNode(child); err != nil {
			return err
		}
	}

	return nil
}

func pruneKeys(node *yaml.Node, keys map[string]struct{}) {
	if node.Kind == yaml.DocumentNode {
		for _, child := range node.Content {
			pruneKeys(child, keys)
		}
		return
	}

	if node.Kind != yaml.MappingNode {
		for _, child := range node.Content {
			pruneKeys(child, keys)
		}
		return
	}

	filtered := make([]*yaml.Node, 0, len(node.Content))
	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i]
		value := node.Content[i+1]
		if _, remove := keys[key.Value]; remove {
			continue
		}
		pruneKeys(value, keys)
		filtered = append(filtered, key, value)
	}
	node.Content = filtered
}

func isEmptyYAMLDocument(node *yaml.Node) bool {
	switch node.Kind {
	case 0:
		return true
	case yaml.DocumentNode:
		return len(node.Content) == 0 || (len(node.Content) == 1 && isEmptyYAMLDocument(node.Content[0]))
	case yaml.MappingNode, yaml.SequenceNode:
		return len(node.Content) == 0
	default:
		return false
	}
}

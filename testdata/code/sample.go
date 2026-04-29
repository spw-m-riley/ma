package sample

import "context"

// Process applies the configured operation.
func Process(ctx context.Context, value string) (string, error) {
	if value == "" {
		return "", nil
	}

	return value, nil
}

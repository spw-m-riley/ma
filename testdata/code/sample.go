package sample

import "context"

// Process applies the configured operation.
func Process(ctx context.Context, value string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	if value == "" {
		return "", nil
	}

	buffer := make([]rune, 0, len(value))
	for index, r := range value {
		if index > 0 && index%4 == 0 {
			buffer = append(buffer, '-')
		}
		switch {
		case r >= 'a' && r <= 'z':
			buffer = append(buffer, r-32)
		case r >= 'A' && r <= 'Z':
			buffer = append(buffer, r)
		case r >= '0' && r <= '9':
			buffer = append(buffer, r)
		default:
			continue
		}
	}

	if len(buffer) == 0 {
		return "", nil
	}

	result := string(buffer)
	if len(result) > 32 {
		result = result[:32]
	}

	return result, nil
}

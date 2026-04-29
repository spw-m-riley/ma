package app

import (
	"encoding/json"
	"fmt"
	"io"
)

type Result struct {
	Command  string   `json:"command"`
	Changed  bool     `json:"changed"`
	Stats    Stats    `json:"stats"`
	Findings []string `json:"findings,omitempty"`
	Output   string   `json:"output,omitempty"`
}

func WriteResult(w io.Writer, result Result, jsonOutput bool) error {
	if jsonOutput {
		encoder := json.NewEncoder(w)
		return encoder.Encode(result)
	}

	// Human mode: render body first if present, then append findings
	if result.Output != "" {
		if _, err := io.WriteString(w, result.Output); err != nil {
			return err
		}
	} else {
		// No body: render command summary line first
		if _, err := fmt.Fprintf(w, "%s changed=%t\n", result.Command, result.Changed); err != nil {
			return err
		}
	}

	// Append findings/warnings in a stable block
	if len(result.Findings) > 0 {
		if _, err := io.WriteString(w, "\nFindings:\n"); err != nil {
			return err
		}
		for _, finding := range result.Findings {
			if _, err := fmt.Fprintf(w, "  - %s\n", finding); err != nil {
				return err
			}
		}
	}

	return nil
}

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

	if result.Output != "" {
		_, err := io.WriteString(w, result.Output)
		return err
	}

	_, err := fmt.Fprintf(w, "%s changed=%t\n", result.Command, result.Changed)
	return err
}

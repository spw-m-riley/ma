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

	_, err := fmt.Fprintf(w, "%s changed=%t\n", result.Command, result.Changed)
	return err
}

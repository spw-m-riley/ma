package app

import "strings"

type Stats struct {
	InputBytes         int `json:"inputBytes"`
	OutputBytes        int `json:"outputBytes"`
	InputWords         int `json:"inputWords"`
	OutputWords        int `json:"outputWords"`
	InputApproxTokens  int `json:"inputApproxTokens"`
	OutputApproxTokens int `json:"outputApproxTokens"`
}

func Measure(input string, output string) Stats {
	return Stats{
		InputBytes:         len(input),
		OutputBytes:        len(output),
		InputWords:         len(strings.Fields(input)),
		OutputWords:        len(strings.Fields(output)),
		InputApproxTokens:  approxTokens(input),
		OutputApproxTokens: approxTokens(output),
	}
}

func approxTokens(input string) int {
	chars := len(input) / 4
	words := (len(strings.Fields(input)) * 4) / 3
	if chars > words {
		return chars
	}
	return words
}

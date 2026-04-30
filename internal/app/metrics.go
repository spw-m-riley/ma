package app

import (
	"strings"

	"github.com/tiktoken-go/tokenizer"
)

var defaultCodec = mustDefaultCodec()

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
		InputApproxTokens:  countTokens(input),
		OutputApproxTokens: countTokens(output),
	}
}

func mustDefaultCodec() tokenizer.Codec {
	codec, err := tokenizer.Get(tokenizer.Cl100kBase)
	if err != nil {
		panic("initialize cl100k_base tokenizer: " + err.Error())
	}
	return codec
}

func countTokens(input string) int {
	ids, _, _ := defaultCodec.Encode(input)
	return len(ids)
}

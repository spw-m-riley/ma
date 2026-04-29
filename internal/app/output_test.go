package app

import (
	"bytes"
	"testing"
)

func TestWriteResultJSON(t *testing.T) {
	var out bytes.Buffer

	result := Result{
		Command: "compress",
		Changed: true,
		Stats: Stats{
			InputBytes:         120,
			OutputBytes:        80,
			InputWords:         30,
			OutputWords:        20,
			InputApproxTokens:  32,
			OutputApproxTokens: 22,
		},
		Findings: []string{"compressed prose"},
		Output:   "reduced content",
	}

	if err := WriteResult(&out, result, true); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := "{\"command\":\"compress\",\"changed\":true,\"stats\":{\"inputBytes\":120,\"outputBytes\":80,\"inputWords\":30,\"outputWords\":20,\"inputApproxTokens\":32,\"outputApproxTokens\":22},\"findings\":[\"compressed prose\"],\"output\":\"reduced content\"}\n"
	if got := out.String(); got != want {
		t.Fatalf("unexpected json output\nwant: %q\ngot:  %q", want, got)
	}
}

package app

import (
	"bytes"
	"strings"
	"testing"
)

type stubCommand struct {
	name string
}

func (c stubCommand) Name() string {
	return c.name
}

func (c stubCommand) Run(_ []string) (Result, error) {
	return Result{}, nil
}

func TestAppHelpListsCommands(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	application := New(
		&stdout,
		&stderr,
		stubCommand{name: "compress"},
		stubCommand{name: "validate"},
		stubCommand{name: "optimize-md"},
	)

	exitCode := application.Run([]string{"--help"})
	if exitCode != 0 {
		t.Fatalf("expected zero exit code, got %d", exitCode)
	}

	help := stdout.String()
	for _, name := range []string{"compress", "validate", "optimize-md"} {
		if !strings.Contains(help, name) {
			t.Fatalf("expected help output to include command %q, got %q", name, help)
		}
	}
}

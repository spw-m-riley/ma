package app

import (
	"bytes"
	"strings"
	"testing"
)

type stubCommand struct {
	name   string
	result Result
}

func (c stubCommand) Name() string {
	return c.name
}

func (c stubCommand) Run(_ []string) (Result, error) {
	return Result{Command: c.name}, nil
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

func TestAppRunWritesCommandResult(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	application := New(
		&stdout,
		&stderr,
		stubCommand{name: "compress"},
	)

	exitCode := application.Run([]string{"compress"})
	if exitCode != 0 {
		t.Fatalf("expected zero exit code, got %d", exitCode)
	}

	if !strings.Contains(stdout.String(), "compress changed=false") {
		t.Fatalf("expected command output to be rendered, got %q", stdout.String())
	}
}

func TestHumanModeOutputWithFindings(t *testing.T) {
	var stdout bytes.Buffer

	result := Result{
		Command:  "compress",
		Changed:  true,
		Output:   "compressed output",
		Findings: []string{"path mismatch", "bullet count drift"},
	}

	err := WriteResult(&stdout, result, false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := stdout.String()
	
	// Should contain the output body
	if !strings.Contains(output, "compressed output") {
		t.Fatalf("expected output to contain body, got %q", output)
	}
	
	// Should contain the findings/warnings
	if !strings.Contains(output, "path mismatch") {
		t.Fatalf("expected output to contain findings, got %q", output)
	}
	if !strings.Contains(output, "bullet count drift") {
		t.Fatalf("expected output to contain all findings, got %q", output)
	}
}

func TestHumanModeOutputWithoutBodyShowsCommandAndFindings(t *testing.T) {
	var stdout bytes.Buffer

	result := Result{
		Command:  "validate",
		Changed:  false,
		Findings: []string{"heading mismatch", "code block mismatch"},
	}

	err := WriteResult(&stdout, result, false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := stdout.String()
	
	// Should contain the command summary line
	if !strings.Contains(output, "validate") {
		t.Fatalf("expected output to show command, got %q", output)
	}
	
	// Should still contain the findings even without body
	if !strings.Contains(output, "heading mismatch") {
		t.Fatalf("expected output to contain findings, got %q", output)
	}
}

package validate

import (
	"fmt"
	"os"
	"strings"

	"github.com/spw-m-riley/ma/internal/app"
)

type Command struct{}

func NewCommand() Command {
	return Command{}
}

func (Command) Name() string {
	return "validate"
}

func (Command) Run(args []string) (app.Result, error) {
	if len(args) != 2 {
		return app.Result{}, fmt.Errorf("usage: ma validate <original> <candidate> [--json]")
	}

	originalBytes, err := os.ReadFile(args[0])
	if err != nil {
		return app.Result{}, err
	}
	candidateBytes, err := os.ReadFile(args[1])
	if err != nil {
		return app.Result{}, err
	}

	original := string(originalBytes)
	candidate := string(candidateBytes)
	report := Compare(original, candidate)
	if !report.Valid {
		return app.Result{}, fmt.Errorf("validation failed: %s", strings.Join(report.Errors, ", "))
	}

	return app.Result{
		Command:  "validate",
		Changed:  false,
		Stats:    app.Measure(original, candidate),
		Findings: report.Warnings,
		Output:   "valid\n",
	}, nil
}

package dedup

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
	return "dedup"
}

func (Command) Run(args []string) (app.Result, error) {
	if len(args) == 0 {
		return app.Result{}, fmt.Errorf("usage: ma dedup <path...> [--json]")
	}

	report, err := Analyze(args, 0.6)
	if err != nil {
		return app.Result{}, err
	}

	output := renderReport(report)
	inputSummary, err := readInputSummary(args)
	if err != nil {
		return app.Result{}, err
	}
	return app.Result{
		Command: "dedup",
		Changed: true,
		Stats:   app.Measure(inputSummary, output),
		Output:  output,
	}, nil
}

func renderReport(report Report) string {
	var out strings.Builder
	out.WriteString("Exact duplicates:\n")
	if len(report.Exact) == 0 {
		out.WriteString("- none\n")
	} else {
		for _, duplicate := range report.Exact {
			out.WriteString(fmt.Sprintf("- %s [%s]\n", duplicate.Sentence, strings.Join(duplicate.Locations, ", ")))
		}
	}

	out.WriteString("Near duplicates:\n")
	if len(report.Near) == 0 {
		out.WriteString("- none\n")
	} else {
		for _, duplicate := range report.Near {
			out.WriteString(fmt.Sprintf("- %.2f %s [%s]\n", duplicate.Similarity, duplicate.Sentence, strings.Join(duplicate.Locations, ", ")))
		}
	}

	return out.String()
}

func readInputSummary(paths []string) (string, error) {
	var out strings.Builder
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		out.Write(content)
	}
	return out.String(), nil
}

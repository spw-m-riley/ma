package history

import (
	"flag"
	"fmt"
	"os"

	"github.com/spw-m-riley/ma/internal/app"
	"github.com/spw-m-riley/ma/internal/detect"
)

type Command struct{}

func NewCommand() Command {
	return Command{}
}

func (Command) Name() string {
	return "compact-history"
}

func (Command) Run(args []string) (app.Result, error) {
	fs := flag.NewFlagSet("compact-history", flag.ContinueOnError)
	write := fs.Bool("write", false, "write compacted transcript back to file")
	if err := fs.Parse(args); err != nil {
		return app.Result{}, err
	}
	if fs.NArg() != 1 {
		return app.Result{}, fmt.Errorf("usage: ma compact-history <transcript> [--write] [--json]")
	}

	path := fs.Arg(0)
	if detect.IsSensitivePath(path) {
		return app.Result{}, fmt.Errorf("refusing sensitive path %q", path)
	}

	inputBytes, err := os.ReadFile(path)
	if err != nil {
		return app.Result{}, err
	}

	output, err := CompactJSON(string(inputBytes))
	if err != nil {
		return app.Result{}, err
	}
	if *write {
		if err := app.WriteWithBackup(path, string(inputBytes), output); err != nil {
			return app.Result{}, err
		}
	}

	return app.Result{
		Command: "compact-history",
		Changed: true,
		Stats:   app.Measure(string(inputBytes), output),
		Output:  output,
	}, nil
}

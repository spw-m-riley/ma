package schema

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spw-m-riley/ma/internal/app"
	"github.com/spw-m-riley/ma/internal/detect"
)

type Command struct{}

func NewCommand() Command {
	return Command{}
}

func (Command) Name() string {
	return "minify-schema"
}

func (Command) Run(args []string) (app.Result, error) {
	fs := flag.NewFlagSet("minify-schema", flag.ContinueOnError)
	write := fs.Bool("write", false, "write minified schema back to file")
	if err := fs.Parse(args); err != nil {
		return app.Result{}, err
	}
	if fs.NArg() != 1 {
		return app.Result{}, fmt.Errorf("usage: ma minify-schema <file> [--write] [--json]")
	}

	path := fs.Arg(0)
	if detect.IsSensitivePath(path) {
		return app.Result{}, fmt.Errorf("refusing sensitive path %q", path)
	}

	inputBytes, err := os.ReadFile(path)
	if err != nil {
		return app.Result{}, err
	}
	input := string(inputBytes)

	output, err := minifyByExtension(path, input)
	if err != nil {
		return app.Result{}, err
	}
	if *write {
		if err := app.WriteWithBackup(path, input, output); err != nil {
			return app.Result{}, err
		}
	}

	return app.Result{
		Command: "minify-schema",
		Changed: output != input,
		Stats:   app.Measure(input, output),
		Output:  output,
	}, nil
}

func minifyByExtension(path string, input string) (string, error) {
	switch filepath.Ext(path) {
	case ".json":
		return MinifyJSON(input)
	case ".yaml", ".yml":
		return MinifyYAML(input)
	default:
		return "", fmt.Errorf("unsupported schema extension %q", filepath.Ext(path))
	}
}

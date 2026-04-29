package prose

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spw-m-riley/ma/internal/app"
	"github.com/spw-m-riley/ma/internal/detect"
	"github.com/spw-m-riley/ma/internal/validate"
)

type Command struct{}

func NewCommand() Command {
	return Command{}
}

func (Command) Name() string {
	return "compress"
}

func (Command) Run(args []string) (app.Result, error) {
	fs := flag.NewFlagSet("compress", flag.ContinueOnError)
	write := fs.Bool("write", false, "write output back to file")
	if err := fs.Parse(args); err != nil {
		return app.Result{}, err
	}
	if fs.NArg() != 1 {
		return app.Result{}, fmt.Errorf("usage: ma compress <file> [--write] [--json]")
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

	if detect.Classify(path, input) != detect.NaturalLanguage {
		return app.Result{}, fmt.Errorf("compress only supports natural language files")
	}

	output := Compress(input)
	report := validate.Compare(input, output)
	if !report.Valid {
		return app.Result{}, report.Error()
	}

	if *write {
		if err := writeOutput(path, input, output); err != nil {
			return app.Result{}, err
		}
	}

	return app.Result{
		Command:  "compress",
		Changed:  output != input,
		Stats:    app.Measure(input, output),
		Findings: report.Warnings,
		Output:   output,
	}, nil
}

func writeOutput(path string, original string, output string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	backupPath := path + ".ma.bak"
	tempPath := path + ".ma.tmp"

	if err := os.WriteFile(backupPath, []byte(original), info.Mode().Perm()); err != nil {
		return err
	}

	restore := func(writeErr error) error {
		_ = os.Remove(tempPath)
		_ = os.WriteFile(path, []byte(original), info.Mode().Perm())
		return writeErr
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return restore(err)
	}
	if err := os.WriteFile(tempPath, []byte(output), info.Mode().Perm()); err != nil {
		return restore(err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return restore(err)
	}

	return nil
}

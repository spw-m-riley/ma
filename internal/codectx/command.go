package codectx

import (
	"fmt"
	"os"

	"github.com/spw-m-riley/ma/internal/app"
	"github.com/spw-m-riley/ma/internal/detect"
)

type SkeletonCommand struct{}

func NewSkeletonCommand() SkeletonCommand {
	return SkeletonCommand{}
}

func (SkeletonCommand) Name() string {
	return "skeleton"
}

func (SkeletonCommand) Run(args []string) (app.Result, error) {
	if len(args) != 1 {
		return app.Result{}, fmt.Errorf("usage: ma skeleton <file> [--json]")
	}
	path := args[0]
	if detect.IsSensitivePathResolved(path) {
		return app.Result{}, fmt.Errorf("refusing sensitive path %q", path)
	}

	input, err := os.ReadFile(path)
	if err != nil {
		return app.Result{}, err
	}

	output, warnings, err := SkeletonFile(path, input)
	if err != nil {
		return app.Result{}, err
	}

	return app.Result{
		Command:  "skeleton",
		Changed:  true,
		Stats:    app.Measure(string(input), output),
		Findings: warnings,
		Output:   output,
	}, nil
}

type TrimImportsCommand struct{}

func NewTrimImportsCommand() TrimImportsCommand {
	return TrimImportsCommand{}
}

func (TrimImportsCommand) Name() string {
	return "trim-imports"
}

func (TrimImportsCommand) Run(args []string) (app.Result, error) {
	if len(args) != 1 {
		return app.Result{}, fmt.Errorf("usage: ma trim-imports <file> [--json]")
	}
	path := args[0]
	if detect.IsSensitivePathResolved(path) {
		return app.Result{}, fmt.Errorf("refusing sensitive path %q", path)
	}

	input, err := os.ReadFile(path)
	if err != nil {
		return app.Result{}, err
	}

	output, warnings, err := TrimImportsFile(path, input)
	if err != nil {
		return app.Result{}, err
	}

	return app.Result{
		Command:  "trim-imports",
		Changed:  true,
		Stats:    app.Measure(string(input), output),
		Findings: warnings,
		Output:   output,
	}, nil
}

package main

import (
	"errors"
	"io"
	"os"

	"github.com/spw-m-riley/ma/internal/app"
)

var errNotImplemented = errors.New("not implemented")

type placeholderCommand struct {
	name string
}

func (c placeholderCommand) Name() string {
	return c.name
}

func (c placeholderCommand) Run(_ []string) (app.Result, error) {
	return app.Result{Command: c.name}, errNotImplemented
}

func newApp(stdout io.Writer, stderr io.Writer) *app.App {
	return app.New(
		stdout,
		stderr,
		placeholderCommand{name: "compress"},
		placeholderCommand{name: "validate"},
		placeholderCommand{name: "optimize-md"},
		placeholderCommand{name: "minify-schema"},
		placeholderCommand{name: "skeleton"},
		placeholderCommand{name: "trim-imports"},
		placeholderCommand{name: "dedup"},
		placeholderCommand{name: "compact-history"},
	)
}

func main() {
	os.Exit(newApp(os.Stdout, os.Stderr).Run(os.Args[1:]))
}

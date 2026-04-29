package app

import (
	"fmt"
	"io"
	"sort"
)

type Command interface {
	Name() string
	Run(args []string) (Result, error)
}

type App struct {
	stdout   io.Writer
	stderr   io.Writer
	commands map[string]Command
}

func New(stdout io.Writer, stderr io.Writer, commands ...Command) *App {
	registry := make(map[string]Command, len(commands))
	for _, command := range commands {
		registry[command.Name()] = command
	}

	return &App{
		stdout:   stdout,
		stderr:   stderr,
		commands: registry,
	}
}

func (a *App) Run(args []string) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		a.renderHelp()
		return 0
	}

	command, ok := a.commands[args[0]]
	if !ok {
		fmt.Fprintf(a.stderr, "unknown command %q\n", args[0])
		a.renderHelp()
		return 2
	}

	if _, err := command.Run(args[1:]); err != nil {
		fmt.Fprintf(a.stderr, "%v\n", err)
		return 1
	}

	return 0
}

func (a *App) renderHelp() {
	names := make([]string, 0, len(a.commands))
	for name := range a.commands {
		names = append(names, name)
	}
	sort.Strings(names)

	fmt.Fprintln(a.stdout, "ma commands:")
	for _, name := range names {
		fmt.Fprintf(a.stdout, "  %s\n", name)
	}
}

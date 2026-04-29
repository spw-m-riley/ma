package main

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/spw-m-riley/ma/internal/app"
	codectxcmd "github.com/spw-m-riley/ma/internal/codectx"
	markdowncmd "github.com/spw-m-riley/ma/internal/markdown"
	"github.com/spw-m-riley/ma/internal/prose"
	schemacmd "github.com/spw-m-riley/ma/internal/schema"
	validatecmd "github.com/spw-m-riley/ma/internal/validate"
)

func newRootCommand(stdout io.Writer, stderr io.Writer) *cobra.Command {
	var jsonOutput bool

	root := &cobra.Command{
		Use:           "ma",
		Short:         "Deterministic context-reduction tooling",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.PersistentFlags().BoolVar(&jsonOutput, "json", false, "emit JSON output")

	root.AddCommand(
		newCompressCommand(stdout, &jsonOutput),
		newValidateCommand(stdout, &jsonOutput),
		newOptimizeMarkdownCommand(stdout, &jsonOutput),
		newMinifySchemaCommand(stdout, &jsonOutput),
		newSkeletonCommand(stdout, &jsonOutput),
		newTrimImportsCommand(stdout, &jsonOutput),
		notImplementedCommand("dedup"),
		notImplementedCommand("compact-history"),
	)

	return root
}

func newCompressCommand(stdout io.Writer, jsonOutput *bool) *cobra.Command {
	var write bool

	command := &cobra.Command{
		Use:   "compress <file>",
		Short: "Compress prose deterministically",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			runArgs := []string{args[0]}
			if write {
				runArgs = append([]string{"--write"}, runArgs...)
			}

			result, err := prose.NewCommand().Run(runArgs)
			if err != nil {
				return err
			}
			return app.WriteResult(stdout, result, *jsonOutput)
		},
	}
	command.Flags().BoolVar(&write, "write", false, "write compressed output back to file")

	return command
}

func newValidateCommand(stdout io.Writer, jsonOutput *bool) *cobra.Command {
	command := &cobra.Command{
		Use:   "validate <original> <candidate>",
		Short: "Validate preserved structure between two files",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			result, err := validatecmd.NewCommand().Run(args)
			if err != nil {
				return err
			}
			return app.WriteResult(stdout, result, *jsonOutput)
		},
	}

	return command
}

func newOptimizeMarkdownCommand(stdout io.Writer, jsonOutput *bool) *cobra.Command {
	var write bool

	command := &cobra.Command{
		Use:   "optimize-md <file>",
		Short: "Optimize markdown structure deterministically",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			runArgs := []string{args[0]}
			if write {
				runArgs = append([]string{"--write"}, runArgs...)
			}

			result, err := markdowncmd.NewCommand().Run(runArgs)
			if err != nil {
				return err
			}
			return app.WriteResult(stdout, result, *jsonOutput)
		},
	}
	command.Flags().BoolVar(&write, "write", false, "write optimized markdown back to file")

	return command
}

func newMinifySchemaCommand(stdout io.Writer, jsonOutput *bool) *cobra.Command {
	var write bool

	command := &cobra.Command{
		Use:   "minify-schema <file>",
		Short: "Minify JSON or YAML schema files",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			runArgs := []string{args[0]}
			if write {
				runArgs = append([]string{"--write"}, runArgs...)
			}

			result, err := schemacmd.NewCommand().Run(runArgs)
			if err != nil {
				return err
			}
			return app.WriteResult(stdout, result, *jsonOutput)
		},
	}
	command.Flags().BoolVar(&write, "write", false, "write minified schema back to file")

	return command
}

func newSkeletonCommand(stdout io.Writer, jsonOutput *bool) *cobra.Command {
	command := &cobra.Command{
		Use:   "skeleton <file>",
		Short: "Reduce source to declarations and signatures",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			result, err := codectxcmd.NewSkeletonCommand().Run(args)
			if err != nil {
				return err
			}
			return app.WriteResult(stdout, result, *jsonOutput)
		},
	}

	return command
}

func newTrimImportsCommand(stdout io.Writer, jsonOutput *bool) *cobra.Command {
	command := &cobra.Command{
		Use:   "trim-imports <file>",
		Short: "Summarize import blocks for code context",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			result, err := codectxcmd.NewTrimImportsCommand().Run(args)
			if err != nil {
				return err
			}
			return app.WriteResult(stdout, result, *jsonOutput)
		},
	}

	return command
}

func notImplementedCommand(name string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: "Not implemented yet",
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("%s not implemented yet", name)
		},
	}
}

func main() {
	if err := newRootCommand(os.Stdout, os.Stderr).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

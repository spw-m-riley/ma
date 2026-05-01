package maintain

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spw-m-riley/ma/internal/app"
	"github.com/spw-m-riley/ma/internal/dedup"
	"github.com/spw-m-riley/ma/internal/detect"
	"github.com/spw-m-riley/ma/internal/prose"
	"github.com/spw-m-riley/ma/internal/validate"
)

type Command struct{}

func NewCommand() Command {
	return Command{}
}

func (Command) Name() string {
	return "maintain"
}

func (Command) Run(args []string) (app.Result, error) {
	fs := flag.NewFlagSet("maintain", flag.ContinueOnError)
	write := fs.Bool("write", false, "write compressed output back to files")
	if err := fs.Parse(args); err != nil {
		return app.Result{}, err
	}
	if fs.NArg() != 1 {
		return app.Result{}, fmt.Errorf("usage: ma maintain <directory> [--write] [--json]")
	}

	root := fs.Arg(0)
	info, err := os.Stat(root)
	if err != nil {
		return app.Result{}, err
	}
	if !info.IsDir() {
		return app.Result{}, fmt.Errorf("%s is not a directory", root)
	}

	files, err := collectFiles(root)
	if err != nil {
		return app.Result{}, err
	}

	if len(files) == 0 {
		return app.Result{
			Command:  "maintain",
			Changed:  false,
			Stats:    app.Stats{},
			Findings: []string{"no eligible files found"},
		}, nil
	}

	var (
		totalStats   app.Stats
		findings     []string
		filesChanged int
		proseFiles   []string
	)

	for _, path := range files {
		inputBytes, err := os.ReadFile(path)
		if err != nil {
			findings = append(findings, fmt.Sprintf("skip %s: %v", path, err))
			continue
		}
		input := string(inputBytes)

		classification := detect.Classify(path, input)
		if classification != detect.NaturalLanguage {
			findings = append(findings, fmt.Sprintf("skip %s: classification=%s", path, classification))
			continue
		}

		proseFiles = append(proseFiles, path)

		output := prose.Compress(input)
		report := validate.Compare(input, output)
		if !report.Valid {
			findings = append(findings, fmt.Sprintf("skip %s: validation failed: %v", path, report.Error()))
			continue
		}

		stats := app.Measure(input, output)
		totalStats.InputBytes += stats.InputBytes
		totalStats.OutputBytes += stats.OutputBytes
		totalStats.InputWords += stats.InputWords
		totalStats.OutputWords += stats.OutputWords
		totalStats.InputApproxTokens += stats.InputApproxTokens
		totalStats.OutputApproxTokens += stats.OutputApproxTokens

		if output != input {
			filesChanged++
			findings = append(findings, fmt.Sprintf("compressed %s: saved %d bytes, %d approx tokens",
				path,
				stats.InputBytes-stats.OutputBytes,
				stats.InputApproxTokens-stats.OutputApproxTokens))

			if *write {
				if err := app.WriteWithBackup(path, input, output); err != nil {
					return app.Result{}, fmt.Errorf("write %s: %w", path, err)
				}
			}
		}
		for _, w := range report.Warnings {
			findings = append(findings, fmt.Sprintf("%s: %s", path, w))
		}
	}

	// Run dedup across all prose files
	if len(proseFiles) > 1 {
		dedupReport, err := dedup.Analyze(proseFiles, 0.6)
		if err != nil {
			findings = append(findings, fmt.Sprintf("dedup analysis failed: %v", err))
		} else {
			for _, d := range dedupReport.Exact {
				findings = append(findings, fmt.Sprintf("exact duplicate: %q in [%s]", d.Sentence, strings.Join(d.Locations, ", ")))
			}
			for _, d := range dedupReport.Near {
				findings = append(findings, fmt.Sprintf("near duplicate (%.0f%%): %q in [%s]", d.Similarity*100, d.Sentence, strings.Join(d.Locations, ", ")))
			}
		}
	}

	summary := fmt.Sprintf("maintain: %d files scanned, %d changed, saved %d bytes / %d approx tokens",
		len(files), filesChanged,
		totalStats.InputBytes-totalStats.OutputBytes,
		totalStats.InputApproxTokens-totalStats.OutputApproxTokens)

	return app.Result{
		Command:  "maintain",
		Changed:  filesChanged > 0,
		Stats:    totalStats,
		Findings: findings,
		Output:   summary,
	}, nil
}

func collectFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if detect.IsSensitivePathResolved(path) {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}

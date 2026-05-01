package smartread

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spw-m-riley/ma/internal/app"
	"github.com/spw-m-riley/ma/internal/codectx"
	"github.com/spw-m-riley/ma/internal/detect"
	"github.com/spw-m-riley/ma/internal/prose"
	"github.com/spw-m-riley/ma/internal/schema"
)

const defaultLineThreshold = 200

type Command struct{}

var supportedCodeExtensions = map[string]struct{}{
	".go":  {},
	".ts":  {},
	".tsx": {},
	".js":  {},
	".jsx": {},
}

var supportedConfigExtensions = map[string]struct{}{
	".json": {},
	".yaml": {},
	".yml":  {},
}

func NewCommand() Command {
	return Command{}
}

func (Command) Name() string {
	return "smart-read"
}

func (Command) Run(args []string) (app.Result, error) {
	if len(args) != 1 {
		return app.Result{}, fmt.Errorf("usage: ma smart-read <file> [--json]")
	}

	path := args[0]
	if detect.IsSensitivePathResolved(path) {
		return app.Result{}, fmt.Errorf("refusing sensitive path %q", path)
	}

	inputBytes, err := os.ReadFile(path)
	if err != nil {
		return app.Result{}, err
	}
	input := string(inputBytes)

	lineCount := strings.Count(input, "\n") + 1
	if lineCount < defaultLineThreshold {
		return passthrough(input, "below_threshold"), nil
	}

	classification := detect.Classify(path, input)
	if !supportsReduction(path, classification) {
		return passthroughWithFindings(input,
			fmt.Sprintf("classification=%s", classification),
			"passthrough=unsupported_reducer",
		), nil
	}

	output, findings, err := reduce(path, input, classification)
	if err != nil {
		return passthroughWithFindings(input,
			fmt.Sprintf("classification=%s", classification),
			"passthrough=reduction_failed",
		), nil
	}

	return app.Result{
		Command:        "smart-read",
		Changed:        output != input,
		ProducedOutput: true,
		Stats:          app.Measure(input, output),
		Findings:       append(findings, fmt.Sprintf("classification=%s", classification)),
		Output:         output,
	}, nil
}

func passthrough(input string, reason string) app.Result {
	return passthroughWithFindings(input, fmt.Sprintf("passthrough=%s", reason))
}

func passthroughWithFindings(input string, findings ...string) app.Result {
	return app.Result{
		Command:        "smart-read",
		Changed:        false,
		ProducedOutput: true,
		Stats:          app.Measure(input, input),
		Findings:       findings,
		Output:         input,
	}
}

func supportsReduction(path string, classification detect.Classification) bool {
	switch classification {
	case detect.NaturalLanguage:
		return true
	case detect.Code:
		_, ok := supportedCodeExtensions[filepath.Ext(path)]
		return ok
	case detect.Config:
		_, ok := supportedConfigExtensions[filepath.Ext(path)]
		return ok
	default:
		return false
	}
}

func reduce(path string, input string, classification detect.Classification) (string, []string, error) {
	switch classification {
	case detect.NaturalLanguage:
		output := prose.Compress(input)
		return output, nil, nil

	case detect.Code:
		output, warnings, err := codectx.ReduceFile(path, []byte(input))
		if err != nil {
			return "", nil, err
		}
		return output, warnings, nil

	case detect.Config:
		output, err := minifyConfig(path, input)
		if err != nil {
			return "", nil, err
		}
		return output, nil, nil

	default:
		return "", nil, fmt.Errorf("unsupported classification %q", classification)
	}
}

func minifyConfig(path string, input string) (string, error) {
	switch filepath.Ext(path) {
	case ".json":
		return schema.MinifyJSON(input)
	case ".yaml", ".yml":
		return schema.MinifyYAML(input)
	default:
		return "", fmt.Errorf("unsupported config extension %q", filepath.Ext(path))
	}
}

package testutil

import (
	"fmt"
	"os"

	"github.com/spw-m-riley/ma/internal/app"
)

type GoldenFixture struct {
	InputPath    string
	ExpectedPath string
	Input        string
	Expected     string
}

func LoadGoldenFixture(basePath string, ext string) (GoldenFixture, error) {
	inputPath := basePath + ".input" + ext
	expectedPath := basePath + ".expected" + ext

	input, err := os.ReadFile(inputPath)
	if err != nil {
		return GoldenFixture{}, err
	}

	expected, err := os.ReadFile(expectedPath)
	if err != nil {
		return GoldenFixture{}, err
	}

	return GoldenFixture{
		InputPath:    inputPath,
		ExpectedPath: expectedPath,
		Input:        string(input),
		Expected:     string(expected),
	}, nil
}

func AssertApproxTokenReductionAtLeast(stats app.Stats, percent float64) error {
	if stats.InputApproxTokens <= 0 {
		return fmt.Errorf("input approx tokens must be positive")
	}

	reduction := float64(stats.InputApproxTokens-stats.OutputApproxTokens) * 100 / float64(stats.InputApproxTokens)
	if reduction < percent {
		return fmt.Errorf("approx token reduction %.2f%% below required %.2f%%", reduction, percent)
	}

	return nil
}

package prose

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/spw-m-riley/ma/internal/app"
	"github.com/spw-m-riley/ma/internal/testutil"
	"github.com/spw-m-riley/ma/internal/validate"
)

var realisticFixtures = []string{
	"verbose-system-prompt",
	"technical-api-docs",
	"project-readme",
	"instructional-guide",
}

var realisticFixtureThresholds = map[string]float64{
	"verbose-system-prompt": 20,
	"technical-api-docs":    20,
	"project-readme":        20,
	"instructional-guide":   20,
}

func TestCompressRealisticFixtures(t *testing.T) {
	for _, name := range realisticFixtures {
		t.Run(name, func(t *testing.T) {
			fixture := loadProseFixture(t, name)
			if got := Compress(fixture.Input); got != fixture.Expected {
				t.Fatalf("unexpected compressed output\nwant: %q\ngot:  %q", fixture.Expected, got)
			}
		})
	}
}

func TestCompressRealisticFixturesPreserveStructure(t *testing.T) {
	for _, name := range realisticFixtures {
		t.Run(name, func(t *testing.T) {
			fixture := loadProseFixture(t, name)
			report := validate.Compare(fixture.Input, Compress(fixture.Input))
			if !report.Valid {
				t.Fatalf("expected structurally valid output: %v", report.Error())
			}
		})
	}
}

func TestCompressRealisticFixtureReduction(t *testing.T) {
	for _, name := range realisticFixtures {
		t.Run(name, func(t *testing.T) {
			fixture := loadProseFixture(t, name)
			stats := app.Measure(fixture.Input, Compress(fixture.Input))
			if err := testutil.AssertApproxTokenReductionAtLeast(stats, realisticFixtureThresholds[name]); err != nil {
				t.Fatalf("expected %s to meet reduction target: %v", name, err)
			}
		})
	}
}

func TestCompressRealisticFixtureAggregateReduction(t *testing.T) {
	var totalInput strings.Builder
	var totalOutput strings.Builder

	for _, name := range realisticFixtures {
		fixture := loadProseFixture(t, name)
		totalInput.WriteString(fixture.Input)
		totalOutput.WriteString(Compress(fixture.Input))
	}

	stats := app.Measure(totalInput.String(), totalOutput.String())
	if err := testutil.AssertApproxTokenReductionAtLeast(stats, 15); err != nil {
		t.Fatalf("expected aggregate realistic reduction to meet target: %v", err)
	}
}

func loadProseFixture(t *testing.T, name string) testutil.GoldenFixture {
	t.Helper()

	base := filepath.Join("..", "..", "testdata", "prose", name)
	fixture, err := testutil.LoadGoldenFixture(base, ".md")
	if err != nil {
		t.Fatalf("load fixture %s: %v", name, err)
	}

	return fixture
}

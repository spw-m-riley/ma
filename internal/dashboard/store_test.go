package dashboard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spw-m-riley/ma/internal/app"
)

func TestStorePersistsRunHistoryAndAggregates(t *testing.T) {
	root := t.TempDir()

	store, err := OpenStore(root)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	startedAt := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(2 * time.Second)

	if err := store.RecordFinished(FinishedRun{
		ID:         "run-1",
		Command:    "compress",
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
		Success:    true,
		Changed:    true,
		Result: app.Result{
			Command: "compress",
			Changed: true,
			Stats: app.Stats{
				InputBytes:         120,
				OutputBytes:        72,
				InputWords:         20,
				OutputWords:        12,
				InputApproxTokens:  28,
				OutputApproxTokens: 17,
			},
			Output: "sensitive output that must not be persisted",
		},
	}); err != nil {
		t.Fatalf("record finished run: %v", err)
	}

	if err := store.RecordFinished(FinishedRun{
		ID:         "run-2",
		Command:    "validate",
		StartedAt:  startedAt.Add(5 * time.Second),
		FinishedAt: finishedAt.Add(5 * time.Second),
		Success:    false,
		Error:      "candidate diverged",
		Result: app.Result{
			Command: "validate",
		},
	}); err != nil {
		t.Fatalf("record failed run: %v", err)
	}

	reopened, err := OpenStore(root)
	if err != nil {
		t.Fatalf("re-open store: %v", err)
	}

	summary, err := reopened.Summary()
	if err != nil {
		t.Fatalf("summary: %v", err)
	}

	if summary.TotalRuns != 2 {
		t.Fatalf("expected 2 total runs, got %d", summary.TotalRuns)
	}
	if summary.SuccessfulRuns != 1 {
		t.Fatalf("expected 1 successful run, got %d", summary.SuccessfulRuns)
	}
	if summary.FailedRuns != 1 {
		t.Fatalf("expected 1 failed run, got %d", summary.FailedRuns)
	}
	if summary.TotalBytesSaved != 48 {
		t.Fatalf("expected 48 total bytes saved, got %d", summary.TotalBytesSaved)
	}
	if summary.TotalWordsSaved != 8 {
		t.Fatalf("expected 8 total words saved, got %d", summary.TotalWordsSaved)
	}
	if summary.TotalApproxTokensSaved != 11 {
		t.Fatalf("expected 11 total tokens saved, got %d", summary.TotalApproxTokensSaved)
	}
	if got := summary.CommandUsage["compress"]; got != 1 {
		t.Fatalf("expected compress usage count 1, got %d", got)
	}
	if got := summary.CommandUsage["validate"]; got != 1 {
		t.Fatalf("expected validate usage count 1, got %d", got)
	}

	historyBytes, err := os.ReadFile(filepath.Join(root, "history.jsonl"))
	if err != nil {
		t.Fatalf("read history file: %v", err)
	}
	if string(historyBytes) == "" {
		t.Fatalf("expected persisted history data")
	}
	if strings.Contains(string(historyBytes), "sensitive output that must not be persisted") {
		t.Fatalf("durable history unexpectedly persisted output body: %q", string(historyBytes))
	}
}

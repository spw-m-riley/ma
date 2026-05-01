package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spw-m-riley/ma/internal/app"
)

func TestServerRendersDashboardSummary(t *testing.T) {
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	startedAt := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	if err := store.RecordFinished(FinishedRun{
		ID:         "run-1",
		Command:    "compress",
		StartedAt:  startedAt,
		FinishedAt: startedAt.Add(2 * time.Second),
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
		},
	}); err != nil {
		t.Fatalf("record run: %v", err)
	}

	handler := NewServer(store).Handler()
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	for _, want := range []string{
		"ma dashboard",
		"Total runs",
		"48 bytes",
		"Recent runs",
		"View stats",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected dashboard body to contain %q, got %q", want, body)
		}
	}
}

func TestServerRendersActivityFirstOverviewLayout(t *testing.T) {
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	handler := NewServer(store).Handler()
	startedAt := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(4 * time.Second)

	if err := store.RecordFinished(FinishedRun{
		ID:         "run-success",
		Command:    "compress",
		StartedAt:  startedAt,
		FinishedAt: startedAt.Add(2 * time.Second),
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
		},
	}); err != nil {
		t.Fatalf("record successful run: %v", err)
	}

	postDashboardEvent(t, handler, RunEvent{
		Kind:          eventKindStarted,
		ID:            "run-active",
		Command:       "validate",
		StartedAt:     startedAt.Add(10 * time.Second),
		ResultSummary: "Comparing candidate against baseline",
	})
	postDashboardEvent(t, handler, RunEvent{
		Kind:          eventKindFailed,
		ID:            "run-failed",
		Command:       "dedup",
		StartedAt:     startedAt.Add(20 * time.Second),
		FinishedAt:    &finishedAt,
		Error:         "duplicate scan exceeded budget",
		ResultSummary: "duplicate scan exceeded budget",
	})

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	for _, want := range []string{
		`id="activity-layout"`,
		`id="savings-band"`,
		`id="recent-runs-list"`,
		`recent-runs-head`,
		`status-pill status-failed`,
		`status-pill status-started`,
		`2026-04-30 12:00:04 UTC`,
		`View stats`,
		`duplicate scan exceeded budget`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected overview body to contain %q, got %q", want, body)
		}
	}
}

func TestServerTracksRunLifecycleEvents(t *testing.T) {
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	handler := NewServer(store).Handler()
	startedAt := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(2 * time.Second)

	postDashboardEvent(t, handler, RunEvent{
		Kind:      eventKindStarted,
		ID:        "run-1",
		Command:   "compress",
		StartedAt: startedAt,
	})
	postDashboardEvent(t, handler, RunEvent{
		Kind:          eventKindFinished,
		ID:            "run-1",
		Command:       "compress",
		StartedAt:     startedAt,
		FinishedAt:    &finishedAt,
		Success:       true,
		Changed:       true,
		ResultSummary: "changed=true, saved 48 bytes, 8 words, 11 approx tokens",
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
		},
	})

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/runs", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var snapshot RunsSnapshot
	if err := json.Unmarshal(recorder.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("decode runs snapshot: %v", err)
	}
	if len(snapshot.Runs) != 1 {
		t.Fatalf("expected 1 run in snapshot, got %d", len(snapshot.Runs))
	}
	if snapshot.Runs[0].Status != eventKindFinished {
		t.Fatalf("expected finished status, got %q", snapshot.Runs[0].Status)
	}
	if snapshot.Runs[0].ResultSummary == "" {
		t.Fatalf("expected result summary in snapshot")
	}
}

func TestServerLoadsRecentRunDetailsFromDurableStore(t *testing.T) {
	root := t.TempDir()
	store, err := OpenStore(root)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	startedAt := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	if err := store.RecordFinished(FinishedRun{
		ID:         "run-1",
		Command:    "compress",
		StartedAt:  startedAt,
		FinishedAt: startedAt.Add(2 * time.Second),
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
			Output: "after",
		},
	}); err != nil {
		t.Fatalf("record run: %v", err)
	}

	reopened, err := OpenStore(root)
	if err != nil {
		t.Fatalf("re-open store: %v", err)
	}

	handler := NewServer(reopened).Handler()
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/runs/run-1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200 for durable run detail, got %d body=%q", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "after") {
		t.Fatalf("expected durable run detail to include output body, got %q", recorder.Body.String())
	}
}

func TestServerExposesOverviewSnapshot(t *testing.T) {
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	finishedStartedAt := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	if err := store.RecordFinished(FinishedRun{
		ID:         "run-finished",
		Command:    "compress",
		StartedAt:  finishedStartedAt,
		FinishedAt: finishedStartedAt.Add(2 * time.Second),
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
		},
	}); err != nil {
		t.Fatalf("record run: %v", err)
	}

	handler := NewServer(store).Handler()
	activeStartedAt := finishedStartedAt.Add(10 * time.Second)
	postDashboardEvent(t, handler, RunEvent{
		Kind:      eventKindStarted,
		ID:        "run-active",
		Command:   "validate",
		StartedAt: activeStartedAt,
	})

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/overview", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%q", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Summary struct {
			TotalRuns              int `json:"totalRuns"`
			SuccessfulRuns         int `json:"successfulRuns"`
			FailedRuns             int `json:"failedRuns"`
			TotalBytesSaved        int `json:"totalBytesSaved"`
			TotalWordsSaved        int `json:"totalWordsSaved"`
			TotalApproxTokensSaved int `json:"totalApproxTokensSaved"`
		} `json:"summary"`
		CommandUsage []commandUsage `json:"commandUsage"`
		ActiveRuns   int            `json:"activeRuns"`
		Runs         []RunView      `json:"runs"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode overview snapshot: %v", err)
	}

	if payload.Summary.TotalRuns != 1 {
		t.Fatalf("expected 1 total run, got %d", payload.Summary.TotalRuns)
	}
	if payload.Summary.SuccessfulRuns != 1 {
		t.Fatalf("expected 1 successful run, got %d", payload.Summary.SuccessfulRuns)
	}
	if payload.Summary.TotalBytesSaved != 48 {
		t.Fatalf("expected 48 bytes saved, got %d", payload.Summary.TotalBytesSaved)
	}
	if payload.ActiveRuns != 1 {
		t.Fatalf("expected 1 active run, got %d", payload.ActiveRuns)
	}
	if len(payload.CommandUsage) != 1 || payload.CommandUsage[0].Command != "compress" {
		t.Fatalf("expected command usage for compress, got %#v", payload.CommandUsage)
	}
	if len(payload.Runs) != 2 {
		t.Fatalf("expected durable and active runs in overview snapshot, got %#v", payload.Runs)
	}
	if payload.Runs[0].Status != eventKindStarted {
		t.Fatalf("expected active run to sort first in overview snapshot, got %#v", payload.Runs)
	}
}

func TestServerRendersLiveOverviewSections(t *testing.T) {
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	startedAt := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	if err := store.RecordFinished(FinishedRun{
		ID:         "run-finished",
		Command:    "compress",
		StartedAt:  startedAt,
		FinishedAt: startedAt.Add(2 * time.Second),
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
		},
	}); err != nil {
		t.Fatalf("record run: %v", err)
	}

	handler := NewServer(store).Handler()
	postDashboardEvent(t, handler, RunEvent{
		Kind:      eventKindStarted,
		ID:        "run-active",
		Command:   "validate",
		StartedAt: startedAt.Add(10 * time.Second),
	})

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	for _, want := range []string{
		"id=\"activity-layout\"",
		"id=\"savings-band\"",
		"id=\"recent-runs-list\"",
		"id=\"running-now\"",
		"Running now",
		"fetch('/api/overview')",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected overview body to contain %q, got %q", want, body)
		}
	}
}

func TestServerShowsRecentRunDetailsWithoutDurableBodies(t *testing.T) {
	root := t.TempDir()
	store, err := OpenStore(root)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	handler := NewServer(store).Handler()
	startedAt := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(2 * time.Second)

	postDashboardEvent(t, handler, RunEvent{
		Kind:          eventKindFinished,
		ID:            "run-1",
		Command:       "compress",
		StartedAt:     startedAt,
		FinishedAt:    &finishedAt,
		Success:       true,
		Changed:       true,
		PayloadStatus: payloadStatusObserved,
		Input:         "before",
		ResultSummary: "changed=true, saved 48 bytes, 8 words, 11 approx tokens",
		Result: app.Result{
			Command: "compress",
			Changed: true,
			Output:  "after",
		},
	})

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/runs/run-1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	for _, want := range []string{"before", "after", "Result summary"} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected detail body to contain %q, got %q", want, body)
		}
	}

	historyBytes, err := os.ReadFile(filepath.Join(root, historyFileName))
	if err == nil && strings.Contains(string(historyBytes), "before") {
		t.Fatalf("expected recent input body to stay out of durable history, got %q", string(historyBytes))
	}
}

func TestServerRunDetailRendersIntentionalPanelStates(t *testing.T) {
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	handler := NewServer(store).Handler()
	startedAt := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(2 * time.Second)

	postDashboardEvent(t, handler, RunEvent{
		Kind:          eventKindFailed,
		ID:            "run-1",
		Command:       "compress",
		StartedAt:     startedAt,
		FinishedAt:    &finishedAt,
		Error:         "candidate diverged",
		PayloadStatus: payloadStatusRedacted,
	})

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/runs/run-1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	for _, want := range []string{
		"comparison-grid",
		"Payload: Withheld",
		"state-redacted",
		"state-unavailable",
		"This run involved a sensitive or protected path",
		"This run ended with an error before output text was captured.",
		"@media (min-width: 72rem)",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected detail body to contain %q, got %q", want, body)
		}
	}
}

func TestServerRendersDedicatedStatsView(t *testing.T) {
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	firstStartedAt := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	secondStartedAt := time.Date(2026, 4, 30, 13, 0, 0, 0, time.UTC)

	if err := store.RecordFinished(FinishedRun{
		ID:         "run-1",
		Command:    "compress",
		StartedAt:  firstStartedAt,
		FinishedAt: firstStartedAt.Add(2 * time.Second),
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
		},
	}); err != nil {
		t.Fatalf("record first run: %v", err)
	}

	if err := store.RecordFinished(FinishedRun{
		ID:         "run-2",
		Command:    "validate",
		StartedAt:  secondStartedAt,
		FinishedAt: secondStartedAt.Add(2 * time.Second),
		Success:    false,
		Error:      "candidate diverged",
		Result: app.Result{
			Command: "validate",
			Stats: app.Stats{
				InputBytes:         90,
				OutputBytes:        90,
				InputWords:         14,
				OutputWords:        14,
				InputApproxTokens:  20,
				OutputApproxTokens: 20,
			},
		},
	}); err != nil {
		t.Fatalf("record second run: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/stats", nil)
	recorder := httptest.NewRecorder()
	NewServer(store).Handler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	for _, want := range []string{
		"Usage trends",
		"2026-03",
		"2026-04",
		"compress",
		"validate",
		"1 successful / 1 failed",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected stats body to contain %q, got %q", want, body)
		}
	}
}

func TestServerRendersStatsAnalysisPanels(t *testing.T) {
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	firstStartedAt := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	secondStartedAt := time.Date(2026, 4, 30, 13, 0, 0, 0, time.UTC)

	if err := store.RecordFinished(FinishedRun{
		ID:         "run-1",
		Command:    "compress",
		StartedAt:  firstStartedAt,
		FinishedAt: firstStartedAt.Add(2 * time.Second),
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
		},
	}); err != nil {
		t.Fatalf("record first run: %v", err)
	}

	if err := store.RecordFinished(FinishedRun{
		ID:         "run-2",
		Command:    "validate",
		StartedAt:  secondStartedAt,
		FinishedAt: secondStartedAt.Add(2 * time.Second),
		Success:    false,
		Error:      "candidate diverged",
		Result: app.Result{
			Command: "validate",
			Stats: app.Stats{
				InputBytes:         90,
				OutputBytes:        90,
				InputWords:         14,
				OutputWords:        14,
				InputApproxTokens:  20,
				OutputApproxTokens: 20,
			},
		},
	}); err != nil {
		t.Fatalf("record second run: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/stats", nil)
	recorder := httptest.NewRecorder()
	NewServer(store).Handler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	for _, want := range []string{
		"Command rankings",
		"Outcome context",
		"success-meter",
		"steady savings",
		"Most active command",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected stats body to contain %q, got %q", want, body)
		}
	}
}

func TestServerExposesStatsSnapshot(t *testing.T) {
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	firstStartedAt := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	secondStartedAt := time.Date(2026, 4, 30, 13, 0, 0, 0, time.UTC)

	if err := store.RecordFinished(FinishedRun{
		ID:         "run-1",
		Command:    "compress",
		StartedAt:  firstStartedAt,
		FinishedAt: firstStartedAt.Add(2 * time.Second),
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
		},
	}); err != nil {
		t.Fatalf("record first run: %v", err)
	}

	if err := store.RecordFinished(FinishedRun{
		ID:         "run-2",
		Command:    "validate",
		StartedAt:  secondStartedAt,
		FinishedAt: secondStartedAt.Add(2 * time.Second),
		Success:    false,
		Error:      "candidate diverged",
		Result: app.Result{
			Command: "validate",
			Stats: app.Stats{
				InputBytes:         90,
				OutputBytes:        90,
				InputWords:         14,
				OutputWords:        14,
				InputApproxTokens:  20,
				OutputApproxTokens: 20,
			},
		},
	}); err != nil {
		t.Fatalf("record second run: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/api/stats", nil)
	recorder := httptest.NewRecorder()
	NewServer(store).Handler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%q", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		SuccessfulRuns int            `json:"successfulRuns"`
		FailedRuns     int            `json:"failedRuns"`
		CommandUsage   []commandUsage `json:"commandUsage"`
		TrendRows      []trendRow     `json:"trendRows"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode stats snapshot: %v", err)
	}

	if payload.SuccessfulRuns != 1 {
		t.Fatalf("expected 1 successful run, got %d", payload.SuccessfulRuns)
	}
	if payload.FailedRuns != 1 {
		t.Fatalf("expected 1 failed run, got %d", payload.FailedRuns)
	}
	if len(payload.CommandUsage) != 2 {
		t.Fatalf("expected 2 command usage rows, got %#v", payload.CommandUsage)
	}
	if len(payload.TrendRows) != 2 {
		t.Fatalf("expected 2 trend rows, got %#v", payload.TrendRows)
	}
	if payload.TrendRows[0].Month != "2026-03" {
		t.Fatalf("expected first trend row to be for 2026-03, got %#v", payload.TrendRows)
	}
}

func TestServerRendersLiveStatsSections(t *testing.T) {
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	startedAt := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	if err := store.RecordFinished(FinishedRun{
		ID:         "run-1",
		Command:    "compress",
		StartedAt:  startedAt,
		FinishedAt: startedAt.Add(2 * time.Second),
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
		},
	}); err != nil {
		t.Fatalf("record run: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/stats", nil)
	recorder := httptest.NewRecorder()
	NewServer(store).Handler().ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	body := recorder.Body.String()
	for _, want := range []string{
		"id=\"stats-outcomes\"",
		"id=\"usage-trend-rows\"",
		"id=\"stats-command-usage\"",
		"fetch('/api/stats')",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected stats body to contain %q, got %q", want, body)
		}
	}
}

func TestServerOverviewShowsRedactedRunState(t *testing.T) {
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	handler := NewServer(store).Handler()
	startedAt := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(2 * time.Second)

	postDashboardEvent(t, handler, RunEvent{
		Kind:          eventKindFinished,
		ID:            "run-1",
		Command:       "compress",
		StartedAt:     startedAt,
		FinishedAt:    &finishedAt,
		Success:       true,
		PayloadStatus: payloadStatusRedacted,
		ResultSummary: "changed=true, saved 48 bytes, 8 words, 11 approx tokens",
	})

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "redacted") {
		t.Fatalf("expected overview to show redacted state, got %q", recorder.Body.String())
	}
}

func postDashboardEvent(t *testing.T, handler http.Handler, event RunEvent) {
	t.Helper()

	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/events", strings.NewReader(string(payload)))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", recorder.Code)
	}
}

package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/spw-m-riley/ma/internal/app"
)

func TestObserveRunPublishesStartedAndFinishedEvents(t *testing.T) {
	root := t.TempDir()
	t.Setenv(stateDirEnv, root)

	var (
		mu     sync.Mutex
		events []RunEvent
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/events" {
			http.NotFound(w, r)
			return
		}
		defer r.Body.Close()

		var event RunEvent
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Fatalf("decode event: %v", err)
		}

		mu.Lock()
		events = append(events, event)
		mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	if err := writeSession(sessionPath(root), Session{Address: strings.TrimPrefix(server.URL, "http://")}); err != nil {
		t.Fatalf("write session: %v", err)
	}

	result, err := ObserveRun("compress", nil, func() (app.Result, error) {
		return app.Result{
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
		}, nil
	})
	if err != nil {
		t.Fatalf("observe run: %v", err)
	}
	if result.Command != "compress" {
		t.Fatalf("expected compress result, got %q", result.Command)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(events) != 2 {
		t.Fatalf("expected 2 published events, got %d", len(events))
	}
	if events[0].Kind != "started" {
		t.Fatalf("expected first event to be started, got %q", events[0].Kind)
	}
	if events[1].Kind != "finished" {
		t.Fatalf("expected second event to be finished, got %q", events[1].Kind)
	}
	if !events[1].Success {
		t.Fatalf("expected finished event to be successful")
	}
	if events[1].Command != "compress" {
		t.Fatalf("expected command compress, got %q", events[1].Command)
	}
}

func TestObserveRunRecordsDiagnosticWhenDeliveryFails(t *testing.T) {
	root := t.TempDir()
	t.Setenv(stateDirEnv, root)

	if err := writeSession(sessionPath(root), Session{Address: "127.0.0.1:1"}); err != nil {
		t.Fatalf("write session: %v", err)
	}

	result, err := ObserveRun("compress", nil, func() (app.Result, error) {
		return app.Result{Command: "compress"}, nil
	})
	if err != nil {
		t.Fatalf("observe run: %v", err)
	}
	if result.Command != "compress" {
		t.Fatalf("expected compress result, got %q", result.Command)
	}

	diagnosticsBytes, err := os.ReadFile(filepath.Join(root, diagnosticsFileName))
	if err != nil {
		t.Fatalf("read diagnostics file: %v", err)
	}
	if !strings.Contains(string(diagnosticsBytes), "event delivery") {
		t.Fatalf("expected delivery diagnostic, got %q", string(diagnosticsBytes))
	}
}

func TestObserveRunRecordsDiagnosticWhenPersistenceFails(t *testing.T) {
	parent := t.TempDir()
	rootFile := filepath.Join(parent, "not-a-directory")
	if err := os.WriteFile(rootFile, []byte("occupied"), 0o644); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}
	t.Setenv(stateDirEnv, rootFile)

	result, err := ObserveRun("compress", nil, func() (app.Result, error) {
		return app.Result{Command: "compress"}, nil
	})
	if err != nil {
		t.Fatalf("observe run: %v", err)
	}
	if result.Command != "compress" {
		t.Fatalf("expected compress result, got %q", result.Command)
	}

	diagnosticsBytes, err := os.ReadFile(filepath.Join(parent, diagnosticsFileName))
	if err != nil {
		t.Fatalf("read diagnostics file: %v", err)
	}
	if !strings.Contains(string(diagnosticsBytes), "history persistence") {
		t.Fatalf("expected persistence diagnostic, got %q", string(diagnosticsBytes))
	}
}

func TestObserveRunPublishesRedactedPayloadForSensitivePaths(t *testing.T) {
	root := t.TempDir()
	t.Setenv(stateDirEnv, root)

	var (
		mu     sync.Mutex
		events []RunEvent
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var event RunEvent
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			t.Fatalf("decode event: %v", err)
		}

		mu.Lock()
		events = append(events, event)
		mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	if err := writeSession(sessionPath(root), Session{Address: strings.TrimPrefix(server.URL, "http://")}); err != nil {
		t.Fatalf("write session: %v", err)
	}

	sensitivePath := filepath.Join(root, ".ssh", "config")
	if _, err := ObserveRun("compress", []string{sensitivePath}, func() (app.Result, error) {
		return app.Result{Command: "compress"}, nil
	}); err != nil {
		t.Fatalf("observe run: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(events) != 2 {
		t.Fatalf("expected 2 published events, got %d", len(events))
	}
	finished := events[1]
	if finished.PayloadStatus != payloadStatusRedacted {
		t.Fatalf("expected redacted payload status, got %q", finished.PayloadStatus)
	}
	if finished.Input != "" {
		t.Fatalf("expected sensitive input to be omitted, got %q", finished.Input)
	}
}

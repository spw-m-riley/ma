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
	"time"

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
	mu.Unlock()

	events = waitForEventCount(t, &mu, &events, 2)
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

	diagnosticsBytes := waitForDiagnostic(t, filepath.Join(root, diagnosticsFileName), "event delivery")
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
	mu.Unlock()

	events = waitForEventCount(t, &mu, &events, 2)
	finished := events[1]
	if finished.PayloadStatus != payloadStatusRedacted {
		t.Fatalf("expected redacted payload status, got %q", finished.PayloadStatus)
	}
	if finished.Input != "" {
		t.Fatalf("expected sensitive input to be omitted, got %q", finished.Input)
	}
}

func TestObserveRunTrimsObservedPayload(t *testing.T) {
	root := t.TempDir()
	t.Setenv(stateDirEnv, root)
	t.Setenv("MA_DASHBOARD_PAYLOAD_LIMIT", "64")

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

	path := filepath.Join(root, "input.txt")
	if err := os.WriteFile(path, []byte(strings.Repeat("0123456789", 20)), 0o644); err != nil {
		t.Fatalf("write large input: %v", err)
	}

	if _, err := ObserveRun("compress", []string{path}, func() (app.Result, error) {
		return app.Result{
			Command: "compress",
			Output:  strings.Repeat("out", 40),
		}, nil
	}); err != nil {
		t.Fatalf("observe run: %v", err)
	}

	events = waitForEventCount(t, &mu, &events, 2)
	finished := events[len(events)-1]
	if !strings.Contains(finished.Input, "[truncated") {
		t.Fatalf("expected trimmed input payload, got %q", finished.Input)
	}
	if !strings.Contains(finished.Result.Output, "[truncated") {
		t.Fatalf("expected trimmed output payload, got %q", finished.Result.Output)
	}
}

func waitForEventCount(t *testing.T, mu *sync.Mutex, events *[]RunEvent, want int) []RunEvent {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for {
		mu.Lock()
		current := append([]RunEvent(nil), (*events)...)
		mu.Unlock()
		if len(current) >= want {
			return current
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d events, got %d", want, len(current))
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitForDiagnostic(t *testing.T, path string, want string) []byte {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for {
		content, err := os.ReadFile(path)
		if err == nil && strings.Contains(string(content), want) {
			return content
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for diagnostic %q in %s", want, path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitForSessionRemoval(t *testing.T, path string) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for {
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for stale session removal at %s", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestObserveRunRemovesStaleSessionFileAfterDeliveryFailure(t *testing.T) {
	root := t.TempDir()
	t.Setenv(stateDirEnv, root)

	if err := writeSession(sessionPath(root), Session{Address: "127.0.0.1:1"}); err != nil {
		t.Fatalf("write session: %v", err)
	}

	if _, err := ObserveRun("compress", nil, func() (app.Result, error) {
		return app.Result{Command: "compress"}, nil
	}); err != nil {
		t.Fatalf("observe run: %v", err)
	}

	waitForSessionRemoval(t, sessionPath(root))
}

func TestObserveRunReturnsWithoutWaitingForSlowDashboard(t *testing.T) {
	root := t.TempDir()
	t.Setenv(stateDirEnv, root)

	var received sync.WaitGroup
	received.Add(2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(750 * time.Millisecond)
		received.Done()
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	if err := writeSession(sessionPath(root), Session{Address: strings.TrimPrefix(server.URL, "http://")}); err != nil {
		t.Fatalf("write session: %v", err)
	}

	start := time.Now()
	if _, err := ObserveRun("compress", nil, func() (app.Result, error) {
		return app.Result{Command: "compress"}, nil
	}); err != nil {
		t.Fatalf("observe run: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 400*time.Millisecond {
		t.Fatalf("expected observe run to return before slow dashboard delivery completed, took %s", elapsed)
	}

	done := make(chan struct{})
	go func() {
		received.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for background dashboard delivery")
	}
}

package dashboard

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCommandRunServesLoopbackDashboardAndSession(t *testing.T) {
	root := t.TempDir()
	t.Setenv(stateDirEnv, root)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- NewCommand(io.Discard).run(ctx, nil)
	}()

	sessionPath := filepath.Join(root, sessionFileName)
	session := waitForSession(t, sessionPath)

	response, err := http.Get("http://" + session.Address + "/")
	if err != nil {
		t.Fatalf("get dashboard overview: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.StatusCode)
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read dashboard overview: %v", err)
	}
	if !strings.Contains(string(bodyBytes), "ma dashboard") {
		t.Fatalf("expected dashboard body, got %q", string(bodyBytes))
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for dashboard command to stop")
	}
}

func waitForSession(t *testing.T, path string) Session {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		payload, err := os.ReadFile(path)
		if err == nil {
			var session Session
			if err := json.Unmarshal(payload, &session); err != nil {
				t.Fatalf("decode session file: %v", err)
			}
			if session.Address == "" {
				t.Fatalf("expected dashboard session address")
			}
			return session
		}
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for dashboard session file %q", path)
	return Session{}
}

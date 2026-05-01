package dashboard

import (
	"testing"
	"time"
)

func TestFlushEventsIsIdempotent(t *testing.T) {
	root := t.TempDir()
	t.Setenv(stateDirEnv, root)

	dispatchEvent(root, RunEvent{Kind: eventKindStarted, ID: "test-1"})
	FlushEvents()
	FlushEvents() // second call must not panic
}

func TestFlushEventsDrainsQueuedEvents(t *testing.T) {
	root := t.TempDir()
	t.Setenv(stateDirEnv, root)

	// No dashboard server running — events will fail delivery silently.
	// The important thing is that the goroutine drains and exits.
	dispatchEvent(root, RunEvent{Kind: eventKindStarted, ID: "drain-1"})
	dispatchEvent(root, RunEvent{Kind: eventKindFinished, ID: "drain-1"})

	done := make(chan struct{})
	go func() {
		FlushEvents()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("FlushEvents did not return within 5 seconds")
	}
}

func TestDispatchEventAfterFlushDoesNotPanic(t *testing.T) {
	root := t.TempDir()
	t.Setenv(stateDirEnv, root)

	// Prime the dispatcher
	dispatchEvent(root, RunEvent{Kind: eventKindStarted, ID: "pre-flush"})
	FlushEvents()

	// This should recover gracefully, not panic on send-to-closed-channel.
	// A new dispatcher is created because FlushEvents deleted the old one.
	dispatchEvent(root, RunEvent{Kind: eventKindStarted, ID: "post-flush"})
	FlushEvents()
}
